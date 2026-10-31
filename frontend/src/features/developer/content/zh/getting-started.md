# 快速开始

Sub2API 提供统一的 AI API 网关。客户端使用兼容 OpenAI、Anthropic 或 Gemini 的协议，并通过一个 Sub2API API Key 发起请求。

## 准备工作

1. 在控制台创建 API Key。
2. 确认 Key 所属分组包含需要使用的模型。
3. 使用 HTTPS 发送请求，并在 Authorization 头中使用 Bearer 方案。

~~~http
Authorization: Bearer <your-api-key>
Content-Type: application/json
~~~

## 第一条请求

~~~bash
curl https://your-api-host/v1/chat/completions \
  -H "Authorization: Bearer $API_KEY" \
  -H "Content-Type: application/json" \
  -d '{"model":"your-model","messages":[{"role":"user","content":"Hello"}]}'
~~~

## 下一步

- 阅读认证与错误处理指南。
- 在 Playground 中先使用 Mock 模式检查请求结构。
- 确认真实请求可能产生费用后，再切换到 Live 模式。

> 实际可用模型、配额和限流由当前 Sub2API 实例及 API Key 分组决定。
