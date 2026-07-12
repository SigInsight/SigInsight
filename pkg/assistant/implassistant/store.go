package implassistant

import (
	"context"
	"database/sql"
	"time"

	"github.com/SigNoz/signoz/pkg/assistant"
	"github.com/SigNoz/signoz/pkg/errors"
	"github.com/SigNoz/signoz/pkg/sqlstore"
	"github.com/uptrace/bun"
)

type storedConfig struct {
	bun.BaseModel `bun:"table:assistant_config"`

	OrgID     string    `bun:"org_id,pk,type:text,notnull"`
	BaseURL   string    `bun:"base_url,type:text,notnull"`
	Model     string    `bun:"model,type:text,notnull"`
	APIKey    string    `bun:"api_key,type:text,notnull"`
	CreatedAt time.Time `bun:"created_at,type:timestamp,notnull"`
	UpdatedAt time.Time `bun:"updated_at,type:timestamp,notnull"`
}

type store struct {
	sqlstore sqlstore.SQLStore
}

func NewStore(sqlstore sqlstore.SQLStore) assistant.Store {
	return &store{sqlstore: sqlstore}
}

func (store *store) GetConfig(ctx context.Context, orgID string) (*assistant.Config, error) {
	config := new(storedConfig)
	err := store.sqlstore.BunDB().NewSelect().Model(config).Where("org_id = ?", orgID).Scan(ctx)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, assistant.ErrConfigNotFound
		}
		return nil, err
	}

	return &assistant.Config{
		BaseURL: config.BaseURL,
		Model:   config.Model,
		APIKey:  config.APIKey,
	}, nil
}

func (store *store) UpsertConfig(ctx context.Context, orgID string, config assistant.Config) error {
	now := time.Now().UTC()
	row := &storedConfig{
		OrgID:     orgID,
		BaseURL:   config.BaseURL,
		Model:     config.Model,
		APIKey:    config.APIKey,
		CreatedAt: now,
		UpdatedAt: now,
	}

	_, err := store.sqlstore.BunDB().NewInsert().Model(row).
		On("CONFLICT (org_id) DO UPDATE").
		Set("base_url = EXCLUDED.base_url").
		Set("model = EXCLUDED.model").
		Set("api_key = EXCLUDED.api_key").
		Set("updated_at = EXCLUDED.updated_at").
		Exec(ctx)
	return err
}
