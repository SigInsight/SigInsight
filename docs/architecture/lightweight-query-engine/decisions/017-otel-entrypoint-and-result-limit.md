# ADR-017：OTel Entrypoint、V5 Identity 与结果截断协议

状态：Accepted
日期：2026-08-02

分页与截断相关的第 4-6 条已由 [ADR-019](019-pagination-and-truncation-semantics.md) 取代；
Entrypoint 和 V5 identity 决策继续有效。

## 背景

Trace Detail 曾用 Collector `top_level_operations` 中的 `(name, serviceName)` 推断
Entrypoint。该目录是 operation 聚合，不是 span 身份集合；同名 operation 会扩大匹配，
Client span 也可能被误判。与此同时，部分前端列表仍兼容 V4 camelCase identity，导致 V5
返回 `span_id` 时去重键变成 `undefined`。单次 raw limit 还无法区分“结果刚好等于 limit”与
“结果被截断”，页面只能静默展示不完整数据。

## 决策

1. Entrypoint 使用 OTel 接收边界语义：span 必须有 parent，`kind` 为 Server(2) 或
   Consumer(5)，且 `is_remote = 'yes'`。Root 保持 `parent_span_id = ''`，两者互不替代。
2. Lite compiler 不再读取 `siginsight_traces.top_level_operations`。Collector 可以继续维护
   该表供其他功能使用，但它不再是 Trace Detail scope 的查询契约。
3. V5 raw transport identity 只使用 `trace_id` 与 `span_id`。前端 API 类型、Trace Explorer、
   Trace Detail filter 和关联日志不得回退读取 `traceID`、`traceId`、`spanID` 或 `spanId`。
   Trace Waterfall 内部领域模型的 camelCase prop 不属于 wire contract。
4. 有结果 limit 的 SQL 实际请求 `limit + 1`。Executor 用额外一行探测溢出，裁回请求 limit，
   并保留已查询数据；V5 response 同时返回 warning code `result_limit_reached`。这是部分成功，
   不是 HTTP error。
5. 所有普通 `/api/v5/query_range` 调用在统一 API client 边界展示 warning。需要自行分页的
   调用可以抑制中间页提示，但必须在自己的总量预算到达时展示最终 warning。
6. Trace Detail filter 每页最多查询 1,000 个 span，按 offset 循环，最多累计 10,000 个；
   中间页依据 `result_limit_reached` 继续，未截断页立即结束。达到总量上限时只高亮前
   10,000 个并明确提醒。
7. raw 查询的自定义排序必须形成稳定全序；compiler 自动在用户排序后追加 transport
   identity tie-breaker，Trace 使用 `span_id`，Logs 使用 `id`，避免 offset 页边界重复或漏行。

## 影响

- Entrypoint 表示服务接收到远端上下文的 OTel span，不再表示“出现在 operation 目录中的
  非 root span”。在同一 trace 有多个跨服务入口时会高亮多条 Server/Consumer span。
- 已存在但缺少正确 `is_remote` 或 `kind` 的历史数据不会被推断为 Entrypoint；这是采用 OTel
  语义后的明确数据质量边界。
- raw、trace summary 和带 limit 的聚合都能区分完整结果与截断结果，前端不会再静默隐藏
  截断事实。
- `limit + 1` 只检测最终结果行是否溢出，不限制 ClickHouse 高基数聚合的中间状态；该风险
  仍按 ADR-016 的 ClickHouse 侧预算后续项处理。

## 验证

- compiler 单测验证 OTel Entrypoint SQL、Root 独立语义、`limit + 1` 和 offset 参数顺序。
- ClickHouse 25.5.6 集成测试直接写入 `kind/is_remote` 并验证 Root/Entrypoint 命中。
- 生产样本 trace `ba4939679b3dc931805817a8c8fefa67` 实测 1,553 spans、1 Root、
  276 OTel Entrypoints。
- executor/adapter 单测验证额外行被裁剪，并返回结构化 `result_limit_reached` warning。
- 前端 API 测试验证统一 toast 及分页抑制；Trace Detail 测试验证 1,000 分页、10,000 上限、
  canonical `span_id` 和拒绝 V4 identity。
