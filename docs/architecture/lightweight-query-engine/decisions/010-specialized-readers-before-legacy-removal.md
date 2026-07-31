# ADR-010: 专用读取 API 先于 Legacy V5 删除

Status: Accepted
日期：2026-07-31
关联里程碑：M8

## 背景

轻量引擎的目标是承担受约束的 Logs、Traces、Metrics、Meter 面板查询，且不复制
V5 的多聚合、`countIf`、Trace Operator、raw SQL、offset 导出和 Live Logs 语义。

静态审计发现，`pkg/querier.Querier.QueryRange` 不只由 HTTP 面板使用：Services、Span
Percentile、Raw Data Export 和 threshold rules 也直接调用它。Services 和 Span
Percentile 构造了 Lite 不表达的多聚合 V5 请求；Raw Export 和 Live Logs 依赖不同的
分页/流式语义。

## 决策

不扩展 Lite IR 来容纳这些专用读取工作流，也不在删除前将其保留为对 legacy V5
`QueryRange` 的隐式依赖。

每个仍需保留的专用工作流必须迁移到一个以其领域响应为中心的读取 API：

- Services 使用专用 Trace service/operation statistics reader；
- Span Percentile 使用专用 percentile reader；
- Raw Export 和 Live Logs 分别使用明确的分页/流式读取契约；
- threshold rules 只接纳 Lite capability matrix 中的基本查询，其他规则在保存或评估
  时返回稳定的不支持错误。

这些 readers 可以共享 TelemetryStore、Schema Catalog 的字段解析或安全的 filter
parser，但不得返回 V5 `QueryRangeResponse` 或重新建立通用 Query Builder DTO。

## 后果

- legacy V5 删除顺序由外部依赖清零决定，而不是由前端是否已显示 Lite editor 决定；
- Services/Percentile 的必要多聚合留在小型领域 SQL 中，不污染 Lite 的单聚合模型；
- Raw Export 和 Live Logs 成为显式保留或单独下线的产品决定；
- 当所有专用调用迁出后，legacy 的 builder/compiler/cache/postprocess 才能按可验证的
  依赖图删除。
