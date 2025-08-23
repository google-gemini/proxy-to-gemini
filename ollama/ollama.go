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

// Package ollama provies handlers that proxies
// ollama API calls to Gemini models.
package ollama

import (
	"encoding/json"
	"fmt"
	"github.com/google-gemini/proxy-to-gemini/internal"
	"github.com/google/generative-ai-go/genai"
	"github.com/gorilla/mux"
	"io"
	"net/http"
	"strings"
	"time"
)

// handlers provides HTTP handlers for the Ollama proxy API.
type handlers struct {
	client *genai.Client
}

func RegisterHandlers(r *mux.Router, client *genai.Client) {
	handlers := &handlers{client: client}
	r.HandleFunc("/api/generate", handlers.generateHandler)
	r.HandleFunc("/api/embed", handlers.embedHandler)
	r.HandleFunc("/api/chat", handlers.chatHandler)
}

func (h *handlers) generateHandler(w http.ResponseWriter, r *http.Request) {
	body, err := io.ReadAll(r.Body)
	if err != nil {
		internal.ErrorHandler(w, r, http.StatusInternalServerError, "failed to read request body: %v", err)
		return
	}
	defer r.Body.Close()

	var req GenerateRequest
	if err := json.Unmarshal(body, &req); err != nil {
		internal.ErrorHandler(w, r, http.StatusInternalServerError, "failed to unmarshal request body: %v", err)
		return
	}

	model := h.client.GenerativeModel(req.Model)
	model.GenerationConfig = genai.GenerationConfig{
		Temperature:     req.Options.Temperature,
		MaxOutputTokens: req.Options.NumPredict,
		TopK:            req.Options.TopK,
		TopP:            req.Options.TopP,
	}
	if req.Options.Stop != nil {
		model.GenerationConfig.StopSequences = []string{*req.Options.Stop}
	}
	if req.System != "" {
		model.SystemInstruction = &genai.Content{
			Role:  "system",
			Parts: []genai.Part{genai.Text(req.System)},
		}
	}
	parts := []genai.Part{genai.Text(req.Prompt)}
	gresp, err := model.GenerateContent(r.Context(), parts...)
	if err != nil {
		internal.ErrorHandler(w, r, http.StatusInternalServerError, "failed to generate content: %v", err)
		return
	}
	if len(gresp.Candidates) == 0 {
		internal.ErrorHandler(w, r, http.StatusInternalServerError, "no candidates returned")
		return
	}

	responseBuilder := &strings.Builder{}
	for _, part := range gresp.Candidates[0].Content.Parts {
		switch v := part.(type) {
		case genai.Text:
			responseBuilder.WriteString(string(v))
		default:
			internal.ErrorHandler(w, r, http.StatusInternalServerError, "unsupported part type: %T", v)
			return
		}
	}
	if err := json.NewEncoder(w).Encode(&GenerateResponse{
		Model:           req.Model,
		Response:        responseBuilder.String(),
		CreatedAt:       time.Now(),
		PromptEvalCount: gresp.UsageMetadata.PromptTokenCount,
		EvalCount:       gresp.UsageMetadata.TotalTokenCount,
		Done:            true,
	}); err != nil {
		internal.ErrorHandler(w, r, http.StatusInternalServerError, "failed to encode generate response: %v", err)
		return
	}
}

func (h *handlers) embedHandler(w http.ResponseWriter, r *http.Request) {
	body, err := io.ReadAll(r.Body)
	if err != nil {
		internal.ErrorHandler(w, r, http.StatusInternalServerError, "failed to read request body: %v", err)
		return
	}
	defer r.Body.Close()

	var req EmbedRequest
	if err := json.Unmarshal(body, &req); err != nil {
		internal.ErrorHandler(w, r, http.StatusInternalServerError, "failed to unmarshal request body: %v", err)
		return
	}

	model := h.client.EmbeddingModel(req.Model)
	batch := model.NewBatch()
	for _, input := range req.Input {
		batch.AddContent(genai.Text(input))
	}

	gresp, err := model.BatchEmbedContents(r.Context(), batch)
	if err != nil {
		internal.ErrorHandler(w, r, http.StatusInternalServerError, "failed to create embedding: %v", err)
		return
	}

	embeddings := make([][]float32, 0, len(gresp.Embeddings))
	for _, embedding := range gresp.Embeddings {
		embeddings = append(embeddings, embedding.Values)
	}

	if err := json.NewEncoder(w).Encode(&EmbedResponse{
		Model:      req.Model,
		Embeddings: embeddings,
	}); err != nil {
		internal.ErrorHandler(w, r, http.StatusInternalServerError, "failed to encode embeddings response: %v", err)
		return
	}
}

// ChatRequest represents a chat completion request for the Ollama API.
type ChatRequest struct {
	Model    string          `json:"model,omitempty"`
	Messages []ChatMessage   `json:"messages,omitempty"`
	Format   json.RawMessage `json:"format,omitempty"`
	Options  Options         `json:"options,omitempty"`
}

// ChatMessage represents a single message in a chat.
type ChatMessage struct {
	Role    string   `json:"role,omitempty"`
	Content string   `json:"content,omitempty"`
	Images  []string `json:"images,omitempty"`
}

// ChatResponse represents a chat completion response for the Ollama API.
type ChatResponse struct {
	Model              string      `json:"model,omitempty"`
	CreatedAt          time.Time   `json:"created_at,omitempty"`
	Message            ChatMessage `json:"message,omitempty"`
	Done               bool        `json:"done,omitempty"`
	PromptEvalCount    int32       `json:"prompt_eval_count"`
	EvalCount          int32       `json:"eval_count"`
	TotalDuration      int64       `json:"total_duration"`
	LoadDuration       int64       `json:"load_duration,omitempty"`
	PromptEvalDuration int64       `json:"prompt_eval_duration,omitempty"`
	EvalDuration       int64       `json:"eval_duration"`
}

func sanitizeJson(s string) string {
	s = strings.ReplaceAll(s, "\n", "")
	s = strings.TrimPrefix(s, "```json")
	s = strings.TrimSuffix(s, "```")
	s = strings.ReplaceAll(s, "'", "\\'")
	return s
}

// chatHandler handles POST /api/chat requests.
func (h *handlers) chatHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		internal.ErrorHandler(w, r, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	body, err := io.ReadAll(r.Body)
	if err != nil {
		internal.ErrorHandler(w, r, http.StatusInternalServerError, "failed to read request body: %v", err)
		return
	}
	defer r.Body.Close()

	var req ChatRequest
	if err := json.Unmarshal(body, &req); err != nil {
		internal.ErrorHandler(w, r, http.StatusInternalServerError, "failed to unmarshal chat request: %v", err)
		return
	}

	// Handle advanced format parameter: JSON mode or JSON schema enforcement
	expectJson := false
	if len(req.Format) > 0 {
		expectJson = true
		var formatVal interface{}
		if err := json.Unmarshal(req.Format, &formatVal); err != nil {
			internal.ErrorHandler(w, r, http.StatusBadRequest, "invalid format parameter: %v", err)
			return
		}
		var instr string
		switch v := formatVal.(type) {
		case string:
			if v == "json" {
				instr = "Please respond with valid JSON."
			} else {
				instr = fmt.Sprintf("Please respond with format: %s.", v)
			}
		default:
			schemaBytes, err := json.MarshalIndent(v, "", "  ")
			if err != nil {
				schemaBytes = req.Format
			}
			instr = fmt.Sprintf("Please format your response according to the following JSON schema:\n%s", string(schemaBytes))
		}
		// Integrate with existing system message if present
		found := false
		for i, m := range req.Messages {
			if m.Role == "system" {
				req.Messages[i].Content = m.Content + "\n\n" + instr
				found = true
				break
			}
		}
		if !found {
			req.Messages = append([]ChatMessage{{Role: "system", Content: instr}}, req.Messages...)
		}
	}

	model := h.client.GenerativeModel(req.Model)
	model.GenerationConfig = genai.GenerationConfig{
		Temperature:     req.Options.Temperature,
		MaxOutputTokens: req.Options.NumPredict,
		TopK:            req.Options.TopK,
		TopP:            req.Options.TopP,
	}
	if req.Options.Stop != nil {
		model.GenerationConfig.StopSequences = []string{*req.Options.Stop}
	}

	chat := model.StartChat()
	var lastPart genai.Part
	for i, m := range req.Messages {
		if m.Role == "system" {
			model.SystemInstruction = &genai.Content{
				Role:  m.Role,
				Parts: []genai.Part{genai.Text(m.Content)},
			}
			continue
		}
		if i == len(req.Messages)-1 {
			lastPart = genai.Text(m.Content)
			break
		}
		chat.History = append(chat.History, &genai.Content{
			Role:  m.Role,
			Parts: []genai.Part{genai.Text(m.Content)},
		})
	}

	// Measure time spent generating the chat response
	start := time.Now()

	gresp, err := chat.SendMessage(r.Context(), lastPart)
	if err != nil {
		internal.ErrorHandler(w, r, http.StatusInternalServerError, "failed to send chat message: %v", err)
		return
	}
	var builder strings.Builder
	if len(gresp.Candidates) > 0 {
		for _, part := range gresp.Candidates[0].Content.Parts {
			if txt, ok := part.(genai.Text); ok {
				builder.WriteString(string(txt))
			}
		}
	}

	var resp ChatResponse
	if expectJson {
		resp = ChatResponse{
			Model:     req.Model,
			CreatedAt: time.Now(),
			Message: ChatMessage{
				Role:    gresp.Candidates[0].Content.Role,
				Content: sanitizeJson(builder.String()),
			},
			Done: true,
		}
	} else {
		resp = ChatResponse{
			Model:     req.Model,
			CreatedAt: time.Now(),
			Message: ChatMessage{
				Role:    gresp.Candidates[0].Content.Role,
				Content: builder.String(),
			},
			Done: true,
		}
	}

	if gresp.UsageMetadata != nil {
		resp.PromptEvalCount = gresp.UsageMetadata.PromptTokenCount
		// Compute number of tokens in the response.
		if gresp.UsageMetadata.CandidatesTokenCount > 0 {
			resp.EvalCount = gresp.UsageMetadata.CandidatesTokenCount
		} else if gresp.UsageMetadata.TotalTokenCount >= gresp.UsageMetadata.PromptTokenCount {
			// Fallback: use total tokens minus prompt tokens
			resp.EvalCount = gresp.UsageMetadata.TotalTokenCount - gresp.UsageMetadata.PromptTokenCount
		}
	}
	// Populate duration metadata (in nanoseconds)
	elapsed := time.Since(start).Nanoseconds()
	resp.TotalDuration = elapsed
	resp.LoadDuration = 0
	resp.PromptEvalDuration = 0
	resp.EvalDuration = elapsed
	if err := json.NewEncoder(w).Encode(&resp); err != nil {
		internal.ErrorHandler(w, r, http.StatusInternalServerError, "failed to encode chat response: %v", err)
		return
	}
}

type GenerateRequest struct {
	Model   string  `json:"model,omitempty"`
	Prompt  string  `json:"prompt,omitempty"`
	Suffix  string  `json:"suffix,omitempty"`
	Options Options `json:"options,omitempty"`
	System  string  `json:"system,omitempty"`

	// TODO: Support images.
	// TODO: Support format.
	// TODO: Support streaming.
}

type GenerateResponse struct {
	Model     string    `json:"model,omitempty"`
	Response  string    `json:"response,omitempty"`
	CreatedAt time.Time `json:"created_at,omitempty"`

	PromptEvalCount int32 `json:"prompt_eval_count,omitempty"`
	EvalCount       int32 `json:"eval_count,omitempty"`

	Done bool `json:"done,omitempty"`
}

type Options struct {
	Temperature *float32 `json:"temperature,omitempty"`
	Stop        *string  `json:"stop,omitempty"`
	NumPredict  *int32   `json:"num_predict,omitempty"`
	TopK        *int32   `json:"top_k,omitempty"`
	TopP        *float32 `json:"top_p,omitempty"`

	// TODO: Anything else to support?
}

type EmbedRequest struct {
	Model string   `json:"model,omitempty"`
	Input []string `json:"input,omitempty"`
}

type EmbedResponse struct {
	Model      string      `json:"model,omitempty"`
	Embeddings [][]float32 `json:"embeddings,omitempty"`
}
