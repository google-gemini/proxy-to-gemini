# Quickstart

## OpenAI API

Obtain a Gemini API key from the [AI Studio](https://aistudio.google.com/).
Then set the following environmental variable to the key and run the proxy:

```sh
$ export GEMINI_API_KEY=<YOUR_API_KEY>
$ docker run -p 5555:5555 -e GEMINI_API_KEY=$GEMINI_API_KEY googlegemini/proxy-to-gemini
```

Set the following environment variable to the proxy:

```sh
$ export OPENAI_BASE_URL="http://127.0.0.1:5555/v1"
```

Then the OpenAI Python client library will use the proxy.
Save the following code in test.py:

```python
from openai import OpenAI

client = OpenAI()
chat_completion = client.chat.completions.create(
    model = "gemini-1.5-pro",
    messages=[
        {
            "role": "user",
            "content": "Say this is a test",
        }
    ],
)
print(chat_completion.chat_completion.choices)
```

Run the Python file:

```sh
$ python test.py
[Choice(finish_reason='stop', index=0, logprobs=None, message=ChatCompletionMessage(content='This is a test. \n', refusal=None, role='model', function_call=None, tool_calls=None))]
```