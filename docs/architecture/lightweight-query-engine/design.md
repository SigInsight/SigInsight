# 轻量查询引擎总体设计

状态：Accepted
最后更新：2026-08-01

## 1. 目标

基于当前 ClickHouse 遥测存储，设计一套面向 Logs、Traces、Metrics 和 Meter 的轻量查询语言与后端执行引擎，并使用同一份能力模型驱动前端查询 UI。

对外继续使用 `POST /api/v5/query_range` 和 V5 统一结果模型。兼容的是 HTTP envelope、核心请求语义和响应，不是现有 Query Builder 的全部高级能力。

项目最终必须证明：

1. 核心遥测 UI 和基本阈值告警可以完全由新引擎支撑。
2. 查询语义不依赖 ClickHouse 表名、列名或 SQL 字符串。
3. 前端无法构造后端不支持的查询组合。
4. 新实现比现有通用引擎更小、更容易测试，并能够明确拒绝超出边界的请求。
5. Collector schema、写入逻辑和查询引擎之间有可执行的兼容验证。

## 2. 非目标

- 不复制 SigNoz Query Builder 的全部表达能力。
- 不在第一版中兼容 Trace Operator。
- 不支持 `builder_join`、`builder_sub_query` 或用户输入 ClickHouse SQL。
- 不支持通用 secondary aggregation、通用 `limitBy` 或任意 Having DSL。
- 不支持 anomaly、EWMA、median、clamp、cutoff、runningDiff、cumulativeSum 等通用函数链。
- 不重新实现 PromQL。
- 不将 Trace Detail、service map、exceptions、retention、rule history 等专用 API 强行并入查询引擎。
- 不为尚未由真实查询证明的性能问题预先修改 ClickHouse schema。

## 3. 功能边界

### 3.1 Logs

第一版支持：

- 原始日志列表和 offset 分页；Live Logs 独立使用 typed `(timestamp, id)` cursor。
- 时间范围、正文、severity、service 和属性过滤。
- 普通字符串正文与 JSON path 查询。
- 字段选择、全局排序和全局限制。
- 按时间桶及属性分组的 count、sum、avg、min、max。
- time series、scalar 和 raw 结果。

Live Logs 暂时保留现有独立链路。是否纳入轻量引擎由后续 ADR 决定，不作为第一版切换阻塞项。

### 3.2 Traces

第一版支持：

- Trace 和 Span 列表及 offset 分页；V5 opaque cursor 明确拒绝。
- service、operation、status、duration、resource/span attribute 过滤。
- Trace Detail 的 All Spans、Root Spans、Entrypoint Spans scope。Root 是无 parent 的拓扑根；
  Entrypoint 是 Collector 物化的 root/cross-service operation 中的非根 span，适合高亮一个
  trace 内所有下游服务入口。
- 按 trace 聚合与返回匹配 trace 的基础查询。
- count、duration avg、p50、p90、p95、p99。
- time series、scalar、raw 和 trace 结果。

Trace Detail、瀑布图和 span tree 继续使用专用详情 API。删除 Trace Operator 后，不再支持跨 span 的父子或祖先关系查询，也不支持此类告警。

### 3.3 Metrics

只支持常见 OpenTelemetry metric 类型和操作：

| 类型 | 支持操作 |
| --- | --- |
| Gauge | latest、avg、min、max |
| Sum | sum、rate、increase |
| Histogram | p50、p90、p95、p99（显式 `.bucket`） |

支持 metric 发现、标签过滤、标签分组、time aggregation、space aggregation、time series 和 scalar 结果。

Metrics 内部允许固定的“时间聚合后按标签聚合”两阶段计划，但它是 `MetricPlan` 的明确语义，不对外暴露为通用 secondary aggregation。

### 3.4 Meter

Meter 是 Metrics 查询的一种明确 source，不建立第四套通用查询系统。

支持：

- Meter metric 发现。
- count、sum、avg、rate、increase。
- time aggregation、space aggregation、标签过滤与标签分组。
- 现有 Cost Meter 页面所需查询。

### 3.5 过滤与组合

公共 Filter AST 第一版支持：

- 单一 flat `AND` 链或单一 flat `OR` 链；混合优先级和括号分组明确拒绝。
- `=`、`!=`、`>`、`>=`、`<`、`<=`。
- `IN`、`NOT IN`、`EXISTS`、`NOT EXISTS`、`CONTAINS`、`NOT CONTAINS`。
- string 字段的 `LIKE`、`NOT LIKE`、`ILIKE`、`NOT ILIKE`、`REGEXP`、`NOT REGEXP`。
- 字符串、布尔、整数、浮点，以及同类型的字符串/数值/布尔数组值。

`CONTAINS` 是大小写不敏感的 literal substring，不解释 `%`/`_`；`LIKE/ILIKE` 使用
ClickHouse wildcard；`REGEXP` 使用 RE2-compatible pattern。pattern 最长 1,024 bytes，
regexp 在编译 SQL 前预校验，所有值仍通过参数传递。

### 3.5.1 Trace Funnel-compatible filter subset

Trace Funnel 的每个 step 可以复用公共 Filter AST 过滤当前 span 的 name、service、status、
duration、resource/span attributes 和错误状态。轻量引擎不负责 step sequencing、跨 span
join、ancestor/descendant、latency pointer 或 Funnel 专用多阶段规划。该边界详见 ADR-018。

禁止将 SQL 片段作为 filter、aggregation 或 order expression 传入。

聚合后过滤不保留任意 Having DSL，只允许一个受类型约束的谓词：

```go
type AggregationPredicate struct {
    Aggregation string
    Operator    ComparisonOperator
    Value       float64
}
```

### 3.6 公式与告警

保留命名查询间的 `+`、`-`、`*`、`/` 简单公式。公式在应用层执行，必须处理依赖环、时间点对齐、标签匹配、缺失值和除零。

基本告警支持：

- 单查询或简单算术公式。
- `last`、`avg`、`min`、`max`、`sum` reduce。
- 数值比较、持续时间和按标签生成告警实例。
- 查询预览和历史状态展示。

不支持 anomaly、预测、Trace Operator 或任意 ClickHouse SQL 告警。

### 3.7 结果和图表

第一版结果类型为：

- `raw`
- `trace`
- `time_series`
- `scalar`

V5 `formatOptions.fillGaps` 作为结果层补零规则保留，而不是通用函数。它按 epoch
bucket 对齐，并严格生成半开时间范围 `[start, end)` 内可能出现的 bucket；在返回 series
为空时也生成零值 series。`timeShift` 不进入第一版，是否
加入由使用数据和 ADR 决定。

前端只提供：

- 时间序列图。
- 表格。
- 数值面板。
- Trace 列表与现有 Trace Detail 跳转。

图表继续使用现有 V2 visualization，不再引入新的图表栈。

Raw/Trace 分页使用 `LIMIT pageSize + 1` 探测下一页，并在每个结果的 `pageInfo` 中返回
`hasNextPage`；分页窗口不属于结果截断，不产生 warning。只有不可分页的聚合结果上限或专用
读取总量预算导致语义结果不完整时，才返回 `result_limit_reached`。详细契约见 ADR-019。

## 4. 目标架构

```text
Lightweight Query UI
        |
POST /api/v5/query_range
        |
V5 compatibility decoder
        |
Lite Query IR
  |-- LogQuery
  |-- TraceQuery
  |-- MetricQuery
  |-- MeterQuery
  `-- ArithmeticFormula
        |
type validation + capability validation + query budget
        |
Logical Plan
        |
Signal Compiler
  |-- Logs Compiler
  |-- Traces Compiler
  `-- Metrics/Meter Compiler
        |
Schema Catalog
semantic field -> table/column/expression/capability
        |
Statement { SQL, Args, Metadata }
        |
ClickHouse 25.5.6
        |
V5 Result -> Frontend View Model
```

### 4.1 HTTP 兼容层

`/api/v5/query_range` 的路由和响应结构保持稳定。兼容层负责：

- 身份和组织上下文。
- JSON 解码和未知字段错误。
- 将受支持的 V5 请求子集恢复为 Lite Query IR。
- 对旧保存查询给出确定的“不支持”错误，不做静默降级。

迁移期允许内部 feature flag 或测试 header 选择新旧引擎，但该选择不成为公开查询协议的一部分。

### 4.2 Lite Query IR

IR 只表达查询意图，不包含 SQL、ClickHouse 表名或物理列名。公共结构保持最小化，信号差异由独立 spec 表达。

```go
type QueryRequest struct {
    Range      TimeRange
    ResultType ResultType
    Queries    []Query
    Formulas   []Formula
}

type CommonQuery struct {
    Name    string
    Filter  FilterNode
    GroupBy []FieldRef
    Order   []Order
    Limit   uint32
    Cursor  string
}
```

`LogQuery`、`TraceQuery` 和 `MetricQuery` 组合 `CommonQuery`，但分别拥有类型化 aggregation 和 select 定义。禁止重新建立包含所有信号可选字段的巨型通用 DTO。

#### 4.2.1 字段解析与元数据消歧

V5 请求中的字段有两种携带方式：

- **结构化字段**（`selectFields`、`groupBy`、`order`）：携带完整的
  `TelemetryFieldKey`，包含 field context 与 data type；
- **筛选表达式字段**（filter expression 文本）：可能**只携带名称**（如
  `host.name`），不携带 context 与 data type。

轻量引擎的字段解析遵循"**应用边界消歧、核心保持确定性**"（ADR-014）：

```text
V5 请求
  -> FieldKeySelectors：批量收集 context/data type 不完整的字段
       （filter 文本 token、select/group/order、日志聚合字段）
  -> metadata store 批量查询（GetKeysMulti）
  -> MetricMetadata.FieldKeys：以纯数据注入适配器
  -> resolveFieldMetadata 按规则消歧
  -> FieldRef { Name, Context, Type } 进入 Lite IR
```

消歧规则按序执行：

1. **intrinsic 静态解析**：裸字段或显式 log/span 字段命中信号 schema 时直接解析；
   `name:string` 等显式类型只校验 schema，不能触发 metadata 查找；
2. **显式上下文/类型优先**：动态字段已指定 context 或 data type 时，metadata 只在该
   约束内匹配，不覆盖显式值；
3. **类型与 fallback 匹配**：未指定 data type 时，先选择与操作符推断类型一致的候选；
4. **裸名 resource 优先**：类型匹配后仍有 resource 与 attribute 同名候选时，选择
   resource（保持既有 V5 行为）；
5. **存储类型约束**：resource map 只存字符串；metadata 未登记的裸 number/bool 字段
   可确定性解析为 attribute；
6. **唯一候选才消歧**：多候选歧义时不猜测，交由后续校验给出明确错误。

前端 QuickFilters 的契约同样是类型化的：Trace 固有列写为 span context，候选字符串在
写回时按字段 data type 恢复为 bool/number，空 `IN/NOT IN` 不序列化。结构化
`filters.items` 变化后必须同步 `filter.expression`，防止 URL 中的旧文本表达式覆盖最新
UI 状态。同步时直接生成 `span.*`、`resource.*`、`attribute.*` 限定名；V5 serializer
保留手写表达式的逻辑只是兼容边界，不能作为 QuickFilters 的 context 补全机制。
结构化字段和表达式字段通过 canonical identity 合并，使 `service.name`（resource
context）与 `resource.service.name` 更新同一谓词；执行前只删除 field、operator、value
完全等价的历史重复项。

metadata 前置解析只处理启用的 builder query，并在访问 store 前验证时间范围；禁用的旧
查询不能因为缺字段或指标 metadata 阻断当前请求。Catalog 在最终物理映射处再次强制
resource/scope string map 的类型约束，显式错误类型也不能绕过 schema 约定。

`pkg/litequery` 核心不依赖 metadata store；字段查询发生在 `querier` / rule runner /
live logs 边界，metadata 以纯数据传入，SQL 编译器保持确定性。完成 metadata 查询后
仍无法消歧的裸字段必须要求显式 `resource.` 或 `attribute.` 上下文，不能退回到
log/span intrinsic 猜测。

### 4.3 Logical Plan

Logical Plan 是经过字段解析和能力校验的内部计划，负责：

- 将用户字段解析成稳定的语义字段 ID。
- 选择 raw、trace、time series 或 scalar 计划。
- 计算合理 step 和查询预算。
- 为 Metric 明确 time aggregation 和 space aggregation 顺序。
- 为 Trace 明确 span 过滤、trace 归并和分页方式。

计划层仍不包含拼接后的 SQL。

### 4.4 Schema Catalog

Schema Catalog 是查询语义与 ClickHouse schema 的唯一接口层：

```go
type SemanticField struct {
    ID                 FieldID
    Signal             Signal
    Name               string
    Type               ValueType
    PhysicalExpression string
    FilterOperators    OperatorSet
    Selectable         bool
    Groupable          bool
}
```

Catalog 同时声明表、时间列、组织列和版本要求。Compiler 不得在 Catalog 之外散落字段到列的映射。

Metrics 的用户语义类型与物理类型允许不同。当前 metadata 将非单调 OTLP Sum 暴露为
Gauge；Catalog/Compiler 必须将该语义 Gauge 映射回原生 Gauge 或
`Sum + is_monotonic=false` series，并通过 fingerprint 选择 points。不得直接用语义 Gauge
生成 `type='Gauge' AND temporality='Unspecified'` 的物理过滤条件（ADR-015）。

任何 Collector schema 变化必须通过真实 ClickHouse 的 schema fingerprint 和 API 协作测试验证，不依赖两个仓库中的人工同步注释。

### 4.5 Compiler 与执行器

每个 signal compiler 只接受对应的已验证计划并返回参数化语句：

```go
type Statement struct {
    SQL      string
    Args     []any
    Metadata StatementMetadata
}
```

Compiler 负责 SQL 生成，不负责执行和结果格式化。Executor 负责超时、并发、扫描、统计、取消和 ClickHouse 错误映射。Result Mapper 负责转换为 V5 Result。

### 4.6 查询预算

当前实现限制：

- 每条 time series 的最大时间点数（11,000）。
- 每条 statement 的最大扫描结果行数（默认 250,000）。
- 最大 raw/trace limit。
- 最大 group by 数量。
- 最大 filter AST 深度和节点数。
- 单请求最大独立查询数。

预算拒绝必须返回稳定的领域错误和用户可操作的信息。

Time Series 的非零 `limit` 当前明确拒绝。ClickHouse 最终结果行是
`timestamp x group`，直接应用 `LIMIT` 会截断时间点而不是选择 top-N series。真正的
series limit 需要“先选择 series、再读取完整时间桶”的两阶段计划；在该计划实现前，
前端不展示 Time Series limit。`limit=0` 仍可能产生高基数 ClickHouse 中间结果，当前
serializer 也会清除视图切换遗留的非零 limit。statement 行预算只保护应用进程，不替代
ClickHouse 侧的扫描/内存预算。

## 5. ClickHouse schema 策略

第一版以当前生产 schema 为基线，不在编译器落地前修改表结构。

完成 Logs、Traces、Metrics/Meter 的真实查询后，依据 ClickHouse query log、扫描字节、延迟和 SQL 复杂度决定是否增加 projection、物化视图或新表。每项存储优化单独形成 ADR，并满足：

1. Collector 写入测试通过。
2. schema migration 在 ClickHouse 25.5.6 上通过。
3. Schema Catalog fingerprint 测试通过。
4. SigInsight 所有核心读取路径通过。
5. 旧字段或表的删除有生产代码引用扫描和真实 API 证据。

## 6. 迁移策略

1. 冻结现有核心页面请求与响应样本。
2. 新引擎在独立 package 中实现，不修改旧 compiler 行为。
3. 对受支持请求执行新旧双跑，比对结果和查询成本。
4. 新前端 UI 只产生 Lite IR 可表达的请求。
5. 将 Explorer、Dashboard、Services 和 Alert 消费者逐一切换。
6. 对旧保存查询进行可迁移子集转换；高级查询明确标记为不支持。
7. 生产 import 清零后删除旧实现和过渡适配器。

## 7. 成功指标

功能正确性：

- 核心 Logs、Traces、Metrics、Meter 和 threshold alert 场景有真实 API 测试。
- 所有生成 SQL 在 ClickHouse 25.5.6 执行。
- schema 变化后无未知表或未知列错误。

可维护性：

- 记录旧/新后端和前端 production LOC。
- 记录 package 数量、核心接口数量、最大文件 LOC 和主要分支数量。
- 新 IR、planner 和 compiler 具备独立直接测试。
- 删除旧 engine、复杂 DTO、函数链和对应 UI 后，生产 import 为零。

性能：

- 记录相同 fixture 的 p50/p95 查询延迟、read rows 和 read bytes。
- 新实现不得因轻量化产生未经说明的数量级性能退化。

## 8. 设计约束

1. 用户模型不包含 ClickHouse SQL 或物理字段。
2. API 解码、IR 校验、计划、编译、执行和结果转换保持独立。
3. 前后端共享的是协议和能力矩阵，不共享运行时代码。
4. 不支持的能力在 UI 隐藏、在 API 明确拒绝。
5. schema 改动由测量驱动，并能通过跨仓库协作测试复现。
6. 每个重构提交必须实现功能、解除删除阻塞、实际删除代码，或提供证明删除安全的测试。
