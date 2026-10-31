# Make a request

The examples below use the OpenAI-compatible Chat Completions endpoint. Replace the host and model with values that the current instance actually exposes.

## cURL

~~~bash
curl https://your-api-host/v1/chat/completions \
  -H "Authorization: Bearer $API_KEY" \
  -H "Content-Type: application/json" \
  -d '{"model":"your-model","messages":[{"role":"user","content":"Hello"}]}'
~~~

## Python

~~~python
import os
import requests

response = requests.post(
    "https://your-api-host/v1/chat/completions",
    headers={"Authorization": f"Bearer {os.environ['API_KEY']}"},
    json={
        "model": "your-model",
        "messages": [{"role": "user", "content": "Hello"}],
    },
    timeout=60,
)
response.raise_for_status()
print(response.json())
~~~

## JavaScript

~~~javascript
const response = await fetch("https://your-api-host/v1/chat/completions", {
  method: "POST",
  headers: {
    Authorization: "Bearer " + process.env.API_KEY,
    "Content-Type": "application/json",
  },
  body: JSON.stringify({
    model: "your-model",
    messages: [{ role: "user", content: "Hello" }],
  }),
});

if (!response.ok) throw new Error(await response.text());
console.log(await response.json());
~~~
