# Getting started

Sub2API provides a unified AI API gateway. Clients can use OpenAI-, Anthropic-, or Gemini-compatible protocols with one Sub2API API key.

## Before you begin

1. Create an API key in the console.
2. Confirm that the key's group includes the models you need.
3. Send requests over HTTPS and pass the key with the Bearer scheme.

~~~http
Authorization: Bearer <your-api-key>
Content-Type: application/json
~~~

## Your first request

~~~bash
curl https://your-api-host/v1/chat/completions \
  -H "Authorization: Bearer $API_KEY" \
  -H "Content-Type: application/json" \
  -d '{"model":"your-model","messages":[{"role":"user","content":"Hello"}]}'
~~~

## Next steps

- Read the authentication and error-handling guides.
- Use Mock mode in the Playground to validate the request shape.
- Switch to Live mode only after acknowledging that real requests may incur charges.

> Available models, quotas, and rate limits depend on the current Sub2API instance and API key group.
