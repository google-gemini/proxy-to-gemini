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
	"reflect"
	"testing"

	"github.com/google/generative-ai-go/genai"
)

func Test_geminiToOpenAIResponse(t *testing.T) {
	tests := []struct {
		name   string
		from   *genai.GenerateContentResponse
		object string
		model  string
		want   ChatCompletionResponse
	}{
		{
			name: "basic",
			from: &genai.GenerateContentResponse{
				Candidates: []*genai.Candidate{
					{
						Index: 0,
						Content: &genai.Content{
							Parts: []genai.Part{
								genai.Text("I'm good, how are you?"),
							},
							Role: "model",
						},
					},
					{
						Index: 1,
						Content: &genai.Content{
							Parts: []genai.Part{
								genai.Text("Is there anything I can help with?"),
							},
							Role: "model",
						},
						FinishReason: genai.FinishReasonMaxTokens,
					},
				},
				UsageMetadata: &genai.UsageMetadata{
					PromptTokenCount:     123,
					CandidatesTokenCount: 456,
					TotalTokenCount:      789,
				},
			},
			object: "chat.completion",
			model:  "gemini1.5",
			want: ChatCompletionResponse{
				Object: "chat.completion",
				Model:  "gemini1.5",
				Choices: []ChatCompletionChoice{
					{
						Index: 0,
						Message: ChatMessage{
							Role:    "model",
							Content: "I'm good, how are you?",
						},
						FinishReason: "",
					},
					{
						Index: 1,
						Message: ChatMessage{
							Role:    "model",
							Content: "Is there anything I can help with?",
						},
						FinishReason: "length",
					},
				},
				Usage: Usage{
					PromptTokens:     123,
					CompletionTokens: 456,
					TotalTokens:      789,
				},
			},
		},
		{
			name: "no parts",
			from: &genai.GenerateContentResponse{
				Candidates: []*genai.Candidate{
					{
						Index: 0,
						Content: &genai.Content{
							Parts: []genai.Part{},
							Role:  "model",
						},
					},
				},
			},
			object: "chat.completion",
			model:  "gemini1.5",
			want: ChatCompletionResponse{
				Object: "chat.completion",
				Model:  "gemini1.5",
				Choices: []ChatCompletionChoice{
					{
						Index: 0,
						Message: ChatMessage{
							Role:    "model",
							Content: "",
						},
					},
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := toOpenAIResponse(tt.from, tt.object, tt.model)
			got.Created = tt.want.Created
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("geminiToOpenAIResponse() = %v, want %v", got, tt.want)
			}
		})
	}
}
