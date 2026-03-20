# 工程架构（目标版）

## 技术栈

- 后端：Go、Hertz、Cobra、Viper、slog、errorx、GORM
- 数据库：SQLite / PostgreSQL（同一套 GORM DAL）
- 前端：pnpm、Vite、Vue3、Pinia、Element Plus

## 分层原则

- `handler -> service -> dal` 单向依赖。
- `infra` 提供外部系统实现（gitlab、git、scheduler、secret、db），由 `app` 组装注入。
- `cmds` 仅负责命令入口，不承载业务逻辑。

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
│   ├── config/
│   ├── handler/
│   │   ├── auth/
│   │   ├── admin/
│   │   ├── project/
│   │   ├── board/
│   │   └── middleware/
│   ├── service/
│   │   ├── auth/
│   │   ├── user/
│   │   ├── project/
│   │   ├── issue/
│   │   └── workitem/
│   ├── dal/
│   │   ├── model/
│   │   ├── repo/
│   │   ├── query/
│   │   ├── tx/
│   │   └── migrate/
│   ├── infra/
│   │   ├── db/
│   │   ├── logger/
│   │   ├── scheduler/
│   │   ├── git/
│   │   ├── secret/
│   │   └── issuetracker/
│   │       ├── port.go
│   │       ├── factory.go
│   │       ├── types.go
│   │       └── gitlab/
│   │           ├── client.go
│   │           ├── mapper.go
│   │           └── api_types.go
│   ├── worker/
│   └── transport/
│       └── httpserver/
│           ├── server.go
│           ├── routes.go
│           ├── webui_embed.go
│           └── static/
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

- 每个 run 使用独立工作目录，不共享。
- 默认根目录：`.agent-coder/workdirs`（可配置为 `runtime.workdir_root`）。
- 路径规范：
  - `<workdir_root>/<project_key>/issue-<issue_iid>/run-<run_no>`
- 推荐通过 `git worktree` 创建并在 run 结束后按策略清理。

## Issue Tracker 抽象

- 统一抽象放在 `internal/infra/issuetracker/port.go`。
- GitLab 实现放在 `internal/infra/issuetracker/gitlab`。
- 后续支持 GitHub 时新增同级实现目录，不改 service 层接口。
- 轮询同步策略：仅将带 `Agent Ready` 的 issue 写入本地 `issues` 表。
- `ProjectBinding` 中必须区分：
  - `provider_url`：Issue Provider API endpoint
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
- run 工作目录由 `issue_runs.workdir_path` 提供，agent 实现只负责在该目录执行。

## 数据库策略（SQLite + PostgreSQL）

- 只维护一个 GORM 实现，不拆双 DAL。
- DB Client 维护 `sqlDialect string`（`sqlite`/`postgres`）。
- 仅在方言敏感点做 `switch`（如 upsert、锁、少量原生 SQL）。
- 常规 CRUD 统一走 GORM 兼容层。

## WebUI 挂载策略（go:embed）

- `webui` 构建产物输出到 `internal/transport/httpserver/static/`。
- Go 侧使用 `go:embed` 打包 static 目录。
- HTTP 服务统一挂载：
  - `/api/*` -> 后端接口
  - 其他路由 -> WebUI `index.html`（SPA fallback）

## 运行入口

- 后端：`go run ./cmd/agent-coder server --config config.yaml`
- Worker：`go run ./cmd/agent-coder worker --config config.yaml`
- 前端：`cd webui && pnpm dev`
- 计划执行器：`python3 scripts/run_codex_on_plan.py --plan-file ...`
