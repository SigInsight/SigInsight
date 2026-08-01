# M2：Logs/Traces Schema Catalog 与 Compiler

状态：Complete
关联 ADR：002、003、004
前置条件：M0、M1 已完成

## 问题与目标

现有 Logs/Traces 查询将字段映射、过滤语义、SQL 片段、metadata fallback 和请求类型处理分散在多个 package。M2 建立可直接测试的 Schema Catalog 与参数化 Compiler，使 Lite Logical Plan 可以独立编译成 ClickHouse statement。

## 范围

- `pkg/litequery` 中的 Schema Catalog、Statement 和 Logs/Traces compiler。
- `logs_v2` 与 `span_index_v3` 的核心字段映射。
- resource/attribute Map、普通 log/span 字段、`body String` JSON path 映射。
- raw、trace、time series、scalar statement。
- typed filter AST、group/order/limit、聚合后 predicate。
- SQL golden test、参数顺序测试和注入边界测试。

## 非目标

- ClickHouse client、行扫描、V5 response mapping 或 HTTP handler。
- V5 compatibility decoder。
- 动态 metadata discovery 和 materialized-column 自动选择。
- Trace Operator、跨 span 关系、secondary aggregation、limitBy。
- Metrics/Meter compiler。
- 改动 Collector schema 或写入。

## 设计

Catalog 是唯一的语义字段映射点：

```go
type Catalog interface {
    Resolve(Signal, FieldRef) (ResolvedField, error)
}

type ResolvedField struct {
    SQL  string
    Args []any
}
```

`SQL` 只能来自静态 Catalog 模板。任何 attribute/resource Map key、JSON path 或过滤值均进入 `Args`；compiler 不接受 SQL expression 字符串。

对 Map 与 JSON path 的正向比较（`=`, `>`, `IN`, `CONTAINS`）自动增加存在性判断。这样缺失的数值 key 不会因为 ClickHouse Map 的默认 `0` 被错误匹配；负向比较保持“可包含缺失字段”的语义。

Statement 使用驱动可绑定的问号占位符：

```go
type Statement struct {
    SQL      string
    Args     []any
    Warnings []string
}
```

Logs 时间戳为纳秒整数；Traces 使用 `DateTime64(9)`。Compiler 统一输出 epoch-millisecond bucket，消除前端对两种物理时间表达的差异。

`ResultTrace` 采用受约束的 trace summary plan：先用任意匹配 span 选择 trace ID，再在
时间范围内统计完整 span count，并返回 root span 的 timestamp、duration、service 与
operation。存在多个 root 时选择最长 root；root 缺失时回退到最长 span，避免部分 trace
被静默丢弃。默认使用 timestamp 与 trace ID 稳定排序。它不表达父子或祖先关系查询。

## 测试计划

- 逐 signal 的 intrinsic/resource/attribute/body field mapping。
- JSON path 和 Map key 作为参数、含引号/反斜杠的恶意输入。
- raw、trace、time series、scalar 的 golden SQL 和 Args。
- Filter logical AST、IN、EXISTS、CONTAINS、aggregation predicate。
- Logs 纳秒与 Traces DateTime64 时间范围、bucket 和 end-exclusive 语义。
- 使用 ClickHouse 25.5.6 执行 compiler 生成的最小真实 query。

可重复命令：

```bash
tests/integration/scripts/run-litequery-compiler-integration.sh
```

## 退出条件

- M2 支持 capability matrix 中的 Logs/Traces 目标查询子集。
- 所有值和动态路径参数化；无法通过 field、cursor 或 sort 注入 SQL。
- Compiler 不依赖旧 `telemetrylogs`、`telemetrytraces` 或 V5 builder package。
- M2 文档记录真实 ClickHouse 验证和遗留限制。

## 实现结果

- 新增 `DefaultCatalog`，将 Logs/Traces 的语义 `FieldRef` 映射到受信任的
  `logs_v2` 和 `span_index_v3` 物理表达式。
- 新增独立 Compiler，支持 Logs raw/time-series/scalar，Traces raw/summary/
  time-series/scalar，以及受约束的聚合、过滤、分组、排序、limit 和聚合 predicate。
- 新增 `Statement` 和 `ResultColumn`。SQL 中仅使用生成的 `field_N` / `group_N`
  alias，调用方通过列元数据恢复语义字段，动态输入不再能够进入 SQL 标识符。
- V5 opaque cursor 显式返回 `unsupported`；只有独立 Live Logs reader 使用 typed
  `(timestamp, id)` cursor。raw/trace 的 offset 保留为有限兼容能力。

## 验证结果

2026-07-31 已执行：

```bash
go test ./pkg/litequery
tests/integration/scripts/run-litequery-compiler-integration.sh
```

第二条命令会启动临时 `clickhouse/clickhouse-server:25.5.6`，通过当前
OtelCollector 执行 `bootstrap`、`sync up`、`async up` 后，在真实 schema 上执行
生成的 Logs raw 和 Traces time-series statement。两类查询均执行成功，临时容器和
构建目录由脚本 trap 清理。

验证中发现 ClickHouse 25.5.6 在 `GROUP BY` 中会把未限定的 `timestamp` 解析成
SELECT alias（epoch `Int64`），而不是 trace 表的 `DateTime64(9)` 物理列。Compiler
现固定使用 `span_index_v3.timestamp` 生成 trace bucket，该行为已进入 SQL golden
test 和真实执行测试。

## 残余风险与后续任务

- Catalog 仍只覆盖 capability matrix 指定的核心字段；动态 metadata discovery 和
  materialized-column 选择不属于轻量引擎。
- M2 尚未执行、扫描行或转换 V5 response；这些职责留给 M4。
- Metrics/Meter 的表选择和聚合语义属于 M3，必须先形成独立 ADR。
