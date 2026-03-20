# Agent 自动 Coding 实施计划（V3 Final）

## 1. 目标

按 Final 需求实现 GitLab 首版闭环与最简用户系统。

## 2. 里程碑

## M0 工程骨架重构

- 重构为 `cmd/ + cmds/ + internal/{handler,service,dal,infra}`。
- 接入 Cobra 命令体系。

## M1 用户系统

- `users` 表与登录鉴权。
- `is_admin` 权限中间件：
  - `RequireLogin`
  - `RequireAdmin`

## M2 项目管理

- `ProjectBinding` 表与 admin 管理 API。
- 支持设置/更新 `project_key`。
- 新增 `project_slug`（`VARCHAR`）并支持唯一性校验。
- `issue_project_id` 改为可空字段（兼容平台差异）。
- 新增 `provider_url`（`api_endpoint`）并设为必填，语义与 `repo_url` 分离。
- 支持项目级标签映射配置（带默认值回退）：
  - `label_agent_ready`
  - `label_in_progress`
  - `label_human_review`
  - `label_rework`
  - `label_verified`
  - `label_merged`

## M3 Issue Tracker 抽象与 GitLab 实现

- `infra/issuetracker/port.go` 抽象接口。
- `infra/issuetracker/gitlab` 具体实现。

## M3.1 Agent 抽象与 Codex 实现

- `infra/agent/base`：定义通用 agent 调用接口和运行结果模型。
- `infra/agent/codex`：提供 codex 命令执行与 resume 支持。
- 保证 worker 仅依赖 `base`，不感知 provider 命令细节。

## M4 调度与执行器

- 定时 pull issue。
- WorkItem 幂等创建与状态流转。
- 落地 `issues.lifecycle_status` 与 `issue_runs.status` 状态机（按文档定义）。
- 介入门禁：
  - 仅 `Agent Ready` 的 issue 才允许写入本地 `issues` 表
  - 先入库（registered）再允许执行开发
  - 仅凭 `In Progress` 等标签不可触发写库和执行

## M5 自动开发 + MR

- 分支命名 `agent-coder/issue-<id>`。
- 一个 issue 一个 MR（复用更新）。
- issue 级生成 `git-tree` 代码目录（可复用）。
- run 级生成 `agent/runs/<run_no>` 运行目录（每次 run 独立）。
- 单次 run 采用 `dev_agent -> review_agent` 循环。
- 使用 `issue_runs.agent_role` + `issue_runs.loop_step` 扁平记录进度。
- 当 `loop_step > max_loop_step` 仍未通过 review，run 置 `failed`。
- 状态标签流转：
  - 开始执行 -> `In Progress`
  - 产出 MR -> `Human Review`
  - 人工要求返工 -> `Rework`
  - 人工确认可合并 -> `Verified`
  - 合并完成 -> `Merged` + 自动关闭 issue

## M6 冲突处理

- 自动冲突解决，重试 5 次。
- 超限置 `failed` 并回写 issue。

## M7 展板只读视图

- 普通用户可查看项目展板。
- 普通用户访问 admin 写接口返回 403。

## 3. 验证清单

- 后端：`go test ./...`
- 前端：`cd webui && pnpm build`
- 权限：
  - admin 可管理用户/项目
  - user 仅可读展板
- 闭环：
  - 轮询 issue -> 开发 -> MR -> 回写 issue
  - 冲突重试次数上限=5
  - 不带 `Agent Ready` 的 issue 不会写入本地 `issues` 表
  - 带 `Agent Ready` 的 issue 先入库后执行
