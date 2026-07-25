package sqlmigration

import (
	"context"
	"database/sql"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/uptrace/bun"
	"github.com/uptrace/bun/dialect/sqlitedialect"
	"github.com/uptrace/bun/migrate"
	_ "modernc.org/sqlite"
)

type recordingMigration struct {
	calls *int
}

func (migration *recordingMigration) Register(*migrate.Migrations) error {
	return nil
}

func (migration *recordingMigration) Up(context.Context, *bun.DB) error {
	*migration.calls++
	return nil
}

func (migration *recordingMigration) Down(context.Context, *bun.DB) error {
	return nil
}

func TestV15Baseline(t *testing.T) {
	testCases := []struct {
		name          string
		existingV15DB bool
		expectedSteps int
	}{
		{name: "empty database", expectedSteps: 2},
		{name: "v1.5 database", existingV15DB: true, expectedSteps: 0},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			database, err := sql.Open("sqlite", ":memory:")
			require.NoError(t, err)
			t.Cleanup(func() { require.NoError(t, database.Close()) })

			db := bun.NewDB(database, sqlitedialect.New())
			t.Cleanup(func() { require.NoError(t, db.Close()) })
			ctx := context.Background()
			if testCase.existingV15DB {
				_, err = db.ExecContext(ctx, `CREATE TABLE organizations (id TEXT PRIMARY KEY)`)
				require.NoError(t, err)
			}

			stepCalls := 0
			consolidateCalls := 0
			migration := &v15Baseline{
				steps: []SQLMigration{
					&recordingMigration{calls: &stepCalls},
					&recordingMigration{calls: &stepCalls},
				},
				consolidate: &recordingMigration{calls: &consolidateCalls},
			}

			require.NoError(t, migration.Up(ctx, db))
			require.Equal(t, testCase.expectedSteps, stepCalls)
			require.Equal(t, 1, consolidateCalls)
		})
	}
}
