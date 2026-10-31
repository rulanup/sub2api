# 发送请求

以下示例使用 OpenAI 兼容的 Chat Completions 接口。将主机地址和模型名替换为当前实例实际提供的值。

## cURL

~~~bash
curl https://your-api-host/v1/chat/completions \
  -H "Authorization: Bearer $API_KEY" \
  -H "Content-Type: application/json" \
  -d '{"model":"your-model","messages":[{"role":"user","content":"你好"}]}'
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
        "messages": [{"role": "user", "content": "你好"}],
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
