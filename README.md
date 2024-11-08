# proxy-to-gemini

[![Go](https://github.com/google-gemini/proxy-to-gemini/actions/workflows/go.yml/badge.svg)](https://github.com/google-gemini/proxy-to-gemini/actions/workflows/go.yml)

> [!IMPORTANT]  
> Gemini API officially supports OpenAI API compatibility and this sidecar is no longer needed.
> See the [blog post](https://developers.googleblog.com/en/gemini-is-now-accessible-from-the-openai-library/) for more details!


A simple proxy server to access Gemini models by using other well-known APIs like OpenAI and Ollama.

## Setup

Obtain a Gemini API key from the [AI Studio](https://aistudio.google.com/).
Then set the following environmental variable to the key.

```sh
$ export GEMINI_API_KEY=<YOUR_API_KEY>
```

## Usage with OpenAI API

Run the binary:

```sh
$ docker run -p 5555:5555 -e GEMINI_API_KEY=$GEMINI_API_KEY googlegemini/proxy-to-gemini
2024/07/20 19:35:21 Starting server on :5555
```

Once server starts, you can access Gemini models through the proxy server
by using OpenAI API and client libraries.

``` sh
$ curl http://127.0.0.1:5555/v1/chat/completions \
  -H "Content-Type: application/json" \
  -d '{
    "model": "gemini-1.5-pro",
    "messages": [{"role": "user", "content": "Hello, world!"}]
  }'
{
  "object": "chat.completion",
  "created": 1721535029,
  "model": "gemini-1.5-pro",
  "choices": [
    {
      "index": 0,
      "message": {
        "role": "model",
        "content": "Hello back to you! \n\nIt's great to hear from you. What can I do for you today? \n"
      },
      "finish_reason": "stop"
    }
  ],
  "usage": {
    "prompt_tokens": 5,
    "total_tokens": 29,
    "completion_tokens": 24
  }
}
```

You can stream the chat responses:

```sh
$ curl http://127.0.0.1:5555/v1/chat/completions \
  -H "Content-Type: application/json" \
  -d '{
    "model": "gemini-1.5-pro",
    "messages": [{"role": "user", "content": "Hello, world!"}],
    "stream": true
  }'
data: {"object":"chat.completion.chunk","created":1725852986,"model":"gemini-1.5-pro","choices":[{"index":0,"message":{"role":"model","content":"Hello"},"finish_reason":"stop"}],"usage":{"prompt_tokens":5,"total_tokens":6,"completion_tokens":1}}
data: {"object":"chat.completion.chunk","created":1725852986,"model":"gemini-1.5-pro","choices":[{"index":0,"message":{"role":"model","content":" back! \n\nIt's nice to be greeted by the classic \""},"finish_reason":"stop"}],"usage":{"prompt_tokens":5,"total_tokens":21,"completion_tokens":16}}
data: {"object":"chat.completion.chunk","created":1725852987,"model":"gemini-1.5-pro","choices":[{"index":0,"message":{"role":"model","content":"Hello, world!\"  What can I help you with today? \n"},"finish_reason":"stop"}],"usage":{"prompt_tokens":5,"total_tokens":35,"completion_tokens":30}}
data: [DONE]
```

You can create embeddings:

```sh
$ curl http://127.0.0.1:5555/v1/embeddings \
  -H "Content-Type: application/json" \
  -d '{
    "model": "text-embedding-004",
    "input": ["hello"]
  }'
{
  "object": "list",
  "data": [
    {
      "object": "embedding",
      "embedding": [
        0.04824496,
        0.0117766075,
        -0.011552069,
        -0.018164534,
        -0.0026110192,
        0.05092675,
        ...
        0.0002852207,
        0.046413545
      ],
      "index": 0
    }
  ],
  "model": "text-embedding-004",
}
```

### Known OpenAI Limitations

* Only [chat completions](https://platform.openai.com/docs/api-reference/chat) and [embeddings](https://platform.openai.com/docs/api-reference/embeddings/create) are planned to be supported.
* Tool support is work in progress.
* Only text input and output is supported for now.
* response_format is not supported yet.

## Usage with Ollama API

``` sh
$ docker run -p 5555:5555 -e GEMINI_API_KEY=$GEMINI_API_KEY googlegemini/proxy-to-gemini -api=ollama
2024/07/20 19:35:21 Starting server on :5555
```
Once server starts, you can access Gemini models through the proxy server
by using Ollama API and client libraries.

``` sh
$ curl http://127.0.0.1:5555/api/generate \
  -H "Content-Type: application/json" \
  -d '{
    "model": "gemini-1.5-pro",
    "prompt": "Hello, how are you?"
  }'
{"model":"gemini-1.5-pro","response":"I'm doing well, thank you! As an AI, I don't have feelings, but I'm here and ready to assist you. \n\nHow can I help you today? \n","created_at":"2024-07-28T14:57:36.25261-07:00","prompt_eval_count":7,"eval_count":47,"done":true}
```

Create embeddings:

```sh
$ curl http://127.0.0.1:5555/api/embed \
  -H "Content-Type: application/json" \
  -d '{
    "model": "text-embedding-004",
    "input": ["hello"]
  }'
{"model":"text-embedding-004","embeddings":[[0.04824496,0.0117766075,-0.011552069,-0.018164534,-0.0026110192,0.05092675,0.08172899,0.007869772,0.054475933,0.026131334,-0.06593486,-0.002256868,0.038781915,...]]}
```

### Known Ollama Limitations
* Streaming is not yet supported.
* Images are not supported.
* Response format is not supported.
* Model parameters not supported by Gemini are ignored.

## Notes

The list of available models are listed at [Gemini API docs](https://ai.google.dev/gemini-api/docs/models/gemini).

This proxy is aiming users to try out the Gemini models easily. Hence,
it mainly supports text based use cases. Please refer to the Gemini SDKs
for media support.
