# 数据表设计（V1）

> 适配场景：GitLab 优先、多项目、`is_admin` 简化权限、Issue 自动开发闭环。  
> 数据库：SQLite / PostgreSQL（同一套 GORM 模型，方言敏感点由 `sqlDialect` 分支处理）。

## 1. 实体关系

- `users` 1:N `projects`（`projects.created_by`）
- `projects` 1:N `issues`
- `projects` 1:N `prompt_templates`（按 `project_key` 绑定模板覆盖）
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
| last_issue_sync_at | TIMESTAMP NULL | 项目 issue 同步游标（`updated_after`） |
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
| issue_dir | TEXT NOT NULL | issue 根目录绝对路径 |
| branch_name | VARCHAR(128) NULL | 当前分支 |
| mr_iid | BIGINT NULL | 当前 MR iid |
| mr_url | TEXT NULL | 当前 MR URL |
| current_run_id | BIGINT NULL | 当前运行 ID（可空，由应用层维护） |
| last_synced_at | TIMESTAMP | NOT NULL |
| closed_at | TIMESTAMP NULL | 远端关闭时间 |
| close_reason | VARCHAR(32) NULL | 本地关闭原因：`merged/manual/need_human_merge` |
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
| run_kind | VARCHAR(16) | NOT NULL，`dev/merge` |
| trigger_type | VARCHAR(16) | NOT NULL，`scheduler/manual/rework/retry` |
| status | VARCHAR(16) | NOT NULL，`queued/running/succeeded/failed/canceled` |
| agent_role | VARCHAR(16) | NOT NULL，`dev/review/merge`，表示当前或最近执行角色 |
| loop_step | INT | NOT NULL，当前循环步（从 1 开始） |
| max_loop_step | INT | NOT NULL，单次 run 最大循环步阈值 |
| queued_at | TIMESTAMP | NOT NULL |
| started_at | TIMESTAMP NULL | |
| finished_at | TIMESTAMP NULL | |
| branch_name | VARCHAR(128) | NOT NULL |
| base_sha | VARCHAR(64) NULL | |
| head_sha | VARCHAR(64) NULL | |
| mr_iid | BIGINT NULL | |
| mr_url | TEXT NULL | |
| git_tree_path | TEXT NOT NULL | 代码工作区目录绝对路径（通常 `<issue_dir>/git-tree`） |
| agent_run_dir | TEXT NOT NULL | agent run 运行目录绝对路径（通常 `<issue_dir>/agent/runs/<run_no>`） |
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
- `INDEX (issue_id, run_kind, status, created_at)`
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

## 2.6 prompt_templates

项目级 Prompt 覆盖表。默认 Prompt 不落库，来自代码内嵌 markdown（`go:embed`）；仅覆盖内容写入本表。

| 字段 | 类型 | 约束/说明 |
|---|---|---|
| id | BIGINT | PK |
| project_key | VARCHAR(64) | NOT NULL，项目键 |
| run_kind | VARCHAR(16) | NOT NULL，`dev/merge` |
| agent_role | VARCHAR(16) | NOT NULL，`dev/review/merge` |
| content | TEXT | NOT NULL，markdown 模板内容 |
| created_at | TIMESTAMP | NOT NULL |
| updated_at | TIMESTAMP | NOT NULL |

索引与约束：

- `UNIQUE (project_key, run_kind, agent_role)`
- `INDEX (project_key, run_kind)`

说明：

- 生效模板为“项目覆盖优先，否则回退 embedded default”。
- 仅允许合法组合：
  - `dev/dev`
  - `dev/review`
  - `merge/merge`
  - `merge/review`

## 3. 枚举建议

`issues.lifecycle_status` 建议值：

- `registered`
- `running`
- `human_review`
- `rework`
- `verified`
- `failed`
- `closed`

`issues.close_reason` 建议值（仅 `lifecycle_status=closed` 时有意义）：

- `merged`：自动合并完成后关闭
- `manual`：远端被人工关闭
- `need_human_merge`：issue tracker 不允许自动合并，转人工合并

`issue_runs.status` 建议值：

- `queued`
- `running`
- `succeeded`
- `failed`
- `canceled`

`issue_runs.run_kind` 建议值：

- `dev`
- `merge`

`issue_runs.agent_role` 建议值：

- `dev`
- `review`
- `merge`

## 4. 状态机定义（细化）

## 4.1 issues.lifecycle_status

状态含义：

- `registered`：已满足门禁并完成本地登记（仅 `Agent Ready` 才会入库），可排队执行
- `running`：当前存在进行中的 run
- `human_review`：MR 已创建，等待人工评审
- `rework`：人工要求返工
- `verified`：人工确认可合并
- `failed`：自动流程失败且已达最大重试上限
- `closed`：issue 已关闭（结合 `close_reason` 区分关闭原因）

切换条件（主路径）：

- `registered -> running`
  - 条件：创建 `run_kind=dev` 的 `issue_runs` 记录并被 worker 拉起（`status=running`）
- `running -> human_review`
  - 条件：`run_kind=dev` 的 run 成功并已创建/更新 MR，打 `Human Review` 标签
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
- `verified -> running`
  - 条件：创建 `run_kind=merge` 的 `issue_runs` 并启动自动合并
- `running -> closed`
  - 条件：`run_kind=merge` 的 run 成功，MR 合并完成，打 `Merged` 标签并关闭 issue，`close_reason=merged`
- `running -> closed`
  - 条件：merge API 返回“需要人工合并”（`ErrNeedHumanMerge`），`close_reason=need_human_merge`
- `* -> closed`
  - 条件：远端 issue 被关闭（轮询同步到关闭态），`close_reason=manual`

约束：

- 只有带 `Agent Ready` 的 issue 才会被创建为 `registered`。
- 仅看到 `In Progress/Human Review/...` 不会进入本地状态机。

## 4.2 issue_runs.status

状态含义：

- `queued`：已创建 run，等待 worker
- `running`：执行中
- `succeeded`：当前 run 完成（`dev` 为完成开发评审，`merge` 为完成合并评审）
- `failed`：执行失败（可重试）
- `canceled`：被人工取消或被新 run 抢占

切换条件：

- `queued -> running`：worker 抢占任务
- `running -> succeeded`：`review` 判定“当前 run_kind 目标已完成并通过”
- `running -> failed`：循环步超过阈值仍未完成，或出现不可继续错误
- `queued|running -> canceled`：人工取消

说明：

- 建议“每次重试创建新 run”，便于审计；`run_no` 递增。
- `issues.current_run_id` 始终指向最新 active run（`queued/running`），由应用层维护。
- 单 issue 仅允许一个 active run（`queued/running`）属于应用层约束，不强依赖数据库硬约束。

## 4.3 单次 run 的 agent loop 规则

一次 `issue_run` 内部执行固定循环，不拆 `issue_sub_runs`。循环由 `run_kind` 决定：

- `run_kind=dev`：
  1. `dev` 执行开发
  2. `review` 执行检查
  3. 若判定完成则结束为 `succeeded`
  4. 否则 `loop_step += 1`，回到步骤 1
- `run_kind=merge`：
  1. `merge` 执行 rebase/merge 与冲突解决
  2. `review` 校验是否可继续合并
  3. 若判定完成则结束为 `succeeded`
  4. 否则 `loop_step += 1`，回到步骤 1

终止条件：

- `loop_step > max_loop_step` 且 `review` 仍不通过 -> `failed`
- run 过程中发生不可恢复错误 -> `failed`

追踪方式（扁平化）：

- 当前轮次保存在 `issue_runs.loop_step`
- 当前/最近执行角色保存在 `issue_runs.agent_role`
- 详细过程写入 `run_logs`（每行一条）

## 5. issue_run 工作目录设计

目录采用“issue 级 + run 级”双层设计：

- 配置项：`work.work_dir`（默认 `.agent-coder/workdirs`）
- issue 根目录：
  - `<workdir_root>/<project_id>/<issue_id>`
- issue 级代码目录（复用）：
  - `<issue_dir>/git-tree`
- run 级 agent 目录（每次 run 独立）：
  - `<issue_dir>/agent/runs/<run_no>`

示例：

- `.agent-coder/workdirs/12/98/git-tree`
- `.agent-coder/workdirs/12/98/agent/runs/3`

推荐实现：

1. 每个项目维护本地仓库缓存（mirror 或主 clone）
2. issue 级 `git-tree` 使用 `git worktree` 管理（按 issue 复用）
3. 每次 run 仅在 `agent_run_dir` 写入输入/输出/日志元数据
4. run 结束后按策略清理 `agent_run_dir`（成功立即清理，失败保留 N 天）

目录约束：

- 同一 issue 的 `git-tree` 唯一（`issues.issue_dir` 决定）
- 同一 run 的 `agent_run_dir` 唯一，不复用
- `run_logs` 与 `agent_run_dir` 一一可追溯

## 6. Issue 同步游标规则

- Worker 调用 issue tracker 使用：
  - `state=all`
  - `updated_after=projects.last_issue_sync_at`（若为空则全量首扫）
- 同步成功后回写 `projects.last_issue_sync_at`。
- 本地入库门禁不变：仅当远端 issue 带 `Agent Ready` 且本地不存在时创建 `issues` 记录。

## 7. SQLite / PostgreSQL 兼容约束

- 统一使用一套 GORM model。
- 在 DAL 中维护 `sqlDialect`（`sqlite/postgres`）。
- 方言分支仅用于：
  - upsert 细节
  - 锁语句差异
  - 少量原生 SQL
- 其余 CRUD 查询统一走 GORM 兼容写法。
