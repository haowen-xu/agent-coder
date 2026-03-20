# Agent 运行设计（infra/agent）

> 目标：将 Agent 执行能力封装为基础设施层能力，先支持 `codex`，后续可扩展其他 provider。

## 1. 目录规划

```text
internal/infra/agent/
├── base/
│   ├── types.go        # 通用请求/响应模型
│   ├── client.go       # 抽象接口
│   ├── runner.go       # 通用命令执行骨架
│   └── events.go       # JSONL 事件解析
└── codex/
    ├── client.go       # codex 具体实现
    └── prompts.go      # 工作/检查提示词模板（可选）
```

## 2. base 抽象（建议）

`base` 只定义“怎么调用 agent”，不绑定业务流程。

核心模型：

- `InvokeRequest`
  - `prompt`
  - `work_dir`
  - `timeout`
  - `env`
- `InvokeResult`
  - `exit_code`
  - `stdout`
  - `stderr`
  - `thread_id`
  - `last_message`

核心接口：

- `Client`
  - `Name() string`
  - `Run(ctx, req)`
  - `RunResume(ctx, threadID, req)`（可选）

## 3. codex 具体实现（建议）

`codex` 实现负责命令拼装与结果解析：

- 首次执行：`codex exec ...`
- 续跑执行：`codex exec resume <thread_id> ...`
- 统一开启 JSON 输出并解析 `thread.started` / `item.completed`

关键行为：

- 支持 `sandbox` 开关
- 支持超时控制
- 将 `thread_id` 回传给上层（用于后续 run 续跑）
- 将 `last_message` 回传给上层（用于完成度判断）

## 4. 与业务层的职责边界

- `infra/agent` 只负责“执行 agent 命令”。
- `service/worker` 负责状态机推进：
  - 何时创建 run
  - 何时重试
  - 何时写入 `run_logs`
  - 何时更新 issue 标签与生命周期状态

## 5. 与 issue_runs 的对接

每次执行 run 时：

1. 读取 `run_kind`（`dev` 或 `merge`）
2. 初始化 `loop_step=1`
3. 根据 `run_kind` 设置初始 `agent_role`：
   - `dev` run -> `agent_role=dev`
   - `merge` run -> `agent_role=merge`
4. 从 `issue_runs.git_tree_path` 取代码工作目录
5. 从 `issue_runs.agent_run_dir` 取 run 运行目录
6. 执行当前角色（`dev` 或 `merge`），写 `run_logs`
7. 切换 `agent_role=review`，执行 review，写 `run_logs`
8. 若 review 判定完成，`issue_runs.status=succeeded`
9. 若未完成，则 `loop_step += 1` 并回到步骤 6
10. 若 `loop_step > max_loop_step`，`issue_runs.status=failed`
11. 同步更新：
   - `issues.lifecycle_status`
   - `issues.current_run_id`

目录约定：

- issue 级：`<work_dir>/<project_id>/<issue_id>/git-tree`
- run 级：`<work_dir>/<project_id>/<issue_id>/agent/runs/<run_no>`

## 6. 配置项建议

建议在 `config.yaml` 增加：

```yaml
agent:
  provider: codex
  codex:
    binary: codex
    sandbox: true
    timeout_sec: 7200
    max_retry: 5
```

配置读取由 `internal/config` 统一管理，运行时通过 atomic pointer 快照获取。

## 7. Codex Prompt 协议（V1）

为保证可循环执行与可机器解析，采用“公共前缀 + 角色模板 + 统一结果 JSON”。

说明：

- 这套协议当前由 `infra/agent/codex` 使用。
- 结构是通用的，后续接其他 agent provider 可复用同一协议。

### 7.1 公共前缀模板

```text
你在 agent-coder 中执行任务。
当前上下文：
- role: {{agent_role}}              # dev | review | merge
- run_kind: {{run_kind}}            # dev | merge
- loop_step: {{loop_step}} / {{max_loop_step}}
- repo_dir: {{git_tree_path}}
- run_dir: {{agent_run_dir}}
- issue: {{issue_title}} (#{{issue_iid}})
- acceptance: {{acceptance_criteria}}
- base_branch: {{base_branch}}
- work_branch: {{work_branch}}

硬约束：
1) 只在 repo_dir 内工作。
2) 不改与当前 issue 无关的文件。
3) 命令失败必须明确报告，不得伪造成功。
4) 最后一段必须输出 RESULT_JSON 代码块（严格 JSON）。
```

统一输出（最后一段）：

```json
{
  "role": "dev|review|merge",
  "decision": "ready_for_review|blocked|pass|rework",
  "summary": "string",
  "changed_files": ["path/to/file"],
  "tests": [
    {
      "cmd": "go test ./...",
      "ok": true,
      "output": "..."
    }
  ],
  "next_action": "string",
  "blocking_reason": "string"
}
```

### 7.2 角色模板

`dev` 模板要点：

- 目标：实现 issue 需求并达到可评审状态。
- 先给最小实现计划，再执行改动。
- 至少运行与改动直接相关的测试。
- 禁止执行 merge/rebase（由 `merge` 角色负责）。
- `decision` 仅允许：
  - `ready_for_review`
  - `blocked`

`review` 模板要点：

- 目标：判断当前 run 是否结束；不通过则给出可执行返工清单。
- 默认不改代码，重点做完成度与质量评审。
- `decision` 仅允许：
  - `pass`
  - `rework`
- 当 `decision=rework` 时，`next_action` 必须是可直接执行的 checklist。

`merge` 模板要点：

- 目标：推进合并，处理 rebase/merge 冲突并完成最小回归验证。
- 冲突修复后必须记录取舍说明与验证结果。
- 不关闭 issue、不打标签（这些由系统层处理）。
- `decision` 仅允许：
  - `ready_for_review`
  - `blocked`

### 7.3 Worker 判定映射

- `run_kind=dev`：
  - `dev.ready_for_review` -> 调用 `review`
  - `review.pass` -> `issue_runs.status=succeeded`
  - `review.rework` -> `loop_step += 1`
- `run_kind=merge`：
  - `merge.ready_for_review` -> 调用 `review`
  - `review.pass` -> `issue_runs.status=succeeded`
  - `review.rework` -> `loop_step += 1`
- 任意角色返回 `blocked`：
  - 记录 `blocking_reason` 到 `run_logs` / `issue_runs.error_summary`
  - 由 worker 按重试策略决定下一步

## 8. Prompt 存储与覆盖接口

## 8.1 默认 Prompt（代码内置）

- 默认模板文件放在 Go 项目 markdown 文件中，并通过 `go:embed` 嵌入二进制。
- 建议路径：
  - `internal/infra/agent/prompts/defaults/dev.dev.md`
  - `internal/infra/agent/prompts/defaults/dev.review.md`
  - `internal/infra/agent/prompts/defaults/merge.merge.md`
  - `internal/infra/agent/prompts/defaults/merge.review.md`

运行时回退规则：

1. 先查项目级数据库覆盖模板
2. 无覆盖则使用 embedded default

## 8.2 数据库存储（项目覆盖）

- 表：`prompt_templates`
- 唯一键：`(project_key, run_kind, agent_role)`
- 仅存“覆盖内容”；默认模板不写库

## 8.3 管理接口（Admin）

说明：以下接口为管理接口，生产环境需挂在 admin 鉴权中间件后。

- `GET /api/v1/admin/prompts/defaults`
  - 返回全部 embedded 默认模板
- `GET /api/v1/admin/projects/{projectKey}/prompts`
  - 返回项目有效模板（覆盖 + 默认合并结果）
- `PUT /api/v1/admin/projects/{projectKey}/prompts/{runKind}/{agentRole}`
  - 写入/更新项目覆盖模板
  - body: `{ "content": "..." }`
- `DELETE /api/v1/admin/projects/{projectKey}/prompts/{runKind}/{agentRole}`
  - 删除项目覆盖模板，恢复默认模板

接口约束：

- 仅允许模板 key：
  - `dev/dev`
  - `dev/review`
  - `merge/merge`
  - `merge/review`
