package app

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/SigNoz/signoz/pkg/query-service/model"
)

type serviceReaderStub struct {
	dependencyErr error
}

func (stub serviceReaderStub) GetTopLevelOperations(context.Context, time.Time, time.Time, []string) (*map[string][]string, error) {
	return nil, nil
}

func (stub serviceReaderStub) GetDependencyGraph(context.Context, *model.GetServicesParams) (*[]model.ServiceMapDependencyResponseItem, error) {
	return nil, stub.dependencyErr
}

func TestDependencyGraphReportsStorageFailureAsInternalError(t *testing.T) {
	handler := &APIHandler{
		logger:   slog.New(slog.NewTextHandler(io.Discard, nil)),
		services: serviceReaderStub{dependencyErr: errors.New("ClickHouse unavailable")},
	}
	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(
		http.MethodPost,
		"/api/v5/services/dependency_graph",
		strings.NewReader(`{"start":"100000000000","end":"200000000000"}`),
	)

	handler.dependencyGraph(recorder, request)

	require.Equal(t, http.StatusInternalServerError, recorder.Code)
	require.Contains(t, recorder.Body.String(), "ClickHouse unavailable")
}
