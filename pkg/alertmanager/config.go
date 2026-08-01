package alertmanager

import (
	"net/url"
	"strings"
	"time"

	"github.com/SigNoz/signoz/pkg/alertmanager/alertmanagerserver"
	"github.com/SigNoz/signoz/pkg/errors"
	"github.com/SigNoz/signoz/pkg/factory"
)

type Config struct {
	// Provider is the provider for the alertmanager service.
	Provider string `mapstructure:"provider"`

	// SigInsight is the internal alertmanager configuration.
	SigInsight SigInsight `mapstructure:"siginsight" yaml:"siginsight"`
}

type SigInsight struct {
	// PollInterval is the interval at which the alertmanager is synced.
	PollInterval time.Duration `mapstructure:"poll_interval"`

	// Config is the config for the alertmanager server.
	alertmanagerserver.Config `mapstructure:",squash" yaml:",squash"`
}

type Legacy struct {
	// ApiURL is the URL of the legacy signoz alertmanager.
	ApiURL *url.URL `mapstructure:"api_url"`
}

func NewConfigFactory() factory.ConfigFactory {
	return factory.NewConfigFactory(factory.MustNewName("alertmanager"), newConfig)
}

func newConfig() factory.Config {
	return Config{
		Provider: "siginsight",
		SigInsight: SigInsight{
			PollInterval: 1 * time.Minute,
			Config:       alertmanagerserver.NewConfig(),
		},
	}
}

func (c Config) Validate() error {
	if c.Provider != "siginsight" {
		return errors.Newf(errors.TypeInvalidInput, errors.CodeInvalidInput, "provider must be one of [%s], got %s", strings.Join([]string{"siginsight"}, ", "), c.Provider)
	}

	return nil
}
