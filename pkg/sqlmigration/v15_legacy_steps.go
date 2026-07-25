package sqlmigration

import (
	"context"
	"database/sql"
	"database/sql/driver"
	"encoding/json"
	"fmt"
	"github.com/SigNoz/signoz/pkg/alertmanager/alertmanagerserver"
	"github.com/SigNoz/signoz/pkg/errors"
	"github.com/SigNoz/signoz/pkg/factory"
	"github.com/SigNoz/signoz/pkg/sqlschema"
	"github.com/SigNoz/signoz/pkg/sqlstore"
	"github.com/SigNoz/signoz/pkg/telemetrystore"
	"github.com/SigNoz/signoz/pkg/transition"
	"github.com/SigNoz/signoz/pkg/types"
	"github.com/SigNoz/signoz/pkg/types/alertmanagertypes"
	"github.com/SigNoz/signoz/pkg/types/authtypes"
	"github.com/SigNoz/signoz/pkg/types/opamptypes"
	"github.com/SigNoz/signoz/pkg/types/quickfiltertypes"
	"github.com/SigNoz/signoz/pkg/types/ruletypes"
	"github.com/SigNoz/signoz/pkg/types/telemetrytypes"
	"github.com/SigNoz/signoz/pkg/valuer"
	"github.com/google/uuid"
	"github.com/oklog/ulid/v2"
	"github.com/tidwall/gjson"
	"github.com/uptrace/bun"
	"github.com/uptrace/bun/dialect"
	"github.com/uptrace/bun/migrate"
	"hash/fnv"
	"log/slog"
	"reflect"
	"strconv"
	"time"
)

type addDataMigrations struct{}

func NewAddDataMigrationsFactory() factory.ProviderFactory[SQLMigration, Config] {
	return factory.NewProviderFactory(factory.MustNewName("add_data_migrations"), newAddDataMigrations)
}

func newAddDataMigrations(_ context.Context, _ factory.ProviderSettings, _ Config) (SQLMigration, error) {
	return &addDataMigrations{}, nil
}

func (migration *addDataMigrations) Register(migrations *migrate.Migrations) error {
	if err := migrations.Register(migration.Up, migration.Down); err != nil {
		return err
	}
	return nil
}

func (migration *addDataMigrations) Up(ctx context.Context, db *bun.DB) error {
	// table:data_migrations
	if _, err := db.NewCreateTable().
		Model(&struct {
			bun.BaseModel `bun:"table:data_migrations"`
			ID            int       `bun:"id,pk,autoincrement"`
			Version       string    `bun:"version,unique,notnull,type:VARCHAR(255)"`
			CreatedAt     time.Time `bun:"created_at,notnull,default:current_timestamp"`
			Succeeded     bool      `bun:"succeeded,notnull,default:false"`
		}{}).
		IfNotExists().
		Exec(ctx); err != nil {
		return err
	}

	return nil
}

func (migration *addDataMigrations) Down(ctx context.Context, db *bun.DB) error {
	return nil
}

type addOrganization struct{}

func NewAddOrganizationFactory() factory.ProviderFactory[SQLMigration, Config] {
	return factory.NewProviderFactory(factory.MustNewName("add_organization"), newAddOrganization)
}

func newAddOrganization(_ context.Context, _ factory.ProviderSettings, _ Config) (SQLMigration, error) {
	return &addOrganization{}, nil
}

func (migration *addOrganization) Register(migrations *migrate.Migrations) error {
	if err := migrations.Register(migration.Up, migration.Down); err != nil {
		return err
	}

	return nil
}

func (migration *addOrganization) Up(ctx context.Context, db *bun.DB) error {

	// table:organizations
	if _, err := db.NewCreateTable().
		Model(&struct {
			bun.BaseModel   `bun:"table:organizations"`
			ID              string `bun:"id,pk,type:text"`
			Name            string `bun:"name,type:text,notnull"`
			CreatedAt       int    `bun:"created_at,notnull"`
			IsAnonymous     int    `bun:"is_anonymous,notnull,default:0"`
			HasOptedUpdates int    `bun:"has_opted_updates,notnull,default:1"`
		}{}).
		IfNotExists().
		Exec(ctx); err != nil {
		return err
	}

	// table:groups
	if _, err := db.NewCreateTable().
		Model(&struct {
			bun.BaseModel `bun:"table:groups"`
			ID            string `bun:"id,pk,type:text"`
			Name          string `bun:"name,type:text,notnull,unique"`
		}{}).
		IfNotExists().
		Exec(ctx); err != nil {
		return err
	}

	// table:users
	if _, err := db.NewCreateTable().
		Model(&struct {
			bun.BaseModel     `bun:"table:users"`
			ID                string `bun:"id,pk,type:text"`
			Name              string `bun:"name,type:text,notnull"`
			Email             string `bun:"email,type:text,notnull,unique"`
			Password          string `bun:"password,type:text,notnull"`
			CreatedAt         int    `bun:"created_at,notnull"`
			ProfilePictureURL string `bun:"profile_picture_url,type:text"`
			GroupID           string `bun:"group_id,type:text,notnull"`
			OrgID             string `bun:"org_id,type:text,notnull"`
		}{}).
		ForeignKey(`("org_id") REFERENCES "organizations" ("id")`).
		ForeignKey(`("group_id") REFERENCES "groups" ("id")`).
		IfNotExists().
		Exec(ctx); err != nil {
		return err
	}

	// table:invites
	if _, err := db.NewCreateTable().
		Model(&struct {
			bun.BaseModel `bun:"table:invites"`
			ID            int    `bun:"id,pk,autoincrement"`
			Name          string `bun:"name,type:text,notnull"`
			Email         string `bun:"email,type:text,notnull,unique"`
			Token         string `bun:"token,type:text,notnull"`
			CreatedAt     int    `bun:"created_at,notnull"`
			Role          string `bun:"role,type:text,notnull"`
			OrgID         string `bun:"org_id,type:text,notnull"`
		}{}).
		ForeignKey(`("org_id") REFERENCES "organizations" ("id")`).
		IfNotExists().
		Exec(ctx); err != nil {
		return err
	}

	// table:reset_password_request
	if _, err := db.NewCreateTable().
		Model(&struct {
			bun.BaseModel `bun:"table:reset_password_request"`
			ID            int    `bun:"id,pk,autoincrement"`
			Token         string `bun:"token,type:text,notnull"`
			UserID        string `bun:"user_id,type:text,notnull"`
		}{}).
		ForeignKey(`("user_id") REFERENCES "users" ("id")`).
		IfNotExists().
		Exec(ctx); err != nil {
		return err
	}

	// table:user_flags
	if _, err := db.NewCreateTable().
		Model(&struct {
			bun.BaseModel `bun:"table:user_flags"`
			UserID        string `bun:"user_id,pk,type:text,notnull"`
			Flags         string `bun:"flags,type:text"`
		}{}).
		ForeignKey(`("user_id") REFERENCES "users" ("id")`).
		IfNotExists().
		Exec(ctx); err != nil {
		return err
	}

	// table:apdex_settings
	if _, err := db.NewCreateTable().
		Model(&struct {
			bun.BaseModel      `bun:"table:apdex_settings"`
			ServiceName        string  `bun:"service_name,pk,type:text"`
			Threshold          float64 `bun:"threshold,type:float,notnull"`
			ExcludeStatusCodes string  `bun:"exclude_status_codes,type:text,notnull"`
		}{}).
		IfNotExists().
		Exec(ctx); err != nil {
		return err
	}

	// table:ingestion_keys
	if _, err := db.NewCreateTable().
		Model(&struct {
			bun.BaseModel `bun:"table:ingestion_keys"`
			KeyId         string    `bun:"key_id,pk,type:text"`
			Name          string    `bun:"name,type:text"`
			CreatedAt     time.Time `bun:"created_at,default:current_timestamp"`
			IngestionKey  string    `bun:"ingestion_key,type:text,notnull"`
			IngestionURL  string    `bun:"ingestion_url,type:text,notnull"`
			DataRegion    string    `bun:"data_region,type:text,notnull"`
		}{}).
		IfNotExists().
		Exec(ctx); err != nil {
		return err
	}

	return nil
}

func (migration *addOrganization) Down(ctx context.Context, db *bun.DB) error {
	return nil
}

type addPreferences struct{}

func NewAddPreferencesFactory() factory.ProviderFactory[SQLMigration, Config] {
	return factory.NewProviderFactory(factory.MustNewName("add_preferences"), newAddPreferences)
}

func newAddPreferences(_ context.Context, _ factory.ProviderSettings, _ Config) (SQLMigration, error) {
	return &addPreferences{}, nil
}

func (migration *addPreferences) Register(migrations *migrate.Migrations) error {
	if err := migrations.Register(migration.Up, migration.Down); err != nil {
		return err
	}

	return nil
}

func (migration *addPreferences) Up(ctx context.Context, db *bun.DB) error {
	// table:user_preference
	if _, err := db.NewCreateTable().
		Model(&struct {
			bun.BaseModel   `bun:"table:user_preference"`
			PreferenceID    string `bun:"preference_id,type:text,pk"`
			PreferenceValue string `bun:"preference_value,type:text"`
			UserID          string `bun:"user_id,type:text,pk"`
		}{}).
		IfNotExists().
		ForeignKey(`("user_id") REFERENCES "users" ("id") ON DELETE CASCADE ON UPDATE CASCADE`).
		Exec(ctx); err != nil {
		return err
	}

	// table:org_preference
	if _, err := db.NewCreateTable().
		Model(&struct {
			bun.BaseModel   `bun:"table:org_preference"`
			PreferenceID    string `bun:"preference_id,pk,type:text,notnull"`
			PreferenceValue string `bun:"preference_value,type:text,notnull"`
			OrgID           string `bun:"org_id,pk,type:text,notnull"`
		}{}).
		ForeignKey(`("org_id") REFERENCES "organizations" ("id") ON DELETE CASCADE ON UPDATE CASCADE`).
		IfNotExists().
		Exec(ctx); err != nil {
		return err
	}

	return nil
}

func (migration *addPreferences) Down(ctx context.Context, db *bun.DB) error {
	return nil
}

type addDashboards struct{}

func NewAddDashboardsFactory() factory.ProviderFactory[SQLMigration, Config] {
	return factory.NewProviderFactory(factory.MustNewName("add_dashboards"), newAddDashboards)
}

func newAddDashboards(_ context.Context, _ factory.ProviderSettings, _ Config) (SQLMigration, error) {
	return &addDashboards{}, nil
}

func (migration *addDashboards) Register(migrations *migrate.Migrations) error {
	if err := migrations.Register(migration.Up, migration.Down); err != nil {
		return err
	}

	return nil
}

func (migration *addDashboards) Up(ctx context.Context, db *bun.DB) error {
	// table:dashboards
	if _, err := db.NewCreateTable().
		Model(&struct {
			bun.BaseModel `bun:"table:dashboards"`
			ID            int       `bun:"id,pk,autoincrement"`
			UUID          string    `bun:"uuid,type:text,notnull,unique"`
			CreatedAt     time.Time `bun:"created_at,notnull"`
			CreatedBy     string    `bun:"created_by,type:text,notnull"`
			UpdatedAt     time.Time `bun:"updated_at,notnull"`
			UpdatedBy     string    `bun:"updated_by,type:text,notnull"`
			Data          string    `bun:"data,type:text,notnull"`
			Locked        int       `bun:"locked,notnull,default:0"`
		}{}).
		IfNotExists().
		Exec(ctx); err != nil {
		return err
	}

	// table:rules
	if _, err := db.NewCreateTable().
		Model(&struct {
			bun.BaseModel `bun:"table:rules"`
			ID            int       `bun:"id,pk,autoincrement"`
			CreatedAt     time.Time `bun:"created_at,notnull"`
			CreatedBy     string    `bun:"created_by,type:text,notnull"`
			UpdatedAt     time.Time `bun:"updated_at,notnull"`
			UpdatedBy     string    `bun:"updated_by,type:text,notnull"`
			Deleted       int       `bun:"deleted,notnull,default:0"`
			Data          string    `bun:"data,type:text,notnull"`
		}{}).
		IfNotExists().
		Exec(ctx); err != nil {
		return err
	}

	// table:notification_channels
	if _, err := db.NewCreateTable().
		Model(&struct {
			bun.BaseModel `bun:"table:notification_channels"`
			ID            int       `bun:"id,pk,autoincrement"`
			CreatedAt     time.Time `bun:"created_at,notnull"`
			UpdatedAt     time.Time `bun:"updated_at,notnull"`
			Name          string    `bun:"name,type:text,notnull,unique"`
			Type          string    `bun:"type,type:text,notnull"`
			Deleted       int       `bun:"deleted,notnull,default:0"`
			Data          string    `bun:"data,type:text,notnull"`
		}{}).
		IfNotExists().
		Exec(ctx); err != nil {
		return err
	}

	// table:planned_maintenance
	if _, err := db.NewCreateTable().
		Model(&struct {
			bun.BaseModel `bun:"table:planned_maintenance"`
			ID            int       `bun:"id,pk,autoincrement"`
			Name          string    `bun:"name,type:text,notnull"`
			Description   string    `bun:"description,type:text"`
			AlertIDs      string    `bun:"alert_ids,type:text"`
			Schedule      string    `bun:"schedule,type:text,notnull"`
			CreatedAt     time.Time `bun:"created_at,notnull"`
			CreatedBy     string    `bun:"created_by,type:text,notnull"`
			UpdatedAt     time.Time `bun:"updated_at,notnull"`
			UpdatedBy     string    `bun:"updated_by,type:text,notnull"`
		}{}).
		IfNotExists().
		Exec(ctx); err != nil {
		return err
	}

	// table:ttl_status
	if _, err := db.NewCreateTable().
		Model(&struct {
			bun.BaseModel  `bun:"table:ttl_status"`
			ID             int       `bun:"id,pk,autoincrement"`
			TransactionID  string    `bun:"transaction_id,type:text,notnull"`
			CreatedAt      time.Time `bun:"created_at,notnull"`
			UpdatedAt      time.Time `bun:"updated_at,notnull"`
			TableName      string    `bun:"table_name,type:text,notnull"`
			TTL            int       `bun:"ttl,notnull,default:0"`
			ColdStorageTTL int       `bun:"cold_storage_ttl,notnull,default:0"`
			Status         string    `bun:"status,type:text,notnull"`
		}{}).
		IfNotExists().
		Exec(ctx); err != nil {
		return err
	}

	return nil
}

func (migration *addDashboards) Down(ctx context.Context, db *bun.DB) error {
	return nil
}

type addSavedViews struct{}

func NewAddSavedViewsFactory() factory.ProviderFactory[SQLMigration, Config] {
	return factory.NewProviderFactory(factory.MustNewName("add_saved_views"), newAddSavedViews)
}

func newAddSavedViews(_ context.Context, _ factory.ProviderSettings, _ Config) (SQLMigration, error) {
	return &addSavedViews{}, nil
}

func (migration *addSavedViews) Register(migrations *migrate.Migrations) error {
	if err := migrations.Register(migration.Up, migration.Down); err != nil {
		return err
	}

	return nil
}

func (migration *addSavedViews) Up(ctx context.Context, db *bun.DB) error {
	// table:saved_views op:create
	if _, err := db.NewCreateTable().
		Model(&struct {
			bun.BaseModel `bun:"table:saved_views"`
			UUID          string    `bun:"uuid,pk,type:text"`
			Name          string    `bun:"name,type:text,notnull"`
			Category      string    `bun:"category,type:text,notnull"`
			CreatedAt     time.Time `bun:"created_at,notnull"`
			CreatedBy     string    `bun:"created_by,type:text"`
			UpdatedAt     time.Time `bun:"updated_at,notnull"`
			UpdatedBy     string    `bun:"updated_by,type:text"`
			SourcePage    string    `bun:"source_page,type:text,notnull"`
			Tags          string    `bun:"tags,type:text"`
			Data          string    `bun:"data,type:text,notnull"`
			ExtraData     string    `bun:"extra_data,type:text"`
		}{}).
		IfNotExists().
		Exec(ctx); err != nil {
		return err
	}

	return nil
}

func (migration *addSavedViews) Down(ctx context.Context, db *bun.DB) error {
	return nil
}

type addAgents struct{}

func NewAddAgentsFactory() factory.ProviderFactory[SQLMigration, Config] {
	return factory.NewProviderFactory(factory.MustNewName("add_agents"), newAddAgents)
}

func newAddAgents(_ context.Context, _ factory.ProviderSettings, _ Config) (SQLMigration, error) {
	return &addAgents{}, nil
}

func (migration *addAgents) Register(migrations *migrate.Migrations) error {
	if err := migrations.Register(migration.Up, migration.Down); err != nil {
		return err
	}

	return nil
}

func (migration *addAgents) Up(ctx context.Context, db *bun.DB) error {
	if _, err := db.NewCreateTable().
		Model(&struct {
			bun.BaseModel   `bun:"table:agents"`
			AgentID         string    `bun:"agent_id,pk,type:text,unique"`
			StartedAt       time.Time `bun:"started_at,notnull"`
			TerminatedAt    time.Time `bun:"terminated_at"`
			CurrentStatus   string    `bun:"current_status,type:text,notnull"`
			EffectiveConfig string    `bun:"effective_config,type:text,notnull"`
		}{}).
		IfNotExists().
		Exec(ctx); err != nil {
		return err
	}

	if _, err := db.NewCreateTable().
		Model(&struct {
			bun.BaseModel  `bun:"table:agent_config_versions"`
			ID             string    `bun:"id,pk,type:text"`
			CreatedBy      string    `bun:"created_by,type:text"`
			CreatedAt      time.Time `bun:"created_at,default:CURRENT_TIMESTAMP"`
			UpdatedBy      string    `bun:"updated_by,type:text"`
			UpdatedAt      time.Time `bun:"updated_at,default:CURRENT_TIMESTAMP"`
			Version        int       `bun:"version,default:1,unique:element_version_idx"`
			Active         int       `bun:"active"`
			IsValid        int       `bun:"is_valid"`
			Disabled       int       `bun:"disabled"`
			ElementType    string    `bun:"element_type,notnull,type:varchar(120),unique:element_version_idx"`
			DeployStatus   string    `bun:"deploy_status,notnull,type:varchar(80),default:'DIRTY'"`
			DeploySequence int       `bun:"deploy_sequence"`
			DeployResult   string    `bun:"deploy_result,type:text"`
			LastHash       string    `bun:"last_hash,type:text"`
			LastConfig     string    `bun:"last_config,type:text"`
		}{}).
		IfNotExists().
		Exec(ctx); err != nil {
		return err
	}

	// add an index on the last_hash column
	if _, err := db.NewCreateIndex().
		Table("agent_config_versions").
		Column("last_hash").
		Index("idx_agent_config_versions_last_hash").
		IfNotExists().
		Exec(ctx); err != nil {
		return err
	}

	if _, err := db.NewCreateTable().
		Model(&struct {
			bun.BaseModel `bun:"table:agent_config_elements"`

			ID          string    `bun:"id,pk,type:text"`
			CreatedBy   string    `bun:"created_by,type:text"`
			CreatedAt   time.Time `bun:"created_at,default:CURRENT_TIMESTAMP"`
			UpdatedBy   string    `bun:"updated_by,type:text"`
			UpdatedAt   time.Time `bun:"updated_at,default:CURRENT_TIMESTAMP"`
			ElementID   string    `bun:"element_id,type:text,notnull,unique:agent_config_elements_u1"`
			ElementType string    `bun:"element_type,type:varchar(120),notnull,unique:agent_config_elements_u1"`
			VersionID   string    `bun:"version_id,type:text,notnull,unique:agent_config_elements_u1"`
		}{}).
		IfNotExists().
		Exec(ctx); err != nil {
		return err
	}

	return nil
}

func (migration *addAgents) Down(ctx context.Context, db *bun.DB) error {
	return nil
}

type addPipelines struct{}

func NewAddPipelinesFactory() factory.ProviderFactory[SQLMigration, Config] {
	return factory.NewProviderFactory(factory.MustNewName("add_pipelines"), newAddPipelines)
}

func newAddPipelines(_ context.Context, _ factory.ProviderSettings, _ Config) (SQLMigration, error) {
	return &addPipelines{}, nil
}

func (migration *addPipelines) Register(migrations *migrate.Migrations) error {
	if err := migrations.Register(migration.Up, migration.Down); err != nil {
		return err
	}

	return nil
}

func (migration *addPipelines) Up(ctx context.Context, db *bun.DB) error {
	if _, err := db.NewCreateTable().
		Model(&struct {
			bun.BaseModel `bun:"table:pipelines"`
			ID            string    `bun:"id,pk,type:text"`
			OrderID       int       `bun:"order_id"`
			Enabled       bool      `bun:"enabled"`
			CreatedBy     string    `bun:"created_by,type:text"`
			CreatedAt     time.Time `bun:"created_at,default:current_timestamp"`
			Name          string    `bun:"name,type:varchar(400),notnull"`
			Alias         string    `bun:"alias,type:varchar(20),notnull"`
			Description   string    `bun:"description,type:text"`
			Filter        string    `bun:"filter,type:text,notnull"`
			ConfigJSON    string    `bun:"config_json,type:text"`
		}{}).
		IfNotExists().
		Exec(ctx); err != nil {
		return err
	}

	return nil
}

func (migration *addPipelines) Down(ctx context.Context, db *bun.DB) error {
	return nil
}

type addIntegrations struct{}

func NewAddIntegrationsFactory() factory.ProviderFactory[SQLMigration, Config] {
	return factory.NewProviderFactory(factory.MustNewName("add_integrations"), newAddIntegrations)
}

func newAddIntegrations(_ context.Context, _ factory.ProviderSettings, _ Config) (SQLMigration, error) {
	return &addIntegrations{}, nil
}

func (migration *addIntegrations) Register(migrations *migrate.Migrations) error {
	if err := migrations.Register(migration.Up, migration.Down); err != nil {
		return err
	}

	return nil
}

func (migration *addIntegrations) Up(ctx context.Context, db *bun.DB) error {
	if _, err := db.NewCreateTable().
		Model(&struct {
			bun.BaseModel `bun:"table:integrations_installed"`

			IntegrationID string    `bun:"integration_id,pk,type:text"`
			ConfigJSON    string    `bun:"config_json,type:text"`
			InstalledAt   time.Time `bun:"installed_at,default:current_timestamp"`
		}{}).
		IfNotExists().
		Exec(ctx); err != nil {
		return err
	}

	if _, err := db.NewCreateTable().
		Model(&struct {
			bun.BaseModel       `bun:"table:cloud_integrations_accounts"`
			CloudProvider       string    `bun:"cloud_provider,type:text,unique:cloud_provider_id"`
			ID                  string    `bun:"id,type:text,notnull,unique:cloud_provider_id"`
			ConfigJSON          string    `bun:"config_json,type:text"`
			CloudAccountID      string    `bun:"cloud_account_id,type:text"`
			LastAgentReportJSON string    `bun:"last_agent_report_json,type:text"`
			CreatedAt           time.Time `bun:"created_at,notnull,default:current_timestamp"`
			RemovedAt           time.Time `bun:"removed_at,type:timestamp"`
		}{}).
		IfNotExists().
		Exec(ctx); err != nil {
		return err
	}

	if _, err := db.NewCreateTable().
		Model(&struct {
			bun.BaseModel  `bun:"table:cloud_integrations_service_configs"`
			CloudProvider  string    `bun:"cloud_provider,type:text,notnull,unique:service_cloud_provider_account"`
			CloudAccountID string    `bun:"cloud_account_id,type:text,notnull,unique:service_cloud_provider_account"`
			ServiceID      string    `bun:"service_id,type:text,notnull,unique:service_cloud_provider_account"`
			ConfigJSON     string    `bun:"config_json,type:text"`
			CreatedAt      time.Time `bun:"created_at,default:current_timestamp"`
		}{}).
		IfNotExists().
		Exec(ctx); err != nil {
		return err
	}

	return nil
}

func (migration *addIntegrations) Down(ctx context.Context, db *bun.DB) error {
	return nil
}

type addLicenses struct{}

func NewAddLicensesFactory() factory.ProviderFactory[SQLMigration, Config] {
	return factory.NewProviderFactory(factory.MustNewName("add_licenses"), newAddLicenses)
}

func newAddLicenses(_ context.Context, _ factory.ProviderSettings, _ Config) (SQLMigration, error) {
	return &addLicenses{}, nil
}

func (migration *addLicenses) Register(migrations *migrate.Migrations) error {
	if err := migrations.Register(migration.Up, migration.Down); err != nil {
		return err
	}

	return nil
}

func (migration *addLicenses) Up(ctx context.Context, db *bun.DB) error {
	if _, err := db.NewCreateTable().
		Model(&struct {
			bun.BaseModel     `bun:"table:licenses"`
			Key               string    `bun:"key,pk,type:text"`
			CreatedAt         time.Time `bun:"createdAt,default:current_timestamp"`
			UpdatedAt         time.Time `bun:"updatedAt,default:current_timestamp"`
			PlanDetails       string    `bun:"planDetails,type:text"`
			ActivationID      string    `bun:"activationId,type:text"`
			ValidationMessage string    `bun:"validationMessage,type:text"`
			LastValidated     time.Time `bun:"lastValidated,default:current_timestamp"`
		}{}).
		IfNotExists().
		Exec(ctx); err != nil {
		return err
	}

	if _, err := db.NewCreateTable().
		Model(&struct {
			bun.BaseModel `bun:"table:sites"`

			UUID      string    `bun:"uuid,pk,type:text"`
			Alias     string    `bun:"alias,type:varchar(180),default:'PROD'"`
			URL       string    `bun:"url,type:varchar(300)"`
			CreatedAt time.Time `bun:"createdAt,default:current_timestamp"`
		}{}).
		IfNotExists().
		Exec(ctx); err != nil {
		return err
	}

	if _, err := db.NewCreateTable().
		Model(&struct {
			bun.BaseModel `bun:"table:feature_status"`
			Name          string `bun:"name,pk,type:text"`
			Active        bool   `bun:"active"`
			Usage         int    `bun:"usage,default:0"`
			UsageLimit    int    `bun:"usage_limit,default:0"`
			Route         string `bun:"route,type:text"`
		}{}).
		IfNotExists().
		Exec(ctx); err != nil {
		return err
	}

	if _, err := db.NewCreateTable().
		Model(&struct {
			bun.BaseModel `bun:"table:licenses_v3"`
			ID            string `bun:"id,pk,type:text"`
			Key           string `bun:"key,type:text,notnull,unique"`
			Data          string `bun:"data,type:text"`
		}{}).
		IfNotExists().
		Exec(ctx); err != nil {
		return err
	}

	return nil
}

func (migration *addLicenses) Down(ctx context.Context, db *bun.DB) error {
	return nil
}

type addPats struct{}

func NewAddPatsFactory() factory.ProviderFactory[SQLMigration, Config] {
	return factory.NewProviderFactory(factory.MustNewName("add_pats"), newAddPats)
}

func newAddPats(_ context.Context, _ factory.ProviderSettings, _ Config) (SQLMigration, error) {
	return &addPats{}, nil
}

func (migration *addPats) Register(migrations *migrate.Migrations) error {
	if err := migrations.Register(migration.Up, migration.Down); err != nil {
		return err
	}

	return nil
}

func (migration *addPats) Up(ctx context.Context, db *bun.DB) error {
	if _, err := db.NewCreateTable().
		Model(&struct {
			bun.BaseModel `bun:"table:org_domains"`

			ID        string `bun:"id,pk,type:text"`
			OrgID     string `bun:"org_id,type:text,notnull"`
			Name      string `bun:"name,type:varchar(50),notnull,unique"`
			CreatedAt int    `bun:"created_at,notnull"`
			UpdatedAt int    `bun:"updated_at"`
			Data      string `bun:"data,type:text,notnull"`
		}{}).
		ForeignKey(`("org_id") REFERENCES "organizations" ("id")`).
		IfNotExists().
		Exec(ctx); err != nil {
		return err
	}

	if _, err := db.NewCreateTable().
		Model(&struct {
			bun.BaseModel `bun:"table:personal_access_tokens"`

			ID              int    `bun:"id,pk,autoincrement"`
			Role            string `bun:"role,type:text,notnull,default:'ADMIN'"`
			UserID          string `bun:"user_id,type:text,notnull"`
			Token           string `bun:"token,type:text,notnull,unique"`
			Name            string `bun:"name,type:text,notnull"`
			CreatedAt       int    `bun:"created_at,notnull,default:0"`
			ExpiresAt       int    `bun:"expires_at,notnull,default:0"`
			UpdatedAt       int    `bun:"updated_at,notnull,default:0"`
			LastUsed        int    `bun:"last_used,notnull,default:0"`
			Revoked         bool   `bun:"revoked,notnull,default:false"`
			UpdatedByUserID string `bun:"updated_by_user_id,type:text,notnull,default:''"`
		}{}).
		ForeignKey(`("user_id") REFERENCES "users" ("id")`).
		IfNotExists().
		Exec(ctx); err != nil {
		return err
	}

	return nil
}

func (migration *addPats) Down(ctx context.Context, db *bun.DB) error {
	return nil
}

type modifyDatetime struct{}

func NewModifyDatetimeFactory() factory.ProviderFactory[SQLMigration, Config] {
	return factory.NewProviderFactory(factory.MustNewName("modify_datetime"), newModifyDatetime)
}

func newModifyDatetime(_ context.Context, _ factory.ProviderSettings, _ Config) (SQLMigration, error) {
	return &modifyDatetime{}, nil
}

func (migration *modifyDatetime) Register(migrations *migrate.Migrations) error {
	if err := migrations.Register(migration.Up, migration.Down); err != nil {
		return err
	}

	return nil
}

func (migration *modifyDatetime) Up(ctx context.Context, db *bun.DB) error {
	// only run this for old sqlite db
	if db.Dialect().Name().String() != "sqlite" {
		return nil
	}

	// begin transaction
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}

	defer func() {
		_ = tx.Rollback()
	}()

	tables := []string{"dashboards", "rules", "planned_maintenance", "ttl_status", "saved_views"}
	columns := []string{"created_at", "updated_at"}
	for _, table := range tables {
		for _, column := range columns {
			if err := modifyColumn(ctx, tx, table, column); err != nil {
				return err
			}
		}
	}

	for _, column := range []string{"started_at", "terminated_at"} {
		if err := modifyColumn(ctx, tx, "agents", column); err != nil {
			return err
		}
	}

	if err := tx.Commit(); err != nil {
		return err
	}

	return nil
}

func modifyColumn(ctx context.Context, tx bun.Tx, table string, column string) error {
	// rename old column
	if _, err := tx.ExecContext(ctx, `ALTER TABLE `+table+` RENAME COLUMN `+column+` TO `+column+`_old`); err != nil {
		return err
	}

	// cannot add not null constraint to the column
	if _, err := tx.ExecContext(ctx, `ALTER TABLE `+table+` ADD COLUMN `+column+` TIMESTAMP`); err != nil {
		return err
	}

	// update the new column with the value of the old column
	if _, err := tx.ExecContext(ctx, `UPDATE `+table+` SET `+column+` = `+column+`_old`); err != nil {
		return err
	}

	// drop the old column
	if _, err := tx.ExecContext(ctx, `ALTER TABLE `+table+` DROP COLUMN `+column+`_old`); err != nil {
		return err
	}
	return nil
}

func (migration *modifyDatetime) Down(ctx context.Context, db *bun.DB) error {
	return nil
}

type modifyOrgDomain struct{}

func NewModifyOrgDomainFactory() factory.ProviderFactory[SQLMigration, Config] {
	return factory.NewProviderFactory(factory.MustNewName("modify_org_domain"), newModifyOrgDomain)
}

func newModifyOrgDomain(_ context.Context, _ factory.ProviderSettings, _ Config) (SQLMigration, error) {
	return &modifyOrgDomain{}, nil
}

func (migration *modifyOrgDomain) Register(migrations *migrate.Migrations) error {
	if err := migrations.Register(migration.Up, migration.Down); err != nil {
		return err
	}

	return nil
}

func (migration *modifyOrgDomain) Up(ctx context.Context, db *bun.DB) error {
	// only run this for old sqlite db
	if db.Dialect().Name() != dialect.SQLite {
		return nil
	}

	// begin transaction
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}

	defer func() {
		_ = tx.Rollback()
	}()

	// rename old column
	if _, err := tx.ExecContext(ctx, `ALTER TABLE org_domains RENAME COLUMN updated_at TO updated_at_old`); err != nil {
		return err
	}

	if _, err := tx.ExecContext(ctx, `ALTER TABLE org_domains ADD COLUMN updated_at INTEGER`); err != nil {
		return err
	}

	if _, err := tx.ExecContext(ctx, `UPDATE org_domains SET updated_at = CAST(updated_at_old AS INTEGER)`); err != nil {
		return err
	}

	// drop the old column
	if _, err := tx.ExecContext(ctx, `ALTER TABLE org_domains DROP COLUMN updated_at_old`); err != nil {
		return err
	}

	if err := tx.Commit(); err != nil {
		return err
	}

	return nil
}

func (migration *modifyOrgDomain) Down(ctx context.Context, db *bun.DB) error {
	return nil
}

type updateOrganization struct {
	store sqlstore.SQLStore
}

func NewUpdateOrganizationFactory(sqlstore sqlstore.SQLStore) factory.ProviderFactory[SQLMigration, Config] {
	return factory.NewProviderFactory(factory.MustNewName("update_organization"), func(ctx context.Context, ps factory.ProviderSettings, c Config) (SQLMigration, error) {
		return newUpdateOrganization(ctx, ps, c, sqlstore)
	})
}

func newUpdateOrganization(_ context.Context, _ factory.ProviderSettings, _ Config, store sqlstore.SQLStore) (SQLMigration, error) {
	return &updateOrganization{
		store: store,
	}, nil
}

func (migration *updateOrganization) Register(migrations *migrate.Migrations) error {
	if err := migrations.Register(migration.Up, migration.Down); err != nil {
		return err
	}

	return nil
}

func (migration *updateOrganization) Up(ctx context.Context, db *bun.DB) error {
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}

	defer func() {
		_ = tx.Rollback()
	}()

	// update apdex settings table
	if err := updateApdexSettings(ctx, tx); err != nil {
		return err
	}

	// drop user_flags table
	if _, err := tx.NewDropTable().IfExists().Table("user_flags").Exec(ctx); err != nil {
		return err
	}

	// add org id to groups table
	if exists, err := migration.store.Dialect().ColumnExists(ctx, tx, "groups", "org_id"); err != nil {
		return err
	} else if !exists {
		if _, err := tx.NewAddColumn().Table("groups").ColumnExpr("org_id TEXT").Exec(ctx); err != nil {
			return err
		}
	}

	// add created_at to groups table
	for _, table := range []string{"groups"} {
		if exists, err := migration.store.Dialect().ColumnExists(ctx, tx, table, "created_at"); err != nil {
			return err
		} else if !exists {
			if _, err := tx.NewAddColumn().Table(table).ColumnExpr("created_at TIMESTAMP").Exec(ctx); err != nil {
				return err
			}
		}
	}

	// add updated_at to organizations, users, groups table
	for _, table := range []string{"organizations", "users", "groups"} {
		if exists, err := migration.store.Dialect().ColumnExists(ctx, tx, table, "updated_at"); err != nil {
			return err
		} else if !exists {
			if _, err := tx.NewAddColumn().Table(table).ColumnExpr("updated_at TIMESTAMP").Exec(ctx); err != nil {
				return err
			}
		}
	}

	// since organizations, users has created_at as integer instead of timestamp
	for _, table := range []string{"organizations", "users", "invites"} {
		if err := migration.store.Dialect().IntToTimestamp(ctx, tx, table, "created_at"); err != nil {
			return err
		}
	}

	// migrate is_anonymous and has_opted_updates to boolean from int
	for _, column := range []string{"is_anonymous", "has_opted_updates"} {
		if err := migration.store.Dialect().IntToBoolean(ctx, tx, "organizations", column); err != nil {
			return err
		}
	}

	if err := tx.Commit(); err != nil {
		return err
	}

	return nil
}

func (migration *updateOrganization) Down(ctx context.Context, db *bun.DB) error {
	return nil
}

func updateApdexSettings(ctx context.Context, tx bun.Tx) error {
	if _, err := tx.NewCreateTable().
		Model(&struct {
			bun.BaseModel      `bun:"table:apdex_settings_new"`
			OrgID              string  `bun:"org_id,pk,type:text"`
			ServiceName        string  `bun:"service_name,pk,type:text"`
			Threshold          float64 `bun:"threshold,type:float,notnull"`
			ExcludeStatusCodes string  `bun:"exclude_status_codes,type:text,notnull"`
		}{}).
		ForeignKey(`("org_id") REFERENCES "organizations" ("id")`).
		IfNotExists().
		Exec(ctx); err != nil {
		return err
	}

	// get org id from organizations table
	var orgID string
	if err := tx.QueryRowContext(ctx, `SELECT id FROM organizations LIMIT 1`).Scan(&orgID); err != nil && !errors.Is(err, sql.ErrNoRows) {
		return err
	}

	if orgID != "" {
		// copy old data
		if _, err := tx.ExecContext(ctx, `INSERT INTO apdex_settings_new (org_id, service_name, threshold, exclude_status_codes) SELECT ?, service_name, threshold, exclude_status_codes FROM apdex_settings`, orgID); err != nil {
			return err
		}
	}

	// drop old table
	if _, err := tx.NewDropTable().IfExists().Table("apdex_settings").Exec(ctx); err != nil {
		return err
	}

	// rename new table to old table
	if _, err := tx.ExecContext(ctx, `ALTER TABLE apdex_settings_new RENAME TO apdex_settings`); err != nil {
		return err
	}

	return nil
}

type addAlertmanager struct {
	store sqlstore.SQLStore
}

func NewAddAlertmanagerFactory(store sqlstore.SQLStore) factory.ProviderFactory[SQLMigration, Config] {
	return factory.NewProviderFactory(factory.MustNewName("add_alertmanager"), func(ctx context.Context, ps factory.ProviderSettings, c Config) (SQLMigration, error) {
		return newAddAlertmanager(ctx, ps, c, store)
	})
}

func newAddAlertmanager(_ context.Context, _ factory.ProviderSettings, _ Config, store sqlstore.SQLStore) (SQLMigration, error) {
	return &addAlertmanager{
		store: store,
	}, nil
}

func (migration *addAlertmanager) Register(migrations *migrate.Migrations) error {
	if err := migrations.Register(migration.Up, migration.Down); err != nil {
		return err
	}

	return nil
}

func (migration *addAlertmanager) Up(ctx context.Context, db *bun.DB) error {
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}

	defer func() {
		_ = tx.Rollback()
	}()

	if exists, err := migration.store.Dialect().ColumnExists(ctx, tx, "notification_channels", "deleted"); err != nil {
		return err
	} else if exists {
		if _, err := tx.
			NewDropColumn().
			Table("notification_channels").
			ColumnExpr("deleted").
			Exec(ctx); err != nil {
			return err
		}
	}

	if exists, err := migration.store.Dialect().ColumnExists(ctx, tx, "notification_channels", "org_id"); err != nil {
		return err
	} else if !exists {
		if _, err := tx.
			NewAddColumn().
			Table("notification_channels").
			ColumnExpr("org_id TEXT REFERENCES organizations(id) ON DELETE CASCADE").
			Exec(ctx); err != nil {
			return err
		}
	}

	if _, err := tx.
		NewCreateTable().
		Model(&struct {
			bun.BaseModel `bun:"table:alertmanager_config"`
			ID            uint64    `bun:"id,pk,autoincrement"`
			Config        string    `bun:"config,notnull,type:text"`
			Hash          string    `bun:"hash,notnull,type:text"`
			CreatedAt     time.Time `bun:"created_at,notnull"`
			UpdatedAt     time.Time `bun:"updated_at,notnull"`
			OrgID         string    `bun:"org_id,notnull,unique"`
		}{}).
		ForeignKey(`("org_id") REFERENCES "organizations" ("id")`).
		IfNotExists().
		Exec(ctx); err != nil {
		return err
	}

	if _, err := tx.
		NewCreateTable().
		Model(&struct {
			bun.BaseModel `bun:"table:alertmanager_state"`
			ID            uint64    `bun:"id,pk,autoincrement"`
			Silences      string    `bun:"silences,nullzero,type:text"`
			NFLog         string    `bun:"nflog,nullzero,type:text"`
			CreatedAt     time.Time `bun:"created_at,notnull"`
			UpdatedAt     time.Time `bun:"updated_at,notnull"`
			OrgID         string    `bun:"org_id,notnull,unique"`
		}{}).
		ForeignKey(`("org_id") REFERENCES "organizations" ("id")`).
		IfNotExists().
		Exec(ctx); err != nil {
		return err
	}

	var orgID string
	err = tx.
		NewSelect().
		ColumnExpr("id").
		Table("organizations").
		Limit(1).
		Scan(ctx, &orgID)
	if err != nil {
		if err != sql.ErrNoRows {
			return err
		}
	}

	if err == nil {
		if err := migration.populateOrgIDInChannels(ctx, tx, orgID); err != nil {
			return err
		}

		if err := migration.populateAlertmanagerConfig(ctx, tx, orgID); err != nil {
			return err
		}
	}

	if err := tx.Commit(); err != nil {
		return err
	}

	return nil
}

func (migration *addAlertmanager) populateOrgIDInChannels(ctx context.Context, tx bun.Tx, orgID string) error {
	if _, err := tx.
		NewUpdate().
		Table("notification_channels").
		Set("org_id = ?", orgID).
		Where("org_id IS NULL").
		Exec(ctx); err != nil {
		return err
	}

	return nil
}

func (migration *addAlertmanager) populateAlertmanagerConfig(ctx context.Context, tx bun.Tx, orgID string) error {
	var channels []*alertmanagertypes.Channel

	err := tx.
		NewSelect().
		Model(&channels).
		Where("org_id = ?", orgID).
		Scan(ctx)
	if err != nil {
		return err
	}

	supportedChannels := make([]*alertmanagertypes.Channel, 0, len(channels))
	supportedReceiverNames := make(map[string]struct{})
	for _, channel := range channels {
		if channel.Type != "email" && channel.Type != "webhook" {
			continue
		}
		supportedChannels = append(supportedChannels, channel)
		supportedReceiverNames[channel.Name] = struct{}{}
	}
	channels = supportedChannels

	var receiversFromChannels []string
	for _, channel := range channels {
		receiversFromChannels = append(receiversFromChannels, channel.Name)
	}

	type matcher struct {
		bun.BaseModel `bun:"table:rules"`
		ID            int    `bun:"id,pk"`
		Data          string `bun:"data"`
	}

	matchers := []matcher{}

	err = tx.
		NewSelect().
		Column("id", "data").
		Model(&matchers).
		Scan(ctx)
	if err != nil {
		return err
	}

	matchersMap := make(map[string][]string)
	for _, matcher := range matchers {
		receivers := gjson.Get(matcher.Data, "preferredChannels").Array()
		for _, receiver := range receivers {
			if _, ok := supportedReceiverNames[receiver.String()]; ok {
				matchersMap[strconv.Itoa(matcher.ID)] = append(matchersMap[strconv.Itoa(matcher.ID)], receiver.String())
			}
		}

		if len(receivers) == 0 {
			matchersMap[strconv.Itoa(matcher.ID)] = append(matchersMap[strconv.Itoa(matcher.ID)], receiversFromChannels...)
		}
	}

	config, err := alertmanagertypes.NewConfigFromChannels(alertmanagerserver.NewConfig().Global, alertmanagerserver.NewConfig().Route, channels, orgID)
	if err != nil {
		return err
	}

	for ruleID, receivers := range matchersMap {
		err = config.CreateRuleIDMatcher(ruleID, receivers)
		if err != nil {
			return err
		}
	}

	if _, err := tx.
		NewInsert().
		Model(config.StoreableConfig()).
		On("CONFLICT (org_id) DO UPDATE").
		Set("config = ?", config.StoreableConfig().Config).
		Set("hash = ?", config.StoreableConfig().Hash).
		Set("updated_at = ?", config.StoreableConfig().UpdatedAt).
		Exec(ctx); err != nil {
		return err
	}

	return nil
}

func (migration *addAlertmanager) Down(ctx context.Context, db *bun.DB) error {
	return nil
}

type updateDashboardAndSavedViews struct {
	store sqlstore.SQLStore
}

func NewUpdateDashboardAndSavedViewsFactory(sqlstore sqlstore.SQLStore) factory.ProviderFactory[SQLMigration, Config] {
	return factory.NewProviderFactory(factory.MustNewName("update_group"), func(ctx context.Context, ps factory.ProviderSettings, c Config) (SQLMigration, error) {
		return newUpdateDashboardAndSavedViews(ctx, ps, c, sqlstore)
	})
}

func newUpdateDashboardAndSavedViews(_ context.Context, _ factory.ProviderSettings, _ Config, store sqlstore.SQLStore) (SQLMigration, error) {
	return &updateDashboardAndSavedViews{
		store: store,
	}, nil
}

func (migration *updateDashboardAndSavedViews) Register(migrations *migrate.Migrations) error {
	if err := migrations.Register(migration.Up, migration.Down); err != nil {
		return err
	}

	return nil
}

func (migration *updateDashboardAndSavedViews) Up(ctx context.Context, db *bun.DB) error {
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}

	defer func() {
		_ = tx.Rollback()
	}()

	// get all org ids
	var orgIDs []string
	if err := migration.store.BunDB().NewSelect().Model((*types.Organization)(nil)).Column("id").Scan(ctx, &orgIDs); err != nil {
		return err
	}

	// add org id to dashboards table
	for _, table := range []string{"dashboards", "saved_views"} {
		if exists, err := migration.store.Dialect().ColumnExists(ctx, tx, table, "org_id"); err != nil {
			return err
		} else if !exists {
			if _, err := tx.NewAddColumn().Table(table).ColumnExpr("org_id TEXT REFERENCES organizations(id) ON DELETE CASCADE").Exec(ctx); err != nil {
				return err
			}

			// check if there is one org ID if yes then set it to all dashboards.
			if len(orgIDs) == 1 {
				orgID := orgIDs[0]
				if _, err := tx.NewUpdate().Table(table).Set("org_id = ?", orgID).Where("org_id IS NULL").Exec(ctx); err != nil {
					return err
				}
			}
		}
	}

	if err := tx.Commit(); err != nil {
		return err
	}

	return nil
}

func (migration *updateDashboardAndSavedViews) Down(ctx context.Context, db *bun.DB) error {
	return nil
}

type updatePatAndOrgDomains struct {
	store sqlstore.SQLStore
}

func NewUpdatePatAndOrgDomainsFactory(sqlstore sqlstore.SQLStore) factory.ProviderFactory[SQLMigration, Config] {
	return factory.NewProviderFactory(factory.MustNewName("update_pat_and_org_domains"), func(ctx context.Context, ps factory.ProviderSettings, c Config) (SQLMigration, error) {
		return newUpdatePatAndOrgDomains(ctx, ps, c, sqlstore)
	})
}

func newUpdatePatAndOrgDomains(_ context.Context, _ factory.ProviderSettings, _ Config, store sqlstore.SQLStore) (SQLMigration, error) {
	return &updatePatAndOrgDomains{
		store: store,
	}, nil
}

func (migration *updatePatAndOrgDomains) Register(migrations *migrate.Migrations) error {
	if err := migrations.Register(migration.Up, migration.Down); err != nil {
		return err
	}

	return nil
}

func (migration *updatePatAndOrgDomains) Up(ctx context.Context, db *bun.DB) error {
	// begin transaction
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}

	defer func() {
		_ = tx.Rollback()
	}()

	// get all org ids
	var orgIDs []string
	if err := tx.NewSelect().Model((*types.Organization)(nil)).Column("id").Scan(ctx, &orgIDs); err != nil {
		return err
	}

	// add org id to pat and org_domains table
	if exists, err := migration.store.Dialect().ColumnExists(ctx, tx, "personal_access_tokens", "org_id"); err != nil {
		return err
	} else if !exists {
		if _, err := tx.NewAddColumn().Table("personal_access_tokens").ColumnExpr("org_id TEXT REFERENCES organizations(id) ON DELETE CASCADE").Exec(ctx); err != nil {
			return err
		}

		// check if there is one org ID if yes then set it to all personal_access_tokens.
		if len(orgIDs) == 1 {
			orgID := orgIDs[0]
			if _, err := tx.NewUpdate().Table("personal_access_tokens").Set("org_id = ?", orgID).Where("org_id IS NULL").Exec(ctx); err != nil {
				return err
			}
		}
	}

	if err := updateOrgId(ctx, tx); err != nil {
		return err
	}

	// change created_at and updated_at from integer to timestamp
	for _, table := range []string{"personal_access_tokens", "org_domains"} {
		if err := migration.store.Dialect().IntToTimestamp(ctx, tx, table, "created_at"); err != nil {
			return err
		}
		if err := migration.store.Dialect().IntToTimestamp(ctx, tx, table, "updated_at"); err != nil {
			return err
		}
	}

	// drop table if exists ingestion_keys
	if _, err := tx.NewDropTable().IfExists().Table("ingestion_keys").Exec(ctx); err != nil {
		return err
	}

	if err := tx.Commit(); err != nil {
		return err
	}

	return nil
}

func (migration *updatePatAndOrgDomains) Down(ctx context.Context, db *bun.DB) error {
	return nil
}

func updateOrgId(ctx context.Context, tx bun.Tx) error {
	if _, err := tx.NewCreateTable().
		Model(&struct {
			bun.BaseModel `bun:"table:org_domains_new"`

			ID        string `bun:"id,pk,type:text"`
			OrgID     string `bun:"org_id,type:text,notnull"`
			Name      string `bun:"name,type:varchar(50),notnull,unique"`
			CreatedAt int    `bun:"created_at,notnull"`
			UpdatedAt int    `bun:"updated_at"`
			Data      string `bun:"data,type:text,notnull"`
		}{}).
		ForeignKey(`("org_id") REFERENCES "organizations" ("id") ON DELETE CASCADE`).
		IfNotExists().
		Exec(ctx); err != nil {
		return err
	}

	// copy data from org_domains to org_domains_new
	if _, err := tx.ExecContext(ctx, `INSERT INTO org_domains_new (id, org_id, name, created_at, updated_at, data) SELECT id, org_id, name, created_at, updated_at, data FROM org_domains`); err != nil {
		return err
	}
	// delete old table
	if _, err := tx.NewDropTable().IfExists().Table("org_domains").Exec(ctx); err != nil {
		return err
	}

	// rename new table to org_domains
	if _, err := tx.ExecContext(ctx, `ALTER TABLE org_domains_new RENAME TO org_domains`); err != nil {
		return err
	}
	return nil
}

type updatePipelines struct {
	store sqlstore.SQLStore
}

func NewUpdatePipelines(sqlstore sqlstore.SQLStore) factory.ProviderFactory[SQLMigration, Config] {
	return factory.NewProviderFactory(factory.MustNewName("update_pipelines"), func(ctx context.Context, ps factory.ProviderSettings, c Config) (SQLMigration, error) {
		return newUpdatePipelines(ctx, ps, c, sqlstore)
	})
}

func newUpdatePipelines(_ context.Context, _ factory.ProviderSettings, _ Config, store sqlstore.SQLStore) (SQLMigration, error) {
	return &updatePipelines{
		store: store,
	}, nil
}

func (migration *updatePipelines) Register(migrations *migrate.Migrations) error {
	if err := migrations.Register(migration.Up, migration.Down); err != nil {
		return err
	}

	return nil
}

func (migration *updatePipelines) Up(ctx context.Context, db *bun.DB) error {
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}

	defer func() {
		_ = tx.Rollback()
	}()

	// get all org ids
	var orgIDs []string
	if err := migration.store.BunDB().NewSelect().Model((*types.Organization)(nil)).Column("id").Scan(ctx, &orgIDs); err != nil {
		return err
	}

	// add org id to pipelines table
	if exists, err := migration.store.Dialect().ColumnExists(ctx, tx, "pipelines", "org_id"); err != nil {
		return err
	} else if !exists {
		if _, err := tx.NewAddColumn().Table("pipelines").ColumnExpr("org_id TEXT REFERENCES organizations(id) ON DELETE CASCADE").Exec(ctx); err != nil {
			return err
		}

		// check if there is one org ID if yes then set it to all pipelines.
		if len(orgIDs) == 1 {
			orgID := orgIDs[0]
			if _, err := tx.NewUpdate().Table("pipelines").Set("org_id = ?", orgID).Where("org_id IS NULL").Exec(ctx); err != nil {
				return err
			}
		}
	}

	// add updated_by to pipelines table
	if exists, err := migration.store.Dialect().ColumnExists(ctx, tx, "pipelines", "updated_by"); err != nil {
		return err
	} else if !exists {
		if _, err := tx.NewAddColumn().Table("pipelines").ColumnExpr("updated_by TEXT").Exec(ctx); err != nil {
			return err
		}
	}

	// add updated_at to pipelines table
	if exists, err := migration.store.Dialect().ColumnExists(ctx, tx, "pipelines", "updated_at"); err != nil {
		return err
	} else if !exists {
		if _, err := tx.NewAddColumn().Table("pipelines").ColumnExpr("updated_at TIMESTAMP").Exec(ctx); err != nil {
			return err
		}
	}

	if err := tx.Commit(); err != nil {
		return err
	}

	return nil
}

func (migration *updatePipelines) Down(ctx context.Context, db *bun.DB) error {
	return nil
}

type dropLicensesSites struct {
	store sqlstore.SQLStore
}

func NewDropLicensesSitesFactory(sqlstore sqlstore.SQLStore) factory.ProviderFactory[SQLMigration, Config] {
	return factory.NewProviderFactory(factory.MustNewName("drop_licenses_sites"), func(ctx context.Context, ps factory.ProviderSettings, c Config) (SQLMigration, error) {
		return newDropLicensesSites(ctx, ps, c, sqlstore)
	})
}

func newDropLicensesSites(_ context.Context, _ factory.ProviderSettings, _ Config, store sqlstore.SQLStore) (SQLMigration, error) {
	return &dropLicensesSites{store: store}, nil
}

func (migration *dropLicensesSites) Register(migrations *migrate.Migrations) error {
	if err := migrations.Register(migration.Up, migration.Down); err != nil {
		return err
	}

	return nil
}

func (migration *dropLicensesSites) Up(ctx context.Context, db *bun.DB) error {
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}

	defer func() {
		_ = tx.Rollback()
	}()

	if _, err := tx.
		NewDropTable().
		IfExists().
		Table("sites").
		Exec(ctx); err != nil {
		return err
	}

	if _, err := tx.
		NewDropTable().
		IfExists().
		Table("licenses").
		Exec(ctx); err != nil {
		return err
	}

	_, err = migration.
		store.
		Dialect().
		RenameColumn(ctx, tx, "saved_views", "uuid", "id")
	if err != nil {
		return err
	}

	if err := tx.Commit(); err != nil {
		return err
	}

	return nil
}

func (migration *dropLicensesSites) Down(context.Context, *bun.DB) error {
	return nil
}

type updateInvites struct {
	store sqlstore.SQLStore
}

type existingInvite struct {
	bun.BaseModel `bun:"table:invites"`

	OrgID     string    `bun:"org_id,type:text,notnull" json:"orgId"`
	ID        int       `bun:"id,pk,autoincrement" json:"id"`
	Name      string    `bun:"name,type:text,notnull" json:"name"`
	Email     string    `bun:"email,type:text,notnull,unique" json:"email"`
	Token     string    `bun:"token,type:text,notnull" json:"token"`
	CreatedAt time.Time `bun:"created_at,notnull" json:"createdAt"`
	Role      string    `bun:"role,type:text,notnull" json:"role"`
}

type newInvite struct {
	bun.BaseModel `bun:"table:user_invite"`

	types.Identifiable
	types.TimeAuditable
	Name  string `bun:"name,type:text,notnull" json:"name"`
	Email string `bun:"email,type:text,notnull,unique" json:"email"`
	Token string `bun:"token,type:text,notnull" json:"token"`
	Role  string `bun:"role,type:text,notnull" json:"role"`
	OrgID string `bun:"org_id,type:text,notnull" json:"orgId"`
}

func NewUpdateInvitesFactory(sqlstore sqlstore.SQLStore) factory.ProviderFactory[SQLMigration, Config] {
	return factory.NewProviderFactory(factory.MustNewName("update_invites"), func(ctx context.Context, ps factory.ProviderSettings, c Config) (SQLMigration, error) {
		return newUpdateInvites(ctx, ps, c, sqlstore)
	})
}

func newUpdateInvites(_ context.Context, _ factory.ProviderSettings, _ Config, store sqlstore.SQLStore) (SQLMigration, error) {
	return &updateInvites{store: store}, nil
}

func (migration *updateInvites) Register(migrations *migrate.Migrations) error {
	if err := migrations.Register(migration.Up, migration.Down); err != nil {
		return err
	}

	return nil
}

func (migration *updateInvites) Up(ctx context.Context, db *bun.DB) error {
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}

	defer func() {
		_ = tx.Rollback()
	}()

	err = migration.
		store.
		Dialect().
		RenameTableAndModifyModel(ctx, tx, new(existingInvite), new(newInvite), []string{OrgReference}, func(ctx context.Context) error {
			existingInvites := make([]*existingInvite, 0)
			err = tx.
				NewSelect().
				Model(&existingInvites).
				Scan(ctx)
			if err != nil {
				if err != sql.ErrNoRows {
					return err
				}
			}

			if err == nil && len(existingInvites) > 0 {
				newInvites := migration.CopyOldInvitesToNewInvites(existingInvites)
				_, err = tx.
					NewInsert().
					Model(&newInvites).
					Exec(ctx)
				if err != nil {
					return err
				}
			}
			return nil
		})
	if err != nil {
		return err
	}

	err = tx.Commit()
	if err != nil {
		return err
	}

	return nil
}

func (migration *updateInvites) Down(context.Context, *bun.DB) error {
	return nil
}

func (migration *updateInvites) CopyOldInvitesToNewInvites(existingInvites []*existingInvite) []*newInvite {
	newInvites := make([]*newInvite, 0)
	for _, invite := range existingInvites {
		newInvites = append(newInvites, &newInvite{
			Identifiable: types.Identifiable{
				ID: valuer.GenerateUUID(),
			},
			TimeAuditable: types.TimeAuditable{
				CreatedAt: invite.CreatedAt,
				UpdatedAt: time.Now(),
			},
			Name:  invite.Name,
			Email: invite.Email,
			Token: invite.Token,
			Role:  invite.Role,
			OrgID: invite.OrgID,
		})
	}

	return newInvites
}

type updatePat struct {
	store sqlstore.SQLStore
}

func NewUpdatePatFactory(sqlstore sqlstore.SQLStore) factory.ProviderFactory[SQLMigration, Config] {
	return factory.NewProviderFactory(factory.MustNewName("update_pat"), func(ctx context.Context, ps factory.ProviderSettings, c Config) (SQLMigration, error) {
		return newUpdatePat(ctx, ps, c, sqlstore)
	})
}

func newUpdatePat(_ context.Context, _ factory.ProviderSettings, _ Config, store sqlstore.SQLStore) (SQLMigration, error) {
	return &updatePat{store: store}, nil
}

func (migration *updatePat) Register(migrations *migrate.Migrations) error {
	if err := migrations.Register(migration.Up, migration.Down); err != nil {
		return err
	}

	return nil
}

func (migration *updatePat) Up(ctx context.Context, db *bun.DB) error {
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}

	defer func() {
		_ = tx.Rollback()
	}()

	for _, column := range []string{"last_used", "expires_at"} {
		if err := migration.
			store.
			Dialect().
			AddNotNullDefaultToColumn(ctx, tx, "personal_access_tokens", column, "INTEGER", "0"); err != nil {
			return err
		}
	}

	if err := migration.
		store.
		Dialect().
		AddNotNullDefaultToColumn(ctx, tx, "personal_access_tokens", "revoked", "BOOLEAN", "false"); err != nil {
		return err
	}

	if err := migration.
		store.
		Dialect().
		AddNotNullDefaultToColumn(ctx, tx, "personal_access_tokens", "updated_by_user_id", "TEXT", "''"); err != nil {
		return err
	}

	if err := tx.Commit(); err != nil {
		return err
	}

	return nil
}

func (migration *updatePat) Down(ctx context.Context, db *bun.DB) error {
	return nil
}

type updateAlertmanager struct {
	store sqlstore.SQLStore
}

type existingChannel struct {
	bun.BaseModel `bun:"table:notification_channels"`
	ID            int       `json:"id" bun:"id,pk,autoincrement"`
	Name          string    `json:"name" bun:"name"`
	Type          string    `json:"type" bun:"type"`
	Data          string    `json:"data" bun:"data"`
	CreatedAt     time.Time `json:"created_at" bun:"created_at"`
	UpdatedAt     time.Time `json:"updated_at" bun:"updated_at"`
	OrgID         string    `json:"org_id" bun:"org_id"`
}

type newChannel struct {
	bun.BaseModel `bun:"table:notification_channel"`
	types.Identifiable
	types.TimeAuditable
	Name  string `json:"name" bun:"name"`
	Type  string `json:"type" bun:"type"`
	Data  string `json:"data" bun:"data"`
	OrgID string `json:"org_id" bun:"org_id"`
}

type existingAlertmanagerConfig struct {
	bun.BaseModel `bun:"table:alertmanager_config"`
	ID            uint64    `bun:"id,pk,autoincrement"`
	Config        string    `bun:"config,notnull,type:text"`
	Hash          string    `bun:"hash,notnull,type:text"`
	CreatedAt     time.Time `bun:"created_at,notnull"`
	UpdatedAt     time.Time `bun:"updated_at,notnull"`
	OrgID         string    `bun:"org_id,notnull,unique"`
}

type newAlertmanagerConfig struct {
	bun.BaseModel `bun:"table:alertmanager_config_new"`
	types.Identifiable
	types.TimeAuditable
	Config string `bun:"config,notnull,type:text"`
	Hash   string `bun:"hash,notnull,type:text"`
	OrgID  string `bun:"org_id,notnull,unique"`
}

type existingAlertmanagerState struct {
	bun.BaseModel `bun:"table:alertmanager_state"`
	ID            uint64    `bun:"id,pk,autoincrement"`
	Silences      string    `bun:"silences,nullzero,type:text"`
	NFLog         string    `bun:"nflog,nullzero,type:text"`
	CreatedAt     time.Time `bun:"created_at,notnull"`
	UpdatedAt     time.Time `bun:"updated_at,notnull"`
	OrgID         string    `bun:"org_id,notnull,unique"`
}

type newAlertmanagerState struct {
	bun.BaseModel `bun:"table:alertmanager_state_new"`
	types.Identifiable
	types.TimeAuditable
	Silences string `bun:"silences,nullzero,type:text"`
	NFLog    string `bun:"nflog,nullzero,type:text"`
	OrgID    string `bun:"org_id,notnull,unique"`
}

func NewUpdateAlertmanagerFactory(sqlstore sqlstore.SQLStore) factory.ProviderFactory[SQLMigration, Config] {
	return factory.NewProviderFactory(factory.MustNewName("update_alertmanager"), func(ctx context.Context, ps factory.ProviderSettings, c Config) (SQLMigration, error) {
		return newUpdateAlertmanager(ctx, ps, c, sqlstore)
	})
}

func newUpdateAlertmanager(_ context.Context, _ factory.ProviderSettings, _ Config, store sqlstore.SQLStore) (SQLMigration, error) {
	return &updateAlertmanager{store: store}, nil
}

func (migration *updateAlertmanager) Register(migrations *migrate.Migrations) error {
	if err := migrations.Register(migration.Up, migration.Down); err != nil {
		return err
	}

	return nil
}

func (migration *updateAlertmanager) Up(ctx context.Context, db *bun.DB) error {
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}

	defer func() {
		_ = tx.Rollback()
	}()

	err = migration.
		store.
		Dialect().
		RenameTableAndModifyModel(ctx, tx, new(existingChannel), new(newChannel), []string{OrgReference}, func(ctx context.Context) error {
			existingChannels := make([]*existingChannel, 0)
			err = tx.
				NewSelect().
				Model(&existingChannels).
				Scan(ctx)
			if err != nil {
				if err != sql.ErrNoRows {
					return err
				}
			}

			if err == nil && len(existingChannels) > 0 {
				newChannels := migration.
					CopyOldChannelToNewChannel(existingChannels)
				_, err = tx.
					NewInsert().
					Model(&newChannels).
					Exec(ctx)
				if err != nil {
					return err
				}
			}
			return nil
		})
	if err != nil {
		return err
	}

	err = migration.
		store.
		Dialect().
		UpdatePrimaryKey(ctx, tx, new(existingAlertmanagerConfig), new(newAlertmanagerConfig), OrgReference, func(ctx context.Context) error {
			existingAlertmanagerConfigs := make([]*existingAlertmanagerConfig, 0)
			err = tx.
				NewSelect().
				Model(&existingAlertmanagerConfigs).
				Scan(ctx)
			if err != nil {
				if err != sql.ErrNoRows {
					return err
				}
			}

			if err == nil && len(existingAlertmanagerConfigs) > 0 {
				newAlertmanagerConfigs := migration.
					CopyOldConfigToNewConfig(existingAlertmanagerConfigs)
				_, err = tx.
					NewInsert().
					Model(&newAlertmanagerConfigs).
					Exec(ctx)
				if err != nil {
					return err
				}
			}
			return nil
		})
	if err != nil {
		return err
	}

	err = migration.
		store.
		Dialect().
		UpdatePrimaryKey(ctx, tx, new(existingAlertmanagerState), new(newAlertmanagerState), OrgReference, func(ctx context.Context) error {
			existingAlertmanagerStates := make([]*existingAlertmanagerState, 0)
			err = tx.
				NewSelect().
				Model(&existingAlertmanagerStates).
				Scan(ctx)
			if err != nil {
				if err != sql.ErrNoRows {
					return err
				}
			}

			if err == nil && len(existingAlertmanagerStates) > 0 {
				newAlertmanagerStates := migration.
					CopyOldStateToNewState(existingAlertmanagerStates)
				_, err = tx.
					NewInsert().
					Model(&newAlertmanagerStates).
					Exec(ctx)
				if err != nil {
					return err
				}
			}
			return nil
		})
	if err != nil {
		return err
	}

	err = tx.Commit()
	if err != nil {
		return err
	}

	return nil
}

func (migration *updateAlertmanager) Down(context.Context, *bun.DB) error {
	return nil
}

func (migration *updateAlertmanager) CopyOldChannelToNewChannel(existingChannels []*existingChannel) []*newChannel {
	newChannels := make([]*newChannel, 0)
	for _, channel := range existingChannels {
		newChannels = append(newChannels, &newChannel{
			Identifiable: types.Identifiable{
				ID: valuer.GenerateUUID(),
			},
			TimeAuditable: types.TimeAuditable{
				CreatedAt: channel.CreatedAt,
				UpdatedAt: channel.UpdatedAt,
			},
			Name:  channel.Name,
			Type:  channel.Type,
			Data:  channel.Data,
			OrgID: channel.OrgID,
		})
	}

	return newChannels
}

func (migration *updateAlertmanager) CopyOldConfigToNewConfig(existingAlertmanagerConfigs []*existingAlertmanagerConfig) []*newAlertmanagerConfig {
	newAlertmanagerConfigs := make([]*newAlertmanagerConfig, 0)
	for _, config := range existingAlertmanagerConfigs {
		newAlertmanagerConfigs = append(newAlertmanagerConfigs, &newAlertmanagerConfig{
			Identifiable: types.Identifiable{
				ID: valuer.GenerateUUID(),
			},
			TimeAuditable: types.TimeAuditable{
				CreatedAt: config.CreatedAt,
				UpdatedAt: config.UpdatedAt,
			},
			Config: config.Config,
			Hash:   config.Hash,
			OrgID:  config.OrgID,
		})
	}

	return newAlertmanagerConfigs
}

func (migration *updateAlertmanager) CopyOldStateToNewState(existingAlertmanagerStates []*existingAlertmanagerState) []*newAlertmanagerState {
	newAlertmanagerStates := make([]*newAlertmanagerState, 0)
	for _, state := range existingAlertmanagerStates {
		newAlertmanagerStates = append(newAlertmanagerStates, &newAlertmanagerState{
			Identifiable: types.Identifiable{
				ID: valuer.GenerateUUID(),
			},
			TimeAuditable: types.TimeAuditable{
				CreatedAt: state.CreatedAt,
				UpdatedAt: state.UpdatedAt,
			},
			Silences: state.Silences,
			NFLog:    state.NFLog,
			OrgID:    state.OrgID,
		})
	}

	return newAlertmanagerStates
}

type updatePreferences struct {
	store sqlstore.SQLStore
}

type existingOrgPreference struct {
	bun.BaseModel   `bun:"table:org_preference"`
	PreferenceID    string `bun:"preference_id,pk,type:text,notnull"`
	PreferenceValue string `bun:"preference_value,type:text,notnull"`
	OrgID           string `bun:"org_id,pk,type:text,notnull"`
}

type newOrgPreference struct {
	bun.BaseModel `bun:"table:org_preference_new"`
	types.Identifiable
	PreferenceID    string `bun:"preference_id,type:text,notnull"`
	PreferenceValue string `bun:"preference_value,type:text,notnull"`
	OrgID           string `bun:"org_id,type:text,notnull"`
}

type existingUserPreference struct {
	bun.BaseModel   `bun:"table:user_preference"`
	PreferenceID    string `bun:"preference_id,type:text,pk"`
	PreferenceValue string `bun:"preference_value,type:text"`
	UserID          string `bun:"user_id,type:text,pk"`
}

type newUserPreference struct {
	bun.BaseModel `bun:"table:user_preference_new"`
	types.Identifiable
	PreferenceID    string `bun:"preference_id,type:text,notnull"`
	PreferenceValue string `bun:"preference_value,type:text,notnull"`
	UserID          string `bun:"user_id,type:text,notnull"`
}

func NewUpdatePreferencesFactory(sqlstore sqlstore.SQLStore) factory.ProviderFactory[SQLMigration, Config] {
	return factory.NewProviderFactory(factory.MustNewName("update_preferences"), func(ctx context.Context, ps factory.ProviderSettings, c Config) (SQLMigration, error) {
		return newUpdatePreferences(ctx, ps, c, sqlstore)
	})
}

func newUpdatePreferences(_ context.Context, _ factory.ProviderSettings, _ Config, store sqlstore.SQLStore) (SQLMigration, error) {
	return &updatePreferences{store: store}, nil
}

func (migration *updatePreferences) Register(migrations *migrate.Migrations) error {
	if err := migrations.Register(migration.Up, migration.Down); err != nil {
		return err
	}

	return nil
}

func (migration *updatePreferences) Up(ctx context.Context, db *bun.DB) error {
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}

	defer func() {
		_ = tx.Rollback()
	}()

	err = migration.
		store.
		Dialect().
		AddPrimaryKey(ctx, tx, new(existingOrgPreference), new(newOrgPreference), OrgReference, func(ctx context.Context) error {
			existingOrgPreferences := make([]*existingOrgPreference, 0)
			err = tx.
				NewSelect().
				Model(&existingOrgPreferences).
				Scan(ctx)
			if err != nil {
				if err != sql.ErrNoRows {
					return err
				}
			}

			if err == nil && len(existingOrgPreferences) > 0 {
				newOrgPreferences := migration.
					CopyOldOrgPreferencesToNewOrgPreferences(existingOrgPreferences)
				_, err = tx.
					NewInsert().
					Model(&newOrgPreferences).
					Exec(ctx)
				if err != nil {
					return err
				}
			}

			tableName := tx.Dialect().Tables().Get(reflect.TypeOf(new(existingOrgPreference))).Name
			_, err = tx.
				ExecContext(ctx, fmt.Sprintf("CREATE UNIQUE INDEX IF NOT EXISTS %s_unique_idx ON %s (preference_id, org_id)", tableName, fmt.Sprintf("%s_new", tableName)))
			if err != nil {
				return err
			}

			return nil
		})
	if err != nil {
		return err
	}

	err = migration.
		store.
		Dialect().
		AddPrimaryKey(ctx, tx, new(existingUserPreference), new(newUserPreference), UserReference, func(ctx context.Context) error {
			existingUserPreferences := make([]*existingUserPreference, 0)
			err = tx.
				NewSelect().
				Model(&existingUserPreferences).
				Scan(ctx)
			if err != nil {
				if err != sql.ErrNoRows {
					return err
				}
			}

			if err == nil && len(existingUserPreferences) > 0 {
				newUserPreferences := migration.CopyOldUserPreferencesToNewUserPreferences(existingUserPreferences)
				_, err = tx.
					NewInsert().
					Model(&newUserPreferences).
					Exec(ctx)
				if err != nil {
					return err
				}
			}

			tableName := tx.Dialect().Tables().Get(reflect.TypeOf(new(existingUserPreference))).Name
			_, err = tx.
				ExecContext(ctx, fmt.Sprintf("CREATE UNIQUE INDEX IF NOT EXISTS %s_unique_idx ON %s (preference_id, user_id)", tableName, fmt.Sprintf("%s_new", tableName)))
			if err != nil {
				return err
			}

			return nil
		})
	if err != nil {
		return err
	}

	err = tx.Commit()
	if err != nil {
		return err
	}

	return nil
}

func (migration *updatePreferences) Down(context.Context, *bun.DB) error {
	return nil
}

func (migration *updatePreferences) CopyOldOrgPreferencesToNewOrgPreferences(existingOrgPreferences []*existingOrgPreference) []*newOrgPreference {
	newOrgPreferences := make([]*newOrgPreference, 0)
	for _, preference := range existingOrgPreferences {
		newOrgPreferences = append(newOrgPreferences, &newOrgPreference{
			Identifiable: types.Identifiable{
				ID: valuer.GenerateUUID(),
			},
			PreferenceID:    preference.PreferenceID,
			PreferenceValue: preference.PreferenceValue,
			OrgID:           preference.OrgID,
		})
	}
	return newOrgPreferences
}

func (migration *updatePreferences) CopyOldUserPreferencesToNewUserPreferences(existingUserPreferences []*existingUserPreference) []*newUserPreference {
	newUserPreferences := make([]*newUserPreference, 0)
	for _, preference := range existingUserPreferences {
		newUserPreferences = append(newUserPreferences, &newUserPreference{
			Identifiable: types.Identifiable{
				ID: valuer.GenerateUUID(),
			},
			PreferenceID:    preference.PreferenceID,
			PreferenceValue: preference.PreferenceValue,
			UserID:          preference.UserID,
		})
	}
	return newUserPreferences
}

type updateApdexTtl struct {
	store sqlstore.SQLStore
}

type existingApdexSettings struct {
	bun.BaseModel      `bun:"table:apdex_settings"`
	OrgID              string  `bun:"org_id,pk,type:text" json:"orgId"`
	ServiceName        string  `bun:"service_name,pk,type:text" json:"serviceName"`
	Threshold          float64 `bun:"threshold,type:float,notnull" json:"threshold"`
	ExcludeStatusCodes string  `bun:"exclude_status_codes,type:text,notnull" json:"excludeStatusCodes"`
}

type newApdexSettings struct {
	bun.BaseModel `bun:"table:apdex_setting"`
	types.Identifiable
	OrgID              string  `bun:"org_id,type:text" json:"orgId"`
	ServiceName        string  `bun:"service_name,type:text" json:"serviceName"`
	Threshold          float64 `bun:"threshold,type:float,notnull" json:"threshold"`
	ExcludeStatusCodes string  `bun:"exclude_status_codes,type:text,notnull" json:"excludeStatusCodes"`
}

type existingTTLStatus struct {
	bun.BaseModel  `bun:"table:ttl_status"`
	ID             int       `bun:"id,pk,autoincrement"`
	TransactionID  string    `bun:"transaction_id,type:text,notnull"`
	CreatedAt      time.Time `bun:"created_at,type:datetime,notnull"`
	UpdatedAt      time.Time `bun:"updated_at,type:datetime,notnull"`
	TableName      string    `bun:"table_name,type:text,notnull"`
	TTL            int       `bun:"ttl,notnull,default:0"`
	ColdStorageTTL int       `bun:"cold_storage_ttl,notnull,default:0"`
	Status         string    `bun:"status,type:text,notnull"`
}

type newTTLStatus struct {
	bun.BaseModel `bun:"table:ttl_setting"`
	types.Identifiable
	types.TimeAuditable
	TransactionID  string `bun:"transaction_id,type:text,notnull"`
	TableName      string `bun:"table_name,type:text,notnull"`
	TTL            int    `bun:"ttl,notnull,default:0"`
	ColdStorageTTL int    `bun:"cold_storage_ttl,notnull,default:0"`
	Status         string `bun:"status,type:text,notnull"`
	OrgID          string `json:"-" bun:"org_id,notnull"`
}

func NewUpdateApdexTtlFactory(sqlstore sqlstore.SQLStore) factory.ProviderFactory[SQLMigration, Config] {
	return factory.NewProviderFactory(factory.MustNewName("update_apdex_ttl"), func(ctx context.Context, ps factory.ProviderSettings, c Config) (SQLMigration, error) {
		return newUpdateApdexTtl(ctx, ps, c, sqlstore)
	})
}

func newUpdateApdexTtl(_ context.Context, _ factory.ProviderSettings, _ Config, store sqlstore.SQLStore) (SQLMigration, error) {
	return &updateApdexTtl{store: store}, nil
}

func (migration *updateApdexTtl) Register(migrations *migrate.Migrations) error {
	if err := migrations.Register(migration.Up, migration.Down); err != nil {
		return err
	}

	return nil
}

func (migration *updateApdexTtl) Up(ctx context.Context, db *bun.DB) error {
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}

	defer func() {
		_ = tx.Rollback()
	}()

	err = migration.
		store.
		Dialect().
		RenameTableAndModifyModel(ctx, tx, new(existingApdexSettings), new(newApdexSettings), []string{OrgReference}, func(ctx context.Context) error {
			existingApdexSettings := make([]*existingApdexSettings, 0)
			err = tx.
				NewSelect().
				Model(&existingApdexSettings).
				Scan(ctx)
			if err != nil {
				if err != sql.ErrNoRows {
					return err
				}
			}

			if err == nil && len(existingApdexSettings) > 0 {
				newSettings := migration.
					CopyExistingApdexSettingsToNewApdexSettings(existingApdexSettings)
				_, err = tx.
					NewInsert().
					Model(&newSettings).
					Exec(ctx)
				if err != nil {
					return err
				}
			}

			tableName := tx.Dialect().Tables().Get(reflect.TypeOf(new(newApdexSettings))).Name
			_, err = tx.
				ExecContext(ctx, fmt.Sprintf("CREATE UNIQUE INDEX IF NOT EXISTS %s_unique_idx ON %s (service_name, org_id)", tableName, tableName))
			if err != nil {
				return err
			}
			return nil
		})
	if err != nil {
		return err
	}

	err = migration.
		store.
		Dialect().
		RenameTableAndModifyModel(ctx, tx, new(existingTTLStatus), new(newTTLStatus), []string{OrgReference}, func(ctx context.Context) error {
			existingTTLStatus := make([]*existingTTLStatus, 0)
			err = tx.
				NewSelect().
				Model(&existingTTLStatus).
				Scan(ctx)
			if err != nil {
				if err != sql.ErrNoRows {
					return err
				}
			}

			if err == nil && len(existingTTLStatus) > 0 {
				var orgID string
				err := migration.
					store.
					BunDB().
					NewSelect().
					Model((*types.Organization)(nil)).
					Column("id").
					Scan(ctx, &orgID)
				if err != nil {
					if err != sql.ErrNoRows {
						return err
					}
				}

				if err == nil {
					newTTLStatus := migration.CopyExistingTTLStatusToNewTTLStatus(existingTTLStatus, orgID)
					_, err = tx.
						NewInsert().
						Model(&newTTLStatus).
						Exec(ctx)
					if err != nil {
						return err
					}
				}

			}
			return nil
		})
	if err != nil {
		return err
	}

	err = tx.Commit()
	if err != nil {
		return err
	}

	return nil
}

func (migration *updateApdexTtl) Down(context.Context, *bun.DB) error {
	return nil
}

func (migration *updateApdexTtl) CopyExistingApdexSettingsToNewApdexSettings(existingApdexSettings []*existingApdexSettings) []*newApdexSettings {
	newSettings := make([]*newApdexSettings, 0)
	for _, apdexSetting := range existingApdexSettings {
		newSettings = append(newSettings, &newApdexSettings{
			Identifiable: types.Identifiable{
				ID: valuer.GenerateUUID(),
			},
			ServiceName:        apdexSetting.ServiceName,
			Threshold:          apdexSetting.Threshold,
			ExcludeStatusCodes: apdexSetting.ExcludeStatusCodes,
			OrgID:              apdexSetting.OrgID,
		})
	}

	return newSettings
}

func (migration *updateApdexTtl) CopyExistingTTLStatusToNewTTLStatus(existingTTLStatus []*existingTTLStatus, orgID string) []*newTTLStatus {
	newTTLStatuses := make([]*newTTLStatus, 0)

	for _, ttl := range existingTTLStatus {
		newTTLStatuses = append(newTTLStatuses, &newTTLStatus{
			Identifiable: types.Identifiable{
				ID: valuer.GenerateUUID(),
			},
			TimeAuditable: types.TimeAuditable{
				CreatedAt: ttl.CreatedAt,
				UpdatedAt: ttl.UpdatedAt,
			},
			TransactionID:  ttl.TransactionID,
			TTL:            ttl.TTL,
			TableName:      ttl.TableName,
			ColdStorageTTL: ttl.ColdStorageTTL,
			Status:         ttl.Status,
			OrgID:          orgID,
		})
	}

	return newTTLStatuses
}

type updateResetPassword struct {
	store sqlstore.SQLStore
}

type existingResetPasswordRequest struct {
	bun.BaseModel `bun:"table:reset_password_request"`
	ID            int    `bun:"id,pk,autoincrement" json:"id"`
	Token         string `bun:"token,type:text,notnull" json:"token"`
	UserID        string `bun:"user_id,type:text,notnull" json:"userId"`
}

type newResetPasswordRequest struct {
	bun.BaseModel `bun:"table:reset_password_request_new"`
	types.Identifiable
	Token  string `bun:"token,type:text,notnull" json:"token"`
	UserID string `bun:"user_id,type:text,notnull" json:"userId"`
}

type existingPersonalAccessToken struct {
	bun.BaseModel `bun:"table:personal_access_tokens"`
	types.TimeAuditable
	OrgID           string `json:"orgId" bun:"org_id,type:text,notnull"`
	ID              int    `json:"id" bun:"id,pk,autoincrement"`
	Role            string `json:"role" bun:"role,type:text,notnull,default:'ADMIN'"`
	UserID          string `json:"userId" bun:"user_id,type:text,notnull"`
	Token           string `json:"token" bun:"token,type:text,notnull,unique"`
	Name            string `json:"name" bun:"name,type:text,notnull"`
	ExpiresAt       int64  `json:"expiresAt" bun:"expires_at,notnull,default:0"`
	LastUsed        int64  `json:"lastUsed" bun:"last_used,notnull,default:0"`
	Revoked         bool   `json:"revoked" bun:"revoked,notnull,default:false"`
	UpdatedByUserID string `json:"updatedByUserId" bun:"updated_by_user_id,type:text,notnull,default:''"`
}

type newPersonalAccessToken struct {
	bun.BaseModel `bun:"table:personal_access_token"`
	types.Identifiable
	types.TimeAuditable
	OrgID           string `json:"orgId" bun:"org_id,type:text,notnull"`
	Role            string `json:"role" bun:"role,type:text,notnull,default:'ADMIN'"`
	UserID          string `json:"userId" bun:"user_id,type:text,notnull"`
	Token           string `json:"token" bun:"token,type:text,notnull,unique"`
	Name            string `json:"name" bun:"name,type:text,notnull"`
	ExpiresAt       int64  `json:"expiresAt" bun:"expires_at,notnull,default:0"`
	LastUsed        int64  `json:"lastUsed" bun:"last_used,notnull,default:0"`
	Revoked         bool   `json:"revoked" bun:"revoked,notnull,default:false"`
	UpdatedByUserID string `json:"updatedByUserId" bun:"updated_by_user_id,type:text,notnull,default:''"`
}

func NewUpdateResetPasswordFactory(sqlstore sqlstore.SQLStore) factory.ProviderFactory[SQLMigration, Config] {
	return factory.NewProviderFactory(factory.MustNewName("update_reset_password"), func(ctx context.Context, ps factory.ProviderSettings, c Config) (SQLMigration, error) {
		return newUpdateResetPassword(ctx, ps, c, sqlstore)
	})
}

func newUpdateResetPassword(_ context.Context, _ factory.ProviderSettings, _ Config, store sqlstore.SQLStore) (SQLMigration, error) {
	return &updateResetPassword{store: store}, nil
}

func (migration *updateResetPassword) Register(migrations *migrate.Migrations) error {
	if err := migrations.Register(migration.Up, migration.Down); err != nil {
		return err
	}

	return nil
}

func (migration *updateResetPassword) Up(ctx context.Context, db *bun.DB) error {
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}

	defer func() {
		_ = tx.Rollback()
	}()

	err = migration.store.Dialect().UpdatePrimaryKey(ctx, tx, new(existingResetPasswordRequest), new(newResetPasswordRequest), UserReference, func(ctx context.Context) error {
		existingResetPasswordRequests := make([]*existingResetPasswordRequest, 0)
		err = tx.
			NewSelect().
			Model(&existingResetPasswordRequests).
			Scan(ctx)
		if err != nil {
			if err != sql.ErrNoRows {
				return err
			}
		}

		if err == nil && len(existingResetPasswordRequests) > 0 {
			newResetPasswordRequests := migration.CopyExistingResetPasswordRequestsToNewResetPasswordRequests(existingResetPasswordRequests)
			_, err = tx.
				NewInsert().
				Model(&newResetPasswordRequests).
				Exec(ctx)
			if err != nil {
				return err
			}
		}
		return nil
	})

	if err != nil {
		return err
	}

	err = migration.store.Dialect().RenameTableAndModifyModel(ctx, tx, new(existingPersonalAccessToken), new(newPersonalAccessToken), []string{OrgReference, UserReference}, func(ctx context.Context) error {
		existingPersonalAccessTokens := make([]*existingPersonalAccessToken, 0)
		err = tx.
			NewSelect().
			Model(&existingPersonalAccessTokens).
			Scan(ctx)
		if err != nil {
			if err != sql.ErrNoRows {
				return err
			}
		}

		if err == nil && len(existingPersonalAccessTokens) > 0 {
			newPersonalAccessTokens := migration.CopyExistingPATsToNewPATs(existingPersonalAccessTokens)
			_, err = tx.NewInsert().Model(&newPersonalAccessTokens).Exec(ctx)
			if err != nil {
				return err
			}
		}
		return nil
	})

	if err != nil {
		return err
	}

	err = tx.Commit()
	if err != nil {
		return err
	}

	return nil
}

func (migration *updateResetPassword) Down(context.Context, *bun.DB) error {
	return nil
}

func (migration *updateResetPassword) CopyExistingResetPasswordRequestsToNewResetPasswordRequests(existingPasswordRequests []*existingResetPasswordRequest) []*newResetPasswordRequest {
	newResetPasswordRequests := make([]*newResetPasswordRequest, 0)
	for _, request := range existingPasswordRequests {
		newResetPasswordRequests = append(newResetPasswordRequests, &newResetPasswordRequest{
			Identifiable: types.Identifiable{
				ID: valuer.GenerateUUID(),
			},
			Token:  request.Token,
			UserID: request.UserID,
		})
	}
	return newResetPasswordRequests
}

func (migration *updateResetPassword) CopyExistingPATsToNewPATs(existingPATs []*existingPersonalAccessToken) []*newPersonalAccessToken {
	newPATs := make([]*newPersonalAccessToken, 0)
	for _, pat := range existingPATs {
		newPATs = append(newPATs, &newPersonalAccessToken{
			Identifiable: types.Identifiable{
				ID: valuer.GenerateUUID(),
			},
			TimeAuditable: types.TimeAuditable{
				CreatedAt: pat.CreatedAt,
				UpdatedAt: pat.UpdatedAt,
			},
			Role:            pat.Role,
			Name:            pat.Name,
			ExpiresAt:       pat.ExpiresAt,
			LastUsed:        pat.LastUsed,
			UserID:          pat.UserID,
			Token:           pat.Token,
			Revoked:         pat.Revoked,
			UpdatedByUserID: pat.UpdatedByUserID,
			OrgID:           pat.OrgID,
		})
	}
	return newPATs
}

type addVirtualFields struct{}

func NewAddVirtualFieldsFactory() factory.ProviderFactory[SQLMigration, Config] {
	return factory.NewProviderFactory(factory.MustNewName("add_virtual_fields"), newAddVirtualFields)
}

func newAddVirtualFields(_ context.Context, _ factory.ProviderSettings, _ Config) (SQLMigration, error) {
	return &addVirtualFields{}, nil
}

func (migration *addVirtualFields) Register(migrations *migrate.Migrations) error {
	if err := migrations.Register(migration.Up, migration.Down); err != nil {
		return err
	}

	return nil
}

func (migration *addVirtualFields) Up(ctx context.Context, db *bun.DB) error {
	// table:virtual_field op:create
	if _, err := db.NewCreateTable().
		Model(&struct {
			bun.BaseModel `bun:"table:virtual_field"`

			types.Identifiable
			types.TimeAuditable
			types.UserAuditable

			Name        string                `bun:"name,type:text,notnull"`
			Expression  string                `bun:"expression,type:text,notnull"`
			Description string                `bun:"description,type:text"`
			Signal      telemetrytypes.Signal `bun:"signal,type:text,notnull"`
			OrgID       string                `bun:"org_id,type:text,notnull"`
		}{}).
		ForeignKey(`("org_id") REFERENCES "organizations" ("id") ON DELETE CASCADE`).
		IfNotExists().
		Exec(ctx); err != nil {
		return err
	}

	return nil
}

func (migration *addVirtualFields) Down(ctx context.Context, db *bun.DB) error {
	return nil
}

type updateIntegrations struct {
	store sqlstore.SQLStore
}

func NewUpdateIntegrationsFactory(sqlstore sqlstore.SQLStore) factory.ProviderFactory[SQLMigration, Config] {
	return factory.NewProviderFactory(factory.MustNewName("update_integrations"), func(ctx context.Context, ps factory.ProviderSettings, c Config) (SQLMigration, error) {
		return newUpdateIntegrations(ctx, ps, c, sqlstore)
	})
}

func newUpdateIntegrations(_ context.Context, _ factory.ProviderSettings, _ Config, store sqlstore.SQLStore) (SQLMigration, error) {
	return &updateIntegrations{store: store}, nil
}

func (migration *updateIntegrations) Register(migrations *migrate.Migrations) error {
	if err := migrations.Register(migration.Up, migration.Down); err != nil {
		return err
	}

	return nil
}

type existingInstalledIntegration struct {
	bun.BaseModel `bun:"table:integrations_installed"`

	IntegrationID string    `bun:"integration_id,pk,type:text"`
	ConfigJSON    string    `bun:"config_json,type:text"`
	InstalledAt   time.Time `bun:"installed_at,default:current_timestamp"`
}

type newInstalledIntegration struct {
	bun.BaseModel `bun:"table:installed_integration"`

	types.Identifiable
	Type        string    `json:"type" bun:"type,type:text,unique:org_id_type"`
	Config      string    `json:"config" bun:"config,type:text"`
	InstalledAt time.Time `json:"installed_at" bun:"installed_at,default:current_timestamp"`
	OrgID       string    `json:"org_id" bun:"org_id,type:text,unique:org_id_type"`
}

type existingCloudIntegration struct {
	bun.BaseModel `bun:"table:cloud_integrations_accounts"`

	CloudProvider       string     `bun:"cloud_provider,type:text,unique:cloud_provider_id"`
	ID                  string     `bun:"id,type:text,notnull,unique:cloud_provider_id"`
	ConfigJSON          string     `bun:"config_json,type:text"`
	CloudAccountID      string     `bun:"cloud_account_id,type:text"`
	LastAgentReportJSON string     `bun:"last_agent_report_json,type:text"`
	CreatedAt           time.Time  `bun:"created_at,notnull,default:current_timestamp"`
	RemovedAt           *time.Time `bun:"removed_at,type:timestamp"`
}

type newCloudIntegration struct {
	bun.BaseModel `bun:"table:cloud_integration"`

	types.Identifiable
	types.TimeAuditable
	Provider        string     `json:"provider" bun:"provider,type:text"`
	Config          string     `json:"config" bun:"config,type:text"`
	AccountID       string     `json:"account_id" bun:"account_id,type:text"`
	LastAgentReport string     `json:"last_agent_report" bun:"last_agent_report,type:text"`
	RemovedAt       *time.Time `json:"removed_at" bun:"removed_at,type:timestamp"`
	OrgID           string     `json:"org_id" bun:"org_id,type:text"`
}

type existingCloudIntegrationService struct {
	bun.BaseModel `bun:"table:cloud_integrations_service_configs,alias:c1"`

	CloudProvider  string    `bun:"cloud_provider,type:text,notnull,unique:service_cloud_provider_account"`
	CloudAccountID string    `bun:"cloud_account_id,type:text,notnull,unique:service_cloud_provider_account"`
	ServiceID      string    `bun:"service_id,type:text,notnull,unique:service_cloud_provider_account"`
	ConfigJSON     string    `bun:"config_json,type:text"`
	CreatedAt      time.Time `bun:"created_at,default:current_timestamp"`
}

type newCloudIntegrationService struct {
	bun.BaseModel `bun:"table:cloud_integration_service,alias:cis"`

	types.Identifiable
	types.TimeAuditable
	Type               string `bun:"type,type:text,notnull,unique:cloud_integration_id_type"`
	Config             string `bun:"config,type:text"`
	CloudIntegrationID string `bun:"cloud_integration_id,type:text,notnull,unique:cloud_integration_id_type"`
}

type StorablePersonalAccessToken struct {
	bun.BaseModel `bun:"table:personal_access_token"`
	types.Identifiable
	types.TimeAuditable
	OrgID           string `json:"orgId" bun:"org_id,type:text,notnull"`
	Role            string `json:"role" bun:"role,type:text,notnull,default:'ADMIN'"`
	UserID          string `json:"userId" bun:"user_id,type:text,notnull"`
	Token           string `json:"token" bun:"token,type:text,notnull,unique"`
	Name            string `json:"name" bun:"name,type:text,notnull"`
	ExpiresAt       int64  `json:"expiresAt" bun:"expires_at,notnull,default:0"`
	LastUsed        int64  `json:"lastUsed" bun:"last_used,notnull,default:0"`
	Revoked         bool   `json:"revoked" bun:"revoked,notnull,default:false"`
	UpdatedByUserID string `json:"updatedByUserId" bun:"updated_by_user_id,type:text,notnull,default:''"`
}

func (migration *updateIntegrations) Up(ctx context.Context, db *bun.DB) error {

	// begin transaction
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer func() {
		_ = tx.Rollback()
	}()

	// don't run the migration if there are multiple org ids
	orgIDs := make([]string, 0)
	err = migration.store.BunDB().NewSelect().Model((*types.Organization)(nil)).Column("id").Scan(ctx, &orgIDs)
	if err != nil {
		return err
	}
	if len(orgIDs) > 1 {
		return nil
	}

	// installed integrations
	err = migration.
		store.
		Dialect().
		RenameTableAndModifyModel(ctx, tx, new(existingInstalledIntegration), new(newInstalledIntegration), []string{OrgReference}, func(ctx context.Context) error {
			existingIntegrations := make([]*existingInstalledIntegration, 0)
			err = tx.
				NewSelect().
				Model(&existingIntegrations).
				Scan(ctx)
			if err != nil {
				if err != sql.ErrNoRows {
					return err
				}
			}

			if err == nil && len(existingIntegrations) > 0 {
				newIntegrations := migration.
					CopyOldIntegrationsToNewIntegrations(tx, orgIDs[0], existingIntegrations)
				_, err = tx.
					NewInsert().
					Model(&newIntegrations).
					Exec(ctx)
				if err != nil {
					return err
				}
			}
			return nil
		})
	if err != nil {
		return err
	}

	// cloud integrations
	err = migration.
		store.
		Dialect().
		RenameTableAndModifyModel(ctx, tx, new(existingCloudIntegration), new(newCloudIntegration), []string{OrgReference}, func(ctx context.Context) error {
			existingIntegrations := make([]*existingCloudIntegration, 0)
			err = tx.
				NewSelect().
				Model(&existingIntegrations).
				Where("removed_at IS NULL"). // we will only copy the accounts that are not removed
				Scan(ctx)
			if err != nil {
				if err != sql.ErrNoRows {
					return err
				}
			}

			if err == nil && len(existingIntegrations) > 0 {
				newIntegrations := migration.
					CopyOldCloudIntegrationsToNewCloudIntegrations(tx, orgIDs[0], existingIntegrations)
				_, err = tx.
					NewInsert().
					Model(&newIntegrations).
					Exec(ctx)
				if err != nil {
					return err
				}
			}
			return nil
		})
	if err != nil {
		return err
	}

	// add unique constraint to cloud_integration table
	_, err = tx.ExecContext(ctx, `CREATE UNIQUE INDEX IF NOT EXISTS unique_cloud_integration ON cloud_integration (id, provider, org_id)`)
	if err != nil {
		return err
	}

	// cloud integration service
	err = migration.
		store.
		Dialect().
		RenameTableAndModifyModel(ctx, tx, new(existingCloudIntegrationService), new(newCloudIntegrationService), []string{CloudIntegrationReference}, func(ctx context.Context) error {
			existingServices := make([]*existingCloudIntegrationService, 0)

			// only one service per provider,account id and type
			// so there won't be any duplicates.
			// just that these will be enabled as soon as the integration for the account is enabled
			err = tx.
				NewSelect().
				Model(&existingServices).
				Scan(ctx)
			if err != nil {
				if err != sql.ErrNoRows {
					return err
				}
			}

			if err == nil && len(existingServices) > 0 {
				newServices := migration.
					CopyOldCloudIntegrationServicesToNewCloudIntegrationServices(tx, orgIDs[0], existingServices)
				if len(newServices) > 0 {
					_, err = tx.
						NewInsert().
						Model(&newServices).
						Exec(ctx)
					if err != nil {
						return err
					}
				}
			}
			return nil
		})
	if err != nil {
		return err
	}

	if len(orgIDs) == 0 {
		err = tx.Commit()
		if err != nil {
			return err
		}
		return nil
	}

	// copy the old aws integration user to the new user
	err = migration.copyOldAwsIntegrationUser(tx, orgIDs[0])
	if err != nil {
		return err
	}

	err = tx.Commit()
	if err != nil {
		return err
	}

	return nil
}

func (migration *updateIntegrations) Down(ctx context.Context, db *bun.DB) error {
	return nil
}

func (migration *updateIntegrations) CopyOldIntegrationsToNewIntegrations(tx bun.IDB, orgID string, existingIntegrations []*existingInstalledIntegration) []*newInstalledIntegration {
	newIntegrations := make([]*newInstalledIntegration, 0)

	for _, integration := range existingIntegrations {
		newIntegrations = append(newIntegrations, &newInstalledIntegration{
			Identifiable: types.Identifiable{
				ID: valuer.GenerateUUID(),
			},
			Type:        integration.IntegrationID,
			Config:      integration.ConfigJSON,
			InstalledAt: integration.InstalledAt,
			OrgID:       orgID,
		})
	}

	return newIntegrations
}

func (migration *updateIntegrations) CopyOldCloudIntegrationsToNewCloudIntegrations(tx bun.IDB, orgID string, existingIntegrations []*existingCloudIntegration) []*newCloudIntegration {
	newIntegrations := make([]*newCloudIntegration, 0)

	for _, integration := range existingIntegrations {
		newIntegrations = append(newIntegrations, &newCloudIntegration{
			Identifiable: types.Identifiable{
				ID: valuer.GenerateUUID(),
			},
			TimeAuditable: types.TimeAuditable{
				CreatedAt: integration.CreatedAt,
				UpdatedAt: integration.CreatedAt,
			},
			Provider:        integration.CloudProvider,
			AccountID:       integration.CloudAccountID,
			Config:          integration.ConfigJSON,
			LastAgentReport: integration.LastAgentReportJSON,
			RemovedAt:       integration.RemovedAt,
			OrgID:           orgID,
		})
	}

	return newIntegrations
}

func (migration *updateIntegrations) CopyOldCloudIntegrationServicesToNewCloudIntegrationServices(tx bun.IDB, orgID string, existingServices []*existingCloudIntegrationService) []*newCloudIntegrationService {
	newServices := make([]*newCloudIntegrationService, 0)

	for _, service := range existingServices {
		var cloudIntegrationID string
		err := tx.NewSelect().
			Model((*newCloudIntegration)(nil)).
			Column("id").
			Where("account_id = ?", service.CloudAccountID).
			Where("provider = ?", service.CloudProvider).
			Where("org_id = ?", orgID).
			Scan(context.Background(), &cloudIntegrationID)
		if err != nil {
			if err == sql.ErrNoRows {
				continue
			}
			return nil
		}
		newServices = append(newServices, &newCloudIntegrationService{
			Identifiable: types.Identifiable{
				ID: valuer.GenerateUUID(),
			},
			TimeAuditable: types.TimeAuditable{
				CreatedAt: service.CreatedAt,
				UpdatedAt: service.CreatedAt,
			},
			Type:               service.ServiceID,
			Config:             service.ConfigJSON,
			CloudIntegrationID: cloudIntegrationID,
		})
	}

	return newServices
}

func (migration *updateIntegrations) copyOldAwsIntegrationUser(tx bun.IDB, orgID string) error {
	type oldUser struct {
		bun.BaseModel `bun:"table:users"`

		types.TimeAuditable
		ID                string `bun:"id,pk,type:text" json:"id"`
		Name              string `bun:"name,type:text,notnull" json:"name"`
		Email             string `bun:"email,type:text,notnull,unique" json:"email"`
		Password          string `bun:"password,type:text,notnull" json:"-"`
		ProfilePictureURL string `bun:"profile_picture_url,type:text" json:"profilePictureURL"`
		GroupID           string `bun:"group_id,type:text,notnull" json:"groupId"`
		OrgID             string `bun:"org_id,type:text,notnull" json:"orgId"`
	}

	user := &oldUser{}
	err := tx.NewSelect().Model(user).Where("email = ?", "aws-integration@signoz.io").Scan(context.Background())
	if err != nil {
		if err == sql.ErrNoRows {
			return nil
		}
		return err
	}

	// check if the id is already an uuid
	if _, err := uuid.Parse(user.ID); err == nil {
		return nil
	}

	// new user
	newUser := &oldUser{
		ID: uuid.New().String(),
		TimeAuditable: types.TimeAuditable{
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		},
		OrgID:    orgID,
		Name:     user.Name,
		Email:    user.Email,
		GroupID:  user.GroupID,
		Password: user.Password,
	}

	// get the pat for old user
	pat := &StorablePersonalAccessToken{}
	err = tx.NewSelect().Model(pat).Where("user_id = ? and revoked = false", "aws-integration").Scan(context.Background())
	if err != nil {
		if err == sql.ErrNoRows {
			// delete the old user
			_, err = tx.ExecContext(context.Background(), `DELETE FROM users WHERE id = ?`, user.ID)
			if err != nil {
				return err
			}
			return nil
		}
		return err
	}

	// new pat
	newPAT := &StorablePersonalAccessToken{
		Identifiable: types.Identifiable{ID: valuer.GenerateUUID()},
		TimeAuditable: types.TimeAuditable{
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		},
		OrgID:     orgID,
		UserID:    newUser.ID,
		Token:     pat.Token,
		Name:      pat.Name,
		ExpiresAt: pat.ExpiresAt,
		LastUsed:  pat.LastUsed,
		Revoked:   pat.Revoked,
		Role:      pat.Role,
	}

	// delete old user
	_, err = tx.ExecContext(context.Background(), `DELETE FROM users WHERE id = ?`, user.ID)
	if err != nil {
		return err
	}

	// insert the new user
	_, err = tx.NewInsert().Model(newUser).Exec(context.Background())
	if err != nil {
		return err
	}

	// insert the new pat
	_, err = tx.NewInsert().Model(newPAT).Exec(context.Background())
	if err != nil {
		return err
	}

	return nil
}

type updateRules struct {
	store sqlstore.SQLStore
}

type AlertIds []string

func (a *AlertIds) Scan(src interface{}) error {
	if data, ok := src.([]byte); ok {
		return json.Unmarshal(data, a)
	}
	return nil
}

func (a *AlertIds) Value() (driver.Value, error) {
	return json.Marshal(a)
}

type existingRule struct {
	bun.BaseModel `bun:"table:rules"`
	ID            int       `bun:"id,pk,autoincrement"`
	CreatedAt     time.Time `bun:"created_at,type:datetime,notnull"`
	CreatedBy     string    `bun:"created_by,type:text,notnull"`
	UpdatedAt     time.Time `bun:"updated_at,type:datetime,notnull"`
	UpdatedBy     string    `bun:"updated_by,type:text,notnull"`
	Deleted       int       `bun:"deleted,notnull,default:0"`
	Data          string    `bun:"data,type:text,notnull"`
}

type newRule struct {
	bun.BaseModel `bun:"table:rule"`
	types.Identifiable
	types.TimeAuditable
	types.UserAuditable
	Deleted int    `bun:"deleted,notnull,default:0"`
	Data    string `bun:"data,type:text,notnull"`
	OrgID   string `bun:"org_id,type:text"`
}

type existingMaintenance struct {
	bun.BaseModel `bun:"table:planned_maintenance"`
	ID            int                 `bun:"id,pk,autoincrement"`
	Name          string              `bun:"name,type:text,notnull"`
	Description   string              `bun:"description,type:text"`
	AlertIDs      *AlertIds           `bun:"alert_ids,type:text"`
	Schedule      *ruletypes.Schedule `bun:"schedule,type:text,notnull"`
	CreatedAt     time.Time           `bun:"created_at,type:datetime,notnull"`
	CreatedBy     string              `bun:"created_by,type:text,notnull"`
	UpdatedAt     time.Time           `bun:"updated_at,type:datetime,notnull"`
	UpdatedBy     string              `bun:"updated_by,type:text,notnull"`
}

type newMaintenance struct {
	bun.BaseModel `bun:"table:planned_maintenance_new"`
	types.Identifiable
	types.TimeAuditable
	types.UserAuditable
	Name        string              `bun:"name,type:text,notnull"`
	Description string              `bun:"description,type:text"`
	Schedule    *ruletypes.Schedule `bun:"schedule,type:text,notnull"`
	OrgID       string              `bun:"org_id,type:text"`
}

type storablePlannedMaintenanceRule struct {
	bun.BaseModel `bun:"table:planned_maintenance_rule"`
	types.Identifiable
	PlannedMaintenanceID valuer.UUID `bun:"planned_maintenance_id,type:text"`
	RuleID               valuer.UUID `bun:"rule_id,type:text"`
}

type ruleHistory struct {
	bun.BaseModel `bun:"table:rule_history"`
	RuleID        int         `bun:"rule_id"`
	RuleUUID      valuer.UUID `bun:"rule_uuid"`
}

func NewUpdateRulesFactory(sqlstore sqlstore.SQLStore) factory.ProviderFactory[SQLMigration, Config] {
	return factory.NewProviderFactory(factory.MustNewName("update_rules"), func(ctx context.Context, ps factory.ProviderSettings, c Config) (SQLMigration, error) {
		return newUpdateRules(ctx, ps, c, sqlstore)
	})
}

func newUpdateRules(_ context.Context, _ factory.ProviderSettings, _ Config, store sqlstore.SQLStore) (SQLMigration, error) {
	return &updateRules{store: store}, nil
}

func (migration *updateRules) Register(migrations *migrate.Migrations) error {
	if err := migrations.Register(migration.Up, migration.Down); err != nil {
		return err
	}

	return nil
}

func (migration *updateRules) Up(ctx context.Context, db *bun.DB) error {
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}

	defer func() {
		_ = tx.Rollback()
	}()

	ruleIDToRuleUUIDMap := map[int]valuer.UUID{}
	err = migration.
		store.
		Dialect().
		RenameTableAndModifyModel(ctx, tx, new(existingRule), new(newRule), []string{OrgReference}, func(ctx context.Context) error {
			existingRules := make([]*existingRule, 0)
			err := tx.
				NewSelect().
				Model(&existingRules).
				Scan(ctx)
			if err != nil {
				if err != sql.ErrNoRows {
					return err
				}
			}
			if err == nil && len(existingRules) > 0 {
				var orgID string
				err := migration.
					store.
					BunDB().
					NewSelect().
					Model((*types.Organization)(nil)).
					Column("id").
					Scan(ctx, &orgID)
				if err != nil {
					if err != sql.ErrNoRows {
						return err
					}
				}
				if err == nil {
					newRules, idUUIDMap := migration.CopyExistingRulesToNewRules(existingRules, orgID)
					ruleIDToRuleUUIDMap = idUUIDMap
					_, err = tx.
						NewInsert().
						Model(&newRules).
						Exec(ctx)
					if err != nil {
						return err
					}
				}
			}
			err = migration.store.Dialect().UpdatePrimaryKey(ctx, tx, new(existingMaintenance), new(newMaintenance), OrgReference, func(ctx context.Context) error {
				_, err := tx.
					NewCreateTable().
					IfNotExists().
					Model(new(storablePlannedMaintenanceRule)).
					ForeignKey(`("planned_maintenance_id") REFERENCES "planned_maintenance_new" ("id") ON DELETE CASCADE ON UPDATE CASCADE`).
					ForeignKey(`("rule_id") REFERENCES "rule" ("id")`).
					Exec(ctx)
				if err != nil {
					return err
				}

				existingMaintenances := make([]*existingMaintenance, 0)
				err = tx.
					NewSelect().
					Model(&existingMaintenances).
					Scan(ctx)
				if err != nil {
					if err != sql.ErrNoRows {
						return err
					}
				}
				if err == nil && len(existingMaintenances) > 0 {
					var orgID string
					err := migration.
						store.
						BunDB().
						NewSelect().
						Model((*types.Organization)(nil)).
						Column("id").
						Scan(ctx, &orgID)
					if err != nil {
						if err != sql.ErrNoRows {
							return err
						}
					}
					if err == nil {
						newMaintenances, newMaintenancesRules, err := migration.CopyExistingMaintenancesToNewMaintenancesAndRules(existingMaintenances, orgID, ruleIDToRuleUUIDMap)
						if err != nil {
							return err
						}

						_, err = tx.
							NewInsert().
							Model(&newMaintenances).
							Exec(ctx)
						if err != nil {
							return err
						}

						if len(newMaintenancesRules) > 0 {
							_, err = tx.
								NewInsert().
								Model(&newMaintenancesRules).
								Exec(ctx)
							if err != nil {
								return err
							}
						}

					}

				}
				return nil
			})
			if err != nil {
				return err
			}

			ruleHistories := make([]*ruleHistory, 0)
			for ruleID, ruleUUID := range ruleIDToRuleUUIDMap {
				ruleHistories = append(ruleHistories, &ruleHistory{
					RuleID:   ruleID,
					RuleUUID: ruleUUID,
				})
			}

			_, err = tx.
				NewCreateTable().
				IfNotExists().
				Model(&ruleHistories).
				Exec(ctx)
			if err != nil {
				return err
			}
			if len(ruleHistories) > 0 {
				_, err = tx.
					NewInsert().
					Model(&ruleHistories).
					Exec(ctx)
				if err != nil {
					return err
				}
			}

			return nil
		})

	if err != nil {
		return err
	}

	err = tx.Commit()
	if err != nil {
		return err
	}

	return nil
}

func (migration *updateRules) Down(context.Context, *bun.DB) error {
	return nil
}

func (migration *updateRules) CopyExistingRulesToNewRules(existingRules []*existingRule, orgID string) ([]*newRule, map[int]valuer.UUID) {
	newRules := make([]*newRule, 0)
	idUUIDMap := map[int]valuer.UUID{}
	for _, rule := range existingRules {
		uuid := valuer.GenerateUUID()
		idUUIDMap[rule.ID] = uuid
		newRules = append(newRules, &newRule{
			Identifiable: types.Identifiable{
				ID: uuid,
			},
			TimeAuditable: types.TimeAuditable{
				CreatedAt: rule.CreatedAt,
				UpdatedAt: rule.UpdatedAt,
			},
			UserAuditable: types.UserAuditable{
				CreatedBy: rule.CreatedBy,
				UpdatedBy: rule.UpdatedBy,
			},
			Deleted: rule.Deleted,
			Data:    rule.Data,
			OrgID:   orgID,
		})
	}
	return newRules, idUUIDMap
}

func (migration *updateRules) CopyExistingMaintenancesToNewMaintenancesAndRules(existingMaintenances []*existingMaintenance, orgID string, ruleIDToRuleUUIDMap map[int]valuer.UUID) ([]*newMaintenance, []*storablePlannedMaintenanceRule, error) {
	newMaintenances := make([]*newMaintenance, 0)
	newMaintenanceRules := make([]*storablePlannedMaintenanceRule, 0)

	for _, maintenance := range existingMaintenances {
		ruleIDs := maintenance.AlertIDs
		maintenanceUUID := valuer.GenerateUUID()
		newMaintenance := newMaintenance{
			Identifiable: types.Identifiable{
				ID: maintenanceUUID,
			},
			TimeAuditable: types.TimeAuditable{
				CreatedAt: maintenance.CreatedAt,
				UpdatedAt: maintenance.UpdatedAt,
			},
			UserAuditable: types.UserAuditable{
				CreatedBy: maintenance.CreatedBy,
				UpdatedBy: maintenance.UpdatedBy,
			},
			Name:        maintenance.Name,
			Description: maintenance.Description,
			Schedule:    maintenance.Schedule,
			OrgID:       orgID,
		}
		newMaintenances = append(newMaintenances, &newMaintenance)
		for _, ruleIDStr := range *ruleIDs {
			ruleID, err := strconv.Atoi(ruleIDStr)
			if err != nil {
				return nil, nil, err
			}

			newMaintenanceRules = append(newMaintenanceRules, &storablePlannedMaintenanceRule{
				Identifiable: types.Identifiable{
					ID: valuer.GenerateUUID(),
				},
				PlannedMaintenanceID: maintenanceUUID,
				RuleID:               ruleIDToRuleUUIDMap[ruleID],
			})
		}
	}
	return newMaintenances, newMaintenanceRules, nil
}

type updateOrganizations struct {
	store sqlstore.SQLStore
}

func NewUpdateOrganizationsFactory(sqlstore sqlstore.SQLStore) factory.ProviderFactory[SQLMigration, Config] {
	return factory.NewProviderFactory(factory.MustNewName("update_organizations"), func(ctx context.Context, ps factory.ProviderSettings, c Config) (SQLMigration, error) {
		return newUpdateOrganizations(ctx, ps, c, sqlstore)
	})
}

func newUpdateOrganizations(_ context.Context, _ factory.ProviderSettings, _ Config, store sqlstore.SQLStore) (SQLMigration, error) {
	return &updateOrganizations{store: store}, nil
}

func (migration *updateOrganizations) Register(migrations *migrate.Migrations) error {
	if err := migrations.Register(migration.Up, migration.Down); err != nil {
		return err
	}

	return nil
}

func (migration *updateOrganizations) Up(ctx context.Context, db *bun.DB) error {
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}

	defer func() {
		_ = tx.Rollback()
	}()

	err = migration.
		store.
		Dialect().
		DropColumn(ctx, tx, "organizations", "is_anonymous")
	if err != nil {
		return err
	}

	err = migration.
		store.
		Dialect().
		DropColumn(ctx, tx, "organizations", "has_opted_updates")
	if err != nil {
		return err
	}

	_, err = migration.
		store.
		Dialect().
		RenameColumn(ctx, tx, "organizations", "name", "display_name")
	if err != nil {
		return err
	}

	err = migration.
		store.
		Dialect().
		AddColumn(ctx, tx, "organizations", "name", "TEXT")
	if err != nil {
		return err
	}

	_, err = tx.
		NewCreateIndex().
		Unique().
		IfNotExists().
		Index("idx_unique_name").
		Table("organizations").
		Column("name").
		Exec(ctx)
	if err != nil {
		return err
	}

	err = migration.
		store.
		Dialect().
		AddColumn(ctx, tx, "organizations", "alias", "TEXT")
	if err != nil {
		return err
	}
	_, err = tx.
		NewCreateIndex().
		Unique().
		IfNotExists().
		Index("idx_unique_alias").
		Table("organizations").
		Column("alias").
		Exec(ctx)
	if err != nil {
		return err
	}

	err = tx.Commit()
	if err != nil {
		return err
	}

	return nil
}

func (migration *updateOrganizations) Down(context.Context, *bun.DB) error {
	return nil
}

type dropGroups struct {
	sqlstore sqlstore.SQLStore
}

func NewDropGroupsFactory(sqlstore sqlstore.SQLStore) factory.ProviderFactory[SQLMigration, Config] {
	return factory.NewProviderFactory(factory.MustNewName("drop_groups"), func(ctx context.Context, providerSettings factory.ProviderSettings, config Config) (SQLMigration, error) {
		return newDropGroups(ctx, providerSettings, config, sqlstore)
	})
}

func newDropGroups(_ context.Context, _ factory.ProviderSettings, _ Config, sqlstore sqlstore.SQLStore) (SQLMigration, error) {
	return &dropGroups{sqlstore: sqlstore}, nil
}

func (migration *dropGroups) Register(migrations *migrate.Migrations) error {
	if err := migrations.Register(migration.Up, migration.Down); err != nil {
		return err
	}

	return nil
}

func (migration *dropGroups) Up(ctx context.Context, db *bun.DB) error {
	type Group struct {
		bun.BaseModel `bun:"table:groups"`

		types.TimeAuditable
		OrgID string `bun:"org_id,type:text"`
		ID    string `bun:"id,pk,type:text" json:"id"`
		Name  string `bun:"name,type:text,notnull,unique" json:"name"`
	}

	exists, err := migration.sqlstore.Dialect().TableExists(ctx, db, new(Group))
	if err != nil {
		return err
	}

	if !exists {
		return nil
	}

	// Disable foreign keys temporarily
	if err := migration.sqlstore.Dialect().ToggleForeignKeyConstraint(ctx, db, false); err != nil {
		return err
	}

	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}

	defer func() {
		_ = tx.Rollback()
	}()

	type existingUser struct {
		bun.BaseModel `bun:"table:users"`

		types.TimeAuditable
		ID                string `bun:"id,pk,type:text" json:"id"`
		Name              string `bun:"name,type:text,notnull" json:"name"`
		Email             string `bun:"email,type:text,notnull,unique" json:"email"`
		Password          string `bun:"password,type:text,notnull" json:"-"`
		ProfilePictureURL string `bun:"profile_picture_url,type:text" json:"profilePictureURL"`
		GroupID           string `bun:"group_id,type:text,notnull" json:"groupId"`
		OrgID             string `bun:"org_id,type:text,notnull" json:"orgId"`
	}

	var existingUsers []*existingUser
	if err := tx.
		NewSelect().
		Model(&existingUsers).
		Scan(ctx); err != nil {
		return err
	}

	var groups []*Group
	if err := tx.
		NewSelect().
		Model(&groups).
		Scan(ctx); err != nil {
		return err
	}

	groupIDToRoleMap := make(map[string]string)
	for _, group := range groups {
		groupIDToRoleMap[group.ID] = group.Name
	}

	roleToUserIDMap := make(map[string][]string)
	for _, user := range existingUsers {
		roleToUserIDMap[groupIDToRoleMap[user.GroupID]] = append(roleToUserIDMap[groupIDToRoleMap[user.GroupID]], user.ID)
	}

	if err := migration.sqlstore.Dialect().DropColumnWithForeignKeyConstraint(ctx, tx, new(struct {
		bun.BaseModel `bun:"table:users"`

		types.TimeAuditable
		ID                string `bun:"id,pk,type:text"`
		Name              string `bun:"name,type:text,notnull"`
		Email             string `bun:"email,type:text,notnull,unique"`
		Password          string `bun:"password,type:text,notnull"`
		ProfilePictureURL string `bun:"profile_picture_url,type:text"`
		OrgID             string `bun:"org_id,type:text,notnull"`
	}), "group_id"); err != nil {
		return err
	}

	if err := migration.sqlstore.Dialect().AddColumn(ctx, tx, "users", "role", "TEXT"); err != nil {
		return err
	}

	for role, userIDs := range roleToUserIDMap {
		if _, err := tx.
			NewUpdate().
			Table("users").
			Set("role = ?", role).
			Where("id IN (?)", bun.In(userIDs)).
			Exec(ctx); err != nil {
			return err
		}
	}

	if err := migration.sqlstore.Dialect().AddNotNullDefaultToColumn(ctx, tx, "users", "role", "TEXT", "'VIEWER'"); err != nil {
		return err
	}

	if _, err := tx.
		NewDropTable().
		Table("groups").
		IfExists().
		Exec(ctx); err != nil {
		return err
	}

	if err := tx.Commit(); err != nil {
		return err
	}

	// Enable foreign keys
	if err := migration.sqlstore.Dialect().ToggleForeignKeyConstraint(ctx, db, true); err != nil {
		return err
	}

	return nil
}

func (migration *dropGroups) Down(ctx context.Context, db *bun.DB) error {
	return nil
}

type createQuickFilters struct {
	store sqlstore.SQLStore
}

type quickFilter struct {
	bun.BaseModel `bun:"table:quick_filter"`
	types.Identifiable
	OrgID  string `bun:"org_id,notnull,unique:org_id_signal,type:text"`
	Filter string `bun:"filter,notnull,type:text"`
	Signal string `bun:"signal,notnull,unique:org_id_signal,type:text"`
	types.TimeAuditable
	types.UserAuditable
}

func NewCreateQuickFiltersFactory(store sqlstore.SQLStore) factory.ProviderFactory[SQLMigration, Config] {
	return factory.NewProviderFactory(factory.MustNewName("create_quick_filters"), func(ctx context.Context, ps factory.ProviderSettings, c Config) (SQLMigration, error) {
		return &createQuickFilters{store: store}, nil
	})
}

func (m *createQuickFilters) Register(migrations *migrate.Migrations) error {
	return migrations.Register(m.Up, m.Down)
}

func (m *createQuickFilters) Up(ctx context.Context, db *bun.DB) error {
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer func() {
		_ = tx.Rollback()
	}()

	// Create table if not exists
	_, err = tx.NewCreateTable().
		Model((*quickFilter)(nil)).
		IfNotExists().
		ForeignKey(`("org_id") REFERENCES "organizations" ("id") ON DELETE CASCADE ON UPDATE CASCADE`).
		Exec(ctx)
	if err != nil {
		return err
	}

	// Get default organization ID
	var defaultOrg valuer.UUID
	err = tx.NewSelect().Table("organizations").Column("id").Limit(1).Scan(ctx, &defaultOrg)
	if err != nil {
		if err == sql.ErrNoRows {
			// No organizations found, nothing to insert, commit and return
			err := tx.Commit()
			if err != nil {
				return err
			}
			return nil
		}
		return err
	}

	// Get the default quick filters
	storableQuickFilters, err := quickfiltertypes.NewDefaultQuickFilter(defaultOrg)
	if err != nil {
		return err
	}

	// Insert all filters at once
	_, err = tx.NewInsert().
		Model(&storableQuickFilters).
		Exec(ctx)

	if err != nil {
		if errors.Ast(m.store.WrapAlreadyExistsErrf(err, errors.CodeAlreadyExists, "Quick Filter already exists"), errors.TypeAlreadyExists) {
			err := tx.Commit()
			if err != nil {
				return err
			}
			return nil
		}
		return err
	}

	// Commit the transaction
	return tx.Commit()
}

func (m *createQuickFilters) Down(ctx context.Context, db *bun.DB) error {
	return nil
}

type updateQuickFilters struct {
	store sqlstore.SQLStore
}

func NewUpdateQuickFiltersFactory(store sqlstore.SQLStore) factory.ProviderFactory[SQLMigration, Config] {
	return factory.NewProviderFactory(factory.MustNewName("update_quick_filters"), func(ctx context.Context, ps factory.ProviderSettings, c Config) (SQLMigration, error) {
		return newUpdateQuickFilters(ctx, ps, c, store)
	})
}

func newUpdateQuickFilters(_ context.Context, _ factory.ProviderSettings, _ Config, store sqlstore.SQLStore) (SQLMigration, error) {
	return &updateQuickFilters{
		store: store,
	}, nil
}

func (migration *updateQuickFilters) Register(migrations *migrate.Migrations) error {
	if err := migrations.Register(migration.Up, migration.Down); err != nil {
		return err
	}

	return nil
}

func (migration *updateQuickFilters) Up(ctx context.Context, db *bun.DB) error {
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}

	defer func() {
		_ = tx.Rollback()
	}()

	// Delete all existing quick filters
	_, err = tx.NewDelete().
		Table("quick_filter").
		Where("1=1"). // Delete all rows
		Exec(ctx)
	if err != nil {
		return err
	}

	// Get all organization IDs as strings
	var orgIDs []string
	err = tx.NewSelect().
		Table("organizations").
		Column("id").
		Scan(ctx, &orgIDs)
	if err != nil {
		if err == sql.ErrNoRows {
			// No organizations found, commit the transaction (deletion is done) and return
			if err := tx.Commit(); err != nil {
				return err
			}
			return nil
		}
		return err
	}

	// For each organization, create new quick filters with the updated NewDefaultQuickFilter function
	for _, orgID := range orgIDs {
		// Get the updated default quick filters
		storableQuickFilters, err := quickfiltertypes.NewDefaultQuickFilter(valuer.MustNewUUID(orgID))
		if err != nil {
			return err
		}

		// Insert all filters for this organization
		_, err = tx.NewInsert().
			Model(&storableQuickFilters).
			Exec(ctx)

		if err != nil {
			if errors.Ast(migration.store.WrapAlreadyExistsErrf(err, errors.CodeAlreadyExists, "Quick Filter already exists"), errors.TypeAlreadyExists) {
				// Skip if filters already exist for this org
				continue
			}
			return err
		}
	}

	if err := tx.Commit(); err != nil {
		return err
	}
	return nil
}

func (migration *updateQuickFilters) Down(ctx context.Context, db *bun.DB) error {
	return nil
}

type authRefactor struct {
	store sqlstore.SQLStore
}

func NewAuthRefactorFactory(sqlstore sqlstore.SQLStore) factory.ProviderFactory[SQLMigration, Config] {
	return factory.NewProviderFactory(factory.MustNewName("auth_refactor"), func(ctx context.Context, ps factory.ProviderSettings, c Config) (SQLMigration, error) {
		return newAuthRefactor(ctx, ps, c, sqlstore)
	})
}

func newAuthRefactor(_ context.Context, _ factory.ProviderSettings, _ Config, store sqlstore.SQLStore) (SQLMigration, error) {
	return &authRefactor{store: store}, nil
}

func (migration *authRefactor) Register(migrations *migrate.Migrations) error {
	if err := migrations.Register(migration.Up, migration.Down); err != nil {
		return err
	}

	return nil
}

type existingUser32 struct {
	bun.BaseModel `bun:"table:users"`

	types.TimeAuditable
	ID                string `bun:"id,pk,type:text" json:"id"`
	Name              string `bun:"name,type:text,notnull" json:"name"`
	Email             string `bun:"email,type:text,notnull,unique" json:"email"`
	Password          string `bun:"password,type:text,notnull" json:"-"`
	ProfilePictureURL string `bun:"profile_picture_url,type:text" json:"profilePictureURL"`
	Role              string `bun:"role,type:text,notnull" json:"role"`
	OrgID             string `bun:"org_id,type:text,notnull" json:"orgId"`
}

type factorPassword32 struct {
	bun.BaseModel `bun:"table:factor_password"`

	types.Identifiable
	types.TimeAuditable
	Password  string `bun:"password,type:text,notnull" json:"password"`
	Temporary bool   `bun:"temporary,type:boolean,notnull" json:"temporary"`
	UserID    string `bun:"user_id,type:text,notnull" json:"userID"`
}

type existingResetPasswordRequest32 struct {
	bun.BaseModel `bun:"table:reset_password_request"`

	types.Identifiable
	Token  string `bun:"token,type:text,notnull" json:"token"`
	UserID string `bun:"user_id,type:text,notnull,unique" json:"userId"`
}

type newResetPasswordRequest32 struct {
	bun.BaseModel `bun:"table:reset_password_token"`

	types.Identifiable
	Token      string `bun:"token,type:text,notnull" json:"token"`
	PasswordID string `bun:"password_id,type:text,notnull" json:"passwordID"`
}

func (migration *authRefactor) Up(ctx context.Context, db *bun.DB) error {

	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}

	defer func() {
		_ = tx.Rollback()
	}()

	if _, err := tx.NewCreateTable().
		Model(new(factorPassword32)).
		ForeignKey(`("user_id") REFERENCES "users" ("id")`).
		IfNotExists().
		Exec(ctx); err != nil {
		return err
	}

	// copy passwords from users table to factor_password table
	err = migration.CopyOldPasswordToNewPassword(ctx, tx)
	if err != nil {
		return err
	}

	// delete profile picture url
	err = migration.store.Dialect().DropColumn(ctx, tx, "users", "profile_picture_url")
	if err != nil {
		return err
	}
	// delete password
	err = migration.store.Dialect().DropColumn(ctx, tx, "users", "password")
	if err != nil {
		return err
	}

	// rename name to display name
	_, err = migration.store.Dialect().RenameColumn(ctx, tx, "users", "name", "display_name")
	if err != nil {
		return err
	}

	err = migration.
		store.
		Dialect().
		RenameTableAndModifyModel(ctx, tx, new(existingResetPasswordRequest32), new(newResetPasswordRequest32), []string{FactorPasswordReference}, func(ctx context.Context) error {
			existingRequests := make([]*existingResetPasswordRequest32, 0)
			err = tx.
				NewSelect().
				Model(&existingRequests).
				Scan(ctx)
			if err != nil {
				if err != sql.ErrNoRows {
					return err
				}
			}

			if err == nil && len(existingRequests) > 0 {
				// copy users and their passwords to new table
				newRequests, err := migration.
					CopyOldResetPasswordToNewResetPassword(ctx, tx, existingRequests)
				if err != nil {
					return err
				}
				_, err = tx.
					NewInsert().
					Model(&newRequests).
					Exec(ctx)
				if err != nil {
					return err
				}
			}
			return nil
		})
	if err != nil {
		return err
	}

	err = tx.Commit()
	if err != nil {
		return err
	}
	return nil
}

func (migration *authRefactor) Down(context.Context, *bun.DB) error {
	return nil
}

func (migration *authRefactor) CopyOldPasswordToNewPassword(ctx context.Context, tx bun.IDB) error {
	// check if data already in factor_password table
	var count int64
	err := tx.NewSelect().Model(new(factorPassword32)).ColumnExpr("COUNT(*)").Scan(ctx, &count)
	if err != nil {
		return err
	}

	if count > 0 {
		return nil
	}

	// check if password column exist in the users table.
	exists, err := migration.store.Dialect().ColumnExists(ctx, tx, "users", "password")
	if err != nil {
		return err
	}
	if !exists {
		return nil
	}

	// get all users from users table
	existingUsers := make([]*existingUser32, 0)
	err = tx.NewSelect().Model(&existingUsers).Scan(ctx)
	if err != nil {
		return err
	}

	newPasswords := make([]*factorPassword32, 0)
	for _, user := range existingUsers {
		newPasswords = append(newPasswords, &factorPassword32{
			Identifiable: types.Identifiable{
				ID: valuer.GenerateUUID(),
			},
			Password:  user.Password,
			Temporary: false,
			UserID:    user.ID,
		})
	}

	// insert
	if len(newPasswords) > 0 {
		_, err = tx.NewInsert().Model(&newPasswords).Exec(ctx)
		if err != nil {
			return err
		}
	}

	return nil
}

func (migration *authRefactor) CopyOldResetPasswordToNewResetPassword(ctx context.Context, tx bun.IDB, existingRequests []*existingResetPasswordRequest32) ([]*newResetPasswordRequest32, error) {
	newRequests := make([]*newResetPasswordRequest32, 0)
	for _, request := range existingRequests {
		// get password id from user id
		var passwordID string
		err := tx.NewSelect().Table("factor_password").Column("id").Where("user_id = ?", request.UserID).Scan(ctx, &passwordID)
		if err != nil {
			return nil, err
		}

		newRequests = append(newRequests, &newResetPasswordRequest32{
			Identifiable: types.Identifiable{
				ID: valuer.GenerateUUID(),
			},
			Token:      request.Token,
			PasswordID: passwordID,
		})
	}
	return newRequests, nil
}

type migratePATToFactorAPIKey struct {
	store sqlstore.SQLStore
}

func NewMigratePATToFactorAPIKey(sqlstore sqlstore.SQLStore) factory.ProviderFactory[SQLMigration, Config] {
	return factory.NewProviderFactory(factory.MustNewName("migrate_pat_to_factor_api_key"), func(ctx context.Context, ps factory.ProviderSettings, c Config) (SQLMigration, error) {
		return newMigratePATToFactorAPIKey(ctx, ps, c, sqlstore)
	})
}

func newMigratePATToFactorAPIKey(_ context.Context, _ factory.ProviderSettings, _ Config, store sqlstore.SQLStore) (SQLMigration, error) {
	return &migratePATToFactorAPIKey{store: store}, nil
}

func (migration *migratePATToFactorAPIKey) Register(migrations *migrate.Migrations) error {
	if err := migrations.Register(migration.Up, migration.Down); err != nil {
		return err
	}

	return nil
}

type existingPersonalAccessToken33 struct {
	bun.BaseModel `bun:"table:personal_access_token"`

	types.Identifiable
	types.TimeAuditable
	OrgID           string `json:"orgId" bun:"org_id,type:text,notnull"`
	Role            string `json:"role" bun:"role,type:text,notnull,default:'ADMIN'"`
	UserID          string `json:"userId" bun:"user_id,type:text,notnull"`
	Token           string `json:"token" bun:"token,type:text,notnull,unique"`
	Name            string `json:"name" bun:"name,type:text,notnull"`
	ExpiresAt       int64  `json:"expiresAt" bun:"expires_at,notnull,default:0"`
	LastUsed        int64  `json:"lastUsed" bun:"last_used,notnull,default:0"`
	Revoked         bool   `json:"revoked" bun:"revoked,notnull,default:false"`
	UpdatedByUserID string `json:"updatedByUserId" bun:"updated_by_user_id,type:text,notnull,default:''"`
}

// we are removing the connection with org,
// the reason we are doing this is the api keys should just have
// one foreign key, we don't want a dangling state where, an API key
// belongs to one org and some user which doesn't belong to that org.
// so going ahead with directly attaching it to user will help dangling states.
type newFactorAPIKey33 struct {
	bun.BaseModel `bun:"table:factor_api_key"`

	types.Identifiable
	CreatedAt time.Time `bun:"created_at,notnull,nullzero,type:timestamptz" json:"createdAt"`
	UpdatedAt time.Time `bun:"updated_at,notnull,nullzero,type:timestamptz" json:"updatedAt"`
	CreatedBy string    `bun:"created_by,notnull" json:"createdBy"`
	UpdatedBy string    `bun:"updated_by,notnull" json:"updatedBy"`
	Token     string    `json:"token" bun:"token,type:text,notnull,unique"`
	Role      string    `json:"role" bun:"role,type:text,notnull"`
	Name      string    `json:"name" bun:"name,type:text,notnull"`
	ExpiresAt time.Time `json:"expiresAt" bun:"expires_at,notnull,nullzero,type:timestamptz"`
	LastUsed  time.Time `json:"lastUsed" bun:"last_used,notnull,nullzero,type:timestamptz"`
	Revoked   bool      `json:"revoked" bun:"revoked,notnull,default:false"`
	UserID    string    `json:"userId" bun:"user_id,type:text,notnull"`
}

func (migration *migratePATToFactorAPIKey) Up(ctx context.Context, db *bun.DB) error {
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}

	defer func() {
		_ = tx.Rollback()
	}()

	err = migration.
		store.
		Dialect().
		RenameTableAndModifyModel(ctx, tx, new(existingPersonalAccessToken33), new(newFactorAPIKey33), []string{UserReferenceNoCascade}, func(ctx context.Context) error {
			existingAPIKeys := make([]*existingPersonalAccessToken33, 0)
			err = tx.
				NewSelect().
				Model(&existingAPIKeys).
				Scan(ctx)
			if err != nil && err != sql.ErrNoRows {
				return err
			}

			if err == nil && len(existingAPIKeys) > 0 {
				newAPIKeys, err := migration.
					CopyOldPatToFactorAPIKey(ctx, tx, existingAPIKeys)
				if err != nil {
					return err
				}
				_, err = tx.
					NewInsert().
					Model(&newAPIKeys).
					Exec(ctx)
				if err != nil {
					return err
				}
			}
			return nil
		})
	if err != nil {
		return err
	}

	err = tx.Commit()
	if err != nil {
		return err
	}

	return nil
}

func (migration *migratePATToFactorAPIKey) Down(ctx context.Context, db *bun.DB) error {
	return nil
}

func (migration *migratePATToFactorAPIKey) CopyOldPatToFactorAPIKey(ctx context.Context, tx bun.IDB, existingAPIKeys []*existingPersonalAccessToken33) ([]*newFactorAPIKey33, error) {
	newAPIKeys := make([]*newFactorAPIKey33, 0)
	for _, apiKey := range existingAPIKeys {

		if apiKey.CreatedAt.IsZero() {
			apiKey.CreatedAt = time.Now()
		}
		if apiKey.UpdatedAt.IsZero() {
			apiKey.UpdatedAt = time.Now()
		}

		// convert expiresAt and lastUsed to time.Time
		expiresAt := time.Unix(apiKey.ExpiresAt, 0)
		lastUsed := time.Unix(apiKey.LastUsed, 0)
		if apiKey.LastUsed == 0 {
			lastUsed = apiKey.CreatedAt
		}

		newAPIKeys = append(newAPIKeys, &newFactorAPIKey33{
			Identifiable: apiKey.Identifiable,
			CreatedAt:    apiKey.CreatedAt,
			UpdatedAt:    apiKey.UpdatedAt,
			CreatedBy:    apiKey.UserID,
			UpdatedBy:    apiKey.UpdatedByUserID,
			Token:        apiKey.Token,
			Role:         apiKey.Role,
			Name:         apiKey.Name,
			ExpiresAt:    expiresAt,
			LastUsed:     lastUsed,
			Revoked:      apiKey.Revoked,
			UserID:       apiKey.UserID,
		})
	}
	return newAPIKeys, nil
}

type updateLicense struct {
	store sqlstore.SQLStore
}

type existingLicense34 struct {
	bun.BaseModel `bun:"table:licenses_v3"`

	ID   string `bun:"id,pk,type:text"`
	Key  string `bun:"key,type:text,notnull,unique"`
	Data string `bun:"data,type:text"`
}

type newLicense34 struct {
	bun.BaseModel `bun:"table:license"`

	types.Identifiable
	types.TimeAuditable
	Key             string         `bun:"key,type:text,notnull,unique"`
	Data            map[string]any `bun:"data,type:text"`
	LastValidatedAt time.Time      `bun:"last_validated_at,notnull"`
	OrgID           string         `bun:"org_id,type:text,notnull" json:"orgID"`
}

func NewUpdateLicenseFactory(store sqlstore.SQLStore) factory.ProviderFactory[SQLMigration, Config] {
	return factory.NewProviderFactory(factory.MustNewName("update_license"), func(ctx context.Context, ps factory.ProviderSettings, c Config) (SQLMigration, error) {
		return newUpdateLicense(ctx, ps, c, store)
	})
}

func newUpdateLicense(_ context.Context, _ factory.ProviderSettings, _ Config, store sqlstore.SQLStore) (SQLMigration, error) {
	return &updateLicense{store: store}, nil
}

func (migration *updateLicense) Register(migrations *migrate.Migrations) error {
	if err := migrations.Register(migration.Up, migration.Down); err != nil {
		return err
	}

	return nil
}

func (migration *updateLicense) Up(ctx context.Context, db *bun.DB) error {
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}

	defer func() {
		_ = tx.Rollback()
	}()

	err = migration.store.Dialect().RenameTableAndModifyModel(ctx, tx, new(existingLicense34), new(newLicense34), []string{OrgReference}, func(ctx context.Context) error {
		existingLicenses := make([]*existingLicense34, 0)
		err = tx.NewSelect().Model(&existingLicenses).Scan(ctx)
		if err != nil {
			if err != sql.ErrNoRows {
				return err
			}
		}

		if err == nil && len(existingLicenses) > 0 {
			var orgID string
			err := migration.
				store.
				BunDB().
				NewSelect().
				Model((*types.Organization)(nil)).
				Column("id").
				Scan(ctx, &orgID)
			if err != nil {
				if err != sql.ErrNoRows {
					return err
				}
			}
			if err == nil {
				newLicenses, err := migration.CopyExistingLicensesToNewLicenses(existingLicenses, orgID)
				if err != nil {
					return err
				}
				_, err = tx.
					NewInsert().
					Model(&newLicenses).
					Exec(ctx)
				if err != nil {
					return err
				}
			}
			return nil
		}
		return nil
	})

	err = tx.Commit()
	if err != nil {
		return err
	}

	return nil
}

func (migration *updateLicense) Down(context.Context, *bun.DB) error {
	return nil
}

func (migration *updateLicense) CopyExistingLicensesToNewLicenses(existingLicenses []*existingLicense34, orgID string) ([]*newLicense34, error) {
	newLicenses := make([]*newLicense34, len(existingLicenses))
	for idx, existingLicense := range existingLicenses {
		licenseID, err := valuer.NewUUID(existingLicense.ID)
		if err != nil {
			return nil, errors.Wrapf(err, errors.TypeInvalidInput, errors.CodeInvalidInput, "license id is not a valid UUID: %s", existingLicense.ID)
		}
		licenseData := map[string]any{}
		err = json.Unmarshal([]byte(existingLicense.Data), &licenseData)
		if err != nil {
			return nil, errors.Wrapf(err, errors.TypeInvalidInput, errors.CodeInvalidInput, "unable to unmarshal license data in map[string]any")
		}
		newLicenses[idx] = &newLicense34{
			Identifiable: types.Identifiable{
				ID: licenseID,
			},
			TimeAuditable: types.TimeAuditable{
				CreatedAt: time.Now(),
				UpdatedAt: time.Now(),
			},
			Key:             existingLicense.Key,
			Data:            licenseData,
			LastValidatedAt: time.Now(),
			OrgID:           orgID,
		}
	}
	return newLicenses, nil
}

type updateApiMonitoringFilters struct {
	store sqlstore.SQLStore
}

func NewUpdateApiMonitoringFiltersFactory(store sqlstore.SQLStore) factory.ProviderFactory[SQLMigration, Config] {
	return factory.NewProviderFactory(factory.MustNewName("update_api_monitoring_filters"), func(ctx context.Context, ps factory.ProviderSettings, c Config) (SQLMigration, error) {
		return newUpdateApiMonitoringFilters(ctx, ps, c, store)
	})
}

func newUpdateApiMonitoringFilters(_ context.Context, _ factory.ProviderSettings, _ Config, store sqlstore.SQLStore) (SQLMigration, error) {
	return &updateApiMonitoringFilters{
		store: store,
	}, nil
}

func (migration *updateApiMonitoringFilters) Register(migrations *migrate.Migrations) error {
	if err := migrations.Register(migration.Up, migration.Down); err != nil {
		return err
	}

	return nil
}

func (migration *updateApiMonitoringFilters) Up(ctx context.Context, db *bun.DB) error {
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}

	defer func() {
		_ = tx.Rollback()
	}()

	// Get all organization IDs as strings
	var orgIDs []string
	err = tx.NewSelect().
		Table("organizations").
		Column("id").
		Scan(ctx, &orgIDs)
	if err != nil {
		if err == sql.ErrNoRows {
			if err := tx.Commit(); err != nil {
				return err
			}
			return nil
		}
		return err
	}

	for _, orgID := range orgIDs {
		// Get the updated default quick filters which includes the new API monitoring filters
		storableQuickFilters, err := quickfiltertypes.NewDefaultQuickFilter(valuer.MustNewUUID(orgID))
		if err != nil {
			return err
		}

		// Find the API monitoring filter from the storable quick filters
		var apiMonitoringFilterJSON string
		for _, filter := range storableQuickFilters {
			if filter.Signal == quickfiltertypes.SignalApiMonitoring {
				apiMonitoringFilterJSON = filter.Filter
				break
			}
		}

		if apiMonitoringFilterJSON != "" {
			_, err = tx.NewUpdate().
				Table("quick_filter").
				Set("filter = ?, updated_at = ?", apiMonitoringFilterJSON, time.Now()).
				Where("signal = ? AND org_id = ?", quickfiltertypes.SignalApiMonitoring, orgID).
				Exec(ctx)

			if err != nil {
				return err
			}
		}
	}

	if err := tx.Commit(); err != nil {
		return err
	}
	return nil
}

func (migration *updateApiMonitoringFilters) Down(ctx context.Context, db *bun.DB) error {
	return nil
}

type addKeyOrganization struct {
	sqlstore sqlstore.SQLStore
}

func NewAddKeyOrganizationFactory(sqlstore sqlstore.SQLStore) factory.ProviderFactory[SQLMigration, Config] {
	return factory.NewProviderFactory(factory.MustNewName("add_key_organization"), func(ctx context.Context, providerSettings factory.ProviderSettings, config Config) (SQLMigration, error) {
		return newAddKeyOrganization(ctx, providerSettings, config, sqlstore)
	})
}

func newAddKeyOrganization(_ context.Context, _ factory.ProviderSettings, _ Config, sqlstore sqlstore.SQLStore) (SQLMigration, error) {
	return &addKeyOrganization{
		sqlstore: sqlstore,
	}, nil
}

func (migration *addKeyOrganization) Register(migrations *migrate.Migrations) error {
	if err := migrations.Register(migration.Up, migration.Down); err != nil {
		return err
	}

	return nil
}

func (migration *addKeyOrganization) Up(ctx context.Context, db *bun.DB) error {
	ok, err := migration.sqlstore.Dialect().ColumnExists(ctx, db, "organizations", "key")
	if err != nil {
		return err
	}

	if ok {
		return nil
	}

	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}

	defer func() {
		_ = tx.Rollback()
	}()

	if _, err := tx.
		NewAddColumn().
		Table("organizations").
		ColumnExpr("key BIGINT").
		Exec(ctx); err != nil {
		return err
	}

	var existingOrgIDs []string
	if err := tx.NewSelect().
		Table("organizations").
		Column("id").
		Scan(ctx, &existingOrgIDs); err != nil {
		return err
	}

	for _, orgID := range existingOrgIDs {
		key := migration.getHash(ctx, orgID)
		if _, err := tx.
			NewUpdate().
			Table("organizations").
			Set("key = ?", key).
			Where("id = ?", orgID).
			Exec(ctx); err != nil {
			return err
		}
	}

	if _, err := tx.
		NewCreateIndex().
		Unique().
		IfNotExists().
		Index("idx_unique_key").
		Table("organizations").
		Column("key").
		Exec(ctx); err != nil {
		return err
	}

	if err := tx.Commit(); err != nil {
		return err
	}

	return nil
}

func (migration *addKeyOrganization) Down(ctx context.Context, db *bun.DB) error {
	return nil
}

func (migration *addKeyOrganization) getHash(_ context.Context, orgID string) uint32 {
	hasher := fnv.New32a()

	// Hasher never returns err.
	_, _ = hasher.Write([]byte(orgID))
	return hasher.Sum32()
}

// funnel Core Data Structure (funnel and funnelStep)
type funnel struct {
	bun.BaseModel      `bun:"table:trace_funnel"`
	types.Identifiable // funnel id
	types.TimeAuditable
	types.UserAuditable
	Name          string       `json:"funnel_name" bun:"name,type:text,notnull"` // funnel name
	Description   string       `json:"description" bun:"description,type:text"`  // funnel description
	OrgID         valuer.UUID  `json:"org_id" bun:"org_id,type:varchar,notnull"`
	Steps         []funnelStep `json:"steps" bun:"steps,type:text,notnull"`
	Tags          string       `json:"tags" bun:"tags,type:text"`
	CreatedByUser *types.User  `json:"user" bun:"rel:belongs-to,join:created_by=id"`
}

type funnelStep struct {
	types.Identifiable
	Name           string `json:"name,omitempty"`        // step name
	Description    string `json:"description,omitempty"` // step description
	Order          int64  `json:"step_order"`
	ServiceName    string `json:"service_name"`
	SpanName       string `json:"span_name"`
	Filters        string `json:"filters,omitempty"`
	LatencyPointer string `json:"latency_pointer,omitempty"`
	LatencyType    string `json:"latency_type,omitempty"`
	HasErrors      bool   `json:"has_errors"`
}

type addTraceFunnels struct {
	sqlstore sqlstore.SQLStore
}

func NewAddTraceFunnelsFactory(sqlstore sqlstore.SQLStore) factory.ProviderFactory[SQLMigration, Config] {
	return factory.NewProviderFactory(factory.MustNewName("add_trace_funnels"), func(ctx context.Context, providerSettings factory.ProviderSettings, config Config) (SQLMigration, error) {
		return newAddTraceFunnels(ctx, providerSettings, config, sqlstore)
	})
}

func newAddTraceFunnels(_ context.Context, _ factory.ProviderSettings, _ Config, sqlstore sqlstore.SQLStore) (SQLMigration, error) {
	return &addTraceFunnels{sqlstore: sqlstore}, nil
}

func (migration *addTraceFunnels) Register(migrations *migrate.Migrations) error {
	if err := migrations.Register(migration.Up, migration.Down); err != nil {
		return err
	}
	return nil
}

func (migration *addTraceFunnels) Up(ctx context.Context, db *bun.DB) error {
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer func() {
		_ = tx.Rollback()
	}()

	_, err = tx.NewCreateTable().
		Model(new(funnel)).
		ForeignKey(`("org_id") REFERENCES "organizations" ("id") ON DELETE CASCADE`).
		IfNotExists().
		Exec(ctx)
	if err != nil {
		return err
	}

	err = tx.Commit()
	if err != nil {
		return err
	}

	return nil
}

func (migration *addTraceFunnels) Down(ctx context.Context, db *bun.DB) error {
	return nil
}

type updateDashboard struct {
	store sqlstore.SQLStore
}

type existingDashboard36 struct {
	bun.BaseModel `bun:"table:dashboards"`

	types.TimeAuditable
	types.UserAuditable
	OrgID  string                 `json:"-" bun:"org_id,notnull"`
	ID     int                    `json:"id" bun:"id,pk,autoincrement"`
	UUID   string                 `json:"uuid" bun:"uuid,type:text,notnull,unique"`
	Data   map[string]interface{} `json:"data" bun:"data,type:text,notnull"`
	Locked *int                   `json:"isLocked" bun:"locked,notnull,default:0"`
}

type newDashboard36 struct {
	bun.BaseModel `bun:"table:dashboard"`

	types.Identifiable
	types.TimeAuditable
	types.UserAuditable
	Data   map[string]interface{} `bun:"data,type:text,notnull"`
	Locked bool                   `bun:"locked,notnull,default:false"`
	OrgID  valuer.UUID            `bun:"org_id,type:text,notnull"`
}

func NewUpdateDashboardFactory(store sqlstore.SQLStore) factory.ProviderFactory[SQLMigration, Config] {
	return factory.NewProviderFactory(factory.MustNewName("update_dashboards"), func(ctx context.Context, ps factory.ProviderSettings, c Config) (SQLMigration, error) {
		return newUpdateDashboard(ctx, ps, c, store)
	})
}

func newUpdateDashboard(_ context.Context, _ factory.ProviderSettings, _ Config, store sqlstore.SQLStore) (SQLMigration, error) {
	return &updateDashboard{store: store}, nil
}

func (migration *updateDashboard) Register(migrations *migrate.Migrations) error {
	if err := migrations.Register(migration.Up, migration.Down); err != nil {
		return err
	}

	return nil
}

func (migration *updateDashboard) Up(ctx context.Context, db *bun.DB) error {
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}

	defer func() {
		_ = tx.Rollback()
	}()

	err = migration.store.Dialect().RenameTableAndModifyModel(ctx, tx, new(existingDashboard36), new(newDashboard36), []string{OrgReference}, func(ctx context.Context) error {
		existingDashboards := make([]*existingDashboard36, 0)
		err = tx.NewSelect().Model(&existingDashboards).Scan(ctx)
		if err != nil {
			if err != sql.ErrNoRows {
				return err
			}
		}

		if err == nil && len(existingDashboards) > 0 {
			newDashboards, err := migration.CopyExistingDashboardsToNewDashboards(existingDashboards)
			if err != nil {
				return err
			}
			_, err = tx.
				NewInsert().
				Model(&newDashboards).
				Exec(ctx)
			if err != nil {
				return err
			}
		}

		return nil
	})
	if err != nil {
		return err
	}

	err = tx.Commit()
	if err != nil {
		return err
	}

	return nil

}
func (migration *updateDashboard) Down(context.Context, *bun.DB) error {
	return nil
}

func (migration *updateDashboard) CopyExistingDashboardsToNewDashboards(existingDashboards []*existingDashboard36) ([]*newDashboard36, error) {
	newDashboards := make([]*newDashboard36, len(existingDashboards))

	for idx, existingDashboard := range existingDashboards {
		dashboardID, err := valuer.NewUUID(existingDashboard.UUID)
		if err != nil {
			return nil, err
		}
		orgID, err := valuer.NewUUID(existingDashboard.OrgID)
		if err != nil {
			return nil, err
		}
		locked := false
		if existingDashboard.Locked != nil && *existingDashboard.Locked == 1 {
			locked = true
		}

		newDashboards[idx] = &newDashboard36{
			Identifiable: types.Identifiable{
				ID: dashboardID,
			},
			TimeAuditable: existingDashboard.TimeAuditable,
			UserAuditable: existingDashboard.UserAuditable,
			Data:          existingDashboard.Data,
			Locked:        locked,
			OrgID:         orgID,
		}
	}

	return newDashboards, nil
}

type dropFeatureSet struct{}

func NewDropFeatureSetFactory() factory.ProviderFactory[SQLMigration, Config] {
	return factory.NewProviderFactory(factory.MustNewName("drop_feature_set"), func(ctx context.Context, ps factory.ProviderSettings, c Config) (SQLMigration, error) {
		return newDropFeatureSet(ctx, ps, c)
	})
}

func newDropFeatureSet(_ context.Context, _ factory.ProviderSettings, _ Config) (SQLMigration, error) {
	return &dropFeatureSet{}, nil
}

func (migration *dropFeatureSet) Register(migrations *migrate.Migrations) error {
	if err := migrations.Register(migration.Up, migration.Down); err != nil {
		return err
	}

	return nil
}

func (migration *dropFeatureSet) Up(ctx context.Context, db *bun.DB) error {
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}

	defer func() {
		_ = tx.Rollback()
	}()

	if _, err := tx.
		NewDropTable().
		IfExists().
		Table("feature_status").
		Exec(ctx); err != nil {
		return err
	}

	if err := tx.Commit(); err != nil {
		return err
	}

	return nil
}

func (migration *dropFeatureSet) Down(context.Context, *bun.DB) error {
	return nil
}

type dropDeprecatedTables struct{}

func NewDropDeprecatedTablesFactory() factory.ProviderFactory[SQLMigration, Config] {
	return factory.NewProviderFactory(factory.MustNewName("drop_deprecated_tables"), func(ctx context.Context, ps factory.ProviderSettings, c Config) (SQLMigration, error) {
		return newDropDeprecatedTables(ctx, ps, c)
	})
}

func newDropDeprecatedTables(_ context.Context, _ factory.ProviderSettings, _ Config) (SQLMigration, error) {
	return &dropDeprecatedTables{}, nil
}

func (migration *dropDeprecatedTables) Register(migrations *migrate.Migrations) error {
	if err := migrations.Register(migration.Up, migration.Down); err != nil {
		return err
	}

	return nil
}

func (migration *dropDeprecatedTables) Up(ctx context.Context, db *bun.DB) error {
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}

	defer func() {
		_ = tx.Rollback()
	}()

	if _, err := tx.
		NewDropTable().
		IfExists().
		Table("rule_history").
		Exec(ctx); err != nil {
		return err
	}

	if _, err := tx.
		NewDropTable().
		IfExists().
		Table("data_migrations").
		Exec(ctx); err != nil {
		return err
	}

	if err := tx.Commit(); err != nil {
		return err
	}

	return nil
}

func (migration *dropDeprecatedTables) Down(context.Context, *bun.DB) error {
	return nil
}

type updateAgents struct {
	store sqlstore.SQLStore
}
type newAgent41 struct {
	bun.BaseModel `bun:"table:agent"`

	types.Identifiable
	types.TimeAuditable
	// AgentID is needed as the ID from opamp client is ULID and not UUID, so we are keeping it like this
	AgentID      string                 `json:"agentId" yaml:"agentId" bun:"agent_id,type:text,notnull,unique"`
	OrgID        string                 `json:"orgId" yaml:"orgId" bun:"org_id,type:text,notnull"`
	TerminatedAt time.Time              `json:"terminatedAt" yaml:"terminatedAt" bun:"terminated_at"`
	Status       opamptypes.AgentStatus `json:"currentStatus" yaml:"currentStatus" bun:"status,type:text,notnull"`
	Config       string                 `bun:"config,type:text,notnull"`
}

type existingAgentConfigVersions41 struct {
	bun.BaseModel  `bun:"table:agent_config_versions"`
	ID             string                  `bun:"id,pk,type:text"`
	CreatedBy      string                  `bun:"created_by,type:text"`
	CreatedAt      time.Time               `bun:"created_at,default:CURRENT_TIMESTAMP"`
	UpdatedBy      string                  `bun:"updated_by,type:text"`
	UpdatedAt      time.Time               `bun:"updated_at,default:CURRENT_TIMESTAMP"`
	Version        int                     `bun:"version,default:1,unique:element_version_idx"`
	Active         int                     `bun:"active"`
	IsValid        int                     `bun:"is_valid"`
	Disabled       int                     `bun:"disabled"`
	ElementType    opamptypes.ElementType  `bun:"element_type,notnull,type:varchar(120),unique:element_version_idx"`
	DeployStatus   opamptypes.DeployStatus `bun:"deploy_status,notnull,type:varchar(80),default:'DIRTY'"`
	DeploySequence int                     `bun:"deploy_sequence"`
	DeployResult   string                  `bun:"deploy_result,type:text"`
	LastHash       string                  `bun:"last_hash,type:text"`
	LastConfig     string                  `bun:"last_config,type:text"`
}

type newAgentConfigVersion41 struct {
	bun.BaseModel `bun:"table:agent_config_version"`

	types.Identifiable
	types.TimeAuditable
	types.UserAuditable
	OrgID          string                  `json:"orgId" bun:"org_id,type:text,notnull,unique:element_version_org_idx"`
	Version        int                     `json:"version" bun:"version,unique:element_version_org_idx"`
	ElementType    opamptypes.ElementType  `json:"elementType" bun:"element_type,type:text,notnull,unique:element_version_org_idx"`
	DeployStatus   opamptypes.DeployStatus `json:"deployStatus" bun:"deploy_status,type:text,notnull,default:'dirty'"`
	DeploySequence int                     `json:"deploySequence" bun:"deploy_sequence"`
	DeployResult   string                  `json:"deployResult" bun:"deploy_result,type:text"`
	Hash           string                  `json:"lastHash" bun:"hash,type:text"`
	Config         string                  `json:"config" bun:"config,type:text"`
}

type existingAgentConfigElement41 struct {
	bun.BaseModel `bun:"table:agent_config_elements"`

	ID          string    `bun:"id,pk,type:text"`
	CreatedBy   string    `bun:"created_by,type:text"`
	CreatedAt   time.Time `bun:"created_at,default:CURRENT_TIMESTAMP"`
	UpdatedBy   string    `bun:"updated_by,type:text"`
	UpdatedAt   time.Time `bun:"updated_at,default:CURRENT_TIMESTAMP"`
	ElementID   string    `bun:"element_id,type:text,notnull,unique:agent_config_elements_u1"`
	ElementType string    `bun:"element_type,type:varchar(120),notnull,unique:agent_config_elements_u1"`
	VersionID   string    `bun:"version_id,type:text,notnull,unique:agent_config_elements_u1"`
}

type newAgentConfigElement41 struct {
	bun.BaseModel `bun:"table:agent_config_element"`

	types.Identifiable
	types.TimeAuditable
	ElementID   string `bun:"element_id,type:text,notnull,unique:element_type_version_idx"`
	ElementType string `bun:"element_type,type:text,notnull,unique:element_type_version_idx"`
	VersionID   string `bun:"version_id,type:text,notnull,unique:element_type_version_idx"`
}

func NewUpdateAgentsFactory(sqlstore sqlstore.SQLStore) factory.ProviderFactory[SQLMigration, Config] {
	return factory.NewProviderFactory(factory.MustNewName("update_agents"), func(ctx context.Context, ps factory.ProviderSettings, c Config) (SQLMigration, error) {
		return newUpdateAgents(ctx, ps, c, sqlstore)
	})
}

func newUpdateAgents(_ context.Context, _ factory.ProviderSettings, _ Config, store sqlstore.SQLStore) (SQLMigration, error) {
	return &updateAgents{
		store: store,
	}, nil
}

func (migration *updateAgents) Register(migrations *migrate.Migrations) error {
	if err := migrations.Register(migration.Up, migration.Down); err != nil {
		return err
	}

	return nil
}

func (migration *updateAgents) Up(ctx context.Context, db *bun.DB) error {

	// begin transaction
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}

	defer func() {
		_ = tx.Rollback()
	}()

	var orgID string
	err = tx.
		NewSelect().
		ColumnExpr("id").
		Table("organizations").
		Limit(1).
		Scan(ctx, &orgID)
	if err != nil {
		if err != sql.ErrNoRows {
			return err
		}
	}

	if _, err := tx.
		NewDropTable().
		IfExists().
		Table("agents").
		Exec(ctx); err != nil {
		return err
	}

	if _, err := tx.
		NewCreateTable().
		IfNotExists().
		Model(new(newAgent41)).
		Exec(ctx); err != nil {
		return err
	}

	err = migration.
		store.
		Dialect().
		RenameTableAndModifyModel(ctx, tx, new(existingAgentConfigVersions41), new(newAgentConfigVersion41), []string{OrgReference}, func(ctx context.Context) error {
			existingAgentConfigVersions := make([]*existingAgentConfigVersions41, 0)
			err = tx.
				NewSelect().
				Model(&existingAgentConfigVersions).
				Scan(ctx)
			if err != nil && err != sql.ErrNoRows {
				return err
			}

			if err == nil && len(existingAgentConfigVersions) > 0 {
				newAgentConfigVersions, err := migration.
					CopyOldAgentConfigVersionToNewAgentConfigVersion(ctx, tx, existingAgentConfigVersions, orgID)
				if err != nil {
					return err
				}
				_, err = tx.
					NewInsert().
					Model(&newAgentConfigVersions).
					Exec(ctx)
				if err != nil {
					return err
				}
			}
			return nil
		})
	if err != nil {
		return err
	}

	err = migration.
		store.
		Dialect().
		RenameTableAndModifyModel(ctx, tx, new(existingAgentConfigElement41), new(newAgentConfigElement41), []string{AgentConfigVersionReference}, func(ctx context.Context) error {
			existingAgentConfigElements := make([]*existingAgentConfigElement41, 0)
			err = tx.
				NewSelect().
				Model(&existingAgentConfigElements).
				Scan(ctx)
			if err != nil && err != sql.ErrNoRows {
				return err
			}

			if err == nil && len(existingAgentConfigElements) > 0 {
				newAgentConfigElements, err := migration.
					CopyOldAgentConfigElementToNewAgentConfigElement(ctx, tx, existingAgentConfigElements, orgID)
				if err != nil {
					return err
				}
				_, err = tx.
					NewInsert().
					Model(&newAgentConfigElements).
					Exec(ctx)
				if err != nil {
					return err
				}
			}
			return nil
		})
	if err != nil {
		return err
	}

	if err := tx.Commit(); err != nil {
		return err
	}

	return nil
}

func (migration *updateAgents) Down(ctx context.Context, db *bun.DB) error {
	return nil
}

func (migration *updateAgents) CopyOldAgentConfigVersionToNewAgentConfigVersion(ctx context.Context, tx bun.IDB, existingAgentConfigVersions []*existingAgentConfigVersions41, orgID string) ([]*newAgentConfigVersion41, error) {
	newAgentConfigVersions := make([]*newAgentConfigVersion41, 0)
	for _, existingAgentConfigVersion := range existingAgentConfigVersions {
		versionID, err := valuer.NewUUID(existingAgentConfigVersion.ID)
		if err != nil {
			return nil, err
		}
		newAgentConfigVersions = append(newAgentConfigVersions, &newAgentConfigVersion41{
			Identifiable: types.Identifiable{ID: versionID},
			TimeAuditable: types.TimeAuditable{
				CreatedAt: time.Unix(existingAgentConfigVersion.CreatedAt.Unix(), 0),
				UpdatedAt: time.Unix(existingAgentConfigVersion.UpdatedAt.Unix(), 0),
			},
			UserAuditable: types.UserAuditable{
				CreatedBy: existingAgentConfigVersion.CreatedBy,
				UpdatedBy: existingAgentConfigVersion.UpdatedBy,
			},
			OrgID:          orgID,
			Version:        existingAgentConfigVersion.Version,
			ElementType:    existingAgentConfigVersion.ElementType,
			DeployStatus:   existingAgentConfigVersion.DeployStatus,
			DeploySequence: existingAgentConfigVersion.DeploySequence,
			DeployResult:   existingAgentConfigVersion.DeployResult,
			Hash:           orgID + existingAgentConfigVersion.LastHash,
			Config:         existingAgentConfigVersion.LastConfig,
		})
	}
	return newAgentConfigVersions, nil
}

func (migration *updateAgents) CopyOldAgentConfigElementToNewAgentConfigElement(ctx context.Context, tx bun.IDB, existingAgentConfigElements []*existingAgentConfigElement41, orgID string) ([]*newAgentConfigElement41, error) {
	newAgentConfigElements := make([]*newAgentConfigElement41, 0)
	for _, existingAgentConfigElement := range existingAgentConfigElements {
		elementID, err := valuer.NewUUID(existingAgentConfigElement.ElementID)
		if err != nil {
			return nil, err
		}
		newAgentConfigElements = append(newAgentConfigElements, &newAgentConfigElement41{
			Identifiable: types.Identifiable{ID: elementID},
			TimeAuditable: types.TimeAuditable{
				CreatedAt: time.Unix(existingAgentConfigElement.CreatedAt.Unix(), 0),
				UpdatedAt: time.Unix(existingAgentConfigElement.UpdatedAt.Unix(), 0),
			},
			VersionID:   existingAgentConfigElement.VersionID,
			ElementID:   existingAgentConfigElement.ElementID,
			ElementType: existingAgentConfigElement.ElementType,
		})
	}
	return newAgentConfigElements, nil
}

type updateUsers struct {
	sqlstore  sqlstore.SQLStore
	sqlschema sqlschema.SQLSchema
}

func NewUpdateUsersFactory(sqlstore sqlstore.SQLStore, sqlschema sqlschema.SQLSchema) factory.ProviderFactory[SQLMigration, Config] {
	return factory.NewProviderFactory(factory.MustNewName("update_users"), func(ctx context.Context, providerSettings factory.ProviderSettings, config Config) (SQLMigration, error) {
		return newUpdateUsers(ctx, providerSettings, config, sqlstore, sqlschema)
	})
}

func newUpdateUsers(_ context.Context, _ factory.ProviderSettings, _ Config, sqlstore sqlstore.SQLStore, sqlschema sqlschema.SQLSchema) (SQLMigration, error) {
	return &updateUsers{
		sqlstore:  sqlstore,
		sqlschema: sqlschema,
	}, nil
}

func (migration *updateUsers) Register(migrations *migrate.Migrations) error {
	if err := migrations.Register(migration.Up, migration.Down); err != nil {
		return err
	}

	return nil
}

func (migration *updateUsers) Up(ctx context.Context, db *bun.DB) error {
	if err := migration.sqlschema.ToggleFKEnforcement(ctx, db, false); err != nil {
		return err
	}

	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}

	defer func() {
		_ = tx.Rollback()
	}()

	table, uniqueConstraints, err := migration.sqlschema.GetTable(ctx, sqlschema.TableName("users"))
	if err != nil {
		return err
	}

	sqls := [][]byte{}

	dropSQLs := migration.sqlschema.Operator().DropConstraint(table, uniqueConstraints, &sqlschema.UniqueConstraint{ColumnNames: []sqlschema.ColumnName{"email"}})
	sqls = append(sqls, dropSQLs...)

	indexSQLs := migration.sqlschema.Operator().CreateIndex(&sqlschema.UniqueIndex{TableName: "users", ColumnNames: []sqlschema.ColumnName{"email", "org_id"}})
	sqls = append(sqls, indexSQLs...)

	for _, sql := range sqls {
		if _, err := tx.ExecContext(ctx, string(sql)); err != nil {
			return err
		}
	}

	if err := tx.Commit(); err != nil {
		return err
	}

	if err := migration.sqlschema.ToggleFKEnforcement(ctx, db, true); err != nil {
		return err
	}

	return nil
}

func (migration *updateUsers) Down(ctx context.Context, db *bun.DB) error {
	return nil
}

type updateUserInvite struct {
	sqlstore  sqlstore.SQLStore
	sqlschema sqlschema.SQLSchema
}

func NewUpdateUserInviteFactory(sqlstore sqlstore.SQLStore, sqlschema sqlschema.SQLSchema) factory.ProviderFactory[SQLMigration, Config] {
	return factory.NewProviderFactory(factory.MustNewName("update_user_invite"), func(ctx context.Context, providerSettings factory.ProviderSettings, config Config) (SQLMigration, error) {
		return newUpdateUserInvite(ctx, providerSettings, config, sqlstore, sqlschema)
	})
}

func newUpdateUserInvite(_ context.Context, _ factory.ProviderSettings, _ Config, sqlstore sqlstore.SQLStore, sqlschema sqlschema.SQLSchema) (SQLMigration, error) {
	return &updateUserInvite{
		sqlstore:  sqlstore,
		sqlschema: sqlschema,
	}, nil
}

func (migration *updateUserInvite) Register(migrations *migrate.Migrations) error {
	if err := migrations.Register(migration.Up, migration.Down); err != nil {
		return err
	}

	return nil
}

func (migration *updateUserInvite) Up(ctx context.Context, db *bun.DB) error {
	if err := migration.sqlschema.ToggleFKEnforcement(ctx, db, false); err != nil {
		return err
	}

	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}

	defer func() {
		_ = tx.Rollback()
	}()

	table, uniqueConstraints, err := migration.sqlschema.GetTable(ctx, sqlschema.TableName("user_invite"))
	if err != nil {
		return err
	}

	sqls := [][]byte{}

	dropSQLs := migration.sqlschema.Operator().DropConstraint(table, uniqueConstraints, &sqlschema.UniqueConstraint{ColumnNames: []sqlschema.ColumnName{"email"}})
	sqls = append(sqls, dropSQLs...)

	indexSQLs := migration.sqlschema.Operator().CreateIndex(&sqlschema.UniqueIndex{TableName: "user_invite", ColumnNames: []sqlschema.ColumnName{"email", "org_id"}})
	sqls = append(sqls, indexSQLs...)

	indexSQLs = migration.sqlschema.Operator().CreateIndex(&sqlschema.UniqueIndex{TableName: "user_invite", ColumnNames: []sqlschema.ColumnName{"token"}})
	sqls = append(sqls, indexSQLs...)

	for _, sql := range sqls {
		if _, err := tx.ExecContext(ctx, string(sql)); err != nil {
			return err
		}
	}

	if err := tx.Commit(); err != nil {
		return err
	}

	if err := migration.sqlschema.ToggleFKEnforcement(ctx, db, true); err != nil {
		return err
	}

	return nil
}

func (migration *updateUserInvite) Down(ctx context.Context, db *bun.DB) error {
	return nil
}

type updateOrgDomain struct {
	sqlstore  sqlstore.SQLStore
	sqlschema sqlschema.SQLSchema
}

func NewUpdateOrgDomainFactory(sqlstore sqlstore.SQLStore, sqlschema sqlschema.SQLSchema) factory.ProviderFactory[SQLMigration, Config] {
	return factory.NewProviderFactory(factory.MustNewName("update_org_domain"), func(ctx context.Context, providerSettings factory.ProviderSettings, config Config) (SQLMigration, error) {
		return newUpdateOrgDomain(ctx, providerSettings, config, sqlstore, sqlschema)
	})
}

func newUpdateOrgDomain(_ context.Context, _ factory.ProviderSettings, _ Config, sqlstore sqlstore.SQLStore, sqlschema sqlschema.SQLSchema) (SQLMigration, error) {
	return &updateOrgDomain{
		sqlstore:  sqlstore,
		sqlschema: sqlschema,
	}, nil
}

func (migration *updateOrgDomain) Register(migrations *migrate.Migrations) error {
	if err := migrations.Register(migration.Up, migration.Down); err != nil {
		return err
	}

	return nil
}

func (migration *updateOrgDomain) Up(ctx context.Context, db *bun.DB) error {
	if err := migration.sqlschema.ToggleFKEnforcement(ctx, db, false); err != nil {
		return err
	}

	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}

	defer func() {
		_ = tx.Rollback()
	}()

	table, uniqueConstraints, err := migration.sqlschema.GetTable(ctx, sqlschema.TableName("org_domains"))
	if err != nil {
		return err
	}

	sqls := [][]byte{}

	dropSQLs := migration.sqlschema.Operator().DropConstraint(table, uniqueConstraints, &sqlschema.UniqueConstraint{ColumnNames: []sqlschema.ColumnName{"name"}})
	sqls = append(sqls, dropSQLs...)

	indexSQLs := migration.sqlschema.Operator().CreateIndex(&sqlschema.UniqueIndex{TableName: "org_domains", ColumnNames: []sqlschema.ColumnName{"name", "org_id"}})
	sqls = append(sqls, indexSQLs...)

	for _, sql := range sqls {
		if _, err := tx.ExecContext(ctx, string(sql)); err != nil {
			return err
		}
	}

	if err := tx.Commit(); err != nil {
		return err
	}

	if err := migration.sqlschema.ToggleFKEnforcement(ctx, db, true); err != nil {
		return err
	}

	return nil
}

func (migration *updateOrgDomain) Down(ctx context.Context, db *bun.DB) error {
	return nil
}

type addFactorIndexes struct {
	sqlstore  sqlstore.SQLStore
	sqlschema sqlschema.SQLSchema
}

func NewAddFactorIndexesFactory(sqlstore sqlstore.SQLStore, sqlschema sqlschema.SQLSchema) factory.ProviderFactory[SQLMigration, Config] {
	return factory.NewProviderFactory(factory.MustNewName("add_factor_indexes"), func(ctx context.Context, providerSettings factory.ProviderSettings, config Config) (SQLMigration, error) {
		return newAddFactorIndexes(ctx, providerSettings, config, sqlstore, sqlschema)
	})
}

func newAddFactorIndexes(_ context.Context, _ factory.ProviderSettings, _ Config, sqlstore sqlstore.SQLStore, sqlschema sqlschema.SQLSchema) (SQLMigration, error) {
	return &addFactorIndexes{
		sqlstore:  sqlstore,
		sqlschema: sqlschema,
	}, nil
}

func (migration *addFactorIndexes) Register(migrations *migrate.Migrations) error {
	if err := migrations.Register(migration.Up, migration.Down); err != nil {
		return err
	}

	return nil
}

func (migration *addFactorIndexes) Up(ctx context.Context, db *bun.DB) error {
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}

	defer func() {
		_ = tx.Rollback()
	}()

	sqls := [][]byte{}

	indexSQLs := migration.sqlschema.Operator().CreateIndex(&sqlschema.UniqueIndex{TableName: "factor_password", ColumnNames: []sqlschema.ColumnName{"user_id"}})
	sqls = append(sqls, indexSQLs...)

	indexSQLs = migration.sqlschema.Operator().CreateIndex(&sqlschema.UniqueIndex{TableName: "reset_password_token", ColumnNames: []sqlschema.ColumnName{"password_id"}})
	sqls = append(sqls, indexSQLs...)

	indexSQLs = migration.sqlschema.Operator().CreateIndex(&sqlschema.UniqueIndex{TableName: "reset_password_token", ColumnNames: []sqlschema.ColumnName{"token"}})
	sqls = append(sqls, indexSQLs...)

	for _, sql := range sqls {
		if _, err := tx.ExecContext(ctx, string(sql)); err != nil {
			return err
		}
	}

	if err := tx.Commit(); err != nil {
		return err
	}

	return nil
}

func (migration *addFactorIndexes) Down(ctx context.Context, db *bun.DB) error {
	return nil
}

type queryBuilderV5Migration struct {
	store          sqlstore.SQLStore
	telemetryStore telemetrystore.TelemetryStore
	logger         *slog.Logger
}

func NewQueryBuilderV5MigrationFactory(
	store sqlstore.SQLStore,
	telemetryStore telemetrystore.TelemetryStore,
) factory.ProviderFactory[SQLMigration, Config] {
	return factory.NewProviderFactory(
		factory.MustNewName("query_builder_v5_migration"),
		func(ctx context.Context, ps factory.ProviderSettings, c Config) (SQLMigration, error) {
			return newQueryBuilderV5Migration(ctx, c, store, telemetryStore, ps.Logger)
		})
}

func newQueryBuilderV5Migration(
	_ context.Context,
	_ Config, store sqlstore.SQLStore,
	telemetryStore telemetrystore.TelemetryStore,
	logger *slog.Logger,
) (SQLMigration, error) {
	return &queryBuilderV5Migration{store: store, telemetryStore: telemetryStore, logger: logger}, nil
}

func (migration *queryBuilderV5Migration) Register(migrations *migrate.Migrations) error {
	if err := migrations.Register(migration.Up, migration.Down); err != nil {
		return err
	}
	return nil
}

func (migration *queryBuilderV5Migration) getTraceDuplicateKeys(ctx context.Context) ([]string, error) {
	query := `
		SELECT tagKey
		FROM signoz_traces.span_attributes_keys
		WHERE tagType IN ('tag', 'resource')
		GROUP BY tagKey
		HAVING COUNT(DISTINCT tagType) > 1
		ORDER BY tagKey
	`

	rows, err := migration.telemetryStore.ClickhouseDB().Query(ctx, query)
	if err != nil {
		migration.logger.WarnContext(ctx, "failed to query trace duplicate keys", errors.Attr(err))
		return nil, nil
	}
	defer rows.Close()

	var keys []string
	for rows.Next() {
		var key string
		if err := rows.Scan(&key); err != nil {
			migration.logger.WarnContext(ctx, "failed to scan trace duplicate key", errors.Attr(err))
			continue
		}
		keys = append(keys, key)
	}

	return keys, nil
}

func (migration *queryBuilderV5Migration) getLogDuplicateKeys(ctx context.Context) ([]string, error) {
	query := `
		SELECT name
		FROM (
			SELECT DISTINCT name FROM signoz_logs.logs_attribute_keys
			INTERSECT
			SELECT DISTINCT name FROM signoz_logs.logs_resource_keys
		)
		ORDER BY name
	`

	rows, err := migration.telemetryStore.ClickhouseDB().Query(ctx, query)
	if err != nil {
		migration.logger.WarnContext(ctx, "failed to query log duplicate keys", errors.Attr(err))
		return nil, nil
	}
	defer rows.Close()

	var keys []string
	for rows.Next() {
		var key string
		if err := rows.Scan(&key); err != nil {
			migration.logger.WarnContext(ctx, "failed to scan log duplicate key", errors.Attr(err))
			continue
		}
		keys = append(keys, key)
	}

	return keys, nil
}

func (migration *queryBuilderV5Migration) Up(ctx context.Context, db *bun.DB) error {

	// fetch keys that have both attribute and resource attribute types
	logsKeys, err := migration.getLogDuplicateKeys(ctx)
	if err != nil {
		return err
	}

	tracesKeys, err := migration.getTraceDuplicateKeys(ctx)
	if err != nil {
		return err
	}

	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}

	defer func() {
		_ = tx.Rollback()
	}()

	if err := migration.migrateSavedViews(ctx, tx, logsKeys, tracesKeys); err != nil {
		return err
	}

	if err := migration.migrateRules(ctx, tx, logsKeys, tracesKeys); err != nil {
		return err
	}

	return tx.Commit()
}

func (migration *queryBuilderV5Migration) Down(ctx context.Context, db *bun.DB) error {
	// this migration is not reversible as we're transforming the structure
	return nil
}

func (migration *queryBuilderV5Migration) migrateSavedViews(
	ctx context.Context,
	tx bun.Tx,
	logsKeys []string,
	tracesKeys []string,
) error {
	var savedViews []struct {
		ID   string `bun:"id"`
		Data string `bun:"data"`
	}

	err := tx.NewSelect().
		Table("saved_views").
		Column("id", "data").
		Scan(ctx, &savedViews)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil
		}
		return err
	}

	savedViewsMigrator := transition.NewSavedViewMigrateV5(migration.logger, logsKeys, tracesKeys)

	for _, savedView := range savedViews {
		var data map[string]any
		if err := json.Unmarshal([]byte(savedView.Data), &data); err != nil {
			continue // invalid JSON
		}

		updated := savedViewsMigrator.Migrate(ctx, data)

		if updated {
			dataJSON, err := json.Marshal(data)

			if err != nil {
				return err
			}

			_, err = tx.NewUpdate().
				Table("saved_views").
				Set("data = ?", string(dataJSON)).
				Where("id = ?", savedView.ID).
				Exec(ctx)
			if err != nil {
				return err
			}
		}
	}

	return nil
}

func (migration *queryBuilderV5Migration) migrateRules(
	ctx context.Context,
	tx bun.Tx,
	logsKeys []string,
	tracesKeys []string,
) error {
	// Fetch all rules
	var rules []struct {
		ID   string         `bun:"id"`
		Data map[string]any `bun:"data"`
	}

	err := tx.NewSelect().
		Table("rule").
		Column("id", "data").
		Scan(ctx, &rules)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil
		}
		return err
	}

	alertsMigrator := transition.NewAlertMigrateV5(migration.logger, logsKeys, tracesKeys)

	for _, rule := range rules {
		migration.logger.InfoContext(ctx, "migrating rule", slog.String("rule_id", rule.ID))

		updated := alertsMigrator.Migrate(ctx, rule.Data)

		if updated {
			dataJSON, err := json.Marshal(rule.Data)
			if err != nil {
				return err
			}

			_, err = tx.NewUpdate().
				Table("rule").
				Set("data = ?", string(dataJSON)).
				Where("id = ?", rule.ID).
				Exec(ctx)
			if err != nil {
				return err
			}
		}
	}

	return nil
}

type addMeterQuickFilters struct {
	sqlstore  sqlstore.SQLStore
	sqlschema sqlschema.SQLSchema
}

func NewAddMeterQuickFiltersFactory(sqlstore sqlstore.SQLStore, sqlschema sqlschema.SQLSchema) factory.ProviderFactory[SQLMigration, Config] {
	return factory.NewProviderFactory(factory.MustNewName("add_meter_quick_filters"), func(ctx context.Context, providerSettings factory.ProviderSettings, config Config) (SQLMigration, error) {
		return newAddMeterQuickFilters(ctx, providerSettings, config, sqlstore, sqlschema)
	})
}

func newAddMeterQuickFilters(_ context.Context, _ factory.ProviderSettings, _ Config, sqlstore sqlstore.SQLStore, sqlschema sqlschema.SQLSchema) (SQLMigration, error) {
	return &addMeterQuickFilters{
		sqlstore:  sqlstore,
		sqlschema: sqlschema,
	}, nil
}

func (migration *addMeterQuickFilters) Register(migrations *migrate.Migrations) error {
	if err := migrations.Register(migration.Up, migration.Down); err != nil {
		return err
	}

	return nil
}

func (migration *addMeterQuickFilters) Up(ctx context.Context, db *bun.DB) error {
	meterFilters := []map[string]interface{}{
		{"key": "deployment.environment", "dataType": "float64", "type": "Sum"},
		{"key": "service.name", "dataType": "float64", "type": "Sum"},
		{"key": "host.name", "dataType": "float64", "type": "Sum"},
	}

	meterJSON, err := json.Marshal(meterFilters)
	if err != nil {
		return errors.Wrapf(err, errors.TypeInternal, errors.CodeInternal, "failed to marshal meter filters")
	}

	type signal struct {
		valuer.String
	}

	type identifiable struct {
		ID valuer.UUID `json:"id" bun:"id,pk,type:text"`
	}

	type timeAuditable struct {
		CreatedAt time.Time `bun:"created_at" json:"createdAt"`
		UpdatedAt time.Time `bun:"updated_at" json:"updatedAt"`
	}

	type quickFilterType struct {
		bun.BaseModel `bun:"table:quick_filter"`
		identifiable
		OrgID  valuer.UUID `bun:"org_id,type:text,notnull"`
		Filter string      `bun:"filter,type:text,notnull"`
		Signal signal      `bun:"signal,type:text,notnull"`
		timeAuditable
	}

	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}

	defer func() {
		_ = tx.Rollback()
	}()

	var orgIDs []string
	err = tx.NewSelect().
		Table("organizations").
		Column("id").
		Scan(ctx, &orgIDs)
	if err != nil && err != sql.ErrNoRows {
		return err
	}

	var meterFiltersToInsert []quickFilterType
	for _, orgIDStr := range orgIDs {
		orgID, err := valuer.NewUUID(orgIDStr)
		if err != nil {
			return err
		}

		meterFiltersToInsert = append(meterFiltersToInsert, quickFilterType{
			identifiable: identifiable{
				ID: valuer.GenerateUUID(),
			},
			OrgID:  orgID,
			Filter: string(meterJSON),
			Signal: signal{valuer.NewString("meter")},
			timeAuditable: timeAuditable{
				CreatedAt: time.Now(),
				UpdatedAt: time.Now(),
			},
		})
	}

	if len(meterFiltersToInsert) > 0 {
		_, err = tx.NewInsert().
			Model(&meterFiltersToInsert).
			On("CONFLICT (org_id, signal) DO UPDATE").
			Set("filter = EXCLUDED.filter, updated_at = EXCLUDED.updated_at").
			Exec(ctx)
		if err != nil {
			return err
		}
	}

	if err := tx.Commit(); err != nil {
		return err
	}

	return nil
}

func (migration *addMeterQuickFilters) Down(ctx context.Context, db *bun.DB) error {
	return nil
}

type updateTTLSettingForCustomRetention struct {
	sqlstore  sqlstore.SQLStore
	sqlschema sqlschema.SQLSchema
}

func NewUpdateTTLSettingForCustomRetentionFactory(sqlstore sqlstore.SQLStore, sqlschema sqlschema.SQLSchema) factory.ProviderFactory[SQLMigration, Config] {
	return factory.NewProviderFactory(factory.MustNewName("update_ttl_setting"), func(ctx context.Context, providerSettings factory.ProviderSettings, config Config) (SQLMigration, error) {
		return newUpdateTTLSettingForCustomRetention(ctx, providerSettings, config, sqlstore, sqlschema)
	})
}

func newUpdateTTLSettingForCustomRetention(_ context.Context, _ factory.ProviderSettings, _ Config, sqlstore sqlstore.SQLStore, sqlschema sqlschema.SQLSchema) (SQLMigration, error) {
	return &updateTTLSettingForCustomRetention{
		sqlstore:  sqlstore,
		sqlschema: sqlschema,
	}, nil
}

func (migration *updateTTLSettingForCustomRetention) Register(migrations *migrate.Migrations) error {
	if err := migrations.Register(migration.Up, migration.Down); err != nil {
		return err
	}
	return nil
}

func (migration *updateTTLSettingForCustomRetention) Up(ctx context.Context, db *bun.DB) error {
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}

	defer func() {
		_ = tx.Rollback()
	}()

	// Get the table and its constraints
	table, _, err := migration.sqlschema.GetTable(ctx, sqlschema.TableName("ttl_setting"))
	if err != nil {
		return err
	}

	// Define the new column
	column := &sqlschema.Column{
		Name:     sqlschema.ColumnName("condition"),
		DataType: sqlschema.DataTypeText,
		Nullable: true,
	}

	sqls := migration.sqlschema.Operator().AddColumn(table, nil, column, nil)
	for _, sql := range sqls {
		if _, err := tx.ExecContext(ctx, string(sql)); err != nil {
			return err
		}
	}

	if err := tx.Commit(); err != nil {
		return err
	}

	return nil
}

func (migration *updateTTLSettingForCustomRetention) Down(ctx context.Context, db *bun.DB) error {
	return nil
}

type addAuthToken struct {
	sqlstore  sqlstore.SQLStore
	sqlschema sqlschema.SQLSchema
}

func NewAddAuthTokenFactory(sqlstore sqlstore.SQLStore, sqlschema sqlschema.SQLSchema) factory.ProviderFactory[SQLMigration, Config] {
	return factory.NewProviderFactory(factory.MustNewName("add_auth_token"), func(ctx context.Context, providerSettings factory.ProviderSettings, config Config) (SQLMigration, error) {
		return newAddAuthToken(ctx, providerSettings, config, sqlstore, sqlschema)
	})
}

func newAddAuthToken(_ context.Context, _ factory.ProviderSettings, _ Config, sqlstore sqlstore.SQLStore, sqlschema sqlschema.SQLSchema) (SQLMigration, error) {
	return &addAuthToken{
		sqlstore:  sqlstore,
		sqlschema: sqlschema,
	}, nil
}

func (migration *addAuthToken) Register(migrations *migrate.Migrations) error {
	if err := migrations.Register(migration.Up, migration.Down); err != nil {
		return err
	}

	return nil
}

func (migration *addAuthToken) Up(ctx context.Context, db *bun.DB) error {
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}

	defer func() {
		_ = tx.Rollback()
	}()

	token := &sqlschema.Table{
		Name: "auth_token",
		Columns: []*sqlschema.Column{
			{Name: sqlschema.ColumnName("id"), DataType: sqlschema.DataTypeText, Nullable: false, Default: ""},
			{Name: sqlschema.ColumnName("meta"), DataType: sqlschema.DataTypeText, Nullable: false, Default: ""},
			{Name: sqlschema.ColumnName("prev_access_token"), DataType: sqlschema.DataTypeText, Nullable: true, Default: ""},
			{Name: sqlschema.ColumnName("access_token"), DataType: sqlschema.DataTypeText, Nullable: false, Default: ""},
			{Name: sqlschema.ColumnName("prev_refresh_token"), DataType: sqlschema.DataTypeText, Nullable: true, Default: ""},
			{Name: sqlschema.ColumnName("refresh_token"), DataType: sqlschema.DataTypeText, Nullable: false, Default: ""},
			{Name: sqlschema.ColumnName("last_observed_at"), DataType: sqlschema.DataTypeTimestamp, Nullable: true, Default: ""},
			{Name: sqlschema.ColumnName("rotated_at"), DataType: sqlschema.DataTypeTimestamp, Nullable: true, Default: ""},
			{Name: sqlschema.ColumnName("created_at"), DataType: sqlschema.DataTypeTimestamp, Nullable: false, Default: ""},
			{Name: sqlschema.ColumnName("updated_at"), DataType: sqlschema.DataTypeTimestamp, Nullable: false, Default: ""},
			{Name: sqlschema.ColumnName("user_id"), DataType: sqlschema.DataTypeText, Nullable: false, Default: ""},
		},
		PrimaryKeyConstraint: &sqlschema.PrimaryKeyConstraint{ColumnNames: []sqlschema.ColumnName{sqlschema.ColumnName("id")}},
		ForeignKeyConstraints: []*sqlschema.ForeignKeyConstraint{
			{ReferencingColumnName: sqlschema.ColumnName("user_id"), ReferencedTableName: sqlschema.TableName("users"), ReferencedColumnName: sqlschema.ColumnName("id")},
		},
	}

	sqls := [][]byte{}
	createSQLs := migration.sqlschema.Operator().CreateTable(token)
	sqls = append(sqls, createSQLs...)

	indexSQLs := migration.sqlschema.Operator().CreateIndex(&sqlschema.UniqueIndex{TableName: "auth_token", ColumnNames: []sqlschema.ColumnName{"access_token"}})
	sqls = append(sqls, indexSQLs...)

	indexSQLs = migration.sqlschema.Operator().CreateIndex(&sqlschema.UniqueIndex{TableName: "auth_token", ColumnNames: []sqlschema.ColumnName{"refresh_token"}})
	sqls = append(sqls, indexSQLs...)

	for _, sql := range sqls {
		if _, err := tx.ExecContext(ctx, string(sql)); err != nil {
			return err
		}
	}

	if err := tx.Commit(); err != nil {
		return err
	}

	return nil
}

func (migration *addAuthToken) Down(ctx context.Context, db *bun.DB) error {
	return nil
}

type addAuthz struct {
	sqlstore  sqlstore.SQLStore
	sqlschema sqlschema.SQLSchema
}

func NewAddAuthzFactory(sqlstore sqlstore.SQLStore, sqlschema sqlschema.SQLSchema) factory.ProviderFactory[SQLMigration, Config] {
	return factory.NewProviderFactory(factory.MustNewName("add_authz"), func(ctx context.Context, ps factory.ProviderSettings, c Config) (SQLMigration, error) {
		return newAddAuthz(ctx, ps, c, sqlstore, sqlschema)
	})
}

func newAddAuthz(_ context.Context, _ factory.ProviderSettings, _ Config, sqlstore sqlstore.SQLStore, sqlschema sqlschema.SQLSchema) (SQLMigration, error) {
	return &addAuthz{
		sqlstore:  sqlstore,
		sqlschema: sqlschema,
	}, nil
}

func (migration *addAuthz) Register(migrations *migrate.Migrations) error {
	if err := migrations.Register(migration.Up, migration.Down); err != nil {
		return err
	}

	return nil
}

func (migration *addAuthz) Up(ctx context.Context, db *bun.DB) error {
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}

	defer func() {
		_ = tx.Rollback()
	}()

	sqls := [][]byte{}
	tableSQLs := migration.sqlschema.Operator().CreateTable(&sqlschema.Table{
		Name: "tuple",
		Columns: []*sqlschema.Column{
			{Name: "store", DataType: sqlschema.DataTypeText, Nullable: false},
			{Name: "object_type", DataType: sqlschema.DataTypeText, Nullable: false},
			{Name: "object_id", DataType: sqlschema.DataTypeText, Nullable: false},
			{Name: "relation", DataType: sqlschema.DataTypeText, Nullable: false},
			{Name: "user_object_type", DataType: sqlschema.DataTypeText, Nullable: false},
			{Name: "user_object_id", DataType: sqlschema.DataTypeText, Nullable: false},
			{Name: "user_relation", DataType: sqlschema.DataTypeText, Nullable: false},
			{Name: "user_type", DataType: sqlschema.DataTypeText, Nullable: false},
			{Name: "ulid", DataType: sqlschema.DataTypeText, Nullable: false},
			{Name: "inserted_at", DataType: sqlschema.DataTypeTimestamp, Nullable: false},
			{Name: "condition_name", DataType: sqlschema.DataTypeText, Nullable: true},
			{Name: "condition_context", DataType: sqlschema.DataTypeText, Nullable: true},
		},
		PrimaryKeyConstraint: &sqlschema.PrimaryKeyConstraint{ColumnNames: []sqlschema.ColumnName{"store", "object_type", "object_id", "relation", "user_object_type", "user_object_id", "user_relation"}},
	})
	sqls = append(sqls, tableSQLs...)

	indexSQLs := migration.sqlschema.Operator().CreateIndex(&sqlschema.UniqueIndex{TableName: "tuple", ColumnNames: []sqlschema.ColumnName{"ulid"}})
	sqls = append(sqls, indexSQLs...)

	tableSQLs = migration.sqlschema.Operator().CreateTable(&sqlschema.Table{
		Name: "authorization_model",
		Columns: []*sqlschema.Column{
			{Name: "store", DataType: sqlschema.DataTypeText, Nullable: false},
			{Name: "authorization_model_id", DataType: sqlschema.DataTypeText, Nullable: false},
			{Name: "schema_version", DataType: sqlschema.DataTypeText, Nullable: false, Default: "1.1"},
			{Name: "serialized_protobuf", DataType: sqlschema.DataTypeText, Nullable: false},
		},
		PrimaryKeyConstraint: &sqlschema.PrimaryKeyConstraint{ColumnNames: []sqlschema.ColumnName{"store", "authorization_model_id"}},
	})
	sqls = append(sqls, tableSQLs...)

	tableSQLs = migration.sqlschema.Operator().CreateTable(&sqlschema.Table{
		Name: "store",
		Columns: []*sqlschema.Column{
			{Name: "id", DataType: sqlschema.DataTypeText, Nullable: false},
			{Name: "name", DataType: sqlschema.DataTypeText, Nullable: false},
			{Name: "created_at", DataType: sqlschema.DataTypeTimestamp, Nullable: false},
			{Name: "updated_at", DataType: sqlschema.DataTypeTimestamp, Nullable: true},
			{Name: "deleted_at", DataType: sqlschema.DataTypeTimestamp, Nullable: true},
		},
		PrimaryKeyConstraint: &sqlschema.PrimaryKeyConstraint{ColumnNames: []sqlschema.ColumnName{"id"}},
	})
	sqls = append(sqls, tableSQLs...)

	tableSQLs = migration.sqlschema.Operator().CreateTable(&sqlschema.Table{
		Name: "assertion",
		Columns: []*sqlschema.Column{
			{Name: "store", DataType: sqlschema.DataTypeText, Nullable: false},
			{Name: "authorization_model_id", DataType: sqlschema.DataTypeText, Nullable: false},
			{Name: "assertions", DataType: sqlschema.DataTypeText, Nullable: true},
		},
		PrimaryKeyConstraint: &sqlschema.PrimaryKeyConstraint{ColumnNames: []sqlschema.ColumnName{"store", "authorization_model_id"}},
	})
	sqls = append(sqls, tableSQLs...)

	tableSQLs = migration.sqlschema.Operator().CreateTable(&sqlschema.Table{
		Name: "changelog",
		Columns: []*sqlschema.Column{
			{Name: "store", DataType: sqlschema.DataTypeText, Nullable: false},
			{Name: "object_type", DataType: sqlschema.DataTypeText, Nullable: false},
			{Name: "object_id", DataType: sqlschema.DataTypeText, Nullable: false},
			{Name: "relation", DataType: sqlschema.DataTypeText, Nullable: false},
			{Name: "user_object_type", DataType: sqlschema.DataTypeText, Nullable: false},
			{Name: "user_object_id", DataType: sqlschema.DataTypeText, Nullable: false},
			{Name: "user_relation", DataType: sqlschema.DataTypeText, Nullable: false},
			{Name: "operation", DataType: sqlschema.DataTypeText, Nullable: false},
			{Name: "ulid", DataType: sqlschema.DataTypeText, Nullable: false},
			{Name: "inserted_at", DataType: sqlschema.DataTypeTimestamp, Nullable: false},
			{Name: "condition_name", DataType: sqlschema.DataTypeText, Nullable: true},
			{Name: "condition_context", DataType: sqlschema.DataTypeText, Nullable: true},
		},
		PrimaryKeyConstraint: &sqlschema.PrimaryKeyConstraint{ColumnNames: []sqlschema.ColumnName{"store", "ulid", "object_type"}},
	})
	sqls = append(sqls, tableSQLs...)

	for _, sql := range sqls {
		if _, err := tx.ExecContext(ctx, string(sql)); err != nil {
			return err
		}
	}

	err = tx.Commit()
	if err != nil {
		return err
	}

	return nil
}

func (migration *addAuthz) Down(context.Context, *bun.DB) error {
	return nil
}

type addPublicDashboards struct {
	sqlstore  sqlstore.SQLStore
	sqlschema sqlschema.SQLSchema
}

func NewAddPublicDashboardsFactory(sqlstore sqlstore.SQLStore, sqlschema sqlschema.SQLSchema) factory.ProviderFactory[SQLMigration, Config] {
	return factory.NewProviderFactory(factory.MustNewName("add_public_dashboards"), func(ctx context.Context, ps factory.ProviderSettings, c Config) (SQLMigration, error) {
		return newAddPublicDashboards(ctx, ps, c, sqlstore, sqlschema)
	})
}

func newAddPublicDashboards(_ context.Context, _ factory.ProviderSettings, _ Config, sqlstore sqlstore.SQLStore, sqlschema sqlschema.SQLSchema) (SQLMigration, error) {
	return &addPublicDashboards{sqlstore: sqlstore, sqlschema: sqlschema}, nil
}

func (migration *addPublicDashboards) Register(migrations *migrate.Migrations) error {
	return migrations.Register(migration.Up, migration.Down)
}

func (migration *addPublicDashboards) Up(ctx context.Context, db *bun.DB) error {
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}

	defer func() {
		_ = tx.Rollback()
	}()

	sqls := [][]byte{}
	tableSQL := migration.sqlschema.Operator().CreateTable(&sqlschema.Table{
		Name: "public_dashboard",
		Columns: []*sqlschema.Column{
			{Name: "id", DataType: sqlschema.DataTypeText, Nullable: false},
			{Name: "time_range_enabled", DataType: sqlschema.DataTypeBoolean, Nullable: false},
			{Name: "default_time_range", DataType: sqlschema.DataTypeText, Nullable: false},
			{Name: "created_at", DataType: sqlschema.DataTypeTimestamp, Nullable: false},
			{Name: "updated_at", DataType: sqlschema.DataTypeTimestamp, Nullable: false},
			{Name: "dashboard_id", DataType: sqlschema.DataTypeText, Nullable: false},
		},
		PrimaryKeyConstraint: &sqlschema.PrimaryKeyConstraint{ColumnNames: []sqlschema.ColumnName{"id"}},
		ForeignKeyConstraints: []*sqlschema.ForeignKeyConstraint{
			{ReferencingColumnName: sqlschema.ColumnName("dashboard_id"), ReferencedTableName: sqlschema.TableName("dashboard"), ReferencedColumnName: sqlschema.ColumnName("id")},
		},
	})
	sqls = append(sqls, tableSQL...)

	indexSQL := migration.sqlschema.Operator().CreateIndex(&sqlschema.UniqueIndex{
		TableName:   "public_dashboard",
		ColumnNames: []sqlschema.ColumnName{"dashboard_id"},
	})
	sqls = append(sqls, indexSQL...)

	for _, sql := range sqls {
		if _, err := tx.ExecContext(ctx, string(sql)); err != nil {
			return err
		}
	}

	err = tx.Commit()
	if err != nil {
		return err
	}

	return nil
}

func (migration *addPublicDashboards) Down(_ context.Context, _ *bun.DB) error {
	return nil
}

type addRole struct {
	sqlstore  sqlstore.SQLStore
	sqlschema sqlschema.SQLSchema
}

func NewAddRoleFactory(sqlstore sqlstore.SQLStore, sqlschema sqlschema.SQLSchema) factory.ProviderFactory[SQLMigration, Config] {
	return factory.NewProviderFactory(factory.MustNewName("add_role"), func(ctx context.Context, providerSettings factory.ProviderSettings, config Config) (SQLMigration, error) {
		return newAddRole(ctx, providerSettings, config, sqlstore, sqlschema)
	})
}

func newAddRole(_ context.Context, _ factory.ProviderSettings, _ Config, sqlstore sqlstore.SQLStore, sqlschema sqlschema.SQLSchema) (SQLMigration, error) {
	return &addRole{
		sqlstore:  sqlstore,
		sqlschema: sqlschema,
	}, nil
}

func (migration *addRole) Register(migrations *migrate.Migrations) error {
	if err := migrations.Register(migration.Up, migration.Down); err != nil {
		return err
	}
	return nil
}

func (migration *addRole) Up(ctx context.Context, db *bun.DB) error {
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}

	defer func() {
		_ = tx.Rollback()
	}()

	sqls := [][]byte{}
	tableSQLs := migration.sqlschema.Operator().CreateTable(&sqlschema.Table{
		Name: "role",
		Columns: []*sqlschema.Column{
			{Name: "id", DataType: sqlschema.DataTypeText, Nullable: false},
			{Name: "created_at", DataType: sqlschema.DataTypeTimestamp, Nullable: false},
			{Name: "updated_at", DataType: sqlschema.DataTypeTimestamp, Nullable: false},
			{Name: "name", DataType: sqlschema.DataTypeText, Nullable: false},
			{Name: "description", DataType: sqlschema.DataTypeText, Nullable: true},
			{Name: "type", DataType: sqlschema.DataTypeText, Nullable: false},
			{Name: "org_id", DataType: sqlschema.DataTypeText, Nullable: false},
		},
		PrimaryKeyConstraint: &sqlschema.PrimaryKeyConstraint{
			ColumnNames: []sqlschema.ColumnName{"id"},
		},
		ForeignKeyConstraints: []*sqlschema.ForeignKeyConstraint{
			{
				ReferencingColumnName: sqlschema.ColumnName("org_id"),
				ReferencedTableName:   sqlschema.TableName("organizations"),
				ReferencedColumnName:  sqlschema.ColumnName("id"),
			},
		},
	})
	sqls = append(sqls, tableSQLs...)

	indexSQLs := migration.sqlschema.Operator().CreateIndex(&sqlschema.UniqueIndex{TableName: "role", ColumnNames: []sqlschema.ColumnName{"name", "org_id"}})
	sqls = append(sqls, indexSQLs...)

	for _, sqlStmt := range sqls {
		if _, err := tx.ExecContext(ctx, string(sqlStmt)); err != nil {
			return err
		}
	}

	if err := tx.Commit(); err != nil {
		return err
	}

	return nil
}

func (migration *addRole) Down(ctx context.Context, db *bun.DB) error {
	return nil
}

type updateAuthz struct {
	sqlstore  sqlstore.SQLStore
	sqlschema sqlschema.SQLSchema
}

func NewUpdateAuthzFactory(sqlstore sqlstore.SQLStore, sqlschema sqlschema.SQLSchema) factory.ProviderFactory[SQLMigration, Config] {
	return factory.NewProviderFactory(factory.MustNewName("update_authz"), func(ctx context.Context, ps factory.ProviderSettings, c Config) (SQLMigration, error) {
		return newUpdateAuthz(ctx, ps, c, sqlstore, sqlschema)
	})
}

func newUpdateAuthz(_ context.Context, _ factory.ProviderSettings, _ Config, sqlstore sqlstore.SQLStore, sqlschema sqlschema.SQLSchema) (SQLMigration, error) {
	return &updateAuthz{
		sqlstore:  sqlstore,
		sqlschema: sqlschema,
	}, nil
}

func (migration *updateAuthz) Register(migrations *migrate.Migrations) error {
	if err := migrations.Register(migration.Up, migration.Down); err != nil {
		return err
	}

	return nil
}

func (migration *updateAuthz) Up(ctx context.Context, db *bun.DB) error {
	if migration.sqlstore.BunDB().Dialect().Name() != dialect.PG {
		return nil
	}

	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}

	defer func() {
		_ = tx.Rollback()
	}()

	sqls := [][]byte{}

	table, _, err := migration.sqlschema.GetTable(ctx, sqlschema.TableName("tuple"))
	if err != nil {
		return err
	}
	dropTableSQLS := migration.sqlschema.Operator().DropTable(table)
	sqls = append(sqls, dropTableSQLS...)

	table, _, err = migration.sqlschema.GetTable(ctx, sqlschema.TableName("authorization_model"))
	if err != nil {
		return err
	}
	dropTableSQLS = migration.sqlschema.Operator().DropTable(table)
	sqls = append(sqls, dropTableSQLS...)

	table, _, err = migration.sqlschema.GetTable(ctx, sqlschema.TableName("changelog"))
	if err != nil {
		return err
	}
	dropTableSQLS = migration.sqlschema.Operator().DropTable(table)
	sqls = append(sqls, dropTableSQLS...)

	table, _, err = migration.sqlschema.GetTable(ctx, sqlschema.TableName("assertion"))
	if err != nil {
		return err
	}
	dropTableSQLS = migration.sqlschema.Operator().DropTable(table)
	sqls = append(sqls, dropTableSQLS...)

	tableSQLs := migration.sqlschema.Operator().CreateTable(&sqlschema.Table{
		Name: "tuple",
		Columns: []*sqlschema.Column{
			{Name: "store", DataType: sqlschema.DataTypeText, Nullable: false},
			{Name: "object_type", DataType: sqlschema.DataTypeText, Nullable: false},
			{Name: "object_id", DataType: sqlschema.DataTypeText, Nullable: false},
			{Name: "relation", DataType: sqlschema.DataTypeText, Nullable: false},
			{Name: "_user", DataType: sqlschema.DataTypeText, Nullable: false},
			{Name: "user_type", DataType: sqlschema.DataTypeText, Nullable: false},
			{Name: "ulid", DataType: sqlschema.DataTypeText, Nullable: false},
			{Name: "inserted_at", DataType: sqlschema.DataTypeTimestamp, Nullable: false},
			{Name: "condition_name", DataType: sqlschema.DataTypeBytea, Nullable: true},
			{Name: "condition_context", DataType: sqlschema.DataTypeText, Nullable: true},
		},
		PrimaryKeyConstraint: &sqlschema.PrimaryKeyConstraint{ColumnNames: []sqlschema.ColumnName{"store", "object_type", "object_id", "relation", "_user"}},
	})
	sqls = append(sqls, tableSQLs...)

	tableSQLs = migration.sqlschema.Operator().CreateTable(&sqlschema.Table{
		Name: "authorization_model",
		Columns: []*sqlschema.Column{
			{Name: "store", DataType: sqlschema.DataTypeText, Nullable: false},
			{Name: "authorization_model_id", DataType: sqlschema.DataTypeText, Nullable: false},
			{Name: "type", DataType: sqlschema.DataTypeText, Nullable: false},
			{Name: "type_definition", DataType: sqlschema.DataTypeBytea, Nullable: true},
			{Name: "schema_version", DataType: sqlschema.DataTypeText, Nullable: false, Default: "1.1"},
			{Name: "serialized_protobuf", DataType: sqlschema.DataTypeBytea, Nullable: false},
		},
		PrimaryKeyConstraint: &sqlschema.PrimaryKeyConstraint{ColumnNames: []sqlschema.ColumnName{"store", "authorization_model_id", "type"}},
	})
	sqls = append(sqls, tableSQLs...)

	tableSQLs = migration.sqlschema.Operator().CreateTable(&sqlschema.Table{
		Name: "changelog",
		Columns: []*sqlschema.Column{
			{Name: "store", DataType: sqlschema.DataTypeText, Nullable: false},
			{Name: "object_type", DataType: sqlschema.DataTypeText, Nullable: false},
			{Name: "object_id", DataType: sqlschema.DataTypeText, Nullable: false},
			{Name: "relation", DataType: sqlschema.DataTypeText, Nullable: false},
			{Name: "_user", DataType: sqlschema.DataTypeText, Nullable: false},
			{Name: "operation", DataType: sqlschema.DataTypeText, Nullable: false},
			{Name: "ulid", DataType: sqlschema.DataTypeText, Nullable: false},
			{Name: "inserted_at", DataType: sqlschema.DataTypeTimestamp, Nullable: false},
			{Name: "condition_name", DataType: sqlschema.DataTypeBytea, Nullable: true},
			{Name: "condition_context", DataType: sqlschema.DataTypeText, Nullable: true},
		},
		PrimaryKeyConstraint: &sqlschema.PrimaryKeyConstraint{ColumnNames: []sqlschema.ColumnName{"store", "ulid", "object_type"}},
	})
	sqls = append(sqls, tableSQLs...)

	tableSQLs = migration.sqlschema.Operator().CreateTable(&sqlschema.Table{
		Name: "assertion",
		Columns: []*sqlschema.Column{
			{Name: "store", DataType: sqlschema.DataTypeText, Nullable: false},
			{Name: "authorization_model_id", DataType: sqlschema.DataTypeText, Nullable: false},
			{Name: "assertions", DataType: sqlschema.DataTypeBytea, Nullable: true},
		},
		PrimaryKeyConstraint: &sqlschema.PrimaryKeyConstraint{ColumnNames: []sqlschema.ColumnName{"store", "authorization_model_id"}},
	})
	sqls = append(sqls, tableSQLs...)

	for _, sql := range sqls {
		if _, err := tx.ExecContext(ctx, string(sql)); err != nil {
			return err
		}
	}

	err = tx.Commit()
	if err != nil {
		return err
	}

	return nil
}

func (migration *updateAuthz) Down(context.Context, *bun.DB) error {
	return nil
}

type oldStorableUserPreference struct {
	bun.BaseModel `bun:"table:user_preference"`
	types.Identifiable
	Name   string      `bun:"preference_id,type:text,notnull"`
	Value  string      `bun:"preference_value,type:text,notnull"`
	UserID valuer.UUID `bun:"user_id,type:text,notnull"`
}

type newStorableUserPreference struct {
	bun.BaseModel `bun:"table:user_preference"`
	types.Identifiable
	Name   string      `bun:"name,type:text,notnull"`
	Value  string      `bun:"value,type:text,notnull"`
	UserID valuer.UUID `bun:"user_id,type:text,notnull"`
}

type updateUserPreference struct {
	sqlstore  sqlstore.SQLStore
	sqlschema sqlschema.SQLSchema
}

func NewUpdateUserPreferenceFactory(sqlstore sqlstore.SQLStore, sqlschema sqlschema.SQLSchema) factory.ProviderFactory[SQLMigration, Config] {
	return factory.NewProviderFactory(factory.MustNewName("update_user_preference"), func(ctx context.Context, settings factory.ProviderSettings, config Config) (SQLMigration, error) {
		return newupdateUserPreference(ctx, settings, config, sqlstore, sqlschema)
	})
}

func newupdateUserPreference(_ context.Context, _ factory.ProviderSettings, _ Config, sqlstore sqlstore.SQLStore, sqlschema sqlschema.SQLSchema) (SQLMigration, error) {
	return &updateUserPreference{
		sqlstore:  sqlstore,
		sqlschema: sqlschema,
	}, nil
}

func (migration *updateUserPreference) Register(migrations *migrate.Migrations) error {
	if err := migrations.Register(migration.Up, migration.Down); err != nil {
		return err
	}

	return nil
}

func (migration *updateUserPreference) Up(ctx context.Context, db *bun.DB) error {
	table, _, err := migration.sqlschema.GetTable(ctx, sqlschema.TableName("user_preference"))
	if err != nil {
		return err
	}

	for _, col := range table.Columns {
		if col.Name == "name" || col.Name == "value" {
			return nil
		}
	}

	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}

	defer func() {
		_ = tx.Rollback()
	}()

	oldUserPreferences := []*oldStorableUserPreference{}
	err = tx.
		NewSelect().
		Model(&oldUserPreferences).
		Scan(ctx)
	if err != nil {
		return err
	}

	dropSQLs := migration.sqlschema.Operator().DropTable(table)
	for _, sql := range dropSQLs {
		if _, err := tx.ExecContext(ctx, string(sql)); err != nil {
			return err
		}
	}

	sqls := [][]byte{}
	tableSQLs := migration.sqlschema.Operator().CreateTable(&sqlschema.Table{
		Name: "user_preference",
		Columns: []*sqlschema.Column{
			{Name: "id", DataType: sqlschema.DataTypeText, Nullable: false},
			{Name: "name", DataType: sqlschema.DataTypeText, Nullable: false},
			{Name: "value", DataType: sqlschema.DataTypeText, Nullable: false},
			{Name: "user_id", DataType: sqlschema.DataTypeText, Nullable: false},
		},
		PrimaryKeyConstraint: &sqlschema.PrimaryKeyConstraint{
			ColumnNames: []sqlschema.ColumnName{"id"},
		},
		ForeignKeyConstraints: []*sqlschema.ForeignKeyConstraint{
			{
				ReferencingColumnName: sqlschema.ColumnName("user_id"),
				ReferencedTableName:   sqlschema.TableName("users"),
				ReferencedColumnName:  sqlschema.ColumnName("id"),
			},
		},
	})
	sqls = append(sqls, tableSQLs...)

	indexSQLs := migration.sqlschema.Operator().CreateIndex(&sqlschema.UniqueIndex{TableName: "user_preference", ColumnNames: []sqlschema.ColumnName{"name", "user_id"}})
	sqls = append(sqls, indexSQLs...)

	for _, sql := range sqls {
		if _, err := tx.ExecContext(ctx, string(sql)); err != nil {
			return err
		}
	}

	userPreferences := []*newStorableUserPreference{}
	for _, preference := range oldUserPreferences {
		userPreferences = append(userPreferences, &newStorableUserPreference{
			Identifiable: preference.Identifiable,
			Name:         preference.Name,
			Value:        preference.Value,
			UserID:       preference.UserID,
		})
	}

	if len(userPreferences) > 0 {
		_, err = tx.
			NewInsert().
			Model(&userPreferences).
			Exec(ctx)
		if err != nil {
			return err
		}
	}

	err = tx.Commit()
	if err != nil {
		return err
	}

	return nil
}

func (migration *updateUserPreference) Down(ctx context.Context, db *bun.DB) error {
	return nil
}

type oldStorableOrgPreference struct {
	bun.BaseModel `bun:"table:org_preference"`
	types.Identifiable
	Name  string      `bun:"preference_id,type:text,notnull"`
	Value string      `bun:"preference_value,type:text,notnull"`
	OrgID valuer.UUID `bun:"org_id,type:text,notnull"`
}

type newStorableOrgPreference struct {
	bun.BaseModel `bun:"table:org_preference"`
	types.Identifiable
	Name  string      `bun:"name,type:text,notnull"`
	Value string      `bun:"value,type:text,notnull"`
	OrgID valuer.UUID `bun:"org_id,type:text,notnull"`
}

type updateOrgPreference struct {
	sqlstore  sqlstore.SQLStore
	sqlschema sqlschema.SQLSchema
}

func NewUpdateOrgPreferenceFactory(sqlstore sqlstore.SQLStore, sqlschema sqlschema.SQLSchema) factory.ProviderFactory[SQLMigration, Config] {
	return factory.NewProviderFactory(factory.MustNewName("update_org_preference"), func(ctx context.Context, settings factory.ProviderSettings, config Config) (SQLMigration, error) {
		return newupdateOrgPreference(ctx, settings, config, sqlstore, sqlschema)
	})
}

func newupdateOrgPreference(_ context.Context, _ factory.ProviderSettings, _ Config, sqlstore sqlstore.SQLStore, sqlschema sqlschema.SQLSchema) (SQLMigration, error) {
	return &updateOrgPreference{
		sqlstore:  sqlstore,
		sqlschema: sqlschema,
	}, nil
}

func (migration *updateOrgPreference) Register(migrations *migrate.Migrations) error {
	if err := migrations.Register(migration.Up, migration.Down); err != nil {
		return err
	}

	return nil
}

func (migration *updateOrgPreference) Up(ctx context.Context, db *bun.DB) error {
	table, _, err := migration.sqlschema.GetTable(ctx, sqlschema.TableName("org_preference"))
	if err != nil {
		return err
	}

	for _, col := range table.Columns {
		if col.Name == "name" || col.Name == "value" {
			return nil
		}
	}

	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}

	defer func() {
		_ = tx.Rollback()
	}()

	oldOrgPreferences := []*oldStorableOrgPreference{}
	err = tx.
		NewSelect().
		Model(&oldOrgPreferences).
		Scan(ctx)
	if err != nil {
		return err
	}

	dropSQLs := migration.sqlschema.Operator().DropTable(table)
	for _, sql := range dropSQLs {
		if _, err := tx.ExecContext(ctx, string(sql)); err != nil {
			return err
		}
	}

	sqls := [][]byte{}
	tableSQLs := migration.sqlschema.Operator().CreateTable(&sqlschema.Table{
		Name: "org_preference",
		Columns: []*sqlschema.Column{
			{Name: "id", DataType: sqlschema.DataTypeText, Nullable: false},
			{Name: "name", DataType: sqlschema.DataTypeText, Nullable: false},
			{Name: "value", DataType: sqlschema.DataTypeText, Nullable: false},
			{Name: "org_id", DataType: sqlschema.DataTypeText, Nullable: false},
		},
		PrimaryKeyConstraint: &sqlschema.PrimaryKeyConstraint{
			ColumnNames: []sqlschema.ColumnName{"id"},
		},
		ForeignKeyConstraints: []*sqlschema.ForeignKeyConstraint{
			{
				ReferencingColumnName: sqlschema.ColumnName("org_id"),
				ReferencedTableName:   sqlschema.TableName("organizations"),
				ReferencedColumnName:  sqlschema.ColumnName("id"),
			},
		},
	})
	sqls = append(sqls, tableSQLs...)

	indexSQLs := migration.sqlschema.Operator().CreateIndex(&sqlschema.UniqueIndex{TableName: "org_preference", ColumnNames: []sqlschema.ColumnName{"name", "org_id"}})
	sqls = append(sqls, indexSQLs...)

	for _, sql := range sqls {
		if _, err := tx.ExecContext(ctx, string(sql)); err != nil {
			return err
		}
	}

	orgPreferences := []*newStorableOrgPreference{}
	for _, preference := range oldOrgPreferences {
		orgPreferences = append(orgPreferences, &newStorableOrgPreference{
			Identifiable: preference.Identifiable,
			Name:         preference.Name,
			Value:        preference.Value,
			OrgID:        preference.OrgID,
		})
	}

	if len(orgPreferences) > 0 {
		_, err = tx.
			NewInsert().
			Model(&orgPreferences).
			Exec(ctx)
		if err != nil {
			return err
		}
	}

	err = tx.Commit()
	if err != nil {
		return err
	}

	return nil
}

func (migration *updateOrgPreference) Down(ctx context.Context, db *bun.DB) error {
	return nil
}

type storableOrgDomain struct {
	bun.BaseModel `bun:"table:org_domains"`

	types.Identifiable
	Name  string      `bun:"name,type:text,notnull" json:"name"`
	Data  string      `bun:"data,type:text,notnull" json:"-"`
	OrgID valuer.UUID `bun:"org_id,type:text,notnull" json:"orgId"`
	types.TimeAuditable
}

type storableAuthDomain struct {
	bun.BaseModel `bun:"table:auth_domain"`

	types.Identifiable
	Name  string      `bun:"name,type:text,notnull" json:"name"`
	Data  string      `bun:"data,type:text,notnull" json:"-"`
	OrgID valuer.UUID `bun:"org_id,type:text,notnull" json:"orgId"`
	types.TimeAuditable
}

type renameOrgDomains struct {
	sqlStore  sqlstore.SQLStore
	sqlSchema sqlschema.SQLSchema
}

func NewRenameOrgDomainsFactory(sqlStore sqlstore.SQLStore, sqlSchema sqlschema.SQLSchema) factory.ProviderFactory[SQLMigration, Config] {
	return factory.NewProviderFactory(factory.MustNewName("rename_org_domains"), func(ctx context.Context, ps factory.ProviderSettings, c Config) (SQLMigration, error) {
		return newRenameOrgDomains(ctx, ps, c, sqlStore, sqlSchema)
	})
}

func newRenameOrgDomains(_ context.Context, _ factory.ProviderSettings, _ Config, sqlStore sqlstore.SQLStore, sqlSchema sqlschema.SQLSchema) (SQLMigration, error) {
	return &renameOrgDomains{
		sqlStore:  sqlStore,
		sqlSchema: sqlSchema,
	}, nil
}

func (migration *renameOrgDomains) Register(migrations *migrate.Migrations) error {
	if err := migrations.Register(migration.Up, migration.Down); err != nil {
		return err
	}
	return nil
}

func (migration *renameOrgDomains) Up(ctx context.Context, db *bun.DB) error {
	// check if the `auth_domain` table already exists
	_, _, err := migration.sqlSchema.GetTable(ctx, sqlschema.TableName("auth_domain"))
	if err == nil {
		return nil
	}

	orgDomainTable, _, err := migration.sqlSchema.GetTable(ctx, sqlschema.TableName("org_domains"))
	if err != nil {
		return err
	}

	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}

	defer func() {
		_ = tx.Rollback()
	}()

	oldOrgDomains := []*storableOrgDomain{}
	err = tx.NewSelect().Model(&oldOrgDomains).Scan(ctx)
	if err != nil {
		return err
	}

	// drop table `org_domains`
	orgDomainDropSQLs := migration.sqlSchema.Operator().DropTable(orgDomainTable)
	for _, sql := range orgDomainDropSQLs {
		if _, err := tx.ExecContext(ctx, string(sql)); err != nil {
			return err
		}
	}

	// create table `auth_domain`
	authDomainTableCreateSQLs := migration.sqlSchema.Operator().CreateTable(&sqlschema.Table{
		Name: "auth_domain",
		Columns: []*sqlschema.Column{
			{Name: "id", DataType: sqlschema.DataTypeText, Nullable: false},
			{Name: "name", DataType: sqlschema.DataTypeText, Nullable: false},
			{Name: "data", DataType: sqlschema.DataTypeText, Nullable: false},
			{Name: "org_id", DataType: sqlschema.DataTypeText, Nullable: false},
			{Name: "created_at", DataType: sqlschema.DataTypeTimestamp, Nullable: false},
			{Name: "updated_at", DataType: sqlschema.DataTypeTimestamp, Nullable: false},
		},
		PrimaryKeyConstraint: &sqlschema.PrimaryKeyConstraint{
			ColumnNames: []sqlschema.ColumnName{"id"},
		},
		ForeignKeyConstraints: []*sqlschema.ForeignKeyConstraint{
			{
				ReferencingColumnName: sqlschema.ColumnName("org_id"),
				ReferencedTableName:   sqlschema.TableName("organizations"),
				ReferencedColumnName:  sqlschema.ColumnName("id"),
			},
		},
	})
	for _, sql := range authDomainTableCreateSQLs {
		if _, err := tx.ExecContext(ctx, string(sql)); err != nil {
			return err
		}
	}

	// create index on `auth_domain`
	authDomainIndexSQLs := migration.sqlSchema.Operator().CreateIndex(&sqlschema.UniqueIndex{TableName: "auth_domain", ColumnNames: []sqlschema.ColumnName{"name", "org_id"}})
	for _, sql := range authDomainIndexSQLs {
		if _, err := tx.ExecContext(ctx, string(sql)); err != nil {
			return err
		}
	}

	// convert old org domains to new auth domains
	authDomains := []*storableAuthDomain{}
	for _, orgDomain := range oldOrgDomains {
		authDomains = append(authDomains, &storableAuthDomain{
			Identifiable:  orgDomain.Identifiable,
			TimeAuditable: orgDomain.TimeAuditable,
			Name:          orgDomain.Name,
			Data:          orgDomain.Data,
			OrgID:         orgDomain.OrgID,
		})
	}

	if len(authDomains) > 0 {
		if _, err := tx.NewInsert().Model(&authDomains).Exec(ctx); err != nil {
			return err
		}
	}

	if err := tx.Commit(); err != nil {
		return err
	}

	return nil
}

func (migration *renameOrgDomains) Down(_ context.Context, _ *bun.DB) error {
	return nil
}

type addResetPasswordTokenExpiry struct {
	sqlstore  sqlstore.SQLStore
	sqlschema sqlschema.SQLSchema
}

func NewAddResetPasswordTokenExpiryFactory(sqlstore sqlstore.SQLStore, sqlschema sqlschema.SQLSchema) factory.ProviderFactory[SQLMigration, Config] {
	return factory.NewProviderFactory(factory.MustNewName("add_reset_password_token_expiry"), func(ctx context.Context, providerSettings factory.ProviderSettings, config Config) (SQLMigration, error) {
		return newAddResetPasswordTokenExpiry(ctx, providerSettings, config, sqlstore, sqlschema)
	})
}

func newAddResetPasswordTokenExpiry(_ context.Context, _ factory.ProviderSettings, _ Config, sqlstore sqlstore.SQLStore, sqlschema sqlschema.SQLSchema) (SQLMigration, error) {
	return &addResetPasswordTokenExpiry{
		sqlstore:  sqlstore,
		sqlschema: sqlschema,
	}, nil
}

func (migration *addResetPasswordTokenExpiry) Register(migrations *migrate.Migrations) error {
	if err := migrations.Register(migration.Up, migration.Down); err != nil {
		return err
	}

	return nil
}

func (migration *addResetPasswordTokenExpiry) Up(ctx context.Context, db *bun.DB) error {
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}

	defer func() {
		_ = tx.Rollback()
	}()

	// get the reset_password_token table
	table, uniqueConstraints, err := migration.sqlschema.GetTable(ctx, sqlschema.TableName("reset_password_token"))
	if err != nil {
		return err
	}

	// add a new column `expires_at`
	column := &sqlschema.Column{
		Name:     sqlschema.ColumnName("expires_at"),
		DataType: sqlschema.DataTypeTimestamp,
		Nullable: true,
	}

	// for existing rows set
	defaultValueForExistingRows := time.Now()

	sqls := migration.sqlschema.Operator().AddColumn(table, uniqueConstraints, column, defaultValueForExistingRows)

	for _, sql := range sqls {
		if _, err := tx.ExecContext(ctx, string(sql)); err != nil {
			return err
		}
	}

	if err := tx.Commit(); err != nil {
		return err
	}

	return nil
}

func (migration *addResetPasswordTokenExpiry) Down(ctx context.Context, db *bun.DB) error {
	return nil
}

type addManagedRoles struct {
	sqlstore  sqlstore.SQLStore
	sqlschema sqlschema.SQLSchema
}

func NewAddManagedRolesFactory(sqlstore sqlstore.SQLStore, sqlschema sqlschema.SQLSchema) factory.ProviderFactory[SQLMigration, Config] {
	return factory.NewProviderFactory(factory.MustNewName("add_managed_roles"), func(ctx context.Context, ps factory.ProviderSettings, c Config) (SQLMigration, error) {
		return newAddManagedRoles(ctx, ps, c, sqlstore, sqlschema)
	})
}

func newAddManagedRoles(_ context.Context, _ factory.ProviderSettings, _ Config, sqlStore sqlstore.SQLStore, sqlSchema sqlschema.SQLSchema) (SQLMigration, error) {
	return &addManagedRoles{sqlstore: sqlStore, sqlschema: sqlSchema}, nil
}

func (migration *addManagedRoles) Register(migrations *migrate.Migrations) error {
	if err := migrations.Register(migration.Up, migration.Down); err != nil {
		return err
	}
	return nil
}

func (migration *addManagedRoles) Up(ctx context.Context, db *bun.DB) error {
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}

	defer func() {
		_ = tx.Rollback()
	}()

	var orgIDs []string
	err = tx.NewSelect().
		Table("organizations").
		Column("id").
		Scan(ctx, &orgIDs)
	if err != nil && err != sql.ErrNoRows {
		return err
	}

	managedRoles := []*authtypes.StorableRole{}
	for _, orgIDStr := range orgIDs {
		orgID, err := valuer.NewUUID(orgIDStr)
		if err != nil {
			return err
		}

		// signoz admin
		signozAdminRole := authtypes.NewRole(authtypes.SigNozAdminRoleName, authtypes.SigNozAdminRoleDescription, authtypes.RoleTypeManaged, orgID)
		managedRoles = append(managedRoles, authtypes.NewStorableRoleFromRole(signozAdminRole))

		// signoz editor
		signozEditorRole := authtypes.NewRole(authtypes.SigNozEditorRoleName, authtypes.SigNozEditorRoleDescription, authtypes.RoleTypeManaged, orgID)
		managedRoles = append(managedRoles, authtypes.NewStorableRoleFromRole(signozEditorRole))

		// signoz viewer
		signozViewerRole := authtypes.NewRole(authtypes.SigNozViewerRoleName, authtypes.SigNozViewerRoleDescription, authtypes.RoleTypeManaged, orgID)
		managedRoles = append(managedRoles, authtypes.NewStorableRoleFromRole(signozViewerRole))

		// signoz anonymous
		signozAnonymousRole := authtypes.NewRole(authtypes.SigNozAnonymousRoleName, authtypes.SigNozAnonymousRoleDescription, authtypes.RoleTypeManaged, orgID)
		managedRoles = append(managedRoles, authtypes.NewStorableRoleFromRole(signozAnonymousRole))
	}

	if len(managedRoles) > 0 {
		_, err = tx.NewInsert().
			Model(&managedRoles).
			On("CONFLICT (org_id, name) DO UPDATE").
			Set("description = EXCLUDED.description, type = EXCLUDED.type, updated_at = EXCLUDED.updated_at").
			Exec(ctx)
		if err != nil {
			return err
		}
	}

	if err := tx.Commit(); err != nil {
		return err
	}

	return nil
}

func (migration *addManagedRoles) Down(_ context.Context, _ *bun.DB) error {
	return nil
}

type addAuthzIndex struct {
	sqlstore  sqlstore.SQLStore
	sqlschema sqlschema.SQLSchema
}

func NewAddAuthzIndexFactory(sqlstore sqlstore.SQLStore, sqlschema sqlschema.SQLSchema) factory.ProviderFactory[SQLMigration, Config] {
	return factory.NewProviderFactory(factory.MustNewName("add_authz_index"), func(ctx context.Context, ps factory.ProviderSettings, c Config) (SQLMigration, error) {
		return newAddAuthzIndex(ctx, ps, c, sqlstore, sqlschema)
	})
}

func newAddAuthzIndex(_ context.Context, _ factory.ProviderSettings, _ Config, sqlstore sqlstore.SQLStore, sqlschema sqlschema.SQLSchema) (SQLMigration, error) {
	return &addAuthzIndex{
		sqlstore:  sqlstore,
		sqlschema: sqlschema,
	}, nil
}

func (migration *addAuthzIndex) Register(migrations *migrate.Migrations) error {
	if err := migrations.Register(migration.Up, migration.Down); err != nil {
		return err
	}

	return nil
}

func (migration *addAuthzIndex) Up(ctx context.Context, db *bun.DB) error {
	if migration.sqlstore.BunDB().Dialect().Name() != dialect.PG {
		return nil
	}

	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}

	defer func() {
		_ = tx.Rollback()
	}()

	sqls := [][]byte{}
	indexSQLs := migration.sqlschema.Operator().CreateIndex(&sqlschema.UniqueIndex{TableName: "tuple", ColumnNames: []sqlschema.ColumnName{"ulid"}})
	sqls = append(sqls, indexSQLs...)

	for _, sql := range sqls {
		if _, err := tx.ExecContext(ctx, string(sql)); err != nil {
			return err
		}
	}

	err = tx.Commit()
	if err != nil {
		return err
	}

	return nil
}

func (migration *addAuthzIndex) Down(context.Context, *bun.DB) error {
	return nil
}

type migrateRbacToAuthz struct {
	sqlstore sqlstore.SQLStore
}

var (
	existingRoleToSigNozManagedRoleMap = map[string]string{
		"ADMIN":  "signoz-admin",
		"EDITOR": "signoz-editor",
		"VIEWER": "signoz-viewer",
	}
)

func NewMigrateRbacToAuthzFactory(sqlstore sqlstore.SQLStore) factory.ProviderFactory[SQLMigration, Config] {
	return factory.NewProviderFactory(factory.MustNewName("migrate_rbac_to_authz"), func(ctx context.Context, ps factory.ProviderSettings, c Config) (SQLMigration, error) {
		return newMigrateRbacToAuthz(ctx, ps, c, sqlstore)
	})
}

func newMigrateRbacToAuthz(_ context.Context, _ factory.ProviderSettings, _ Config, sqlstore sqlstore.SQLStore) (SQLMigration, error) {
	return &migrateRbacToAuthz{
		sqlstore: sqlstore,
	}, nil
}

func (migration *migrateRbacToAuthz) Register(migrations *migrate.Migrations) error {
	if err := migrations.Register(migration.Up, migration.Down); err != nil {
		return err
	}

	return nil
}

func (migration *migrateRbacToAuthz) Up(ctx context.Context, db *bun.DB) error {
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}

	defer func() {
		_ = tx.Rollback()
	}()

	// for upgrades from version where authz service wasn't introduced the store won't be present, hence we need to ensure store exists.
	var storeID string
	err = tx.QueryRowContext(ctx, `SELECT id FROM store WHERE name = ? LIMIT 1`, "signoz").Scan(&storeID)
	if err != nil && err != sql.ErrNoRows {
		return err
	}
	if storeID == "" {
		// based on openfga ids to avoid any scan issues.
		// ref: https://github.com/openfga/openfga/blob/main/pkg/server/commands/create_store.go#L45
		storeID = ulid.Make().String()
		_, err := tx.ExecContext(ctx, `INSERT INTO store (id, name, created_at, updated_at) VALUES (?, ?, ?, ?)`, storeID, "signoz", time.Now().UTC(), time.Now().UTC())
		if err != nil {
			return err
		}
	}

	// fetch all the orgs for which we need to insert user role grant tuples.
	orgIDs := []string{}
	rows, err := tx.QueryContext(ctx, `SELECT id FROM organizations`)
	if err != nil {
		return err
	}
	defer rows.Close()

	for rows.Next() {
		var orgID string
		if err := rows.Scan(&orgID); err != nil {
			return err
		}
		orgIDs = append(orgIDs, orgID)
	}

	type tuple struct {
		OrgID    string
		Type     string
		ID       string
		RoleName string
	}
	tuples := []tuple{}

	for _, orgID := range orgIDs {
		userRows, err := tx.QueryContext(ctx, `
			SELECT id, role FROM users WHERE org_id = ?`, orgID)
		if err != nil {
			return err
		}
		defer userRows.Close()

		for userRows.Next() {
			var id, role string
			if err := userRows.Scan(&id, &role); err != nil {
				return err
			}

			managedRole, ok := existingRoleToSigNozManagedRoleMap[role]
			if !ok {
				return errors.Newf(errors.TypeInternal, errors.CodeInternal, "invalid role assignment: %s for user_id: %s", role, id)
			}
			tuples = append(tuples, tuple{
				OrgID:    orgID,
				ID:       id,
				Type:     "user",
				RoleName: managedRole,
			})
		}

		tuples = append(tuples, tuple{
			OrgID:    orgID,
			ID:       authtypes.AnonymousUser.StringValue(),
			Type:     "anonymous",
			RoleName: "signoz-anonymous",
		})

	}

	_, err = tx.ExecContext(ctx, `DELETE FROM tuple`)
	if err != nil {
		return err
	}

	_, err = tx.ExecContext(ctx, `DELETE FROM changelog`)
	if err != nil {
		return err
	}

	for _, tuple := range tuples {
		// based on openfga tuple and changelog id's are same for writes.
		// ref: https://github.com/openfga/openfga/blob/main/pkg/storage/sqlite/sqlite.go#L467
		entropy := ulid.DefaultEntropy()
		now := time.Now().UTC()
		tupleID := ulid.MustNew(ulid.Timestamp(now), entropy).String()

		if migration.sqlstore.BunDB().Dialect().Name() == dialect.PG {
			result, err := tx.ExecContext(ctx, `
			INSERT INTO tuple (store, object_type, object_id, relation, _user, user_type, ulid, inserted_at)
			VALUES (?, ?, ?, ?, ?, ?, ?, ?)
			ON CONFLICT (store, object_type, object_id, relation, _user) DO NOTHING`,
				storeID, "role", "organization/"+tuple.OrgID+"/role/"+tuple.RoleName, "assignee", tuple.Type+":organization/"+tuple.OrgID+"/"+tuple.Type+"/"+tuple.ID, "user", tupleID, now,
			)
			if err != nil {
				return err
			}

			rowsAffected, err := result.RowsAffected()
			if err != nil {
				return err
			}

			if rowsAffected == 0 {
				continue
			}

			_, err = tx.ExecContext(ctx, `
			INSERT INTO changelog (store, object_type, object_id, relation, _user, operation, ulid, inserted_at)
			VALUES (?, ?, ?, ?, ?, ?, ?, ?)
			ON CONFLICT (store, ulid, object_type) DO NOTHING`,
				storeID, "role", "organization/"+tuple.OrgID+"/role/"+tuple.RoleName, "assignee", tuple.Type+":organization/"+tuple.OrgID+"/"+tuple.Type+"/"+tuple.ID, "TUPLE_OPERATION_WRITE", tupleID, now,
			)
			if err != nil {
				return err
			}

		} else {
			result, err := tx.ExecContext(ctx, `
			INSERT INTO tuple (store, object_type, object_id, relation, user_object_type, user_object_id, user_relation, user_type, ulid, inserted_at)
			VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
			ON CONFLICT (store, object_type, object_id, relation, user_object_type, user_object_id, user_relation) DO NOTHING`,
				storeID, "role", "organization/"+tuple.OrgID+"/role/"+tuple.RoleName, "assignee", tuple.Type, "organization/"+tuple.OrgID+"/"+tuple.Type+"/"+tuple.ID, "", "user", tupleID, now,
			)
			if err != nil {
				return err
			}

			rowsAffected, err := result.RowsAffected()
			if err != nil {
				return err
			}

			if rowsAffected == 0 {
				continue
			}

			_, err = tx.ExecContext(ctx, `
			INSERT INTO changelog (store, object_type, object_id, relation, user_object_type, user_object_id, user_relation, operation, ulid, inserted_at)
			VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
			ON CONFLICT (store, ulid, object_type) DO NOTHING`,
				storeID, "role", "organization/"+tuple.OrgID+"/role/"+tuple.RoleName, "assignee", tuple.Type, "organization/"+tuple.OrgID+"/"+tuple.Type+"/"+tuple.ID, "", 0, tupleID, now,
			)
			if err != nil {
				return err
			}
		}

	}

	err = tx.Commit()
	if err != nil {
		return err
	}

	return nil
}

func (migration *migrateRbacToAuthz) Down(context.Context, *bun.DB) error {
	return nil
}

type migratePublicDashboards struct {
	sqlstore sqlstore.SQLStore
}

func NewMigratePublicDashboardsFactory(sqlstore sqlstore.SQLStore) factory.ProviderFactory[SQLMigration, Config] {
	return factory.NewProviderFactory(factory.MustNewName("migrate_public_dashboards"), func(ctx context.Context, ps factory.ProviderSettings, c Config) (SQLMigration, error) {
		return newMigratePublicDashboards(ctx, ps, c, sqlstore)
	})
}

func newMigratePublicDashboards(_ context.Context, _ factory.ProviderSettings, _ Config, sqlstore sqlstore.SQLStore) (SQLMigration, error) {
	return &migratePublicDashboards{
		sqlstore: sqlstore,
	}, nil
}

func (migration *migratePublicDashboards) Register(migrations *migrate.Migrations) error {
	if err := migrations.Register(migration.Up, migration.Down); err != nil {
		return err
	}

	return nil
}

func (migration *migratePublicDashboards) Up(ctx context.Context, db *bun.DB) error {
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}

	defer func() {
		_ = tx.Rollback()
	}()

	var storeID string
	err = tx.QueryRowContext(ctx, `SELECT id FROM store WHERE name = ? LIMIT 1`, "signoz").Scan(&storeID)
	if err != nil {
		return err
	}

	// fetch all the orgs for which we need to insert user role grant tuples.
	orgIDs := []string{}
	rows, err := tx.QueryContext(ctx, `SELECT id FROM organizations`)
	if err != nil {
		return err
	}
	defer rows.Close()

	for rows.Next() {
		var orgID string
		if err := rows.Scan(&orgID); err != nil {
			return err
		}
		orgIDs = append(orgIDs, orgID)
	}

	type tuple struct {
		OrgID       string
		DashboardID string
		RoleName    string
	}
	tuples := []tuple{}

	for _, orgID := range orgIDs {
		publicDashboards, err := tx.QueryContext(ctx, `
			SELECT public_dashboard.id
			FROM public_dashboard
			INNER JOIN dashboard ON dashboard.id = public_dashboard.dashboard_id
			WHERE dashboard.org_id = ?`, orgID)
		if err != nil {
			return err
		}
		defer publicDashboards.Close()

		for publicDashboards.Next() {
			var id string
			if err := publicDashboards.Scan(&id); err != nil {
				return err
			}

			tuples = append(tuples, tuple{
				OrgID:       orgID,
				DashboardID: id,
				RoleName:    "signoz-anonymous",
			})
		}
	}

	for _, tuple := range tuples {
		// based on openfga tuple and changelog id's are same for writes.
		// ref: https://github.com/openfga/openfga/blob/main/pkg/storage/sqlite/sqlite.go#L467
		entropy := ulid.DefaultEntropy()
		now := time.Now().UTC()
		tupleID := ulid.MustNew(ulid.Timestamp(now), entropy).String()

		if migration.sqlstore.BunDB().Dialect().Name() == dialect.PG {
			result, err := tx.ExecContext(ctx, `
			INSERT INTO tuple (store, object_type, object_id, relation, _user, user_type, ulid, inserted_at)
			VALUES (?, ?, ?, ?, ?, ?, ?, ?)
			ON CONFLICT (store, object_type, object_id, relation, _user) DO NOTHING`,
				storeID, "metaresource", "organization/"+tuple.OrgID+"/public-dashboard/"+tuple.DashboardID, "read", "role:organization/"+tuple.OrgID+"/role/"+tuple.RoleName+"#assignee", "userset", tupleID, now,
			)
			if err != nil {
				return err
			}

			rowsAffected, err := result.RowsAffected()
			if err != nil {
				return err
			}

			if rowsAffected == 0 {
				continue
			}

			_, err = tx.ExecContext(ctx, `
			INSERT INTO changelog (store, object_type, object_id, relation, _user, operation, ulid, inserted_at)
			VALUES (?, ?, ?, ?, ?, ?, ?, ?)
			ON CONFLICT (store, ulid, object_type) DO NOTHING`,
				storeID, "metaresource", "organization/"+tuple.OrgID+"/public-dashboard/"+tuple.DashboardID, "read", "role:organization/"+tuple.OrgID+"/role/"+tuple.RoleName+"#assignee", "TUPLE_OPERATION_WRITE", tupleID, now,
			)
			if err != nil {
				return err
			}

		} else {
			result, err := tx.ExecContext(ctx, `
			INSERT INTO tuple (store, object_type, object_id, relation, user_object_type, user_object_id, user_relation, user_type, ulid, inserted_at)
			VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
			ON CONFLICT (store, object_type, object_id, relation, user_object_type, user_object_id, user_relation) DO NOTHING`,
				storeID, "metaresource", "organization/"+tuple.OrgID+"/public-dashboard/"+tuple.DashboardID, "read", "role", "organization/"+tuple.OrgID+"/role/"+tuple.RoleName, "assignee", "userset", tupleID, now,
			)
			if err != nil {
				return err
			}

			rowsAffected, err := result.RowsAffected()
			if err != nil {
				return err
			}

			if rowsAffected == 0 {
				continue
			}

			_, err = tx.ExecContext(ctx, `
			INSERT INTO changelog (store, object_type, object_id, relation, user_object_type, user_object_id, user_relation, operation, ulid, inserted_at)
			VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
			ON CONFLICT (store, ulid, object_type) DO NOTHING`,
				storeID, "metaresource", "organization/"+tuple.OrgID+"/public-dashboard/"+tuple.DashboardID, "read", "role", "organization/"+tuple.OrgID+"/role/"+tuple.RoleName, "assignee", 0, tupleID, now,
			)
			if err != nil {
				return err
			}
		}

	}

	err = tx.Commit()
	if err != nil {
		return err
	}

	return nil
}

func (migration *migratePublicDashboards) Down(context.Context, *bun.DB) error {
	return nil
}

type addAnonymousPublicDashboardTransaction struct {
	sqlstore sqlstore.SQLStore
}

func NewAddAnonymousPublicDashboardTransactionFactory(sqlstore sqlstore.SQLStore) factory.ProviderFactory[SQLMigration, Config] {
	return factory.NewProviderFactory(factory.MustNewName("add_public_dashboard_txn"), func(ctx context.Context, ps factory.ProviderSettings, c Config) (SQLMigration, error) {
		return newAddAnonymousPublicDashboardTransaction(ctx, ps, c, sqlstore)
	})
}

func newAddAnonymousPublicDashboardTransaction(_ context.Context, _ factory.ProviderSettings, _ Config, sqlstore sqlstore.SQLStore) (SQLMigration, error) {
	return &addAnonymousPublicDashboardTransaction{
		sqlstore: sqlstore,
	}, nil
}

func (migration *addAnonymousPublicDashboardTransaction) Register(migrations *migrate.Migrations) error {
	if err := migrations.Register(migration.Up, migration.Down); err != nil {
		return err
	}

	return nil
}

func (migration *addAnonymousPublicDashboardTransaction) Up(ctx context.Context, db *bun.DB) error {
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}

	defer func() {
		_ = tx.Rollback()
	}()

	var storeID string
	err = tx.QueryRowContext(ctx, `SELECT id FROM store WHERE name = ? LIMIT 1`, "signoz").Scan(&storeID)
	if err != nil {
		return err
	}

	// fetch all the orgs for which we need to insert the anonymous public dashboard transaction tuple.
	orgIDs := []string{}
	rows, err := tx.QueryContext(ctx, `SELECT id FROM organizations`)
	if err != nil {
		return err
	}
	defer rows.Close()

	for rows.Next() {
		var orgID string
		if err := rows.Scan(&orgID); err != nil {
			return err
		}
		orgIDs = append(orgIDs, orgID)
	}

	for _, orgID := range orgIDs {
		// based on openfga tuple and changelog id's are same for writes.
		// ref: https://github.com/openfga/openfga/blob/main/pkg/storage/sqlite/sqlite.go#L467
		entropy := ulid.DefaultEntropy()
		now := time.Now().UTC()
		tupleID := ulid.MustNew(ulid.Timestamp(now), entropy).String()

		// Add wildcard (*) transaction for signoz-anonymous role to read all public-dashboards
		// This grants the signoz-anonymous role read access to all public dashboards in the organization
		if migration.sqlstore.BunDB().Dialect().Name() == dialect.PG {
			result, err := tx.ExecContext(ctx, `
			INSERT INTO tuple (store, object_type, object_id, relation, _user, user_type, ulid, inserted_at)
			VALUES (?, ?, ?, ?, ?, ?, ?, ?)
			ON CONFLICT (store, object_type, object_id, relation, _user) DO NOTHING`,
				storeID, "metaresource", "organization/"+orgID+"/public-dashboard/*", "read", "role:organization/"+orgID+"/role/"+authtypes.SigNozAnonymousRoleName+"#assignee", "userset", tupleID, now,
			)
			if err != nil {
				return err
			}

			rowsAffected, err := result.RowsAffected()
			if err != nil {
				return err
			}

			if rowsAffected == 0 {
				continue
			}

			_, err = tx.ExecContext(ctx, `
			INSERT INTO changelog (store, object_type, object_id, relation, _user, operation, ulid, inserted_at)
			VALUES (?, ?, ?, ?, ?, ?, ?, ?)
			ON CONFLICT (store, ulid, object_type) DO NOTHING`,
				storeID, "metaresource", "organization/"+orgID+"/public-dashboard/*", "read", "role:organization/"+orgID+"/role/"+authtypes.SigNozAnonymousRoleName+"#assignee", "TUPLE_OPERATION_WRITE", tupleID, now,
			)
			if err != nil {
				return err
			}

		} else {
			result, err := tx.ExecContext(ctx, `
			INSERT INTO tuple (store, object_type, object_id, relation, user_object_type, user_object_id, user_relation, user_type, ulid, inserted_at)
			VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
			ON CONFLICT (store, object_type, object_id, relation, user_object_type, user_object_id, user_relation) DO NOTHING`,
				storeID, "metaresource", "organization/"+orgID+"/public-dashboard/*", "read", "role", "organization/"+orgID+"/role/"+authtypes.SigNozAnonymousRoleName, "assignee", "userset", tupleID, now,
			)
			if err != nil {
				return err
			}

			rowsAffected, err := result.RowsAffected()
			if err != nil {
				return err
			}

			if rowsAffected == 0 {
				continue
			}

			_, err = tx.ExecContext(ctx, `
			INSERT INTO changelog (store, object_type, object_id, relation, user_object_type, user_object_id, user_relation, operation, ulid, inserted_at)
			VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
			ON CONFLICT (store, ulid, object_type) DO NOTHING`,
				storeID, "metaresource", "organization/"+orgID+"/public-dashboard/*", "read", "role", "organization/"+orgID+"/role/"+authtypes.SigNozAnonymousRoleName, "assignee", 0, tupleID, now,
			)
			if err != nil {
				return err
			}
		}

	}

	err = tx.Commit()
	if err != nil {
		return err
	}

	return nil
}

func (migration *addAnonymousPublicDashboardTransaction) Down(context.Context, *bun.DB) error {
	return nil
}

type addRootUser struct {
	sqlstore  sqlstore.SQLStore
	sqlschema sqlschema.SQLSchema
}

func NewAddRootUserFactory(sqlstore sqlstore.SQLStore, sqlschema sqlschema.SQLSchema) factory.ProviderFactory[SQLMigration, Config] {
	return factory.NewProviderFactory(factory.MustNewName("add_root_user"), func(ctx context.Context, providerSettings factory.ProviderSettings, config Config) (SQLMigration, error) {
		return &addRootUser{
			sqlstore:  sqlstore,
			sqlschema: sqlschema,
		}, nil
	})
}

func (migration *addRootUser) Register(migrations *migrate.Migrations) error {
	if err := migrations.Register(migration.Up, migration.Down); err != nil {
		return err
	}

	return nil
}

func (migration *addRootUser) Up(ctx context.Context, db *bun.DB) error {
	if err := migration.sqlschema.ToggleFKEnforcement(ctx, db, false); err != nil {
		return err
	}

	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}

	defer func() {
		_ = tx.Rollback()
	}()

	table, uniqueConstraints, err := migration.sqlschema.GetTable(ctx, sqlschema.TableName("users"))
	if err != nil {
		return err
	}

	column := &sqlschema.Column{
		Name:     sqlschema.ColumnName("is_root"),
		DataType: sqlschema.DataTypeBoolean,
		Nullable: false,
	}

	sqls := migration.sqlschema.Operator().AddColumn(table, uniqueConstraints, column, false)
	for _, sql := range sqls {
		if _, err := tx.ExecContext(ctx, string(sql)); err != nil {
			return err
		}
	}

	if err := tx.Commit(); err != nil {
		return err
	}

	if err := migration.sqlschema.ToggleFKEnforcement(ctx, db, true); err != nil {
		return err
	}

	return nil
}

func (migration *addRootUser) Down(ctx context.Context, db *bun.DB) error {
	return nil
}

type addUserEmailOrgIDIndex struct {
	sqlstore  sqlstore.SQLStore
	sqlschema sqlschema.SQLSchema
}

func NewAddUserEmailOrgIDIndexFactory(sqlstore sqlstore.SQLStore, sqlschema sqlschema.SQLSchema) factory.ProviderFactory[SQLMigration, Config] {
	return factory.NewProviderFactory(factory.MustNewName("add_user_email_org_id_index"), func(ctx context.Context, providerSettings factory.ProviderSettings, config Config) (SQLMigration, error) {
		return &addUserEmailOrgIDIndex{
			sqlstore:  sqlstore,
			sqlschema: sqlschema,
		}, nil
	})
}

func (migration *addUserEmailOrgIDIndex) Register(migrations *migrate.Migrations) error {
	if err := migrations.Register(migration.Up, migration.Down); err != nil {
		return err
	}

	return nil
}

func (migration *addUserEmailOrgIDIndex) Up(ctx context.Context, db *bun.DB) error {
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}

	defer func() {
		_ = tx.Rollback()
	}()

	sqls := migration.sqlschema.Operator().CreateIndex(&sqlschema.UniqueIndex{TableName: "users", ColumnNames: []sqlschema.ColumnName{"email", "org_id"}})
	for _, sql := range sqls {
		if _, err := tx.ExecContext(ctx, string(sql)); err != nil {
			return err
		}
	}

	if err := tx.Commit(); err != nil {
		return err
	}

	return nil
}

func (migration *addUserEmailOrgIDIndex) Down(ctx context.Context, db *bun.DB) error {
	return nil
}

type migrateRulesV4ToV5 struct {
	store          sqlstore.SQLStore
	telemetryStore telemetrystore.TelemetryStore
	logger         *slog.Logger
}

func NewMigrateRulesV4ToV5Factory(
	store sqlstore.SQLStore,
	telemetryStore telemetrystore.TelemetryStore,
) factory.ProviderFactory[SQLMigration, Config] {
	return factory.NewProviderFactory(
		factory.MustNewName("migrate_rules_post_deprecation"),
		func(ctx context.Context, ps factory.ProviderSettings, c Config) (SQLMigration, error) {
			return &migrateRulesV4ToV5{
				store:          store,
				telemetryStore: telemetryStore,
				logger:         ps.Logger,
			}, nil
		})
}

func (migration *migrateRulesV4ToV5) Register(migrations *migrate.Migrations) error {
	if err := migrations.Register(migration.Up, migration.Down); err != nil {
		return err
	}
	return nil
}

func (migration *migrateRulesV4ToV5) getLogDuplicateKeys(ctx context.Context) ([]string, error) {
	query := `
		SELECT name
		FROM (
			SELECT DISTINCT name FROM signoz_logs.logs_attribute_keys
			INTERSECT
			SELECT DISTINCT name FROM signoz_logs.logs_resource_keys
		)
		ORDER BY name
	`

	rows, err := migration.telemetryStore.ClickhouseDB().Query(ctx, query)
	if err != nil {
		migration.logger.WarnContext(ctx, "failed to query log duplicate keys", errors.Attr(err))
		return nil, nil
	}
	defer rows.Close()

	var keys []string
	for rows.Next() {
		var key string
		if err := rows.Scan(&key); err != nil {
			migration.logger.WarnContext(ctx, "failed to scan log duplicate key", errors.Attr(err))
			continue
		}
		keys = append(keys, key)
	}

	return keys, nil
}

func (migration *migrateRulesV4ToV5) getTraceDuplicateKeys(ctx context.Context) ([]string, error) {
	query := `
		SELECT tagKey
		FROM signoz_traces.span_attributes_keys
		WHERE tagType IN ('tag', 'resource')
		GROUP BY tagKey
		HAVING COUNT(DISTINCT tagType) > 1
		ORDER BY tagKey
	`

	rows, err := migration.telemetryStore.ClickhouseDB().Query(ctx, query)
	if err != nil {
		migration.logger.WarnContext(ctx, "failed to query trace duplicate keys", errors.Attr(err))
		return nil, nil
	}
	defer rows.Close()

	var keys []string
	for rows.Next() {
		var key string
		if err := rows.Scan(&key); err != nil {
			migration.logger.WarnContext(ctx, "failed to scan trace duplicate key", errors.Attr(err))
			continue
		}
		keys = append(keys, key)
	}

	return keys, nil
}

func (migration *migrateRulesV4ToV5) Up(ctx context.Context, db *bun.DB) error {
	logsKeys, err := migration.getLogDuplicateKeys(ctx)
	if err != nil {
		return err
	}

	tracesKeys, err := migration.getTraceDuplicateKeys(ctx)
	if err != nil {
		return err
	}

	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}

	defer func() {
		_ = tx.Rollback()
	}()

	var rules []struct {
		ID   string         `bun:"id"`
		Data map[string]any `bun:"data"`
	}

	err = tx.NewSelect().
		Table("rule").
		Column("id", "data").
		Scan(ctx, &rules)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil
		}
		return err
	}

	alertsMigrator := transition.NewAlertMigrateV5(migration.logger, logsKeys, tracesKeys)

	count := 0

	for _, rule := range rules {
		version, _ := rule.Data["version"].(string)

		selectedQueryUpdated := transition.EnsureAlertSelectedQuery(rule.Data)
		if version == "v5" && !selectedQueryUpdated {
			continue
		}

		if version == "" {
			migration.logger.WarnContext(ctx, "unexpected empty version for rule", slog.String("rule_id", rule.ID))
		}

		migration.logger.InfoContext(ctx, "migrating rule v4 to v5", slog.String("rule_id", rule.ID), slog.String("current_version", version))

		// Check if the queries envelope already exists and is non-empty
		hasQueriesEnvelope := false
		if condition, ok := rule.Data["condition"].(map[string]any); ok {
			if compositeQuery, ok := condition["compositeQuery"].(map[string]any); ok {
				if queries, ok := compositeQuery["queries"].([]any); ok && len(queries) > 0 {
					hasQueriesEnvelope = true
				}
			}
		}

		if hasQueriesEnvelope {
			// already has queries envelope, just bump version
			// this is because user made a mistake of choosing version
			migration.logger.InfoContext(ctx, "rule already has queries envelope, bumping version", slog.String("rule_id", rule.ID))
			rule.Data["version"] = "v5"
		} else {
			// old format, run full migration
			migration.logger.InfoContext(ctx, "rule has old format, running full migration", slog.String("rule_id", rule.ID))
			updated := alertsMigrator.Migrate(ctx, rule.Data)
			if !updated {
				migration.logger.WarnContext(ctx, "expected updated to be true but got false", slog.String("rule_id", rule.ID))
				continue
			}
			rule.Data["version"] = "v5"
		}

		dataJSON, err := json.Marshal(rule.Data)
		if err != nil {
			return err
		}

		_, err = tx.NewUpdate().
			Table("rule").
			Set("data = ?", string(dataJSON)).
			Where("id = ?", rule.ID).
			Exec(ctx)
		if err != nil {
			return err
		}
		count++
	}
	if count != 0 {
		migration.logger.InfoContext(ctx, "migrate v4 alerts", slog.Int("count", count))
	}

	return tx.Commit()
}

func (migration *migrateRulesV4ToV5) Down(ctx context.Context, db *bun.DB) error {
	return nil
}

type addStatusUser struct {
	sqlstore  sqlstore.SQLStore
	sqlschema sqlschema.SQLSchema
}

func NewAddStatusUserFactory(sqlstore sqlstore.SQLStore, sqlschema sqlschema.SQLSchema) factory.ProviderFactory[SQLMigration, Config] {
	return factory.NewProviderFactory(
		factory.MustNewName("add_status_user"),
		func(ctx context.Context, ps factory.ProviderSettings, c Config) (SQLMigration, error) {
			return &addStatusUser{
				sqlstore:  sqlstore,
				sqlschema: sqlschema,
			}, nil
		},
	)
}

func (migration *addStatusUser) Register(migrations *migrate.Migrations) error {
	if err := migrations.Register(migration.Up, migration.Down); err != nil {
		return err
	}

	return nil
}

func (migration *addStatusUser) Up(ctx context.Context, db *bun.DB) error {
	if err := migration.sqlschema.ToggleFKEnforcement(ctx, db, false); err != nil {
		return err
	}

	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}

	defer func() {
		_ = tx.Rollback()
	}()

	table, uniqueConstraints, err := migration.sqlschema.GetTable(ctx, sqlschema.TableName("users"))
	if err != nil {
		return err
	}

	statusColumn := &sqlschema.Column{
		Name:     sqlschema.ColumnName("status"),
		DataType: sqlschema.DataTypeText,
		Nullable: false,
	}

	sqls := migration.sqlschema.Operator().AddColumn(table, uniqueConstraints, statusColumn, "active")

	// add deleted_at (zero time = not deleted, non-zero = deletion timestamp) to enable the
	// composite unique index that replaces the partial index approach
	deletedAtColumn := &sqlschema.Column{
		Name:     sqlschema.ColumnName("deleted_at"),
		DataType: sqlschema.DataTypeTimestamp,
		Nullable: false,
	}

	sqls = append(sqls, migration.sqlschema.Operator().AddColumn(table, uniqueConstraints, deletedAtColumn, time.Time{})...)

	// we need to drop the unique index on (email, org_id)
	sqls = append(sqls, migration.sqlschema.Operator().DropIndex(&sqlschema.UniqueIndex{TableName: "users", ColumnNames: []sqlschema.ColumnName{"email", "org_id"}})...)

	for _, sql := range sqls {
		if _, err := tx.ExecContext(ctx, string(sql)); err != nil {
			return err
		}
	}

	// add a composite unique index on (org_id, email, deleted_at).
	// active and pending users have deleted_at=time.Time{} (zero), forming a unique (org_id, email, zero) tuple.
	// soft-deleted users have deleted_at set to the deletion timestamp, making each deleted row unique
	// and allowing the same email to be re-invited after deletion without a constraint violation.
	newIndexSqls := migration.sqlschema.Operator().CreateIndex(&sqlschema.UniqueIndex{TableName: "users", ColumnNames: []sqlschema.ColumnName{"org_id", "email", "deleted_at"}})
	for _, sql := range newIndexSqls {
		if _, err := tx.ExecContext(ctx, string(sql)); err != nil {
			return err
		}
	}

	if err := tx.Commit(); err != nil {
		return err
	}

	if err := migration.sqlschema.ToggleFKEnforcement(ctx, db, true); err != nil {
		return err
	}

	return nil
}

func (migration *addStatusUser) Down(ctx context.Context, db *bun.DB) error {
	return nil
}

type deprecateUserInvite struct {
	sqlstore  sqlstore.SQLStore
	sqlschema sqlschema.SQLSchema
}

type userInviteRow struct {
	bun.BaseModel `bun:"table:user_invite"`

	ID        string    `bun:"id"`
	Name      string    `bun:"name"`
	Email     string    `bun:"email"`
	Role      string    `bun:"role"`
	OrgID     string    `bun:"org_id"`
	Token     string    `bun:"token"`
	CreatedAt time.Time `bun:"created_at"`
	UpdatedAt time.Time `bun:"updated_at"`
}

type pendingInviteUser struct {
	bun.BaseModel `bun:"table:users"`

	ID          string    `bun:"id"`
	DisplayName string    `bun:"display_name"`
	Email       string    `bun:"email"`
	Role        string    `bun:"role"`
	OrgID       string    `bun:"org_id"`
	IsRoot      bool      `bun:"is_root"`
	Status      string    `bun:"status"`
	CreatedAt   time.Time `bun:"created_at"`
	UpdatedAt   time.Time `bun:"updated_at"`
	DeletedAt   time.Time `bun:"deleted_at"`
}

func NewDeprecateUserInviteFactory(sqlstore sqlstore.SQLStore, sqlschema sqlschema.SQLSchema) factory.ProviderFactory[SQLMigration, Config] {
	return factory.NewProviderFactory(
		factory.MustNewName("deprecate_user_invite"),
		func(ctx context.Context, ps factory.ProviderSettings, c Config) (SQLMigration, error) {
			return &deprecateUserInvite{
				sqlstore:  sqlstore,
				sqlschema: sqlschema,
			}, nil
		},
	)
}

func (migration *deprecateUserInvite) Register(migrations *migrate.Migrations) error {
	if err := migrations.Register(migration.Up, migration.Down); err != nil {
		return err
	}

	return nil
}

func (migration *deprecateUserInvite) Up(ctx context.Context, db *bun.DB) error {
	table, _, err := migration.sqlschema.GetTable(ctx, sqlschema.TableName("user_invite"))
	if err != nil {
		if err == sql.ErrNoRows || errors.Ast(err, errors.TypeNotFound) {
			return nil
		}
		return err
	}

	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}

	defer func() {
		_ = tx.Rollback()
	}()

	// existing invites
	var invites []*userInviteRow
	err = tx.NewSelect().Model(&invites).Scan(ctx)
	if err != nil && err != sql.ErrNoRows {
		return err
	}

	// move all invitations to the users table as a pending_invite user
	// skipping any invite whose email+org already has a user entry with non-deleted status
	users := make([]*pendingInviteUser, 0, len(invites))

	for _, invite := range invites {
		existingCount, err := tx.NewSelect().
			TableExpr("users").
			Where("email = ?", invite.Email).
			Where("org_id = ?", invite.OrgID).
			Where("status != ?", "deleted").
			Count(ctx)
		if err != nil {
			return err
		}

		if existingCount > 0 {
			continue
		}

		user := &pendingInviteUser{
			ID:          invite.ID,
			DisplayName: invite.Name,
			Email:       invite.Email,
			Role:        invite.Role,
			OrgID:       invite.OrgID,
			IsRoot:      false,
			Status:      "pending_invite",
			CreatedAt:   invite.CreatedAt,
			UpdatedAt:   time.Now(),
			DeletedAt:   time.Time{},
		}

		users = append(users, user)
	}

	if len(users) > 0 {
		_, err = tx.NewInsert().Model(&users).Exec(ctx)
		if err != nil {
			return err
		}
	}

	// finally drop the user_invite table
	dropTableSqls := migration.sqlschema.Operator().DropTable(table)

	for _, sql := range dropTableSqls {
		if _, err := tx.ExecContext(ctx, string(sql)); err != nil {
			return err
		}
	}

	if err := tx.Commit(); err != nil {
		return err
	}

	return nil
}

func (migration *deprecateUserInvite) Down(ctx context.Context, db *bun.DB) error {
	return nil
}

type updateCloudIntegrationUniqueIndex struct {
	sqlstore  sqlstore.SQLStore
	sqlschema sqlschema.SQLSchema
}

func NewUpdateCloudIntegrationUniqueIndexFactory(sqlstore sqlstore.SQLStore, sqlschema sqlschema.SQLSchema) factory.ProviderFactory[SQLMigration, Config] {
	return factory.NewProviderFactory(
		factory.MustNewName("update_cloud_integration_index"),
		func(ctx context.Context, ps factory.ProviderSettings, c Config) (SQLMigration, error) {
			return &updateCloudIntegrationUniqueIndex{
				sqlstore:  sqlstore,
				sqlschema: sqlschema,
			}, nil
		},
	)
}

func (migration *updateCloudIntegrationUniqueIndex) Register(migrations *migrate.Migrations) error {
	if err := migrations.Register(migration.Up, migration.Down); err != nil {
		return err
	}

	return nil
}

type cloudIntegrationRow struct {
	bun.BaseModel `bun:"table:cloud_integration"`

	ID        string    `bun:"id"`
	AccountID string    `bun:"account_id"`
	Provider  string    `bun:"provider"`
	OrgID     string    `bun:"org_id"`
	Config    string    `bun:"config"`
	UpdatedAt time.Time `bun:"updated_at"`
}

type cloudIntegrationAccountConfig struct {
	Regions []string `json:"regions"`
}

// duplicateGroup holds the keeper (first element) and losers (rest) for a duplicate (account_id, provider, org_id) group.
type duplicateGroup struct {
	keeper *cloudIntegrationRow
	losers []*cloudIntegrationRow
}

func (migration *updateCloudIntegrationUniqueIndex) Up(ctx context.Context, db *bun.DB) error {
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer func() {
		_ = tx.Rollback()
	}()

	sqls := [][]byte{}

	// Step 1: Drop the wrong index on (id, provider, org_id)
	dropSqls := migration.sqlschema.Operator().DropIndex(
		(&sqlschema.UniqueIndex{
			TableName:   "cloud_integration",
			ColumnNames: []sqlschema.ColumnName{"id", "provider", "org_id"},
		}).Named("unique_cloud_integration"),
	)
	sqls = append(sqls, dropSqls...)

	// Step 2: Normalize empty-string account_id to NULL
	// Older table structure could store "" instead of NULL for unconnected accounts.
	// Empty strings would violate the partial unique index since '' = '' (unlike NULL != NULL).
	_, err = tx.NewUpdate().
		TableExpr("cloud_integration").
		Set("account_id = NULL").
		Where("account_id = ''").
		Exec(ctx)
	if err != nil {
		return err
	}

	// Step 3: Fetch all active rows with non-null account_id, ordered for grouping
	var activeRows []*cloudIntegrationRow
	err = tx.NewSelect().
		Model(&activeRows).
		Where("removed_at IS NULL").
		Where("account_id IS NOT NULL").
		OrderExpr("account_id, provider, org_id, updated_at DESC").
		Scan(ctx)
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return err
	}

	// Group by (account_id, provider, org_id)
	groups := groupCloudIntegrationRows(activeRows)

	now := time.Now()
	var loserIDs []string

	for _, group := range groups {
		if len(group.losers) == 0 {
			continue
		}

		// Step 4: Merge config from losers into keeper
		if err = mergeCloudIntegrationConfigs(ctx, tx, group); err != nil {
			return err
		}

		// Step 5: Reassign non-conflicting cloud_integration_service rows to keeper
		for _, loser := range group.losers {
			_, err = tx.NewUpdate().
				TableExpr("cloud_integration_service").
				Set("cloud_integration_id = ?", group.keeper.ID).
				Where("cloud_integration_id = ?", loser.ID).
				Where("type NOT IN (?)",
					tx.NewSelect().
						TableExpr("cloud_integration_service").
						Column("type").
						Where("cloud_integration_id = ?", group.keeper.ID),
				).
				Exec(ctx)
			if err != nil {
				return err
			}

			loserIDs = append(loserIDs, loser.ID)
		}
	}

	// Step 6: Soft-delete all loser rows
	if len(loserIDs) > 0 {
		_, err = tx.NewUpdate().
			TableExpr("cloud_integration").
			Set("removed_at = ?", now).
			Set("updated_at = ?", now).
			Where("id IN (?)", bun.In(loserIDs)).
			Exec(ctx)
		if err != nil {
			return err
		}
	}

	// Step 7: Create the correct partial unique index on (account_id, provider, org_id) WHERE removed_at IS NULL
	createSqls := migration.sqlschema.Operator().CreateIndex(
		&sqlschema.PartialUniqueIndex{
			TableName:   "cloud_integration",
			ColumnNames: []sqlschema.ColumnName{"account_id", "provider", "org_id"},
			Where:       "removed_at IS NULL",
		},
	)
	sqls = append(sqls, createSqls...)

	for _, sql := range sqls {
		if _, err = tx.ExecContext(ctx, string(sql)); err != nil {
			return err
		}
	}

	return tx.Commit()
}

func (migration *updateCloudIntegrationUniqueIndex) Down(ctx context.Context, db *bun.DB) error {
	return nil
}

// groupCloudIntegrationRows groups rows by (account_id, provider, org_id).
// Rows must be pre-sorted by account_id, provider, org_id, updated_at DESC
// so the first row in each group is the keeper (most recently updated).
func groupCloudIntegrationRows(rows []*cloudIntegrationRow) []duplicateGroup {
	if len(rows) == 0 {
		return nil
	}

	var groups []duplicateGroup
	var current duplicateGroup
	current.keeper = rows[0]

	for i := 1; i < len(rows); i++ {
		row := rows[i]
		if row.AccountID == current.keeper.AccountID &&
			row.Provider == current.keeper.Provider &&
			row.OrgID == current.keeper.OrgID {
			current.losers = append(current.losers, row)
		} else {
			groups = append(groups, current)
			current = duplicateGroup{keeper: row}
		}
	}
	groups = append(groups, current)

	return groups
}

// mergeCloudIntegrationConfigs unions the EnabledRegions from all rows in the group into the keeper's config and updates
func mergeCloudIntegrationConfigs(ctx context.Context, tx bun.Tx, group duplicateGroup) error {
	regionSet := make(map[string]struct{})

	// Parse keeper's config
	parseRegions(group.keeper.Config, regionSet)

	// Parse each loser's config
	for _, loser := range group.losers {
		parseRegions(loser.Config, regionSet)
	}

	// Build merged config
	mergedRegions := make([]string, 0, len(regionSet))
	for region := range regionSet {
		mergedRegions = append(mergedRegions, region)
	}

	merged := cloudIntegrationAccountConfig{Regions: mergedRegions}
	mergedJSON, err := json.Marshal(merged)
	if err != nil {
		return err
	}

	// Update keeper's config
	_, err = tx.NewUpdate().
		TableExpr("cloud_integration").
		Set("config = ?", string(mergedJSON)).
		Where("id = ?", group.keeper.ID).
		Exec(ctx)
	return err
}

// parseRegions unmarshals a config JSON string and adds its regions to the set.
func parseRegions(configJSON string, regionSet map[string]struct{}) {
	if configJSON == "" {
		return
	}

	var config cloudIntegrationAccountConfig
	if err := json.Unmarshal([]byte(configJSON), &config); err != nil {
		return
	}

	for _, region := range config.Regions {
		regionSet[region] = struct{}{}
	}
}

var (
	userRoleToSigNozManagedRoleMap = map[string]string{
		"ADMIN":  "signoz-admin",
		"EDITOR": "signoz-editor",
		"VIEWER": "signoz-viewer",
	}
)

type userRow struct {
	ID    string `bun:"id"`
	Role  string `bun:"role"`
	OrgID string `bun:"org_id"`
}

type roleRow struct {
	ID    string `bun:"id"`
	Name  string `bun:"name"`
	OrgID string `bun:"org_id"`
}

type orgRoleKey struct {
	OrgID    string
	RoleName string
}

type addUserRole struct {
	sqlstore  sqlstore.SQLStore
	sqlschema sqlschema.SQLSchema
}

type userRoleRow struct {
	bun.BaseModel `bun:"table:user_role"`

	types.Identifiable
	UserID string `bun:"user_id"`
	RoleID string `bun:"role_id"`
	types.TimeAuditable
}

func NewAddUserRoleFactory(sqlstore sqlstore.SQLStore, sqlschema sqlschema.SQLSchema) factory.ProviderFactory[SQLMigration, Config] {
	return factory.NewProviderFactory(factory.MustNewName("add_user_role"), func(ctx context.Context, ps factory.ProviderSettings, c Config) (SQLMigration, error) {
		return &addUserRole{
			sqlstore:  sqlstore,
			sqlschema: sqlschema,
		}, nil
	})
}

func (migration *addUserRole) Register(migrations *migrate.Migrations) error {
	if err := migrations.Register(migration.Up, migration.Down); err != nil {
		return err
	}

	return nil
}

func (migration *addUserRole) Up(ctx context.Context, db *bun.DB) error {
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}

	defer func() {
		_ = tx.Rollback()
	}()

	sqls := [][]byte{}

	tableSQLs := migration.sqlschema.Operator().CreateTable(&sqlschema.Table{
		Name: "user_role",
		Columns: []*sqlschema.Column{
			{Name: "id", DataType: sqlschema.DataTypeText, Nullable: false},
			{Name: "user_id", DataType: sqlschema.DataTypeText, Nullable: false},
			{Name: "role_id", DataType: sqlschema.DataTypeText, Nullable: false},
			{Name: "created_at", DataType: sqlschema.DataTypeTimestamp, Nullable: false},
			{Name: "updated_at", DataType: sqlschema.DataTypeTimestamp, Nullable: false},
		},
		PrimaryKeyConstraint: &sqlschema.PrimaryKeyConstraint{
			ColumnNames: []sqlschema.ColumnName{"id"},
		},
		ForeignKeyConstraints: []*sqlschema.ForeignKeyConstraint{
			{
				ReferencingColumnName: sqlschema.ColumnName("user_id"),
				ReferencedTableName:   sqlschema.TableName("users"),
				ReferencedColumnName:  sqlschema.ColumnName("id"),
			},
			{
				ReferencingColumnName: sqlschema.ColumnName("role_id"),
				ReferencedTableName:   sqlschema.TableName("role"),
				ReferencedColumnName:  sqlschema.ColumnName("id"),
			},
		},
	})

	sqls = append(sqls, tableSQLs...)

	indexSQLs := migration.sqlschema.Operator().CreateIndex(
		&sqlschema.UniqueIndex{
			TableName:   "user_role",
			ColumnNames: []sqlschema.ColumnName{"user_id", "role_id"},
		},
	)

	sqls = append(sqls, indexSQLs...)

	for _, sql := range sqls {
		if _, err := tx.ExecContext(ctx, string(sql)); err != nil {
			return err
		}
	}

	// fill the new user_role table for existing users
	var users []userRow
	err = tx.NewSelect().TableExpr("users").ColumnExpr("id, role, org_id").Scan(ctx, &users)
	if err != nil {
		return err
	}

	if len(users) == 0 {
		return tx.Commit()
	}

	orgIDs := make(map[string]struct{})
	for _, u := range users {
		orgIDs[u.OrgID] = struct{}{}
	}

	orgIDList := make([]string, 0, len(orgIDs))
	for oid := range orgIDs {
		orgIDList = append(orgIDList, oid)
	}

	var roles []roleRow
	err = tx.NewSelect().TableExpr("role").ColumnExpr("id, name, org_id").Where("org_id IN (?)", bun.In(orgIDList)).Scan(ctx, &roles)
	if err != nil {
		return err
	}

	roleMap := make(map[orgRoleKey]string)
	for _, r := range roles {
		roleMap[orgRoleKey{OrgID: r.OrgID, RoleName: r.Name}] = r.ID
	}

	now := time.Now()
	userRoles := make([]*userRoleRow, 0, len(users))
	for _, u := range users {
		managedRoleName, ok := userRoleToSigNozManagedRoleMap[u.Role]
		if !ok {
			managedRoleName = "signoz-viewer" // fallback
		}

		roleID := roleMap[orgRoleKey{OrgID: u.OrgID, RoleName: managedRoleName}]

		userRoles = append(userRoles, &userRoleRow{
			Identifiable: types.Identifiable{ID: valuer.GenerateUUID()},
			UserID:       u.ID,
			RoleID:       roleID,
			TimeAuditable: types.TimeAuditable{
				CreatedAt: now,
				UpdatedAt: now,
			},
		})
	}

	if len(userRoles) > 0 {
		if _, err := tx.NewInsert().Model(&userRoles).Exec(ctx); err != nil {
			return err
		}
	}

	if err := tx.Commit(); err != nil {
		return err
	}

	return nil
}

func (migration *addUserRole) Down(ctx context.Context, db *bun.DB) error {
	return nil
}

type dropUserRoleColumn struct {
	sqlStore  sqlstore.SQLStore
	sqlSchema sqlschema.SQLSchema
}

func NewDropUserRoleColumnFactory(sqlStore sqlstore.SQLStore, sqlSchema sqlschema.SQLSchema) factory.ProviderFactory[SQLMigration, Config] {
	return factory.NewProviderFactory(factory.MustNewName("drop_user_role_column"), func(ctx context.Context, ps factory.ProviderSettings, c Config) (SQLMigration, error) {
		return &dropUserRoleColumn{
			sqlStore:  sqlStore,
			sqlSchema: sqlSchema,
		}, nil
	})
}

func (migration *dropUserRoleColumn) Register(migrations *migrate.Migrations) error {
	if err := migrations.Register(migration.Up, migration.Down); err != nil {
		return err
	}

	return nil
}

func (migration *dropUserRoleColumn) Up(ctx context.Context, db *bun.DB) error {
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}

	defer func() {
		_ = tx.Rollback()
	}()

	table, _, err := migration.sqlSchema.GetTable(ctx, sqlschema.TableName("users"))
	if err != nil {
		return err
	}

	roleColumn := &sqlschema.Column{
		Name:     sqlschema.ColumnName("role"),
		DataType: sqlschema.DataTypeText,
		Nullable: false,
	}

	sqls := migration.sqlSchema.Operator().DropColumn(table, roleColumn)

	for _, sql := range sqls {
		if _, err := tx.ExecContext(ctx, string(sql)); err != nil {
			return err
		}
	}

	if err := tx.Commit(); err != nil {
		return err
	}

	return nil
}

func (migration *dropUserRoleColumn) Down(ctx context.Context, db *bun.DB) error {
	return nil
}

type addAssistantConfig struct {
	sqlschema sqlschema.SQLSchema
}

func NewAddAssistantConfigFactory(sqlschema sqlschema.SQLSchema) factory.ProviderFactory[SQLMigration, Config] {
	return factory.NewProviderFactory(factory.MustNewName("add_assistant_config"), func(ctx context.Context, providerSettings factory.ProviderSettings, config Config) (SQLMigration, error) {
		return &addAssistantConfig{sqlschema: sqlschema}, nil
	})
}

func (migration *addAssistantConfig) Register(migrations *migrate.Migrations) error {
	return migrations.Register(migration.Up, migration.Down)
}

func (migration *addAssistantConfig) Up(ctx context.Context, db *bun.DB) error {
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()

	sqls := migration.sqlschema.Operator().CreateTable(&sqlschema.Table{
		Name: "assistant_config",
		Columns: []*sqlschema.Column{
			{Name: "org_id", DataType: sqlschema.DataTypeText, Nullable: false},
			{Name: "base_url", DataType: sqlschema.DataTypeText, Nullable: false},
			{Name: "model", DataType: sqlschema.DataTypeText, Nullable: false},
			{Name: "api_key", DataType: sqlschema.DataTypeText, Nullable: false},
			{Name: "created_at", DataType: sqlschema.DataTypeTimestamp, Nullable: false},
			{Name: "updated_at", DataType: sqlschema.DataTypeTimestamp, Nullable: false},
		},
		PrimaryKeyConstraint: &sqlschema.PrimaryKeyConstraint{ColumnNames: []sqlschema.ColumnName{"org_id"}},
		ForeignKeyConstraints: []*sqlschema.ForeignKeyConstraint{{
			ReferencingColumnName: "org_id",
			ReferencedTableName:   "organizations",
			ReferencedColumnName:  "id",
		}},
	})
	for _, sql := range sqls {
		if _, err := tx.ExecContext(ctx, string(sql)); err != nil {
			return err
		}
	}

	return tx.Commit()
}

func (migration *addAssistantConfig) Down(ctx context.Context, db *bun.DB) error {
	return nil
}

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

const logPipelinesElementType = "log_pipelines"

type dropLogPipelines struct{}

func NewDropLogPipelinesFactory() factory.ProviderFactory[SQLMigration, Config] {
	return factory.NewProviderFactory(factory.MustNewName("drop_log_pipelines"), func(context.Context, factory.ProviderSettings, Config) (SQLMigration, error) {
		return &dropLogPipelines{}, nil
	})
}

func (migration *dropLogPipelines) Register(migrations *migrate.Migrations) error {
	return migrations.Register(migration.Up, migration.Down)
}

func (migration *dropLogPipelines) Up(ctx context.Context, db *bun.DB) error {
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()

	if _, err := tx.NewDelete().
		Table("agent_config_element").
		Where("element_type = ?", logPipelinesElementType).
		Exec(ctx); err != nil {
		return err
	}

	if _, err := tx.NewDelete().
		Table("agent_config_version").
		Where("element_type = ?", logPipelinesElementType).
		Exec(ctx); err != nil {
		return err
	}

	if _, err := tx.NewDropTable().
		IfExists().
		Table("pipelines").
		Exec(ctx); err != nil {
		return err
	}

	return tx.Commit()
}

func (migration *dropLogPipelines) Down(context.Context, *bun.DB) error {
	return nil
}

const publicDashboardObjectPattern = "organization/%/public-dashboard/%"

type dropDashboardTables struct{}

func NewDropDashboardTablesFactory() factory.ProviderFactory[SQLMigration, Config] {
	return factory.NewProviderFactory(
		factory.MustNewName("drop_dashboard_tables"),
		func(context.Context, factory.ProviderSettings, Config) (SQLMigration, error) {
			return &dropDashboardTables{}, nil
		},
	)
}

func (migration *dropDashboardTables) Register(migrations *migrate.Migrations) error {
	return migrations.Register(migration.Up, migration.Down)
}

func (migration *dropDashboardTables) Up(ctx context.Context, db *bun.DB) error {
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()

	for _, table := range []string{"tuple", "changelog"} {
		if _, err := tx.NewDelete().
			Table(table).
			Where("object_id LIKE ?", publicDashboardObjectPattern).
			Exec(ctx); err != nil {
			return err
		}
	}

	for _, table := range []string{"public_dashboard", "dashboard", "dashboards"} {
		if _, err := tx.NewDropTable().IfExists().Table(table).Exec(ctx); err != nil {
			return err
		}
	}

	return tx.Commit()
}

func (migration *dropDashboardTables) Down(context.Context, *bun.DB) error {
	return nil
}
