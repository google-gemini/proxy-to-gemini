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
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/google-gemini/proxy-to-gemini/internal"
	"github.com/google/generative-ai-go/genai"
	"github.com/gorilla/mux"
)

type handlers struct {
	client *genai.Client
}

func RegisterHandlers(r *mux.Router, client *genai.Client) {
	handlers := &handlers{client: client}
	r.HandleFunc("/api/generate", handlers.generateHandler)
	r.HandleFunc("/api/embed", handlers.embedHandler)
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
