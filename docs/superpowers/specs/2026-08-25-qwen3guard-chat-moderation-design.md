# Qwen3Guard Chat Completions 审核适配设计

状态：已获初步方案批准，等待实现前的书面规格审阅

日期：2026-08-25

## 1. 背景与现状

当前内容审核服务将审核请求固定发送到 `{base_url}/v1/moderations`，并按 OpenAI Moderation 的 `flagged` 与 `category_scores` 响应处理。Gitee AI 返回“暂不支持该接口”，因为它提供的是 OpenAI 兼容的 `/v1/chat/completions`，而不是 `/v1/moderations`。

Gitee 已提供 Qwen3Guard-Gen-8B 等安全审查模型。Qwen3Guard-Gen 官方模型卡将审查结果定义为 `Safe`、`Unsafe`、`Controversial` 三档，并给出 `Safety: ...`、`Categories: ...` 的文本输出格式：

- Gitee Chat Completions 接口：[https://ai.gitee.com/solutions/serverless](https://ai.gitee.com/solutions/serverless)
- Gitee Qwen3Guard 模型列表：[https://ai.gitee.com/serverless-api?model=jina-embeddings-v4](https://ai.gitee.com/serverless-api?model=jina-embeddings-v4)
- Qwen3Guard-Gen-8B 模型卡：[https://huggingface.co/Qwen/Qwen3Guard-Gen-8B](https://huggingface.co/Qwen/Qwen3Guard-Gen-8B)

## 2. 目标

1. 保留现有 OpenAI Moderation 兼容能力。
2. 增加 Qwen3Guard Chat Completions 审核协议，使 Gitee AI 可作为独立审核节点。
3. 将 Qwen3Guard 的三档结果转换为现有内容审核的统一决策、日志、哈希和用户风险处理链路。
4. 主上游请求只在审核结果允许或审核节点不可用时继续；明确判定为风险的输入不能发送到主上游。
5. 遵循既有运行策略：审核节点超时、连接失败、HTTP 错误或输出格式错误时，记录异常并通知管理员，但不阻断正常请求。
6. 不把审核 Key 写入配置响应、日志、错误消息或前端状态明文。

## 3. 非目标

- 不把 Gitee 的 Chat Completions 接口伪装成 OpenAI `/v1/moderations`。
- 不让普通聊天模型默认承担安全审核职责；首期只针对明确配置的 Qwen3Guard Chat 模式。
- 不在本期实现 Qwen3Guard-Stream 的流式逐 token 审核。
- 不改变主模型上游的路由、计费、重试和流式协议。
- 不使用真实违法操作指令做自动化测试；测试使用合成响应、无害输入和脱敏样本。

## 4. 配置设计

在现有内容审核配置中增加审核协议字段，默认保持向后兼容：

```json
{
  "protocol": "openai_moderation",
  "base_url": "https://api.openai.com",
  "model": "omni-moderation-latest",
  "controversial_action": "allow"
}
```

支持的协议：

- `openai_moderation`：现有默认值，请求 `{base_url}/v1/moderations`。
- `qwen3guard_chat`：请求 `{base_url}/v1/chat/completions`，推荐 Base URL 为 `https://ai.gitee.com`，模型由管理员填写，例如 `Qwen3Guard-Gen-8B`。

`controversial_action` 取值：

- `allow`：Controversial 记录风险但不拦截，默认值，避免上下文正常的用户请求被过度阻断。
- `block`：Controversial 与 Unsafe 一样拦截，用于严格风控场景。

配置接口、前端表单、配置视图和配置复制逻辑都必须透传该字段；旧配置缺少字段时按 `openai_moderation` 与 `allow` 解析。

## 5. 请求与解析流程

### 5.1 OpenAI Moderation 模式

保持现有请求和响应逻辑不变。

### 5.2 Qwen3Guard Chat 模式

1. 从网关请求中提取已规范化的文本内容。
2. 首期只发送一个 `user` 消息，消息内容为待审核文本，避免额外的系统提示改变 Qwen3Guard 官方审查模板语义。
3. 请求体使用非流式、低输出长度、确定性参数：

```json
{
  "model": "Qwen3Guard-Gen-8B",
  "messages": [{"role": "user", "content": "<审核文本>"}],
  "stream": false,
  "temperature": 0,
  "max_tokens": 128
}
```

4. 从 `choices[0].message.content` 读取结果。
5. 解析大小写、空格和代码块包裹的容错形式，但只接受明确的 `Safety: Safe|Unsafe|Controversial` 标签；缺少标签或出现无法识别的标签视为审核节点异常。
6. 从 `Categories:` 读取逗号分隔类别，并映射到内部类别：

| Qwen3Guard 类别 | 内部类别 |
|---|---|
| Violent | violence |
| Non-violent Illegal Acts | illicit |
| Sexual Content or Sexual Acts | sexual |
| PII | pii |
| Suicide & Self-Harm | self-harm |
| Unethical Acts | unethical |
| Jailbreak | jailbreak |
| Copyright Violation | copyright |
| Politically Sensitive Topics | political |
| None | 无类别 |

7. 为兼容现有评分链路，将标签转换为离散分数：Safe 为 0、Controversial 为 0.5、Unsafe 为 1.0；最高类别使用对应类别，必要时保留原始 Qwen 标签供审核结果和测试接口展示。
8. `Unsafe` 始终视为命中；`Controversial` 按 `controversial_action` 决定是否命中；`Safe` 放行。

### 5.3 图片输入

Qwen3Guard Chat 首期仅支持文本审核。检测到图片输入时：

- 不把图片 base64 强行塞入文本审核模型。
- 记录“审核节点不支持图片”的审核异常并通知管理员。
- 按现有审核节点不可用策略放行，建议管理员对包含图片的端点继续使用 OpenAI Moderation 或其他视觉审核节点。

## 6. 错误处理与安全边界

- 明确的 `Unsafe` 或配置为拦截的 `Controversial`：返回现有网络安全策略拦截响应，不调用主上游。
- Gitee HTTP 400/401/403/429/5xx、超时、网络断开、空响应、无 `choices`、标签解析失败：记录 `error`，更新节点健康状态，发送管理员通知；不阻断本次正常请求。
- 400 “暂不支持该接口”应在审核测试界面显示为“审核协议与节点不匹配”，并建议切换到 `qwen3guard_chat`，不把它误判为 API Key 冻结。
- 审核请求可以发送到管理员配置的 Gitee 审核节点；被判定为风险的原始输入不得继续发送到主模型上游。
- 错误日志和内容审核日志继续使用现有密钥脱敏、输入摘要脱敏和长度限制。

## 7. 前端交互

在“内容审计设置 → 基础”中增加：

- “审核协议”下拉框。
- 选择 `qwen3guard_chat` 时展示 Gitee Base URL、模型示例和文本输入限制说明。
- 展示 Controversial 处理方式下拉框。
- 审核测试结果显示原始安全标签、类别、映射后的最高类别和是否拦截。
- 对 400 不支持接口显示可操作提示，而不是只显示原始 JSON。

OpenAI 模式的现有字段和交互保持不变。

## 8. 测试设计

### 后端单元测试

- Qwen3Guard `Safe`、`Unsafe`、`Controversial` 三种文本解析。
- 类别映射、未知类别、空类别和代码块包裹响应。
- 缺少 Safety 标签、空 choices、非文本 content 视为错误。
- `controversial_action=allow/block` 的统一决策。
- Qwen Chat 请求路径、请求体参数和 Authorization 头。
- 明确命中时不调用主上游；审核节点异常时按现有策略放行。
- 图片输入在 Qwen Chat 模式下生成可观测审核错误。
- 旧 JSON 配置无 `protocol` 字段时仍使用 OpenAI 模式。

### 前端测试

- 协议字段回显、切换默认值和保存 payload。
- Qwen 测试结果标签与错误提示渲染。
- 现有 OpenAI 审核配置测试保持通过。

### 集成验证

- 使用本地 httptest 模拟 Gitee Chat Completions，使用无害输入验证 Safe 放行。
- 使用本地 httptest 返回合成 `Safety: Unsafe` 验证拦截且不调用主上游。
- 使用本地 httptest 返回 400/超时/格式错误验证通知和放行。
- 不向真实 Gitee 或主上游发送真实违法操作指令。

## 9. 兼容性与发布

- 配置读取向后兼容，默认仍为 OpenAI Moderation。
- 数据库不新增表；仅扩展已有 JSON 配置和审核结果传输字段。
- 先提交设计文档，再创建实现计划；实现完成后运行后端内容审核测试、前端风险控制测试和构建工作流。
