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
	"io"
	"net/http"

	"github.com/google-gemini/proxy-to-gemini/internal"
	"github.com/google/generative-ai-go/genai"
)

func (h *handlers) EmbeddingsHandler(w http.ResponseWriter, r *http.Request) {
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

	var embeddingsReq EmbeddingsRequest
	if err := json.Unmarshal(body, &embeddingsReq); err != nil {
		internal.ErrorHandler(w, r, http.StatusInternalServerError, "failed to unmarshal request body: %v", err)
		return
	}

	model := h.geminiClient.EmbeddingModel(embeddingsReq.Model)
	batch := model.NewBatch()
	for _, content := range embeddingsReq.Input {
		batch.AddContent(genai.Text(content))
	}

	geminiResp, err := model.BatchEmbedContents(r.Context(), batch)
	if err != nil {
		internal.ErrorHandler(w, r, http.StatusInternalServerError, "failed to make embeddings request: %v", err)
		return
	}

	embeddingsResp := &EmbeddingsResponse{
		Object: "list",
		Model:  embeddingsReq.Model,
		Data:   make([]EmbeddingData, 0, len(geminiResp.Embeddings)),
	}
	for i, contentEmbedding := range geminiResp.Embeddings {
		embeddingsResp.Data = append(embeddingsResp.Data, EmbeddingData{
			Index:     i,
			Object:    "embedding",
			Embedding: contentEmbedding.Values,
		})
	}
	if err := json.NewEncoder(w).Encode(embeddingsResp); err != nil {
		internal.ErrorHandler(w, r, http.StatusInternalServerError, "failed to encode embeddings response: %v", err)
		return
	}
}
