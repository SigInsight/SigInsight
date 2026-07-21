package sqlmigration

import (
	"context"
	"testing"

	"database/sql"

	"github.com/stretchr/testify/require"
	"github.com/uptrace/bun"
	"github.com/uptrace/bun/dialect/sqlitedialect"
	_ "modernc.org/sqlite"
)

func TestDropLogPipelines(t *testing.T) {
	database, err := sql.Open("sqlite", ":memory:")
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, database.Close()) })
	db := bun.NewDB(database, sqlitedialect.New())
	t.Cleanup(func() { require.NoError(t, db.Close()) })

	ctx := context.Background()
	statements := []string{
		`CREATE TABLE agent_config_version (id TEXT PRIMARY KEY, element_type TEXT NOT NULL)`,
		`CREATE TABLE agent_config_element (id TEXT PRIMARY KEY, element_type TEXT NOT NULL, version_id TEXT NOT NULL)`,
		`CREATE TABLE pipelines (id TEXT PRIMARY KEY)`,
		`INSERT INTO agent_config_version VALUES ('pipeline-version', 'log_pipelines'), ('sampling-version', 'sampling_rules')`,
		`INSERT INTO agent_config_element VALUES ('pipeline-element', 'log_pipelines', 'pipeline-version'), ('sampling-element', 'sampling_rules', 'sampling-version')`,
		`INSERT INTO pipelines VALUES ('pipeline')`,
	}
	for _, statement := range statements {
		_, err := db.ExecContext(ctx, statement)
		require.NoError(t, err)
	}

	migration := &dropLogPipelines{}
	require.NoError(t, migration.Up(ctx, db))

	count, err := db.NewSelect().Table("agent_config_version").Where("element_type = ?", logPipelinesElementType).Count(ctx)
	require.NoError(t, err)
	require.Zero(t, count)
	count, err = db.NewSelect().Table("agent_config_element").Where("element_type = ?", logPipelinesElementType).Count(ctx)
	require.NoError(t, err)
	require.Zero(t, count)
	count, err = db.NewSelect().Table("agent_config_version").Where("element_type = ?", "sampling_rules").Count(ctx)
	require.NoError(t, err)
	require.Equal(t, 1, count)

	count, err = db.NewSelect().TableExpr("sqlite_master").Where("type = 'table' AND name = 'pipelines'").Count(ctx)
	require.NoError(t, err)
	require.Zero(t, count)
}
