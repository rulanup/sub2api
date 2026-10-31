# 客户端配置

Sub2API 网关支持多个协议入口。选择客户端原生支持的协议可以减少转换复杂度。

| 协议 | 入口示例 |
| --- | --- |
| OpenAI Chat Completions | /v1/chat/completions |
| OpenAI Responses | /v1/responses |
| Anthropic Messages | /v1/messages |
| Gemini | /v1beta/models |

## Base URL

通常将客户端 Base URL 设置为实例根地址。部分客户端会自动追加 /v1，此时不要重复添加版本段。

## 环境

Request Workspace 支持 Development、Staging 和 Production 三个本地环境标签。环境标签不会保存 API Key、Authorization、Cookie 或完整敏感请求正文。
