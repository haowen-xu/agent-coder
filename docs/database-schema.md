# 数据表设计（V1）

> 适配场景：GitLab 优先、多项目、`is_admin` 简化权限、Issue 自动开发闭环。  
> 数据库：SQLite / PostgreSQL（同一套 GORM 模型，方言敏感点由 `sqlDialect` 分支处理）。

## 1. 实体关系

- `users` 1:N `projects`（`projects.created_by`）
- `projects` 1:N `issues`
- `issues` 1:N `issue_runs`
- `issue_runs` 1:N `run_logs`

## 2. 表结构

## 2.1 users

用户与权限（最简模型：`is_admin`）。

| 字段 | 类型 | 约束/说明 |
|---|---|---|
| id | BIGINT | PK |
| username | VARCHAR(64) | NOT NULL, UNIQUE |
| password_hash | VARCHAR(255) | NOT NULL |
| is_admin | BOOLEAN | NOT NULL, DEFAULT FALSE |
| enabled | BOOLEAN | NOT NULL, DEFAULT TRUE |
| last_login_at | TIMESTAMP NULL | 最近登录时间 |
| created_at | TIMESTAMP | NOT NULL |
| updated_at | TIMESTAMP | NOT NULL |
| deleted_at | TIMESTAMP NULL | 软删除 |

索引与约束：

- `UNIQUE (username)`
- `INDEX (enabled)`

## 2.2 projects

项目绑定（repo + issue provider + 标签映射）。

| 字段 | 类型 | 约束/说明 |
|---|---|---|
| id | BIGINT | PK |
| project_key | VARCHAR(64) | NOT NULL, UNIQUE，系统内项目唯一键 |
| project_slug | VARCHAR(128) | NOT NULL, UNIQUE，平台可读标识（如 `group/repo`） |
| name | VARCHAR(128) | NOT NULL |
| provider | VARCHAR(16) | NOT NULL，默认 `gitlab` |
| provider_url | VARCHAR(255) | NOT NULL，API endpoint（如 `https://gitlab.x/api/v4`） |
| repo_url | TEXT | NOT NULL，代码仓库地址 |
| default_branch | VARCHAR(64) | NOT NULL，默认 `main` |
| issue_project_id | VARCHAR(64) NULL | 远端 issue 项目标识，可空 |
| credential_ref | VARCHAR(128) | NOT NULL，密钥引用 |
| poll_interval_sec | INT | NOT NULL，默认 60 |
| enabled | BOOLEAN | NOT NULL，默认 TRUE |
| label_agent_ready | VARCHAR(64) | NOT NULL，默认 `Agent Ready` |
| label_in_progress | VARCHAR(64) | NOT NULL，默认 `In Progress` |
| label_human_review | VARCHAR(64) | NOT NULL，默认 `Human Review` |
| label_rework | VARCHAR(64) | NOT NULL，默认 `Rework` |
| label_verified | VARCHAR(64) | NOT NULL，默认 `Verified` |
| label_merged | VARCHAR(64) | NOT NULL，默认 `Merged` |
| created_by | BIGINT | NOT NULL，FK -> users.id |
| created_at | TIMESTAMP | NOT NULL |
| updated_at | TIMESTAMP | NOT NULL |
| deleted_at | TIMESTAMP NULL | 软删除 |

索引与约束：

- `UNIQUE (project_key)`
- `UNIQUE (project_slug)`
- `INDEX (enabled, poll_interval_sec)`
- `FOREIGN KEY (created_by) REFERENCES users(id)`

## 2.3 issues

Issue 主实体与执行门禁状态。

| 字段 | 类型 | 约束/说明 |
|---|---|---|
| id | BIGINT | PK |
| project_id | BIGINT | NOT NULL，FK -> projects.id |
| issue_iid | BIGINT | NOT NULL，项目内 issue 编号 |
| issue_uid | VARCHAR(64) NULL | 远端全局 ID（可选） |
| title | TEXT | NOT NULL |
| state | VARCHAR(16) | NOT NULL，`open/closed/...` |
| labels_json | TEXT | NOT NULL，存储标签快照（JSON 数组） |
| registered_at | TIMESTAMP | NOT NULL，写入本地登记时间 |
| lifecycle_status | VARCHAR(24) | NOT NULL，默认 `registered` |
| branch_name | VARCHAR(128) NULL | 当前分支 |
| mr_iid | BIGINT NULL | 当前 MR iid |
| mr_url | TEXT NULL | 当前 MR URL |
| current_run_id | BIGINT NULL | 当前运行 ID（可空，由应用层维护） |
| last_synced_at | TIMESTAMP | NOT NULL |
| closed_at | TIMESTAMP NULL | 远端关闭时间 |
| created_at | TIMESTAMP | NOT NULL |
| updated_at | TIMESTAMP | NOT NULL |
| deleted_at | TIMESTAMP NULL | 软删除 |

索引与约束：

- `UNIQUE (project_id, issue_iid)`
- `INDEX (project_id, lifecycle_status)`
- `INDEX (project_id, registered_at)`
- `FOREIGN KEY (project_id) REFERENCES projects(id)`

硬门禁规则：

- 本地 `issues` 表仅保存带 `Agent Ready` 标签的 issue。
- 不带 `Agent Ready` 的远端 issue 不写入本地 `issues` 表。
- 仅凭 `In Progress/Human Review/...` 标签，不允许写入和执行。

## 2.4 issue_runs

每次开发尝试一条 run（同一 issue 可多次 run）。

| 字段 | 类型 | 约束/说明 |
|---|---|---|
| id | BIGINT | PK |
| issue_id | BIGINT | NOT NULL，FK -> issues.id |
| run_no | INT | NOT NULL，issue 内递增序号 |
| trigger_type | VARCHAR(16) | NOT NULL，`scheduler/manual/rework/retry` |
| status | VARCHAR(16) | NOT NULL，`queued/running/succeeded/failed/canceled` |
| queued_at | TIMESTAMP | NOT NULL |
| started_at | TIMESTAMP NULL | |
| finished_at | TIMESTAMP NULL | |
| branch_name | VARCHAR(128) | NOT NULL |
| base_sha | VARCHAR(64) NULL | |
| head_sha | VARCHAR(64) NULL | |
| mr_iid | BIGINT NULL | |
| mr_url | TEXT NULL | |
| workdir_path | TEXT NOT NULL | 本次 run 的工作目录绝对路径 |
| conflict_retry_count | INT | NOT NULL，默认 0 |
| max_conflict_retry | INT | NOT NULL，默认 5 |
| exit_code | INT NULL | |
| error_summary | TEXT NULL | |
| executor_session_id | VARCHAR(128) NULL | agent/codex session id |
| created_by_user_id | BIGINT NULL | 手动触发时记录 |
| created_at | TIMESTAMP | NOT NULL |
| updated_at | TIMESTAMP | NOT NULL |

索引与约束：

- `UNIQUE (issue_id, run_no)`
- `INDEX (issue_id, status, created_at)`
- `INDEX (status, created_at)`
- `FOREIGN KEY (issue_id) REFERENCES issues(id)`
- `FOREIGN KEY (created_by_user_id) REFERENCES users(id)`

## 2.5 run_logs

每个 run 的结构化日志，按行存储。

| 字段 | 类型 | 约束/说明 |
|---|---|---|
| id | BIGINT | PK |
| run_id | BIGINT | NOT NULL，FK -> issue_runs.id |
| seq | INT | NOT NULL，run 内递增序号 |
| at | TIMESTAMP | NOT NULL，日志时间 |
| level | VARCHAR(8) | NOT NULL，`DEBUG/INFO/WARN/ERROR` |
| stage | VARCHAR(32) | NOT NULL，`sync/plan/code/test/mr/conflict/...` |
| event_type | VARCHAR(32) | NOT NULL |
| message | TEXT | NOT NULL |
| payload_json | TEXT NULL | 扩展字段 |
| created_at | TIMESTAMP | NOT NULL |

索引与约束：

- `UNIQUE (run_id, seq)`
- `INDEX (run_id, at)`
- `INDEX (level, at)`
- `FOREIGN KEY (run_id) REFERENCES issue_runs(id)`

## 3. 枚举建议

`issues.lifecycle_status` 建议值：

- `registered`
- `running`
- `human_review`
- `rework`
- `verified`
- `merged`
- `failed`
- `closed`

`issue_runs.status` 建议值：

- `queued`
- `running`
- `succeeded`
- `failed`
- `canceled`

## 4. 状态机定义（细化）

## 4.1 issues.lifecycle_status

状态含义：

- `registered`：已满足门禁并完成本地登记（仅 `Agent Ready` 才会入库），可排队执行
- `running`：当前存在进行中的 run
- `human_review`：MR 已创建，等待人工评审
- `rework`：人工要求返工
- `verified`：人工确认可合并
- `merged`：已完成合并
- `failed`：自动流程失败且已达最大重试上限
- `closed`：issue 已关闭（可由 `merged` 或人工关闭触发）

切换条件（主路径）：

- `registered -> running`
  - 条件：创建 `issue_runs` 记录并被 worker 拉起（`status=running`）
- `running -> human_review`
  - 条件：run 成功并已创建/更新 MR，打 `Human Review` 标签
- `running -> registered`
  - 条件：agent error 且 `retry_count < max_retry`，创建下一次 run 继续执行
- `running -> failed`
  - 条件：agent error 且 `retry_count >= max_retry`
- `human_review -> rework`
  - 条件：人工打 `Rework` 标签或触发返工动作
- `rework -> registered`
  - 条件：重新入队（新建 run）
- `human_review -> verified`
  - 条件：人工打 `Verified` 标签
- `verified -> merged`
  - 条件：MR 合并成功，打 `Merged` 标签并关闭 issue
- `* -> closed`
  - 条件：远端 issue 被关闭（轮询同步到关闭态）

约束：

- 只有带 `Agent Ready` 的 issue 才会被创建为 `registered`。
- 仅看到 `In Progress/Human Review/...` 不会进入本地状态机。

## 4.2 issue_runs.status

状态含义：

- `queued`：已创建 run，等待 worker
- `running`：执行中
- `succeeded`：完成开发和验证，并成功推进到 MR 阶段
- `failed`：执行失败（可重试）
- `canceled`：被人工取消或被新 run 抢占

切换条件：

- `queued -> running`：worker 抢占任务
- `running -> succeeded`：开发/验证/MR 推进成功
- `running -> failed`：命令失败、API 失败、冲突超重试上限或其他不可继续错误
- `queued|running -> canceled`：人工取消

说明：

- 建议“每次重试创建新 run”，便于审计；`run_no` 递增。
- `issues.current_run_id` 始终指向最新 active run（`queued/running`），由应用层维护。
- 单 issue 仅允许一个 active run（`queued/running`）属于应用层约束，不强依赖数据库硬约束。

## 5. issue_run 工作目录设计

`issue_runs.workdir_path` 为必填，并采用统一可计算路径：

- 配置项：`runtime.workdir_root`（默认 `.agent-coder/workdirs`）
- run 目录：
  - `<workdir_root>/<project_key>/issue-<issue_iid>/run-<run_no>`

示例：

- `.agent-coder/workdirs/acme-bot/issue-128/run-3`

推荐实现：

1. 每个项目维护本地仓库缓存（mirror 或主 clone）
2. 每个 run 用 `git worktree` 创建独立目录到 `workdir_path`
3. run 结束后按策略清理（成功立即清理，失败保留 N 天）

目录约束：

- 同一 `workdir_path` 只能绑定一个 run
- 不复用 run 目录，避免脏状态污染
- `run_logs` 与 `workdir_path` 一一可追溯

## 6. SQLite / PostgreSQL 兼容约束

- 统一使用一套 GORM model。
- 在 DAL 中维护 `sqlDialect`（`sqlite/postgres`）。
- 方言分支仅用于：
  - upsert 细节
  - 锁语句差异
  - 少量原生 SQL
- 其余 CRUD 查询统一走 GORM 兼容写法。
