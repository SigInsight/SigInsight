# ADR-005：Metrics/Meter 数据源与双阶段聚合契约

状态：Accepted
日期：2026-07-31
关联里程碑：M3、M4、M5

## 背景

现有 Metrics Explorer 将 metadata 查找、多个下采样表选择、时间聚合、空间聚合、
函数链和 response 后处理混在同一条 builder pipeline 中。其 SQL 可支持很宽的能力，
但无法作为轻量引擎的稳定基础。

Collector 的当前 schema 将普通点保存于 `siginsight_metrics.samples_v4`，并把标签与
resource/scope/point attributes 保存于 `time_series_v4`。Meter 则在
`siginsight_meter.samples` 中将标签和点一起保存。显式 Histogram 被展开为以 `.bucket`
结尾、带 `le` label 的累计 bucket series。

## 决策

M3 使用固定的原始数据源：

| Signal | 点表 | series/标签表 |
| --- | --- | --- |
| Metrics Gauge/Sum/explicit Histogram | `siginsight_metrics.samples_v4` | `siginsight_metrics.time_series_v4` |
| Meter | `siginsight_meter.samples` | 同一张点表的 `labels` |

Metrics 查询始终分两阶段：

1. 按 `fingerprint + time bucket + group labels` 聚合为每条 series 的值。
2. 按 `time bucket + group labels` 做受限制的空间聚合。

`rate` 和 `increase` 仅适用于 Sum/Meter：Delta series 以 bucket sum 为增量；
Cumulative series 用相邻 bucket 的差值并将 counter reset 解释为当前值。首个 bucket
没有前值时为 NULL，不以零伪造数据。Histogram percentile 使用已展开的 `.bucket`
series，并保留其物理 temporality：Delta 点在查询 bucket 内求和，Cumulative 点取最后
快照后按相邻查询 bucket 求差。随后内部保留 `le`、由累计 bucket 值计算非累计权重，
并以 ClickHouse 25.5.6
提供的 `quantileExactWeighted` 在最终阶段计算。该算法返回离散 bucket 上界，不提供旧
`histogramQuantile` 的插值语义。

所有 metric name、temporality、label key、attribute key、filter value 和时间范围均为
绑定参数。SQL 中只有受 Catalog 控制的表、列、aggregation enum 和常量 quantile。

## 明确不支持

- `samples_v4_agg_5m`、`samples_v4_agg_30m`、`samples_agg_1d` 和其表选择 heuristics。
- 指数直方图 `exp_hist`、summary、任意函数链、EWMA/anomaly、二次聚合和 raw SQL。
- 自动 metadata fallback、自动 materialized-column 选择和跨表 retention 补齐。
- Histogram 的任意 threshold/interpolation 参数；仅支持预定义 p50/p90/p95/p99。

固定原始表意味着 Metrics 查询只覆盖 Collector 当前 `samples_v4` 保留期，Meter 覆盖
其 `samples` 保留期。超出保留期时返回空结果而不是悄悄切换近似数据源。

## 影响

- SQL 的表选择不依赖 range heuristic，测试可固定并直接审计。
- Metric UI 只能展示 Catalog/metadata 已确认的 Gauge、Sum 或 `.bucket` Histogram。
- 若后续需要长期范围或指数直方图，必须新增版本化 source capability 和独立 ADR，不能
  在 compiler 内恢复旧 builder 的隐式分支。
