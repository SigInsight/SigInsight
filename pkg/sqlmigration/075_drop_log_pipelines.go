package sqlmigration

import (
	"context"

	"github.com/SigNoz/signoz/pkg/factory"
	"github.com/uptrace/bun"
	"github.com/uptrace/bun/migrate"
)

const logPipelinesElementType = "log_pipelines"

type dropLogPipelines struct{}

func NewDropLogPipelinesFactory() factory.ProviderFactory[SQLMigration, Config] {
	return factory.NewProviderFactory(factory.MustNewName("drop_log_pipelines"), func(context.Context, factory.ProviderSettings, Config) (SQLMigration, error) {
		return &dropLogPipelines{}, nil
	})
}

func (migration *dropLogPipelines) Register(migrations *migrate.Migrations) error {
	return migrations.Register(migration.Up, migration.Down)
}

func (migration *dropLogPipelines) Up(ctx context.Context, db *bun.DB) error {
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()

	if _, err := tx.NewDelete().
		Table("agent_config_element").
		Where("element_type = ?", logPipelinesElementType).
		Exec(ctx); err != nil {
		return err
	}

	if _, err := tx.NewDelete().
		Table("agent_config_version").
		Where("element_type = ?", logPipelinesElementType).
		Exec(ctx); err != nil {
		return err
	}

	if _, err := tx.NewDropTable().
		IfExists().
		Table("pipelines").
		Exec(ctx); err != nil {
		return err
	}

	return tx.Commit()
}

func (migration *dropLogPipelines) Down(context.Context, *bun.DB) error {
	return nil
}
