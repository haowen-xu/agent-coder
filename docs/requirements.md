# 产品需求定稿

> 更新日期：2026-03-21  
> 说明：本文件承接历史需求定稿中的有效约束，作为当前唯一需求定稿入口。

## 1. 产品定位

系统用于基于仓库协作平台的自动研发执行，覆盖从 Issue 到 MR 的闭环。

## 2. 功能目标

- 自动读取 Issue
- 自动更新 Issue（评论、状态、标签、关联链接）
- 自动开发并执行验证
- 自动创建/更新 MR
- 自动处理 merge/rebase 冲突

## 3. 平台与触发策略

- 平台优先级：先支持 GitLab
- 触发方式：定时 pull issue（轮询）

## 4. 执行策略

- 分支命名固定：`agent-coder/issue-<id>`
- MR 策略：一个 issue 一个 MR（存在则复用更新）
- 冲突策略：自动重试 5 次，超限标记 `failed` 并回写 issue

## 5. 多项目能力

支持多项目。每个项目绑定一个代码仓库、一个仓库协作平台项目（V1 为 GitLab project）、独立 `project_key`，以及项目级标签映射配置。

## 6. 用户系统与权限

采用最简权限模型：`users.is_admin`

- `is_admin=true`：管理用户、管理项目、设置项目密钥（`project_key`）
- `is_admin=false`：仅可查看现有项目展板（只读）

不引入复杂 RBAC / ACL。

## 7. 关键领域对象

详细字段、索引、状态机与运行目录规范以 [database-schema.md](database-schema.md) 为准。

核心对象：

- `User`
- `ProjectBinding`
- `WorkItem`
- `IssueRun`

## 8. 工程实现约束

- 后端分层：`handler / service / dal / infra`
- 命令入口：`cmds/` + `spf13/cobra`
- `infra` 必须包含仓库协作平台抽象与 GitLab 实现
- WebUI 编译产物由 Go `go:embed` 挂载
- GORM 使用单实现，依赖 `sqlDialect string` 在方言敏感点切换

## 9. 验收标准

- admin/user 权限行为符合第 6 节
- 可配置多项目绑定并轮询 issue
- 能完成 issue -> 分支 -> MR 闭环
- 冲突重试上限 5 次生效
- 普通用户可查看展板且不可写管理接口
- Go 单元测试覆盖率门禁：
  - 总体覆盖率 `>= 80%`
  - 含具体代码逻辑的 Go 文件不允许 `0%` 覆盖率
  - 单元测试文件必须与被测文件同名配对：`xxx.go` 对应 `xxx_test.go`
- 测试分层约束：
  - 简单 e2e/集成测试（不需要多系统/外部系统协作）必须用 Go 测试方式实现，并包含在 `make test`
  - 复杂多系统/外部系统协作测试放在 `tests/e2e/`，由 Python `unittest` 脚本编排环境并验证 API/数据库/文件系统状态
  - 前端行为 e2e 测试放在 `tests/playwright/`，由 Python `unittest` 脚本编排环境并调用 Playwright，必要时验证 API/数据库/文件系统状态
- 时间与时区门禁：
  - Go 生产代码统一使用 `internal/utils.NowUTC()` 获取当前时间
  - 时间传输格式统一为 UTC RFC3339（ISO）
  - WebUI 时间展示统一使用 `webui/src/utils/format.ts` 的 `formatLocalDateTime`

## 9.1 项目标识约束

- `issue_project_id` 允许为 `NULL`（兼容不提供项目 ID 的平台场景）
- `project_slug` 用于保存平台侧稳定可读标识（如 `group/repo`、`owner/repo`）
- `provider_url` 必填，表示仓库协作平台 API endpoint（例如 `https://gitlab.example.com/api/v4`）
- `repo_url` 仅表示代码仓库地址（例如 `git@...` 或 `https://...git`），与 `provider_url` 不能混用

## 10. Issue 标签工作流（默认值）

每个项目可自定义标签名；若未配置则使用以下默认值：

- `Agent Ready`：等待 Agent 系统接收
- `In Progress`：Agent 正在开发
- `Human Review`：已生成 MR，等待人工 Review
- `Rework`：需要重新修改
- `Verified`：人工确认可合并
- `Merged`：已完成合并，自动关闭 issue

补充（本地生命周期语义）：

- 本地 `issues.lifecycle_status` 不再使用 `merged`，统一收敛为 `closed`
- 通过 `issues.close_reason` 区分关闭原因：
  - `merged`：自动合并完成
  - `manual`：远端人工关闭
  - `need_human_merge`：仓库协作平台不允许自动合并，转人工合并

## 11. Agent 介入硬门禁

Agent 仅在满足以下条件时才允许进入本地系统：

1. 看到 `Agent Ready` 标签
2. 写入本地数据库（创建 issue 记录并登记 WorkItem）

也就是说：

- `Agent Ready` 之前：本地 `issues` 表无该记录
- `Agent Ready` 之后：先入库，再执行

## 12. 单次 IssueRun 执行循环

单次 run 采用固定 loop，不引入 `issue_sub_runs`。

角色与类型约束：

- `agent_role` 仅允许：`dev/review/merge`
- `run_kind=dev` 使用 `dev -> review`
- `run_kind=merge` 使用 `merge -> review`

`run_kind=dev` 循环：

1. `dev` 进行开发修改
2. `review` 进行完成度与质量检查
3. 若检查通过，run 标记 `succeeded`
4. 若检查不通过，`loop_step += 1` 后继续下一轮

`run_kind=merge` 循环：

1. `merge` 进行 rebase/merge 与冲突处理
2. `review` 检查是否满足合并条件
3. 若检查通过，run 标记 `succeeded`
4. 若检查不通过，`loop_step += 1` 后继续下一轮

失败条件：

- `loop_step > max_loop_step` 仍未通过 review，则 run 标记 `failed`
- 出现不可恢复执行错误，则 run 标记 `failed`

## 13. Prompt 管理（默认 + 项目覆盖）

- 默认 Prompt 以 markdown 文件形式放在 Go 项目中，通过 `go:embed` 内嵌
- 项目可在数据库配置 Prompt 覆盖（优先级高于默认模板）
- 覆盖维度：`project_key + run_kind + agent_role`
- 需提供 Admin 接口：查询默认模板、查询项目有效模板、写入/更新项目覆盖模板、删除项目覆盖模板（回退默认模板）
