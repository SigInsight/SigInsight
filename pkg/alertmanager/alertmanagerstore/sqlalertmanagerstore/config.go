package sqlalertmanagerstore

import (
	"context"
	"database/sql"
	"encoding/json"

	"github.com/SigNoz/signoz/pkg/errors"
	"github.com/SigNoz/signoz/pkg/sqlstore"
	"github.com/SigNoz/signoz/pkg/types/alertmanagertypes"
	"github.com/SigNoz/signoz/pkg/types/ruletypes"
	"github.com/SigNoz/signoz/pkg/valuer"
)

type config struct {
	sqlstore sqlstore.SQLStore
}

func NewConfigStore(sqlstore sqlstore.SQLStore) alertmanagertypes.ConfigStore {
	return &config{sqlstore: sqlstore}
}

// Get implements alertmanagertypes.ConfigStore.
func (store *config) Get(ctx context.Context, orgID string) (*alertmanagertypes.Config, error) {
	storeableConfig := new(alertmanagertypes.StoreableConfig)

	err := store.
		sqlstore.
		BunDB().
		NewSelect().
		Model(storeableConfig).
		Where("org_id = ?", orgID).
		Scan(ctx)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, errors.Newf(errors.TypeNotFound, alertmanagertypes.ErrCodeAlertmanagerConfigNotFound, "cannot find alertmanager config for orgID %s", orgID)
		}

		return nil, err
	}

	cfg, err := alertmanagertypes.NewConfigFromStoreableConfig(storeableConfig)
	if err != nil {
		return nil, err
	}

	return cfg, nil
}

// Set implements alertmanagertypes.ConfigStore.
func (store *config) Set(ctx context.Context, config *alertmanagertypes.Config, opts ...alertmanagertypes.StoreOption) error {
	return store.wrap(ctx, func(ctx context.Context) error {
		if _, err := store.
			sqlstore.
			BunDBCtx(ctx).
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
	}, opts...)
}

func (store *config) CreateChannel(ctx context.Context, channel *alertmanagertypes.Channel, opts ...alertmanagertypes.StoreOption) error {
	return store.wrap(ctx, func(ctx context.Context) error {
		if _, err := store.
			sqlstore.
			BunDBCtx(ctx).
			NewInsert().
			Model(channel).
			Exec(ctx); err != nil {
			return err
		}

		return nil
	}, opts...)
}

func (store *config) GetChannelByID(ctx context.Context, orgID string, id valuer.UUID) (*alertmanagertypes.Channel, error) {
	channel := new(alertmanagertypes.Channel)

	err := store.
		sqlstore.
		BunDB().
		NewSelect().
		Model(channel).
		Where("org_id = ?", orgID).
		Where("id = ?", id.StringValue()).
		Scan(ctx)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, errors.Newf(errors.TypeNotFound, alertmanagertypes.ErrCodeAlertmanagerChannelNotFound, "cannot find channel with id %s", id.StringValue())
		}
		return nil, err
	}

	return channel, nil
}

func (store *config) UpdateChannel(ctx context.Context, orgID string, channel *alertmanagertypes.Channel, opts ...alertmanagertypes.StoreOption) error {
	return store.wrap(ctx, func(ctx context.Context) error {
		if _, err := store.
			sqlstore.
			BunDBCtx(ctx).
			NewUpdate().
			Model(channel).
			WherePK().
			Exec(ctx); err != nil {
			return err
		}

		return nil
	}, opts...)
}

func (store *config) DeleteChannelByID(ctx context.Context, orgID string, id valuer.UUID, opts ...alertmanagertypes.StoreOption) error {
	return store.wrap(ctx, func(ctx context.Context) error {
		channel := new(alertmanagertypes.Channel)

		if _, err := store.
			sqlstore.
			BunDBCtx(ctx).
			NewDelete().
			Model(channel).
			Where("org_id = ?", orgID).
			Where("id = ?", id.StringValue()).
			Exec(ctx); err != nil {
			return err
		}

		return nil
	}, opts...)
}

func (store *config) ListChannels(ctx context.Context, orgID string) ([]*alertmanagertypes.Channel, error) {
	var channels []*alertmanagertypes.Channel

	err := store.
		sqlstore.
		BunDB().
		NewSelect().
		Model(&channels).
		Where("org_id = ?", orgID).
		Scan(ctx)
	if err != nil {
		return nil, err
	}

	return channels, nil
}

func (store *config) GetMatchers(ctx context.Context, orgID string) (map[string][]string, error) {
	rules := []ruletypes.Rule{}

	err := store.
		sqlstore.
		BunDB().
		NewSelect().
		Model(&rules).
		Where("org_id = ?", orgID).
		Where("deleted = ?", 0).
		Scan(ctx)
	if err != nil {
		return nil, err
	}

	matchersMap := make(map[string][]string)
	for _, rule := range rules {
		for _, receiver := range ruleChannels(rule.Data) {
			matchersMap[rule.ID.StringValue()] = append(matchersMap[rule.ID.StringValue()], receiver)
		}
	}

	return matchersMap, nil
}

// ruleChannels keeps alertmanager's persisted route in sync with the current
// threshold schema while retaining the old preferredChannels representation.
func ruleChannels(data string) []string {
	var stored struct {
		PreferredChannels []string `json:"preferredChannels"`
		Condition         struct {
			Thresholds struct {
				Spec []struct {
					Channels []string `json:"channels"`
				} `json:"spec"`
			} `json:"thresholds"`
		} `json:"condition"`
	}
	if err := json.Unmarshal([]byte(data), &stored); err != nil {
		return nil
	}

	channels := make([]string, 0, len(stored.PreferredChannels))
	seen := make(map[string]struct{}, len(stored.PreferredChannels))
	appendChannel := func(channel string) {
		if _, exists := seen[channel]; exists {
			return
		}
		seen[channel] = struct{}{}
		channels = append(channels, channel)
	}
	for _, threshold := range stored.Condition.Thresholds.Spec {
		for _, channel := range threshold.Channels {
			appendChannel(channel)
		}
	}
	if len(channels) == 0 {
		for _, channel := range stored.PreferredChannels {
			appendChannel(channel)
		}
	}

	return channels
}

func (store *config) wrap(ctx context.Context, fn func(ctx context.Context) error, opts ...alertmanagertypes.StoreOption) error {
	storeOpts := alertmanagertypes.NewStoreOptions(opts...)

	if storeOpts.Cb == nil {
		return fn(ctx)
	}

	return store.sqlstore.RunInTxCtx(ctx, nil, func(ctx context.Context) error {
		if err := fn(ctx); err != nil {
			return err
		}

		return storeOpts.Cb(ctx)
	})
}
