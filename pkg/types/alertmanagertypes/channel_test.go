package alertmanagertypes

import (
	"testing"
	"time"

	"github.com/SigNoz/signoz/pkg/types"
	"github.com/SigNoz/signoz/pkg/valuer"
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

func TestEmailChannelCreatedBeforeDefaultsInheritsGlobalSMTPConfig(t *testing.T) {
	receiver, err := NewReceiver(`{
		"name":"email",
		"email_configs":[{"to":"alerts@example.com"}]
	}`)
	require.NoError(t, err)
	channel, err := NewChannelFromReceiver(receiver, "org")
	require.NoError(t, err)
	assert.NotContains(t, channel.Data, "smtp.example.com")

	global := GlobalConfig{
		SMTPFrom:         "current@example.com",
		SMTPHello:        "current-host",
		SMTPSmarthost:    config.HostPort{Host: "smtp.example.com", Port: "587"},
		SMTPAuthUsername: "current-user",
		SMTPAuthPassword: "current-password",
		SMTPRequireTLS:   true,
	}

	cfg, err := NewConfigFromChannels(
		global,
		RouteConfig{GroupInterval: time.Minute, GroupWait: time.Minute, RepeatInterval: time.Minute},
		Channels{channel},
		"org",
	)
	require.NoError(t, err)
	receiver, err = cfg.GetReceiver("email")
	require.NoError(t, err)
	require.Len(t, receiver.EmailConfigs, 1)
	emailConfig := receiver.EmailConfigs[0]
	assert.Equal(t, global.SMTPFrom, emailConfig.From)
	assert.Equal(t, global.SMTPHello, emailConfig.Hello)
	assert.Equal(t, global.SMTPSmarthost, emailConfig.Smarthost)
	assert.Equal(t, global.SMTPAuthUsername, emailConfig.AuthUsername)
	assert.Equal(t, global.SMTPAuthPassword, emailConfig.AuthPassword)
	require.NotNil(t, emailConfig.RequireTLS)
	assert.True(t, *emailConfig.RequireTLS)
}

func TestResolveChannelNames(t *testing.T) {
	channelID := valuer.GenerateUUID()
	channels := Channels{{
		Identifiable: types.Identifiable{ID: channelID},
		Name:         "test notification channel",
	}}

	references := ResolveChannelNames(channels, []string{channelID.StringValue(), "legacy-receiver"})
	assert.Equal(t, []string{"test notification channel", "legacy-receiver"}, references)
}

func TestEmailChannelDataPreservesExplicitSMTPTransport(t *testing.T) {
	receiver, err := NewReceiver(`{
		"name":"email",
		"email_configs":[{
			"to":"alerts@example.com",
			"from":"sender@example.com",
			"smarthost":"smtp.example.com:587",
			"auth_username":"user",
			"auth_password":"secret"
		}]
	}`)
	require.NoError(t, err)

	channel, err := NewChannelFromReceiver(receiver, "org")
	require.NoError(t, err)
	assert.Contains(t, channel.Data, "smtp.example.com:587")
	assert.Contains(t, channel.Data, "sender@example.com")
	assert.Contains(t, channel.Data, "auth_username")
	assert.Contains(t, channel.Data, "alerts@example.com")
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
