package alertmanagernotify

import (
	"log/slog"

	"github.com/SigNoz/signoz/pkg/types/alertmanagertypes"
	"github.com/prometheus/alertmanager/config/receiver"
	"github.com/prometheus/alertmanager/notify"
	"github.com/prometheus/alertmanager/template"
)

func NewReceiverIntegrations(nc alertmanagertypes.Receiver, tmpl *template.Template, logger *slog.Logger) ([]notify.Integration, error) {
	upstreamIntegrations, err := receiver.BuildReceiverIntegrations(nc, tmpl, logger)
	if err != nil {
		return nil, err
	}

	integrations := make([]notify.Integration, 0, len(upstreamIntegrations))
	for _, integration := range upstreamIntegrations {
		if integration.Name() == "email" || integration.Name() == "webhook" {
			integrations = append(integrations, integration)
		}
	}

	return integrations, nil
}
