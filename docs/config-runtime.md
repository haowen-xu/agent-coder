# 配置与本地工作区规范

> 本文档以当前 `internal/config/config.go` 与 worker 实现为准。

## 1. 统一配置文件

系统使用单一 `config.yaml`，由 Viper 加载并做默认值填充 + 校验。

示例（与 `config.example.yaml` 一致）：

```yaml
app:
  name: agent-coder
  env: dev

server:
  host: 0.0.0.0
  port: 8080
  read_timeout: 15s
  write_timeout: 15s
  shutdown_timeout: 10s

log:
  level: info
  format: text # text | json
  add_source: false

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

auth:
  session_ttl: 72h

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

# 兼容字段（推荐改为 repo_provider）
issue_provider:
  http_timeout_sec: 30

bootstrap:
  admin_username: admin
  admin_password: admin123
```

## 2. config 包运行时行为

`internal/config` 使用原子指针保存当前配置快照：

- `Current() *Config`：读取快照
- `Replace(*Config)`：替换快照

约束：

- 业务层只读配置，不直接修改配置对象。
- 加载流程固定为 `Load -> Validate -> Store`。
- 校验失败时不得覆盖旧配置。

## 3. 关键校验规则

- `server.port` 必须在 `1~65535`
- `server.read_timeout/write_timeout/shutdown_timeout` 必须是合法 duration
- `db.conn_max_lifetime` 必须是合法 duration
- `db.driver=sqlite` 时要求 `db.sqlite_path`
- `db.driver=postgres` 时要求 `db.postgres_dsn`
- `agent.codex.timeout_sec/max_retry/max_loop_step` 必须大于 0
- `bootstrap.admin_username/admin_password` 不可为空

## 4. 仓库平台超时配置兼容

当前优先级：

1. `repo_provider.http_timeout_sec`
2. `issue_provider.http_timeout_sec`（兼容旧配置）
3. 默认值 `30`

说明：新配置建议统一使用 `repo_provider`。

## 5. 本地工作区目录

根目录由 `work.work_dir` 控制，worker 目录约定：

```text
<work_dir>/
  <project_id>/
    <issue_id>/
      git-tree/
      agent/
        runs/
          <run_no>/
```

字段映射：

- `issues.issue_dir = <work_dir>/<project_id>/<issue_id>`
- `issue_runs.git_tree_path = <issue_dir>/git-tree`
- `issue_runs.agent_run_dir = <issue_dir>/agent/runs/<run_no>`

## 6. Agent sandbox 行为说明

`agent.codex.sandbox` 是 provider 初始化参数；worker 实际按角色决定是否启用 sandbox：

- `dev` / `merge`：固定关闭
- `review`：受项目配置 `projects.sandbox_plan_review` 控制

因此，排查 sandbox 行为时需同时看全局配置与项目配置。

## 7. 密钥读取约束

- 项目优先使用 `project_token`（若设置）。
- 未设置时使用 `credential_ref` 通过 secret manager 读取。
- `secret.provider=env` 时环境变量键格式：
  - `<env_prefix><SANITIZED_REF>`
  - `SANITIZED_REF`：大写且非字母数字替换为 `_`
