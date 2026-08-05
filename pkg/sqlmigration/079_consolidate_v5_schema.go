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

// consolidateV5Schema applies all schema and stored-query changes introduced
// after the v1.5 baseline. Each operation is idempotent so databases upgraded
// through the former individual migrations remain supported.
type consolidateV5Schema struct {
}

func NewConsolidateV5SchemaFactory() factory.ProviderFactory[SQLMigration, Config] {
	return factory.NewProviderFactory(
		factory.MustNewName("consolidate_v5_schema"),
		func(_ context.Context, _ factory.ProviderSettings, _ Config) (SQLMigration, error) {
			return &consolidateV5Schema{}, nil
		},
	)
}

func (migration *consolidateV5Schema) Register(migrations *migrate.Migrations) error {
	return migrations.Register(migration.Up, migration.Down)
}

func (migration *consolidateV5Schema) Up(ctx context.Context, db *bun.DB) error {
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()

	if _, err := tx.NewDropTable().IfExists().Table("auth_domain").Exec(ctx); err != nil {
		return err
	}

	var quickFilters []*quickfiltertypes.StorableQuickFilter
	if err := tx.NewSelect().Model(&quickFilters).Where("signal = ?", quickfiltertypes.SignalTraces).Scan(ctx); err != nil {
		return err
	}
	for _, quickFilter := range quickFilters {
		updatedFilter, changed, err := migrateTraceQuickFilterFields(quickFilter.Filter)
		if err != nil {
			return err
		}
		if !changed {
			continue
		}
		if _, err := tx.NewUpdate().Model(quickFilter).
			Set("filter = ?, updated_at = ?", updatedFilter, time.Now()).
			WherePK().
			Exec(ctx); err != nil {
			return err
		}
	}

	return tx.Commit()
}

func migrateTraceQuickFilterFields(filterJSON string) (string, bool, error) {
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
		switch key {
		case "hasError":
			filters[idx]["key"] = json.RawMessage(`"has_error"`)
			changed = true
		case "http.method":
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

func (migration *consolidateV5Schema) Down(context.Context, *bun.DB) error {
	return nil
}
