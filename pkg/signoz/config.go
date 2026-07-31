package signoz

import (
	"context"
	"log/slog"
	"reflect"

	"github.com/SigNoz/signoz/pkg/alertmanager"
	"github.com/SigNoz/signoz/pkg/analytics"
	"github.com/SigNoz/signoz/pkg/apiserver"
	"github.com/SigNoz/signoz/pkg/cache"
	"github.com/SigNoz/signoz/pkg/config"
	"github.com/SigNoz/signoz/pkg/emailing"
	"github.com/SigNoz/signoz/pkg/errors"
	"github.com/SigNoz/signoz/pkg/factory"
	"github.com/SigNoz/signoz/pkg/flagger"
	"github.com/SigNoz/signoz/pkg/identn"
	"github.com/SigNoz/signoz/pkg/instrumentation"
	"github.com/SigNoz/signoz/pkg/modules/metricsexplorer"
	"github.com/SigNoz/signoz/pkg/modules/user"
	"github.com/SigNoz/signoz/pkg/pprof"
	"github.com/SigNoz/signoz/pkg/querier"
	"github.com/SigNoz/signoz/pkg/ruler"
	"github.com/SigNoz/signoz/pkg/sharder"
	"github.com/SigNoz/signoz/pkg/sqlmigration"
	"github.com/SigNoz/signoz/pkg/sqlmigrator"
	"github.com/SigNoz/signoz/pkg/sqlstore"
	"github.com/SigNoz/signoz/pkg/statsreporter"
	"github.com/SigNoz/signoz/pkg/telemetrystore"
	"github.com/SigNoz/signoz/pkg/tokenizer"
	"github.com/SigNoz/signoz/pkg/valuer"
	"github.com/SigNoz/signoz/pkg/version"
	"github.com/SigNoz/signoz/pkg/web"
)

// Config defines the entire input configuration of signoz.
type Config struct {
	// Version config
	Version version.Config `mapstructure:"version"`

	// Instrumentation config
	Instrumentation instrumentation.Config `mapstructure:"instrumentation"`

	// PProf config
	PProf pprof.Config `mapstructure:"pprof"`

	// Analytics config
	Analytics analytics.Config `mapstructure:"analytics"`

	// Web config
	Web web.Config `mapstructure:"web"`

	// Cache config
	Cache cache.Config `mapstructure:"cache"`

	// SQLStore config
	SQLStore sqlstore.Config `mapstructure:"sqlstore"`

	// SQLMigration config
	SQLMigration sqlmigration.Config `mapstructure:"sqlmigration"`

	// SQLMigrator config
	SQLMigrator sqlmigrator.Config `mapstructure:"sqlmigrator"`

	// API Server config
	APIServer apiserver.Config `mapstructure:"apiserver"`

	// TelemetryStore config
	TelemetryStore telemetrystore.Config `mapstructure:"telemetrystore"`

	// Alertmanager config
	Alertmanager alertmanager.Config `mapstructure:"alertmanager" yaml:"alertmanager"`

	// Querier config
	Querier querier.Config `mapstructure:"querier"`

	// Ruler config
	Ruler ruler.Config `mapstructure:"ruler"`

	// Emailing config
	Emailing emailing.Config `mapstructure:"emailing" yaml:"emailing"`

	// Sharder config
	Sharder sharder.Config `mapstructure:"sharder" yaml:"sharder"`

	// StatsReporter config
	StatsReporter statsreporter.Config `mapstructure:"statsreporter"`

	// Tokenizer config
	Tokenizer tokenizer.Config `mapstructure:"tokenizer"`

	// MetricsExplorer config
	MetricsExplorer metricsexplorer.Config `mapstructure:"metricsexplorer"`

	// Flagger config
	Flagger flagger.Config `mapstructure:"flagger"`

	// User config
	User user.Config `mapstructure:"user"`

	// IdentN config
	IdentN identn.Config `mapstructure:"identn"`
}

func NewConfig(ctx context.Context, logger *slog.Logger, resolverConfig config.ResolverConfig) (Config, error) {
	configFactories := []factory.ConfigFactory{
		version.NewConfigFactory(),
		instrumentation.NewConfigFactory(),
		pprof.NewConfigFactory(),
		analytics.NewConfigFactory(),
		web.NewConfigFactory(),
		cache.NewConfigFactory(),
		sqlstore.NewConfigFactory(),
		sqlmigrator.NewConfigFactory(),
		apiserver.NewConfigFactory(),
		telemetrystore.NewConfigFactory(),
		alertmanager.NewConfigFactory(),
		querier.NewConfigFactory(),
		ruler.NewConfigFactory(),
		emailing.NewConfigFactory(),
		sharder.NewConfigFactory(),
		statsreporter.NewConfigFactory(),
		tokenizer.NewConfigFactory(),
		metricsexplorer.NewConfigFactory(),
		flagger.NewConfigFactory(),
		user.NewConfigFactory(),
		identn.NewConfigFactory(),
	}

	conf, err := config.New(ctx, resolverConfig, configFactories)
	if err != nil {
		return Config{}, err
	}

	var config Config
	if err := conf.Unmarshal("", &config, "yaml"); err != nil {
		return Config{}, err
	}

	if err := validateConfig(config); err != nil {
		return Config{}, err
	}

	return config, nil
}

func validateConfig(config Config) error {
	rvConfig := reflect.ValueOf(config)
	for i := 0; i < rvConfig.NumField(); i++ {
		factoryConfig, ok := rvConfig.Field(i).Interface().(factory.Config)
		if !ok {
			return errors.NewInvalidInputf(errors.CodeInvalidInput, "%q is not of type \"factory.Config\"", rvConfig.Type().Field(i).Name)
		}

		if err := factoryConfig.Validate(); err != nil {
			return errors.WrapInvalidInputf(err, errors.CodeInvalidInput, "failed to validate config %q", rvConfig.Type().Field(i).Tag.Get("mapstructure"))
		}
	}

	return nil
}

func (config Config) Collect(_ context.Context, _ valuer.UUID) (map[string]any, error) {
	stats := make(map[string]any)

	// SQL Store Config Stats
	stats["config.sqlstore.provider"] = config.SQLStore.Provider

	// Tokenizer Config Stats
	stats["config.tokenizer.provider"] = config.Tokenizer.Provider

	// Cache Config Stats
	stats["config.cache.provider"] = config.Cache.Provider

	return stats, nil
}
