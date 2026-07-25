package alertmanagertest

import (
	"context"

	"github.com/SigNoz/signoz/pkg/types/alertmanagertypes"
	"github.com/SigNoz/signoz/pkg/valuer"
	"github.com/prometheus/alertmanager/config"
	"github.com/stretchr/testify/mock"
)

// MockAlertmanager is the test implementation of alertmanager.Alertmanager.
type MockAlertmanager struct{ mock.Mock }

func NewMockAlertmanager(t interface {
	mock.TestingT
	Cleanup(func())
}) *MockAlertmanager {
	m := &MockAlertmanager{}
	m.Test(t)
	t.Cleanup(func() { m.AssertExpectations(t) })
	return m
}

type MockAlertmanager_Expecter struct{ mock *mock.Mock }

func (m *MockAlertmanager) EXPECT() *MockAlertmanager_Expecter {
	return &MockAlertmanager_Expecter{mock: &m.Mock}
}

func (m *MockAlertmanager) GetAlerts(ctx context.Context, orgID string, params alertmanagertypes.GettableAlertsParams) (alertmanagertypes.GettableAlerts, error) {
	ret := m.Called(ctx, orgID, params)
	var alerts alertmanagertypes.GettableAlerts
	if result, ok := ret.Get(0).(alertmanagertypes.GettableAlerts); ok {
		alerts = result
	}
	return alerts, ret.Error(1)
}

type MockAlertmanager_GetAlerts_Call struct{ *mock.Call }

func (e *MockAlertmanager_Expecter) GetAlerts(ctx, orgID, params any) *MockAlertmanager_GetAlerts_Call {
	return &MockAlertmanager_GetAlerts_Call{Call: e.mock.On("GetAlerts", ctx, orgID, params)}
}

func (c *MockAlertmanager_GetAlerts_Call) Return(alerts alertmanagertypes.GettableAlerts, err error) *MockAlertmanager_GetAlerts_Call {
	c.Call.Return(alerts, err)
	return c
}

func (m *MockAlertmanager) PutAlerts(ctx context.Context, orgID string, alerts alertmanagertypes.PostableAlerts) error {
	return m.Called(ctx, orgID, alerts).Error(0)
}

func (m *MockAlertmanager) TestReceiver(ctx context.Context, orgID string, receiver alertmanagertypes.Receiver) error {
	return m.Called(ctx, orgID, receiver).Error(0)
}

func (m *MockAlertmanager) TestAlert(ctx context.Context, orgID, ruleID string, receivers map[*alertmanagertypes.PostableAlert][]string) error {
	return m.Called(ctx, orgID, ruleID, receivers).Error(0)
}

func (m *MockAlertmanager) ListChannels(ctx context.Context, orgID string) ([]*alertmanagertypes.Channel, error) {
	ret := m.Called(ctx, orgID)
	channels, _ := ret.Get(0).([]*alertmanagertypes.Channel)
	return channels, ret.Error(1)
}

func (m *MockAlertmanager) GetChannelByID(ctx context.Context, orgID string, id valuer.UUID) (*alertmanagertypes.Channel, error) {
	ret := m.Called(ctx, orgID, id)
	channel, _ := ret.Get(0).(*alertmanagertypes.Channel)
	return channel, ret.Error(1)
}

func (m *MockAlertmanager) UpdateChannelByReceiverAndID(ctx context.Context, orgID string, receiver alertmanagertypes.Receiver, id valuer.UUID) error {
	return m.Called(ctx, orgID, receiver, id).Error(0)
}

func (m *MockAlertmanager) CreateChannel(ctx context.Context, orgID string, receiver alertmanagertypes.Receiver) (*alertmanagertypes.Channel, error) {
	ret := m.Called(ctx, orgID, receiver)
	channel, _ := ret.Get(0).(*alertmanagertypes.Channel)
	return channel, ret.Error(1)
}

func (m *MockAlertmanager) DeleteChannelByID(ctx context.Context, orgID string, id valuer.UUID) error {
	return m.Called(ctx, orgID, id).Error(0)
}

func (m *MockAlertmanager) SetConfig(ctx context.Context, config *alertmanagertypes.Config) error {
	return m.Called(ctx, config).Error(0)
}

func (m *MockAlertmanager) GetConfig(ctx context.Context, orgID string) (*alertmanagertypes.Config, error) {
	ret := m.Called(ctx, orgID)
	config, _ := ret.Get(0).(*alertmanagertypes.Config)
	return config, ret.Error(1)
}

func (m *MockAlertmanager) SetDefaultConfig(ctx context.Context, orgID string) error {
	return m.Called(ctx, orgID).Error(0)
}

func (m *MockAlertmanager) SetNotificationConfig(ctx context.Context, orgID valuer.UUID, ruleID string, config *alertmanagertypes.NotificationConfig) error {
	return m.Called(ctx, orgID, ruleID, config).Error(0)
}

func (m *MockAlertmanager) DeleteNotificationConfig(ctx context.Context, orgID valuer.UUID, ruleID string) error {
	return m.Called(ctx, orgID, ruleID).Error(0)
}

func (m *MockAlertmanager) CreateInhibitRules(ctx context.Context, orgID valuer.UUID, rules []config.InhibitRule) error {
	return m.Called(ctx, orgID, rules).Error(0)
}

func (m *MockAlertmanager) DeleteAllInhibitRulesByRuleId(ctx context.Context, orgID valuer.UUID, ruleID string) error {
	return m.Called(ctx, orgID, ruleID).Error(0)
}

func (m *MockAlertmanager) Collect(ctx context.Context, orgID valuer.UUID) (map[string]any, error) {
	ret := m.Called(ctx, orgID)
	stats, _ := ret.Get(0).(map[string]any)
	return stats, ret.Error(1)
}

func (m *MockAlertmanager) Start(ctx context.Context) error { return m.Called(ctx).Error(0) }
func (m *MockAlertmanager) Stop(ctx context.Context) error  { return m.Called(ctx).Error(0) }
