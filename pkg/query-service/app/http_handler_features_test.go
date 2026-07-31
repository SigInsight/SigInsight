package app

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/SigNoz/signoz/pkg/types/licensetypes"
)

func TestUIFeatureFlagsAdvertiseRunningLightweightEngine(t *testing.T) {
	flags := uiFeatureFlags(false, true, true)
	require.Len(t, flags, 3)
	require.Equal(t, licensetypes.LightweightQueryEngineEnabled, flags[2].Name)
	require.True(t, flags[2].Active)
}

func TestUIFeatureFlagsKeepLightweightEngineDisabledByDefault(t *testing.T) {
	flags := uiFeatureFlags(false, false, false)
	require.False(t, flags[2].Active)
}
