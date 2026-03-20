# Agent 自动 Coding 需求（V4 定稿，简化权限版）

> 日期：2026-03-20  
> 在 V3 基础上新增用户系统，并按“最简权限模型”收敛。

## 1. 产品目标

构建一个基于 Issue Tracker 的自动研发执行系统，围绕 Issue 实现端到端闭环：

- 自动读取/更新 Issue
- 自动开发并验证
- 自动创建 MR
- 自动解决 merge 冲突

## 2. 已确认策略

- 触发方式：定时 pull issue（轮询）
- 分支命名：`agent-coder/issue-<id>`
- MR 策略：一个 issue 一个 MR
- 冲突策略：自动重试 `N=5`
- 平台优先级：先做 GitLab

## 3. 用户系统（简化）

## 3.1 权限模型

不引入复杂 RBAC。仅使用用户表字段：

- `is_admin: bool`

权限规则：

- `is_admin = true`：
  - 管理用户（增删改查/启停）
  - 管理项目（增删改查）
  - 设置项目密钥（`project_key`）
- `is_admin = false`：
  - 只能查看现有项目展板（只读）
  - 不可管理用户
  - 不可管理项目

## 3.2 非目标

- 不做角色组、策略引擎、细粒度资源授权
- 不做项目级成员 ACL（默认普通用户可查看全部项目展板）

## 4. 领域模型（V4）

## 4.1 User

- `id`
- `username`
- `password_hash`
- `is_admin`（核心权限位）
- `enabled`
- `created_at`
- `updated_at`

## 4.2 ProjectBinding

- `project_key`（由 admin 设置，唯一）
- `repo_url`
- `repo_default_branch`
- `gitlab_project_id`
- `credential_ref`
- `poll_interval_sec`
- `enabled`

## 4.3 WorkItem

- `work_item_id`
- `project_key`
- `issue_iid`
- `status`：`queued` | `running` | `blocked` | `done` | `failed`
- `branch_name`
- `mr_iid`
- `retry_count`
- `last_error`

## 5. 核心流程（V4）

1. admin 创建项目绑定并设置 `project_key`。
2. 系统按项目轮询 GitLab Issue。
3. 生成/复用 WorkItem，切分支 `agent-coder/issue-<id>`。
4. Agent 开发并验证，创建/更新 MR（一 issue 一 MR）。
5. 冲突自动重试最多 5 次，超限则标记 `blocked` 并回写 Issue。
6. 普通用户仅可在展板查看项目与 WorkItem 状态。

## 6. API 范围（V4）

## 6.1 认证

- `POST /api/v1/auth/login`
- `POST /api/v1/auth/logout`

## 6.2 admin 接口

- `GET /api/v1/admin/users`
- `POST /api/v1/admin/users`
- `PATCH /api/v1/admin/users/:id`
- `GET /api/v1/admin/projects`
- `POST /api/v1/admin/projects`
- `PATCH /api/v1/admin/projects/:project_key`

## 6.3 普通用户只读接口

- `GET /api/v1/projects/boards`
- `GET /api/v1/projects/:project_key/boards`

## 7. 验收标准（V4）

- 存在可登录用户系统，且 `is_admin` 生效。
- admin 可管理用户和项目，可设置/更新 `project_key`。
- 普通用户仅可读取项目展板，不可访问 admin 写接口。
- GitLab 自动闭环能力满足 V3 所有验收项。
