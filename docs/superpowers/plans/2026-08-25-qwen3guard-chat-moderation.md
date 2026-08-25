# Qwen3Guard Chat Completions 内容审核实现计划

> **面向 AI 代理的工作者：** 必需子技能：使用 `superpowers:subagent-driven-development`（推荐）或 `superpowers:executing-plans`（如果内联执行）逐任务实现此计划。步骤使用复选框（`- [ ]`）语法来跟踪进度。

**目标：** 在保留现有 OpenAI Moderation API 能力、风险评分、命中日志、自动封禁和 fail-open 节点故障策略的前提下，增加 Gitee AI 的 Qwen3Guard Chat Completions 审计协议。Qwen3Guard 返回 Unsafe 时在主请求发送到上游前拦截；返回 Controversial 时按管理员配置默认放行或选择拦截；审计节点超时、不可用、接口错误或图片能力不支持时记录可见的管理员诊断信息并放行主请求。

**架构：** 把 Qwen3Guard 适配限制在现有 `ContentModerationService` 的审计调用边界内。配置新增协议和 Controversial 行为，OpenAI 协议保持默认和完全兼容。Qwen 文本结果统一转换成现有分数/阈值模型，因此既有同步 pre-block、异步审计、风险记录和 API key 轮询继续复用；协议错误不会进入主 AI 上游调用，也不会因为审计节点故障阻断正常用户请求。测试接口返回协议、严重级别、分类和可操作错误信息，前端据此显示配置提示与测试结果。

**技术栈：** Go、Gin、现有 `ContentModerationService`、Go `httptest`、Vue 3、TypeScript、Naive UI、Vitest、pnpm、Docker Compose。

---

## 文件与职责

- `backend/internal/service/content_moderation.go`：新增协议/Controversial 配置字段、默认值、校验、配置视图、请求分派、Qwen 结果映射，以及测试结果和运行状态中的诊断字段；保持无代理模式绕过进程代理、显式代理继续生效。
- `backend/internal/service/content_moderation_qwen3guard_test.go`：Qwen3Guard 文本解析、请求体、响应映射、Safe/Unsafe/Controversial 行为、异常 fail-open 的单元测试。
- `backend/internal/service/content_moderation_proxy_test.go`：保留并扩展直连模式回归覆盖，确保新增协议不重新引入环境代理问题。
- `backend/internal/handler/admin/content_moderation_handler.go`：透传协议和 Controversial 行为到 service，透传测试请求的协议覆盖字段。
- `frontend/src/api/admin/riskControl.ts`：新增协议、Controversial 行为、Qwen 测试结果和审计诊断的类型。
- `frontend/src/views/admin/RiskControlView.vue`：增加协议选择、Qwen 默认模型提示、Controversial 行为选择、测试结果和 400 不支持接口的友好提示。
- `frontend/src/i18n/locales/zh/admin/channels.ts`、`frontend/src/i18n/locales/en/admin/channels.ts`：新增中英文文案。
- `frontend/src/views/admin/__tests__/RiskControlView.spec.ts`：覆盖协议表单提交、Qwen 测试结果和错误提示。
- `frontend/src/i18n/__tests__/riskControlLocales.spec.ts`：保证新增文案在中英文 locale 中同时存在。
- `docs/superpowers/specs/2026-08-25-qwen3guard-chat-moderation-design.md`：已确认的设计规格；实现过程中只在发现契约冲突时同步修正。

## 实现任务

### 1. 先建立 Qwen3Guard 解析器的失败测试

- [ ] 在 `backend/internal/service/content_moderation_qwen3guard_test.go` 编写 `TestQwen3GuardParseText`：覆盖官方格式 `Safety: Safe`、`Safety: Unsafe`、`Safety: Controversial`，覆盖 `Categories:` 的多分类、`None`、大小写/空白，以及包裹在 Markdown code fence 中的返回文本。
- [ ] 编写未知 Safety、缺少 Safety、空响应的失败用例，断言返回可诊断错误而不是默认为安全。
- [ ] 编写分类映射表测试，至少验证 `Non-violent Illegal Acts -> illicit`、`Jailbreak -> jailbreak`、`PII -> pii`、`Suicide & Self-Harm -> self-harm`、`Politically Sensitive Topics -> political`。
- [ ] 执行红灯命令：

  ```bash
  cd /home/guili/sub2api/backend && go test ./internal/service -run 'TestQwen3GuardParse' -count=1
  ```

  预期：由于解析器和类型尚未实现，测试编译失败或测试失败；不得修改生产代码来绕过红灯。
- [ ] 在测试中固定合成响应文本，不使用真实危险提示词；完成后执行 `git diff --check`，将测试作为独立提交：

  ```bash
  git add backend/internal/service/content_moderation_qwen3guard_test.go && git commit -m "test: define qwen3guard moderation contract"
  ```

### 2. 实现 Qwen3Guard 解析和统一结果适配

- [ ] 在 `backend/internal/service/content_moderation.go` 增加内部协议常量、Qwen 请求/响应 DTO、解析结果结构和分类映射；解析必须区分 Safe、Unsafe、Controversial，未知或缺失 Safety 返回错误。
- [ ] 将 Safe 映射为兼容分数 `0`，Controversial 映射为 `0.5`，Unsafe 映射为 `1.0`；解析结果保留原始严重级别和规范化分类，供测试接口显示。
- [ ] 按测试驱动循环执行：先运行 `go test ./internal/service -run 'TestQwen3GuardParse' -count=1` 确认仍有失败，再实现最小解析器，重复运行直到全部通过。
- [ ] 增加协议请求测试前先写 `httptest.Server` 用例，断言请求为 `POST /v1/chat/completions`，请求体包含配置模型、单条 user message、`stream:false`、`temperature:0`、`max_tokens:128`，并携带 Bearer key；断言响应 `Safety: Unsafe` 被统一结果标记为 flagged。
- [ ] 实现请求适配器：对 Qwen 使用 `/v1/chat/completions`，从 `choices[0].message.content` 读取文本；响应结构缺失、HTTP 非 2xx、JSON 无法解析均返回带 HTTP 状态/协议上下文的错误。
- [ ] 运行 `go test ./internal/service -run 'TestQwen3Guard|TestContentModerationWithoutConfiguredProxy' -count=1`，预期全部通过后提交：

  ```bash
  git add backend/internal/service/content_moderation.go backend/internal/service/content_moderation_qwen3guard_test.go && git commit -m "feat: adapt qwen3guard chat moderation"
  ```

### 3. 接入配置、阈值行为和审计节点故障诊断

- [ ] 在 `ContentModerationConfig`、view、update input、测试 input、默认配置、解析/规范化、配置校验和持久化流程中新增 `protocol` 与 `controversial_action`；旧 JSON 缺失字段时分别回退到 `openai_moderation` 和 `allow`。
- [ ] 校验协议只允许 `openai_moderation`、`qwen3guard_chat`，Controversial 行为只允许 `allow`、`block`；Qwen 未填写模型时默认 `Qwen3Guard-Gen-8B`，OpenAI 默认值保持 `omni-moderation-latest`。
- [ ] 在 `callModerationOnceWithInput` 按协议分派 endpoint、请求 DTO 和响应 decoder；保持现有 OpenAI 文本/图片输入格式不变。Qwen 首版只发送文本，若输入包含图片则返回明确的 unsupported-image 审计错误，并沿用 fail-open 策略，同时写入 `content_moderation.audit_api_failed` 结构化日志供管理员查看。
- [ ] 在同步检查路径中将 Qwen Unsafe 按现有 pre-block 逻辑拦截；Controversial 只有在配置为 `block` 时拦截，配置为 `allow` 时不设置 flagged，避免触发自动封禁和主请求阻断；审计节点错误、超时、unsupported-image 都允许主请求继续。
- [ ] 不把 Qwen 的 400“接口不支持”误判成 key 冻结条件；测试 API 返回协议、严重级别、分类和错误类别，管理员可以从测试响应与现有 status/log 页面定位节点问题。继续复用现有 `audit_api_failed` 日志，不引入数据库迁移或新的邮件依赖。
- [ ] 先写配置 round-trip、Qwen Controversial allow/block、Qwen image fail-open、HTTP 400 不冻结 key 和 synthetic Unsafe 不触达主上游的测试；运行：

  ```bash
  cd /home/guili/sub2api/backend && go test ./internal/service -run 'TestContentModeration|TestQwen3Guard' -count=1
  ```

  预期全部通过；如有失败，先定位既有 OpenAI 回归，再继续前端工作。通过后提交：

  ```bash
  git add backend/internal/service/content_moderation.go backend/internal/service/content_moderation_qwen3guard_test.go backend/internal/service/content_moderation_proxy_test.go && git commit -m "feat: configure moderation protocols and fail-open diagnostics"
  ```

### 4. 透传管理接口并先写前端失败测试

- [ ] 在 `backend/internal/handler/admin/content_moderation_handler.go` 的配置请求和测试请求中增加协议/Controversial 行为字段并完整透传；执行对应 handler/service 编译与测试。
- [ ] 在 `frontend/src/api/admin/riskControl.ts` 增加 `ModerationProtocol`、`ControversialAction`、配置字段、测试 payload 字段，以及 `severity`、`categories`、`error_type` 等测试结果字段，确保旧字段仍可选/兼容。
- [ ] 在 `frontend/src/views/admin/__tests__/RiskControlView.spec.ts` 先增加失败测试：选择 Qwen 协议后保存应发送 `protocol`、`model`、`controversial_action`；Qwen Unsafe 结果应展示严重级别/分类；400 unsupported-interface 响应应展示友好中文提示。
- [ ] 执行前端红灯命令：

  ```bash
  cd /home/guili/sub2api/frontend && pnpm exec vitest run src/views/admin/__tests__/RiskControlView.spec.ts
  ```

  预期新增断言失败，保留失败输出作为实现依据。

### 5. 完成前端配置界面、文案和测试

- [ ] 在 `frontend/src/views/admin/RiskControlView.vue` 添加协议下拉选项；切换 Qwen 时显示 Gitee Chat Completions endpoint 和 `Qwen3Guard-Gen-8B` 提示，保留用户自定义模型；添加 Controversial allow/block 选择，并将值放入保存请求。
- [ ] 将测试请求带上当前协议/行为；根据响应显示 Safe/Controversial/Unsafe、规范化分类、最高分和诊断错误。对 HTTP 400 且错误文本含 unsupported interface/暂不支持该接口的情况显示“当前审计节点不支持该接口，请改用 Chat Completions 协议或更换节点”，不显示为 API key 无效。
- [ ] 在中英文 `channels.ts` 添加所有新 key；在 `riskControlLocales.spec.ts` 添加中英文 key 对齐断言。
- [ ] 用 Node 20 容器执行 pnpm 测试（宿主 pnpm 路径不可直接用于 WSL 容器时使用项目 lockfile 的 pnpm 版本）：

  ```bash
  docker run --rm -v /home/guili/sub2api:/workspace -w /workspace/frontend node:20-bookworm bash -lc "corepack enable && corepack prepare pnpm@9 --activate >/dev/null && pnpm install --frozen-lockfile --offline >/dev/null && pnpm exec vitest run src/views/admin/__tests__/RiskControlView.spec.ts src/i18n/__tests__/riskControlLocales.spec.ts"
  ```

  预期新增测试和已有测试全部通过。执行 `git diff --check` 后提交：

  ```bash
  git add backend/internal/handler/admin/content_moderation_handler.go frontend/src/api/admin/riskControl.ts frontend/src/views/admin/RiskControlView.vue frontend/src/views/admin/__tests__/RiskControlView.spec.ts frontend/src/i18n/locales/zh/admin/channels.ts frontend/src/i18n/locales/en/admin/channels.ts frontend/src/i18n/__tests__/riskControlLocales.spec.ts && git commit -m "feat: expose qwen3guard moderation settings"
  ```

### 6. 集成验证、文档和 fork 分支交付

- [ ] 运行后端相关测试、前端相关测试和静态检查：

  ```bash
  cd /home/guili/sub2api/backend && go test ./internal/service -run 'TestContentModeration|TestQwen3Guard' -count=1
  cd /home/guili/sub2api/frontend && pnpm exec vitest run src/views/admin/__tests__/RiskControlView.spec.ts src/i18n/__tests__/riskControlLocales.spec.ts
  cd /home/guili/sub2api && git diff --check
  ```

- [ ] 重建隔离 fork 容器并确认 `http://127.0.0.1:8081/` 和 `/health` 返回 200；通过本地管理员账号只发送 benign 文本和 httptest 合成 Unsafe 响应，确认 Unsafe 在主 AI 请求前被拦截、审计超时允许主请求继续、日志包含 `content_moderation.audit_api_failed`。不使用真实违法提示词、不使用真实上游 key。
- [ ] 更新 `README.md` 的审计配置说明：OpenAI Moderation 与 Qwen3Guard Chat Completions 的 endpoint/model/图片限制、Controversial 行为、节点故障 fail-open 和 400 不支持接口处理。
- [ ] 检查 `git status --short`、`git log --oneline -8` 和提交内容，确认没有修改 upstream remote 或创建 PR；将 `fork-merged` 推送到 fork：

  ```bash
  git push fork fork-merged
  ```

- [ ] 最终报告提交哈希、测试命令和容器验证结果；如果 Docker/Gitee 环境不可用，只报告具体失败原因，不把未验证的状态写成成功。

## 交付前自检

- [ ] OpenAI Moderation 默认路径、图片输入、阈值和既有日志/自动封禁测试保持通过。
- [ ] Qwen Unsafe 会阻断主请求，Controversial 行为可配置，Safe 正常放行。
- [ ] Qwen 400、超时、异常 JSON、图片不支持不会阻断正常主请求，并能在管理员可见的测试结果、状态或结构化日志中定位。
- [ ] 协议和行为字段从前端到 handler、service、持久化 JSON 完整贯通，旧配置可安全读取。
- [ ] 前端中英文文案齐全，pnpm 测试通过，README 已同步。
- [ ] 所有变更只提交并推送 fork 分支，不向 upstream 创建合并请求。
