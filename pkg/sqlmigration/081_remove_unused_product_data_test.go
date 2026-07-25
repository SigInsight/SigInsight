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

func TestRemoveUnusedProductData(t *testing.T) {
	database, err := sql.Open("sqlite", ":memory:")
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, database.Close()) })

	db := bun.NewDB(database, sqlitedialect.New())
	t.Cleanup(func() { require.NoError(t, db.Close()) })

	ctx := context.Background()
	for _, statement := range []string{
		`CREATE TABLE org_preference (name TEXT NOT NULL)`,
		`CREATE TABLE tuple (object_id TEXT NOT NULL)`,
		`CREATE TABLE changelog (object_id TEXT NOT NULL)`,
		`CREATE TABLE factor_api_key (id TEXT PRIMARY KEY)`,
		`CREATE TABLE service_account_role (id TEXT PRIMARY KEY)`,
		`CREATE TABLE service_account (id TEXT PRIMARY KEY)`,
		`CREATE TABLE cloud_integration_service (id TEXT PRIMARY KEY)`,
		`CREATE TABLE cloud_integrations_service_configs (id TEXT PRIMARY KEY)`,
		`CREATE TABLE cloud_integrations_accounts (id TEXT PRIMARY KEY)`,
		`CREATE TABLE cloud_integration (id TEXT PRIMARY KEY)`,
		`CREATE TABLE installed_integration (id TEXT PRIMARY KEY)`,
		`INSERT INTO org_preference VALUES ('org_onboarding'), ('keep')`,
		`INSERT INTO tuple VALUES ('organization/a/serviceaccount/b'), ('organization/a/role/admin')`,
		`INSERT INTO changelog VALUES ('organization/a/serviceaccount/b'), ('organization/a/role/admin')`,
	} {
		_, err := db.ExecContext(ctx, statement)
		require.NoError(t, err)
	}

	migration := &removeUnusedProductData{}
	require.NoError(t, migration.Up(ctx, db))
	require.NoError(t, migration.Up(ctx, db))

	for _, table := range []string{
		"factor_api_key", "service_account_role", "service_account",
		"cloud_integration_service", "cloud_integrations_service_configs",
		"cloud_integrations_accounts", "cloud_integration", "installed_integration",
	} {
		count, err := db.NewSelect().TableExpr("sqlite_master").
			Where("type = 'table' AND name = ?", table).Count(ctx)
		require.NoError(t, err)
		require.Zero(t, count)
	}

	for _, table := range []string{"org_preference", "tuple", "changelog"} {
		count, err := db.NewSelect().Table(table).Count(ctx)
		require.NoError(t, err)
		require.Equal(t, 1, count)
	}
}

func TestRemoveUnusedProductDataWithoutOptionalTables(t *testing.T) {
	database, err := sql.Open("sqlite", ":memory:")
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, database.Close()) })

	db := bun.NewDB(database, sqlitedialect.New())
	t.Cleanup(func() { require.NoError(t, db.Close()) })

	require.NoError(t, (&removeUnusedProductData{}).Up(context.Background(), db))
}
