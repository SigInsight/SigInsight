package litequery

import "time"

type Signal string

const (
	SignalLogs    Signal = "logs"
	SignalTraces  Signal = "traces"
	SignalMetrics Signal = "metrics"
	SignalMeter   Signal = "meter"
)

func (s Signal) valid() bool {
	switch s {
	case SignalLogs, SignalTraces, SignalMetrics, SignalMeter:
		return true
	default:
		return false
	}
}

type ResultType string

const (
	ResultRaw        ResultType = "raw"
	ResultTrace      ResultType = "trace"
	ResultTimeSeries ResultType = "time_series"
	ResultScalar     ResultType = "scalar"
)

func (r ResultType) valid() bool {
	switch r {
	case ResultRaw, ResultTrace, ResultTimeSeries, ResultScalar:
		return true
	default:
		return false
	}
}

type TimeRange struct {
	StartMS int64
	EndMS   int64
}

func (r TimeRange) Duration() time.Duration {
	return time.Duration(r.EndMS-r.StartMS) * time.Millisecond
}

type FormatOptions struct {
	FillGaps bool
	FillZero bool
}

// Request is the storage-independent input to a lightweight query plan.
// Queries use one typed aggregation each; callers compose results through
// named queries and simple formulas instead of multi-aggregation DTOs.
type Request struct {
	Range      TimeRange
	ResultType ResultType
	StepMS     int64
	Queries    []Query
	Formulas   []Formula
	Format     FormatOptions
}

type FieldContext string

const (
	FieldContextResource  FieldContext = "resource"
	FieldContextAttribute FieldContext = "attribute"
	FieldContextSpan      FieldContext = "span"
	FieldContextLog       FieldContext = "log"
	FieldContextBody      FieldContext = "body"
	FieldContextScope     FieldContext = "scope"
	FieldContextMetric    FieldContext = "metric"
	FieldContextLabel     FieldContext = "label"
)

func (c FieldContext) valid() bool {
	switch c {
	case FieldContextResource, FieldContextAttribute, FieldContextSpan, FieldContextLog,
		FieldContextBody, FieldContextScope, FieldContextMetric, FieldContextLabel:
		return true
	default:
		return false
	}
}

type ValueType string

const (
	ValueTypeString ValueType = "string"
	ValueTypeNumber ValueType = "number"
	ValueTypeBool   ValueType = "bool"
)

func (t ValueType) valid() bool {
	return t == ValueTypeString || t == ValueTypeNumber || t == ValueTypeBool
}

// FieldRef is a semantic field reference. Catalog resolution in M2 maps it to
// a physical ClickHouse expression.
type FieldRef struct {
	Name    string
	Context FieldContext
	Type    ValueType
}

type SortDirection string

const (
	SortAscending  SortDirection = "asc"
	SortDescending SortDirection = "desc"
)

type OrderTarget string

const (
	OrderByField       OrderTarget = "field"
	OrderByAggregation OrderTarget = "aggregation"
)

type Order struct {
	Target    OrderTarget
	Field     FieldRef
	Direction SortDirection
}

// AggregationPredicate is the constrained replacement for arbitrary HAVING.
// It always applies to the single aggregation of its owning query.
type AggregationPredicate struct {
	Operator ComparisonOperator
	Value    float64
}

type ComparisonOperator string

const (
	CompareEqual              ComparisonOperator = "eq"
	CompareNotEqual           ComparisonOperator = "neq"
	CompareGreaterThan        ComparisonOperator = "gt"
	CompareGreaterThanOrEqual ComparisonOperator = "gte"
	CompareLessThan           ComparisonOperator = "lt"
	CompareLessThanOrEqual    ComparisonOperator = "lte"
)

func (o ComparisonOperator) valid() bool {
	switch o {
	case CompareEqual, CompareNotEqual, CompareGreaterThan, CompareGreaterThanOrEqual,
		CompareLessThan, CompareLessThanOrEqual:
		return true
	default:
		return false
	}
}

type CommonQuery struct {
	Name    string
	Filter  FilterNode
	Select  []FieldRef
	GroupBy []FieldRef
	Order   []Order
	Limit   uint32
	Cursor  string
	// After is a typed raw-log cursor. It is deliberately separate from the
	// V5-compatible opaque Cursor string: raw log readers need a stable,
	// lexicographic storage position rather than offset pagination.
	After     *RawLogCursor
	Predicate *AggregationPredicate
}

// RawLogCursor identifies the last emitted log in ClickHouse's
// (timestamp, id) ordering. TimestampNS is the physical UInt64 timestamp.
type RawLogCursor struct {
	TimestampNS uint64
	ID          string
}

// Query is intentionally closed to the signal-specific specs in this package.
// This keeps later compilers exhaustive when dispatching by concrete type.
type Query interface {
	query()
	GetCommon() CommonQuery
	QuerySignal() Signal
}

type LogAggregation string

const (
	LogAggregateCount LogAggregation = "count"
	LogAggregateSum   LogAggregation = "sum"
	LogAggregateAvg   LogAggregation = "avg"
	LogAggregateMin   LogAggregation = "min"
	LogAggregateMax   LogAggregation = "max"
)

type LogQuery struct {
	Common      CommonQuery
	Aggregation LogAggregation
	Field       FieldRef
}

func (LogQuery) query()                   {}
func (q LogQuery) GetCommon() CommonQuery { return q.Common }
func (LogQuery) QuerySignal() Signal      { return SignalLogs }

type TraceAggregation string

const (
	TraceAggregateCount       TraceAggregation = "count"
	TraceAggregateDurationAvg TraceAggregation = "duration_avg"
	TraceAggregateDurationP50 TraceAggregation = "duration_p50"
	TraceAggregateDurationP90 TraceAggregation = "duration_p90"
	TraceAggregateDurationP95 TraceAggregation = "duration_p95"
	TraceAggregateDurationP99 TraceAggregation = "duration_p99"
)

type TraceQuery struct {
	Common      CommonQuery
	Aggregation TraceAggregation
}

func (TraceQuery) query()                   {}
func (q TraceQuery) GetCommon() CommonQuery { return q.Common }
func (TraceQuery) QuerySignal() Signal      { return SignalTraces }

type MetricType string

const (
	MetricGauge     MetricType = "gauge"
	MetricSum       MetricType = "sum"
	MetricHistogram MetricType = "histogram"
)

type Temporality string

const (
	TemporalityUnspecified Temporality = "unspecified"
	TemporalityDelta       Temporality = "delta"
	TemporalityCumulative  Temporality = "cumulative"
)

type TimeAggregation string

const (
	TimeAggregateLatest   TimeAggregation = "latest"
	TimeAggregateSum      TimeAggregation = "sum"
	TimeAggregateAvg      TimeAggregation = "avg"
	TimeAggregateMin      TimeAggregation = "min"
	TimeAggregateMax      TimeAggregation = "max"
	TimeAggregateCount    TimeAggregation = "count"
	TimeAggregateRate     TimeAggregation = "rate"
	TimeAggregateIncrease TimeAggregation = "increase"
)

type SpaceAggregation string

const (
	SpaceAggregateSum   SpaceAggregation = "sum"
	SpaceAggregateAvg   SpaceAggregation = "avg"
	SpaceAggregateMin   SpaceAggregation = "min"
	SpaceAggregateMax   SpaceAggregation = "max"
	SpaceAggregateCount SpaceAggregation = "count"
	SpaceAggregateP50   SpaceAggregation = "p50"
	SpaceAggregateP90   SpaceAggregation = "p90"
	SpaceAggregateP95   SpaceAggregation = "p95"
	SpaceAggregateP99   SpaceAggregation = "p99"
)

type MetricAggregation struct {
	MetricName       string
	Type             MetricType
	Temporality      Temporality
	TimeAggregation  TimeAggregation
	SpaceAggregation SpaceAggregation
}

type MetricQuery struct {
	Common      CommonQuery
	Aggregation MetricAggregation
}

func (MetricQuery) query()                   {}
func (q MetricQuery) GetCommon() CommonQuery { return q.Common }
func (MetricQuery) QuerySignal() Signal      { return SignalMetrics }

type MeterQuery struct {
	Common      CommonQuery
	Aggregation MetricAggregation
}

func (MeterQuery) query()                   {}
func (q MeterQuery) GetCommon() CommonQuery { return q.Common }
func (MeterQuery) QuerySignal() Signal      { return SignalMeter }

type Formula struct {
	Name       string
	Expression string
}
