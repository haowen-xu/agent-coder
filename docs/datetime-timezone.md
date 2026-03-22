# 时间与时区规范

> 更新日期：2026-03-22

本文档约束后端存储/传输、Go 内部时间处理，以及 WebUI 显示格式，避免时区混乱导致的排序、过期判断和展示偏差。

## 1. 存储与传输规则

- 数据库存储统一使用 UTC 时间语义。
- Go 服务对外 JSON 传输统一使用 ISO/RFC3339（`time.Time` 默认序列化格式）。
- 非 `time.Time` 的时间字符串字段，必须明确为 RFC3339 且为 UTC（例如带 `Z`）。

## 2. Go 内部处理规则

- 统一时间入口：`internal/utils.NowUTC()`。
- 生产代码禁止直接调用 `time.Now()`，避免混入本地时区时间。
- 需要“当前时间 + 持续时长”的场景，先取 `now := utils.NowUTC()` 再 `Add(...)`。
- GORM 时间戳统一通过 `NowFunc: utils.NowUTC` 生成。

## 3. WebUI 展示规则

- WebUI 仅在展示层做“本地时区格式化”。
- 统一使用 `webui/src/utils/format.ts` 的 `formatLocalDateTime(...)`。
- 禁止在业务组件中直接调用：
  - `toLocaleString(...)`
  - `toLocaleDateString(...)`
  - `toLocaleTimeString(...)`

## 4. 门禁脚本

门禁脚本统一放在 `scripts/agents/`：

- `check_go_coverage_gate.py`：Go 覆盖率门禁（总体 >=80%，逻辑文件不允许 0%）。
- `check_datetime_gate.py`：datetime 门禁（Go 禁止直接 `time.Now()`；WebUI 禁止组件内直接 `toLocale*`）。
- `check_all_gates.py`：统一入口，串行执行上述门禁。
- 所有门禁脚本必须由仓库根目录 `.venv` Python 执行。

本地执行：

```bash
.venv/bin/python scripts/agents/check_all_gates.py
```
