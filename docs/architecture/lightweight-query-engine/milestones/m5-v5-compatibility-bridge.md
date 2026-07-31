# M5：V5 兼容桥与受控 API 接入

状态：Complete
关联 ADR：002、004、006、007
前置条件：M1、M2、M3、M4 已完成

## 范围

- 将 V5 builder/formula DTO 转换成 `pkg/litequery.Request`，并将执行结果恢复为 V5 response。
- 使用 V5 已有的 ANTLR filter grammar 生成受限 Filter AST，不复用旧 SQL visitor。
- 将 ClickHouse driver 的动态 ScanType 适配限制在 `pkg/querier`。
- 通过 `querier.lightweight_engine_enabled` 控制 `/api/v5/query_range` 的 opt-in 路径。
- 对不属于 Lite capability matrix 的 V5 请求保持可诊断的回退，而不丢弃字段或改变语义。

## 当前支持

- `raw`、`trace`、`time_series`、`scalar` V5 result 类型。
- Logs/Traces 的单个受支持 aggregation，Metrics/Meter 的单个受支持 aggregation。
- `AND`、`OR`、括号、比较、`IN`/`NOT IN`、`EXISTS`/`NOT EXISTS`、`CONTAINS` 的结构化过滤子集。
- 简单算术 formula；metric type/temporality 由现有 metadata store 读取后作为 adapter 输入。

## 当前明确排除

- raw ClickHouse SQL、Trace Operator、join、sub-query。
- 多 aggregation、secondary aggregation、`limitBy`、offset、cursor、任意 Having。
- `LIKE`/`ILIKE`/`REGEXP`/`BETWEEN`、full-text、函数调用、变量 filter 值和 unary `NOT`。
- 所有 V5 后处理函数及 `formatOptions.fillGaps`。这些能力尚无 Lite 等价实现，不能被忽略。

## 实现结果

- 新增 `pkg/querier/liteadapter`，持有 V5 -> Lite、filter AST 与 Lite -> V5 response 转换。
- `pkg/litequery` 继续只依赖领域类型和 `Rows`/`QueryFunc` 接口；具体 ClickHouse `driver.Rows`
  适配器在 `pkg/querier/litequery.go`。
- `querier.QueryRange` 在开关启用时先尝试转换；适配器返回 `UnsupportedError` 时保留 legacy
  engine。转换/校验/执行失败则作为该次 Lite 请求的错误返回，不再回退以避免掩盖缺陷。
- type/temporality 未携带在 V5 JSON metric aggregation 中，因此 adapter 输入包含由 metadata store
  解析出的只读 map，避免复制旧 metrics statement builder 的表选择逻辑。

## 验证结果

2026-07-31 已执行：

```bash
go test ./pkg/litequery ./pkg/querier/...
go test ./pkg/types/querybuildertypes/querybuildertypesv5
go test -race ./pkg/litequery ./pkg/querier/liteadapter
tests/integration/scripts/run-litequery-v5-bridge-integration.sh
```

`liteadapter` 单测覆盖结构化 filter、metric metadata 注入、不支持 V5 feature 和 V5 result
整形。最后一条脚本启动当前分支 SigInsight、SQLite 和 ClickHouse 25.5.6，以
`SIGNOZ_QUERIER_LIGHTWEIGHT__ENGINE__ENABLED=true` 发起认证的 `/api/v5/query_range` 请求。
容器日志确认每个请求进入 Lite bridge；Metrics、Meter 和 `meter * 2` formula 返回非空 V5 series。

Logs/Traces 请求也已通过认证 API、V5 转换和实际 SQL 执行，但 integration fixture 的 synthetic
writer 写入后不能被 SigInsight 的 native ClickHouse client 读回。这不是成功数据断言的替代；使用
当前 Collector 进行真实 OTLP 写入并验证 Logs/Traces 值，是 M7 的强制切换门槛。

## 后续退出项

- 为受支持 V5 fixtures 执行 legacy/Lite 双跑，比较响应值、labels、时间桶与 raw 行。
- 在 ClickHouse 25.5.6 + 当前 Collector 的真实 OTLP 写入环境中检查 logs/traces query log 和返回值。
- 实现或继续明确拒绝 fill gap、cursor、variables 和 raw/trace UI 必需的字段语义。
