# ADR-013: 默认 Lite 部署在 V5 边界拒绝高级能力

日期：2026-07-31

## 决策

当 `querier.lightweight_engine_enabled=true` 时，`/api/v5/query_range` 只执行 Lite
支持的查询。适配器遇到 raw ClickHouse SQL、join/subquery、offset/cursor、任意 Having、
secondary aggregation 或 post-processing functions 时返回稳定的 invalid-input capability
error，不再隐式回退到 `pkg/querier` 的 legacy compiler。

把配置显式设为 `false` 仍可整体使用 legacy engine，供旧保存查询迁移或紧急回退；它不是
默认生产路径。Trace Operator 保持既有的显式拒绝，无论开关状态都不会恢复执行。

Lite feature flag active 时，告警编辑器不显示 ClickHouse SQL 标签。加载旧的 ClickHouse
告警 URL 会切回 Builder 标签，避免让用户新建默认运行时已拒绝的请求。

## 原因

受支持的 Logs、Traces、Metrics 与 Meter 已在 Collector 写入的 ClickHouse 25.5.6 schema
上和 legacy 双跑对比。继续让未支持的请求静默回退，会让默认部署同时依赖两套 compiler、
cache 和 postprocess 语义，阻碍删除。

## 后果

保存的高级 V5 查询在默认配置下会得到可识别的错误，而不是部分降级或隐藏执行路径。后续
删除 legacy 代码前，需要逐项删除其前端入口或提供独立的、受限的专用 reader。
