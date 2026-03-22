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
- Worker 当前使用常规执行接口（`Run`）推进 `issue_runs` 的 `run_kind/agent_role/loop_step` 循环。
- `codex resume` 仅用于计划自动化脚本 `scripts/run_codex_on_plan.py`，不在 worker 主流程中启用。

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
- `internal/app/httpserver`：HTTP Server 装配与静态资源挂载
- `internal/handler/*`：按对象拆分的 Hertz Handler（`auth/user/project/issue/issue_run/ops`）
- `internal/service/*`：按对象拆分的业务服务（`user/project/issue/issue_run/ops`）
- `internal/service/core`：历史兼容服务层（逐步迁移中）
- `internal/service/orch`：调度与状态机执行（orchestrator service）
- `internal/dal`：GORM 数据访问（SQLite/PostgreSQL）
- `internal/infra/repo/*`：仓库协作平台实现（当前 GitLab）
- `internal/infra/agent/*`：Agent 执行实现（当前 Codex）

## 运行方式

```bash
# 调试/启动 API（会先执行 webui build）
make run

# 启动 Worker
go run ./cmds worker --config config.yaml

# 打包（会先执行 webui build）
make build
```

质量门禁脚本统一位于 `scripts/agents/`，统一入口为：

```bash
.venv/bin/python scripts/agents/check_all_gates.py
```

测试分层约束：

- 简单 e2e/集成测试（不需要多系统/外部系统协作）用 Go 测试编写并纳入 `make test`。
- 复杂多系统/外部系统协作测试放在 `tests/e2e/`，由 Python `unittest` 脚本编排环境并验证 API/DB/文件系统。
- 需要验证前端行为的 e2e 测试放在 `tests/playwright/`，由 Python `unittest` 脚本编排环境并调用 Playwright。

推荐执行：

```bash
make test
.venv/bin/python -m unittest tests.e2e.test_runtime_e2e -v
.venv/bin/python -m unittest tests.playwright.test_playwright_e2e -v
```

更多设计与字段约束请参考：

- `docs/architecture.md`
- `docs/agent-runtime.md`
- `docs/config-runtime.md`
- `docs/database-schema.md`
- `docs/testing-strategy.md`
