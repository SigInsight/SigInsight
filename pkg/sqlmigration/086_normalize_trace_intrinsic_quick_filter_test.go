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

func TestNormalizeTraceIntrinsicQuickFilter(t *testing.T) {
	database, err := sql.Open("sqlite", ":memory:")
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, database.Close()) })

	db := bun.NewDB(database, sqlitedialect.New())
	t.Cleanup(func() { require.NoError(t, db.Close()) })
	ctx := context.Background()
	_, err = db.ExecContext(ctx, `CREATE TABLE quick_filter (id TEXT PRIMARY KEY, signal TEXT NOT NULL, filter TEXT NOT NULL)`)
	require.NoError(t, err)

	rows := [][]any{
		{"traces", "traces", `[{"key":"name","dataType":"string","type":"tag","custom":"preserved"},{"key":"has_error","dataType":"bool","type":""},{"key":"rpc.method","dataType":"string","type":"tag"},{"key":"service.name","dataType":"string","type":"resource"}]`},
		{"logs", "logs", `[{"key":"name","dataType":"string","type":"tag"}]`},
	}
	for _, args := range rows {
		_, err = db.ExecContext(ctx, `INSERT INTO quick_filter (id, signal, filter) VALUES (?, ?, ?)`, args...)
		require.NoError(t, err)
	}

	migration := &normalizeTraceIntrinsicQuickFilter{}
	require.NoError(t, migration.Up(ctx, db))
	require.NoError(t, migration.Up(ctx, db))

	var stored []struct {
		ID     string `bun:"id"`
		Filter string `bun:"filter"`
	}
	require.NoError(t, db.NewSelect().Table("quick_filter").Column("id", "filter").Order("id").Scan(ctx, &stored))
	require.Len(t, stored, 2)

	filtersByID := map[string][]map[string]any{}
	for _, row := range stored {
		var filters []map[string]any
		require.NoError(t, json.Unmarshal([]byte(row.Filter), &filters))
		filtersByID[row.ID] = filters
	}
	require.Equal(t, []map[string]any{{"key": "name", "dataType": "string", "type": "tag"}}, filtersByID["logs"])
	require.Equal(t, []map[string]any{
		{"key": "name", "dataType": "string", "type": "span", "custom": "preserved"},
		{"key": "has_error", "dataType": "bool", "type": "span"},
		{"key": "rpc.method", "dataType": "string", "type": "tag"},
		{"key": "service.name", "dataType": "string", "type": "resource"},
	}, filtersByID["traces"])
}
