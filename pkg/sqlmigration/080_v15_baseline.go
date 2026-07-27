package sqlmigration

import (
	"context"

	"github.com/SigNoz/signoz/pkg/factory"
	"github.com/uptrace/bun"
	"github.com/uptrace/bun/migrate"
)

// v15Schema is the complete SQLite schema required by the current product.
// It intentionally excludes product surfaces removed after v1.5.
var v15Schema = []string{
	`CREATE TABLE IF NOT EXISTS organizations (id TEXT NOT NULL PRIMARY KEY, display_name TEXT NOT NULL, updated_at TIMESTAMP, created_at TIMESTAMP, name TEXT, alias TEXT, key BIGINT)`,
	`CREATE UNIQUE INDEX IF NOT EXISTS idx_unique_alias ON organizations (alias)`,
	`CREATE UNIQUE INDEX IF NOT EXISTS idx_unique_key ON organizations (key)`,
	`CREATE UNIQUE INDEX IF NOT EXISTS idx_unique_name ON organizations (name)`,
	`CREATE TABLE IF NOT EXISTS users (created_at TIMESTAMP, updated_at TIMESTAMP, id TEXT NOT NULL PRIMARY KEY, display_name TEXT NOT NULL, email TEXT NOT NULL, org_id TEXT NOT NULL, is_root BOOLEAN NOT NULL, status TEXT NOT NULL, deleted_at TIMESTAMP NOT NULL, FOREIGN KEY (org_id) REFERENCES organizations (id))`,
	`CREATE UNIQUE INDEX IF NOT EXISTS uq_users_org_id_email_deleted_at ON users (org_id, email, deleted_at)`,
	`CREATE TABLE IF NOT EXISTS factor_password (id TEXT NOT NULL PRIMARY KEY, created_at TIMESTAMP, updated_at TIMESTAMP, password TEXT NOT NULL, temporary BOOLEAN NOT NULL, user_id TEXT NOT NULL, FOREIGN KEY (user_id) REFERENCES users (id))`,
	`CREATE UNIQUE INDEX IF NOT EXISTS uq_factor_password_user_id ON factor_password (user_id)`,
	`CREATE TABLE IF NOT EXISTS reset_password_token (id TEXT NOT NULL PRIMARY KEY, token TEXT NOT NULL, password_id TEXT NOT NULL, expires_at TIMESTAMP, FOREIGN KEY (password_id) REFERENCES factor_password (id))`,
	`CREATE UNIQUE INDEX IF NOT EXISTS uq_reset_password_token_password_id ON reset_password_token (password_id)`,
	`CREATE UNIQUE INDEX IF NOT EXISTS uq_reset_password_token_token ON reset_password_token (token)`,
	`CREATE TABLE IF NOT EXISTS auth_token (id TEXT NOT NULL PRIMARY KEY, meta TEXT NOT NULL, prev_access_token TEXT, access_token TEXT NOT NULL, prev_refresh_token TEXT, refresh_token TEXT NOT NULL, last_observed_at TIMESTAMP, rotated_at TIMESTAMP, created_at TIMESTAMP NOT NULL, updated_at TIMESTAMP NOT NULL, user_id TEXT NOT NULL, FOREIGN KEY (user_id) REFERENCES users (id))`,
	`CREATE UNIQUE INDEX IF NOT EXISTS uq_auth_token_access_token ON auth_token (access_token)`,
	`CREATE UNIQUE INDEX IF NOT EXISTS uq_auth_token_refresh_token ON auth_token (refresh_token)`,
	`CREATE TABLE IF NOT EXISTS role (id TEXT NOT NULL PRIMARY KEY, created_at TIMESTAMP NOT NULL, updated_at TIMESTAMP NOT NULL, name TEXT NOT NULL, description TEXT, type TEXT NOT NULL, org_id TEXT NOT NULL, FOREIGN KEY (org_id) REFERENCES organizations (id))`,
	`CREATE UNIQUE INDEX IF NOT EXISTS uq_role_name_org_id ON role (name, org_id)`,
	`CREATE TABLE IF NOT EXISTS user_role (id TEXT NOT NULL PRIMARY KEY, user_id TEXT NOT NULL, role_id TEXT NOT NULL, created_at TIMESTAMP NOT NULL, updated_at TIMESTAMP NOT NULL, FOREIGN KEY (user_id) REFERENCES users (id), FOREIGN KEY (role_id) REFERENCES role (id))`,
	`CREATE UNIQUE INDEX IF NOT EXISTS uq_user_role_user_id_role_id ON user_role (user_id, role_id)`,
	`CREATE TABLE IF NOT EXISTS org_preference (id TEXT NOT NULL PRIMARY KEY, name TEXT NOT NULL, value TEXT NOT NULL, org_id TEXT NOT NULL, FOREIGN KEY (org_id) REFERENCES organizations (id))`,
	`CREATE UNIQUE INDEX IF NOT EXISTS uq_org_preference_name_org_id ON org_preference (name, org_id)`,
	`CREATE TABLE IF NOT EXISTS user_preference (id TEXT NOT NULL PRIMARY KEY, name TEXT NOT NULL, value TEXT NOT NULL, user_id TEXT NOT NULL, FOREIGN KEY (user_id) REFERENCES users (id))`,
	`CREATE UNIQUE INDEX IF NOT EXISTS uq_user_preference_name_user_id ON user_preference (name, user_id)`,
	`CREATE TABLE IF NOT EXISTS quick_filter (id TEXT NOT NULL PRIMARY KEY, org_id TEXT NOT NULL, filter TEXT NOT NULL, signal TEXT NOT NULL, created_at TIMESTAMP, updated_at TIMESTAMP, created_by TEXT, updated_by TEXT, CONSTRAINT org_id_signal UNIQUE (org_id, signal), FOREIGN KEY (org_id) REFERENCES organizations (id) ON DELETE CASCADE ON UPDATE CASCADE)`,
	`CREATE TABLE IF NOT EXISTS saved_views (id TEXT NOT NULL PRIMARY KEY, name TEXT NOT NULL, category TEXT NOT NULL, created_by TEXT, updated_by TEXT, source_page TEXT NOT NULL, tags TEXT, data TEXT NOT NULL, extra_data TEXT, created_at TIMESTAMP, updated_at TIMESTAMP, org_id TEXT REFERENCES organizations (id) ON DELETE CASCADE)`,
	`CREATE TABLE IF NOT EXISTS rule (id TEXT NOT NULL PRIMARY KEY, created_at TIMESTAMP, updated_at TIMESTAMP, created_by TEXT, updated_by TEXT, deleted INTEGER NOT NULL DEFAULT 0, data TEXT NOT NULL, org_id TEXT, FOREIGN KEY (org_id) REFERENCES organizations (id))`,
	`CREATE TABLE IF NOT EXISTS notification_channel (id TEXT NOT NULL PRIMARY KEY, created_at TIMESTAMP, updated_at TIMESTAMP, name VARCHAR, type VARCHAR, data VARCHAR, org_id VARCHAR, FOREIGN KEY (org_id) REFERENCES organizations (id))`,
	`CREATE TABLE IF NOT EXISTS alertmanager_config (id TEXT NOT NULL PRIMARY KEY, created_at TIMESTAMP, updated_at TIMESTAMP, config TEXT NOT NULL, hash TEXT NOT NULL, org_id VARCHAR NOT NULL UNIQUE, FOREIGN KEY (org_id) REFERENCES organizations (id))`,
	`CREATE TABLE IF NOT EXISTS alertmanager_state (id TEXT NOT NULL PRIMARY KEY, created_at TIMESTAMP, updated_at TIMESTAMP, silences TEXT, nflog TEXT, org_id VARCHAR NOT NULL UNIQUE, FOREIGN KEY (org_id) REFERENCES organizations (id))`,
	`CREATE TABLE IF NOT EXISTS apdex_setting (id TEXT NOT NULL PRIMARY KEY, org_id TEXT, service_name TEXT, threshold FLOAT NOT NULL, exclude_status_codes TEXT NOT NULL, FOREIGN KEY (org_id) REFERENCES organizations (id))`,
	`CREATE UNIQUE INDEX IF NOT EXISTS apdex_setting_unique_idx ON apdex_setting (service_name, org_id)`,
	`CREATE TABLE IF NOT EXISTS trace_funnel (id TEXT NOT NULL PRIMARY KEY, created_at TIMESTAMP, updated_at TIMESTAMP, created_by TEXT, updated_by TEXT, name TEXT NOT NULL, description TEXT, org_id VARCHAR NOT NULL, steps TEXT NOT NULL, tags TEXT, FOREIGN KEY (org_id) REFERENCES organizations (id) ON DELETE CASCADE)`,
	`CREATE TABLE IF NOT EXISTS ttl_setting (id TEXT NOT NULL PRIMARY KEY, created_at TIMESTAMP, updated_at TIMESTAMP, transaction_id TEXT NOT NULL, table_name TEXT NOT NULL, ttl INTEGER NOT NULL DEFAULT 0, cold_storage_ttl INTEGER NOT NULL DEFAULT 0, status TEXT NOT NULL, org_id VARCHAR NOT NULL, condition TEXT, FOREIGN KEY (org_id) REFERENCES organizations (id))`,
	`CREATE TABLE IF NOT EXISTS agent (id TEXT NOT NULL PRIMARY KEY, created_at TIMESTAMP, updated_at TIMESTAMP, agent_id TEXT NOT NULL UNIQUE, org_id TEXT NOT NULL, terminated_at TIMESTAMP, status TEXT NOT NULL, config TEXT NOT NULL)`,
	`CREATE TABLE IF NOT EXISTS agent_config_version (id TEXT NOT NULL PRIMARY KEY, created_at TIMESTAMP, updated_at TIMESTAMP, created_by TEXT, updated_by TEXT, org_id TEXT NOT NULL, version INTEGER, element_type TEXT NOT NULL, deploy_status TEXT NOT NULL DEFAULT 'dirty', deploy_sequence INTEGER, deploy_result TEXT, hash TEXT, config TEXT, CONSTRAINT element_version_org_idx UNIQUE (org_id, version, element_type), FOREIGN KEY (org_id) REFERENCES organizations (id))`,
	`CREATE TABLE IF NOT EXISTS agent_config_element (id TEXT NOT NULL PRIMARY KEY, created_at TIMESTAMP, updated_at TIMESTAMP, element_id TEXT NOT NULL, element_type TEXT NOT NULL, version_id TEXT NOT NULL, CONSTRAINT element_type_version_idx UNIQUE (element_id, element_type, version_id), FOREIGN KEY (version_id) REFERENCES agent_config_version (id))`,
	`CREATE TABLE IF NOT EXISTS assistant_config (org_id TEXT NOT NULL PRIMARY KEY, base_url TEXT NOT NULL, model TEXT NOT NULL, api_key TEXT NOT NULL, created_at TIMESTAMP NOT NULL, updated_at TIMESTAMP NOT NULL, FOREIGN KEY (org_id) REFERENCES organizations (id))`,
	`CREATE TABLE IF NOT EXISTS store (id TEXT NOT NULL PRIMARY KEY, name TEXT NOT NULL, created_at TIMESTAMP NOT NULL, updated_at TIMESTAMP, deleted_at TIMESTAMP)`,
	`CREATE TABLE IF NOT EXISTS authorization_model (store TEXT NOT NULL, authorization_model_id TEXT NOT NULL, schema_version TEXT NOT NULL DEFAULT '1.1', serialized_protobuf TEXT NOT NULL, PRIMARY KEY (store, authorization_model_id))`,
	`CREATE TABLE IF NOT EXISTS assertion (store TEXT NOT NULL, authorization_model_id TEXT NOT NULL, assertions TEXT, PRIMARY KEY (store, authorization_model_id))`,
	`CREATE TABLE IF NOT EXISTS tuple (store TEXT NOT NULL, object_type TEXT NOT NULL, object_id TEXT NOT NULL, relation TEXT NOT NULL, user_object_type TEXT NOT NULL, user_object_id TEXT NOT NULL, user_relation TEXT NOT NULL, user_type TEXT NOT NULL, ulid TEXT NOT NULL, inserted_at TIMESTAMP NOT NULL, condition_name TEXT, condition_context TEXT, PRIMARY KEY (store, object_type, object_id, relation, user_object_type, user_object_id, user_relation))`,
	`CREATE UNIQUE INDEX IF NOT EXISTS uq_tuple_ulid ON tuple (ulid)`,
	`CREATE TABLE IF NOT EXISTS changelog (store TEXT NOT NULL, object_type TEXT NOT NULL, object_id TEXT NOT NULL, relation TEXT NOT NULL, user_object_type TEXT NOT NULL, user_object_id TEXT NOT NULL, user_relation TEXT NOT NULL, operation TEXT NOT NULL, ulid TEXT NOT NULL, inserted_at TIMESTAMP NOT NULL, condition_name TEXT, condition_context TEXT, PRIMARY KEY (store, ulid, object_type))`,
}

type v15Baseline struct {
	consolidate SQLMigration
}

func NewV15BaselineFactory(consolidateFactory factory.ProviderFactory[SQLMigration, Config]) factory.ProviderFactory[SQLMigration, Config] {
	return factory.NewProviderFactory(factory.MustNewName("v15_baseline"), func(ctx context.Context, settings factory.ProviderSettings, config Config) (SQLMigration, error) {
		consolidate, err := consolidateFactory.New(ctx, settings, config)
		if err != nil {
			return nil, err
		}
		return &v15Baseline{consolidate: consolidate}, nil
	})
}

func (migration *v15Baseline) Register(migrations *migrate.Migrations) error {
	return migrations.Register(migration.Up, migration.Down)
}

func (migration *v15Baseline) Up(ctx context.Context, db *bun.DB) error {
	tableCount, err := db.NewSelect().TableExpr("sqlite_master").Where("type = 'table' AND name = 'organizations'").Count(ctx)
	if err != nil {
		return err
	}
	if tableCount == 0 {
		if err := db.RunInTx(ctx, nil, func(ctx context.Context, tx bun.Tx) error {
			for _, statement := range v15Schema {
				if _, err := tx.ExecContext(ctx, statement); err != nil {
					return err
				}
			}
			return nil
		}); err != nil {
			return err
		}
	}

	return migration.consolidate.Up(ctx, db)
}

func (migration *v15Baseline) Down(context.Context, *bun.DB) error { return nil }
