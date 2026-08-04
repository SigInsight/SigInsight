# ADR-014：字段元数据消歧契约

状态：Accepted
日期：2026-08-01
关联里程碑：M5、M6、M8

## 背景

V5 请求中的字段有两种携带方式：

1. 结构化字段（`selectFields`、`groupBy`、`order`）：携带完整的 `TelemetryFieldKey`，
   包含 field context（resource/attribute/span 等）与 data type。
2. 筛选表达式（filter `expression`）：以文本形式嵌入语法树，字段**可能只携带名称**
   （如 `host.name`），不携带 context 和 data type。

旧引擎在编译阶段通过 metadata store 对裸字段名做消歧：查询实际数据中的字段注册表，
推断 `host.name` 是 resource 上下文。轻量引擎早期的 `liteadapter` 没有接入该机制，
仅依赖硬编码的"常见 resource 字段白名单"（如 `service.name`），导致 `host.name`、
`k8s.*` 等未列入白名单的裸字段被错误拒绝：

```text
invalid lightweight query: log field "host.name" is not in the schema catalog
```

该错误不是单点漏项，而是适配器丢失了字段元数据消歧能力。逐个扩充白名单无法收敛。

## 决策

在 V5 兼容层（`pkg/querier/liteadapter`）与执行边界（`pkg/querier/litequery.go`、
`pkg/query-service/rules/litequery_runner.go`、`pkg/livelogs/handler.go`）恢复字段元数据
消歧，规则如下：

### 1. 消歧发生在应用边界，核心保持确定性

- `pkg/litequery` 核心**不依赖** metadata store；字段查询发生在
  `liteadapter.FieldKeySelectors` 与 `querier.liteMetadata` 边界。
- metadata 以纯数据（`MetricMetadata.FieldKeys map[string][]*TelemetryFieldKey`）
  传入适配器，适配器保持可测试、确定性。

### 2. 批量收集不完整字段

`FieldKeySelectors(request)` 在转换前遍历 V5 请求，收集所有 context 或 data type
未指定的字段：

- filter expression：通过 ANTLR lexer 提取 KEY token；
- `selectFields`、`groupBy`、`order`：结构化字段中信息不完整的；
- Logs aggregation 字段（`sum(x)`、`avg(x)` 等表达式的内层字段）。

收集结果去重后批量查询 metadata store（`GetKeysMulti`），返回
`FieldKeys map[name][]candidate` 供适配器消歧。

### 3. 消歧规则

在 `resolveFieldMetadata` 中按序执行：

| 优先级 | 规则 |
| --- | --- |
| 1 | **intrinsic 字段静态解析**：裸字段或显式 log/span 上下文命中信号固有字段表时，不查询 metadata；显式 data type 只用于校验 schema 类型，不能把 intrinsic 降级成动态属性 |
| 2 | **显式上下文/类型优先**：动态字段已指定 context 或 data type 时，metadata 只在该约束内匹配，不覆盖显式值 |
| 3 | **类型与 fallback 匹配**：未指定 data type 时，优先选择与操作符推断类型（fallback）一致的候选，避免数值操作误选只能存字符串的 resource map |
| 4 | **同类型裸名 resource 优先**：类型匹配后仍同时存在 resource 与 record/attribute 同名候选时，选择 resource（保持既有 V5 行为） |
| 5 | **存储类型约束**：resource map 只存字符串；metadata 未登记的裸 number/bool 字段可确定性解析为 attribute |
| 6 | **唯一候选才消歧**：过滤后仍存在多个候选时**不猜测**，保持原字段解析，由后续编译/校验给出明确错误 |

完成 metadata 查询后仍无法解析的裸字段不再降级为 log/span intrinsic 并产生误导性的
schema catalog 错误，而是要求调用者显式指定 `resource.` 或 `attribute.`。Live Logs 在
SSE 建连时完成一次相同的批量解析，后续轮询不重复查询 metadata。

### 4. 前端配合：结构化字段保留上下文

前端生成筛选表达式时保留字段的 `resource`/`tag` 上下文（`prepareQueryRangePayloadV5.ts`、
`utils.ts`），减少后端对结构化字段的猜测；文本筛选中的裸字段名仍由上述规则兜底。
Trace QuickFilters 中的固有列必须使用 `span` context，不能沿用旧 UI 将所有非 resource
字段标成 `tag` 的做法。Checkbox 候选值虽然以字符串显示，写回查询状态时必须按
`dataType` 恢复 number/bool；`IN/NOT IN` 空列表不构成谓词，不允许下发到 V5。

Query Builder 同时保留结构化 `filters.items` 和用于 URL/文本编辑的 `filter.expression`。
QuickFilters 每次修改后必须同步两者；否则 V5 serializer 会为保留手写表达式而选择旧的
`filter.expression`，表现为 UI 已更新但请求仍重复执行旧筛选。

QuickFilters 在状态变更时还必须把结构化 context 写入表达式：trace attribute、resource
和 intrinsic 分别序列化为 `attribute.http.route`、`resource.service.name` 和
`span.name`。不能依赖最终 V5 serializer 猜测或补写 context，因为混有历史/手写谓词时
`filter.expression` 是权威输入，serializer 会原样保留它。裸动态字段只在 telemetry
metadata 覆盖当前查询时间窗时才可能消歧，因此不是可靠的 UI 契约。

表达式合并以 canonical field identity 匹配结构化字段与文本谓词：例如结构化
`service.name`（resource context）和文本 `resource.service.name` 是同一字段，不能追加为
两条筛选。历史状态中若已经存在 canonical field、operator、value 都相同的重复项，
QuickFilters 和 V5 serializer 会安全去重，并将 scalar `IN/NOT IN` 重新写为列表语法；
context、operator 或 value 不同的谓词不会被静默合并。

## 备选方案

### 扩充硬编码白名单

不采用。白名单与真实数据字段必然漂移，持续漏项，且无法表达 resource/attribute
同名歧义。

### 让 litequery 核心依赖 metadata store

不采用。破坏 SQL 编译器的确定性、可测试性与 `go list -deps` 纯净性约束，违反
ADR-003/ADR-004 的分层边界。

### 拒绝所有无上下文字段

不采用。破坏既有保存查询与手写表达式兼容性，且与 V5 边界（ADR-002）冲突。

## 影响

- `liteadapter` 增加 metadata 输入参数（纯数据），`ToLite` 签名扩展为
  `ToLite(request, MetricMetadata)`。
- 告警执行器（`RuleQueryRunner`）与 UI 查询使用同一套字段解析规则。
- Live Logs 与普通 Logs 查询使用同一套字段解析规则。
- 歧义字段得到明确错误而非静默猜测，错误可归因、可测试。
- 无 metadata store 的测试环境通过注入构造的 `MetricMetadata` 保持确定。

## 迁移与回滚

- 该机制仅在 adapter 内部生效，不改变 Lite IR、不改变公共协议。
- SQLite migration `086` 将已有组织保存的 Trace QuickFilters 中固有列 context
  从旧的 `tag`（或空值）规范化为 `span`；动态 trace attributes 保持原 context。
  迁移可重复执行，并保留每个筛选项的其他字段。
- 回滚即恢复旧 adapter 行为（不带 metadata 的裸字段解析），不影响已结构化字段。

## 验证

- 契约测试覆盖：`host.name` 动态 resource 字段、动态 attribute 字段、
  resource/attribute 同名优先级、显式上下文覆盖、带类型 intrinsic 字段静态解析。
- Logs/Traces 常用 resource、attribute、intrinsic 字段的跨层（前端 payload →
  adapter → catalog）测试。
- QuickFilters 覆盖 span/resource/attribute context、bool/number 值恢复、空列表丢弃和
  双状态同步、qualified/unqualified key 合并及历史重复项恢复；已认证 V5 集成用例覆盖
  `attribute.http.route` 不依赖 metadata 的执行路径。

## 后续动作

- 观察真实查询中"歧义多候选"的频率，若持续出现，评估在 UI 层要求显式上下文。
