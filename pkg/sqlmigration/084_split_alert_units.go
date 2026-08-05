package sqlmigration

import (
	"context"

	"github.com/SigNoz/signoz/pkg/factory"
	"github.com/uptrace/bun"
	"github.com/uptrace/bun/migrate"
)

type splitAlertUnits struct{}

func NewSplitAlertUnitsFactory() factory.ProviderFactory[SQLMigration, Config] {
	return factory.NewProviderFactory(factory.MustNewName("split_alert_units"), func(context.Context, factory.ProviderSettings, Config) (SQLMigration, error) {
		return &splitAlertUnits{}, nil
	})
}

func (migration *splitAlertUnits) Register(migrations *migrate.Migrations) error {
	return migrations.Register(migration.Up, migration.Down)
}

// Up remains registered so databases that already recorded this migration keep
// a complete migration history. The old unit-splitting conversion accepted the
// retired alert JSON shape and is intentionally no longer performed.
func (migration *splitAlertUnits) Up(context.Context, *bun.DB) error { return nil }

func (migration *splitAlertUnits) Down(context.Context, *bun.DB) error { return nil }
