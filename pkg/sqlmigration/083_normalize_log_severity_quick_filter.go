package sqlmigration

import (
	"context"
	"encoding/json"

	"github.com/SigNoz/signoz/pkg/factory"
	"github.com/SigNoz/signoz/pkg/types/quickfiltertypes"
	"github.com/uptrace/bun"
	"github.com/uptrace/bun/migrate"
)

// normalizeLogSeverityQuickFilter corrects the historical resource context for
// severity_text. Severity is an intrinsic log field, not a resource attribute.
type normalizeLogSeverityQuickFilter struct{}

func NewNormalizeLogSeverityQuickFilterFactory() factory.ProviderFactory[SQLMigration, Config] {
	return factory.NewProviderFactory(factory.MustNewName("normalize_log_severity_filter"), func(context.Context, factory.ProviderSettings, Config) (SQLMigration, error) {
		return &normalizeLogSeverityQuickFilter{}, nil
	})
}

func (migration *normalizeLogSeverityQuickFilter) Register(migrations *migrate.Migrations) error {
	return migrations.Register(migration.Up, migration.Down)
}

func (migration *normalizeLogSeverityQuickFilter) Up(ctx context.Context, db *bun.DB) error {
	var rows []struct {
		ID     string `bun:"id"`
		Filter string `bun:"filter"`
	}
	if err := db.NewSelect().
		Table("quick_filter").
		Column("id", "filter").
		Where("signal = ?", quickfiltertypes.SignalLogs.StringValue()).
		Scan(ctx, &rows); err != nil {
		return err
	}

	for _, row := range rows {
		var filters []map[string]any
		if err := json.Unmarshal([]byte(row.Filter), &filters); err != nil {
			return err
		}

		changed := false
		for _, filter := range filters {
			if filter["key"] == "severity_text" && filter["type"] == "resource" {
				filter["type"] = ""
				changed = true
			}
		}
		if !changed {
			continue
		}

		filterJSON, err := json.Marshal(filters)
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

func (migration *normalizeLogSeverityQuickFilter) Down(context.Context, *bun.DB) error { return nil }
