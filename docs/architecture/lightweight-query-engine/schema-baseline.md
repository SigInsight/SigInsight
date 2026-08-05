# ClickHouse Schema Baseline

状态：Accepted（M16 canonical schema）
最后验证：2026-08-06
ClickHouse：`clickhouse/clickhouse-server:25.5.6`

## 目的

本文件冻结 Lightweight Query Engine 开始实现时所依赖的存储边界。它不是新的 schema 定义；唯一的 schema 所有者仍是 OtelCollector 的 migration engine。

查询代码只能依赖后续 Schema Catalog 声明的语义字段。任何物理表、列、projection 或 materialized view 的变更必须同时更新对应 ADR、Catalog 测试和真实协作验证。

## M16 Canonical Fingerprint

以下值由 Collector 的 `v1SchemaFingerprints` 定义，并由 `make test-migration-integration`
在 ClickHouse 25.5.6 验证。变量名保留用于迁移框架兼容；其内容是 M16
direct canonical baseline 的最终 `system.tables` 和 `system.columns` 指纹。

| Database | Tables | Table hash | Columns | Column hash |
| --- | ---: | ---: | ---: | ---: |
| `siginsight_analytics` | 1 | 12242196818159068904 | 11 | 9151060719887308879 |
| `siginsight_logs` | 6 | 4907456606029717637 | 43 | 6450537337900337272 |
| `siginsight_metadata` | 0 | 11160318154034397263 | 0 | 11160318154034397263 |
| `siginsight_meter` | 3 | 4794694533261956589 | 38 | 16828873058290010379 |
| `siginsight_metrics` | 15 | 13305681492122153582 | 174 | 8325196865590917439 |
| `siginsight_traces` | 15 | 9151180053864431952 | 131 | 14213257424511927958 |

来源：`OtelCollector/cmd/siginsightschemamigrator/schema_migrator/v1_baseline_migrations.go`。

## 当前读取范围

核心查询读取范围在 Collector 的真实 migration test 中以 `LIMIT 0` 验证：

| Signal | 核心表 | 核心物理字段 |
| --- | --- | --- |
| Logs | `siginsight_logs.logs` | `timestamp`、`body`、`resource`、attributes/resource maps |
| Traces | `siginsight_traces.spans` | `timestamp`、`trace_id`、`span_id`、`resource`、span/resource maps |
| Metrics | `siginsight_metrics.metric_points`、`metric_series` | `metric_name`、`fingerprint`、`unix_milli`、`value`、labels/attrs/resource attrs |
| Meter | `siginsight_meter.meter_points`、`meter_rollup_1d` | `metric_name`、`fingerprint`、`unix_milli`、`value`、labels |

本基线明确不支持任何版本后缀业务表、`body_v2`、`body_promoted`、JSON path metadata
tables、旧 distributed tables 或 Trace camelCase alias。日志 body JSON 查询使用 `body String`
上的 ClickHouse JSON 函数。

## 可重复验证

从 SigInsight 根目录运行：

```bash
tests/integration/scripts/run-canonical-schema-cutover.sh
```

该脚本先运行：

```bash
make -C ../OtelCollector test-migration-integration
```

Collector 测试从空 ClickHouse 直接创建 schema、写入 OTLP logs/traces/metrics，并检查：

- M16 canonical fingerprint。
- 当前核心表和列可读。
- 旧 JSON 列、metadata 表和旧 distributed/过渡表不存在。
- Collector 写入的数据能够由当前 schema 保存。
- `body String` 的 JSON path 查询正常工作。

随后 SigInsight 集成测试以 SQLite、认证 API 和同一 ClickHouse 版本覆盖 capability matrix 中的 Logs、Traces、Metrics、Meter 和基本 Alert 查询。

## 变更规则

1. 不修改 v1 baseline DDL 或本文件的 v1 fingerprint 来“适配”新功能。
2. 新 schema 通过 post-baseline migration 引入，使用新的 migration ID；M16 本身没有 create/drop cleanup 链。
3. 影响核心读取范围的改动必须在 M2/M3 的 Schema Catalog 中显式声明。
4. 删除任何列或表前，必须执行 production import 扫描、Collector 写入测试和 SigInsight 真实 API 测试。
