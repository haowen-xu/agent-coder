# WebUI 模块结构

## 目录说明

```text
webui/src/
  router/        # 路由注册与前置守卫
  layouts/       # 页面布局（认证页/控制台）
  views/         # 路由页面
  components/    # 可复用 UI 组件（common/board/admin）
  composables/   # 业务流程逻辑（项目上下文、Prompt、Ops）
  api/           # 按域拆分的请求入口
  types/         # 前端领域类型
  stores/        # Pinia 状态
```

## 路由

- `/login`
- `/board`
- `/admin/overview`
- `/admin/users`
- `/admin/projects`
- `/admin/prompts`
- `/admin/ops`

路由守卫规则：

- 未登录访问受保护页面时重定向到 `/login`
- 已登录访问 `/login` 时重定向到 `/board`
- 非管理员访问 `/admin/**` 时重定向到 `/board`

## 分层约定

- `view` 只处理页面编排与事件绑定。
- 复用流程逻辑优先放 `composables/*`。
- 跨页面共享状态放 `stores/*`。
- HTTP 请求统一走 `api/*`，页面不直接 `fetch`。
- 领域类型统一在 `types/*`，避免 `any` 扩散。

## 常用命令

```bash
pnpm dev
pnpm build
```
