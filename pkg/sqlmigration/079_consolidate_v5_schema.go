package sqlmigration

import (
	"context"
	"database/sql"
	"encoding/json"
	"log/slog"

	"github.com/SigNoz/signoz/pkg/factory"
	"github.com/SigNoz/signoz/pkg/transition"
	"github.com/uptrace/bun"
	"github.com/uptrace/bun/migrate"
)

// consolidateV5Schema applies all schema and stored-query changes introduced
// after the v1.5 baseline. Each operation is idempotent so databases upgraded
// through the former individual migrations remain supported.
type consolidateV5Schema struct {
	logger *slog.Logger
}

func NewConsolidateV5SchemaFactory() factory.ProviderFactory[SQLMigration, Config] {
	return factory.NewProviderFactory(
		factory.MustNewName("consolidate_v5_schema"),
		func(_ context.Context, settings factory.ProviderSettings, _ Config) (SQLMigration, error) {
			return &consolidateV5Schema{logger: settings.Logger}, nil
		},
	)
}

func (migration *consolidateV5Schema) Register(migrations *migrate.Migrations) error {
	return migrations.Register(migration.Up, migration.Down)
}

func (migration *consolidateV5Schema) Up(ctx context.Context, db *bun.DB) error {
	if migration.logger == nil {
		migration.logger = slog.Default()
	}
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()

	var savedViews []struct {
		ID   string `bun:"id"`
		Data string `bun:"data"`
	}
	if err := tx.NewSelect().Table("saved_views").Column("id", "data").Scan(ctx, &savedViews); err != nil {
		if err == sql.ErrNoRows {
			return nil
		}
		return err
	}

	migrator := transition.NewSavedViewMigrateV5(migration.logger, nil, nil)
	for _, savedView := range savedViews {
		var data map[string]any
		if err := json.Unmarshal([]byte(savedView.Data), &data); err != nil {
			migration.logger.WarnContext(ctx, "skipping invalid saved view data", slog.String("saved_view_id", savedView.ID))
			continue
		}

		migrator.Migrate(ctx, data)
		delete(data, "version")
		updatedData, err := json.Marshal(data)
		if err != nil {
			return err
		}
		if _, err := tx.NewUpdate().Table("saved_views").Set("data = ?", string(updatedData)).Where("id = ?", savedView.ID).Exec(ctx); err != nil {
			return err
		}
	}

	if _, err := tx.NewDropTable().IfExists().Table("auth_domain").Exec(ctx); err != nil {
		return err
	}

	return tx.Commit()
}

func (migration *consolidateV5Schema) Down(context.Context, *bun.DB) error {
	return nil
}
