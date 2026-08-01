package querier

import (
	"context"
	"strings"
	"testing"

	qbtypes "github.com/SigNoz/signoz/pkg/types/querybuildertypes/querybuildertypesv5"
	"github.com/SigNoz/signoz/pkg/valuer"
)

func TestQueryRangeRejectsRetiredTraceOperatorBeforeExecution(t *testing.T) {
	query := &querier{}
	_, err := query.QueryRange(context.Background(), valuer.UUID{}, &qbtypes.QueryRangeRequest{
		CompositeQuery: qbtypes.CompositeQuery{Queries: []qbtypes.QueryEnvelope{{
			Type: qbtypes.QueryTypeTraceOperator,
			Spec: qbtypes.QueryBuilderTraceOperator{
				Name:       "TO1",
				Expression: "A -> B",
			},
		}}},
	})
	if err == nil || !strings.Contains(err.Error(), "trace operator queries are no longer supported") {
		t.Fatalf("QueryRange() error = %v, want retired trace operator error", err)
	}
}

func TestQueryRangeRejectsUnsupportedCapabilityWhenLightweightEnabled(t *testing.T) {
	query := &querier{}
	_, err := query.QueryRange(context.Background(), valuer.UUID{}, &qbtypes.QueryRangeRequest{
		RequestType: qbtypes.RequestTypeTimeSeries,
		CompositeQuery: qbtypes.CompositeQuery{Queries: []qbtypes.QueryEnvelope{{
			Type: qbtypes.QueryTypeClickHouseSQL,
			Spec: qbtypes.ClickHouseQuery{Name: "A", Query: "SELECT 1"},
		}}},
	})
	if err == nil || !strings.Contains(err.Error(), "unsupported lightweight query capability: V5 query type clickhouse_sql") {
		t.Fatalf("QueryRange() error = %v, want lightweight capability error", err)
	}
}
