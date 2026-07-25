package identn

import (
	"github.com/SigNoz/signoz/pkg/errors"
	"github.com/SigNoz/signoz/pkg/factory"
)

type Config struct {
	// Config for tokenizer identN resolver
	Tokenizer TokenizerConfig `mapstructure:"tokenizer"`

	// Config for impersonation identN resolver
	Impersonation ImpersonationConfig `mapstructure:"impersonation"`
}

type ImpersonationConfig struct {
	// Toggles the identN resolver
	Enabled bool `mapstructure:"enabled"`
}

type TokenizerConfig struct {
	// Toggles the identN resolver
	Enabled bool `mapstructure:"enabled"`

	// Headers to extract from incoming requests
	Headers []string `mapstructure:"headers"`
}

func NewConfigFactory() factory.ConfigFactory {
	return factory.NewConfigFactory(factory.MustNewName("identn"), newConfig)
}

func newConfig() factory.Config {
	return &Config{
		Tokenizer: TokenizerConfig{
			Enabled: true,
			Headers: []string{"Authorization", "Sec-WebSocket-Protocol"},
		},
		Impersonation: ImpersonationConfig{
			Enabled: false,
		},
	}
}

func (c Config) Validate() error {
	if c.Impersonation.Enabled {
		if c.Tokenizer.Enabled {
			return errors.New(errors.TypeInvalidInput, errors.CodeInvalidInput, "identn::impersonation cannot be enabled if identn::tokenizer is enabled")
		}

	}

	return nil
}
