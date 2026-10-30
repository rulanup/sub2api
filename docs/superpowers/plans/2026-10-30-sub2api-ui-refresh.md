# sub2api 全站 UI 刷新实施计划

## 目标

按已批准方案 A「冷白大厂极简」，在不改变后端业务、路由、权限、状态管理、国际化、运行时主题和现有组件体系的前提下，统一收敛 sub2api 的视觉语言。优先通过设计令牌和通用组件覆盖全站，再处理代表性页面与高噪声残留，最后在远程 Linux 环境完成构建和真实浏览器验收。

## 架构

- 视觉契约集中在 `frontend/tailwind.config.js` 与 `frontend/src/style.css`，保留现有工具类、`primary-*` 别名和运行时 CSS 变量接口。
- 应用壳层继续沿用 `AppLayout`、`AppHeader`、`AppSidebar`、`AuthLayout` 和 `TablePageLayout`，只调整结构、类名和样式，不重写交互逻辑。
- 通用组件继续使用现有自研组件；不改 API，不增加运行时依赖。
- 页面级修改只处理代表性页面和全站装饰残留，不逐页随机重写。
- 新增一个基于源码文本的 Vitest 设计契约测试，锁定容易被后续样式改动破坏的关键规则；视觉质量最终由远程构建和浏览器检查验证。

## 技术栈

- Vue 3.5
- Vite 5
- TypeScript
- Tailwind CSS 3.4
- Pinia
- vue-router
- vue-i18n
- Vitest

## 规格

- `docs/superpowers/specs/2026-10-30-sub2api-ui-refresh-design.md`

## 全局约束

- 不修改后端 API、数据库、计费、权限或业务流程。
- 不引入 Ant Design、Element Plus 等第二套组件库。
- 不升级依赖，不在本地安装依赖。
- 保留浅色、深色、运行时品牌色、语言切换、响应式导航、表格虚拟滚动和移动端卡片行为。
- 所有本地编辑使用 `apply_patch`；不使用 Python 写文件。
- 工作分支为 `codex/ui-refresh`；未经授权不 push，不修改远程 `main`。
- 远程构建命令为 `cd /home/guili/go/sub2api/frontend && pnpm build`。
- 验收视口至少覆盖 1440px、1024px、768px、390px，并检查浅色、深色和关键浮层。

## Review Focus

- 全局令牌调整是否破坏运行时品牌色、深色模式、可读性或焦点可见性。
- 应用壳层和认证布局是否移除了 mesh、光球、玻璃与彩色发光，同时保留移动端抽屉、折叠侧栏和导航功能。
- 表格是否只在自身容器横向滚动，固定列、粘性表头、虚拟滚动和移动端卡片是否仍正常。
- 设置页中文页签是否不截断，窄屏是否可横向滚动。
- Dashboard 是否从同权 KPI 堆叠收敛为明确主次，同时保留所有数据和交互。
- 全站是否仍有代表性页面可见的 `glass`、`text-gradient`、`shadow-glow`、大圆角或彩虹装饰残留。
- 构建、类型检查、i18n 测试、设计契约测试和真实登录页面检查是否全部有可复现证据。

## 任务 1：建立设计契约测试与冷白令牌

### 目标

先用失败测试锁定全局视觉规则，再建立冷白中性画布、稳重品牌蓝、8 至 10px 圆角、微边框、轻阴影和可访问焦点态。

### 文件

- 新增：`frontend/src/__tests__/uiDesignContract.spec.ts`
- 修改：`frontend/tailwind.config.js`
- 修改：`frontend/src/style.css`
- 修改：`frontend/src/utils/theme.ts`（仅在默认主题或运行时变量需要同步时）

### 步骤

1. 写设计契约测试，读取 `style.css`、`tailwind.config.js` 和后续关键组件源码，断言：
   - 默认品牌色为产品蓝而非 Teal；
   - 全局卡片无装饰性阴影，常规圆角不超过 10px；
   - 存在 reduced-motion 规则；
   - 应用布局不再使用 `bg-mesh-gradient`；
   - 认证布局不再使用 `card-glass`、`shadow-glass` 或 `text-gradient`；
   - 表格容器包含横向滚动规则；
   - 设置页签不使用中文截断类。
2. 运行 `pnpm test:run -- uiDesignContract`，确认测试因当前视觉规则失败，记录失败断言。
3. 调整 Tailwind 色彩、圆角、阴影和字体层级；默认品牌色改为稳重产品蓝。
4. 收敛 `.btn`、`.card`、`.input`、`.select`、表格、页签、开关、徽章、浮层、空状态、Spinner 和焦点态。
5. 将 `.glass`、`.glass-card`、`.card-glass`、`.text-gradient` 等兼容类保留为纯色产品面或无装饰实现，避免全站旧类名失效。
6. 添加 `prefers-reduced-motion: reduce` 规则，并把常规过渡限制在 120 至 180ms。
7. 重新运行契约测试；令牌相关断言通过，页面相关断言仍可按预期失败。

### 验证

- 命令：`pnpm test:run -- uiDesignContract`
- 预期：令牌与 reduced-motion 相关断言通过；尚未改造的布局断言明确失败。

## 任务 2：应用壳层产品化

### 文件

- 修改：`frontend/src/components/layout/AppLayout.vue`
- 修改：`frontend/src/components/layout/AppHeader.vue`
- 修改：`frontend/src/components/layout/AppSidebar.vue`
- 必要时修改：`frontend/src/components/layout/ThemePicker.vue`

### 步骤

1. 先让契约测试中的壳层断言保持失败并确认原因。
2. 删除 `AppLayout` 全屏 mesh 层，使用冷白/深灰中性画布、稳定内容宽度和克制的桌面/移动内边距。
3. 将顶栏从玻璃胶囊改为实色微边框工具栏；降低余额区域和头像装饰权重，保留公告、语言、余额、帮助和用户菜单逻辑。
4. 将侧栏品牌区和导航项降权；当前项改为浅品牌底、品牌字和 2px 左侧指示线。
5. 保持长菜单滚动、折叠、移动端抽屉和主题入口行为不变。
6. 清理 header/sidebar 中不再需要的渐变、发光、过度阴影和超大圆角。
7. 运行壳层相关契约测试和 `pnpm typecheck`。

### 验证

- 命令：`pnpm test:run -- uiDesignContract`
- 预期：壳层相关断言通过。
- 命令：`pnpm typecheck`
- 预期：通过。

## 任务 3：认证布局与认证页

### 文件

- 修改：`frontend/src/components/layout/AuthLayout.vue`
- 修改：`frontend/src/views/auth/LoginView.vue`
- 修改：`frontend/src/views/auth/RegisterView.vue`
- 修改：`frontend/src/views/auth/ForgotPasswordView.vue`
- 必要时修改同目录其他认证页

### 步骤

1. 保持认证页契约断言失败并确认当前使用渐变、玻璃和装饰光球。
2. 将认证画布改为中性背景，移除网格、光球、渐变文字、玻璃和装饰阴影。
3. 使用居中的纯白/深灰卡片，桌面宽度 380 至 440px，移动端使用安全边距。
4. 统一认证页标题、说明、字段间距、主操作和帮助链接层级；一个表单只保留一个 Primary。
5. 不改登录、注册、找回密码的请求、校验和跳转逻辑。
6. 运行认证相关契约测试、类型检查和 i18n 检查。

### 验证

- 命令：`pnpm test:run -- uiDesignContract`
- 预期：认证相关断言通过。
- 命令：`pnpm run check:i18n`
- 预期：通过。

## 任务 4：表格、列表页与通用控件

### 文件

- 修改：`frontend/src/components/layout/TablePageLayout.vue`
- 修改：`frontend/src/components/common/DataTable.vue`
- 修改：`frontend/src/components/common/StatCard.vue`
- 修改：`frontend/src/components/common/BaseDialog.vue`
- 修改：`frontend/src/components/common/Input.vue`
- 修改：`frontend/src/components/common/Select.vue`
- 必要时修改同目录按钮、徽章、开关、空状态、Spinner 和抽屉组件

### 步骤

1. 保持表格契约断言失败并确认当前固定高度、渐变固定列阴影或滚动条样式不符合要求。
2. 调整列表页骨架为标题、唯一主操作、筛选区和表格的清晰顺序；页面根节点禁止横向溢出。
3. 收敛表格行高、表头字重和分隔线，减少大写/字距噪声。
4. 将固定列阴影改为细中性边界，统一固定列表头与单元格底色。
5. 保留 `.table-wrapper` 横向滚动、粘性表头、虚拟滚动高度链、选择、排序和移动端卡片模式。
6. 将滚动条改为低对比中性样式，移除不必要的 `!important`。
7. 统一输入、选择、弹窗、抽屉、徽章、空状态和 Spinner 的圆角、边框、焦点态与轻阴影。
8. 运行契约测试、类型检查和现有测试套件。

### 验证

- 命令：`pnpm test:run -- uiDesignContract`
- 预期：表格和控件相关断言通过。
- 命令：`pnpm test:run`
- 预期：现有测试全部通过。

## 任务 5：管理员 Dashboard 与设置页

### 文件

- 修改：`frontend/src/views/admin/DashboardView.vue`
- 修改：`frontend/src/views/admin/SettingsView.vue`
- 修改：`frontend/src/views/admin/UsersView.vue`
- 修改：`frontend/src/views/admin/ChannelsView.vue`

### 步骤

1. 保持设置页签契约断言失败并确认中文标签被截断。
2. 将管理员 Dashboard 的 8 张同权 KPI 重组为 4 至 6 个核心指标和更轻量的次要摘要；保留所有原数据、日期范围、粒度和快捷操作。
3. 统一 KPI 图标为中性浅底和品牌/语义色，移除彩虹色随机感。
4. 将设置页签改为轻量分段/底线式导航；中文标签禁止 `text-ellipsis` 和 `overflow-hidden` 截断，窄屏允许横向滚动。
5. 清理设置页玻璃 shell、重阴影和渐变 active 线，降低说明文字噪声。
6. 检查用户管理和渠道管理页面的筛选、表格、分页和唯一主操作层级。
7. 运行契约测试、类型检查和 i18n 检查。

### 验证

- 命令：`pnpm test:run -- uiDesignContract`
- 预期：Dashboard 和设置页签相关断言通过。
- 命令：`pnpm typecheck`
- 预期：通过。

## 任务 6：用户代表页与全站装饰残留

### 文件

- 修改：`frontend/src/views/user/DashboardView.vue`
- 修改：`frontend/src/views/user/KeysView.vue`
- 修改：`frontend/src/views/user/UsageView.vue`
- 修改：`frontend/src/views/user/ProfileView.vue`
- 按 `grep` 结果修改其他包含可见装饰残留的 `.vue` 页面

### 步骤

1. 用 `grep -RInE "bg-mesh-gradient|card-glass|glass-card|shadow-glass|shadow-glow|text-gradient|rounded-3xl" frontend/src --include='*.vue'` 建立残留清单。
2. 优先清理认证、壳层、列表、表单、弹窗和用户高频页面中的装饰性渐变、玻璃、发光、网格和超大圆角。
3. 保留仅用于品牌 Logo、图表数据可视化或已有语义表达的渐变；在交付摘要中说明保留理由。
4. 统一用户 Dashboard、API Key、用量和资料页的页面标题、卡片、表格、表单和空状态层级。
5. 确认所有旧工具类仍有安全的全局实现，避免未逐一修改的低频页面失去样式。
6. 重新运行残留扫描，目标为代表性高频页面零装饰残留，低频兼容类不再产生玻璃/发光效果。

### 验证

- 命令：`grep -RInE "bg-mesh-gradient|card-glass|glass-card|shadow-glass|shadow-glow|text-gradient|rounded-3xl" frontend/src --include='*.vue'`
- 预期：高频页面无命中；允许项均有明确业务理由。
- 命令：`pnpm test:run`
- 预期：全部通过。

## 任务 7：远程同步、构建与浏览器验收

### 步骤

1. 本地运行 `git diff --check`、`git status`，确认只包含前端 UI、设计契约测试和文档改动。
2. 将 `codex/ui-refresh` 分支同步到远程项目，不覆盖远程 `main`。
3. 在远程执行 `cd /home/guili/go/sub2api/frontend && pnpm build`。
4. 确认前端 5173 与后端 8080 服务健康。
5. 使用管理员账号登录，检查认证页、管理员 Dashboard、用户管理、渠道管理、系统设置。
6. 检查用户 Dashboard、API Key、用量和资料页；至少打开一个弹窗、一个抽屉、一个下拉菜单和一个危险操作确认框。
7. 在 1440px、1024px、768px、390px 视口检查布局；重点验证表格容器横向滚动、固定列、设置页中文页签和移动端导航。
8. 切换深色模式和语言，确认不退化。
9. 收集构建输出、HTTP 健康状态、登录和页面检查证据；若有截图，保留绝对路径。

### 验证

- 命令：`pnpm build`
- 预期：i18n 测试、`vue-tsc -b` 和 Vite 生产构建全部成功。
- 命令：`pnpm test:run`
- 预期：全部通过。
- 浏览器：上述页面、视口、主题和浮层检查全部通过。

## 提交策略

- 每个任务独立提交，提交信息使用 Conventional Commits。
- 计划与规格属于 ignored 文档路径，提交时使用 `git add --sparse -f`。
- 不推送远程分支，不修改 `main`；最终由用户决定是否合并或发布。

