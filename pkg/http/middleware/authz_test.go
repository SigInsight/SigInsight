package middleware

import (
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/SigNoz/signoz/pkg/types"
	"github.com/SigNoz/signoz/pkg/types/authtypes"
	"github.com/stretchr/testify/require"
)

func TestEditAccessAPIKeyRoles(t *testing.T) {
	tests := []struct {
		name           string
		role           types.Role
		expectedStatus int
		expectedCall   bool
	}{
		{name: "viewer is denied", role: types.RoleViewer, expectedStatus: http.StatusForbidden},
		{name: "editor is allowed", role: types.RoleEditor, expectedStatus: http.StatusNoContent, expectedCall: true},
		{name: "admin is allowed", role: types.RoleAdmin, expectedStatus: http.StatusNoContent, expectedCall: true},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			middleware := NewAuthZ(slog.New(slog.DiscardHandler), nil, nil)
			handlerCalled := false
			handler := middleware.EditAccess(func(rw http.ResponseWriter, _ *http.Request) {
				handlerCalled = true
				rw.WriteHeader(http.StatusNoContent)
			})

			request := httptest.NewRequest(http.MethodPost, "/api/v1/integrations/install", nil)
			request = request.WithContext(authtypes.NewContextWithClaims(request.Context(), authtypes.Claims{
				Role:           test.role,
				IdentNProvider: authtypes.IdentNProviderAPIKey.StringValue(),
			}))
			response := httptest.NewRecorder()

			handler.ServeHTTP(response, request)

			require.Equal(t, test.expectedStatus, response.Code)
			require.Equal(t, test.expectedCall, handlerCalled)
		})
	}
}
