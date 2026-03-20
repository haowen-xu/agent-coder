# agent-coder

`agent-coder` 是一个围绕 Issue 的自动研发执行系统。它将仓库协作平台、代码执行 Agent、任务状态机和 Web 管理界面整合到同一套服务中，用于实现从需求到代码再到合并的自动化闭环。

## 项目目标

- 建立从 Issue 到 Merge 的可追踪自动化流程。
- 用统一状态机管理执行过程（创建 run、执行、评审、重试、收敛）。
- 支持多项目接入，并提供项目级配置与 Prompt 覆盖能力。
- 通过 WebUI + API 提供可观测、可运维的控制面。

## 支持的后端（当前实现）

### 仓库协作平台：GitLab

- Provider 抽象在 `internal/infra/repo/common`。
- 当前已实现 `internal/infra/repo/gitlab`。
- Worker 会按项目轮询 GitLab Issue，并执行同步、回写、MR 相关动作。

### Agent 执行后端：Codex

- Agent 抽象在 `internal/infra/agent/base`。
- 当前已实现 `internal/infra/agent/codex`。
- 支持常规执行与 resume，结合 `issue_runs` 的 `run_kind/agent_role/loop_step` 推进循环。

## 工作流（核心）

1. 项目配置完成后，系统按周期同步 Issue。
2. 命中执行条件的 Issue 会创建 `issue_run` 并入队。
3. Worker 拉起 run，在工作目录执行 Agent：
   - `run_kind=dev`：`dev -> review`
   - `run_kind=merge`：`merge -> review`
4. `review` 判定 `pass/rework`：
   - `pass`：run 成功并推进 Issue 生命周期。
   - `rework`：`loop_step` 增长并继续下一轮。
5. 超过最大轮次或出现阻塞则标记失败，并记录错误摘要与日志。

## 总体架构

系统采用分层架构，依赖方向保持单向：`handler -> service -> dal`。

- `cmds/`：CLI 唯一入口（Cobra）
  - `server`：启动 HTTP API + WebUI
  - `worker`：启动调度与执行循环
  - `migrate`：初始化/迁移数据库
  - `sync-issues`：手动触发一次同步
- `internal/handler/httpserver`：HTTP 路由、鉴权、中间件、静态资源挂载
- `internal/service/core`：核心业务（项目、Issue、管理端能力）
- `internal/service/worker`：调度与状态机执行
- `internal/dal`：GORM 数据访问（SQLite/PostgreSQL）
- `internal/infra/repo/*`：仓库协作平台实现（当前 GitLab）
- `internal/infra/agent/*`：Agent 执行实现（当前 Codex）

## 运行方式

```bash
# 启动 API 服务
go run ./cmds server --config config.yaml

# 启动 Worker
go run ./cmds worker --config config.yaml
```

更多设计与字段约束请参考：

- `docs/architecture.md`
- `docs/agent-runtime.md`
- `docs/config-runtime.md`
- `docs/database-schema.md`
