# 配置与本地工作区规范

> 本文档定义 `config.yaml`、`config` 包运行时行为、工作目录布局以及 DB 方言切换约束。

## 1. 统一配置文件

系统使用单一 `config.yaml`，由 Viper 加载。

示例：

```yaml
app:
  name: agent-coder
  env: dev

server:
  host: 0.0.0.0
  port: 8080

db:
  enabled: true
  driver: sqlite # sqlite | postgres
  sqlite_path: agent-coder.db
  postgres_dsn: "host=127.0.0.1 user=postgres password=postgres dbname=agent_coder port=5432 sslmode=disable TimeZone=Asia/Shanghai"
  max_open_conns: 20
  max_idle_conns: 10
  conn_max_lifetime: 30m
  auto_migrate: true

secret:
  provider: env
  env_prefix: AGENT_CODER_SECRET_

work:
  work_dir: .agent-coder/workdirs

agent:
  provider: codex
  codex:
    binary: codex
    sandbox: true
    timeout_sec: 7200
    max_retry: 5
    max_loop_step: 5

scheduler:
  enabled: true
  poll_interval_sec: 30
  run_every: 30s

issue_provider:
  http_timeout_sec: 30

bootstrap:
  admin_username: admin
  admin_password: admin123
```

## 2. config 包与热更新预留

`internal/config` 包使用原子指针保存当前配置快照：

- `var cfgPtr atomic.Pointer[Config]`
- `Current() *Config`：读取当前快照
- `Replace(*Config)`：原子替换

约束：

- 业务代码只读配置快照，不可修改原对象。
- 配置重载时必须 `Load -> Validate -> Replace`，失败不覆盖旧配置。
- 先实现静态加载，再预留 `WatchConfig` 热更新入口。

## 3. 本地工作区目录

工作目录根由 `work.work_dir` 配置控制。

目录规范（按你的要求）：

```text
work_dir/
  <project_id>/
    <issue_id>/
      git-tree/                 # git worktree 工作区（代码目录）
      agent/
        runs/
          <run_no>/
            input/
            output/
            logs/
            meta.json
```

说明：

- `project_id` / `issue_id` 均为本地数据库主键（非 slug / iid）。
- `run_no` 用于区分同一 issue 的多次 run。
- 建议保存：
  - `issues.issue_dir = work_dir/<project_id>/<issue_id>`
  - `issue_runs.git_tree_path = .../git-tree`
  - `issue_runs.agent_run_dir = .../agent/runs/<run_no>`
- `git-tree` 与 `agent` 目录职责分离，避免执行临时文件污染代码工作区。

推荐约束：

- 每个 issue 只有一个 `git-tree`。
- 每个 run 只有一个 `agent_run_dir`。
- run 结束后可按策略清理 `agent/runs/<run_no>`，`git-tree` 可复用。

## 4. DB 配置与 GORM 方言切换

系统仅维护一套 GORM 实现，通过 `db.driver` 选择方言：

- `sqlite` -> `gorm.io/driver/sqlite`
- `postgres` -> `gorm.io/driver/postgres`

运行时 DB client 维护：

- `sqlDialect string`（`sqlite` / `postgres`）

规则：

- 常规 CRUD 统一用 GORM 兼容写法。
- 仅在方言敏感点按 `sqlDialect` 分支（如 upsert/锁/少量原生 SQL）。
- `db.driver/sqlite_path/postgres_dsn` 变更默认视为“需重启”项，不做在线热切换。

## 5. 密钥读取约束

- `projects.credential_ref` 保存密钥引用名，不直接存 token。
- `secret.provider=env` 时，读取环境变量：
  - key 格式：`<env_prefix><SANITIZED_REF>`
  - 默认前缀：`AGENT_CODER_SECRET_`
- `SANITIZED_REF` 规则：转大写，非字母数字字符替换为 `_`。
