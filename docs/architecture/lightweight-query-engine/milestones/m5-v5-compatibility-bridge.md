# M5：V5 兼容桥与受控 API 接入

状态：Complete
关联 ADR：002、004、006、007、014
前置条件：M1、M2、M3、M4 已完成

> 历史说明：本里程碑记录迁移期 bridge。M8 已删除 `lightweight_engine_enabled`、legacy
> executor 和 fallback；最终运行时边界见 ADR-013。

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
- 多 aggregation、secondary aggregation、`limitBy`、opaque cursor、任意 Having。
- Time Series 非零 limit（尚无 top-series 两阶段计划）。raw/trace offset 可用。
- `LIKE`/`ILIKE`/`REGEXP`/`BETWEEN`、full-text、函数调用、变量 filter 值和 unary `NOT`。
- 所有 V5 后处理函数。`formatOptions.fillGaps` 由 V5 result adapter 明确实现，不进入
  Lite IR 或 SQL compiler。

## 实现结果

- 新增 `pkg/querier/liteadapter`，持有 V5 -> Lite、filter AST 与 Lite -> V5 response 转换。
- `pkg/litequery` 继续只依赖领域类型和 `Rows`/`QueryFunc` 接口；具体 ClickHouse `driver.Rows`
  适配器在 `pkg/querier/litequery.go`。
- `querier.QueryRange` 当前始终进入 Lite adapter；`UnsupportedError` 在 V5 边界映射为稳定的
  invalid-input capability error，不存在 legacy executor 或运行时回退。
- type/temporality 缺失时由 metadata store 批量解析成只读 map 后注入 adapter；Gauge 不需要
  temporality，Sum 与 Histogram 保留物理 Delta/Cumulative 语义，避免复制旧 metrics
  statement builder 的表选择逻辑。

## 字段元数据消歧（ADR-014）

2026-08-01 补充。早期 adapter 对 filter 文本中的裸字段名（如 `host.name`）依赖硬编码
白名单，导致 `log field "host.name" is not in the schema catalog` 一类错误持续漏项。
修复恢复旧引擎的字段元数据消歧能力，但限定在应用边界：

- 新增 `liteadapter.FieldKeySelectors`：转换前批量收集 context/data type 不完整的字段
  （filter 文本 token、select/group/order、日志聚合字段），去重后经
  `querier.liteMetadata` 调用 metadata store `GetKeysMulti` 查询。
- 查询结果以纯数据 `MetricMetadata.FieldKeys` 注入 `ToLite(request, metadata)`，核心
  `pkg/litequery` 不依赖 metadata store，SQL 编译器保持确定性。
- `resolveFieldMetadata` 按"显式上下文/类型优先 -> fallback 类型匹配 -> 同类型裸名
  resource 优先 -> 唯一候选才消歧 -> intrinsic 字段兜底"的顺序解析；多候选歧义不猜测，
  返回明确错误。
- 告警执行器（`RuleQueryRunner`）与 UI 查询共用同一套字段解析规则。
- 前端生成筛选表达式时保留 `resource`/`tag` 上下文，结构化字段不再丢失上下文。
- 时间范围在 metadata store 访问前校验；禁用 builder 不参与字段或指标 metadata 查询。
- Catalog 对 Logs/Traces resource 和 Logs scope 的 string-only 物理 map 进行最终类型校验。

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
`SIGINSIGHT_QUERIER_LIGHTWEIGHT__ENGINE__ENABLED=true` 发起认证的 `/api/v5/query_range` 请求。
容器日志确认每个请求进入 Lite bridge；Metrics、Meter 和 `meter * 2` formula 返回非空 V5 series。

2026-08-01，M7 的当前 Collector fixture 已完成这一门槛：真实 OTLP Logs、Traces、Metrics
和 Meter 写入后，四类请求均经认证 API、V5 转换和实际 SQL 返回非空结果；ClickHouse query
log 中对应四张物理表的查询均无错误。

## M7 闭环结果

- `tests/integration/scripts/run-litequery-collector-collaboration.sh` 固化 ClickHouse 25.5.6、
  当前 Collector migration/OTLP 写入、SQLite 身份数据和 SigInsight 认证查询。
- opaque cursor、variables 和未支持的 Time Series limit 继续明确拒绝，不执行静默降级。
- legacy engine 已在 M8 删除，不再维护双跑等价矩阵；Lite 支持边界由能力矩阵和显式错误定义。
