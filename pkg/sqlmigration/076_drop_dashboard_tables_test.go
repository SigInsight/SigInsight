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

func TestDropDashboardTables(t *testing.T) {
	database, err := sql.Open("sqlite", ":memory:")
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, database.Close()) })

	db := bun.NewDB(database, sqlitedialect.New())
	t.Cleanup(func() { require.NoError(t, db.Close()) })

	ctx := context.Background()
	for _, statement := range []string{
		`CREATE TABLE dashboard (id TEXT PRIMARY KEY)`,
		`CREATE TABLE dashboards (id TEXT PRIMARY KEY)`,
		`CREATE TABLE public_dashboard (id TEXT PRIMARY KEY, dashboard_id TEXT NOT NULL)`,
		`CREATE TABLE tuple (object_id TEXT NOT NULL)`,
		`CREATE TABLE changelog (object_id TEXT NOT NULL)`,
		`INSERT INTO dashboard VALUES ('dashboard-id')`,
		`INSERT INTO dashboards VALUES ('legacy-dashboard-id')`,
		`INSERT INTO public_dashboard VALUES ('public-dashboard-id', 'dashboard-id')`,
		`INSERT INTO tuple VALUES ('organization/org-id/public-dashboard/public-dashboard-id'), ('organization/org-id/role/admin')`,
		`INSERT INTO changelog VALUES ('organization/org-id/public-dashboard/*'), ('organization/org-id/role/admin')`,
	} {
		_, err := db.ExecContext(ctx, statement)
		require.NoError(t, err)
	}

	migration := &dropDashboardTables{}
	require.NoError(t, migration.Up(ctx, db))

	for _, table := range []string{"dashboard", "dashboards", "public_dashboard"} {
		count, err := db.NewSelect().
			TableExpr("sqlite_master").
			Where("type = 'table' AND name = ?", table).
			Count(ctx)
		require.NoError(t, err)
		require.Zero(t, count)
	}

	for _, table := range []string{"tuple", "changelog"} {
		count, err := db.NewSelect().
			Table(table).
			Where("object_id LIKE ?", publicDashboardObjectPattern).
			Count(ctx)
		require.NoError(t, err)
		require.Zero(t, count)

		count, err = db.NewSelect().Table(table).Count(ctx)
		require.NoError(t, err)
		require.Equal(t, 1, count)
	}
}