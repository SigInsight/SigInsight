package analytics

import (
	"github.com/SigNoz/signoz/pkg/factory"
)

type Config struct {
	Enabled bool `mapstructure:"enabled"`
}

func NewConfigFactory() factory.ConfigFactory {
	return factory.NewConfigFactory(factory.MustNewName("analytics"), newConfig)
}

func newConfig() factory.Config {
	return Config{
		Enabled: false,
	}
}

func (c Config) Validate() error {
	return nil
}

// Provider always returns "noop": the Segment analytics backend has been
// removed from this community-only fork, so analytics never phones home.
func (c Config) Provider() string {
	return "noop"
}
