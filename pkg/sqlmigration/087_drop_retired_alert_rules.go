package sqlmigration

import (
	"context"
	"encoding/json"

	"github.com/SigNoz/signoz/pkg/factory"
	"github.com/SigNoz/signoz/pkg/types/ruletypes"
	"github.com/uptrace/bun"
	"github.com/uptrace/bun/migrate"
)

// dropRetiredAlertRules removes persisted rules that cannot be consumed by the
// v3-only alert API. This is a destructive boundary by design: legacy alert
// JSON has no conversion path in the lightweight alert engine.
type dropRetiredAlertRules struct{}

func NewDropRetiredAlertRulesFactory() factory.ProviderFactory[SQLMigration, Config] {
	return factory.NewProviderFactory(factory.MustNewName("drop_retired_alert_rules"), func(context.Context, factory.ProviderSettings, Config) (SQLMigration, error) {
		return &dropRetiredAlertRules{}, nil
	})
}

func (migration *dropRetiredAlertRules) Register(migrations *migrate.Migrations) error {
	return migrations.Register(migration.Up, migration.Down)
}

func (migration *dropRetiredAlertRules) Up(ctx context.Context, db *bun.DB) error {
	var rows []struct {
		ID   string `bun:"id"`
		Data string `bun:"data"`
	}
	if err := db.NewSelect().Table("rule").Column("id", "data").Scan(ctx, &rows); err != nil {
		return err
	}

	for _, row := range rows {
		var rule ruletypes.PostableRule
		if err := json.Unmarshal([]byte(row.Data), &rule); err == nil {
			continue
		}
		if _, err := db.NewDelete().Table("rule").Where("id = ?", row.ID).Exec(ctx); err != nil {
			return err
		}
	}

	return nil
}

func (migration *dropRetiredAlertRules) Down(context.Context, *bun.DB) error { return nil }
