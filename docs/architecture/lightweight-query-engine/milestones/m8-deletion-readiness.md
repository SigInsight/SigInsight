# M8: Legacy 删除准备与量化收敛

Status: In Progress

关联决策：[ADR-012：在 V5 边界退役 Trace Operator](../decisions/012-retire-trace-operator-at-v5-boundary.md)

## 目标

在不扩大 Lite 功能边界的前提下，清除 legacy V5 query engine 的生产依赖，并量化
删除带来的规模收益。M8 不把专用读取工作流伪装成 Lite 查询。

## 2026-07-31 基线

| 区域 | 生产 Go/TS 行数 | 当前状态 |
| --- | ---: | --- |
| `pkg/querier`（含 `liteadapter`） | 5,840 | Lite bridge 与 legacy builder/cache/postprocess 共存 |
| `pkg/litequery` | 2,686 | 独立 IR、compiler、executor |
| `pkg/querier/liteadapter` | 864 | V5 边界适配层 |
| `frontend/src/components/QueryBuilder` | 9,213 | legacy editor、Trace Operator 与 Lite 选择入口共存 |
| `frontend/src/container/QueryBuilder` | 8,420 | legacy 状态、字段与高级控件仍活跃 |
| `frontend/src/features/lite-query` | 1,428 | 受约束的新编辑器与 capability model |

行数是 `wc -l` 静态基线，用于比较趋势，不能单独作为可删除证据。

## 已确认依赖图

```text
HTTP /api/v5/query_range
  -> Lite adapter (supported request)
  -> legacy QueryRange fallback (unsupported request)
       -> builder / raw SQL / cache / postprocess

Span Percentile -----+-> legacy QueryRange
Raw Data Export -----/
Threshold rules -----+-> Lite RuleQueryRunner
API Monitoring ------/ (removed as non-core)
```

Services 已迁移为专用的 parameterized reader，直接读取 `signoz_index_v3` 和
`top_level_operations`，不再构造 V5 request/response 或依赖 `Querier`。其查询保留
resource/span attribute 的 `in`/`notin`、服务概览、Top Operations、Entry Point
Operations 所需的固定多聚合。ClickHouse 25.5.6 不支持绑定 `LIMIT` 参数，因此仅将
经过 `1..5000` 校验的整数内联；所有 tag 键和值继续使用命名参数绑定。

Span Percentile 同样已迁移为单行的 parameterized reader：固定 p50/p90/p99、阈值
位置和 resource attribute 等值筛选直接执行在 `signoz_index_v3`，并在零 span 时保留
`NotFound` 响应。它们都不是 Lite 的遗漏能力，而是 ADR-010 所定义的专用读取边界。

## 删除顺序

1. Services 与 Span Percentile 已迁移为专用 readers，并删除其 `Querier` 依赖及 V5
   request/response 转换代码。
2. Raw Export 已作为非核心高级能力下线：删除下载入口和 `/api/v5/export_raw_data`，
   不再保留 offset/Trace Operator 导出链。Live Logs 已迁为独立 SSE reader：固定六个
   原始日志列和 500 行批次，复用 Lite filter AST，并以 `(timestamp, id)` 严格游标轮询。
   失效的 V5 `raw_stream` request type、response stream model 和 generic builder 分支已删除。
3. threshold rules 已通过独立 `RuleQueryRunner` 迁至 Lite：保存的 V5 定义在执行时完成
   元数据补全、Lite 编译和 V5 结果适配；raw SQL、Trace Operator 等超出 capability 的
   规则返回稳定的不支持错误，不能回退 legacy。现有测试工厂保留 test-only legacy adapter，
   以逐步替换历史 fixture。
4. `/api/v5/api-monitoring/overview/*` 是无前端调用的第三方 domain overview API，依赖
   多聚合 legacy translator。它不在核心 UI 范围内，路由、translator、DTO 和测试已删除；
   因而应用中不再有 `/api/v5/query_range` 之外的生产 `Querier.QueryRange` 调用者。
5. `querier.lightweight_engine_enabled` 已默认开启：core capability 的 Logs、Traces、Metrics
   与 Meter V5 请求默认走 Lite；高级保存查询暂保留受控 fallback，开关仍可用于回归处置。
6. Trace Operator 已完成退役：Alert 和 Trace Explorer 不再提供入口，共享 QueryBuilder
   控件、ANTLR parser、V5 执行器和 trace CTE builder 已删除。为兼容已保存 JSON，V5 DTO
   仍能解码 `builder_trace_operator`，但 validation 和 Querier 入口都会在任何 SQL 或
   ClickHouse 调用之前返回稳定的“不再支持”错误。
7. 已用同一 Collector 写入的 ClickHouse 25.5.6 fixture 对 Lite 与 legacy 双跑 Logs、
   Traces、Metrics 和 Meter 查询；比较 V5 结果的 labels、时间桶和 values，并检查
   `system.query_log` 没有 SQL exception。
8. 在 supported UI/default configuration 中移除 legacy fallback；此时不支持的 V5
   请求必须返回稳定的 capability error。
9. 通过 production import、路由、动态 import 和生成代码引用检查后，按依赖顺序删除
   `builder_query`、`clickhouse_query`、`bucket_cache`、
   `postprocess` 及关联前端高级控件、parser、mocks、测试和样式。

## 双运行协作证据（2026-07-31）

`tests/integration/scripts/run-litequery-collector-collaboration.sh` 会启动当前
Collector、两个连接同一 ClickHouse/SQLite 的 SigInsight 实例（Lite 启用和显式禁用），再
通过认证的 `/api/v5/query_range` 请求读取 Collector 经 OTLP 写入的数据。请求关闭 cache，
因此每次比较都触发真实 ClickHouse 查询。

在本机 `clickhouse/clickhouse-server:25.5.6` 上，该脚本通过（`1 passed in 104.02s`）：

| 信号 | Lite read rows / bytes | Legacy read rows / bytes | V5 数据契约 |
| --- | ---: | ---: | --- |
| Logs | 1 / 8 | 2 / 148 | labels、时间桶和值相等 |
| Traces | 1 / 8 | 2 / 148 | labels、时间桶和值相等 |
| Metrics | 2 / 215 | 2 / 172 | labels、时间桶和值相等 |
| Meter | 4 / 258 | 2 / 129 | labels、时间桶和值相等 |

这些 read rows/read bytes 是仅含少量 OTLP 样本的 fixture 诊断数据，用来发现 SQL 路径或
schema 回归，不能外推为生产性能结论。该测试不覆盖 legacy 的 opaque cursor、raw export
offset 或已退役的 Trace Operator；这些能力不属于 Lite 的支持范围。

## 当前阻塞项

- Lite 只实现了 Live Logs 所需的 typed `(timestamp, id)` cursor；V5 opaque cursor 和
  Raw Export 所需的 offset pagination 仍不在 Lite 范围内。
- V5 的 retired Trace Operator DTO 仍保留只读解码和明确拒绝；这是保护保存查询不被
  静默降级的兼容边界，不是执行依赖。

## 验收证据

- 每迁出一个调用者，直接测试其领域 reader 且 `rg` 证明调用者不再 import `querier`。
- 每删除一组 legacy 文件，production import 与生成路由引用为零。
- 全量 Go、前端 build/test、关键浏览器回归与当前 Collector 协作脚本通过。
- 最终报告比较 M8 基线与完成后的行数、主要模块数、最大文件、测试覆盖以及同 fixture
  的 query-log 延迟、read rows、read bytes。
