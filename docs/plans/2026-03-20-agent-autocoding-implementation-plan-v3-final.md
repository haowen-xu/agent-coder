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

## M3 Issue Tracker 抽象与 GitLab 实现

- `infra/issuetracker/port.go` 抽象接口。
- `infra/issuetracker/gitlab` 具体实现。

## M4 调度与执行器

- 定时 pull issue。
- WorkItem 幂等创建与状态流转。

## M5 自动开发 + MR

- 分支命名 `agent-coder/issue-<id>`。
- 一个 issue 一个 MR（复用更新）。

## M6 冲突处理

- 自动冲突解决，重试 5 次。
- 超限置 `blocked` 并回写 issue。

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
