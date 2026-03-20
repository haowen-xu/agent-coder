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

1. 初始化 `loop_step=1`，并设置 `agent_role=dev_agent`
2. 从 `issue_runs.git_tree_path` 取代码工作目录
3. 从 `issue_runs.agent_run_dir` 取 run 运行目录
4. `dev_agent` 执行开发，写 `run_logs`
5. 切换 `agent_role=review_agent`，执行 review，写 `run_logs`
6. 若 review 判定完成，`issue_runs.status=succeeded`
7. 若未完成，则 `loop_step += 1` 并回到步骤 4
8. 若 `loop_step > max_loop_step`，`issue_runs.status=failed`
9. 同步更新：
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
