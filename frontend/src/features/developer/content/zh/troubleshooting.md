# 故障排查

## 浏览器显示 Failed to fetch

这通常表示网络错误、TLS 问题或被 CORS 拒绝。浏览器不会向 JavaScript 暴露被 CORS 阻止响应的完整细节。检查浏览器 Network 面板和目标服务的 Access-Control-Allow-Origin 配置。

## 401 或 403

确认 Key 使用 Bearer 方案、没有多余空格，并属于允许当前模型的分组。不要在 URL 中传递 Key。

## 请求超时

检查目标地址、网络、网关负载和上游耗时。Playground 默认设置请求超时，用户可以主动停止请求。

## 流式响应中断

检查网络是否稳定、是否主动取消，以及服务端是否返回了结构化错误事件。保留 Request ID，并在 Debug Center 中查看脱敏报告。
