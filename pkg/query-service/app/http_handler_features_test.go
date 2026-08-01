package app

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestUIFeatureFlagsExposeOnlyRemainingServerCapabilities(t *testing.T) {
	flags := uiFeatureFlags(false, true)
	require.Len(t, flags, 2)
	require.False(t, flags[0].Active)
	require.True(t, flags[1].Active)
}
