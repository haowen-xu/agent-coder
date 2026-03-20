# 工程架构

## 技术栈

- 后端：Go、`log/slog`、Viper、ErrorX、Hertz、GORM（SQLite / PostgreSQL）
- 前端：`pnpm`、Vite、Vue 3、Pinia、Element Plus

## 目录结构

```text
.
├── cmd/server                 # 服务启动入口
├── internal
│   ├── app                    # 组装层
│   ├── config                 # 配置加载与校验（viper）
│   ├── db                     # GORM 初始化、模型
│   ├── httpserver             # Hertz 路由与服务生命周期
│   ├── logger                 # slog 初始化
│   └── xerr                   # errorx 错误类型
├── scripts
│   └── run_codex_on_plan.py   # 计划驱动自动化执行器
├── webui                      # 前端应用
└── docs                       # 工程文档
```

## 运行入口

- 后端开发：`make run`
- 前端开发：`make webui-dev`
- 执行计划（演练模式）：`make codex-plan`
