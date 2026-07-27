package rulebasednotification

import (
	"context"
	"testing"
	"time"

	"github.com/SigNoz/signoz/pkg/alertmanager/nfmanager"
	"github.com/SigNoz/signoz/pkg/instrumentation/instrumentationtest"
	"github.com/SigNoz/signoz/pkg/types/alertmanagertypes"
	"github.com/prometheus/common/model"
	"github.com/stretchr/testify/require"
)

func TestMatchUsesRuleChannels(t *testing.T) {
	provider, err := New(context.Background(), instrumentationtest.New().ToProviderSettings(), nfmanager.Config{})
	require.NoError(t, err)

	config := &alertmanagertypes.NotificationConfig{
		Channels: []string{"email", "webhook"},
		NotificationGroup: map[model.LabelName]struct{}{
			"service.name": {},
		},
		Renotify: alertmanagertypes.ReNotificationConfig{RenotifyInterval: time.Hour},
	}
	require.NoError(t, provider.SetNotificationConfig("org", "rule", config))

	channels, err := provider.Match(context.Background(), "org", "rule", model.LabelSet{"service.name": "api"})
	require.NoError(t, err)
	require.Equal(t, []string{"email", "webhook"}, channels)
}

func TestNotificationConfigRejectsInvalidIdentity(t *testing.T) {
	provider, err := New(context.Background(), instrumentationtest.New().ToProviderSettings(), nfmanager.Config{})
	require.NoError(t, err)

	config := &alertmanagertypes.NotificationConfig{}
	require.Error(t, provider.SetNotificationConfig("", "rule", config))
	require.Error(t, provider.SetNotificationConfig("org", "", config))
	require.Error(t, provider.SetNotificationConfig("org", "rule", nil))
}
