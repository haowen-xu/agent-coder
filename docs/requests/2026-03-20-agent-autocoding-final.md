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
- 冲突策略：自动重试 5 次，超限标记 `failed` 并回写 issue

## 5. 多项目能力

支持多项目。每个项目绑定：

- 一个代码仓库
- 一个 issue tracker 项目（V1 为 GitLab project）
- 独立 `project_key`
- 项目级标签映射配置（可覆盖默认值）

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

详细字段与索引见：`docs/database-schema.md`。
状态机切换规则与 `issue_run` 工作目录规范也在该文档中定义。

- `User`
  - `username`
  - `password_hash`
  - `is_admin`
  - `enabled`
- `ProjectBinding`
  - `provider`（如 `gitlab`）
  - `provider_url`（`api_endpoint`，与 `repo_url` 不同）
  - `project_key`
  - `project_slug`
  - `repo_url`
  - `repo_default_branch`
  - `issue_project_id`（可为空）
  - `credential_ref`
  - `poll_interval_sec`
  - `enabled`
  - `label_agent_ready`
  - `label_in_progress`
  - `label_human_review`
  - `label_rework`
  - `label_verified`
  - `label_merged`
- `WorkItem`
  - `project_key`
  - `issue_iid`
  - `status`
  - `branch_name`
  - `mr_iid`
  - `retry_count`
  - `last_error`
  - `workdir_path`

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

## 9.1 项目标识兼容约束

- `issue_project_id` 允许为 `NULL`（兼容不提供项目 ID 的平台场景）。
- 新增 `project_slug`（`VARCHAR`），用于保存平台侧稳定可读标识（如 `group/repo`、`owner/repo`）。
- `provider_url` 必填，表示 issue provider 的 API endpoint（例如 `https://gitlab.example.com/api/v4`）。
- `repo_url` 仅表示代码仓库地址（例如 `git@...` 或 `https://...git`），两者不能混用。

## 10. Issue 标签工作流（默认值）

每个项目可自定义标签名；若未配置则使用以下默认值：

- `Agent Ready`：等待 Agent 系统接收
- `In Progress`：Agent 正在开发
- `Human Review`：已生成 MR，等待人工 Review
- `Rework`：需要重新修改
- `Verified`：人工确认可合并
- `Merged`：已完成合并，自动关闭 issue

## 11. Agent 介入硬门禁

普通 issue 即使带有 `In Progress` / `Human Review` 等标签，也不允许 Agent 直接介入，且不会写入本地 `issues` 表。

Agent 仅在满足以下条件时才允许进入本地系统：

1. 看到 `Agent Ready` 标签
2. 写入本地数据库（创建 issue 记录并登记 WorkItem）

也就是说：

- `Agent Ready` 之前：本地 `issues` 表无该记录
- `Agent Ready` 之后：先入库，再执行
