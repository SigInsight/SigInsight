package implfields

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/SigNoz/signoz/pkg/instrumentation/instrumentationtest"
	"github.com/SigNoz/signoz/pkg/types/telemetrytypes"
	"github.com/SigNoz/signoz/pkg/types/telemetrytypes/telemetrytypestest"
	"github.com/stretchr/testify/require"
)

func TestGetFieldsValuesIncludesBoolValues(t *testing.T) {
	metadata := telemetrytypestest.NewMockMetadataStore()
	metadata.AllValuesMap["has_error-traces"] = &telemetrytypes.TelemetryFieldValues{
		BoolValues: []bool{true, false},
	}
	handler := NewHandler(instrumentationtest.New().ToProviderSettings(), metadata)
	req := httptest.NewRequest(http.MethodGet, "/api/v5/fields/values?signal=traces&name=has_error", nil)
	rr := httptest.NewRecorder()

	handler.GetFieldsValues(rr, req)

	require.Equal(t, http.StatusOK, rr.Code)
	require.JSONEq(t, `{
		"status":"success",
		"data":{
			"values":{"boolValues":[true,false]},
			"complete":true
		}
	}`, rr.Body.String())
}
