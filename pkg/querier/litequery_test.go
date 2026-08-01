package querier

import (
	"context"
	"testing"
	"time"

	"github.com/SigNoz/signoz/pkg/litequery"
	"github.com/SigNoz/signoz/pkg/querier/liteadapter"
	qbtypes "github.com/SigNoz/signoz/pkg/types/querybuildertypes/querybuildertypesv5"
	"github.com/SigNoz/signoz/pkg/types/telemetrytypes"
	"github.com/SigNoz/signoz/pkg/types/telemetrytypes/telemetrytypestest"
)

func TestLiteMetadataResolvesUnqualifiedFieldsForAdapter(t *testing.T) {
	metadataStore := telemetrytypestest.NewMockMetadataStore()
	metadataStore.SetKeys([]*telemetrytypes.TelemetryFieldKey{
		{Name: "host.name", Signal: telemetrytypes.SignalLogs, FieldContext: telemetrytypes.FieldContextResource, FieldDataType: telemetrytypes.FieldDataTypeString},
		{Name: "http.status_code", Signal: telemetrytypes.SignalLogs, FieldContext: telemetrytypes.FieldContextAttribute, FieldDataType: telemetrytypes.FieldDataTypeNumber},
	})
	request := &qbtypes.QueryRangeRequest{
		Start: 1_000, End: 61_000, RequestType: qbtypes.RequestTypeTimeSeries,
		CompositeQuery: qbtypes.CompositeQuery{Queries: []qbtypes.QueryEnvelope{{
			Type: qbtypes.QueryTypeBuilder,
			Spec: qbtypes.QueryBuilderQuery[qbtypes.LogAggregation]{
				Name: "A", Signal: telemetrytypes.SignalLogs, StepInterval: qbtypes.Step{Duration: time.Minute},
				Filter:       &qbtypes.Filter{Expression: "host.name = 'worker-1' AND http.status_code >= 500"},
				Aggregations: []qbtypes.LogAggregation{{Expression: "count()"}},
			},
		}}},
	}

	metadata, err := (&querier{metadataStore: metadataStore}).liteMetadata(context.Background(), request)
	if err != nil {
		t.Fatalf("liteMetadata() error = %v", err)
	}
	converted, err := liteadapter.ToLite(request, metadata)
	if err != nil {
		t.Fatalf("ToLite() error = %v", err)
	}
	query := converted.Queries[0].(litequery.LogQuery)
	filter := query.Common.Filter.(litequery.LogicalFilter)
	host := filter.Items[0].(litequery.Predicate).Field
	status := filter.Items[1].(litequery.Predicate).Field
	if host.Context != litequery.FieldContextResource || status.Context != litequery.FieldContextAttribute || status.Type != litequery.ValueTypeNumber {
		t.Fatalf("resolved fields = host %#v, status %#v", host, status)
	}
}
