# 工程架构（目标版）

## 技术栈

- 后端：Go、Hertz、Cobra、Viper、slog、errorx、GORM
- 数据库：SQLite / PostgreSQL（同一套 GORM DAL）
- 前端：pnpm、Vite、Vue3、Pinia、Element Plus

## 分层原则

- `handler -> service -> dal` 单向依赖。
- `infra` 提供外部系统实现（gitlab、git、scheduler、secret、db），由 `app` 组装注入。
- `cmds` 仅负责命令入口，不承载业务逻辑。
- 仓库协作平台密钥通过 `infra/secret` 读取（默认 env provider），项目仅保存 `credential_ref`。

## 目标目录结构

```text
.
├── cmd/
│   └── agent-coder/
│       └── main.go
├── cmds/
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
│   │   └── prompt_template.go
│   ├── handler/
│   │   └── httpserver/
│   │       ├── server.go
│   │       ├── auth_handlers.go
│   │       ├── admin_handlers.go
│   │       ├── board_handlers.go
│   │       ├── middleware.go
│   │       ├── static.go
│   │       └── static/
│   ├── service/
│   │   ├── core/
│   │   │   └── service.go
│   │   └── worker/
│   │       └── service.go
│   ├── infra/
│   │   ├── agent/
│   │   │   ├── base/
│   │   │   ├── codex/
│   │   │   ├── prompts/
│   │   │   └── promptstore/
│   │   ├── git/
│   │   │   └── client.go
│   │   └── repo/
│   │       ├── common/
│   │       │   ├── port.go
│   │       │   ├── types.go
│   │       │   └── errors.go
│   │       └── gitlab/
│   │           ├── client.go
│   │           └── api_types.go
│   ├── logger/
│   └── xerr/
├── webui/
├── scripts/
└── docs/
```

## 命令体系（Cobra）

- `cmd/agent-coder/main.go` 仅执行 `cmds.Execute()`。
- `cmds/root.go` 挂载全局参数（配置文件、日志级别等）。
- 子命令：
  - `server`：启动 API 服务
  - `worker`：启动轮询与执行器
  - `migrate`：数据库迁移
  - `sync-issues`：手动触发一次 issue 同步

## 执行工作目录（issue_run）

- 目录分为 issue 级和 run 级两层，不混用：
  - issue 级：`git-tree`（代码工作区）
  - run 级：`agent/runs/<run_no>`（agent 运行态）
- 默认根目录：`.agent-coder/workdirs`（可配置为 `work.work_dir`）。
- 路径规范：
  - `<workdir_root>/<project_id>/<issue_id>/git-tree`
  - `<workdir_root>/<project_id>/<issue_id>/agent/runs/<run_no>`
- 推荐通过 `git worktree` 管理 `git-tree`，并在 run 结束后按策略清理 `agent/runs/<run_no>`。

## 仓库协作平台抽象

- 统一抽象放在 `internal/infra/repo/common/port.go`。
- GitLab 实现放在 `internal/infra/repo/gitlab`。
- 后续支持 GitHub 时新增同级实现目录，不改 service 层接口。
- 轮询同步策略：仅将带 `Agent Ready` 的 issue 写入本地 `issues` 表。
- `ProjectBinding` 中必须区分：
  - `provider_url`：仓库协作平台 API endpoint
  - `repo_url`：代码仓库地址
- 需支持项目级标签映射配置（默认值 + 覆盖）：
  - `Agent Ready`
  - `In Progress`
  - `Human Review`
  - `Rework`
  - `Verified`
  - `Merged`

## Agent 执行抽象

- 归属层级：`internal/infra/agent`
- 分层：
  - `base`：统一执行抽象与通用运行骨架
  - `codex`：具体 provider 实现
- 业务层只依赖 `base.Client`，不直接依赖 `codex` 命令细节。
- agent 实现在 `issue_runs.git_tree_path` 执行代码任务，在 `issue_runs.agent_run_dir` 写入运行态文件。
- `issue_runs` 通过 `run_kind` 区分 `dev/merge` 两类 run。
- 单次 run 循环：
  - `run_kind=dev`：`dev -> review`
  - `run_kind=merge`：`merge -> review`
- 扁平追踪字段：`issue_runs.agent_role`、`issue_runs.loop_step`。
- Prompt 模板来源：
  - 默认模板：`internal/infra/agent/prompts/defaults/*.md`（`go:embed`）
  - 项目覆盖模板：数据库 `prompt_templates`（按 `project_key + run_kind + agent_role`）

## 数据库策略（SQLite + PostgreSQL）

- 只维护一个 GORM 实现，不拆双 DAL。
- DB Client 维护 `sqlDialect string`（`sqlite`/`postgres`）。
- 仅在方言敏感点做 `switch`（如 upsert、锁、少量原生 SQL）。
- 常规 CRUD 统一走 GORM 兼容层。

## WebUI 挂载策略（go:embed）

- `webui` 构建产物输出到 `internal/handler/httpserver/static/`。
- Go 侧使用 `go:embed` 打包 static 目录。
- HTTP 服务统一挂载：
  - `/api/*` -> 后端接口
  - 其他路由 -> WebUI `index.html`（SPA fallback）

## 运行入口

- 后端：`go run ./cmd/agent-coder server --config config.yaml`
- Worker：`go run ./cmd/agent-coder worker --config config.yaml`
- 前端：`cd webui && pnpm dev`
- 计划执行器：`python3 scripts/run_codex_on_plan.py --plan-file ...`
