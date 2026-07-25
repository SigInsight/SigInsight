package sqlmigration

import (
	"context"
	"database/sql"
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/uptrace/bun"
	"github.com/uptrace/bun/dialect/sqlitedialect"
	_ "modernc.org/sqlite"
)

func TestConsolidateV5Schema(t *testing.T) {
	database, err := sql.Open("sqlite", ":memory:")
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, database.Close()) })

	db := bun.NewDB(database, sqlitedialect.New())
	t.Cleanup(func() { require.NoError(t, db.Close()) })

	ctx := context.Background()
	_, err = db.ExecContext(ctx, `CREATE TABLE saved_views (id TEXT PRIMARY KEY, data TEXT NOT NULL)`)
	require.NoError(t, err)
	_, err = db.ExecContext(ctx, `CREATE TABLE auth_domain (id TEXT PRIMARY KEY)`)
	require.NoError(t, err)

	testCases := []struct {
		id            string
		data          string
		expectedType  string
		expectedCount int
	}{
		{"legacy", `{"version":"v4","panelType":"table"}`, "", 0},
		{"current", `{"version":"v5","panelType":"table","queries":[{"type":"clickhouse_sql","spec":{"name":"A","query":"SELECT 1"}}]}`, "clickhouse_sql", 1},
		{"builder", `{"version":"v4","builderQueries":{"A":{}}}`, "builder_formula", 1},
		{"promql", `{"version":"v4","promQueries":{"A":{"query":"up","disabled":false}}}`, "promql", 1},
		{"clickhouse", `{"version":"v4","chQueries":{"A":{"query":"SELECT 1","disabled":false}}}`, "clickhouse_sql", 1},
	}
	for _, testCase := range testCases {
		_, err = db.ExecContext(ctx, "INSERT INTO saved_views (id, data) VALUES (?, ?)", testCase.id, testCase.data)
		require.NoError(t, err)
	}

	migration := &consolidateV5Schema{}
	require.NoError(t, migration.Up(ctx, db))
	require.NoError(t, migration.Up(ctx, db))

	for _, testCase := range testCases {
		t.Run(testCase.id, func(t *testing.T) {
			var raw string
			require.NoError(t, db.NewSelect().Table("saved_views").Column("data").Where("id = ?", testCase.id).Scan(ctx, &raw))

			var data map[string]any
			require.NoError(t, json.Unmarshal([]byte(raw), &data))
			require.NotContains(t, data, "version")
			require.NotContains(t, data, "builderQueries")
			require.NotContains(t, data, "promQueries")
			require.NotContains(t, data, "chQueries")

			queries, _ := data["queries"].([]any)
			require.Len(t, queries, testCase.expectedCount)
			if testCase.expectedCount > 0 {
				envelope, ok := queries[0].(map[string]any)
				require.True(t, ok)
				require.Equal(t, testCase.expectedType, envelope["type"])
			}
		})
	}

	count, err := db.NewSelect().
		TableExpr("sqlite_master").
		Where("type = 'table' AND name = 'auth_domain'").
		Count(ctx)
	require.NoError(t, err)
	require.Zero(t, count)
}
