// Package nfmanager provides interfaces and implementations for alert notification grouping strategies.
package nfmanager

import (
	"context"

	"github.com/SigNoz/signoz/pkg/types/alertmanagertypes"
	"github.com/prometheus/common/model"
)

// NotificationManager defines how alerts should be grouped and configured for notification.
type NotificationManager interface {
	// Notification Config CRUD
	GetNotificationConfig(orgID string, ruleID string) (*alertmanagertypes.NotificationConfig, error)
	SetNotificationConfig(orgID string, ruleID string, config *alertmanagertypes.NotificationConfig) error
	DeleteNotificationConfig(orgID string, ruleID string) error

	// Route matching
	Match(ctx context.Context, orgID string, ruleID string, set model.LabelSet) ([]string, error)
}
