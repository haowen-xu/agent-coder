# Agent 自动 Coding 需求（V3 定稿）

> 日期：2026-03-20  
> 基于 V2 讨论结论定稿。

## 1. 产品目标

构建一个基于仓库协作平台的自动研发执行系统，围绕 Issue 实现端到端闭环：

- 自动读取/更新 Issue
- 自动开发并验证
- 自动创建 MR
- 自动解决 merge 冲突

## 2. 范围定义

### In Scope（V1 实现范围）

- 平台优先级：先支持 **GitLab**
- 多项目管理：每个项目绑定 `repo + 仓库协作平台项目`
- 定时拉取 Issue 并创建自动开发任务
- 每个 Issue 固定创建一个 MR
- 冲突自动重试，最大 5 次

### Out of Scope（V1 不做）

- GitHub/Jira 适配
- 复杂调度（优先级队列、抢占、资源配额）
- 多 Agent 并行协作优化

## 3. 关键策略（已确认）

## 3.1 触发策略

- 使用定时任务 pull issue（非 webhook 触发）。
- 建议默认轮询周期：60 秒（可配置）。

## 3.2 分支命名

- 固定命名：`agent-coder/issue-<id>`

## 3.3 MR 策略

- 一个 Issue 固定一个 MR。
- 若已存在该 Issue 对应 MR，则复用并更新，不重复创建。

## 3.4 冲突处理策略

- 发生 merge/rebase 冲突时自动重试，`N=5`。
- 超过 5 次仍失败：
  - WorkItem 标记 `blocked`
  - 回写 Issue 评论说明失败原因和冲突摘要

## 3.5 平台策略

- 第一阶段只实现 GitLab 全链路闭环。

## 4. 领域模型（V1）

## 4.1 ProjectBinding

- `project_key`
- `repo_url`
- `repo_default_branch`
- `gitlab_project_id`
- `credential_ref`
- `poll_interval_sec`
- `enabled`

## 4.2 WorkItem

- `work_item_id`
- `project_key`
- `issue_iid`
- `status`：`queued` | `running` | `blocked` | `done` | `failed`
- `branch_name`
- `mr_iid`
- `retry_count`
- `last_error`

## 5. 流程定义（V1）

1. 定时轮询 GitLab Issue（按项目绑定）。
2. 根据规则筛选可执行 Issue（状态、标签、是否已在处理）。
3. 创建/复用 `WorkItem`，切分支 `agent-coder/issue-<id>`。
4. Agent 执行开发、测试、提交。
5. push 分支并创建/更新 MR（每个 Issue 一个 MR）。
6. 若冲突则自动重试，最多 5 次。
7. 回写 Issue（进度、MR 链接、失败原因、最终状态）。

## 6. 验收标准（V1）

- 可管理多个 ProjectBinding。
- 单项目内可从 Issue 自动推进到 MR。
- MR 与 Issue 一一对应且可重入执行。
- 冲突重试次数遵循 `N=5` 且有明确失败回写。
- 全流程有日志与状态追踪（WorkItem 维度）。

## 7. 下一步文档

- 实施计划：`docs/plans/2026-03-20-agent-autocoding-implementation-plan-v1.md`
