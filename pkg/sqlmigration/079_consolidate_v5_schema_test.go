package sqlmigration

import (
	"context"
	"database/sql"
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
	_, err = db.ExecContext(ctx, `CREATE TABLE auth_domain (id TEXT PRIMARY KEY)`)
	require.NoError(t, err)
	_, err = db.ExecContext(ctx, `CREATE TABLE quick_filter (id TEXT PRIMARY KEY, org_id TEXT NOT NULL, filter TEXT NOT NULL, signal TEXT NOT NULL, created_at TIMESTAMP, updated_at TIMESTAMP)`)
	require.NoError(t, err)
	_, err = db.ExecContext(ctx, `INSERT INTO quick_filter (id, org_id, filter, signal) VALUES ('00000000-0000-0000-0000-000000000001', '00000000-0000-0000-0000-000000000002', '[{"key":"hasError"},{"key":"http.method"}]', 'traces')`)
	require.NoError(t, err)

	migration := &consolidateV5Schema{}
	require.NoError(t, migration.Up(ctx, db))
	require.NoError(t, migration.Up(ctx, db))

	count, err := db.NewSelect().
		TableExpr("sqlite_master").
		Where("type = 'table' AND name = 'auth_domain'").
		Count(ctx)
	require.NoError(t, err)
	require.Zero(t, count)

	var traceQuickFilter string
	require.NoError(t, db.NewSelect().Table("quick_filter").Column("filter").Where("id = ?", "00000000-0000-0000-0000-000000000001").Scan(ctx, &traceQuickFilter))
	require.JSONEq(t, `[{"key":"has_error"},{"key":"http_method"}]`, traceQuickFilter)
}
