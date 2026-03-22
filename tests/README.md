# E2E Tests

本目录承载复杂集成与前端行为测试的统一入口。

## 目录

- `tests/e2e/`：Python `unittest`，复杂多系统/外部系统协作测试。
- `tests/playwright/`：Python `unittest`，环境编排并调用 Playwright。

## 运行方式

```bash
# 复杂多系统/外部系统协作测试
.venv/bin/python -m unittest tests.e2e.test_runtime_e2e -v

# 前端行为 e2e（Python 编排 + Playwright）
.venv/bin/python -m unittest tests.playwright.test_playwright_e2e -v
```

说明：测试会在临时目录下生成隔离 `config/db/workdir`，并拉起子系统后执行断言。
