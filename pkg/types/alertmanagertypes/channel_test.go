package alertmanagertypes

import (
	"testing"
	"time"

	"github.com/prometheus/alertmanager/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewConfigFromChannelsAcceptsSupportedChannels(t *testing.T) {
	cfg, err := NewConfigFromChannels(
		GlobalConfig{SMTPSmarthost: config.HostPort{Host: "localhost", Port: "25"}, SMTPFrom: "alerts@example.com"},
		RouteConfig{
			GroupInterval:  time.Minute,
			GroupWait:      time.Minute,
			RepeatInterval: time.Minute,
		},
		Channels{
			{Name: "email", Type: "email", Data: `{"name":"email","email_configs":[{"to":"alerts@example.com"}]}`},
			{Name: "webhook", Type: "webhook", Data: `{"name":"webhook","webhook_configs":[{"url":"https://example.com/alerts"}]}`},
		},
		"org",
	)
	require.NoError(t, err)
	assert.Len(t, cfg.alertmanagerConfig.Receivers, 3)
}

func TestNewConfigFromChannelsRejectsUnsupportedChannel(t *testing.T) {
	_, err := NewConfigFromChannels(
		GlobalConfig{},
		RouteConfig{
			GroupInterval:  time.Minute,
			GroupWait:      time.Minute,
			RepeatInterval: time.Minute,
		},
		Channels{{Name: "legacy", Type: "slack", Data: `{"name":"legacy","slack_configs":[{}]}`}},
		"org",
	)
	assert.Error(t, err)
}

func TestNewChannelFromReceiverSupportsEmailAndWebhookOnly(t *testing.T) {
	tests := []struct {
		name     string
		receiver config.Receiver
		typeName string
		valid    bool
	}{
		{
			name: "email",
			receiver: config.Receiver{
				Name:         "email",
				EmailConfigs: []*config.EmailConfig{{To: "alerts@example.com"}},
			},
			typeName: "email",
			valid:    true,
		},
		{
			name: "webhook",
			receiver: config.Receiver{
				Name:           "webhook",
				WebhookConfigs: []*config.WebhookConfig{{URL: config.SecretTemplateURL("https://example.com/alerts")}},
			},
			typeName: "webhook",
			valid:    true,
		},
		{
			name: "unsupported slack",
			receiver: config.Receiver{
				Name:         "slack",
				SlackConfigs: []*config.SlackConfig{{Channel: "#alerts"}},
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			channel, err := NewChannelFromReceiver(test.receiver, "org")
			if !test.valid {
				assert.Error(t, err)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, test.typeName, channel.Type)
		})
	}
}
