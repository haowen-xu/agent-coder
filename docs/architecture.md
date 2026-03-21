# 工程架构（当前实现）

## 技术栈

- 后端：Go、Hertz、Cobra、Viper、slog、errorx、GORM
- 数据库：SQLite / PostgreSQL（同一套 GORM DAL）
- 前端：pnpm、Vite、Vue3、Pinia、Element Plus
- 自动化脚本：Python（`scripts/run_codex_on_plan.py`）

## 分层原则

- 依赖方向保持单向：`handler -> service -> dal`。
- `infra` 提供外部系统能力（repo provider、agent、git、secret），由 `app` 组装注入。
- `cmds` 仅承担进程入口和参数装配，不承载业务决策。

## 核心目录结构（当前）

```text
.
├── cmds/
│   ├── main.go
│   ├── root.go
│   ├── server.go
│   ├── worker.go
│   ├── migrate.go
│   └── sync_issues.go
├── internal/
│   ├── app/
│   ├── auth/
│   ├── config/
│   ├── dal/
│   │   ├── db.go
│   │   ├── model.go
│   │   ├── user_repo.go
│   │   ├── project_repo.go
│   │   ├── issue_repo.go
│   │   ├── run_repo.go
│   │   ├── metrics_repo.go
│   │   └── prompt_template.go
│   ├── handler/
│   │   └── httpserver/
│   ├── service/
│   │   ├── core/
│   │   └── worker/
│   ├── infra/
│   │   ├── agent/
│   │   │   ├── base/
│   │   │   ├── codex/
│   │   │   ├── prompts/
│   │   │   └── promptstore/
│   │   ├── git/
│   │   ├── repo/
│   │   │   ├── common/
│   │   │   └── gitlab/
│   │   └── secret/
│   ├── logger/
│   └── xerr/
├── webui/
├── scripts/
└── docs/
```

## 命令入口（Cobra）

- `go run ./cmds server --config config.yaml`：启动 API 服务 + 静态资源。
- `go run ./cmds worker --config config.yaml`：启动轮询与执行循环。
- `go run ./cmds migrate --config config.yaml`：执行数据库迁移。
- `go run ./cmds sync-issues --config config.yaml`：手动触发一次 issue 同步。

## HTTP 路由分组

- 公共：`GET /healthz`、`GET /api/v1/meta`、`POST /api/v1/auth/login`
- 登录后：
  - `GET /api/v1/auth/me`
  - `GET /api/v1/board/projects`
  - `GET /api/v1/board/projects/:projectKey/issues`
- 管理端（admin）：
  - 用户：`/api/v1/admin/users`
  - 项目：`/api/v1/admin/projects`
  - Prompt：`/api/v1/admin/prompts/defaults`、`/api/v1/admin/projects/:projectKey/prompts/*`
  - 运行态：`/api/v1/admin/issues/:issueID/runs`、`/api/v1/admin/runs/:runID/logs` 等

## Worker 执行模型

- 调度来源：扫描 `issues`，为可执行 issue 创建 `issue_runs`（`dev` 或 `merge`）。
- 单次 run 循环：
  - `run_kind=dev`：`dev -> review`
  - `run_kind=merge`：`merge -> review`
- 判定规则：
  - `review.pass` -> 当前 run 成功
  - 超过 `max_loop_step` 或主流程 `blocked/error` -> 当前 run 失败
- 完成后推进 issue 生命周期，并与远端标签 / MR / issue 状态同步。

## 工作目录

- 根目录：`work.work_dir`（默认 `.agent-coder/workdirs`）
- issue 级代码目录：`<root>/<project_id>/<issue_id>/git-tree`
- run 级目录：`<root>/<project_id>/<issue_id>/agent/runs/<run_no>`

## 仓库协作平台与密钥

- provider 抽象：`internal/infra/repo/common`
- 当前实现：`internal/infra/repo/gitlab`
- 项目保存 `credential_ref` 或 `project_token`，密钥读取走 `infra/secret`。

## Agent 抽象

- 业务层依赖 `base.Client` 接口，不绑定具体 provider。
- 当前 provider：`infra/agent/codex`
- Prompt 来源：embedded defaults + 项目级覆盖（`prompt_templates`）

## 自动化脚本范围

- 自动化计划执行当前仅支持 `scripts/run_codex_on_plan.py`。
- 仓库内不再维护内置 agents 脚本，统一由外部 Agent-Coder 管理。
- Autofix 产物目录统一使用 `.ai-docs/autofix/`。
