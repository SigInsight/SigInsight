# ADR-016：查询正确性不变量与结果预算

状态：Accepted
日期：2026-08-01

## 背景

Lite engine 替换 legacy engine 后，审计发现若干不会产生 SQL error、但会返回错误或不稳定
结果的风险：Trace filter 只统计命中 span、缺 root 的 trace 被丢弃；formula/series key 可能
碰撞；Time Series 点顺序受 SQL order 影响；物理表名与时间表达式分散；Time Series
`LIMIT` 实际截断 bucket 行而不是选择 series；无限结果行可能耗尽应用内存。

## 决策

1. Trace summary 分三步执行：匹配 trace ID、读取时间范围内完整 span、生成统计与代表
   span。代表 span 优先最长 root，root 缺失时回退到最长 span；duration 是代表 span
   duration，不是所有 span duration 之和；span count 是完整 trace span 数。
2. Formula 依赖必须拥有相同 timestamp/group 列 schema。对齐 key 使用长度前缀和运行时
   类型编码；缺失对齐值以零参与并产生 warning；输出保持公式声明顺序。
3. V5 Time Series response 保持 series 首次出现顺序，并对每条 series 的 points 按
   timestamp 稳定升序。非有限数不进入 Time Series，Scalar 中转换为 `null`。
4. 表名、时间范围与 bucket 表达式全部由同一个 Catalog table 选择驱动，禁止 compiler
   同时使用自定义 table 与硬编码默认表。
5. Time Series 非零 limit 在 top-series 计划落地前明确拒绝。Executor 对每条 statement
   默认最多扫描 250,000 个结果行，超限关闭 rows 并返回 budget error。
6. V5 opaque cursor 不支持；Live Logs 的 typed `(timestamp,id)` cursor 是独立契约。
   `formatOptions.fillGaps` 在 result adapter 补零，不进入 Lite IR，并与 SQL 一样使用
   `[start,end)` 半开区间和 epoch-aligned bucket 计数。
7. Histogram 必须携带 metadata 解析出的 Delta/Cumulative temporality。Delta 点在查询
   bucket 内求和；Cumulative 点取最新快照并跨 bucket 求差，禁止默认伪装为 Cumulative。
8. Executor deadline 使用独立的领域 timeout 分类；V5 边界映射为标准 timeout 错误，不能
   伪装成 unsupported/invalid-input。调用方取消继续透传 `context.Canceled`。
9. `selectFields` 只属于 raw 结果；前端切换到 aggregate/trace-summary 视图时不发送陈旧
   select，直接 API 请求则明确拒绝。raw select 的输出名必须唯一，避免 V5 map 覆盖同名字段。
10. 前端 formula capability 校验与 Lite IR 共用同一语法边界：名称、十进制常量、括号和
    `+ - * /`；函数、unary 运算和 literal-zero division 均拒绝，合法常量公式不能被 UI 误判。
11. metadata 前置解析必须先验证 `[start,end)`，且只收集启用 builder 的字段和 metric；
    禁用的保存查询不能触发 store 访问或阻断启用查询。Logs/Traces resource 与 Logs scope
    的物理 map 只接受 string，由 Catalog 作为最终 schema 边界强制执行。
12. 前端 serializer 清除 Time Series 的陈旧 limit；执行 deadline 到达后，即使 driver 返回
    自有中断错误，也按执行 context 映射为领域 timeout，不能泄漏成内部错误。
13. Trace raw 结果无条件投影 `timestamp`、`trace_id`、`span_id`。它们是 Trace Detail
    跳转和列表时间显示所需的 transport identity，不属于用户选择的展示列；前端不得把缺失值
    序列化为 `/trace/undefined` 或 `spanId=undefined`。
14. `isRoot` 与 `isEntryPoint` 是 V5 的语义 trace scope，不是 Catalog 字段，也不能由前端
    改写为物理列。编译器是唯一的展开点。Entrypoint 的具体定义已由 ADR-017 改为 OTel
    接收边界语义，不再依赖 Collector `top_level_operations` operation 目录。

## 影响

- Trace 列表在 child span 命中过滤时仍展示 root 信息，并能展示部分/orphan trace。
- 相同值但不同边界的 labels 不再合并，公式 schema 不兼容会提前失败。
- Time Series 图表不依赖 SQL 返回顺序，但现阶段不提供 top-N series。
- 行预算保护 SigInsight 进程；它不限制 ClickHouse 聚合中间状态。高基数 group-by 的
  ClickHouse 侧预算和真正 top-series 选择仍需后续独立设计。
- Trace Explorer 可隐藏 identity 列，但每个 raw 行仍可稳定打开对应 Trace Detail。
- Trace Detail 的 All/Root/Entrypoint 三种 scope 继续使用 V5 语义字段；它们不会查询
  不存在的 `isRoot`/`isEntryPoint` 物理列，且 Entrypoint 能高亮同一 trace 中所有 OTel
  Server/Consumer remote-parent 接收边界。

## 验证

- `pkg/litequery/compiler_test.go`：Trace CTE、scope 展开、Catalog table、typed IN 和排序字段。
- `pkg/litequery/compiler_clickhouse_integration_test.go`：ClickHouse 25.5.6 上 root、entrypoint、
  child、orphan trace 与 Metrics/Meter 实际执行。
- `pkg/litequery/executor_test.go`：formula 对齐/schema/顺序、结果行预算和 rows 关闭。
- `pkg/querier/liteadapter/adapter_test.go`：确定性 series、点排序、非有限数和 gap filling。
- `pkg/querier/litemetadata` 与 `liteadapter` 测试：范围前置校验、禁用查询在 metadata、能力
  解析、范围约束与 gap-fill step 选择中全部短路，intrinsic/aggregate alias 跳过和同名字段
  消歧。
- 前端 payload/component 测试：raw/trace 状态清理和按结果类型隐藏 Group By/Limit。
- `pkg/litequery/compiler_test.go` 与 Trace ListView 单测：raw trace identity 投影、V5
  snake-case identity 和详情链接。
- `tests/integration/scripts/run-litequery-collector-collaboration.sh`：2026-08-01 在 ClickHouse
  25.5.6 上运行当前 Collector migrations，经 OTLP 写入 Logs、Traces、Metrics、Meter，再由
  最新 SigInsight 的认证 V5 API 读回；四张物理表的 ClickHouse query log 均无错误。
