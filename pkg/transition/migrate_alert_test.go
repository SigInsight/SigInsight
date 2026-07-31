package transition

import (
	"context"
	"log/slog"
	"testing"

	"github.com/stretchr/testify/require"
)

func alertRuleWithQueries(names ...string) map[string]any {
	queries := make([]any, 0, len(names))
	for _, name := range names {
		queries = append(queries, map[string]any{
			"type": "builder_query",
			"spec": map[string]any{"name": name},
		})
	}
	return map[string]any{
		"version": "v5",
		"condition": map[string]any{
			"compositeQuery": map[string]any{"queries": queries},
		},
	}
}

func TestEnsureAlertSelectedQuery(t *testing.T) {
	rule := alertRuleWithQueries("A", "B")
	require.True(t, EnsureAlertSelectedQuery(rule))
	require.Equal(t, "B", rule["condition"].(map[string]any)["selectedQueryName"])
	require.False(t, EnsureAlertSelectedQuery(rule))

	rule = alertRuleWithQueries("A", "F1", "Z")
	require.True(t, EnsureAlertSelectedQuery(rule))
	require.Equal(t, "F1", rule["condition"].(map[string]any)["selectedQueryName"])

	require.False(t, EnsureAlertSelectedQuery(map[string]any{}))
}

func TestAlertMigrateV5DiscardsPromQL(t *testing.T) {
	rule := map[string]any{
		"version": "v4",
		"condition": map[string]any{
			"compositeQuery": map[string]any{
				"queryType": "promql",
				"promQueries": map[string]any{
					"A": map[string]any{"query": "up", "disabled": false},
				},
			},
		},
	}
	migrator := NewAlertMigrateV5(slog.Default(), nil, nil)

	require.True(t, migrator.Migrate(context.Background(), rule))
	require.Equal(t, "v5", rule["version"])

	condition := rule["condition"].(map[string]any)
	require.NotContains(t, condition, "selectedQueryName")
	compositeQuery := condition["compositeQuery"].(map[string]any)
	require.NotContains(t, compositeQuery, "promQueries")
	require.Empty(t, compositeQuery["queries"])

	require.False(t, migrator.Migrate(context.Background(), rule))
}
