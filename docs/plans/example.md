# 初始化工程完善计划

## 背景

- 目标：完成后端与前端初始化后的收尾工作，并验证可运行性。
- 约束：当前仅使用 `scripts/run_codex_on_plan.py`，不使用 `scripts/agents/`。

## Task 列表

- [ ] T1: 后端依赖与编译校验
  - 验收标准：`go mod tidy` 和 `go test ./...` 能通过
  - 相关文件：`go.mod`、`internal/**`、`cmds/server.go`

- [ ] T2: 前端依赖与构建校验
  - 验收标准：`pnpm install`、`pnpm build` 能通过
  - 相关文件：`webui/package.json`、`webui/src/**`

- [ ] T3: 文档完整性检查
  - 验收标准：`docs/` 中架构与自动化说明一致且路径可用
  - 相关文件：`docs/**`

## 验证要求

- 执行并记录命令：
  - `go test ./...`
  - `cd webui && pnpm build`
- 若任一命令失败，需要定位根因并修复后重试。
