package sqlmigration

import (
	"context"

	"github.com/SigNoz/signoz/pkg/factory"
	"github.com/SigNoz/signoz/pkg/sqlschema"
	"github.com/uptrace/bun"
	"github.com/uptrace/bun/migrate"
)

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
