package sqlmigration

import (
	"context"
	"database/sql"
	"sort"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/uptrace/bun"
	"github.com/uptrace/bun/dialect/sqlitedialect"
	"github.com/uptrace/bun/migrate"
	_ "modernc.org/sqlite"
)

type recordingMigration struct{ calls *int }

func (migration *recordingMigration) Register(*migrate.Migrations) error { return nil }
func (migration *recordingMigration) Up(context.Context, *bun.DB) error {
	*migration.calls++
	return nil
}
func (migration *recordingMigration) Down(context.Context, *bun.DB) error { return nil }

func newBaselineTestDB(t *testing.T) *bun.DB {
	t.Helper()
	database, err := sql.Open("sqlite", ":memory:")
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, database.Close()) })
	db := bun.NewDB(database, sqlitedialect.New())
	t.Cleanup(func() { require.NoError(t, db.Close()) })
	return db
}

func TestV15BaselineCreatesFinalSchema(t *testing.T) {
	db := newBaselineTestDB(t)
	ctx := context.Background()
	consolidateCalls := 0
	migration := &v15Baseline{consolidate: &recordingMigration{calls: &consolidateCalls}}

	require.NoError(t, migration.Up(ctx, db))
	require.NoError(t, migration.Up(ctx, db), "baseline must be idempotent")
	require.Equal(t, 2, consolidateCalls)

	var tables []string
	require.NoError(t, db.NewSelect().TableExpr("sqlite_master").Column("name").Where("type = 'table' AND name NOT LIKE 'sqlite_%'").Order("name").Scan(ctx, &tables))
	expectedTables := []string{
		"agent", "agent_config_element", "agent_config_version", "alertmanager_config", "alertmanager_state",
		"apdex_setting", "assertion", "assistant_config", "auth_token", "authorization_model", "changelog",
		"factor_password", "notification_channel", "org_preference", "organizations", "quick_filter",
		"reset_password_token", "role", "rule", "saved_views", "store", "trace_funnel", "ttl_setting",
		"tuple", "user_preference", "user_role", "users",
	}
	sort.Strings(expectedTables)
	require.Equal(t, expectedTables, tables)

	for table, expectedColumns := range map[string][]string{
		"organizations": {"id", "display_name", "updated_at", "created_at", "name", "alias", "key"},
		"users":         {"created_at", "updated_at", "id", "display_name", "email", "org_id", "is_root", "status", "deleted_at"},
		"rule":          {"id", "created_at", "updated_at", "created_by", "updated_by", "deleted", "data", "org_id"},
		"saved_views":   {"id", "name", "category", "created_by", "updated_by", "source_page", "tags", "data", "extra_data", "created_at", "updated_at", "org_id"},
		"ttl_setting":   {"id", "created_at", "updated_at", "transaction_id", "table_name", "ttl", "cold_storage_ttl", "status", "org_id", "condition"},
	} {
		rows, err := db.QueryContext(ctx, "SELECT name FROM pragma_table_info(?) ORDER BY cid", table)
		require.NoError(t, err)
		var columns []string
		for rows.Next() {
			var column string
			require.NoError(t, rows.Scan(&column))
			columns = append(columns, column)
		}
		require.NoError(t, rows.Err())
		require.NoError(t, rows.Close())
		require.Equal(t, expectedColumns, columns, table)
	}

	var foreignKeyViolations int
	require.NoError(t, db.NewRaw("SELECT COUNT(*) FROM pragma_foreign_key_check").Scan(ctx, &foreignKeyViolations))
	require.Zero(t, foreignKeyViolations)

	for _, index := range []string{
		"uq_users_org_id_email_deleted_at", "uq_factor_password_user_id", "uq_auth_token_access_token",
		"uq_role_name_org_id", "uq_user_role_user_id_role_id", "uq_org_preference_name_org_id",
		"uq_user_preference_name_user_id", "uq_tuple_ulid",
	} {
		count, err := db.NewSelect().TableExpr("sqlite_master").Where("type = 'index' AND name = ?", index).Count(ctx)
		require.NoError(t, err)
		require.Equal(t, 1, count, index)
	}
}

func TestV15BaselineSkipsExistingDatabase(t *testing.T) {
	db := newBaselineTestDB(t)
	ctx := context.Background()
	_, err := db.ExecContext(ctx, `CREATE TABLE organizations (id TEXT PRIMARY KEY)`)
	require.NoError(t, err)

	consolidateCalls := 0
	migration := &v15Baseline{consolidate: &recordingMigration{calls: &consolidateCalls}}
	require.NoError(t, migration.Up(ctx, db))
	require.Equal(t, 1, consolidateCalls)

	count, err := db.NewSelect().TableExpr("sqlite_master").Where("type = 'table' AND name = 'users'").Count(ctx)
	require.NoError(t, err)
	require.Zero(t, count)
}
