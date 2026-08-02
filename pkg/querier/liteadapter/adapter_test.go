package liteadapter

import (
	"math"
	"strings"
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

func TestToLiteConvertsTypedInFilters(t *testing.T) {
	metadata := MetricMetadata{FieldKeys: map[string][]*telemetrytypes.TelemetryFieldKey{
		"http.status_code": {{Name: "http.status_code", Signal: telemetrytypes.SignalLogs, FieldContext: telemetrytypes.FieldContextAttribute, FieldDataType: telemetrytypes.FieldDataTypeNumber}},
		"error":            {{Name: "error", Signal: telemetrytypes.SignalLogs, FieldContext: telemetrytypes.FieldContextAttribute, FieldDataType: telemetrytypes.FieldDataTypeBool}},
	}}
	for _, test := range []struct {
		expression string
		kind       litequery.ValueKind
	}{
		{"http.status_code in [200, 500]", litequery.ValueNumberList},
		{"error not in [true, false]", litequery.ValueBoolList},
	} {
		t.Run(test.expression, func(t *testing.T) {
			filter, err := ParseFilterWithMetadata(test.expression, litequery.SignalLogs, metadata.FieldKeys)
			if err != nil {
				t.Fatalf("ParseFilterWithMetadata() error = %v", err)
			}
			predicate := filter.(litequery.Predicate)
			if predicate.Value.Kind != test.kind {
				t.Fatalf("value = %#v, want kind %q", predicate.Value, test.kind)
			}
		})
	}
}

func TestToLiteRejectsMixedTypeInFilter(t *testing.T) {
	if _, err := ParseFilter("attribute.value in [1, 'two']", litequery.SignalLogs); err == nil {
		t.Fatal("ParseFilter() accepted a mixed-type IN list")
	}
}

func TestToLiteDropsSumTemporalityWhenMetadataNormalizesMetricToGauge(t *testing.T) {
	request := &qbtypes.QueryRangeRequest{
		Start: 1_000, End: 61_000, RequestType: qbtypes.RequestTypeTimeSeries,
		CompositeQuery: qbtypes.CompositeQuery{Queries: []qbtypes.QueryEnvelope{{
			Type: qbtypes.QueryTypeBuilder,
			Spec: qbtypes.QueryBuilderQuery[qbtypes.MetricAggregation]{
				Name: "A", Signal: telemetrytypes.SignalMetrics, StepInterval: qbtypes.Step{Duration: time.Minute},
				Aggregations: []qbtypes.MetricAggregation{{
					MetricName: "http.server.request.count", Temporality: metrictypes.Cumulative,
					TimeAggregation: metrictypes.TimeAggregationLatest, SpaceAggregation: metrictypes.SpaceAggregationAvg,
				}},
			},
		}}},
	}
	metadata := MetricMetadata{Types: map[string]metrictypes.Type{"http.server.request.count": metrictypes.GaugeType}}

	converted, err := ToLite(request, metadata)
	if err != nil {
		t.Fatalf("ToLite() error = %v", err)
	}
	aggregation := converted.Queries[0].(litequery.MetricQuery).Aggregation
	if aggregation.Type != litequery.MetricGauge || aggregation.Temporality != litequery.TemporalityUnspecified {
		t.Fatalf("aggregation = %#v, want gauge without temporality", aggregation)
	}
}

func TestToLiteResolvesUnqualifiedLogResourceFieldFromMetadata(t *testing.T) {
	request := &qbtypes.QueryRangeRequest{
		Start: 1_000, End: 61_000, RequestType: qbtypes.RequestTypeTimeSeries,
		CompositeQuery: qbtypes.CompositeQuery{Queries: []qbtypes.QueryEnvelope{{
			Type: qbtypes.QueryTypeBuilder,
			Spec: qbtypes.QueryBuilderQuery[qbtypes.LogAggregation]{
				Name: "A", Signal: telemetrytypes.SignalLogs, StepInterval: qbtypes.Step{Duration: time.Minute},
				Aggregations: []qbtypes.LogAggregation{{Expression: "count()"}},
				Filter:       &qbtypes.Filter{Expression: "host.name = 'worker-1'"},
			},
		}}},
	}
	metadata := MetricMetadata{FieldKeys: map[string][]*telemetrytypes.TelemetryFieldKey{
		"host.name": {{Name: "host.name", Signal: telemetrytypes.SignalLogs, FieldContext: telemetrytypes.FieldContextResource, FieldDataType: telemetrytypes.FieldDataTypeString}},
	}}

	converted, err := ToLite(request, metadata)
	if err != nil {
		t.Fatalf("ToLite() error = %v", err)
	}
	query := converted.Queries[0].(litequery.LogQuery)
	predicate := query.Common.Filter.(litequery.Predicate)
	if predicate.Field != (litequery.FieldRef{Name: "host.name", Context: litequery.FieldContextResource, Type: litequery.ValueTypeString}) {
		t.Fatalf("filter field = %#v, want resource host.name", predicate.Field)
	}
	plan, err := (litequery.DefaultPlanner{}).Plan(converted)
	if err != nil {
		t.Fatalf("Plan() error = %v", err)
	}
	if _, err := litequery.NewCompiler(nil).Compile(plan); err != nil {
		t.Fatalf("Compile() error = %v", err)
	}
}

func TestFieldToLiteMatchesV5MetadataResolutionRules(t *testing.T) {
	metadata := MetricMetadata{FieldKeys: map[string][]*telemetrytypes.TelemetryFieldKey{
		"deployment.environment": {
			{Name: "deployment.environment", Signal: telemetrytypes.SignalTraces, FieldContext: telemetrytypes.FieldContextAttribute, FieldDataType: telemetrytypes.FieldDataTypeString},
			{Name: "deployment.environment", Signal: telemetrytypes.SignalTraces, FieldContext: telemetrytypes.FieldContextResource, FieldDataType: telemetrytypes.FieldDataTypeString},
		},
		"http.status_code": {
			{Name: "http.status_code", Signal: telemetrytypes.SignalTraces, FieldContext: telemetrytypes.FieldContextAttribute, FieldDataType: telemetrytypes.FieldDataTypeNumber},
		},
	}}
	tests := []struct {
		name     string
		key      telemetrytypes.TelemetryFieldKey
		fallback litequery.ValueType
		want     litequery.FieldRef
	}{
		{
			name: "unqualified ambiguous name prefers resource",
			key:  telemetrytypes.TelemetryFieldKey{Name: "deployment.environment"}, fallback: litequery.ValueTypeString,
			want: litequery.FieldRef{Name: "deployment.environment", Context: litequery.FieldContextResource, Type: litequery.ValueTypeString},
		},
		{
			name: "explicit attribute context overrides resource priority",
			key:  telemetrytypes.TelemetryFieldKey{Name: "deployment.environment", FieldContext: telemetrytypes.FieldContextAttribute}, fallback: litequery.ValueTypeString,
			want: litequery.FieldRef{Name: "deployment.environment", Context: litequery.FieldContextAttribute, Type: litequery.ValueTypeString},
		},
		{
			name: "numeric value resolves attribute type",
			key:  telemetrytypes.TelemetryFieldKey{Name: "http.status_code"}, fallback: litequery.ValueTypeNumber,
			want: litequery.FieldRef{Name: "http.status_code", Context: litequery.FieldContextAttribute, Type: litequery.ValueTypeNumber},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got, err := fieldToLite(test.key, litequery.SignalTraces, test.fallback, metadata)
			if err != nil {
				t.Fatalf("fieldToLite() error = %v", err)
			}
			if got != test.want {
				t.Fatalf("fieldToLite() = %#v, want %#v", got, test.want)
			}
		})
	}
}

func TestFieldToLiteInfersUnqualifiedNumericFieldAsAttributeWithoutMetadataKey(t *testing.T) {
	field, err := fieldToLite(
		telemetrytypes.TelemetryFieldKey{Name: "thread.id"},
		litequery.SignalLogs,
		litequery.ValueTypeNumber,
		MetricMetadata{FieldKeys: map[string][]*telemetrytypes.TelemetryFieldKey{}},
	)
	if err != nil {
		t.Fatalf("fieldToLite() error = %v", err)
	}
	want := litequery.FieldRef{Name: "thread.id", Context: litequery.FieldContextAttribute, Type: litequery.ValueTypeNumber}
	if field != want {
		t.Fatalf("fieldToLite() = %#v, want %#v", field, want)
	}
}

func TestFieldToLiteUsesOperatorTypeBeforeResourcePreference(t *testing.T) {
	metadata := MetricMetadata{FieldKeys: map[string][]*telemetrytypes.TelemetryFieldKey{
		"instance.id": {
			{Name: "instance.id", Signal: telemetrytypes.SignalLogs, FieldContext: telemetrytypes.FieldContextResource, FieldDataType: telemetrytypes.FieldDataTypeString},
			{Name: "instance.id", Signal: telemetrytypes.SignalLogs, FieldContext: telemetrytypes.FieldContextAttribute, FieldDataType: telemetrytypes.FieldDataTypeNumber},
		},
	}}
	field, err := fieldToLite(telemetrytypes.TelemetryFieldKey{Name: "instance.id"}, litequery.SignalLogs, litequery.ValueTypeNumber, metadata)
	if err != nil {
		t.Fatalf("fieldToLite() error = %v", err)
	}
	want := litequery.FieldRef{Name: "instance.id", Context: litequery.FieldContextAttribute, Type: litequery.ValueTypeNumber}
	if field != want {
		t.Fatalf("fieldToLite() = %#v, want %#v", field, want)
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

	converted, err := ToLite(request, MetricMetadata{FieldKeys: map[string][]*telemetrytypes.TelemetryFieldKey{
		"service.name": {{Name: "service.name", Signal: telemetrytypes.SignalTraces, FieldContext: telemetrytypes.FieldContextResource, FieldDataType: telemetrytypes.FieldDataTypeString}},
	}})
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

func TestToLiteDoesNotResolveAggregationOrderAsTelemetryField(t *testing.T) {
	request := &qbtypes.QueryRangeRequest{
		Start: 1_000, End: 61_000, RequestType: qbtypes.RequestTypeTimeSeries,
		CompositeQuery: qbtypes.CompositeQuery{Queries: []qbtypes.QueryEnvelope{{
			Type: qbtypes.QueryTypeBuilder,
			Spec: qbtypes.QueryBuilderQuery[qbtypes.LogAggregation]{
				Name: "A", Signal: telemetrytypes.SignalLogs, StepInterval: qbtypes.Step{Duration: time.Minute},
				Aggregations: []qbtypes.LogAggregation{{Expression: "count()"}},
				Order:        []qbtypes.OrderBy{{Key: qbtypes.OrderByKey{TelemetryFieldKey: telemetrytypes.TelemetryFieldKey{Name: "__result_0"}}, Direction: qbtypes.OrderDirectionDesc}},
			},
		}}},
	}
	converted, err := ToLite(request, MetricMetadata{FieldKeys: map[string][]*telemetrytypes.TelemetryFieldKey{}})
	if err != nil {
		t.Fatalf("ToLite() error = %v", err)
	}
	order := converted.Queries[0].GetCommon().Order
	if len(order) != 1 || order[0].Target != litequery.OrderByAggregation || order[0].Direction != litequery.SortDescending {
		t.Fatalf("order = %#v, want descending aggregation order", order)
	}
}

func TestToLiteRecognizesTraceSummaryVirtualOrderFields(t *testing.T) {
	request := &qbtypes.QueryRangeRequest{
		Start: 1_000, End: 61_000, RequestType: qbtypes.RequestTypeTrace,
		CompositeQuery: qbtypes.CompositeQuery{Queries: []qbtypes.QueryEnvelope{{
			Type: qbtypes.QueryTypeBuilder,
			Spec: qbtypes.QueryBuilderQuery[qbtypes.TraceAggregation]{
				Name: "A", Signal: telemetrytypes.SignalTraces,
				Order: []qbtypes.OrderBy{{Key: qbtypes.OrderByKey{TelemetryFieldKey: telemetrytypes.TelemetryFieldKey{Name: "span_count"}}, Direction: qbtypes.OrderDirectionDesc}},
			},
		}}},
	}
	converted, err := ToLite(request, MetricMetadata{FieldKeys: map[string][]*telemetrytypes.TelemetryFieldKey{}})
	if err != nil {
		t.Fatalf("ToLite() error = %v", err)
	}
	order := converted.Queries[0].GetCommon().Order
	if len(order) != 1 || order[0].Field.Name != "span_count" || order[0].Field.Type != litequery.ValueTypeNumber {
		t.Fatalf("trace summary order = %#v", order)
	}
}

func TestValidateRequestRangeBeforeMetadataLookup(t *testing.T) {
	if err := ValidateRequestRange(nil); err == nil {
		t.Fatal("ValidateRequestRange(nil) error = nil")
	}
	for _, request := range []*qbtypes.QueryRangeRequest{
		{Start: 2, End: 2},
		{Start: 3, End: 2},
	} {
		if err := ValidateRequestRange(request); err == nil {
			t.Fatalf("ValidateRequestRange(%d, %d) error = nil", request.Start, request.End)
		}
	}
	tooLarge := &qbtypes.QueryRangeRequest{Start: 1, End: uint64(math.MaxInt64) + 1}
	if err := ValidateRequestRange(tooLarge); err == nil {
		t.Fatal("ValidateRequestRange() accepted range above int64")
	}
	logOverflow := &qbtypes.QueryRangeRequest{
		Start: 1, End: uint64(math.MaxInt64/1_000_000) + 1,
		CompositeQuery: qbtypes.CompositeQuery{Queries: []qbtypes.QueryEnvelope{{
			Type: qbtypes.QueryTypeBuilder,
			Spec: qbtypes.QueryBuilderQuery[qbtypes.LogAggregation]{Name: "A"},
		}}},
	}
	if err := ValidateRequestRange(logOverflow); err == nil {
		t.Fatal("ValidateRequestRange() accepted log nanosecond overflow")
	}
	logOverflow.CompositeQuery.Queries[0].Spec = qbtypes.QueryBuilderQuery[qbtypes.LogAggregation]{Name: "A", Disabled: true}
	if err := ValidateRequestRange(logOverflow); err != nil {
		t.Fatalf("ValidateRequestRange() rejected range due to disabled log query: %v", err)
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

func TestToLiteAppliesMetadataTemporalityToHistogram(t *testing.T) {
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
	if query.Aggregation.Type != litequery.MetricHistogram || query.Aggregation.Temporality != litequery.TemporalityCumulative || query.Aggregation.TimeAggregation != litequery.TimeAggregateCount || query.Aggregation.SpaceAggregation != litequery.SpaceAggregateP95 {
		t.Fatalf("aggregation = %#v, want cumulative histogram", query.Aggregation)
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
		})
	}
}

func TestParseFilterResolvesTypedIntrinsicBeforeTelemetryMetadata(t *testing.T) {
	tests := []struct {
		name       string
		expression string
		signal     litequery.Signal
		wantField  litequery.FieldRef
		wantKind   litequery.ValueKind
	}{
		{
			name:       "typed trace name not in",
			expression: "name:string NOT IN ['GET /health']",
			signal:     litequery.SignalTraces,
			wantField:  litequery.FieldRef{Name: "name", Context: litequery.FieldContextSpan, Type: litequery.ValueTypeString},
			wantKind:   litequery.ValueStringList,
		},
		{
			name:       "typed trace error flag not in",
			expression: "has_error:bool NOT IN [true]",
			signal:     litequery.SignalTraces,
			wantField:  litequery.FieldRef{Name: "has_error", Context: litequery.FieldContextSpan, Type: litequery.ValueTypeBool},
			wantKind:   litequery.ValueBoolList,
		},
		{
			name:       "physical root span predicate",
			expression: "span.parent_span_id = ''",
			signal:     litequery.SignalTraces,
			wantField:  litequery.FieldRef{Name: "parent_span_id", Context: litequery.FieldContextSpan, Type: litequery.ValueTypeString},
			wantKind:   litequery.ValueString,
		},
		{
			name:       "root span scope without metadata",
			expression: "isRoot = true",
			signal:     litequery.SignalTraces,
			wantField:  litequery.FieldRef{Name: "isRoot", Context: litequery.FieldContextSpan, Type: litequery.ValueTypeBool},
			wantKind:   litequery.ValueBool,
		},
		{
			name:       "entrypoint span scope without metadata",
			expression: "isEntryPoint = true",
			signal:     litequery.SignalTraces,
			wantField:  litequery.FieldRef{Name: "isEntryPoint", Context: litequery.FieldContextSpan, Type: litequery.ValueTypeBool},
			wantKind:   litequery.ValueBool,
		},
		{
			name:       "typed log severity",
			expression: "severity_text:string = 'ERROR'",
			signal:     litequery.SignalLogs,
			wantField:  litequery.FieldRef{Name: "severity_text", Context: litequery.FieldContextLog, Type: litequery.ValueTypeString},
			wantKind:   litequery.ValueString,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			filter, err := ParseFilterWithMetadata(test.expression, test.signal, map[string][]*telemetrytypes.TelemetryFieldKey{})
			if err != nil {
				t.Fatalf("ParseFilterWithMetadata() error = %v", err)
			}
			predicate, ok := filter.(litequery.Predicate)
			if !ok {
				t.Fatalf("filter = %#v, want Predicate", filter)
			}
			if predicate.Field != test.wantField || predicate.Value.Kind != test.wantKind {
				t.Fatalf("predicate = %#v, want field %#v and value kind %q", predicate, test.wantField, test.wantKind)
			}
		})
	}
}

func TestParseFilterRejectsIncorrectTypedIntrinsic(t *testing.T) {
	_, err := ParseFilterWithMetadata("name:number = 1", litequery.SignalTraces, map[string][]*telemetrytypes.TelemetryFieldKey{})
	if err == nil || !strings.Contains(err.Error(), "intrinsic field \"name\" has type string, not number") {
		t.Fatalf("ParseFilterWithMetadata() error = %v, want intrinsic type mismatch", err)
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

func TestToLiteSkipsDisabledQueriesBeforeCapabilityValidation(t *testing.T) {
	request := &qbtypes.QueryRangeRequest{
		Start: 1_000, End: 61_000, RequestType: qbtypes.RequestTypeScalar,
		CompositeQuery: qbtypes.CompositeQuery{Queries: []qbtypes.QueryEnvelope{
			{
				Type: qbtypes.QueryTypeBuilder,
				Spec: qbtypes.QueryBuilderQuery[qbtypes.LogAggregation]{
					Name: "A", Signal: telemetrytypes.SignalLogs,
					Aggregations: []qbtypes.LogAggregation{{Expression: "count()"}},
				},
			},
			{
				Type: qbtypes.QueryTypeBuilder,
				Spec: qbtypes.QueryBuilderQuery[qbtypes.TraceAggregation]{
					Name: "retired_trace", Disabled: true,
					Functions: []qbtypes.Function{{Name: qbtypes.FunctionNameEWMA3}},
				},
			},
			{
				Type: qbtypes.QueryTypeBuilder,
				Spec: qbtypes.QueryBuilderQuery[qbtypes.MetricAggregation]{
					Name: "retired_metric", Disabled: true,
					Aggregations: []qbtypes.MetricAggregation{{MetricName: "unknown"}, {MetricName: "duplicate"}},
				},
			},
			{
				Type: qbtypes.QueryTypeFormula,
				Spec: qbtypes.QueryBuilderFormula{
					Name: "retired_formula", Disabled: true,
					Functions: []qbtypes.Function{{Name: qbtypes.FunctionNameEWMA3}},
				},
			},
		}},
	}

	converted, err := ToLite(request, MetricMetadata{})
	if err != nil {
		t.Fatalf("ToLite() error = %v", err)
	}
	if len(converted.Queries) != 1 || converted.Queries[0].GetCommon().Name != "A" {
		t.Fatalf("queries = %#v, want only enabled query A", converted.Queries)
	}
	if len(converted.Formulas) != 0 {
		t.Fatalf("formulas = %#v, want disabled formula ignored", converted.Formulas)
	}
}

func TestFromLiteProducesV5TimeSeriesAndRawData(t *testing.T) {
	timeSeriesRequest := &qbtypes.QueryRangeRequest{Start: 1, End: 120_000, RequestType: qbtypes.RequestTypeTimeSeries, CompositeQuery: qbtypes.CompositeQuery{Queries: []qbtypes.QueryEnvelope{{
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

	rawRequest := &qbtypes.QueryRangeRequest{Start: 1, End: 2, RequestType: qbtypes.RequestTypeRaw}
	traceID := litequery.FieldRef{Name: "trace_id", Context: litequery.FieldContextLog, Type: litequery.ValueTypeString}
	spanID := litequery.FieldRef{Name: "span_id", Context: litequery.FieldContextLog, Type: litequery.ValueTypeString}
	rawResult, err := FromLite(rawRequest, litequery.ExecutionResult{Queries: []litequery.QueryResult{{
		Name:    "L",
		Columns: []litequery.ResultColumn{{Name: "field_0", Field: &litequery.FieldRef{Name: "timestamp", Context: litequery.FieldContextLog, Type: litequery.ValueTypeNumber}}, {Name: "field_1", Field: &litequery.FieldRef{Name: "body", Context: litequery.FieldContextBody, Type: litequery.ValueTypeString}}, {Name: "field_2", Field: &traceID}, {Name: "field_3", Field: &spanID}},
		Rows:    [][]any{{int64(1_000_000_000), "hello", "trace-1", "span-1"}},
	}}})
	if err != nil {
		t.Fatalf("FromLite() raw error = %v", err)
	}
	raw := rawResult.Data.Results[0].(*qbtypes.RawData)
	if raw.Rows[0].Data["body"] != "hello" || raw.Rows[0].Data["trace_id"] != "trace-1" || raw.Rows[0].Data["span_id"] != "span-1" || raw.Rows[0].Timestamp.UnixNano() != 1_000_000_000 {
		t.Fatalf("raw response = %#v", raw)
	}
}

func TestFromLiteMarksTruncatedResults(t *testing.T) {
	request := &qbtypes.QueryRangeRequest{Start: 1, End: 2, RequestType: qbtypes.RequestTypeRaw}
	result, err := FromLite(request, litequery.ExecutionResult{
		Queries:  []litequery.QueryResult{{Name: "A", Truncated: true}},
		Warnings: []string{"query \"A\" returned more than 1000 rows"},
	})
	if err != nil {
		t.Fatalf("FromLite() error = %v", err)
	}
	if result.Warning == nil || result.Warning.Code != qbtypes.QueryWarningCodeResultLimit || len(result.Warning.Warnings) != 1 {
		t.Fatalf("warning = %#v, want result-limit warning", result.Warning)
	}
}

func TestFromLiteAcceptsClickHouseUnsignedScanValues(t *testing.T) {
	request := &qbtypes.QueryRangeRequest{
		Start: 1, End: 120_000, RequestType: qbtypes.RequestTypeTimeSeries,
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

func TestFromLiteKeepsDistinctSeriesInFirstSeenOrder(t *testing.T) {
	first := litequery.FieldRef{Name: "first", Context: litequery.FieldContextLabel, Type: litequery.ValueTypeString}
	second := litequery.FieldRef{Name: "second", Context: litequery.FieldContextLabel, Type: litequery.ValueTypeString}
	request := &qbtypes.QueryRangeRequest{
		Start: 1, End: 3_000, RequestType: qbtypes.RequestTypeTimeSeries,
		CompositeQuery: qbtypes.CompositeQuery{Queries: []qbtypes.QueryEnvelope{{
			Type: qbtypes.QueryTypeBuilder,
			Spec: qbtypes.QueryBuilderQuery[qbtypes.MetricAggregation]{Name: "A", StepInterval: qbtypes.Step{Duration: time.Second}},
		}}},
	}
	response, err := FromLite(request, litequery.ExecutionResult{Queries: []litequery.QueryResult{{
		Name:    "A",
		Columns: []litequery.ResultColumn{{Name: "timestamp"}, {Name: "group_0", Field: &first}, {Name: "group_1", Field: &second}, {Name: "value"}},
		Rows: [][]any{
			{int64(2_000), "a b", "c", float64(3)},
			{int64(1_000), "a", "b c", float64(2)},
			{int64(1_000), "a b", "c", float64(1)},
		},
	}}})
	if err != nil {
		t.Fatalf("FromLite() error = %v", err)
	}
	series := response.Data.Results[0].(*qbtypes.TimeSeriesData).Aggregations[0].Series
	if len(series) != 2 || len(series[0].Values) != 2 || series[0].Labels[0].Value != "a b" || series[1].Labels[0].Value != "a" {
		t.Fatalf("series = %#v, want two collision-safe series in first-seen order", series)
	}
	if series[0].Values[0].Timestamp != 1_000 || series[0].Values[1].Timestamp != 2_000 {
		t.Fatalf("series values = %#v, want timestamp order", series[0].Values)
	}
	if series[0].Labels[0].Key.FieldContext != telemetrytypes.FieldContextMetric {
		t.Fatalf("metric label context = %s, want metric", series[0].Labels[0].Key.FieldContext.StringValue())
	}
}

func TestFromLiteSanitizesNonFiniteScalarAndRejectsMalformedRows(t *testing.T) {
	request := &qbtypes.QueryRangeRequest{Start: 1, End: 2, RequestType: qbtypes.RequestTypeScalar}
	response, err := FromLite(request, litequery.ExecutionResult{Queries: []litequery.QueryResult{{
		Name: "F", Columns: []litequery.ResultColumn{{Name: "value"}}, Rows: [][]any{{math.NaN()}, {math.Inf(1)}},
	}}})
	if err != nil {
		t.Fatalf("FromLite() error = %v", err)
	}
	data := response.Data.Results[0].(*qbtypes.ScalarData).Data
	if data[0][0] != nil || data[1][0] != nil {
		t.Fatalf("scalar data = %#v, want non-finite values converted to nil", data)
	}

	_, err = FromLite(request, litequery.ExecutionResult{Queries: []litequery.QueryResult{{
		Name: "broken", Columns: []litequery.ResultColumn{{Name: "value"}}, Rows: [][]any{{float64(1), float64(2)}},
	}}})
	if err == nil {
		t.Fatal("FromLite() error = nil, want malformed scalar row error")
	}
}

func TestFromLiteFillsTimeSeriesGapsIncludingEmptyResults(t *testing.T) {
	request := &qbtypes.QueryRangeRequest{
		Start: 1_500, End: 3_500, RequestType: qbtypes.RequestTypeTimeSeries,
		FormatOptions: &qbtypes.FormatOptions{FillGaps: true},
		CompositeQuery: qbtypes.CompositeQuery{Queries: []qbtypes.QueryEnvelope{{
			Type: qbtypes.QueryTypeBuilder,
			Spec: qbtypes.QueryBuilderQuery[qbtypes.LogAggregation]{Name: "A", StepInterval: qbtypes.Step{Duration: time.Second}},
		}}},
	}
	response, err := FromLite(request, litequery.ExecutionResult{Queries: []litequery.QueryResult{{
		Name: "A", Columns: []litequery.ResultColumn{{Name: "timestamp"}, {Name: "value"}}, Rows: [][]any{{int64(2_000), float64(5)}},
	}}})
	if err != nil {
		t.Fatalf("FromLite() error = %v", err)
	}
	values := response.Data.Results[0].(*qbtypes.TimeSeriesData).Aggregations[0].Series[0].Values
	if len(values) != 3 || values[0].Timestamp != 1_000 || values[0].Value != 0 || values[1].Value != 5 || values[2].Timestamp != 3_000 {
		t.Fatalf("filled values = %#v", values)
	}

	empty, err := FromLite(request, litequery.ExecutionResult{Queries: []litequery.QueryResult{{
		Name: "A", Columns: []litequery.ResultColumn{{Name: "timestamp"}, {Name: "value"}},
	}}})
	if err != nil {
		t.Fatalf("FromLite(empty) error = %v", err)
	}
	emptySeries := empty.Data.Results[0].(*qbtypes.TimeSeriesData).Aggregations[0].Series
	if len(emptySeries) != 1 || len(emptySeries[0].Values) != 3 {
		t.Fatalf("empty filled series = %#v", emptySeries)
	}
}

func TestFromLiteGapFillTreatsEndAsExclusive(t *testing.T) {
	request := &qbtypes.QueryRangeRequest{
		Start: 1_000, End: 3_000, RequestType: qbtypes.RequestTypeTimeSeries,
		FormatOptions: &qbtypes.FormatOptions{FillGaps: true},
		CompositeQuery: qbtypes.CompositeQuery{Queries: []qbtypes.QueryEnvelope{{
			Type: qbtypes.QueryTypeBuilder,
			Spec: qbtypes.QueryBuilderQuery[qbtypes.LogAggregation]{Name: "A", StepInterval: qbtypes.Step{Duration: time.Second}},
		}}},
	}
	response, err := FromLite(request, litequery.ExecutionResult{Queries: []litequery.QueryResult{{
		Name: "A", Columns: []litequery.ResultColumn{{Name: "timestamp"}, {Name: "value"}},
	}}})
	if err != nil {
		t.Fatalf("FromLite() error = %v", err)
	}
	values := response.Data.Results[0].(*qbtypes.TimeSeriesData).Aggregations[0].Series[0].Values
	if len(values) != 2 || values[0].Timestamp != 1_000 || values[1].Timestamp != 2_000 {
		t.Fatalf("filled values = %#v, want buckets in [start, end)", values)
	}
}

func TestFromLiteUsesEnabledBuilderStepForFormulaGapFill(t *testing.T) {
	request := &qbtypes.QueryRangeRequest{
		Start: 1_000, End: 3_000, RequestType: qbtypes.RequestTypeTimeSeries,
		FormatOptions: &qbtypes.FormatOptions{FillGaps: true},
		CompositeQuery: qbtypes.CompositeQuery{Queries: []qbtypes.QueryEnvelope{
			{
				Type: qbtypes.QueryTypeBuilder,
				Spec: qbtypes.QueryBuilderQuery[qbtypes.LogAggregation]{Name: "F", Disabled: true, StepInterval: qbtypes.Step{Duration: 5 * time.Second}},
			},
			{
				Type: qbtypes.QueryTypeBuilder,
				Spec: qbtypes.QueryBuilderQuery[qbtypes.LogAggregation]{Name: "A", StepInterval: qbtypes.Step{Duration: time.Second}},
			},
			{
				Type: qbtypes.QueryTypeFormula,
				Spec: qbtypes.QueryBuilderFormula{Name: "F", Expression: "not a valid legacy expression("},
			},
		}},
	}
	response, err := FromLite(request, litequery.ExecutionResult{Queries: []litequery.QueryResult{{
		Name: "F", Columns: []litequery.ResultColumn{{Name: "timestamp"}, {Name: "value"}},
	}}})
	if err != nil {
		t.Fatalf("FromLite() error = %v", err)
	}
	values := response.Data.Results[0].(*qbtypes.TimeSeriesData).Aggregations[0].Series[0].Values
	if len(values) != 2 {
		t.Fatalf("formula gap-filled values = %#v, want builder step", values)
	}
}
