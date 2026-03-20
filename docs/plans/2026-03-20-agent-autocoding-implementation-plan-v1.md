# Agent 自动 Coding 实施计划（V1）

## 1. 目标

基于 GitLab 打通从 Issue 到 MR 的自动化闭环，满足以下已确认策略：

- 定时 pull issue
- 分支命名：`agent-coder/issue-<id>`
- 一个 issue 一个 MR
- 冲突自动重试 5 次

## 2. 里程碑

## M1 项目绑定与存储层

- 新增 `ProjectBinding`、`WorkItem` 表结构与仓储。
- 支持增删改查项目绑定配置（repo + gitlab project + token ref + poll interval）。
- 交付物：
  - 数据模型与迁移
  - 基础 CRUD API

## M2 GitLab 接入层（Issue + MR）

- 封装 GitLab 客户端：
  - 拉取 issue 列表与详情
  - 更新 issue（评论、标签、状态）
  - 创建/查询/更新 MR
- 交付物：
  - `internal/infra/gitlab` 适配层
  - 接口契约与错误映射

## M3 调度与执行器

- 定时轮询器（按 ProjectBinding 逐个项目拉取）。
- 生成 WorkItem 并防重（幂等）。
- 执行状态机：`queued -> running -> done/blocked/failed`
- 交付物：
  - scheduler + worker 骨架
  - WorkItem 生命周期管理

## M4 自动开发与 MR 流程

- 针对 WorkItem 执行：
  - 拉代码、切分支、调用 Agent 开发
  - 本地验证（lint/test）
  - push + 创建/更新 MR
- 交付物：
  - Issue->Branch->Commit->MR 全链路

## M5 冲突处理与重试

- 检测 merge/rebase 冲突。
- 自动冲突解决重试，最大 5 次。
- 超限后置 `blocked`，自动回写 issue。
- 交付物：
  - 冲突处理策略实现
  - 重试与失败回写机制

## 3. API 草案（V1）

- `POST /api/v1/projects/bindings`
- `GET /api/v1/projects/bindings`
- `PATCH /api/v1/projects/bindings/:project_key`
- `POST /api/v1/work-items/:id/run`（手动补偿执行）
- `GET /api/v1/work-items`

## 4. 验证清单

- `go test ./...`
- `pnpm build`
- 基于测试仓库的 E2E：
  - 创建测试 issue
  - 轮询识别并生成 work item
  - 产出 MR
  - 人工制造冲突并验证重试上限=5

## 5. 风险与控制

- 风险：GitLab API 限流  
  控制：轮询节流 + 指数退避

- 风险：重复触发导致重复 MR  
  控制：Issue->MR 映射唯一键 + 幂等校验

- 风险：Agent 修改质量不稳定  
  控制：固定验证命令门禁 + 失败回写 issue

## 6. 第一迭代建议拆分

1. 先做 M1 + M2（数据+GitLab 接入）
2. 再做 M3（调度）
3. 最后补 M4 + M5（自动开发和冲突处理）
