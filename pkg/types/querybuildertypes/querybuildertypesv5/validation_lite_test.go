package querybuildertypesv5

import (
	"testing"

	"github.com/SigNoz/signoz/pkg/types/telemetrytypes"
	"github.com/SigNoz/signoz/pkg/valuer"
	"github.com/stretchr/testify/require"
)

func liteLogEnvelope(disabled bool) QueryEnvelope {
	return QueryEnvelope{
		Type: QueryTypeBuilder,
		Spec: QueryBuilderQuery[LogAggregation]{
			Name:         "A",
			Signal:       telemetrytypes.SignalLogs,
			Disabled:     disabled,
			Aggregations: []LogAggregation{{Expression: "count()"}},
		},
	}
}

func TestValidateQueryEnvelopeSupportedLiteTypes(t *testing.T) {
	require.NoError(t, validateQueryEnvelope(liteLogEnvelope(false)))
	require.NoError(t, validateQueryEnvelope(QueryEnvelope{
		Type: QueryTypeFormula,
		Spec: QueryBuilderFormula{Name: "F1", Expression: "A"},
	}))
}

func TestValidateQueryEnvelopeRejectsRetiredType(t *testing.T) {
	err := validateQueryEnvelope(QueryEnvelope{
		Type: QueryType{String: valuer.NewString("builder_trace_operator")},
		Spec: map[string]any{"name": "T"},
	})
	require.ErrorContains(t, err, "unknown query type")
}

func TestCompositeQueryRejectsDuplicateBuilderNames(t *testing.T) {
	query := CompositeQuery{Queries: []QueryEnvelope{
		liteLogEnvelope(false),
		liteLogEnvelope(false),
	}}
	require.ErrorContains(t, query.Validate(), "duplicate query name")
}

func TestQueryRangeRequestRequiresEnabledQuery(t *testing.T) {
	request := QueryRangeRequest{
		Start:          1,
		End:            2,
		RequestType:    RequestTypeTimeSeries,
		CompositeQuery: CompositeQuery{Queries: []QueryEnvelope{liteLogEnvelope(true)}},
	}
	require.ErrorContains(t, request.Validate(), "all queries are disabled")

	request.CompositeQuery.Queries[0] = liteLogEnvelope(false)
	require.NoError(t, request.Validate())
}

func TestFormulaRequiresExpression(t *testing.T) {
	err := validateQueryEnvelope(QueryEnvelope{
		Type: QueryTypeFormula,
		Spec: QueryBuilderFormula{Name: "F1"},
	})
	require.ErrorContains(t, err, "formula expression is required")
}
