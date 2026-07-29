package servicestore

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"strings"
	"testing"
	"time"

	"github.com/ClickHouse/clickhouse-go/v2"
	"github.com/ClickHouse/clickhouse-go/v2/lib/driver"
	cmock "github.com/srikanthccv/ClickHouse-go-mock"
	"github.com/stretchr/testify/require"

	"github.com/SigNoz/signoz/pkg/query-service/model"
)

type queryConn struct {
	clickhouse.Conn
	query func(context.Context, string, ...any) (driver.Rows, error)
}

func (c queryConn) Query(ctx context.Context, query string, args ...any) (driver.Rows, error) {
	return c.query(ctx, query, args...)
}

type selectConn struct {
	clickhouse.Conn
	selectFn func(context.Context, any, string, ...any) error
}

func (c selectConn) Select(ctx context.Context, dest any, query string, args ...any) error {
	return c.selectFn(ctx, dest, query, args...)
}

func TestGetTopLevelOperationsMapsOperationsByService(t *testing.T) {
	reader := New(slog.New(slog.NewTextHandler(io.Discard, nil)), queryConn{
		query: func(_ context.Context, query string, args ...any) (driver.Rows, error) {
			require.Contains(t, query, "serviceName IN @services")
			require.Len(t, args, 2)
			return cmock.NewRows([]cmock.ColumnType{{Name: "name", Type: "String"}, {Name: "serviceName", Type: "String"}, {Name: "ts", Type: "DateTime"}}, [][]any{{"GET /orders", "api", time.Unix(1, 0)}}), nil
		},
	}, Config{TraceDB: "signoz_traces", TopLevelOperationsTable: "top_level_operations"})

	operations, apiErr := reader.GetTopLevelOperations(context.Background(), time.Unix(0, 0), time.Now(), []string{"api"})
	require.Nil(t, apiErr)
	require.Equal(t, []string{"overflow_operation", "GET /orders"}, (*operations)["api"])
}

func TestGetDependencyGraphMapsSelectedRows(t *testing.T) {
	reader := New(slog.New(slog.NewTextHandler(io.Discard, nil)), selectConn{
		selectFn: func(_ context.Context, dest any, query string, args ...any) error {
			require.Contains(t, query, "FROM signoz_traces.dependency_graph_minutes_v2")
			require.Equal(t, 3, len(args))
			response := dest.(*[]model.ServiceMapDependencyResponseItem)
			*response = []model.ServiceMapDependencyResponseItem{{Parent: "frontend", Child: "api", CallCount: 4}}
			return nil
		},
	}, Config{TraceDB: "signoz_traces", DependencyGraphTable: "dependency_graph_minutes_v2"})

	start := time.Unix(100, 0)
	end := time.Unix(160, 0)
	response, err := reader.GetDependencyGraph(context.Background(), &model.GetServicesParams{Start: &start, End: &end})
	require.NoError(t, err)
	require.Equal(t, "frontend", (*response)[0].Parent)
	require.Equal(t, "api", (*response)[0].Child)
	require.False(t, strings.Contains((*response)[0].Parent, " "))
}

func TestGetDependencyGraphPreservesQueryError(t *testing.T) {
	expected := errors.New("clickhouse unavailable")
	reader := New(slog.New(slog.NewTextHandler(io.Discard, nil)), selectConn{
		selectFn: func(context.Context, any, string, ...any) error {
			return expected
		},
	}, Config{})
	start := time.Unix(100, 0)
	end := time.Unix(160, 0)

	_, err := reader.GetDependencyGraph(context.Background(), &model.GetServicesParams{Start: &start, End: &end})

	require.ErrorIs(t, err, expected)
	require.ErrorContains(t, err, "error in processing sql query")
}

func TestGetTopLevelOperationsPreservesQueryError(t *testing.T) {
	expected := errors.New("clickhouse unavailable")
	reader := New(slog.New(slog.NewTextHandler(io.Discard, nil)), queryConn{
		query: func(context.Context, string, ...any) (driver.Rows, error) {
			return nil, expected
		},
	}, Config{})

	_, err := reader.GetTopLevelOperations(context.Background(), time.Now(), time.Now(), nil)
	require.ErrorIs(t, err, expected)
	require.ErrorContains(t, err, "query top-level operations")
}
