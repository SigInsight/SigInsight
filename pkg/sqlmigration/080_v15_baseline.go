package sqlmigration

import (
	"context"

	"github.com/SigNoz/signoz/pkg/factory"
	"github.com/uptrace/bun"
	"github.com/uptrace/bun/migrate"
)

// v15Baseline initializes new installations through the final v1.5 schema.
// Existing v1.5+ databases already contain that schema, so only the
// consolidated post-baseline data migration is applied to them.
type v15Baseline struct {
	steps       []SQLMigration
	consolidate SQLMigration
}

func NewV15BaselineFactory(
	stepFactories []factory.ProviderFactory[SQLMigration, Config],
	consolidateFactory factory.ProviderFactory[SQLMigration, Config],
) factory.ProviderFactory[SQLMigration, Config] {
	return factory.NewProviderFactory(
		factory.MustNewName("v15_baseline"),
		func(ctx context.Context, settings factory.ProviderSettings, config Config) (SQLMigration, error) {
			steps := make([]SQLMigration, 0, len(stepFactories))
			for _, stepFactory := range stepFactories {
				step, err := stepFactory.New(ctx, settings, config)
				if err != nil {
					return nil, err
				}
				steps = append(steps, step)
			}

			consolidate, err := consolidateFactory.New(ctx, settings, config)
			if err != nil {
				return nil, err
			}

			return &v15Baseline{steps: steps, consolidate: consolidate}, nil
		},
	)
}

func (migration *v15Baseline) Register(migrations *migrate.Migrations) error {
	return migrations.Register(migration.Up, migration.Down)
}

func (migration *v15Baseline) Up(ctx context.Context, db *bun.DB) error {
	tableCount, err := db.NewSelect().
		TableExpr("sqlite_master").
		Where("type = 'table' AND name = ?", "organizations").
		Count(ctx)
	if err != nil {
		return err
	}

	if tableCount == 0 {
		for _, step := range migration.steps {
			if err := step.Up(ctx, db); err != nil {
				return err
			}
		}
	}

	return migration.consolidate.Up(ctx, db)
}

func (migration *v15Baseline) Down(context.Context, *bun.DB) error {
	return nil
}
