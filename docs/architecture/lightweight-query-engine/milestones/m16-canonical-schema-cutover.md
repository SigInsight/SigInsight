# M16: Canonical ClickHouse Schema Cutover

状态：Planned

负责人：SigInsight query/storage 实施者

起止提交：SigInsight TBD；OtelCollector `777a8a9`

关联 ADR：[ADR-003](../decisions/003-schema-catalog-contract.md)、[ADR-010](../decisions/010-specialized-readers-before-legacy-removal.md)

## 问题与目标

Collector 已实现无版本后缀的 ClickHouse 存储契约，并在 ClickHouse 25.5.6 验证了新表、
Materialized View 链和 OTLP 写入。SigInsight 仍读取 `span_index_v3`、`logs_v2`、`samples_v4`
等旧对象，Trace 查询仍依赖已被删除的 camelCase alias。两侧若不同时切换，新
Collector 写入的数据不会被旧 SigInsight 读到，async cleanup 后则会直接出现 unknown
table/column。

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

- Collector branch `refactor/schema-convergence-s1` 的 `777a8a9` 包含 S1/S2：canonical
  sync-create/async-drop migration、writer 切换、MV 重写和 ClickHouse 25.5.6 集成脚本。
- 新表由 sync migration 创建；旧 versioned 表只能在两端切换后由 async migration 删除。
- `RENAME TABLE` 不可用：ClickHouse 25.5.6 不会随表重命名更新 Materialized View
  的 source/target。
- 当前 SigInsight 仍有旧表名/旧列的运行时引用；在此完成前不允许部署
  Collector S2 或执行其 async cleanup。

## 设计

Schema Catalog 是 SigInsight 语义字段和物理列之间的唯一映射层。语义字段名不变；
仅 Catalog 中的 ClickHouse 物理实现从旧名改为 canonical 名。专用 reader 没有
Catalog 抽象，因此必须显式替换其参数化 SQL 中的表名和 Trace 列名。

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

1. 停止 SigInsight 和 Collector 写入。
2. 用 Collector `migrate sync up` 创建 canonical 表和 MV。不复制旧数据。
3. 部署包含本阶段的 SigInsight 和 Collector `777a8a9`。
4. 启动前运行 schema readiness；缺失新表或列必须使启动失败。
5. 写入新的 OTLP fixture并完成下方验收矩阵。
6. 只在验收通过后运行 Collector `migrate async up` 删除旧对象。

这是丢旧数据的破坏性升级，无运行时回滚。在 async cleanup 前只能通过停服、重新部署旧两侧到旧表恢复；
一旦执行 cleanup，回滚等价于丢弃当前空数据并重建 schema。

## 测试计划

1. 替换每个 table/column 常量后添加 SQL golden 测试：Lite Logs/Traces/Metrics/Meter compiler
   和专用 reader 均要断言 canonical 表名。
2. 为 Trace Catalog 增加 `service_name`、`http_route`、`*_present` 的字段映射与
   `NOT`/不存在语义测试；断言无 alias 可以生成。
3. 更新 Retention、schema readiness、OpenAPI 示例、fixture 和旧名 denylist 测试。
4. 在 ClickHouse 25.5.6 上运行 Collector `scripts/test-v1-baseline.sh`，再启动最新
   SigInsight，视需求使用 SQLite 作为本地业务数据库。
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

未开始。完成时需记录 SigInsight 提交、Collector 提交、实际命令、认证 API 结果、
Playwright 关键页面回归和 query-log 摘要。

## 删除内容

预期删除旧表名常量、仅为 camelCase alias 存在的 Trace SQL 和对应测试 fixture。
不在 M16 删除 S3 候选的 `resource` JSON、`exp_hist`、`env`、rollup 或字段元数据表。

## 度量变化

待实现后填写：新增/删除 production LOC、删除的 SQL 常量和测试数量。

## 残余风险与后续任务

- `resource` JSON 仍是旧 field mapper 与非字符串 resource 值的依赖；属于 S3。
- 指数直方图在 Collector 两份默认配置中仍启用，SigInsight 前端/后端仍有
  类型和聚合选项；在 UI 明确拒绝、Collector 关闭写入前不可删除 `exp_hist`。
- `env` 是 Metrics 主键的首列；需证明 resource identity 完全代替其语义后才能删。
- 多级 metric rollup 需要真实长时间窗 workload 和 query log，不可只用 fixture 判断。
