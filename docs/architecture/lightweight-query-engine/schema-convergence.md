# Schema 收敛审核：面向轻量查询引擎的 ClickHouse 第一版契约

状态：Reviewed（仅设计审核，尚未授权代码、DDL 或 migration 实现）
日期：2026-08-05
适用 ClickHouse：`clickhouse/clickhouse-server:25.5.6`

## 1. 结论

当前 ClickHouse schema 可以在不降低现有 UI 和轻量查询能力的前提下收敛，但不能只按 Lite Compiler 的直接 SQL 删除对象。Trace Detail、Service Map、Exceptions、Metrics Explorer、字段补全、Retention 和资源过滤仍使用专用表或 reader。

建议分三步实施：

1. 先删除已确认无业务读取和有效写入的历史对象，继续使用当前表名。这一步不引入 canonical 新表。
2. 在一个停机协作里程碑中创建无版本后缀的 canonical schema，同时切换 Collector writer、SigInsight Catalog 和专用 reader。允许丢弃旧数据，不双写、不回填。
3. 只在局部消费者先完成替换后，再删除 `resource` JSON、exponential histogram 等仍有代码依赖的对象。

本轮审核确认的首批新增删除项是：

- traces `durationSort`、`span_index_v2`、`legacy_spans`、`usage_explorer`、`usage_explorer_mv`；
- `siginsight_metadata.attributes_metadata` 及其唯一 writer `metadataexporter`；
- metrics `samples_v4.inserted_at_unix_milli`、`time_series_v4.inserted_at_unix_milli`；
- canonical Trace 主表中的全部 camelCase `ALIAS`，但必须与仓库残留的旧 SQL 常量清理在同一提交完成。

`root_operations`、`sub_root_operations` 不是旧 MV。它们从 `span_index_v3` 写入 `top_level_operations`，支撑 Services operation 列表，必须作为同一个能力单元保留或一起重构。

## 2. 审核约束

1. 允许直接丢弃旧 ClickHouse 数据；不双写、不回填。
2. 删除依据是当前 SigInsight 生产入口、专用 API、Lite Catalog、Collector writer 和真实 workload 的交集。
3. 不降低 Logs、Traces、Metrics、Meter UI、Trace Detail、Service Map、Exceptions、基本 Alert、字段补全和 Retention 的当前能力。
4. 不进行大规模查询引擎重写；只接受能减少对象、重复存储、写放大或过渡契约的局部调整。
5. 每项实现必须同时补 Collector migration/exporter 测试、SigInsight reader/Catalog 测试和 ClickHouse 25.5.6 协作测试。
6. 新业务表名和列名不使用 `_v2`、`_v3` 等版本后缀。版本由 migration ID、schema fingerprint 和 readiness capability 表达。

已有 `schema_migrations_v2` 是当前 migrator 的内部账本，不是新业务表。本轮不因命名策略单独重写 migrator；若以后替换账本，新名称应为 `schema_migrations`。

## 3. 证据和优先级

删除判断的证据优先级如下：

1. 最新工作分支的生产路由、前端调用、reader 和 SQL；
2. Collector writer、pipeline 配置、MV source/target 和 DDL；
3. ClickHouse 25.5.6 真实协作测试；
4. 覆盖一个完整业务周期的 `system.query_log`；
5. 当前数据行数和大小。

本机快照使用 ClickHouse 25.5.6，共 53 个 `siginsight_*` 对象。主要数据量为：

| 对象 | 行数 | 压缩大小 |
| --- | ---: | ---: |
| `siginsight_logs.logs_v2` | 51,134,302 | 4.40 GiB |
| `siginsight_traces.span_index_v3` | 91,654,020 | 8.08 GiB |
| `siginsight_traces.dependency_graph_minutes_v2` | 17,699 | 165.70 MiB |
| `siginsight_metrics.samples_v4` | 31,436,301 | 44.68 MiB |
| `siginsight_metrics.time_series_v4` | 704,660 | 25.08 MiB |
| `siginsight_metadata.attributes_metadata` | 196,995 | 31.08 MiB |

query log 只作否定结论的辅助证据。本机运行的 SigInsight 镜像不等于最新工作分支，例如旧镜像仍查询 `updated_metadata`，而最新代码已经删除该 reader。因此“旧镜像查询过”不能推翻目标版本的源码证据；反过来，“测试数据没有查询过”也不能单独证明可删。

## 4. 对象审核矩阵

### 4.1 首批删除：无需降低可见能力

| 对象 | 证据 | 实现时必须同步清理 |
| --- | --- | --- |
| traces `durationSort` | 空表；无 writer；最新 SigInsight 无读取 | baseline/bootstrap 兼容项和 denylist 测试 |
| traces `span_index_v2` | 空表；writer 只写 v3；无当前 SELECT | bootstrap rename、历史 retention 映射 |
| traces `legacy_spans` | 空表；无 writer、无当前 reader | bootstrap rename、历史 retention 映射 |
| traces `usage_explorer`、`usage_explorer_mv` | 表为空；MV 错误地从不再写入的 `span_index_v2` 读取；业务代码只把目标表加入 TTL 列表 | Retention config、TTL 集成 fixture 和 baseline DDL |
| metadata `attributes_metadata` | 30 小时无 SELECT；只服务 `existingQuery` 的 related-values 推荐；当前生产前端从不传 `existingQuery` | 后端 related-values 分支、metadata table 常量 |
| Collector `metadataexporter` | 唯一持久化输出是 `attributes_metadata`；其他字段补全由各 signal keys/value 表提供 | component 注册、三条 pipeline、配置、缓存和 exporter 测试 |
| metrics 两个 `inserted_at_unix_milli` 列 | 最新 SigInsight 无读取；不参与 `ORDER BY`、TTL 或 ReplacingMergeTree version | Collector INSERT、batch struct、DDL fingerprint |

以下对象已由现有 post-baseline migration 删除或目标实例不存在。canonical baseline 不得重新创建：

- `span_attributes`、`samples_v2`、`time_series_v2`；
- 所有 `distributed_*` 对象；
- `column_evolution_metadata`、`updated_metadata`；
- `body_v2`、`body_promoted`、`json_path_types`、`distributed_json_path_types` 及相关索引。

### 4.2 必须保留：当前存在有效消费者

| 能力单元 | 必须保留的对象 |
| --- | --- |
| Logs 查询和资源过滤 | `logs_v2`、`logs_v2_resource` |
| Logs 字段和值补全 | `logs_attribute_keys`、`logs_resource_keys`、logs `tag_attributes_v2` |
| Logs 用量和 retention | logs `usage`、主表/资源表 retention 列 |
| Traces 查询和资源过滤 | `span_index_v3`、`traces_v3_resource` |
| Trace 字段和值补全 | `span_attributes_keys`、traces `tag_attributes_v2` |
| Trace Detail | `trace_summary`、`trace_summary_mv` 以及主表的 `events`、`links` |
| Services operation | `top_level_operations`、`root_operations`、`sub_root_operations` |
| Service Map | `dependency_graph_minutes_v2` 和三个生产 MV |
| Exceptions | `error_index_v2` |
| Trace 用量 | traces `usage` |
| Metrics 查询 | `samples_v4`、`time_series_v4` |
| Metrics 长时间范围 | `samples_v4_agg_5m/30m` 及 MV、`time_series_v4_6hrs/1day/1week` 及 MV |
| Metrics Explorer/metadata | metrics `metadata`、`time_series_v4*` |
| Metrics 用量 | metrics `usage` |
| Meter | `samples`、`samples_agg_1d`、`samples_agg_1d_mv` |
| Alert history | `rule_state_history_v0` |

### 4.3 第二批：必须先替换消费者

| 候选 | 当前阻塞 | 允许删除的前置条件 |
| --- | --- | --- |
| `exp_hist` | Collector 配置仍启用 exponential histogram；前端仍展示该 metric type | 明确 capability 不支持、关闭 exporter 分支、移除 UI 选项和 retention 项，并补拒绝测试 |
| Logs/Traces `resource` JSON | 旧 field mapper 仍将 resource context 映射到 JSON；可能保留非字符串 resource 值 | 所有当前入口统一使用 typed/resource maps，并验证数值/布尔资源属性的产品边界 |
| Trace `links` | Trace Detail 直接读取 `links AS references` 构造 span 关系及展示 | 分离 `parent_span_id` 树关系和 OTel Span Links 展示语义后再决定，不能用 parent 列直接替代全部 links |
| Metrics `env` | writer 从 `deployment.environment` 生成，且当前是主键首列，虽然 SigInsight 不直接过滤 | 证明 deployment environment 已包含在 resource identity/fingerprint，并做重复 series 测试 |
| 多级 metric rollup | Metrics Explorer 和 metadata reader 仍按时间窗选择 | 用真实时间跨度 workload 证明可由较少层级满足延迟和成本目标 |
| 各 signal metadata 表合并 | 当前 `/fields/keys`、`/fields/values` 使用不同表和静态字段路径 | 先建立统一 `field_catalog` reader/writer；本轮不为“表少”进行大改造 |

## 5. Canonical 主表列 allowlist

这里的 allowlist 是目标契约，不代表现在可以逐列执行 `ALTER DROP`。实现时所有 reader、writer、索引、TTL 和 MVs 必须同时通过测试。

### 5.1 `siginsight_logs.logs`

保留：

```text
ts_bucket_start, resource_fingerprint,
timestamp, observed_timestamp, id, trace_id, span_id, trace_flags,
severity_text, severity_number, body,
attributes_string, attributes_number, attributes_bool, resources_string,
scope_name, scope_version, scope_string,
_retention_days, _retention_days_cold,
resource
```

`resource` 是第二批候选，在旧 field mapper 退出前暂列 allowlist。`observed_timestamp` 属于 OTel 日志语义且已暴露给 Lite Catalog，不能因默认 UI 不展示而删。

### 5.2 `siginsight_traces.spans`

保留：

```text
ts_bucket_start, resource_fingerprint,
timestamp, trace_id, span_id, trace_state, parent_span_id, flags,
name, kind, kind_string, duration_nano,
status_code, status_message, status_code_string,
attributes_string, attributes_number, attributes_bool, resources_string,
events, links,
response_status_code, external_http_url, http_url, external_http_method,
http_method, http_host, db_name, db_operation, has_error, is_remote,
service_name, http_route, messaging_system, messaging_operation,
db_system, rpc_system, rpc_service, rpc_method, peer_service,
service_name_present, http_route_present, messaging_system_present,
messaging_operation_present, db_system_present, rpc_system_present,
rpc_service_present, rpc_method_present, peer_service_present,
resource
```

目标 canonical 名称将当前 `resource_string_service$$name` 等物化列改为可读名称；Lite Catalog 仍负责语义字段到物理列的唯一映射。`*_present` 必须保留，因为 `NOT`、不存在判断和空字符串的语义不同。

不进入 canonical 表的 camelCase `ALIAS`：

```text
traceID, spanID, parentSpanID, spanKind, durationNano,
statusCode, statusMessage, statusCodeString, references,
responseStatusCode, externalHttpUrl, httpUrl, externalHttpMethod,
httpMethod, httpHost, dbName, dbOperation, hasError, isRemote,
serviceName, httpRoute, msgSystem, msgOperation, dbSystem,
rpcSystem, rpcService, rpcMethod, peerService
```

这些 alias 不占数据存储，收益是缩小 SQL 契约而非节省容量。仓库仍有一组只定义未调用的旧 Trace Explorer SQL 常量引用它们；实现必须先删除这些死常量并增加“生成 SQL 不包含 alias”的测试。

### 5.3 `siginsight_metrics.metric_points`

```text
env, temporality, metric_name, fingerprint, unix_milli, value, flags
```

`inserted_at_unix_milli` 不进入 canonical allowlist。`env` 暂保留，等待第二批 identity 审核。

### 5.4 `siginsight_metrics.metric_series`

```text
env, temporality, metric_name, description, unit, type, is_monotonic,
fingerprint, unix_milli, labels, attrs, scope_attrs, resource_attrs,
__normalized
```

`labels` 仍被 Lite metric compiler、Metrics Explorer 和 metric metadata 读取；`__normalized` 被 Explorer 大量用于区分 series，不能删除。

### 5.5 `siginsight_meter.meter_points`

```text
temporality, metric_name, description, unit, type, is_monotonic,
labels, fingerprint, unix_milli, value
```

Meter 当前把 series identity 和 point 放在同一张表，本轮不为形式统一拆表。

## 6. Canonical 对象命名

第一版业务对象不带版本后缀：

| 当前对象 | Canonical 对象 |
| --- | --- |
| `logs_v2` | `logs` |
| `logs_v2_resource` | `resource_sets`（database 为 `siginsight_logs`） |
| logs `tag_attributes_v2` | `field_values` |
| `span_index_v3` | `spans` |
| `traces_v3_resource` | `resource_sets`（database 为 `siginsight_traces`） |
| traces `tag_attributes_v2` | `field_values` |
| `top_level_operations` | `operations` |
| `dependency_graph_minutes_v2` | `service_edges` |
| `error_index_v2` | `exceptions` |
| `samples_v4` | `metric_points` |
| `time_series_v4` | `metric_series` |
| `samples_v4_agg_5m/30m` | `metric_rollup_5m/30m` |
| `time_series_v4_6hrs/1day/1week` | `series_rollup_6h/1d/1w` |
| analytics `rule_state_history_v0` | `rule_state_history` |
| meter `samples` | `meter_points` |
| meter `samples_agg_1d` | `meter_rollup_1d` |

MV 名称可使用 `_mv`，因为它描述对象类型而不是 schema 版本。`root_operations`/`sub_root_operations` 在实现时应重命名为表达输入语义的 MV 名称，但不得改变它们共同写入 `operations` 的行为。

## 7. 实施里程碑

### S1：删除死对象

- 删除 4 个 legacy trace 表/目标和 `usage_explorer_mv`；
- 删除 `attributes_metadata` 和整个 `metadataexporter` pipeline；
- 从现有 metric 表删除两个 `inserted_at_unix_milli`；
- 保持所有现有核心表名和查询不变；
- 通过真实字段补全、Retention 和 exporter 回归。

### S2：停机切换 canonical schema

1. 停止 SigInsight 和 Collector 写入。
2. 创建 canonical schema；不从旧表复制数据。
3. 同一发布单元切换 Collector INSERT、Lite Catalog、资源过滤和全部专用 reader。
4. 启动前执行 schema readiness；缺失任一表/列立即失败。
5. 写入新的 OTLP fixtures 后执行全部真实 API/UI 验证。
6. 删除旧 versioned 对象并执行 denylist。

不允许先切 writer 或 reader 的一侧，也不允许在运行时按“新表失败则读旧表”回退；那会重新引入双契约。

### S3：有前置重构的精简

- 评估并删除 `resource` JSON；
- 明确禁用 exponential histogram 后删除 `exp_hist`；
- 用 workload 决定 metric rollup 层级；
- 评估 `env` 是否由 resource identity 完整替代；
- 只有明确减少总 SQL/对象复杂度时才统一 field catalog。

## 8. 测试门槛

### Collector

- `system.tables/system.columns` canonical allowlist fingerprint；
- 旧对象、旧列、camelCase alias 和业务 `_v2/_v3` 名称 denylist；
- logs/traces/metrics/meter exporter INSERT 列顺序、类型和默认值；
- 25.5.6 真实 OTLP Gauge、Sum、Regular Histogram、logs、traces、meter 写入；
- MVs 的 source、target、字段和结果行验证；
- metadataexporter 删除后，pipeline 启动且 keys/value metadata 仍产生；
- TTL、partition、skip index 和 resource fingerprint 行为。

### SigInsight

- Catalog 每个 semantic field 的 SQL golden test；
- Logs/Traces/Metrics/Meter raw、time series、scalar、分页和 truncation warning；
- `/api/v5/fields/keys`、`/api/v5/fields/values`，覆盖 string/number/bool、LIKE/REGEX 和无 `relatedValues`；
- Trace Detail、Services、Service Map、Exceptions、Metrics Explorer、Retention、Alert history；
- 所有生成 SQL 不包含旧表名或 Trace camelCase alias；
- schema readiness 缺失表/列时拒绝启动，而不是在请求时返回 unknown table/column。

### 前端与真实协作

- Logs/Trace/Metrics 常见筛选、分页、Trace Detail 跳转和 span tree；
- QueryBuilderSearchV3 字段和值补全；
- Metrics Explorer summary/explorer/inspect；
- 基本 Alert 创建、预览、执行和 history；
- ClickHouse query log 无未知表、未知列、旧 versioned 对象；
- Playwright desktop/mobile 回归。

## 9. 实现准入

只有以下条件全部满足才进入代码阶段：

- S1/S2 的对象清单和 canonical 名称已审核；
- 每个删除列都有最新代码引用扫描、writer 证明和测试计划；
- canonical DDL 可在空 ClickHouse 25.5.6 独立创建；
- Collector 与 SigInsight 使用同一个 schema fingerprint；
- 已准备可重复执行的真实协作脚本；
- 明确 S2 是停机且丢数据的破坏性升级，不承诺旧 Collector、旧 SigInsight 或旧 SQL 客户端兼容。
