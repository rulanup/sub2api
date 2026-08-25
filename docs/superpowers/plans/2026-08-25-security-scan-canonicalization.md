# 安全扫描多层编码归一化实现计划

> **面向 AI 代理的工作者：** 必需子技能：使用 superpowers:executing-plans 逐任务实现此计划。步骤使用复选框（`- [ ]`）语法来跟踪进度。

**目标：** 让本地安全策略和外部内容审核识别 Base64、HTML、URL、Unicode、零宽和全角混淆，同时不修改主上游原始请求。

**架构：** 在 `internal/pkg/securitytext` 提供有界的扫描副本归一化函数；`securityaudit` 用它做本地硬规则和风险路由前检查，`service` 用它构造发送给审核模型的文本。解码失败保持原文，节点异常继续由现有 fail-open 路径处理。

**技术栈：** Go、`golang.org/x/text/unicode/norm`、现有 `testify/require` 测试、现有 Qwen3Guard Chat 适配。

---

### 任务 1：共享扫描文本归一化

**文件：**
- 创建：`backend/internal/pkg/securitytext/canonicalize.go`
- 创建：`backend/internal/pkg/securitytext/canonicalize_test.go`

- [x] **步骤 1：编写失败测试**

  覆盖真实文本、Base64 片段、HTML 十进制/十六进制实体、URL UTF-8 百分号编码、字面量 `\\uXXXX`/`\\UXXXXXXXX`、零宽字符、全角字符，以及 Base64 → HTML 实体 → 百分号 → `\\uXXXX` → 中文的交错编码；断言归一化文本和紧凑文本一致。加入输入长度上限、固定点停止、大小写保留、不可读 Base64 和无效编码保留原文测试。

- [x] **步骤 2：运行测试确认失败**

  运行：`cd backend && go test ./internal/pkg/securitytext -run TestCanonicalize -count=1`

  预期：因包和函数尚未创建而失败。

- [x] **步骤 3：实现最少归一化代码**

  提供 `Canonicalize(input string) Result`，Result 包含 `Text` 和 `Compact`。按最多 4 轮固定点循环执行 HTML 实体、逐个合法 `%HH` 片段、字面量 Unicode 解码和有界可读 Base64 片段解码；每轮检查字节上限，文本不再变化时提前停止。随后应用 NFKC、去除 Unicode `Cf`/不可见格式字符、全角 ASCII 折叠、空白折叠，并生成删除分隔空白且大小写折叠后的紧凑文本。不要写回请求体。

- [x] **步骤 4：运行测试确认通过**

  运行：`cd backend && go test ./internal/pkg/securitytext -run TestCanonicalize -count=1`

- [x] **步骤 5：Commit**

  运行：`git add backend/internal/pkg/securitytext && git commit -m "feat: canonicalize encoded security scan text"`

### 任务 2：本地网络安全策略接入归一化文本

**文件：**
- 修改：`backend/internal/securityaudit/prompt_local_policy.go`
- 修改：`backend/internal/securityaudit/prompt_local_policy_test.go`

- [x] **步骤 1：编写失败测试**

  在现有高风险样例旁增加 HTML、URL、字面量 Unicode、零宽、全角和混合编码的 jailbreak/攻击工具输入；断言仍为 `Blocked`，类别为 `jailbreak` 或对应网络攻击类别。增加“防御/授权语境经过编码”的测试，断言不直接 Block 且 `NeedsAI` 为 true。

- [x] **步骤 2：运行测试确认失败**

  运行：`cd backend && go test ./internal/securityaudit -run 'TestLocalSecurityPolicy' -count=1`

  预期：编码样例当前不会命中本地规则，测试失败。

- [x] **步骤 3：接入归一化结果并保留现有规则**

  用共享 `securitytext.Canonicalize` 生成 normalized/compact 文本，再执行现有 action/target/ambiguous 规则。补充“开发者模式、攻击工具、攻击脚本、恶意工具、安全规则”等必要术语，仍使用保护性语境优先转 `NeedsAI` 的逻辑。不要把所有包含“漏洞/模型”的文本无条件封禁。

- [x] **步骤 4：运行测试确认通过**

  运行：`cd backend && go test ./internal/securityaudit -run 'TestLocalSecurityPolicy' -count=1`

- [x] **步骤 5：Commit**

  运行：`git add backend/internal/securityaudit && git commit -m "fix: block encoded network abuse prompts locally"`

### 任务 3：内容审核模型使用归一化副本

**文件：**
- 修改：`backend/internal/service/content_moderation_input.go`
- 修改：`backend/internal/service/content_moderation_input_test.go`
- 修改：`backend/internal/service/content_moderation_qwen3guard_test.go`

- [x] **步骤 1：编写失败测试**

  断言 `ContentModerationInput.Normalize` 会把编码混淆转成模型可识别的扫描文本，并保留图片和原始网关 body 不变。

- [x] **步骤 2：运行测试确认失败**

  运行：`cd backend && go test ./internal/service -run 'TestContentModerationInput|TestContentModerationQwen3Guard' -count=1`

  预期：当前 Normalize 只折叠空白，编码样例断言失败。

- [x] **步骤 3：实现最少接入**

  在 `ContentModerationInput.Normalize` 中对 Text 使用共享归一化 Text；保持 `ModerationInput` 只返回审核副本，不能修改 `ContentModerationCheckInput.Body`。

- [x] **步骤 4：运行目标测试确认通过**

  运行：`cd backend && go test ./internal/service -run 'TestContentModerationInput|TestContentModerationQwen3Guard' -count=1`

- [x] **步骤 5：Commit**

  运行：`git add backend/internal/service && git commit -m "fix: normalize encoded moderation inputs"`

### 任务 4：集成验证与交付

**文件：**
- 修改：`docs/superpowers/specs/2026-08-25-security-scan-canonicalization-design.md`
- 修改：`docs/superpowers/plans/2026-08-25-security-scan-canonicalization.md`

- [x] **步骤 1：运行相关完整测试**

  运行：`cd backend && go test ./internal/pkg/securitytext ./internal/securityaudit ./internal/service -count=1`

- [x] **步骤 2：运行全后端测试**

  运行：`cd backend && go test ./... -count=1`

- [x] **步骤 3：检查差异和主上游保护**

  运行：`git diff --check && git status --short`

  确认没有修改请求转发 body、没有加入任何上游 PR、没有输出或记录 API Key。

- [x] **步骤 4：Commit**

  运行：`git add docs/superpowers/specs/2026-08-25-security-scan-canonicalization-design.md docs/superpowers/plans/2026-08-25-security-scan-canonicalization.md && git commit -m "docs: plan encoded security scan hardening"`
