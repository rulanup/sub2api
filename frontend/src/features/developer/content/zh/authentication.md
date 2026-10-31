# 认证

网关接口使用 API Key 认证。将 Key 放在 Authorization 请求头中：

~~~http
Authorization: Bearer <your-api-key>
~~~

## 安全原则

- 不要把 API Key 提交到 Git、公开 Issue 或前端代码。
- 不要在 URL 查询参数中传递 Key。
- 为不同应用创建独立 Key，便于撤销和追踪用量。
- 生产环境只通过 HTTPS 发送请求。

## 浏览器调试

开发者门户的 Live Playground 只在当前页面内存中保存 Key。刷新、关闭页面或点击清除后，Key 会丢失，不会写入 localStorage、IndexedDB、Cookie 或 URL。

浏览器直连还受目标服务的 CORS 策略约束。Sub2API 不会为 Playground 默认提供携带用户凭据的公共代理。
