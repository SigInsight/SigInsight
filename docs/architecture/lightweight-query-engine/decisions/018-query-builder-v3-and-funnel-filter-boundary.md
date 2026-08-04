# ADR-018：QueryBuilderSearchV3 与 Trace Funnel 过滤边界

状态：Accepted
日期：2026-08-02

## 背景

系统曾同时维护 chip 型 `QueryBuilderSearch`、`QueryBuilderSearchV2`、自由文本
`QuerySearch`、ClientSideQBSearch 和轻量文本框。它们分别维护字段限定、operator、值类型和
校验规则，导致同一个 `http.route` 在不同页面可能成为裸字段、resource 字段或 span
attribute。前端还能补全后端明确拒绝的能力。

Trace Funnel 的 Where 条件需要字符串模式匹配，但完整 Funnel 还包含步骤顺序、跨 span
关系和 latency pointer。把这些能力加入公共 Lite Filter 会重新引入通用 join planner，违背
轻量引擎的确定性目标。

## 决策

1. `QueryBuilderSearchV3` 是主要查询输入组件。它始终编辑一条文本表达式，不把已输入条件
   变成 chip/tag。
2. V3 的字段、operator、值和 AND/OR 补全只展示能力矩阵允许的集合。完整表达式通过
   `parseLiteFilterExpression` 后才更新结构化 `TagFilter`；不完整或非法草稿只保存在编辑器。
3. 候选项必须插入限定字段名。少量稳定 OTel 语义映射优先：
   `http.route -> attribute.http.route`、`service.name -> resource.service.name`、
   `host.name -> resource.host.name`；其余字段使用服务端 metadata context。Trace/Log intrinsic
   分别使用 `span.*`/`log.*`。这样旧 metadata 中同名的 materialized/span catalog 项不会制造
   两个不同限定名。
4. 显示层限定名和结构化字段分离。例如 `attribute.http.route` 解析为
   `{key: "http.route", type: "attribute", dataType: "string"}`，不能把前缀留在 raw key 中。
5. literal 由语法决定类型：`true` 是 bool，`'true'` 是 string，不做字符串布尔强转。operator
   候选按字段类型收窄；`isRoot`、`isEntryPoint` 只允许 `= true`。
6. 公共字符串谓词增加 `LIKE`、`ILIKE`、`REGEXP`、对应 `NOT` 形式和 `NOT CONTAINS`。
   只允许 string 字段，pattern 最长 1,024 bytes；regexp 在 Go 中先按 RE2 语法编译；SQL
   始终使用 positional args。`CONTAINS` 保持不含 wildcard 的大小写不敏感子串语义。
7. Trace Funnel 只复用上述“单步骤、单 span 的字段谓词”。每个 step 可过滤 span name、
   service、status、duration、resource/span attributes 和错误状态；多个 step 的业务编排仍由
   Funnel 专用模块负责。
8. Lite 引擎不支持 Funnel step sequencing、ancestor/descendant、跨 span join、latency
   pointer、Trace Operator 或 Funnel 专用多阶段 planner。需要这些能力时 API 必须明确拒绝，
   不能把多步语义降级为独立过滤。
9. 删除旧编辑器前先迁移全部生产消费者，并将仍被其他控件使用的 scope/helper 提取到独立
   模块。不得保留旧路径兼容别名。

## 影响

- 前端和后端共享同一轻量能力边界，但仍通过 V5 `TagFilter`/filter expression 契约协作，
  不共享 TypeScript/Go 代码。
- 用户输入裸 `http.route` 仍可由 metadata 边界消歧；通过补全选择时总是得到确定的
  `attribute.http.route`。
- 本实现称为“Trace Funnel-compatible filter subset”，不是完整 Trace Funnel 查询引擎。
- flat AND 或 flat OR 仍是第一版表达式形状；混合 AND/OR 和括号分组继续明确拒绝。

## 验证

- completion 纯函数测试覆盖字段限定、光标阶段、类型化 operator 和 literal escaping。
- parser 测试覆盖全部字符串谓词、bool/string 区分和 field metadata round-trip。
- compiler 测试覆盖参数化 SQL、存在性语义、非法/超长 regexp。
- ClickHouse 25.5.6 集成测试执行 ILIKE、REGEXP 和 NOT LIKE。
- 前端类型检查、生产 build 和主要消费者测试必须在删除旧组件后通过。
