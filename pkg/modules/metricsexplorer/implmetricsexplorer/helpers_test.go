package implmetricsexplorer

import (
	"testing"

	"github.com/SigNoz/signoz/pkg/types/metricsexplorertypes"
	"github.com/SigNoz/signoz/pkg/types/metrictypes"
	"github.com/stretchr/testify/require"
)

func TestEnrichStatsWithMetadataIncludesQuerySemantics(t *testing.T) {
	stats := []metricsexplorertypes.Stat{{MetricName: "requests"}}
	metadata := map[string]*metricsexplorertypes.MetricMetadata{
		"requests": {
			Description: "Request count",
			MetricType:  metrictypes.SumType,
			MetricUnit:  "{request}",
			Temporality: metrictypes.Cumulative,
			IsMonotonic: true,
		},
	}

	enrichStatsWithMetadata(stats, metadata)

	require.Equal(t, "Request count", stats[0].Description)
	require.Equal(t, metrictypes.SumType, stats[0].MetricType)
	require.Equal(t, "{request}", stats[0].MetricUnit)
	require.Equal(t, metrictypes.Cumulative, stats[0].Temporality)
	require.True(t, stats[0].IsMonotonic)
}
