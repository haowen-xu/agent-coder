# Agent 自动 Coding 需求（V2）

## 1. 产品定位

本项目定位为一个基于 Issue Tracker 的自动研发执行系统，核心目标是让 Agent 围绕 Issue 完成端到端交付闭环。

## 2. 核心能力（你已确认）

### P0 能力

- 自动读取 Issue（列表、详情、标签、状态、评论）
- 自动修改 Issue（评论进展、打标签、改状态、关联链接）
- 自动开发（基于 Issue 内容进行代码修改与本地验证）
- 自动提 Merge Request / Pull Request
- 自动解决 merge 冲突（rebase/merge 冲突自动修复并继续）

### 平台能力

- 支持多项目（multi-project）
- 每个项目绑定：
  - 一个代码仓库（Repository）
  - 一个 Issue Tracker 项目（Project）
- 代码仓库与 Issue Tracker 可以来自同一平台（如 GitLab/GitHub），也允许跨平台组合

## 3. 领域模型（建议）

## 3.1 ProjectBinding

每个受管项目一条绑定记录：

- `project_key`：内部项目唯一标识
- `repo.provider`：`github` | `gitlab` | `gitea` | `other`
- `repo.url`：仓库地址
- `repo.default_branch`：默认分支（通常 `main`）
- `issue.provider`：`github` | `gitlab` | `jira` | `other`
- `issue.project_id`：Issue Tracker 的项目标识
- `credential_ref`：凭据引用（不直接存明文 token）

## 3.2 WorkItem

- `work_item_id`：内部工作项 ID
- `project_key`：归属项目
- `issue_ref`：外部 issue 编号/URL
- `status`：`queued` | `running` | `blocked` | `done` | `failed`
- `branch_name`：自动开发分支
- `mr_url`：生成的 MR/PR 链接

## 4. 端到端流程（目标形态）

1. 轮询/订阅 Issue 事件，筛选可执行 Issue。
2. 创建工作分支，Agent 读取 Issue 上下文并实施改动。
3. 执行 lint/test/build，失败则循环修复。
4. push 分支并创建 MR/PR。
5. 若目标分支变化导致冲突，Agent 自动拉取最新基线并解决冲突。
6. 回写 Issue：进度、MR 链接、失败原因、最终状态。

## 5. In Scope / Out of Scope

### In Scope（当前阶段）

- GitHub/GitLab 至少一种平台打通全链路
- 项目绑定管理（repo + issue project）
- 从单个 Issue 自动开发到 MR 的闭环
- 基础冲突解决与重试机制

### Out of Scope（暂缓）

- 复杂多 Agent 协同编排
- 组织级权限审计平台
- 高级排队调度（优先级、资源配额、SLA）

## 6. 非功能要求（建议基线）

- 幂等性：同一 Issue 重复触发不会创建重复 MR
- 可观测性：每个 WorkItem 可追踪日志与关键事件
- 可恢复性：中断后可从上次进度恢复（含 thread/session）
- 安全性：Token 最小权限、脱敏日志、按项目隔离凭据

## 7. 待你确认的关键策略

## Q1 触发策略

- 是手动触发（按钮/命令）优先，还是 webhook 自动触发优先？

## Q2 分支与命名

- 分支规范是否固定为：`autofix/issue-<id>`？

## Q3 MR 策略

- 一个 Issue 固定一个 MR，还是允许分阶段多 MR？

## Q4 冲突处理边界

- 自动冲突解决失败后，是转人工接管，还是继续重试 N 次？

## Q5 平台优先级

- 先做 GitLab 还是 GitHub（建议先一个平台闭环）？

## 8. 下一步输出

在你确认 Q1~Q5 后，我会继续产出：

- `docs/requests/2026-03-20-agent-autocoding-requirements-v3.md`（定稿）
- `docs/plans/` 的开发实施计划（按里程碑拆分）
- 后端 API 草案与数据表草案
