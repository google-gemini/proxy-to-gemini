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
	"net/http"

	"github.com/google-gemini/proxy-to-gemini/internal"
	"github.com/google/generative-ai-go/genai"
	"google.golang.org/api/iterator"
)

func streamingChatCompletionsHandler(w http.ResponseWriter, r *http.Request, model string, genModel *genai.GenerativeModel, parts []genai.Part) {
	iter := genModel.GenerateContentStream(r.Context(), parts...)

	encoder := json.NewEncoder(w)
	for {
		geminiResp, err := iter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			internal.ErrorHandler(w, r, http.StatusInternalServerError, "failed to stream response: %v", err)
			return
		}
		resp := toOpenAIResponse(geminiResp, "chat.completion.chunk", model)
		if err := encoder.Encode(resp); err != nil {
			internal.ErrorHandler(w, r, http.StatusInternalServerError, "failed to encode gemini response: %v", err)
			return
		}
	}
}