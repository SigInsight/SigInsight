# M16: Canonical ClickHouse Schema Cutover

状态：Complete

负责人：SigInsight query/storage 实施者

起止提交：SigInsight TBD；OtelCollector `777a8a9`

关联 ADR：[ADR-003](../decisions/003-schema-catalog-contract.md)、[ADR-010](../decisions/010-specialized-readers-before-legacy-removal.md)

## 问题与目标

Collector 与 SigInsight 必须共享唯一的无版本后缀 ClickHouse 契约。旧版以 frozen v1
baseline 建表、随后 sync 创建新表、async 删除旧表的实现，意味着全新安装也会短暂拥有
`span_index_v3`、`logs_v2`、`samples_v4` 等旧对象。这既扩大了 schema 契约，也使启动、
测试和 query log 无法证明旧读路径已经消失。

本阶段将 SigInsight 所有运行时 reader、Lite Schema Catalog、专用查询、retention 与
配置切换至 canonical schema，与 Collector `777a8a9` 构成一个停机、丢旧数据的发布单元。

## 范围

### 对象名映射

| 数据库 | 旧对象 | Canonical 对象 |
| --- | --- | --- |
| Logs | `logs_v2` | `logs` |
| Logs | `logs_v2_resource` | `resource_sets` |
| Logs | `tag_attributes_v2` | `field_values` |
| Traces | `span_index_v3` | `spans` |
| Traces | `traces_v3_resource` | `resource_sets` |
| Traces | `tag_attributes_v2` | `field_values` |
| Traces | `error_index_v2` | `exceptions` |
| Traces | `top_level_operations` | `operations` |
| Traces | `root_operations` / `sub_root_operations` | `root_operations_mv` / `sub_root_operations_mv` |
| Traces | `trace_summary_mv` | `trace_summary_from_spans_mv` |
| Service Map | `dependency_graph_minutes_v2` 及 3 个 MV | `service_edges` 及 `service_edges_*_mv` |
| Metrics | `samples_v4` | `metric_points` |
| Metrics | `time_series_v4` | `metric_series` |
| Metrics | `samples_v4_agg_5m/30m` | `metric_rollup_5m/30m` |
| Metrics | `time_series_v4_6hrs/1day/1week` | `metric_series_6h/1d/1w` |
| Analytics | `rule_state_history_v0` | `rule_state_history` |
| Meter | `samples` / `samples_agg_1d` | `meter_points` / `meter_rollup_1d` |

Logs keys tables、Trace `span_attributes_keys`、Metrics `metadata`、Trace `trace_summary`和各 signal
`usage` 表的表名不变。

### Trace 主表列名

`siginsight_traces.spans` 不再存在 `traceID`、`serviceName`、`httpRoute`等 camelCase
`ALIAS`。所有 SQL 必须使用原生 snake_case 列，例如 `trace_id`、`span_id`、
`parent_span_id`、`duration_nano`、`status_code`、`has_error`、`links`。

下列高频字段从历史物化列改为可读名称：

| 旧列 | Canonical 列 |
| --- | --- |
| `resource_string_service$$name` | `service_name` |
| `attribute_string_http$$route` | `http_route` |
| `attribute_string_messaging$$system` | `messaging_system` |
| `attribute_string_messaging$$operation` | `messaging_operation` |
| `attribute_string_db$$system` | `db_system` |
| `attribute_string_rpc$$system` | `rpc_system` |
| `attribute_string_rpc$$service` | `rpc_service` |
| `attribute_string_rpc$$method` | `rpc_method` |
| `attribute_string_peer$$service` | `peer_service` |
| 上述每列的 `_exists` 列 | 对应的 `*_present` |

`*_present` 不能由 `field != ''` 替代：缺失字段与空字符串在 `NOT`、`not_in`
等过滤中语义不同。Lite Catalog 必须将语义字段唯一映射到这些列。

### 必须切换的 SigInsight 读路径

1. `pkg/litequery/catalog.go` 及 compiler/golden tests：主表、高频字段和 `*_present` 语义。
2. `pkg/telemetrylogs`、`pkg/telemetrytraces`、`pkg/telemetrymetrics`、`pkg/querybuilder/resourcefilter`
   的表名常量。
3. Trace Detail、Exceptions、Services、Service Map、Span Percentile、Trace Funnel、Live Logs、
   Rule State History、Metrics Explorer/metadata 与 stats reporter 的专用 SQL。这些路径不经
   Lite compiler，但同样是生产读路径。
4. Storage config、retention 表列表、schema readiness、启动检查、测试 fixture 和 OpenAPI
   SQL 示例。
5. 遗留 Trace Explorer SQL 常量如仅是未调用死路径，应删除而不是为保兼容而改写。

## 非目标

- 不为旧 ClickHouse 数据回填、双写或运行时 fallback。
- 不保留旧表名或 camelCase alias 以兼容外部 SQL。
- 不在本阶段删除 Logs/Traces `resource` JSON、`exp_hist`、Metrics `env`或多级 rollup。
- 不将专用 reader 强行改造为 `/api/v5/query_range`。

## 当前依赖与前置

- Collector `v2.0.1` 是唯一 writer，首次 `migrate sync up` 直接创建 M16 最终 schema；
  `async up` 在此版本没有 schema cleanup 工作。
- `RENAME TABLE` 不可用：ClickHouse 25.5.6 不会随表重命名更新 Materialized View
  的 source/target；本设计也不使用 rename 或数据复制。
- SigInsight 启动时在创建 Querier 前执行 schema readiness。缺少最终表/列、或仍存在任意
  M16 legacy 表时启动失败。

## 设计

Schema Catalog 是 SigInsight 语义字段和物理列之间的唯一映射层。语义字段名不变；
仅 Catalog 中的 ClickHouse 物理实现从旧名改为 canonical 名。专用 reader 没有
Catalog 抽象，因此必须显式替换其参数化 SQL 中的表名和 Trace 列名。Collector 仍以
冻结 DDL 作为 source，但只按原依赖顺序选择最终对象，重写 source/target 引用；不在运行
时保留旧对象或执行删除链。

不添加“新表失败时读旧表”的 fallback。它会重新引入两个 schema
契约，使可删除性、query log 审计和代码路径都不再确定。schema readiness
必须在启动时拒绝缺少 canonical 表/列的实例。

## API/IR/schema 变化

V5 HTTP 协议、Lite IR、前端语义字段名、分页和结果截断协议不变。变更只作用于
ClickHouse 物理存储资源：

- 表名按上方映射交换；
- Trace fast-path 列按上方映射交换；
- 生成 SQL不得含旧 versioned 业务表名、`resource_string_*$$*`、
  `attribute_string_*$$*` 或 camelCase Trace alias。

## 迁移与回滚

1. 停止 SigInsight 和 Collector，删除原有 `siginsight_*` ClickHouse 数据库或重建实例。
2. 使用本版本 Collector 依次执行 `migrate bootstrap`、`migrate sync up`、`migrate async up`。
   后两者的最终状态已相同；没有旧表创建或 cleanup 阶段。
3. 部署本阶段的 SigInsight 与 Collector v2.0.1，并启动 schema readiness。
4. 写入新的 OTLP fixture，完成下方验收矩阵后才恢复生产流量。

这是丢旧数据的破坏性升级，无运行时回滚。旧 Collector/SigInsight 不能连接最终 schema；恢复
只能停止服务、丢弃当前数据并用旧版本重新建库。

## 测试计划

1. 替换每个 table/column 常量后添加 SQL golden 测试：Lite Logs/Traces/Metrics/Meter compiler
   和专用 reader 均要断言 canonical 表名。
2. 为 Trace Catalog 增加 `service_name`、`http_route`、`*_present` 的字段映射与
   `NOT`/不存在语义测试；断言无 alias 可以生成。
3. 更新 Retention、schema readiness、OpenAPI 示例、fixture 和旧名 denylist 测试。
4. 在 ClickHouse 25.5.6 上运行 Collector `scripts/test-v1-baseline.sh`。脚本必须验证
   最终对象清单、表/列 fingerprint、无旧 DDL、MV source/target 与真实 OTLP 写入；再启动
   最新 SigInsight（SQLite 业务库）执行认证 API 协作测试。
5. 执行已认证 API 和前端回归，检查 ClickHouse `system.query_log`：无 unknown table/
   column，无旧表名或 alias 读取。

## 验收矩阵

| 场景 | 必须证明 |
| --- | --- |
| Logs | 原始查询、字段/values 补全、resource 过滤、分页、Live Logs |
| Traces | 面板查询、过滤、字段补全、Trace Detail、span tree、Funnel、Percentile |
| Services | operation 列表、Service Map、Exceptions |
| Metrics/Meter | 面板查询、Explorer、metadata、rollup 窗口、Meter |
| 告警 | 基础 Alert 预览/执行、rule history |
| Retention | 新表 TTL 读取、设置与异步状态 |
| 安全性 | 缺表/列启动失败，无 runtime fallback，query log 无旧名 |

## 实现结果

已完成。Collector v2.0.1 直接创建 canonical baseline，删除 legacy storage adoption、旧对象
create/drop chain、metadata seed 和无读取的 metric ingestion timestamp。SigInsight readers、
Lite Catalog、专用 reader、fixtures、OpenAPI 示例、retention 与 readiness 已切换至
canonical 名称。`schema readiness` 会拒绝缺少 canonical 表/列或仍存在任一 M16 legacy
对象的实例，因此不会静默回退到旧 schema。

`Materialized` 元数据不再是生成任意动态物化列名的授权。Logs 一律使用 `resource` JSON
和 typed map；Traces 仅 Schema Catalog 明确声明的九个 fast-path 可使用 canonical 列，其余
字段同样回落到 map。这样历史 metadata 中的该标志也不会再生成已删除的
`attribute_string_*$$*` 或 `resource_string_*$$*` SQL。

实现中发现并修正一项跨表正确性问题：metrics retention 曾依据旧表名推断时间列，导致
`metric_points` 错误使用 `timestamp_ms`。现在 canonical metrics 一律使用 `unix_milli`，并且
`GetTTL` 以 `database = 'siginsight_metrics'` 限定系统表查询。TTL 集成测试也以 database/name
联合定位表，避免同名 `resource_sets` 的 Logs 与 Traces 行被混淆。

### 实际验证（2026-08-06）

Collector 在 ClickHouse 25.5.6 上通过：

```bash
go test ./cmd/siginsightschemamigrator/schema_migrator \
  ./cmd/siginsightotelcollector/migrate \
  ./exporter/clickhouselogsexporter \
  ./exporter/clickhousetracesexporter \
  ./exporter/signozclickhousemetrics
./scripts/test-v1-baseline.sh
```

该 baseline 测试断言最终对象/列 fingerprint、MV source/target、无 legacy DDL 或
cluster/distributed 对象，并写入真实 OTLP Logs、Traces、Metrics 与 Meter。

通过最新 SigInsight、SQLite 业务库和本地构建 Collector image 的已认证协作测试：

```bash
SIGINSIGHT_OTEL_COLLECTOR_ROOT=/home/cbw/code/OtelCollector \
SIGINSIGHT_DOCKER_BUILD_NETWORK=host \
uv run pytest --clickhouse-version=25.5.6 -vv -s \
  src/compat/03_litequery_collector_collaboration.py

SIGINSIGHT_OTEL_COLLECTOR_IMAGE=siginsight-otel-collector:m16-local \
SIGINSIGHT_DOCKER_BUILD_NETWORK=host \
uv run pytest --clickhouse-version=25.5.6 -vv -s \
  src/compat/04_materialized_catalog.py src/ttl/01_ttl.py
```

完整脚本在两次独立运行中均为 `17 passed`（最新运行 `246.78s`）。协作测试的
`system.query_log` 摘要确认 Logs、Traces、Metrics 和 Meter 均实际读取各自的 canonical 表且返回
非空结果；Meter rollup 的异步触发会影响具体行数。没有 unknown table/column 或旧表读路径。

从 SigInsight 根目录可通过以下命令完整复现：

```bash
tests/integration/scripts/run-canonical-schema-cutover.sh
```

## 删除内容

预期删除旧表名常量、仅为 camelCase alias 存在的 Trace SQL 和对应测试 fixture。
不在 M16 删除 S3 候选的 `resource` JSON、`exp_hist`、`env`、rollup 或字段元数据表。

## 度量变化

Collector 净删除约 950 行 legacy bootstrap、rename/adoption、post-baseline cleanup 和相应测试；
SigInsight 删除旧 schema 读取/fixture 引用并增加 readiness、直接协作和 TTL 定位测试。该阶段的
重点是收敛唯一物理契约，而非为新功能增加抽象层。

## 残余风险与后续任务

- `resource` JSON 仍是旧 field mapper 与非字符串 resource 值的依赖；属于 S3。
- 指数直方图在 Collector 两份默认配置中仍启用，SigInsight 前端/后端仍有
  类型和聚合选项；在 UI 明确拒绝、Collector 关闭写入前不可删除 `exp_hist`。
- `env` 是 Metrics 主键的首列；需证明 resource identity 完全代替其语义后才能删。
- 多级 metric rollup 需要真实长时间窗 workload 和 query log，不可只用 fixture 判断。
