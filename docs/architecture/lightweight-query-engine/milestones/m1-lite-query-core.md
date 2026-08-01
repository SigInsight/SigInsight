# M1：Lite Query Core

状态：Complete
关联 ADR：001、002
前置条件：M0 capability matrix、schema baseline 和真实协作测试通过

## 问题与目标

现有 V5 `QueryBuilderQuery[T]` 同时承载多个 signal、SQL expression、Trace Operator、函数链和历史兼容字段。M1 要建立独立的领域模型，让后续 planner 和 compiler 只接收已验证、受约束的查询意图。

M1 不接管 HTTP handler，不生成 SQL，不修改 Collector schema。

## 范围

- `pkg/litequery`：不依赖 ClickHouse、HTTP、handler 或旧 querybuilder package。
- 时间范围、结果类型、signal、字段引用、过滤 AST、排序、分页、聚合和查询预算。
- Logs、Traces、Metrics、Meter 的类型化 query spec。
- 简单公式的引用、依赖图和静态校验。
- 可测试的 Logical Plan 结构和 planner contract。
- 领域错误及其稳定 code。

## 非目标

- V5 JSON decoder/encoder 和现有 Query Builder 的转换。
- SQL、表名、列名、ClickHouse client 和结果扫描。
- 动态字段元数据查询。
- 前端代码、Alert handler 或保存查询迁移。
- Trace Operator、join、sub-query、raw SQL 或复杂函数。

## 包边界

```text
pkg/litequery
  types.go       request, query, field, result and aggregation types
  filter.go      typed filter AST and operator validation
  validate.go    request/spec/capability/budget validation
  formula.go     formula tokenization and dependency validation
  plan.go        validated logical plans and planner interface
  errors.go      domain error codes; no HTTP response types
```

`pkg/litequery` 只能依赖 Go 标准库。它不应导入：

- `pkg/types/querybuildertypes/querybuildertypesv5`
- `pkg/telemetrylogs`、`pkg/telemetrytraces`、`pkg/telemetrymetrics`
- ClickHouse client
- `net/http` 或 handler/model response 类型

## 核心模型

```go
type Request struct {
    Range      TimeRange
    ResultType ResultType
    Queries    []Query
    Formulas   []Formula
    Format     FormatOptions
}

type Query interface {
    QueryName() string
    QuerySignal() Signal
    Common() CommonQuery
}

type CommonQuery struct {
    Name    string
    Filter  FilterNode
    Select  []FieldRef
    GroupBy []FieldRef
    Order   []Order
    Limit   uint32
    Cursor  string
}
```

具体 query 为 `LogQuery`、`TraceQuery`、`MetricQuery` 和 `MeterQuery`。每个 query 只有一个类型化 aggregation；多指标图表由多个命名查询和简单公式表达。禁止接受用户提供的任意 aggregation expression。

Filter 使用递归 AST：

```go
type FilterNode interface { isFilterNode() }

type LogicalFilter struct {
    Op    BooleanOperator
    Items []FilterNode
}

type Predicate struct {
    Field FieldRef
    Op    FilterOperator
    Value Value
}
```

`FieldRef` 只包含 semantic name、context 和 value type，不包含 ClickHouse 表名或 SQL expression。

## 校验顺序

1. 检查 request range、result type、query 名称和 query 数量。
2. 检查 signal 与 query spec 类型匹配。
3. 检查每类 aggregation、Metric type 和 temporality 的合法组合。
4. 检查 filter AST 深度、节点数、值类型和 operator。
5. 检查 group/order/limit/cursor 与 result type 的组合。
6. 检查 formula 引用、重复引用和依赖环。
7. 应用查询预算，输出领域错误而不是静默修改请求。

## 查询预算初始值

这些值必须通过显式 `Limits` 注入，不能散落为常量：

| 限制 | 初始值 |
| --- | ---: |
| 最大独立查询数 | 8 |
| 最大 formula 数 | 4 |
| 最大 filter AST 深度 | 8 |
| 最大 filter 节点数 | 64 |
| 最大 group by 字段数 | 4 |
| 最大 raw/trace limit | 1,000 |
| 最大 time-series point 数 | 11,000 |

执行器另外对每条 statement 默认限制 250,000 个扫描结果行。该值属于执行基础设施
保护，不是 `Limits` 中的查询语义；调用方可通过 `ExecutorConfig.MaxRows` 收紧。
Time Series 的非零 limit 在 top-series 两阶段计划实现前拒绝，不能直接截断 bucket 行。

时间范围和最小 step 属于 signal 相关规则，M1 只定义 contract，M2/M3 在 planner 中给出具体 policy。

## 错误模型

M1 返回普通 Go `error`，具体为：

```go
type Error struct {
    Code    ErrorCode
    Message string
    Field   string
}
```

初始 code：`invalid_request`、`unsupported`、`invalid_filter`、`invalid_aggregation`、`invalid_formula`、`budget_exceeded`。HTTP 状态映射属于 M4 handler adapter。

## 测试计划

- 每个 enum、metric type/operation 组合的表驱动测试。
- Filter AST 结构、类型、深度和节点数边界测试。
- query 名称、result type、group/order/limit 组合测试。
- formula 缺失引用、循环、重复名称、除零语义的静态测试。
- `go test -fuzz` 对 filter/formula parser 或构造器执行至少一次。
- dependency test：`go list -deps` 证明 core 不导入 HTTP、ClickHouse 或旧 V5 类型。

## 实现结果

- 已建立 `pkg/litequery` 的独立领域模型、Filter AST、预算、公式语法/依赖校验和 Logical Plan。
- `Plan` 已按 signal 分派 query，但尚未解析字段或生成 SQL。
- 包只依赖 Go 标准库；`go list -deps` 确认没有 ClickHouse、HTTP、旧 V5 DTO 或现有 signal compiler 依赖。

## 验证结果

2026-07-31 已执行：

```bash
go test ./pkg/litequery
go test -race ./pkg/litequery
go test -fuzz=FuzzValidateFormula -fuzztime=3s ./pkg/litequery
go list -deps ./pkg/litequery
yarn --cwd frontend refactor:baseline
```

- 单元测试和 race 检查通过。
- Formula fuzz 以 16 workers 执行约 1,154,336 次，没有崩溃或违反 validator contract。
- dependency scan 未发现 ClickHouse、HTTP、旧 V5 DTO 或现有 signal compiler import。
- Lightweight query engine：999 production Go LOC，1 个直接测试文件。

## 残余风险与后续任务

- IR 目前是 Go 领域模型，M2 前仍没有 V5 compatibility decoder；旧请求不能直接进入 Lite IR。
- Filter AST 还没有字段元数据和物理 schema capability 校验，这属于 M2 Schema Catalog。
- Formula 只做语法和依赖校验，时间序列标签对齐与求值属于 M4。

## 退出条件

- `pkg/litequery` 有独立、直接的单元测试，且不存在 SQL 字符串或 ClickHouse import。
- capability matrix 中的所有保留/拒绝能力可以由 validator 表达。
- M2 可以接收 validated logical plan 而不依赖旧 `QueryBuilderQuery`。
- M1 文档填写实现结果、测试证据、已知缺口和新的 LOC 数据。
