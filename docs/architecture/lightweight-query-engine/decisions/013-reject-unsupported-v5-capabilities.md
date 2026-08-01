# ADR-013: Lite-only V5 边界拒绝高级能力

日期：2026-07-31

## 决策

`/api/v5/query_range` 只执行 Lite 支持的查询。适配器遇到 raw ClickHouse SQL、
join/subquery、offset/cursor、任意 Having、secondary aggregation 或 post-processing
functions 时返回稳定的 invalid-input capability error。legacy compiler、cache 和
postprocess 已物理删除，因此不存在配置开关或隐式回退路径。

Trace Operator 保持显式拒绝。告警编辑器不显示 ClickHouse SQL 标签；加载旧的
ClickHouse 告警 URL 会切回 Builder 标签，避免让用户新建运行时会拒绝的请求。

## 原因

受支持的 Logs、Traces、Metrics 与 Meter 已在 Collector 写入的 ClickHouse 25.5.6 schema
上完成真实写入和认证 API 读回。迁移期的 legacy/Lite 双跑曾用于建立等价证据；继续保留
回退会让生产部署永久依赖两套 compiler、cache 和 postprocess 语义，阻碍删除。

## 后果

保存的高级 V5 查询得到可识别的错误，而不是部分降级或隐藏执行路径。前端提供“Replace
query”迁移入口；不能由 Lite 表达的能力必须作为受限专用 reader 单独设计，不能重新引入
通用 SQL compiler。
