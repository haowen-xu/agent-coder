# 计划文件规范

`scripts/run_codex_on_plan.py` 不解析计划结构本身，而是将计划文件路径直接交给 codex 执行。

## 建议结构

```markdown
# 任务标题

## 背景
- 业务目标
- 约束条件

## Task 列表
- [ ] T1: 任务描述
  - 验收标准
  - 相关文件
- [ ] T2: 任务描述
  - 验收标准
  - 相关文件

## 验证要求
- 必跑命令
- 通过标准
```

## 示例

- 计划示例：[`example.md`](example.md)

## 当前计划

- [`2026-03-20-agent-autocoding-implementation-plan-v3-final.md`](2026-03-20-agent-autocoding-implementation-plan-v3-final.md)

## 清理策略

- 目录仅保留“当前可执行”计划。
- 历史迭代版本（如 `v1/v2`）默认清理，必要时通过 Git 历史追溯。
