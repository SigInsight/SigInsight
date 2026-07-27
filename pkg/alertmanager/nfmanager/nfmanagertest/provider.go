package nfmanagertest

import (
	"context"

	"github.com/SigNoz/signoz/pkg/types/alertmanagertypes"
	"github.com/prometheus/common/model"
)

// MockNotificationManager is an in-memory NotificationManager for tests.
type MockNotificationManager struct {
	configs map[string]*alertmanagertypes.NotificationConfig
	errors  map[string]error
}

func NewMock() *MockNotificationManager {
	return &MockNotificationManager{
		configs: make(map[string]*alertmanagertypes.NotificationConfig),
		errors:  make(map[string]error),
	}
}

func getKey(orgID string, ruleID string) string {
	return orgID + ":" + ruleID
}

func (m *MockNotificationManager) GetNotificationConfig(orgID string, ruleID string) (*alertmanagertypes.NotificationConfig, error) {
	key := getKey(orgID, ruleID)
	if err := m.errors[key]; err != nil {
		return nil, err
	}
	if config := m.configs[key]; config != nil {
		return config, nil
	}

	config := alertmanagertypes.GetDefaultNotificationConfig()
	return &config, nil
}

func (m *MockNotificationManager) SetNotificationConfig(orgID string, ruleID string, config *alertmanagertypes.NotificationConfig) error {
	key := getKey(orgID, ruleID)
	if err := m.errors[key]; err != nil {
		return err
	}
	m.configs[key] = config
	return nil
}

func (m *MockNotificationManager) DeleteNotificationConfig(orgID string, ruleID string) error {
	key := getKey(orgID, ruleID)
	if err := m.errors[key]; err != nil {
		return err
	}
	delete(m.configs, key)
	return nil
}

func (m *MockNotificationManager) Match(_ context.Context, orgID string, ruleID string, _ model.LabelSet) ([]string, error) {
	config, err := m.GetNotificationConfig(orgID, ruleID)
	if err != nil {
		return nil, err
	}
	return config.Channels, nil
}

func (m *MockNotificationManager) SetMockConfig(orgID string, ruleID string, config *alertmanagertypes.NotificationConfig) {
	m.configs[getKey(orgID, ruleID)] = config
}

func (m *MockNotificationManager) SetMockError(orgID string, ruleID string, err error) {
	m.errors[getKey(orgID, ruleID)] = err
}

func (m *MockNotificationManager) ClearMockData() {
	m.configs = make(map[string]*alertmanagertypes.NotificationConfig)
	m.errors = make(map[string]error)
}

func (m *MockNotificationManager) HasConfig(orgID string, ruleID string) bool {
	_, exists := m.configs[getKey(orgID, ruleID)]
	return exists
}
