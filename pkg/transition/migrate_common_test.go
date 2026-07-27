package transition

import (
	"context"
	"log/slog"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestUpdateQueryDataCanonicalizesLegacyTraceFields(t *testing.T) {
	migration := NewMigrateCommon(slog.Default())
	query := map[string]any{
		"dataSource":        "traces",
		"aggregateOperator": "p90",
		"aggregateAttribute": map[string]any{
			"key": "durationNano",
		},
		"groupBy": []any{map[string]any{
			"key":  "serviceName",
			"type": "resource",
		}},
		"selectColumns": []any{map[string]any{
			"key": "httpRoute",
		}},
		"orderBy": []any{map[string]any{
			"columnName": "durationNano",
			"order":      "desc",
		}},
		"filters": map[string]any{
			"op": "AND",
			"items": []any{map[string]any{
				"key":   map[string]any{"key": "httpRoute"},
				"op":    "=",
				"value": "/checkout",
			}},
		},
	}

	require.True(t, migration.canonicalizeTraceFields(query))
	require.Equal(t, "duration_nano", query["aggregateAttribute"].(map[string]any)["key"])
	require.Equal(t, "service.name", query["groupBy"].([]any)[0].(map[string]any)["key"])
	require.Equal(t, "http.route", query["selectColumns"].([]any)[0].(map[string]any)["key"])
	require.Equal(t, "duration_nano", query["orderBy"].([]any)[0].(map[string]any)["columnName"])
	require.Equal(t, "http.route", query["filters"].(map[string]any)["items"].([]any)[0].(map[string]any)["key"].(map[string]any)["key"])

	require.True(t, migration.updateQueryData(context.Background(), query, "v4", "table"))

	aggregations := query["aggregations"].([]any)
	require.Equal(t, "p90(duration_nano)", aggregations[0].(map[string]any)["expression"])
	require.Equal(t, "http.route = '/checkout' AND service.name EXISTS", query["filter"].(map[string]any)["expression"])
}

func TestCanonicalizeTraceFieldExpression(t *testing.T) {
	canonicalize := func(value string) string { return canonicalTraceFieldName(value) }
	require.Equal(t, "p90(duration_nano) AND http.route = 'httpRoute'", canonicalizeTraceFieldExpression("p90(durationNano) AND httpRoute = 'httpRoute'", canonicalize))
}
