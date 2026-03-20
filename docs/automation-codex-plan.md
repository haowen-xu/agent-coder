# 自动化执行流（Codex Plan）

本文档描述基于 `scripts/run_codex_on_plan.py` 的计划执行自动化流程。

## 设计目标

- 以非交互方式多轮调用 codex 执行计划任务。
- 每轮执行后强制检查计划是否完成，直到完成或达到最大轮次。
- 可选复用 codex 会话上下文，减少重复上下文注入成本。

## 命令示例

```bash
python3 scripts/run_codex_on_plan.py \
  --plan-file docs/plans/example.md \
  --resume-context \
  --sandbox \
  --max-iteration 50
```

## 常用参数

- `--plan-file`：计划文件路径（必填）。
- `--status-file`：可选状态文件，记录每轮完成情况。
- `--resume-context`：是否启用 `codex exec resume` 续跑。
- `--session-id`：指定已有 codex 线程 ID。
- `--sandbox/--no-sandbox`：是否在沙盒模式运行 codex。
- `--dry-run`：仅打印将执行的命令，不真正调用 codex。
- `--timeout-sec`：单轮调用超时时间（秒）。

## 输出文件

若未显式指定，脚本会使用与计划同名文件：

- `*.json`：上下文文件（`thread_id`）
- `*.log`：执行日志文件

## 建议执行顺序

1. 先用 `--dry-run` 检查命令和参数是否正确。
2. 再正式运行，观察 `*.log` 中每轮结果。
3. 若中断，可通过 `--resume-context` 或 `--session-id` 续跑。

## 范围说明

- 当前自动化仅覆盖 `scripts/run_codex_on_plan.py`。
- `scripts/agents/` 暂不纳入自动化流程。
