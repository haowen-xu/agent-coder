# AI Agent 导航

> 关键约束与协作规范请优先阅读下列文档。

## 快速入口

- 文档总览: [docs/README.md](docs/README.md)
- 工程架构: [docs/architecture.md](docs/architecture.md)
- 数据库设计: [docs/database-schema.md](docs/database-schema.md)
- Agent 运行设计: [docs/agent-runtime.md](docs/agent-runtime.md)
- 配置与运行时规范: [docs/config-runtime.md](docs/config-runtime.md)
- 自动化执行流: [docs/automation-codex-plan.md](docs/automation-codex-plan.md)
- 代码规范: [docs/code-style/README.md](docs/code-style/README.md)
- 产品需求定稿: [docs/requirements.md](docs/requirements.md)
- 计划文件规范: [docs/plans/README.md](docs/plans/README.md)

## 当前约束

- 计划执行自动化目前仅支持 `scripts/run_codex_on_plan.py`。
- Autofix 产物目录统一使用 `.ai-docs/autofix/`。
- 仓库内不再维护内置 agents 脚本，统一由外部 Agent-Coder 管理。
