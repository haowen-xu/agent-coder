# 代码规范

## 目标

- 保持后端 Go 与前端 Vue/TS 风格一致，降低维护成本。
- 每次改动都能快速通过格式化、编译和基础验证。

## Go（后端）

### 目录与命名

- CLI 入口统一放在 `cmds/main.go`，子命令放在 `cmds/*.go`。
- 业务实现放在 `internal/*`，避免对外暴露内部实现细节。
- package 名使用小写短词，不使用下划线。

### 错误处理

- 统一使用 `error` 返回，不使用 panic 作为常规流程。
- 需要分类时使用 `internal/xerr`（`errorx`）包装上下文。
- 错误信息写清动作和对象，例如：`"load config"`、`"open gorm db"`。

### 日志

- 统一使用 `log/slog`。
- 结构化字段优先：`slog.String(...)`、`slog.Any(...)`。
- 禁止在业务代码中使用 `fmt.Println` 作为正式日志。

### 格式与校验

- 提交前至少执行：

```bash
gofmt -w $(rg --files -g '*.go')
go test ./...
./scripts/check_go_coverage_gate.sh
```

覆盖率门禁要求：

- Go 代码总体覆盖率必须 `>= 80%`
- 含有具体代码逻辑（存在可统计语句）的 Go 文件不允许 `0%` 覆盖率
- Go 单元测试文件必须按同名规则放置：`xxx.go` 的单元测试必须写在 `xxx_test.go`（例如 `client.go` -> `client_test.go`）

## Vue / TypeScript（前端）

### 组件与状态

- 页面级组件统一放 `webui/src/views/*`，`App.vue` 仅做布局或路由容器。
- 跨组件状态统一放 `webui/src/stores/*`（Pinia）。
- store 负责数据请求和状态变更，组件负责展示与交互。

### TypeScript

- 新增数据结构优先定义接口/类型，避免 `any` 扩散。
- 异步请求统一处理错误路径，保证 UI 可感知失败状态。

### 样式

- 样式优先写在组件内 `scoped` 块，减少全局污染。
- 颜色和布局保持简洁一致，不随意混入多套视觉风格。

### 构建校验

- 提交前至少执行：

```bash
cd webui && pnpm build
```

## 通用约定

- 一个 PR/提交聚焦一个目标，避免混入无关改动。
- 文档、代码、配置保持同步更新。
- 自动化流程当前仅使用 `scripts/run_codex_on_plan.py`。
- 仓库内不再维护内置 agents 脚本，统一由外部 Agent-Coder 管理。
