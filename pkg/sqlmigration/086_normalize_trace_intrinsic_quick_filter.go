package sqlmigration

import (
	"context"
	"encoding/json"

	"github.com/SigNoz/signoz/pkg/factory"
	"github.com/SigNoz/signoz/pkg/types/quickfiltertypes"
	"github.com/uptrace/bun"
	"github.com/uptrace/bun/migrate"
)

var traceIntrinsicQuickFilterKeys = map[string]struct{}{
	"duration_nano":        {},
	"has_error":            {},
	"name":                 {},
	"response_status_code": {},
	"http_host":            {},
	"http_method":          {},
	"http_url":             {},
	"trace_id":             {},
}

// normalizeTraceIntrinsicQuickFilter replaces the legacy "tag" context used
// for physical span columns. Dynamic trace attributes remain tags.
type normalizeTraceIntrinsicQuickFilter struct{}

func NewNormalizeTraceIntrinsicQuickFilterFactory() factory.ProviderFactory[SQLMigration, Config] {
	return factory.NewProviderFactory(factory.MustNewName("trace_intrinsic_filter"), func(context.Context, factory.ProviderSettings, Config) (SQLMigration, error) {
		return &normalizeTraceIntrinsicQuickFilter{}, nil
	})
}

func (migration *normalizeTraceIntrinsicQuickFilter) Register(migrations *migrate.Migrations) error {
	return migrations.Register(migration.Up, migration.Down)
}

func (migration *normalizeTraceIntrinsicQuickFilter) Up(ctx context.Context, db *bun.DB) error {
	var rows []struct {
		ID     string `bun:"id"`
		Filter string `bun:"filter"`
	}
	if err := db.NewSelect().
		Table("quick_filter").
		Column("id", "filter").
		Where("signal = ?", quickfiltertypes.SignalTraces.StringValue()).
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
			key, ok := filter["key"].(string)
			if _, intrinsic := traceIntrinsicQuickFilterKeys[key]; ok && intrinsic && filter["type"] != "span" {
				filter["type"] = "span"
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

func (migration *normalizeTraceIntrinsicQuickFilter) Down(context.Context, *bun.DB) error {
	return nil
}
