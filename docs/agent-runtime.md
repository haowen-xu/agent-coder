# Agent 运行设计（当前实现）

> 范围：`internal/infra/agent/*` + `internal/service/worker` 的实际行为。

## 1. 目录结构

```text
internal/infra/agent/
├── base/
│   ├── client.go      # Client 接口
│   └── types.go       # InvokeRequest / InvokeResult / Decision
├── codex/
│   ├── client.go      # codex provider 实现
│   └── client_test.go
├── prompts/
│   ├── defaults.go
│   └── defaults/*.md  # 内置默认模板
└── promptstore/
    └── service.go      # 项目覆盖模板读写与合并
```

## 2. base 抽象

`base.Client` 当前只有 2 个方法：

- `Name() string`
- `Run(ctx context.Context, req InvokeRequest) (*InvokeResult, error)`

`InvokeRequest` 关键字段：

- `RunKind`：`dev` / `merge`
- `Role`：`dev` / `review` / `merge`
- `Prompt`
- `WorkDir`、`RunDir`
- `Timeout`
- `Env`
- `UseSandbox`

`InvokeResult` 关键字段：

- `ExitCode`、`Stdout`、`Stderr`
- `LastMessage`
- `Decision`
- `ThreadID`（字段保留，当前 codex 实现不写入）

## 3. codex provider 行为

实现文件：`internal/infra/agent/codex/client.go`

执行命令：

- 基础参数：`codex exec --skip-git-repo-check`
- `UseSandbox=true`：追加 `--full-auto --sandbox workspace-write`
- `UseSandbox=false`：追加 `--dangerously-bypass-approvals-and-sandbox`

输出解析：

- 从 stdout 中解析 `RESULT_JSON` 代码块。
- 解析失败时返回降级决策：`decision=blocked`，并给出 `blocking_reason`。
- 命令执行失败且未得到阻塞决策时，会强制转成 `blocked`。

## 4. Worker 对 Agent 的调用约定

调用点：`internal/service/worker/service.go`

- 每轮先执行主角色：
  - `run_kind=dev` -> `role=dev`
  - `run_kind=merge` -> `role=merge`
- 然后执行 `role=review`
- `review.pass` 才视为本轮通过；否则进入下一轮
- 超过 `max_loop_step` 则 run 失败

当前实现的判定重点：

- 主角色返回 `blocked` 会立即失败。
- `review` 阶段只有 `pass` 会结束成功；其他值按“未通过”处理。

## 5. Prompt 组装协议

Worker 会注入公共前缀上下文，再拼接角色模板：

- role / run_kind / loop_step
- repo_dir / run_dir
- issue 标题与 IID
- base_branch / work_branch
- 硬约束（只改 repo_dir、禁止伪造成功、必须输出 `RESULT_JSON`）

角色模板来源：

1. 项目覆盖（DB 表 `prompt_templates`）
2. 无覆盖时回退 embedded defaults（`internal/infra/agent/prompts/defaults/*.md`）

合法模板键：

- `dev/dev`
- `dev/review`
- `merge/merge`
- `merge/review`

## 6. Sandbox 策略

当前由 `shouldUseSandboxForRole` 决定：

- `dev` / `merge`：固定关闭 sandbox
- `review`（以及保留的 `plan` 角色名）：使用项目配置 `sandbox_plan_review`

## 7. Prompt 管理接口（Admin）

- `GET /api/v1/admin/prompts/defaults`
- `GET /api/v1/admin/projects/:projectKey/prompts`
- `PUT /api/v1/admin/projects/:projectKey/prompts/:runKind/:agentRole`
- `DELETE /api/v1/admin/projects/:projectKey/prompts/:runKind/:agentRole`

## 8. 与旧设计差异说明

以下内容在当前代码中未实现，不应作为现状依赖：

- `base.Client.RunResume(...)`
- `base/runner.go`、`base/events.go`
- 基于 `thread.started/item.completed` 的统一事件流解析

如需恢复这些能力，应先补齐 `base` 抽象与 `codex` 实现，再更新文档。
