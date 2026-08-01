package alertmanager

import (
	"context"
	"net/url"
	"testing"
	"time"

	"github.com/SigNoz/signoz/pkg/config"
	"github.com/SigNoz/signoz/pkg/config/envprovider"
	"github.com/SigNoz/signoz/pkg/factory"
	"github.com/prometheus/common/model"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewWithEnvProvider(t *testing.T) {
	t.Setenv("SIGINSIGHT_ALERTMANAGER_PROVIDER", "siginsight")
	t.Setenv("SIGINSIGHT_ALERTMANAGER_LEGACY_API__URL", "http://localhost:9093/api")
	t.Setenv("SIGINSIGHT_ALERTMANAGER_SIGINSIGHT_ROUTE_REPEAT__INTERVAL", "5m")
	t.Setenv("SIGINSIGHT_ALERTMANAGER_SIGINSIGHT_EXTERNAL__URL", "https://example.com/test")
	t.Setenv("SIGINSIGHT_ALERTMANAGER_SIGINSIGHT_GLOBAL_RESOLVE__TIMEOUT", "10s")

	conf, err := config.New(
		context.Background(),
		config.ResolverConfig{
			Uris: []string{"env:"},
			ProviderFactories: []config.ProviderFactory{
				envprovider.NewFactory(),
			},
		},
		[]factory.ConfigFactory{
			NewConfigFactory(),
		},
	)
	require.NoError(t, err)

	actual := &Config{}
	err = conf.Unmarshal("alertmanager", actual, "yaml")
	require.NoError(t, err)

	def := NewConfigFactory().New().(Config)
	def.SigInsight.Global.ResolveTimeout = model.Duration(10 * time.Second)
	def.SigInsight.Route.RepeatInterval = 5 * time.Minute
	def.SigInsight.ExternalURL = &url.URL{
		Scheme: "https",
		Host:   "example.com",
		Path:   "/test",
	}

	expected := &Config{
		Provider:   "siginsight",
		SigInsight: def.SigInsight,
	}

	assert.Equal(t, expected, actual)
	assert.NoError(t, actual.Validate())
}
