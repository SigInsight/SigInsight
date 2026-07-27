package alertmanager

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/SigNoz/signoz/pkg/alertmanager/alertmanagertest"
	"github.com/SigNoz/signoz/pkg/types/alertmanagertypes"
	"github.com/SigNoz/signoz/pkg/types/authtypes"
	"github.com/go-openapi/strfmt"
	"github.com/prometheus/alertmanager/api/v2/models"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func TestGetAlertsReturnsCurrentAlertmanagerModel(t *testing.T) {
	alertmanager := alertmanagertest.NewMockAlertmanager(t)
	api := NewAPI(alertmanager)

	orgID := "01982f1a-18f0-7000-8000-000000000001"
	receiverName := "primary"
	fingerprint := "abc123"
	state := "active"
	now := strfmt.DateTime(time.Date(2026, time.July, 24, 10, 0, 0, 0, time.UTC))

	alertmanager.EXPECT().
		GetAlerts(mock.Anything, orgID, mock.Anything).
		Return(alertmanagertypes.GettableAlerts{
			{
				Alert: models.Alert{
					Labels:       models.LabelSet{"alertname": "HighLatency"},
					GeneratorURL: strfmt.URI("http://localhost/alerts/1"),
				},
				Annotations: models.LabelSet{"summary": "latency is high"},
				StartsAt:    &now,
				EndsAt:      &now,
				UpdatedAt:   &now,
				Fingerprint: &fingerprint,
				Receivers:   []*models.Receiver{{Name: &receiverName}},
				Status: &models.AlertStatus{
					InhibitedBy: []string{},
					MutedBy:     []string{},
					SilencedBy:  []string{},
					State:       &state,
				},
			},
		}, nil)

	req := httptest.NewRequest(http.MethodGet, "/alerts", nil)
	req = req.WithContext(authtypes.NewContextWithClaims(req.Context(), authtypes.Claims{OrgID: orgID}))
	recorder := httptest.NewRecorder()

	api.GetAlerts(recorder, req)

	require.Equal(t, http.StatusOK, recorder.Code)
	var response struct {
		Data []struct {
			Labels map[string]string `json:"labels"`
			Status struct {
				State string `json:"state"`
			} `json:"status"`
			Receivers []struct {
				Name string `json:"name"`
			} `json:"receivers"`
			UpdatedAt   string `json:"updatedAt"`
			Fingerprint string `json:"fingerprint"`
		} `json:"data"`
	}
	require.NoError(t, json.Unmarshal(recorder.Body.Bytes(), &response))
	require.Len(t, response.Data, 1)
	require.Equal(t, "HighLatency", response.Data[0].Labels["alertname"])
	require.Equal(t, "active", response.Data[0].Status.State)
	require.Equal(t, "primary", response.Data[0].Receivers[0].Name)
	require.NotEmpty(t, response.Data[0].UpdatedAt)
	require.Equal(t, fingerprint, response.Data[0].Fingerprint)
}
