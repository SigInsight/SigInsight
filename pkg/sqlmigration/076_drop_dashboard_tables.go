package sqlmigration

import (
	"context"

	"github.com/SigNoz/signoz/pkg/factory"
	"github.com/uptrace/bun"
	"github.com/uptrace/bun/migrate"
)

const publicDashboardObjectPattern = "organization/%/public-dashboard/%"

type dropDashboardTables struct{}

func NewDropDashboardTablesFactory() factory.ProviderFactory[SQLMigration, Config] {
	return factory.NewProviderFactory(
		factory.MustNewName("drop_dashboard_tables"),
		func(context.Context, factory.ProviderSettings, Config) (SQLMigration, error) {
			return &dropDashboardTables{}, nil
		},
	)
}

func (migration *dropDashboardTables) Register(migrations *migrate.Migrations) error {
	return migrations.Register(migration.Up, migration.Down)
}

func (migration *dropDashboardTables) Up(ctx context.Context, db *bun.DB) error {
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()

	for _, table := range []string{"tuple", "changelog"} {
		if _, err := tx.NewDelete().
			Table(table).
			Where("object_id LIKE ?", publicDashboardObjectPattern).
			Exec(ctx); err != nil {
			return err
		}
	}

	for _, table := range []string{"public_dashboard", "dashboard", "dashboards"} {
		if _, err := tx.NewDropTable().IfExists().Table(table).Exec(ctx); err != nil {
			return err
		}
	}

	return tx.Commit()
}

func (migration *dropDashboardTables) Down(context.Context, *bun.DB) error {
	return nil
}