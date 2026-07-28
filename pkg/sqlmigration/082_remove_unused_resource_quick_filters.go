package sqlmigration

import (
	"context"
	"encoding/json"
	"strings"

	"github.com/SigNoz/signoz/pkg/factory"
	"github.com/uptrace/bun"
	"github.com/uptrace/bun/migrate"
)

type removeUnusedResourceQuickFilters struct{}

func NewRemoveUnusedResourceQuickFiltersFactory() factory.ProviderFactory[SQLMigration, Config] {
	return factory.NewProviderFactory(factory.MustNewName("remove_unused_resource_filters"), func(context.Context, factory.ProviderSettings, Config) (SQLMigration, error) {
		return &removeUnusedResourceQuickFilters{}, nil
	})
}

func (migration *removeUnusedResourceQuickFilters) Register(migrations *migrate.Migrations) error {
	return migrations.Register(migration.Up, migration.Down)
}

func (migration *removeUnusedResourceQuickFilters) Up(ctx context.Context, db *bun.DB) error {
	var rows []struct {
		ID     string `bun:"id"`
		Filter string `bun:"filter"`
	}
	if err := db.NewSelect().
		Table("quick_filter").
		Column("id", "filter").
		Scan(ctx, &rows); err != nil {
		return err
	}

	for _, row := range rows {
		var filters []json.RawMessage
		if err := json.Unmarshal([]byte(row.Filter), &filters); err != nil {
			return err
		}

		filtered := make([]json.RawMessage, 0, len(filters))
		for _, filter := range filters {
			var identity struct {
				Key string `json:"key"`
			}
			if err := json.Unmarshal(filter, &identity); err != nil {
				return err
			}
			if identity.Key == "deployment.environment" || strings.HasPrefix(identity.Key, "k8s.") {
				continue
			}
			filtered = append(filtered, filter)
		}
		if len(filtered) == len(filters) {
			continue
		}

		filterJSON, err := json.Marshal(filtered)
		if err != nil {
			return err
		}
		if _, err := db.NewUpdate().
			Table("quick_filter").
			Set("filter = ?", string(filterJSON)).
			Where("id = ?", row.ID).
			Exec(ctx); err != nil {
			return err
		}
	}
	return nil
}

func (migration *removeUnusedResourceQuickFilters) Down(context.Context, *bun.DB) error {
	return nil
}
