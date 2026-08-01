# M3：Metrics/Meter Compiler

状态：Complete
关联 ADR：001、003、004、005
前置条件：M1、M2 已完成

## 问题与目标

Metrics Explorer 的复杂度主要来自时序身份与聚合：数值点只保存 fingerprint，标签在
另一张表中；同一图表还需要时间聚合、空间聚合和 counter 语义。M3 在不复制旧 SQL
pipeline 的前提下，提供 Gauge、Sum、explicit Histogram 与 Meter 所需的最小查询路径。

## 范围

- `samples_v4 + time_series_v4` 的 Metrics source、label/resource/scope/attribute filters。
- `siginsight_meter.samples` 的 Meter source 和 label filters。
- 受约束的 time/space aggregation，Sum/Meter 的 rate/increase，explicit Histogram
  p50/p90/p95/p99。
- 固定原始表的参数化 SQL、golden tests 和 ClickHouse 25.5.6 真实执行测试。

## 非目标

- 旧聚合表和 table-selection heuristic。
- 指数 Histogram、summary、任意函数链、动态 metadata fallback、V5 decoder、executor。
- 任意 having、secondary aggregation、Trace Operator 或跨 signal join。

## 退出条件

- Gauge、Sum、explicit Histogram、Meter 的 capability matrix 目标条目均由 Compiler
  生成参数化 statement。
- SQL 显式维持 per-fingerprint time aggregation 后的 space aggregation。
- `rate`/`increase`、histogram percentile、label filtering 和缺失 label 语义均有单测。
- 临时 ClickHouse 25.5.6、Collector 当前 migrations 和插入的真实点数据可执行所有
  目标 statement。

## 实现结果

- `Catalog.MetricSource` 固定了 Metrics 的 `samples_v4/time_series_v4` 与 Meter 的
  `samples` 物理边界；Compiler 不再选择旧聚合表或 distributed 表。
- Metrics series CTE 将 label、attrs、scope attrs、resource attrs 绑定到 fingerprint，
  再以 `samples_v4` 做 per-series time aggregation 和 space aggregation。Meter 通过
  同表 JSON labels 走同一套 statement contract。
- Gauge/Sum 支持 latest/sum/avg/min/max/count；Sum/Meter 支持 delta/cumulative 的
  rate/increase；explicit Histogram 通过 `<name>.bucket` 和 `le` 计算 p50/p90/p95/p99。
- Histogram 使用 `quantileExactWeighted` 和非累计 bucket 权重，兼容 ClickHouse 25.5.6；
  不复用服务端不存在的 `histogramQuantile`。
- scalar counter 也按原始点时间排序后计算差值，避免把整个范围错误折叠为一个 latest 值。

## 验证结果

2026-07-31 已执行：

```bash
go test ./pkg/litequery
go test -race ./pkg/litequery
tests/integration/scripts/run-litequery-compiler-integration.sh
```

集成脚本启动 `clickhouse/clickhouse-server:25.5.6`，通过 Collector 的
`bootstrap/sync up/async up`，插入当前时间窗口内的 Gauge/Sum/Histogram/Meter 最小数据，
执行并扫描生成 statement，断言各查询至少返回一个正值。旧 Logs/Traces statement 也
在同一轮执行，所有测试通过，临时容器由脚本清理。

## 残余风险与后续任务

- 当前没有 V5 decoder、executor 或 result scanner；M4 才把 Statement 接入 API。
- Histogram 是离散 bucket 上界 percentile，没有旧实现的插值；指数 Histogram、summary
  和长期下采样表明确拒绝。
- 原始表保留期之外不自动回退到聚合表，长范围查询可能为空，属于明确的轻量化边界。
