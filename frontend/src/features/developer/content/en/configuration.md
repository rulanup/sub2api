# Client configuration

Sub2API exposes several protocol entry points. Use the protocol your client already supports to reduce translation complexity.

| Protocol | Example entry point |
| --- | --- |
| OpenAI Chat Completions | /v1/chat/completions |
| OpenAI Responses | /v1/responses |
| Anthropic Messages | /v1/messages |
| Gemini | /v1beta/models |

## Base URL

In most cases, set the client base URL to the instance root. Some clients append /v1 automatically; avoid duplicating the version segment.

## Environments

Request Workspace supports local Development, Staging, and Production labels. Environment labels never persist API keys, Authorization, cookies, or complete sensitive request bodies.
