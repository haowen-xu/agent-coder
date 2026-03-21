# Issue Draft: WebUI 模块化拆分与 Router 引入（基于当前单页实现）

> 日期：2026-03-21  
> 状态：Implemented (2026-03-21)  
> 目标版本：V1.2

## 1. 背景

当前 `webui` 以单文件 `App.vue` 承担登录、展板、管理台（用户/项目/Prompt/运维）全部 UI 与交互逻辑，后续功能迭代成本高：

- 页面逻辑耦合重（登录态、管理员态、项目选择、运维操作集中在一个组件）
- 复用困难（同类表格操作、项目选择状态、异步加载模式重复）
- 可测试性弱（难以做页面级与流程级测试）
- 难以渐进增加页面（如系统健康、运行详情、未来 todo 流程可视化）

目标是将当前单页实现拆分为合理的前端项目结构，并引入 `vue-router`。

## 2. 目标

- 引入路由层，形成明确页面边界（`views`）。
- 拆分可复用业务逻辑到 `composables`。
- 拆分 API 与类型定义，减少页面层直接处理请求细节。
- 保持现有功能与权限行为不变（不改变后端接口语义）。

## 3. In Scope / Out of Scope

In Scope：

- 引入 `vue-router` 与页面级路由守卫（登录态 + admin 权限）。
- `App.vue` 拆分为布局 + 路由容器。
- 新建 `views`、`components`、`composables`、`router`、`types` 目录并迁移现有逻辑。
- 管理台按功能域拆分为子视图（用户、项目、Prompt、运维日志）。
- 保持当前 API 路径与 store 行为兼容。

Out of Scope：

- UI 视觉重设计（颜色、品牌、交互风格大改）。
- 后端 API 重构或协议调整。
- 引入复杂状态管理方案替换 Pinia。

## 4. 建议目录结构

```text
webui/src/
  main.ts
  App.vue
  router/
    index.ts
    guards.ts
  layouts/
    AuthLayout.vue
    ConsoleLayout.vue
  views/
    LoginView.vue
    BoardView.vue
    admin/
      AdminUsersView.vue
      AdminProjectsView.vue
      AdminPromptsView.vue
      AdminOpsView.vue
      AdminOverviewView.vue
  components/
    common/
      PageHeader.vue
      ProjectSelect.vue
      AsyncStateAlert.vue
    board/
      ProjectTable.vue
      IssueTable.vue
    admin/
      UserCreateForm.vue
      UserTable.vue
      ProjectForm.vue
      ProjectTable.vue
      PromptPanel.vue
      OpsPanel.vue
  composables/
    useProjectForm.ts
    useProjectContext.ts
    usePromptEditor.ts
    useOpsActions.ts
  api/
    client.ts
    auth.ts
    board.ts
    admin.ts
  types/
    auth.ts
    board.ts
    admin.ts
  stores/
    session.ts
    board.ts
    admin.ts
    health.ts
```

## 5. 路由设计建议

- `/login`
- `/board`
- `/admin`（重定向到 `/admin/overview`）
- `/admin/overview`
- `/admin/users`
- `/admin/projects`
- `/admin/prompts`
- `/admin/ops`

路由守卫：

- 未登录统一跳转 `/login`
- 非 admin 访问 `/admin/**` 时跳转 `/board`
- 已登录用户访问 `/login` 时跳转 `/board`

## 6. 拆分策略（分阶段）

## Phase 1: 脚手架与兼容层

- 新增 `router/index.ts` 并接入 `main.ts`
- 保留现有 `App.vue` 逻辑，先作为过渡容器
- 增加布局组件（`AuthLayout` / `ConsoleLayout`）

验收：

- `pnpm build` 通过
- 登录、登出、展板访问路径可用

## Phase 2: 页面拆分

- 将登录区块迁移到 `LoginView.vue`
- 将展板区块迁移到 `BoardView.vue`
- 将管理台拆成 4 个子视图（users/projects/prompts/ops）

验收：

- 所有现有功能可从新路由访问
- 功能回归通过（用户管理、项目管理、Prompt 管理、运维日志）

## Phase 3: 逻辑下沉

- 将重复逻辑下沉到 `composables`（项目选择、Prompt 编辑、Ops 动作）
- API 调用按域拆分到 `api/*`
- 页面层只处理展示与交互

验收：

- 页面文件显著收敛（避免再出现超大单文件）
- 请求错误处理路径一致（统一通过组件或 composable 暴露）

## Phase 4: 清理与补充

- 删除迁移后遗留的重复代码
- 补充基础文档（路由结构、页面边界、composable 约定）
- 评估并接入 `health` 页（可作为 `/admin/overview` 的一部分）

验收：

- 无死代码
- 文档可指导后续新增页面

## 7. 任务清单（可直接转实施计划）

- [x] T1: 引入 router + 路由守卫 + 基础布局
- [x] T2: 拆分登录与展板视图
- [x] T3: 拆分管理台视图（users/projects/prompts/ops）
- [x] T4: 抽离 composables 与 API 域模块
- [x] T5: 回归测试与文档更新

## 8. 验收标准

- 前端构建：`cd webui && pnpm build` 通过
- 权限行为不变：
  - admin 可访问 `/admin/**`
  - non-admin 被正确拦截
- 核心流程不变：登录 -> 展板 -> 管理台操作可用
- 代码结构符合第 4 节目录约束

## 9. 风险与控制

- 风险：拆分中引入行为回归  
  控制：按 Phase 分段迁移，每阶段完成后运行构建并做功能冒烟

- 风险：路由守卫与 session 初始化竞态  
  控制：在路由前置守卫中加入 `session.fetchMe()` 的一次性初始化策略

- 风险：页面状态散落导致维护成本上升  
  控制：明确“页面态在 view，跨页面共享态在 store，流程逻辑在 composable”的分层规则

## 10. 建议 Issue 标题

`refactor(webui): split monolithic App.vue into routed modular architecture (views/composables/components)`

## 11. 2026-03-21 执行记录

- 已引入 `vue-router`，并通过前置守卫实现登录态与 admin 权限拦截。
- `App.vue` 已收敛为路由容器，页面逻辑迁移到 `views/*` 与 `layouts/*`。
- 管理台完成 users/projects/prompts/ops 子视图拆分，并补充可复用组件。
- API 已按 `auth/board/admin` 分域，`types/*` 与 store 引用已收敛。
- 逻辑已继续下沉到 composables（`useProjectForm`、`useProjectContext`、`usePromptEditor`、`useOpsActions`）。
- 已补充通用组件（`common/*`）与展板组件（`board/*`），减少页面重复代码。
- 本地构建校验：`cd webui && pnpm build` 通过。
- 后续可选优化：
  - 将 Element Plus 调整为按需引入，进一步降低 `element-plus` chunk 体积告警。
