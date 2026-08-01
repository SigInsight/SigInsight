package liteadapter

import (
	"testing"
	"time"

	qbtypes "github.com/SigNoz/signoz/pkg/types/querybuildertypes/querybuildertypesv5"
	"github.com/SigNoz/signoz/pkg/types/telemetrytypes"
)

func TestFieldKeySelectorsCollectsUnresolvedV5Fields(t *testing.T) {
	request := &qbtypes.QueryRangeRequest{
		Start: 1_000, End: 61_000,
		CompositeQuery: qbtypes.CompositeQuery{Queries: []qbtypes.QueryEnvelope{
			{
				Type: qbtypes.QueryTypeBuilder,
				Spec: qbtypes.QueryBuilderQuery[qbtypes.LogAggregation]{
					Name: "logs", Signal: telemetrytypes.SignalLogs, StepInterval: qbtypes.Step{Duration: time.Minute},
					Filter:       &qbtypes.Filter{Expression: "host.name = 'worker-1' AND severity_text = 'ERROR'"},
					Aggregations: []qbtypes.LogAggregation{{Expression: "sum(attribute.response.size)"}},
				},
			},
			{
				Type: qbtypes.QueryTypeBuilder,
				Spec: qbtypes.QueryBuilderQuery[qbtypes.TraceAggregation]{
					Name: "traces", Signal: telemetrytypes.SignalTraces,
					Filter:       &qbtypes.Filter{Expression: "service.name = 'api'"},
					Aggregations: []qbtypes.TraceAggregation{{Expression: "count()"}},
				},
			},
		}},
	}

	selectors := FieldKeySelectors(request)
	want := map[string]bool{
		"logs;;host.name":              false,
		"logs;attribute;response.size": false,
		"traces;;service.name":         false,
	}
	for _, selector := range selectors {
		identity := selector.Signal.StringValue() + ";" + selector.FieldContext.StringValue() + ";" + selector.Name
		if _, ok := want[identity]; ok {
			want[identity] = true
		}
		if selector.SelectorMatchType != telemetrytypes.FieldSelectorMatchTypeExact {
			t.Fatalf("selector %#v is not exact", selector)
		}
	}
	for identity, found := range want {
		if !found {
			t.Errorf("missing selector %s in %#v", identity, selectors)
		}
	}
}

func TestFieldKeySelectorsSkipsIntrinsicFields(t *testing.T) {
	request := &qbtypes.QueryRangeRequest{
		Start: 1_000, End: 61_000,
		CompositeQuery: qbtypes.CompositeQuery{Queries: []qbtypes.QueryEnvelope{
			{Type: qbtypes.QueryTypeBuilder, Spec: qbtypes.QueryBuilderQuery[qbtypes.LogAggregation]{
				Name: "logs", Signal: telemetrytypes.SignalLogs,
				SelectFields: []telemetrytypes.TelemetryFieldKey{{Name: "timestamp"}, {Name: "body"}},
			}},
			{Type: qbtypes.QueryTypeBuilder, Spec: qbtypes.QueryBuilderQuery[qbtypes.TraceAggregation]{
				Name: "traces", Signal: telemetrytypes.SignalTraces,
				SelectFields: []telemetrytypes.TelemetryFieldKey{{Name: "duration_nano"}, {Name: "trace_id"}},
			}},
		}},
	}

	if selectors := FieldKeySelectors(request); len(selectors) != 0 {
		t.Fatalf("selectors = %#v, want no intrinsic metadata lookups", selectors)
	}
}

func TestFieldKeySelectorsSkipsAggregateOrderAliases(t *testing.T) {
	request := &qbtypes.QueryRangeRequest{
		Start: 1_000, End: 61_000, RequestType: qbtypes.RequestTypeTimeSeries,
		CompositeQuery: qbtypes.CompositeQuery{Queries: []qbtypes.QueryEnvelope{{
			Type: qbtypes.QueryTypeBuilder,
			Spec: qbtypes.QueryBuilderQuery[qbtypes.LogAggregation]{
				Name: "logs", Signal: telemetrytypes.SignalLogs,
				GroupBy: []qbtypes.GroupByKey{{TelemetryFieldKey: telemetrytypes.TelemetryFieldKey{Name: "host.name"}}},
				Order: []qbtypes.OrderBy{
					{Key: qbtypes.OrderByKey{TelemetryFieldKey: telemetrytypes.TelemetryFieldKey{Name: "__result_0"}}, Direction: qbtypes.OrderDirectionDesc},
					{Key: qbtypes.OrderByKey{TelemetryFieldKey: telemetrytypes.TelemetryFieldKey{Name: "host.name"}}, Direction: qbtypes.OrderDirectionAsc},
				},
			},
		}}},
	}
	selectors := FieldKeySelectors(request)
	if len(selectors) != 1 || selectors[0].Name != "host.name" {
		t.Fatalf("selectors = %#v, want only host.name", selectors)
	}
}

func TestFieldKeySelectorsSkipsTraceSummaryVirtualOrderFields(t *testing.T) {
	request := &qbtypes.QueryRangeRequest{
		Start: 1_000, End: 61_000, RequestType: qbtypes.RequestTypeTrace,
		CompositeQuery: qbtypes.CompositeQuery{Queries: []qbtypes.QueryEnvelope{{
			Type: qbtypes.QueryTypeBuilder,
			Spec: qbtypes.QueryBuilderQuery[qbtypes.TraceAggregation]{
				Name: "traces", Signal: telemetrytypes.SignalTraces,
				Order: []qbtypes.OrderBy{
					{Key: qbtypes.OrderByKey{TelemetryFieldKey: telemetrytypes.TelemetryFieldKey{Name: "span_count"}}, Direction: qbtypes.OrderDirectionDesc},
					{Key: qbtypes.OrderByKey{TelemetryFieldKey: telemetrytypes.TelemetryFieldKey{Name: "trace_duration"}}, Direction: qbtypes.OrderDirectionDesc},
					{Key: qbtypes.OrderByKey{TelemetryFieldKey: telemetrytypes.TelemetryFieldKey{Name: "service.name"}}, Direction: qbtypes.OrderDirectionAsc},
				},
			},
		}}},
	}

	selectors := FieldKeySelectors(request)
	if len(selectors) != 1 || selectors[0].Name != "service.name" {
		t.Fatalf("selectors = %#v, want only service.name", selectors)
	}
}

func TestFieldKeySelectorsSkipsDisabledQueries(t *testing.T) {
	request := &qbtypes.QueryRangeRequest{
		Start: 1_000, End: 61_000,
		CompositeQuery: qbtypes.CompositeQuery{Queries: []qbtypes.QueryEnvelope{
			{Type: qbtypes.QueryTypeBuilder, Spec: qbtypes.QueryBuilderQuery[qbtypes.LogAggregation]{
				Name: "disabled_logs", Disabled: true,
				Filter: &qbtypes.Filter{Expression: "host.name = 'old-host'"},
			}},
			{Type: qbtypes.QueryTypeBuilder, Spec: qbtypes.QueryBuilderQuery[qbtypes.TraceAggregation]{
				Name: "disabled_traces", Disabled: true,
				GroupBy: []qbtypes.GroupByKey{{TelemetryFieldKey: telemetrytypes.TelemetryFieldKey{Name: "service.name"}}},
			}},
		}},
	}

	if selectors := FieldKeySelectors(request); len(selectors) != 0 {
		t.Fatalf("selectors = %#v, want disabled queries ignored", selectors)
	}
}
