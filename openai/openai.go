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

// Package openai implements HTTP handlers that implements
// the OpenAI API and make calls to Gemini models.
package openai

import (
	"github.com/google/generative-ai-go/genai"
	"github.com/gorilla/mux"
)

// handlers provides various HTTP handlers
// to transform OpenAI protocol to Gemini calls.
type handlers struct {
	geminiClient *genai.Client
}

// RegisterHandlers registers the HTTP handlers on the mux.
func RegisterHandlers(r *mux.Router, geminiClient *genai.Client) {
	handlers := &handlers{geminiClient: geminiClient}
	r.HandleFunc("/v1/embeddings", handlers.EmbeddingsHandler)
	r.HandleFunc("/v1/chat/completions", handlers.ChatCompletionsHandler)
}

type EmbeddingsRequest struct {
	Model string   `json:"model"`
	Input []string `json:"input"`
	User  string   `json:"user,omitempty"`
}

type EmbeddingsResponse struct {
	Object string          `json:"object"`
	Data   []EmbeddingData `json:"data"`
	Model  string          `json:"model"`
	Usage  Usage           `json:"usage"`
	Error  interface{}     `json:"error,omitempty"`
}

type EmbeddingData struct {
	Object    string    `json:"object"`
	Embedding []float32 `json:"embedding"`
	Index     int       `json:"index"`
}

type Usage struct {
	PromptTokens     int32 `json:"prompt_tokens,omitempty"`
	TotalTokens      int32 `json:"total_tokens,omitempty"`
	CompletionTokens int32 `json:"completion_tokens,omitempty"`
}

type ToolFunctionParamsProperties map[string]struct {
	Type        string `json:"type,omitempty"`
	Description string `json:"description,omitempty"`
}

type ToolFunctionParams struct {
	Type                 string                       `json:"type,omitempty"`
	Properties           ToolFunctionParamsProperties `json:"properties,omitempty"`
	Required             []string                     `json:"required,omitempty"`
	AdditionalParameters bool                         `json:"additional_parameters,omitempty"`
}

type ToolFunction struct {
	Name        string             `json:"name,omitempty"`
	Description string             `json:"description,omitempty"`
	Parameters  ToolFunctionParams `json:"parameters,omitempty"`
	Strict      bool               `json:"strict,omitempty"`
}

type Tool struct {
	Type     string       `json:"type,omitempty"`
	Function ToolFunction `json:"function,omitempty"`
}

type ChatCompletionRequest struct {
	// TODO: Support response_format
	// TODO: Add logit bias and logprobs/top_logprobs
	// TODO: Support tools
	// TODO: Support Stop to be string only
	Model    string        `json:"model"`
	Messages []ChatMessage `json:"messages"`

	Stream        bool          `json:"stream,omitempty"`
	StreamOptions StreamOptions `json:"stream_options,omitempty"`

	N                *int32   `json:"n,omitempty"`
	Stop             []string `json:"stop,omitempty"`
	MaxTokens        *int32   `json:"max_tokens,omitempty"`
	FrequencyPenalty *float32 `json:"frequency_penalty,omitempty"`
	PresencePenalty  *float32 `json:"presence_penalty,omitempty"`
	Temperature      *float32 `json:"temperature,omitempty"`
	TopP             *float32 `json:"top_p,omitempty"`

	Tools             []Tool `json:"tools,omitempty"`
	ToolChoice        string `json:"tool_choice,omitempty"`
	ParallelToolCalls bool   `json:"parallel_tool_calls,omitempty"`

	User string `json:"user,omitempty"`
}

type ChatMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type ChatCompletionResponse struct {
	ID      string                 `json:"id,omitempty"`
	Object  string                 `json:"object,omitempty"`
	Created int64                  `json:"created,omitempty"`
	Model   string                 `json:"model,omitempty"`
	Choices []ChatCompletionChoice `json:"choices,omitempty"`
	Usage   Usage                  `json:"usage,omitempty"`
}

type ChatCompletionChoice struct {
	Index        int         `json:"index"`
	Message      ChatMessage `json:"message"`
	FinishReason string      `json:"finish_reason"`
}

type StreamOptions struct {
	IncludeUsage bool `json:"include_usage,omitempty"`
}
