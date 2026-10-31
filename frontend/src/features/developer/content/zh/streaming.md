# 流式响应

Chat Completions 和 Responses 接口支持请求体中的 stream 参数。服务端通过 SSE 返回增量事件。

~~~json
{
  "model": "your-model",
  "stream": true,
  "messages": [{"role": "user", "content": "Write a short poem"}]
}
~~~

## 处理边界

- 逐块读取响应，不要假设每个网络 chunk 都恰好包含一条完整事件。
- 保留跨 chunk 的缓冲区，按空行拆分事件。
- 用户取消时中止请求并释放读取器。
- 对超时、网络错误和格式错误的流分别处理。

Playground 的 Live 模式会显示流式文本，并提供停止按钮。流式内容只保留在当前页面状态中。
