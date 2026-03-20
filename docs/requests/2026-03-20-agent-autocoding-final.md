# Agent 自动 Coding 需求定稿（Final）

> 日期：2026-03-20

## 1. 产品定位

系统用于基于 Issue Tracker 的自动研发执行，覆盖从 Issue 到 MR 的闭环。

## 2. 功能目标

- 自动读取 Issue
- 自动更新 Issue（评论、状态、标签、关联链接）
- 自动开发并执行验证
- 自动创建/更新 MR
- 自动处理 merge/rebase 冲突

## 3. 平台与触发策略

- 平台优先级：先支持 GitLab
- 触发方式：定时 pull issue（轮询）

## 4. 代码流程策略

- 分支命名固定：`agent-coder/issue-<id>`
- MR 策略：一个 issue 一个 MR（存在则复用更新）
- 冲突策略：自动重试 5 次，超限标记 `blocked` 并回写 issue

## 5. 多项目能力

支持多项目。每个项目绑定：

- 一个代码仓库
- 一个 issue tracker 项目（V1 为 GitLab project）
- 独立 `project_key`

## 6. 用户系统与权限

采用最简权限模型：`users.is_admin`

- `is_admin=true`
  - 管理用户
  - 管理项目
  - 设置项目密钥（`project_key`）
- `is_admin=false`
  - 仅可查看现有项目展板（只读）

不引入复杂 RBAC / ACL。

## 7. 关键领域对象

- `User`
  - `username`
  - `password_hash`
  - `is_admin`
  - `enabled`
- `ProjectBinding`
  - `project_key`
  - `repo_url`
  - `repo_default_branch`
  - `gitlab_project_id`
  - `credential_ref`
  - `poll_interval_sec`
  - `enabled`
- `WorkItem`
  - `project_key`
  - `issue_iid`
  - `status`
  - `branch_name`
  - `mr_iid`
  - `retry_count`
  - `last_error`

## 8. 工程实现约束

- 后端分层：`handler / service / dal / infra`
- 命令入口：`cmds/` + `spf13/cobra`
- `infra` 必须包含 issue tracker 抽象与 GitLab 实现
- WebUI 编译产物由 Go `go:embed` 挂载
- GORM 使用单实现，依赖 `sqlDialect string` 在方言敏感点切换

## 9. 验收标准

- admin/user 权限行为符合第 6 节
- 可配置多项目绑定并轮询 issue
- 能完成 issue -> 分支 -> MR 闭环
- 冲突重试上限 5 次生效
- 普通用户可查看展板且不可写管理接口
