# Issue Draft: 引入 Plan Agent + Todo 闭环开发流程（基于当前代码）

> 日期：2026-03-20  
> 状态：Draft  
> 目标版本：V1.1（在现有 V1 基础上增量改造）

## 1. 背景（当前代码现状）

当前实现已经具备 `dev/review/merge` 三类角色和 `issue -> run -> MR` 主链路，但还不满足“以 todo-list 驱动开发”的流程要求。

关键现状（代码位置）：

- Issue Tracker 抽象接口仅有 6 个方法，不支持“更新已有 note”。
  - `ListIssues / SetIssueLabels / CreateIssueNote / CloseIssue / EnsureMergeRequest / MergeMergeRequest`
  - 见 `internal/infra/issuetracker/port.go`
- Worker 当前主循环是：`(dev|merge) -> review`。
  - 在 `executeRun` 内通过 `for step := run.LoopStep; step <= run.MaxLoopStep; step++` 执行
  - 见 `internal/service/worker/service.go`
- 当前流程没有 `plan` 角色，也没有“本地 todo 文件”协议。
- `CreateIssueNote` 当前只新增评论，不返回 note id，也无法修改历史评论。
- 同步 issue 时只拉取基础字段（title/state/labels/...），未包含 issue 描述正文，限制了计划拆解质量。

## 2. 需求目标

把现有开发流程改造成“Plan -> Dev(todo 打钩) -> Review(可能给新 todo)”的闭环，且由 Go Worker 代码主控。

目标流程（用户要求）：

1. 启动 `plan` agent：结合 issue + 代码库输出 todo-list。
2. `dev` agent 面向 todo-list 开发并打钩；在启动 dev 前把 todo 落本地文件并允许 dev 修改。
3. Go 主循环检查 todo 是否全部完成；未完成则继续 dev。
4. `review` agent 检查代码与 todo，可给出新的 comment/todo-list。
5. 若 review 给出新 todo，则继续下一轮；直到 review 不再给新 todo 才结束 run。
6. 每次 `plan/dev/review` 产生的 comment/todo 都通过本地文件产出，由 Worker 统一上传到 issue tracker；旧 todo 的更新也要回写到 issue tracker。

## 3. In Scope / Out of Scope

In Scope：

- 增加 `plan` 角色（先覆盖 `run_kind=dev`）。
- 增加本地文件协议（plan/dev/review 输入输出约定）。
- Worker 主循环重构为“外层 plan/review，内层 dev 打钩”。
- 扩展 Issue Tracker 接口以支持 note 更新。
- GitLab provider 实现 note 修改。
- 将 todo/comment 通过 Worker 同步到 issue tracker（创建或更新）。
- 数据结构最小增量支持（角色枚举、note id 跟踪等）。

Out of Scope（本 issue 不做）：

- 复杂多人协作冲突合并策略（仅保留单 worker 主控）。
- 跨 provider（GitHub/Jira）实现。
- WebUI 的复杂可视化编辑器（先用已有 run_logs + issue tracker note）。

## 4. 关键结论

## 4.1 `CreateIssueNote` 是否支持 markdown todo-list

支持。GitLab issue note 本质是 markdown 文本，`- [ ]` / `- [x]` checklist 可直接展示。

结论：

- 现有 `CreateIssueNote(body string)` 可以发布 todo-list comment。
- 但要“更新旧 todo”，必须新增“修改 note”能力。

## 4.2 是否需要新增 `ModifyIssueNote` 接口

需要。否则每一轮只能追加 comment，无法稳定维护“同一份 canonical todo-list”。

建议命名：`UpdateIssueNote`（比 `Modify` 更贴近现有命名）。

## 5. 设计方案（结合当前代码）

## 5.1 Issue Tracker 抽象扩展

文件：`internal/infra/issuetracker/port.go`

新增/调整：

1. `CreateIssueNote` 返回 note 元信息（至少 note id）
2. 新增 `UpdateIssueNote`
3. 新增 `GetIssue`（拉取 issue 正文，供 plan 参考）
4. 可选新增 `ListIssueNotes`（用于恢复/排障）

建议签名：

```go
type IssueNote struct {
    ID        int64
    Body      string
    WebURL    string
    CreatedAt time.Time
    UpdatedAt time.Time
}

type IssueDetail struct {
    IID         int64
    UID         string
    Title       string
    Description string
    State       string
    Labels      []string
    WebURL      string
    ClosedAt    *time.Time
    UpdatedAt   time.Time
}

type Client interface {
    ListIssues(ctx context.Context, project db.Project, opt ListIssuesOptions) ([]Issue, error)
    GetIssue(ctx context.Context, project db.Project, issueIID int64) (*IssueDetail, error)

    SetIssueLabels(ctx context.Context, project db.Project, issueIID int64, labels []string) error

    CreateIssueNote(ctx context.Context, project db.Project, issueIID int64, body string) (*IssueNote, error)
    UpdateIssueNote(ctx context.Context, project db.Project, issueIID int64, noteID int64, body string) (*IssueNote, error)

    CloseIssue(ctx context.Context, project db.Project, issueIID int64) error
    EnsureMergeRequest(ctx context.Context, project db.Project, req CreateOrUpdateMRRequest) (*MergeRequest, error)
    MergeMergeRequest(ctx context.Context, project db.Project, mrIID int64) error
}
```

GitLab 端映射：

- `POST /projects/:id/issues/:issue_iid/notes`（create）
- `PUT /projects/:id/issues/:issue_iid/notes/:note_id`（update）
- `GET /projects/:id/issues/:issue_iid`（get issue detail）

## 5.2 角色与 Prompt 维度调整

当前 `agent_role`：`dev/review/merge`。  
目标新增：`plan`。

影响：

- `db.AgentRolePlan = "plan"`
- `prompt_templates` 合法组合增加：
  - `dev/plan`
  - `dev/dev`
  - `dev/review`
  - `merge/merge`
  - `merge/review`

默认模板文件增加：

- `internal/infra/agent/prompts/defaults/dev.plan.md`

## 5.3 Worker 主循环改造（仅 `run_kind=dev`）

当前 `executeRun`（`internal/service/worker/service.go`）需改为以下结构：

```text
for loop_step in [1..max_loop_step] {
  if loop_step == 1 {
    plan -> 生成 todo
  }

  while todo 存在未完成项 {
    dev -> 修改代码 + 更新 todo 勾选
    同步 todo 到 issue tracker（更新 canonical todo note）
    若连续无进展，判定 blocked/fail
  }

  review -> 输出 review comment + 可选 new_todo
  发布 review comment 到 issue tracker

  if review 未给 new_todo {
    run succeeded
    break
  }

  用 review new_todo 覆盖当前 todo
  同步新 todo 到 issue tracker（更新 canonical todo note）
}

若超过 max_loop_step 仍有 new_todo -> failed
```

说明：

- `loop_step` 表示“plan/review 轮次”，不是 dev 子循环次数。
- dev 子循环可通过 `run_logs.stage=dev_loop` 记录 `dev_attempt`。
- 当前 `autoCommitAndPush` 触发点需从“review pass 后一次”调整为“每轮 dev 结束后按策略提交（可配置）”或“维持 review pass 后提交”；默认建议维持现状，避免提交噪声。

## 5.4 本地文件协议（Agent I/O Contract）

根目录：`<issue_dir>/agent/runs/<run_no>/`

统一子目录：

- `inputs/`
- `outputs/`
- `state/`

核心文件：

1. `state/todo.current.md`
   - 当前生效 todo（markdown checklist）
2. `state/todo.history/<ts>-<source>.md`
   - todo 历史快照（source: plan/review/dev）
3. `outputs/plan.comment.md`
4. `outputs/plan.todo.md`
5. `outputs/dev.comment.md`
6. `outputs/dev.todo.updated.md`
7. `outputs/review.comment.md`
8. `outputs/review.todo.new.md`（可空；空代表无新 todo）
9. `outputs/<role>.result.json`

约束：

- Agent 只能通过上述文件向 Worker 回传 comment/todo。
- Worker 负责校验文件存在性、格式合法性，然后写 run_logs 并同步 issue tracker。
- Worker 不信任 agent 直接调用远端 API，统一由 Go 端执行。

## 5.5 issue tracker comment/todo 同步策略

引入“canonical todo note”：

- 第一次（plan 完成）调用 `CreateIssueNote` 创建 todo note，保存 `todo_note_id`。
- 后续 dev/review 导致 todo 变化时，调用 `UpdateIssueNote` 更新同一条 note。
- plan/dev/review 各自 comment 仍用新增 note（审计可读）。

这样同时满足：

- 有持续更新的“老 todo”（同一 note）
- 有每轮新增 comment（追溯上下文）

## 5.6 数据模型调整（最小必要）

建议改动：

1. `issue_runs`
   - 新增 `todo_note_id BIGINT NULL`（记录 canonical todo note id）
   - 可选 `todo_version INT NOT NULL DEFAULT 0`
2. `issues`
   - 新增 `description TEXT NULL`（可选，但建议；提升 plan 质量）
3. 枚举
   - `agent_role` 增加 `plan`

备注：

- 不新增 `issue_sub_runs`，保持你要求的扁平化追踪。
- 单 issue 单 active run 仍由应用层控制（沿用现有 `scheduleRuns + BindIssueRunIfIdle`）。

## 5.7 决策协议扩展（RESULT_JSON）

在 `base.Decision` 增加 todo 相关字段，避免只靠自由文本判断：

```json
{
  "role": "plan|dev|review|merge",
  "decision": "todo_ready|todo_in_progress|pass|rework|blocked|ready_for_review",
  "summary": "...",
  "next_action": "...",
  "todo_stats": {
    "total": 8,
    "done": 5
  },
  "new_todo": true,
  "blocking_reason": "..."
}
```

兼容策略：

- 旧字段保留；Worker 优先读新字段。
- 若缺失 `todo_stats`，Worker 以 `todo.current.md` 实际解析结果为准。

## 5.8 失败与保护机制

新增保护：

1. `max_loop_step` 超限：`run failed`
2. dev 子循环连续 N 次无 todo 进展：`run failed`（避免死循环）
3. plan/review 输出 todo 非法（无法解析 markdown checklist）：`run failed`
4. issue tracker note 更新失败：可重试 K 次，超限 fail

## 5.9 对现有流程的兼容

- `run_kind=merge` 暂不引入 `plan`，保持 `merge -> review`（后续可扩展）。
- 现有 `issue` 标签状态流不变：`Agent Ready -> In Progress -> Human Review -> Rework/Verified -> Merged`（本地生命周期收敛为 `closed + close_reason`）。
- `CreateIssueNote("agent run failed: ...")` 等失败提示保留。

## 6. 验收标准

功能验收：

1. `run_kind=dev` 启动后，首轮必须先执行 `plan`。
2. 若 todo 未全部勾选，Worker 不允许进入最终成功。
3. review 给出新 todo 时，必须进入下一轮，不得直接 `pass` 结束。
4. issue tracker 上存在：
   - 一条持续更新的 canonical todo note
   - 每轮 plan/dev/review comment
5. `max_loop_step` 超限时 run 必须 `failed`，issue 回退/失败状态符合现有重试策略。

测试验收：

1. 单元测试：
   - todo markdown 解析（done/undone）
   - 无进展检测
   - 角色状态迁移（plan/dev/review）
2. GitLab 集成/e2e：
   - 创建 note + 更新 note 流程
   - todo note 内容按轮次正确更新
3. 回归测试：
   - merge run 现有逻辑不回归

## 7. 实施拆分建议

M1（接口层）：

- 扩展 `issuetracker.Client` 与 GitLab client（GetIssue/CreateNote 返回值/UpdateNote）

M2（数据层）：

- DB 迁移：`issue_runs.todo_note_id`、（可选）`issues.description`
- 枚举新增 `agent_role=plan`

M3（运行时）：

- Worker `executeRun` 重构为新双层循环
- 本地文件协议落地 + run_logs 细化

M4（Prompt）：

- 新增 `dev.plan` 默认模板
- PromptStore 合法组合放开 `dev/plan`

M5（测试）：

- 单测 + e2e + 回归

## 8. 风险与注意事项

1. 现有 codex fallback 逻辑对 `review` 默认 `pass`，会放大误判风险；引入 todo 闭环后建议改为 `rework` 或 `blocked`。
2. 若 issue 正文不入库（无 `description`），plan 质量会明显受限。
3. note 更新失败需要明确重试与降级策略（避免本地与远端 todo 分叉）。

## 9. 需要确认的决策点

1. 是否在本期增加 `GetIssue` 并把 issue `description` 持久化到 `issues` 表？（建议是）
2. dev 子循环“无进展阈值”默认值取多少？（建议 3）
3. todo note 是否固定一条更新，还是每轮新建？（建议固定一条 + 每轮评论增量）
4. `run_kind=merge` 是否后续也接入 `plan`？（本期建议否）
