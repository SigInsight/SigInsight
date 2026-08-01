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
		"logs;;severity_text":          false,
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
