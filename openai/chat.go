// Copyright 2024 Google LLC

// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at

//     https://www.apache.org/licenses/LICENSE-2.0

// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package openai

import (
	"encoding/json"
	"io/ioutil"
	"log"
	"net/http"
	"reflect"
	"strings"
	"time"

	"github.com/google-gemini/proxy-to-gemini/internal"
	"github.com/google/generative-ai-go/genai"
)

func (h *handlers) ChatCompletionsHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		internal.ErrorHandler(w, r, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	body, err := ioutil.ReadAll(r.Body)
	if err != nil {
		internal.ErrorHandler(w, r, http.StatusInternalServerError, "failed to read request body: %v", err)
		return
	}
	defer r.Body.Close()

	var chatReq ChatCompletionRequest
	if err := json.Unmarshal(body, &chatReq); err != nil {
		internal.ErrorHandler(w, r, http.StatusInternalServerError, "failed to parse chat completions body: %v", err)
		return
	}

	model := h.geminiClient.GenerativeModel(chatReq.Model)
	model.GenerationConfig = genai.GenerationConfig{
		CandidateCount:   chatReq.N,
		StopSequences:    chatReq.Stop,
		ResponseMIMEType: "text/plain",
		MaxOutputTokens:  chatReq.MaxTokens,
		Temperature:      chatReq.Temperature,
		TopP:             chatReq.TopP,
	}
	parts := []genai.Part{}
	for _, r := range chatReq.Messages {
		if r.Role == "system" {
			model.SystemInstruction = &genai.Content{
				Role:  r.Role,
				Parts: []genai.Part{genai.Text(r.Content)},
			}
			continue
		}
		// TODO: parts don't support role for model.GenerateContent
		parts = append(parts, genai.Text(r.Content))
	}

	if chatReq.Stream {
		streamingChatCompletionsHandler(w, r, chatReq.Model, model, parts)
		return
	}

	geminiResp, err := model.GenerateContent(r.Context(), parts...)
	if err != nil {
		internal.ErrorHandler(w, r, http.StatusInternalServerError, "failed to generate content: %v", err)
		return
	}

	resp := toOpenAIResponse(geminiResp, "chat.completion", chatReq.Model)
	if err := json.NewEncoder(w).Encode(resp); err != nil {
		internal.ErrorHandler(w, r, http.StatusInternalServerError, "failed to encode chat completions response: %v", err)
		return
	}
}

func toOpenAIResponse(from *genai.GenerateContentResponse, object, model string) (to ChatCompletionResponse) {
	to.Object = object
	to.Created = time.Now().Unix()
	to.Model = model
	if from.UsageMetadata != nil {
		to.Usage = Usage{
			PromptTokens:     from.UsageMetadata.PromptTokenCount,
			CompletionTokens: from.UsageMetadata.CandidatesTokenCount,
			TotalTokens:      from.UsageMetadata.TotalTokenCount,
		}
	}

	to.Choices = make([]ChatCompletionChoice, 0, len(from.Candidates))
	for i, c := range from.Candidates {
		var builder strings.Builder
		for _, p := range c.Content.Parts {
			content, ok := p.(genai.Text)
			if !ok {
				log.Printf("failed to process content part; type = %v", reflect.TypeOf(p))
				continue
			}
			builder.WriteString(string(content))
		}
		choice := ChatCompletionChoice{
			Index: i,
			Message: ChatMessage{
				Role:    c.Content.Role,
				Content: builder.String(),
			},
		}

		finishReason := toGeminiFinishReason(c.FinishReason)
		if finishReason != "" {
			choice.FinishReason = finishReason
		}
		to.Choices = append(to.Choices, choice)
	}
	return to
}

func toGeminiFinishReason(code genai.FinishReason) string {
	switch code {
	case genai.FinishReasonStop:
		return "stop"
	case genai.FinishReasonMaxTokens:
		return "length"
	case genai.FinishReasonRecitation:
		return "content_filter"
	case genai.FinishReasonSafety:
		return "content_filter"
	case genai.FinishReasonOther:
		return "other"
	default:
		return ""
	}
}
