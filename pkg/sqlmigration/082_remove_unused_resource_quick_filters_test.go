package sqlmigration

import (
	"context"
	"database/sql"
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/uptrace/bun"
	"github.com/uptrace/bun/dialect/sqlitedialect"
	_ "modernc.org/sqlite"
)

func TestRemoveUnusedResourceQuickFilters(t *testing.T) {
	database, err := sql.Open("sqlite", ":memory:")
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, database.Close()) })

	db := bun.NewDB(database, sqlitedialect.New())
	t.Cleanup(func() { require.NoError(t, db.Close()) })

	ctx := context.Background()
	_, err = db.ExecContext(ctx, `CREATE TABLE quick_filter (id TEXT PRIMARY KEY, signal TEXT NOT NULL, filter TEXT NOT NULL)`)
	require.NoError(t, err)

	rows := [][]any{
		{"exceptions", "exceptions", `[{"key":"service.name","custom":"preserved"},{"key":"k8s.cluster.name"},{"key":"deployment.environment"},{"key":"host.name"}]`},
		{"logs", "logs", `[{"key":"k8s.cluster.name"},{"key":"service.name"},{"key":"k8s.pod.name"}]`},
		{"traces", "traces", `[{"key":"deployment.environment"},{"key":"trace_id"}]`},
	}
	for _, args := range rows {
		_, err := db.ExecContext(ctx, `INSERT INTO quick_filter (id, signal, filter) VALUES (?, ?, ?)`, args...)
		require.NoError(t, err)
	}

	migration := &removeUnusedResourceQuickFilters{}
	require.NoError(t, migration.Up(ctx, db))
	require.NoError(t, migration.Up(ctx, db))

	var storedRows []struct {
		ID     string `bun:"id"`
		Filter string `bun:"filter"`
	}
	require.NoError(t, db.NewSelect().Table("quick_filter").Column("id", "filter").Order("id").Scan(ctx, &storedRows))
	require.Len(t, storedRows, 3)

	filtersByID := map[string][]map[string]any{}
	for _, row := range storedRows {
		var filters []map[string]any
		require.NoError(t, json.Unmarshal([]byte(row.Filter), &filters))
		filtersByID[row.ID] = filters
	}
	assert.Equal(t, []map[string]any{
		{"key": "service.name", "custom": "preserved"},
		{"key": "host.name"},
	}, filtersByID["exceptions"])
	assert.Equal(t, []map[string]any{{"key": "service.name"}}, filtersByID["logs"])
	assert.Equal(t, []map[string]any{{"key": "trace_id"}}, filtersByID["traces"])
}
