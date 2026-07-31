package querier

import (
	"time"

	"github.com/SigNoz/signoz/pkg/errors"
	"github.com/SigNoz/signoz/pkg/factory"
)

// Config contains query-adjacent settings still consumed by specialized
// readers. V5 query execution itself is always served by Lite.
type Config struct {
	// FluxInterval is the interval for recent data that should not be cached
	// by Trace Detail's specialized reader.
	FluxInterval time.Duration `yaml:"flux_interval" mapstructure:"flux_interval"`
}

// NewConfigFactory creates a new config factory for querier
func NewConfigFactory() factory.ConfigFactory {
	return factory.NewConfigFactory(factory.MustNewName("querier"), newConfig)
}

func newConfig() factory.Config {
	return Config{
		FluxInterval: 5 * time.Minute,
	}
}

// Validate validates the configuration
func (c Config) Validate() error {
	if c.FluxInterval <= 0 {
		return errors.NewInvalidInputf(errors.CodeInvalidInput, "flux_interval must be positive, got %v", c.FluxInterval)
	}
	return nil
}

func (c Config) Provider() string {
	return "signoz"
}
