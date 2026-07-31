# M4：Executor、Result 与基本 Formula

状态：Complete
关联 ADR：004、006
前置条件：M1、M2、M3 已完成

## 范围

- 带 context deadline/cancellation 的 Statement executor。
- 动态 ClickHouse row 扫描、列契约、执行统计和资源关闭。
- 受限制 arithmetic formula 的时间桶/分组对齐。
- executor 的错误、取消、timeout、扫描失败和并发测试。

## 非目标

- V5 JSON decoder/encoder、HTTP handler、旧 Querier 替换。
- Trace Operator、join、secondary aggregation、复杂函数和 anomaly。
- Alert rule wiring；只提供可被 M5/M6 adapter 使用的结果边界。

## 退出条件

- ClickHouse client 只通过 Queryer 函数注入，core 包无具体连接依赖。
- 每条 rows 在成功、扫描失败和 context 取消路径都关闭。
- 基础 query 按声明顺序返回，formula 不改变基础 query 的错误语义。
- timeout/cancel、公式对齐和除零行为都有直接测试。

## 实现结果

- 新增 `Executor`、`QueryFunc` 和 `Rows` 边界。Executor 限制并发、统一 deadline、按
  statement 顺序保留结果，并在扫描失败时关闭 rows。
- `QueryResult` 保留 Statement 的列契约和动态行数据，未把 ClickHouse driver 或 V5
  response 类型引入核心包。
- 公式在基础查询完成后按 timestamp/group key 对齐，支持 forward formula dependency、
  `+ - * /` 和括号；除数为零产生 `NaN`，依赖缺少时生成 warning。
- ClickHouse integration test 增加了 driver rows adapter，验证真实动态列扫描和双 query
  公式结果；driver-specific ScanType 转换留在基础设施 adapter，不污染 executor contract。

## 验证结果

2026-07-31 已执行：

```bash
go test ./pkg/litequery
go test -race ./pkg/litequery
tests/integration/scripts/run-litequery-compiler-integration.sh
```

单测和 race 检查通过；同一临时 Collector schema/ClickHouse 25.5.6 环境中，Executor
实际扫描 Meter time-series rows 并完成 formula，Logs/Traces/Metrics/Meter compiler
查询也同时通过。

## 残余风险与后续任务

- V5 decoder/response adapter 尚未接入 `/api/v5/query_range`，因此生产请求仍走旧 Querier。
- ClickHouse driver 的具体 `ScanType` adapter 目前在 integration fixture 中，M5/M6 接入
  时需放入 query-service infrastructure package。
- Executor 尚未输出 ClickHouse progress rows/bytes，M4 只保证结果和生命周期；统计接入
  属于 M6 query-log/协作验证。
