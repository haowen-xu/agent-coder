# Agent 自动 Coding 实施计划（V2，含简化用户系统）

## 1. 目标

在 GitLab 自动 coding 闭环基础上，加入最简权限模型：

- 用户表仅使用 `is_admin` 区分权限
- admin 管理用户/项目（含 `project_key`）
- 普通用户仅查看项目展板（只读）

## 2. 里程碑

## M0 用户系统与认证（新增）

- 新增 `users` 表：`username/password_hash/is_admin/enabled`
- 登录鉴权（JWT 或等价会话）
- 中间件：`RequireAdmin`、`RequireLogin`
- 交付物：
  - 认证 API
  - admin/user 最简权限拦截

## M1 项目绑定与存储层

- 新增 `ProjectBinding`、`WorkItem` 表结构与仓储。
- 项目绑定管理由 admin 执行，重点字段：
  - `project_key`（唯一、可配置）
  - `repo_url`
  - `gitlab_project_id`
  - `credential_ref`
  - `poll_interval_sec`

## M2 GitLab 接入层（Issue + MR）

- 拉取/更新 issue
- 创建/更新 MR
- 按 issue 建立 MR 映射并防重（一个 issue 一个 MR）

## M3 调度与执行器

- 定时轮询器（按项目 pull issue）
- WorkItem 幂等创建与状态机管理

## M4 自动开发与 MR 流程

- 分支：`agent-coder/issue-<id>`
- 自动开发、验证、push、更新 MR

## M5 冲突处理与重试

- 冲突自动修复重试，最大 `N=5`
- 超限后置 `blocked` 并回写 issue

## M6 展板与权限校验

- admin 视图：用户管理、项目管理
- user 视图：项目展板只读
- 接口权限验证：
  - 普通用户访问 admin 接口必须 403

## 3. API 草案（V2）

- 认证：
  - `POST /api/v1/auth/login`
  - `POST /api/v1/auth/logout`
- admin：
  - `GET/POST/PATCH /api/v1/admin/users...`
  - `GET/POST/PATCH /api/v1/admin/projects...`
- 展板只读：
  - `GET /api/v1/projects/boards`
  - `GET /api/v1/projects/:project_key/boards`

## 4. 验证清单

- 权限验证：
  - admin 可写用户/项目
  - user 不可写（403）
  - user 可读展板
- 自动 coding 验证：
  - 定时拉取 issue
  - 自动生成/更新 MR
  - 冲突重试上限=5 生效
