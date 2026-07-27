package sqlmigration

import (
	"context"
	"strings"

	"github.com/SigNoz/signoz/pkg/factory"
	"github.com/uptrace/bun"
	"github.com/uptrace/bun/migrate"
)

// removeUnusedProductData drops storage belonging to features removed from the
// community product after the v1.5 baseline was introduced.
type removeUnusedProductData struct{}

func NewRemoveUnusedProductDataFactory() factory.ProviderFactory[SQLMigration, Config] {
	return factory.NewProviderFactory(factory.MustNewName("remove_unused_product_data"), func(context.Context, factory.ProviderSettings, Config) (SQLMigration, error) {
		return &removeUnusedProductData{}, nil
	})
}

func (migration *removeUnusedProductData) Register(migrations *migrate.Migrations) error {
	return migrations.Register(migration.Up, migration.Down)
}

func (migration *removeUnusedProductData) Up(ctx context.Context, db *bun.DB) error {
	for _, table := range []string{"org_preference", "tuple", "changelog"} {
		var exists bool
		if err := db.NewSelect().
			TableExpr("sqlite_master").
			ColumnExpr("COUNT(*) > 0").
			Where("type = 'table' AND name = ?", table).
			Scan(ctx, &exists); err != nil {
			return err
		}
		if !exists {
			continue
		}

		var statement string
		switch table {
		case "org_preference":
			statement = `DELETE FROM org_preference WHERE name = 'org_onboarding'`
		default:
			statement = "DELETE FROM " + table + " WHERE object_id LIKE '%/serviceaccount/%'"
		}
		if _, err := db.ExecContext(ctx, statement); err != nil {
			return err
		}
	}

	for _, statement := range []string{
		`DROP TABLE IF EXISTS public_dashboard`,
		`DROP TABLE IF EXISTS dashboard`,
		`DROP TABLE IF EXISTS dashboards`,
		`DROP TABLE IF EXISTS planned_maintenance_rule`,
		`DROP TABLE IF EXISTS planned_maintenance`,
		`DROP TABLE IF EXISTS route_policy`,
		`DROP TABLE IF EXISTS virtual_field`,
		`DROP TABLE IF EXISTS license`,
		`DROP TABLE IF EXISTS user_invite`,
		`DROP TABLE IF EXISTS factor_api_key`,
		`DROP TABLE IF EXISTS service_account_role`,
		`DROP TABLE IF EXISTS service_account`,
		`DROP TABLE IF EXISTS cloud_integration_service`,
		`DROP TABLE IF EXISTS cloud_integrations_service_configs`,
		`DROP TABLE IF EXISTS cloud_integrations_accounts`,
		`DROP TABLE IF EXISTS cloud_integration`,
		`DROP TABLE IF EXISTS installed_integration`,
	} {
		if _, err := db.ExecContext(ctx, statement); err != nil {
			return err
		}
	}
	if _, err := db.ExecContext(ctx, `DELETE FROM notification_channel WHERE type NOT IN ('email', 'webhook')`); err != nil {
		// notification_channel was created after the v1.5 baseline. It may not
		// exist in a partially initialized database.
		if !isMissingTableError(err) {
			return err
		}
	}
	return nil
}

func (migration *removeUnusedProductData) Down(context.Context, *bun.DB) error { return nil }

func isMissingTableError(err error) bool {
	return strings.Contains(err.Error(), "no such table")
}
