package app

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/SigNoz/signoz/pkg/types/timeseriestypes"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type traceFunnelQueryReaderStub struct {
	query string
	rows  []*timeseriestypes.Row
	err   error
}

func (stub *traceFunnelQueryReaderStub) ExecuteTraceFunnelQuery(_ context.Context, query string) ([]*timeseriestypes.Row, error) {
	stub.query = query
	return stub.rows, stub.err
}

func TestRespondTraceFunnelQuery(t *testing.T) {
	reader := &traceFunnelQueryReaderStub{
		rows: []*timeseriestypes.Row{{Data: map[string]interface{}{"count": uint64(3)}}},
	}
	handler := &APIHandler{traceFunnelQuery: reader}
	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodPost, "/api/v5/trace-funnels/analytics/overview", nil)

	handler.respondTraceFunnelQuery(recorder, request, "SELECT 3")

	require.Equal(t, http.StatusOK, recorder.Code)
	assert.Equal(t, "SELECT 3", reader.query)
	assert.JSONEq(t, `{"status":"success","data":[{"timestamp":"0001-01-01T00:00:00Z","data":{"count":3}}]}`, recorder.Body.String())
}
