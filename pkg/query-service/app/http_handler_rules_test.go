package app

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/SigNoz/signoz/pkg/query-service/model"
	"github.com/SigNoz/signoz/pkg/query-service/rules"
	"github.com/gorilla/mux"
	"github.com/stretchr/testify/require"
)

func TestRuleStateHistoryEndpointsRejectInvalidRuleID(t *testing.T) {
	handler := &APIHandler{}
	tests := []struct {
		name    string
		handler http.HandlerFunc
	}{
		{name: "stats", handler: handler.getRuleStats},
		{name: "overall status", handler: handler.getOverallStateTransitions},
		{name: "timeline", handler: handler.getRuleStateHistory},
		{name: "top contributors", handler: handler.getRuleStateHistoryTopContributors},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			request := httptest.NewRequest(http.MethodPost, "/api/v5/rules/not-a-uuid/history", strings.NewReader("{}"))
			request = mux.SetURLVars(request, map[string]string{"id": "not-a-uuid"})
			recorder := httptest.NewRecorder()

			tt.handler(recorder, request)

			require.Equal(t, http.StatusBadRequest, recorder.Code)
			require.Contains(t, recorder.Body.String(), `"status":"error"`)
			require.Contains(t, recorder.Body.String(), `"errorType":"bad_data"`)
		})
	}
}

func TestTestRuleResponseUsesStructuredEvaluationPreview(t *testing.T) {
	response := testRuleResponse{
		EvaluationPreview: rules.EvaluationPreview{
			AlertCount:  2,
			State:       model.StateFiring,
			EvaluatedAt: 1780000000000,
		},
		Message: "notification sent",
	}

	encoded, err := json.Marshal(response)
	require.NoError(t, err)
	require.JSONEq(t, `{
		"evaluationPreview": {
			"alertCount": 2,
			"state": "firing",
			"evaluatedAt": 1780000000000
		},
		"message": "notification sent"
	}`, string(encoded))
}
