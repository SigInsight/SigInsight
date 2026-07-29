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

func TestNormalizeLogSeverityQuickFilter(t *testing.T) {
	database, err := sql.Open("sqlite", ":memory:")
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, database.Close()) })

	db := bun.NewDB(database, sqlitedialect.New())
	t.Cleanup(func() { require.NoError(t, db.Close()) })
	ctx := context.Background()
	_, err = db.ExecContext(ctx, `CREATE TABLE quick_filter (id TEXT PRIMARY KEY, signal TEXT NOT NULL, filter TEXT NOT NULL)`)
	require.NoError(t, err)

	rows := [][]any{
		{"logs-wrong", "logs", `[{"key":"severity_text","dataType":"string","type":"resource","custom":"preserved"},{"key":"service.name","dataType":"string","type":"resource"}]`},
		{"logs-correct", "logs", `[{"key":"severity_text","dataType":"string","type":""}]`},
		{"traces", "traces", `[{"key":"severity_text","dataType":"string","type":"resource"}]`},
	}
	for _, args := range rows {
		_, err = db.ExecContext(ctx, `INSERT INTO quick_filter (id, signal, filter) VALUES (?, ?, ?)`, args...)
		require.NoError(t, err)
	}

	migration := &normalizeLogSeverityQuickFilter{}
	require.NoError(t, migration.Up(ctx, db))
	require.NoError(t, migration.Up(ctx, db))

	var stored []struct {
		ID     string `bun:"id"`
		Filter string `bun:"filter"`
	}
	require.NoError(t, db.NewSelect().Table("quick_filter").Column("id", "filter").Order("id").Scan(ctx, &stored))
	require.Len(t, stored, 3)

	filtersByID := map[string][]map[string]any{}
	for _, row := range stored {
		var filters []map[string]any
		require.NoError(t, json.Unmarshal([]byte(row.Filter), &filters))
		filtersByID[row.ID] = filters
	}
	require.Equal(t, []map[string]any{
		{"key": "severity_text", "dataType": "string", "type": "", "custom": "preserved"},
		{"key": "service.name", "dataType": "string", "type": "resource"},
	}, filtersByID["logs-wrong"])
	require.Equal(t, []map[string]any{{"key": "severity_text", "dataType": "string", "type": ""}}, filtersByID["logs-correct"])
	require.Equal(t, []map[string]any{{"key": "severity_text", "dataType": "string", "type": "resource"}}, filtersByID["traces"])
}
