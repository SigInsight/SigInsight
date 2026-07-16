package sqlmigration

import (
	"context"
	"encoding/json"
	"time"

	"github.com/SigNoz/signoz/pkg/factory"
	"github.com/SigNoz/signoz/pkg/types/quickfiltertypes"
	"github.com/uptrace/bun"
	"github.com/uptrace/bun/migrate"
)

type updateTraceHTTPMethodQuickFilter struct{}

func NewUpdateTraceHTTPMethodQuickFilterFactory() factory.ProviderFactory[SQLMigration, Config] {
	return factory.NewProviderFactory(factory.MustNewName("update_trace_http_method_filter"), func(context.Context, factory.ProviderSettings, Config) (SQLMigration, error) {
		return &updateTraceHTTPMethodQuickFilter{}, nil
	})
}

func (migration *updateTraceHTTPMethodQuickFilter) Register(migrations *migrate.Migrations) error {
	return migrations.Register(migration.Up, migration.Down)
}

func (migration *updateTraceHTTPMethodQuickFilter) Up(ctx context.Context, db *bun.DB) error {
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()

	var filters []*quickfiltertypes.StorableQuickFilter
	if err := tx.NewSelect().
		Model(&filters).
		Where("signal = ?", quickfiltertypes.SignalTraces).
		Scan(ctx); err != nil {
		return err
	}

	for _, filter := range filters {
		updatedFilter, changed, err := migrateTraceHTTPMethodQuickFilter(filter.Filter)
		if err != nil {
			return err
		}
		if !changed {
			continue
		}

		if _, err := tx.NewUpdate().
			Model(filter).
			Set("filter = ?, updated_at = ?", updatedFilter, time.Now()).
			WherePK().
			Exec(ctx); err != nil {
			return err
		}
	}

	return tx.Commit()
}

func (migration *updateTraceHTTPMethodQuickFilter) Down(context.Context, *bun.DB) error {
	return nil
}

func migrateTraceHTTPMethodQuickFilter(filterJSON string) (string, bool, error) {
	var filters []map[string]json.RawMessage
	if err := json.Unmarshal([]byte(filterJSON), &filters); err != nil {
		return "", false, err
	}

	changed := false
	for idx := range filters {
		rawKey, ok := filters[idx]["key"]
		if !ok {
			continue
		}

		var key string
		if err := json.Unmarshal(rawKey, &key); err != nil {
			return "", false, err
		}
		if key == "http.method" {
			filters[idx]["key"] = json.RawMessage(`"http_method"`)
			changed = true
		}
	}
	if !changed {
		return filterJSON, false, nil
	}

	updatedFilter, err := json.Marshal(filters)
	if err != nil {
		return "", false, err
	}

	return string(updatedFilter), true, nil
}
