package litequery

import (
	"math"
	"testing"

	"github.com/SigNoz/signoz/pkg/errors"
)

func TestValidateSupportedRequests(t *testing.T) {
	service := FieldRef{Name: "service.name", Context: FieldContextResource, Type: ValueTypeString}
	tests := []struct {
		name string
		req  Request
	}{
		{
			name: "logs raw with body filter",
			req: Request{
				Range: TimeRange{StartMS: 1, EndMS: 10}, ResultType: ResultRaw,
				Queries: []Query{LogQuery{Common: CommonQuery{Name: "logs", Select: []FieldRef{{Name: "body", Context: FieldContextBody, Type: ValueTypeString}}, Filter: Predicate{Field: FieldRef{Name: "body", Context: FieldContextBody, Type: ValueTypeString}, Op: FilterContains, Value: Value{Kind: ValueString, String: "error"}}, Limit: 100}, Aggregation: LogAggregateCount}},
			},
		},
		{
			name: "trace time series",
			req: Request{
				Range: TimeRange{StartMS: 1, EndMS: 60_001}, ResultType: ResultTimeSeries, StepMS: 1_000,
				Queries: []Query{TraceQuery{Common: CommonQuery{Name: "traces", GroupBy: []FieldRef{service}}, Aggregation: TraceAggregateDurationP95}},
			},
		},
		{
			name: "gauge",
			req: Request{
				Range: TimeRange{StartMS: 1, EndMS: 60_001}, ResultType: ResultTimeSeries, StepMS: 1_000,
				Queries: []Query{MetricQuery{Common: CommonQuery{Name: "cpu"}, Aggregation: MetricAggregation{MetricName: "system.cpu.utilization", Type: MetricGauge, Temporality: TemporalityUnspecified, TimeAggregation: TimeAggregateLatest, SpaceAggregation: SpaceAggregateAvg}}},
			},
		},
		{
			name: "cumulative sum rate",
			req: Request{
				Range: TimeRange{StartMS: 1, EndMS: 60_001}, ResultType: ResultTimeSeries, StepMS: 1_000,
				Queries: []Query{MetricQuery{Common: CommonQuery{Name: "requests"}, Aggregation: MetricAggregation{MetricName: "http.server.request.count", Type: MetricSum, Temporality: TemporalityCumulative, TimeAggregation: TimeAggregateRate, SpaceAggregation: SpaceAggregateSum}}},
			},
		},
		{
			name: "histogram percentile",
			req: Request{
				Range: TimeRange{StartMS: 1, EndMS: 60_001}, ResultType: ResultTimeSeries, StepMS: 1_000,
				Queries: []Query{MetricQuery{Common: CommonQuery{Name: "latency"}, Aggregation: MetricAggregation{MetricName: "http.server.duration", Type: MetricHistogram, Temporality: TemporalityUnspecified, TimeAggregation: TimeAggregateCount, SpaceAggregation: SpaceAggregateP95}}},
			},
		},
		{
			name: "meter",
			req: Request{
				Range: TimeRange{StartMS: 1, EndMS: 60_001}, ResultType: ResultTimeSeries, StepMS: 1_000,
				Queries: []Query{MeterQuery{Common: CommonQuery{Name: "cost"}, Aggregation: MetricAggregation{MetricName: "signoz.meter.log.size", Type: MetricSum, Temporality: TemporalityDelta, TimeAggregation: TimeAggregateSum, SpaceAggregation: SpaceAggregateSum}}},
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if err := Validate(test.req, DefaultLimits()); err != nil {
				t.Fatalf("Validate() error = %v", err)
			}
		})
	}
}

func TestValidateRejectsUnsupportedOrInvalidRequests(t *testing.T) {
	stringField := FieldRef{Name: "service.name", Context: FieldContextResource, Type: ValueTypeString}
	validLog := LogQuery{Common: CommonQuery{Name: "A"}, Aggregation: LogAggregateCount}
	tests := []struct {
		name string
		req  Request
		code ErrorCode
	}{
		{
			name: "metric raw result", code: ErrorInvalidRequest,
			req: Request{Range: TimeRange{StartMS: 1, EndMS: 2}, ResultType: ResultRaw, Queries: []Query{MetricQuery{Common: CommonQuery{Name: "M"}, Aggregation: MetricAggregation{MetricName: "g", Type: MetricGauge, Temporality: TemporalityUnspecified, TimeAggregation: TimeAggregateLatest, SpaceAggregation: SpaceAggregateAvg}}}},
		},
		{
			name: "histogram rate", code: ErrorInvalidAggregation,
			req: Request{Range: TimeRange{StartMS: 1, EndMS: 2}, ResultType: ResultScalar, Queries: []Query{MetricQuery{Common: CommonQuery{Name: "H"}, Aggregation: MetricAggregation{MetricName: "h", Type: MetricHistogram, TimeAggregation: TimeAggregateRate, SpaceAggregation: SpaceAggregateP95}}}},
		},
		{
			name: "meter gauge", code: ErrorInvalidAggregation,
			req: Request{Range: TimeRange{StartMS: 1, EndMS: 2}, ResultType: ResultScalar, Queries: []Query{MeterQuery{Common: CommonQuery{Name: "Meter"}, Aggregation: MetricAggregation{MetricName: "m", Type: MetricGauge, TimeAggregation: TimeAggregateLatest, SpaceAggregation: SpaceAggregateAvg}}}},
		},
		{
			name: "too deep filter", code: ErrorBudgetExceeded,
			req: Request{Range: TimeRange{StartMS: 1, EndMS: 2}, ResultType: ResultRaw, Queries: []Query{LogQuery{Common: CommonQuery{Name: "L", Filter: nestedFilter(stringField, 9)}, Aggregation: LogAggregateCount}}},
		},
		{
			name: "unknown formula reference", code: ErrorInvalidFormula,
			req: Request{Range: TimeRange{StartMS: 1, EndMS: 2}, ResultType: ResultScalar, Queries: []Query{validLog}, Formulas: []Formula{{Name: "F", Expression: "A + Missing"}}},
		},
		{
			name: "formula dependency cycle", code: ErrorInvalidFormula,
			req: Request{Range: TimeRange{StartMS: 1, EndMS: 2}, ResultType: ResultScalar, Queries: []Query{validLog}, Formulas: []Formula{{Name: "F", Expression: "G + A"}, {Name: "G", Expression: "F + A"}}},
		},
		{
			name: "query name starts with a letter", code: ErrorInvalidRequest,
			req: Request{Range: TimeRange{StartMS: 1, EndMS: 2}, ResultType: ResultScalar, Queries: []Query{LogQuery{Common: CommonQuery{Name: "_A"}, Aggregation: LogAggregateCount}}},
		},
		{
			name: "aggregation predicate needs aggregate result", code: ErrorInvalidRequest,
			req: Request{Range: TimeRange{StartMS: 1, EndMS: 2}, ResultType: ResultRaw, Queries: []Query{LogQuery{Common: CommonQuery{Name: "A", Predicate: &AggregationPredicate{Operator: CompareGreaterThan, Value: 1}}, Aggregation: LogAggregateCount}}},
		},
		{
			name: "filter rejects non finite number", code: ErrorInvalidFilter,
			req: Request{Range: TimeRange{StartMS: 1, EndMS: 2}, ResultType: ResultRaw, Queries: []Query{LogQuery{Common: CommonQuery{Name: "A", Filter: Predicate{Field: FieldRef{Name: "duration", Context: FieldContextLog, Type: ValueTypeNumber}, Op: FilterGreaterThan, Value: Value{Kind: ValueNumber, Number: math.NaN()}}}, Aggregation: LogAggregateCount}}},
		},
		{
			name: "time series rejects offset", code: ErrorInvalidRequest,
			req: Request{Range: TimeRange{StartMS: 1, EndMS: 2}, ResultType: ResultTimeSeries, StepMS: 1, Queries: []Query{LogQuery{Common: CommonQuery{Name: "A", Offset: 1}, Aggregation: LogAggregateCount}}},
		},
		{
			name: "literal division by zero", code: ErrorInvalidFormula,
			req: Request{Range: TimeRange{StartMS: 1, EndMS: 2}, ResultType: ResultScalar, Queries: []Query{validLog}, Formulas: []Formula{{Name: "F", Expression: "A / 0"}}},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			err := Validate(test.req, DefaultLimits())
			var queryErr *Error
			if !errors.As(err, &queryErr) || queryErr.Code != test.code {
				t.Fatalf("Validate() error = %v, want error code %q", err, test.code)
			}
		})
	}
}

func TestValidateFormulaAllowsForwardReference(t *testing.T) {
	req := Request{
		Range: TimeRange{StartMS: 1, EndMS: 2}, ResultType: ResultScalar,
		Queries: []Query{LogQuery{Common: CommonQuery{Name: "A"}, Aggregation: LogAggregateCount}},
		Formulas: []Formula{
			{Name: "F", Expression: "G + A"},
			{Name: "G", Expression: "A * 2"},
		},
	}
	if err := Validate(req, DefaultLimits()); err != nil {
		t.Fatalf("Validate() error = %v", err)
	}
}

func TestDefaultPlanner(t *testing.T) {
	req := Request{
		Range: TimeRange{StartMS: 1, EndMS: 2}, ResultType: ResultScalar,
		Queries: []Query{LogQuery{Common: CommonQuery{Name: "A"}, Aggregation: LogAggregateCount}},
	}
	plan, err := (DefaultPlanner{}).Plan(req)
	if err != nil {
		t.Fatalf("Plan() error = %v", err)
	}
	if got := plan.Queries[0].Query.GetCommon().Name; got != "A" {
		t.Fatalf("Plan() query name = %q, want A", got)
	}
	if got := plan.Queries[0].Signal; got != SignalLogs {
		t.Fatalf("Plan() signal = %q, want %q", got, SignalLogs)
	}
}

func nestedFilter(field FieldRef, depth int) FilterNode {
	var result FilterNode = Predicate{Field: field, Op: FilterEqual, Value: Value{Kind: ValueString, String: "api"}}
	for range depth {
		result = LogicalFilter{Operator: BooleanAnd, Items: []FilterNode{result, Predicate{Field: field, Op: FilterEqual, Value: Value{Kind: ValueString, String: "api"}}}}
	}
	return result
}

func FuzzValidateFormula(f *testing.F) {
	f.Add("A + B")
	f.Add("A / 0")
	f.Add("A + (B * 2)")
	f.Fuzz(func(t *testing.T, expression string) {
		req := Request{
			Range: TimeRange{StartMS: 1, EndMS: 2}, ResultType: ResultScalar,
			Queries:  []Query{LogQuery{Common: CommonQuery{Name: "A"}, Aggregation: LogAggregateCount}, LogQuery{Common: CommonQuery{Name: "B"}, Aggregation: LogAggregateCount}},
			Formulas: []Formula{{Name: "F", Expression: expression}},
		}
		_ = Validate(req, DefaultLimits())
	})
}
