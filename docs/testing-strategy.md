# 测试分层策略

> 更新日期：2026-03-22
> 本文定义仓库统一测试分层与执行入口。

## 1. 目标

- 简单场景优先快速执行，纳入常规 CI 与 `make test`。
- 复杂场景通过独立环境编排，确保可重复、可追踪。
- 前端行为验证与后端状态验证统一纳入端到端流程。

## 2. 测试分层

### 2.1 简单 e2e / 集成测试（Go）

适用范围：

- 不需要多子系统协作。
- 不依赖外部系统（或可通过本地 mock 完成）。

约束：

- 以 Go 测试方式编写（`*_test.go`）。
- 通过 `go test ./...` 执行。
- 默认包含在 `make test` 中。

### 2.2 复杂多系统/外部系统协作测试（Python unittest）

适用范围：

- 需要同时拉起多个子系统。
- 需要与外部系统协作（例如 GitLab、外部 Agent 执行器、消息系统等）。

约束：

- 放在 `tests/e2e/`。
- 由 Python `unittest` 脚本负责：
  - 创建干净 `tempdir`。
  - 生成隔离配置、数据库与工作目录。
  - 启动并管理各子系统进程。
  - 断言 API 输出、数据库状态、文件系统状态。

### 2.3 前端行为 e2e（Python unittest + Playwright）

适用范围：

- 需要验证 WebUI 页面交互、路由行为、表单流程、可见反馈。

约束：

- 放在 `tests/playwright/`。
- 由 Python `unittest` 脚本负责环境编排：
  - 创建干净环境（tempdir/config/db/workdir）。
  - 启动后端相关子系统。
  - 调用 Playwright 执行前端行为测试。
  - 必要时补充验证 API、数据库、文件系统状态。

## 3. 运行约定

- Python 测试脚本统一使用仓库根目录 `.venv`。
- Go 测试仍是默认快速回归入口；复杂 e2e 与 Playwright e2e 作为分层补充。
- WebUI 调试/打包前需要先执行 `cd webui && pnpm build`。

推荐命令：

```bash
make test
.venv/bin/python -m unittest tests.e2e.test_runtime_e2e -v
.venv/bin/python -m unittest tests.playwright.test_playwright_e2e -v
```

## 4. 目录收敛

- 复杂 e2e 统一在 `tests/e2e/`。
- 前端行为 e2e 统一在 `tests/playwright/`。
- 不再新增 `webui/e2e/` 直连式 Playwright 编排脚本。
