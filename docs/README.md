# 文档总览

本目录只保留“当前实现仍生效”的设计与规范文档。

## 核心文档

- [architecture.md](architecture.md)：架构、分层与路由分组
- [database-schema.md](database-schema.md)：数据库模型与状态机约束
- [agent-runtime.md](agent-runtime.md)：Agent 抽象与 worker 调用行为
- [config-runtime.md](config-runtime.md)：配置模型与工作区规则
- [automation-codex-plan.md](automation-codex-plan.md)：`run_codex_on_plan.py` 执行流
- [code-style/README.md](code-style/README.md)：代码规范
- [requirements.md](requirements.md)：产品需求定稿与验收标准

## 子目录

- [plans/README.md](plans/README.md)：计划文档规范与当前计划

## 清理原则

- 同一主题不长期保留 `v1/v2/v3` 并行版本。
- 过时草稿默认清理，历史需要时通过 Git 历史追溯。
