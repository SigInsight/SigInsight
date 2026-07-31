package sqlmigration

import (
	"context"
	"encoding/json"

	"github.com/SigNoz/signoz/pkg/factory"
	"github.com/SigNoz/signoz/pkg/types/ruletypes"
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

func (migration *splitAlertUnits) Up(ctx context.Context, db *bun.DB) error {
	var rows []struct {
		ID   string `bun:"id"`
		Data string `bun:"data"`
	}
	if err := db.NewSelect().Table("rule").Column("id", "data").Scan(ctx, &rows); err != nil {
		return err
	}

	for _, row := range rows {
		var rule ruletypes.PostableRule
		if err := json.Unmarshal([]byte(row.Data), &rule); err != nil {
			continue
		}
		encoded, err := json.Marshal(&rule)
		if err != nil {
			return err
		}
		if string(encoded) == row.Data {
			continue
		}
		if _, err := db.NewUpdate().
			Table("rule").
			Set("data = ?", string(encoded)).
			Where("id = ?", row.ID).
			Exec(ctx); err != nil {
			return err
		}
	}

	return nil
}

func (migration *splitAlertUnits) Down(context.Context, *bun.DB) error { return nil }
