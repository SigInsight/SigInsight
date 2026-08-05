# ADR-012: 在 V5 边界退役 Trace Operator

Status: Superseded by M14
日期：2026-07-31
关联里程碑：M8

## 背景

Trace Operator 将多个 span 查询用父子、祖先或布尔关系组合。它要求前端专用编辑器和
ANTLR parser，并要求后端为引用查询构造跨 CTE SQL；该能力不属于核心 Logs、Traces、
Metrics/Meter 面板或基本阈值告警的范围。

Lite IR 从 M1 起就不表达 Trace Operator。此前 generic V5 fallback 仍可以执行保存的
`builder_trace_operator` 请求，这使“新 UI 不显示”与“生产仍支持”不一致。

## 决策

- 删除 Alert、Trace Explorer 和共享 QueryBuilder 的 Trace Operator 入口、控件、样式、
  前端 grammar/parser 及其测试。
- 删除 generic Querier 的 Trace Operator query、postprocess 分支、statement-builder
  interface 和 trace CTE compiler。
- 继续保留 V5 DTO 的 JSON 解码和 enum 值，以读取已保存的查询；validation 和
  `Querier.QueryRange` 都在 SQL 编译之前返回 `trace operator queries are no longer
  supported`。

## 后果

- 已保存的复杂 trace 查询会得到确定错误，不会丢弃关系条件后作为普通 trace 查询执行。
- 新页面无法创建该能力；核心单查询 trace filter/list 工作流保持不变。
- DTO 兼容层是临时的只读迁移边界，不能重新引入 SQL compiler 或 legacy fallback。
- 后续确认不再需要读取旧保存对象后，才可以删除 V5 DTO、生成的 OpenAPI variant 和
  前端 `queryTraceOperator` 兼容字段。

## 后续决策

M14 明确允许丢弃旧保存查询数据，因此上述临时只读兼容边界已删除。V5 不再声明、解码
或校验 `builder_trace_operator`；旧对象必须删除或重新创建为当前 Builder Query/Formula。
