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

1. 从 `issue_runs.workdir_path` 取工作目录
2. 调用 `infra/agent` 执行一次工作提示词
3. 记录 stdout/stderr 到 `run_logs`
4. 根据结果更新：
   - `issue_runs.status`
   - `issues.lifecycle_status`
   - `issues.current_run_id`

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
