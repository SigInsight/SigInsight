package liteadapter

import (
	"testing"
	"time"

	"github.com/SigNoz/signoz/pkg/errors"
	"github.com/SigNoz/signoz/pkg/litequery"
	"github.com/SigNoz/signoz/pkg/types/metrictypes"
	qbtypes "github.com/SigNoz/signoz/pkg/types/querybuildertypes/querybuildertypesv5"
	"github.com/SigNoz/signoz/pkg/types/telemetrytypes"
	"github.com/SigNoz/signoz/pkg/valuer"
)

func TestToLiteConvertsStructuredLogFilter(t *testing.T) {
	request := &qbtypes.QueryRangeRequest{
		Start: 1_000, End: 61_000, RequestType: qbtypes.RequestTypeTimeSeries,
		CompositeQuery: qbtypes.CompositeQuery{Queries: []qbtypes.QueryEnvelope{{
			Type: qbtypes.QueryTypeBuilder,
			Spec: qbtypes.QueryBuilderQuery[qbtypes.LogAggregation]{
				Name: "A", Signal: telemetrytypes.SignalLogs, StepInterval: qbtypes.Step{Duration: time.Minute},
				Aggregations: []qbtypes.LogAggregation{{Expression: "count()"}},
				Filter:       &qbtypes.Filter{Expression: "resource.service.name = 'api' AND severity_text = 'ERROR'"},
				GroupBy:      []qbtypes.GroupByKey{{TelemetryFieldKey: telemetrytypes.TelemetryFieldKey{Name: "service.name", FieldContext: telemetrytypes.FieldContextResource, FieldDataType: telemetrytypes.FieldDataTypeString}}},
			},
		}}},
	}

	converted, err := ToLite(request, MetricMetadata{})
	if err != nil {
		t.Fatalf("ToLite() error = %v", err)
	}
	if converted.ResultType != litequery.ResultTimeSeries || converted.StepMS != 60_000 || len(converted.Queries) != 1 {
		t.Fatalf("ToLite() = %#v", converted)
	}
	query, ok := converted.Queries[0].(litequery.LogQuery)
	if !ok {
		t.Fatalf("query type = %T, want LogQuery", converted.Queries[0])
	}
	if query.Aggregation != litequery.LogAggregateCount || len(query.Common.GroupBy) != 1 {
		t.Fatalf("query = %#v", query)
	}
	filter, ok := query.Common.Filter.(litequery.LogicalFilter)
	if !ok || filter.Operator != litequery.BooleanAnd || len(filter.Items) != 2 {
		t.Fatalf("filter = %#v", query.Common.Filter)
	}
}

func TestToLiteInfersUnqualifiedServiceNameAsTraceResourceField(t *testing.T) {
	request := &qbtypes.QueryRangeRequest{
		Start: 1_000, End: 61_000, RequestType: qbtypes.RequestTypeTimeSeries,
		CompositeQuery: qbtypes.CompositeQuery{Queries: []qbtypes.QueryEnvelope{{
			Type: qbtypes.QueryTypeBuilder,
			Spec: qbtypes.QueryBuilderQuery[qbtypes.TraceAggregation]{
				Name: "A", Signal: telemetrytypes.SignalTraces, StepInterval: qbtypes.Step{Duration: time.Minute},
				Aggregations: []qbtypes.TraceAggregation{{Expression: "count()"}},
				Filter:       &qbtypes.Filter{Expression: "service.name = 'api'"},
			},
		}}},
	}

	converted, err := ToLite(request, MetricMetadata{})
	if err != nil {
		t.Fatalf("ToLite() error = %v", err)
	}
	query, ok := converted.Queries[0].(litequery.TraceQuery)
	if !ok {
		t.Fatalf("query type = %T, want TraceQuery", converted.Queries[0])
	}
	predicate, ok := query.Common.Filter.(litequery.Predicate)
	if !ok {
		t.Fatalf("filter = %#v, want Predicate", query.Common.Filter)
	}
	if predicate.Field != (litequery.FieldRef{Name: "service.name", Context: litequery.FieldContextResource, Type: litequery.ValueTypeString}) {
		t.Fatalf("filter field = %#v, want resource service.name", predicate.Field)
	}
	plan, err := (litequery.DefaultPlanner{}).Plan(converted)
	if err != nil {
		t.Fatalf("Plan() error = %v", err)
	}
	if _, err := litequery.NewCompiler(nil).Compile(plan); err != nil {
		t.Fatalf("Compile() error = %v", err)
	}
}

func TestToLitePreservesRawLogOffset(t *testing.T) {
	request := &qbtypes.QueryRangeRequest{
		Start: 1_000, End: 61_000, RequestType: qbtypes.RequestTypeRaw,
		CompositeQuery: qbtypes.CompositeQuery{Queries: []qbtypes.QueryEnvelope{{
			Type: qbtypes.QueryTypeBuilder,
			Spec: qbtypes.QueryBuilderQuery[qbtypes.LogAggregation]{
				Name: "A", Signal: telemetrytypes.SignalLogs, Offset: 100,
				Aggregations: []qbtypes.LogAggregation{{Expression: "count()"}},
			},
		}}},
	}
	converted, err := ToLite(request, MetricMetadata{})
	if err != nil {
		t.Fatalf("ToLite() error = %v", err)
	}
	query, ok := converted.Queries[0].(litequery.LogQuery)
	if !ok || query.Common.Offset != 100 {
		t.Fatalf("query = %#v, want raw log offset 100", converted.Queries[0])
	}
}

func TestToLiteUsesMetricMetadata(t *testing.T) {
	request := &qbtypes.QueryRangeRequest{
		Start: 1_000, End: 61_000, RequestType: qbtypes.RequestTypeTimeSeries,
		CompositeQuery: qbtypes.CompositeQuery{Queries: []qbtypes.QueryEnvelope{{
			Type: qbtypes.QueryTypeBuilder,
			Spec: qbtypes.QueryBuilderQuery[qbtypes.MetricAggregation]{
				Name: "A", Signal: telemetrytypes.SignalMetrics, StepInterval: qbtypes.Step{Duration: time.Minute},
				Aggregations: []qbtypes.MetricAggregation{{MetricName: "http.server.requests", TimeAggregation: metrictypes.TimeAggregationRate, SpaceAggregation: metrictypes.SpaceAggregationSum}},
			},
		}}},
	}
	converted, err := ToLite(request, MetricMetadata{
		Temporality: map[string]metrictypes.Temporality{"http.server.requests": metrictypes.Cumulative},
		Types:       map[string]metrictypes.Type{"http.server.requests": metrictypes.SumType},
	})
	if err != nil {
		t.Fatalf("ToLite() error = %v", err)
	}
	query, ok := converted.Queries[0].(litequery.MetricQuery)
	if !ok {
		t.Fatalf("query type = %T, want MetricQuery", converted.Queries[0])
	}
	if query.Aggregation.Type != litequery.MetricSum || query.Aggregation.Temporality != litequery.TemporalityCumulative {
		t.Fatalf("aggregation = %#v", query.Aggregation)
	}
}

func TestToLiteDoesNotApplyMetadataTemporalityToHistogram(t *testing.T) {
	request := &qbtypes.QueryRangeRequest{
		Start: 1_000, End: 61_000, RequestType: qbtypes.RequestTypeTimeSeries,
		CompositeQuery: qbtypes.CompositeQuery{Queries: []qbtypes.QueryEnvelope{{
			Type: qbtypes.QueryTypeBuilder,
			Spec: qbtypes.QueryBuilderQuery[qbtypes.MetricAggregation]{
				Name: "A", Signal: telemetrytypes.SignalMetrics, StepInterval: qbtypes.Step{Duration: time.Minute},
				Aggregations: []qbtypes.MetricAggregation{{MetricName: "http.server.request.duration", TimeAggregation: metrictypes.TimeAggregation{String: valuer.NewString("p95")}, SpaceAggregation: metrictypes.SpaceAggregationSum}},
			},
		}}},
	}

	converted, err := ToLite(request, MetricMetadata{
		Temporality: map[string]metrictypes.Temporality{"http.server.request.duration": metrictypes.Cumulative},
		Types:       map[string]metrictypes.Type{"http.server.request.duration": metrictypes.HistogramType},
	})
	if err != nil {
		t.Fatalf("ToLite() error = %v", err)
	}
	query, ok := converted.Queries[0].(litequery.MetricQuery)
	if !ok {
		t.Fatalf("query type = %T, want MetricQuery", converted.Queries[0])
	}
	if query.Aggregation.Type != litequery.MetricHistogram || query.Aggregation.Temporality != litequery.TemporalityUnspecified || query.Aggregation.TimeAggregation != litequery.TimeAggregateCount || query.Aggregation.SpaceAggregation != litequery.SpaceAggregateP95 {
		t.Fatalf("aggregation = %#v, want histogram with unspecified temporality", query.Aggregation)
	}
	if _, err := (litequery.DefaultPlanner{}).Plan(converted); err != nil {
		t.Fatalf("Plan() error = %v", err)
	}
}

func TestToLiteInfersUnspecifiedIntrinsicFieldTypes(t *testing.T) {
	tests := []struct {
		name  string
		spec  any
		field litequery.FieldRef
	}{
		{
			name: "log timestamp",
			spec: qbtypes.QueryBuilderQuery[qbtypes.LogAggregation]{
				Name: "A", Signal: telemetrytypes.SignalLogs,
				Aggregations: []qbtypes.LogAggregation{{Expression: "count()"}},
				SelectFields: []telemetrytypes.TelemetryFieldKey{{Name: "timestamp"}},
				GroupBy:      []qbtypes.GroupByKey{{TelemetryFieldKey: telemetrytypes.TelemetryFieldKey{Name: "timestamp"}}},
				Order:        []qbtypes.OrderBy{{Key: qbtypes.OrderByKey{TelemetryFieldKey: telemetrytypes.TelemetryFieldKey{Name: "timestamp"}}}},
			},
			field: litequery.FieldRef{Name: "timestamp", Context: litequery.FieldContextLog, Type: litequery.ValueTypeNumber},
		},
		{
			name: "trace duration",
			spec: qbtypes.QueryBuilderQuery[qbtypes.TraceAggregation]{
				Name: "A", Signal: telemetrytypes.SignalTraces,
				Aggregations: []qbtypes.TraceAggregation{{Expression: "count()"}},
				SelectFields: []telemetrytypes.TelemetryFieldKey{{Name: "duration_nano"}},
				GroupBy:      []qbtypes.GroupByKey{{TelemetryFieldKey: telemetrytypes.TelemetryFieldKey{Name: "duration_nano"}}},
				Order:        []qbtypes.OrderBy{{Key: qbtypes.OrderByKey{TelemetryFieldKey: telemetrytypes.TelemetryFieldKey{Name: "duration_nano"}}}},
			},
			field: litequery.FieldRef{Name: "duration_nano", Context: litequery.FieldContextSpan, Type: litequery.ValueTypeNumber},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			request := &qbtypes.QueryRangeRequest{
				Start: 1_000, End: 61_000, RequestType: qbtypes.RequestTypeRaw,
				CompositeQuery: qbtypes.CompositeQuery{Queries: []qbtypes.QueryEnvelope{{
					Type: qbtypes.QueryTypeBuilder, Spec: test.spec,
				}}},
			}

			converted, err := ToLite(request, MetricMetadata{})
			if err != nil {
				t.Fatalf("ToLite() error = %v", err)
			}
			common := converted.Queries[0].GetCommon()
			if len(common.Select) != 1 || common.Select[0] != test.field {
				t.Fatalf("Select = %#v, want %#v", common.Select, test.field)
			}
			if len(common.GroupBy) != 1 || common.GroupBy[0] != test.field {
				t.Fatalf("GroupBy = %#v, want %#v", common.GroupBy, test.field)
			}
			if len(common.Order) != 1 || common.Order[0].Field != test.field {
				t.Fatalf("Order = %#v, want field %#v", common.Order, test.field)
			}
			plan, err := (litequery.DefaultPlanner{}).Plan(converted)
			if err != nil {
				t.Fatalf("Plan() error = %v", err)
			}
			if _, err := litequery.NewCompiler(nil).Compile(plan); err != nil {
				t.Fatalf("Compile() error = %v", err)
			}
		})
	}
}

func TestToLiteRejectsUnsupportedV5Features(t *testing.T) {
	tests := []struct {
		name    string
		request *qbtypes.QueryRangeRequest
	}{
		{
			name: "clickhouse SQL",
			request: &qbtypes.QueryRangeRequest{Start: 1, End: 2, RequestType: qbtypes.RequestTypeScalar, CompositeQuery: qbtypes.CompositeQuery{Queries: []qbtypes.QueryEnvelope{{
				Type: qbtypes.QueryTypeClickHouseSQL, Spec: qbtypes.ClickHouseQuery{Name: "A", Query: "SELECT 1"},
			}}}},
		},
		{
			name: "post processing function",
			request: &qbtypes.QueryRangeRequest{Start: 1, End: 2, RequestType: qbtypes.RequestTypeScalar, CompositeQuery: qbtypes.CompositeQuery{Queries: []qbtypes.QueryEnvelope{{
				Type: qbtypes.QueryTypeBuilder,
				Spec: qbtypes.QueryBuilderQuery[qbtypes.LogAggregation]{Name: "A", Signal: telemetrytypes.SignalLogs, Aggregations: []qbtypes.LogAggregation{{Expression: "count()"}}, Functions: []qbtypes.Function{{Name: qbtypes.FunctionNameEWMA3}}},
			}}}},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			_, err := ToLite(test.request, MetricMetadata{})
			var unsupported *UnsupportedError
			if !errors.As(err, &unsupported) {
				t.Fatalf("ToLite() error = %v, want UnsupportedError", err)
			}
		})
	}
}

func TestFromLiteProducesV5TimeSeriesAndRawData(t *testing.T) {
	timeSeriesRequest := &qbtypes.QueryRangeRequest{RequestType: qbtypes.RequestTypeTimeSeries, CompositeQuery: qbtypes.CompositeQuery{Queries: []qbtypes.QueryEnvelope{{
		Type: qbtypes.QueryTypeBuilder,
		Spec: qbtypes.QueryBuilderQuery[qbtypes.LogAggregation]{Name: "A", StepInterval: qbtypes.Step{Duration: time.Minute}},
	}}}}
	result, err := FromLite(timeSeriesRequest, litequery.ExecutionResult{Queries: []litequery.QueryResult{{
		Name:    "A",
		Columns: []litequery.ResultColumn{{Name: "timestamp"}, {Name: "group_0", Field: &litequery.FieldRef{Name: "service.name", Context: litequery.FieldContextResource, Type: litequery.ValueTypeString}}, {Name: "value"}},
		Rows:    [][]any{{int64(60_000), "api", float64(3)}},
	}}})
	if err != nil {
		t.Fatalf("FromLite() error = %v", err)
	}
	data := result.Data.Results[0].(*qbtypes.TimeSeriesData)
	if len(data.Aggregations) != 1 || len(data.Aggregations[0].Series) != 1 || data.Aggregations[0].Series[0].Values[0].Value != 3 {
		t.Fatalf("time series response = %#v", data)
	}

	rawRequest := &qbtypes.QueryRangeRequest{RequestType: qbtypes.RequestTypeRaw}
	rawResult, err := FromLite(rawRequest, litequery.ExecutionResult{Queries: []litequery.QueryResult{{
		Name:    "L",
		Columns: []litequery.ResultColumn{{Name: "field_0", Field: &litequery.FieldRef{Name: "timestamp", Context: litequery.FieldContextLog, Type: litequery.ValueTypeNumber}}, {Name: "field_1", Field: &litequery.FieldRef{Name: "body", Context: litequery.FieldContextBody, Type: litequery.ValueTypeString}}},
		Rows:    [][]any{{int64(1_000_000_000), "hello"}},
	}}})
	if err != nil {
		t.Fatalf("FromLite() raw error = %v", err)
	}
	raw := rawResult.Data.Results[0].(*qbtypes.RawData)
	if raw.Rows[0].Data["body"] != "hello" || raw.Rows[0].Timestamp.UnixNano() != 1_000_000_000 {
		t.Fatalf("raw response = %#v", raw)
	}
}

func TestFromLiteAcceptsClickHouseUnsignedScanValues(t *testing.T) {
	request := &qbtypes.QueryRangeRequest{
		RequestType: qbtypes.RequestTypeTimeSeries,
		CompositeQuery: qbtypes.CompositeQuery{Queries: []qbtypes.QueryEnvelope{{
			Type: qbtypes.QueryTypeBuilder,
			Spec: qbtypes.QueryBuilderQuery[qbtypes.LogAggregation]{
				Name:         "A",
				StepInterval: qbtypes.Step{Duration: time.Minute},
			},
		}}},
	}
	result, err := FromLite(request, litequery.ExecutionResult{Queries: []litequery.QueryResult{{
		Name:    "A",
		Columns: []litequery.ResultColumn{{Name: "timestamp"}, {Name: "value"}},
		Rows:    [][]any{{uint32(60_000), uint16(3)}},
	}}})
	if err != nil {
		t.Fatalf("FromLite() error = %v", err)
	}
	data := result.Data.Results[0].(*qbtypes.TimeSeriesData)
	if len(data.Aggregations[0].Series) != 1 || data.Aggregations[0].Series[0].Values[0].Value != 3 {
		t.Fatalf("time series response = %#v", data)
	}
}
