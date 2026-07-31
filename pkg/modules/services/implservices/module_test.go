package implservices

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/ClickHouse/clickhouse-go/v2"
	"github.com/ClickHouse/clickhouse-go/v2/lib/driver"
	cmock "github.com/srikanthccv/ClickHouse-go-mock"
	"github.com/stretchr/testify/require"

	"github.com/SigNoz/signoz/pkg/telemetrystore"
	"github.com/SigNoz/signoz/pkg/types/servicetypes/servicetypesv1"
	"github.com/SigNoz/signoz/pkg/valuer"
)

type serviceQueryConn struct {
	clickhouse.Conn
	query func(context.Context, string, ...any) (driver.Rows, error)
}

func (c serviceQueryConn) Query(ctx context.Context, query string, args ...any) (driver.Rows, error) {
	return c.query(ctx, query, args...)
}

type serviceTelemetryStore struct{ db clickhouse.Conn }

var _ telemetrystore.TelemetryStore = serviceTelemetryStore{}

func (s serviceTelemetryStore) ClickhouseDB() clickhouse.Conn { return s.db }

func TestGetUsesDedicatedAggregateQuery(t *testing.T) {
	queryCount := 0
	m := &module{TelemetryStore: serviceTelemetryStore{db: serviceQueryConn{query: func(_ context.Context, query string, args ...any) (driver.Rows, error) {
		queryCount++
		if queryCount == 1 {
			require.Contains(t, query, "FROM signoz_traces.signoz_index_v3")
			require.Contains(t, query, "quantile(0.99)(duration_nano)")
			require.Contains(t, query, "parent_span_id = ''")
			require.Contains(t, query, "mapContains(resources_string, @filter_0_key)")
			require.NotContains(t, query, "production')")
			require.GreaterOrEqual(t, len(args), 7)
			return cmock.NewRows([]cmock.ColumnType{
				{Name: "service_name", Type: "String"}, {Name: "p99", Type: "Float64"}, {Name: "avg_duration", Type: "Float64"},
				{Name: "num_calls", Type: "UInt64"}, {Name: "num_errors", Type: "UInt64"}, {Name: "num_4xx", Type: "UInt64"},
			}, [][]any{{"checkout", 120.0, 50.0, uint64(10), uint64(2), uint64(1)}}), nil
		}
		require.Contains(t, query, "FROM signoz_traces.top_level_operations")
		return cmock.NewRows([]cmock.ColumnType{{Name: "name", Type: "String"}, {Name: "serviceName", Type: "String"}, {Name: "ts", Type: "DateTime"}}, [][]any{{"POST /orders", "checkout", time.Unix(1, 0)}}), nil
	}}}}

	items, err := m.Get(context.Background(), valuer.GenerateUUID(), &servicetypesv1.Request{
		Start: "1000000000", End: "11000000000",
		Tags: []servicetypesv1.TagFilterItem{{Key: "deployment.environment", Operator: "in", StringValues: []string{"production') OR 1=1 --"}, TagType: "ResourceAttribute"}},
	})
	require.NoError(t, err)
	require.Equal(t, 2, queryCount)
	require.Len(t, items, 1)
	require.Equal(t, "checkout", items[0].ServiceName)
	require.Equal(t, 1.0, items[0].CallRate)
	require.Equal(t, 20.0, items[0].ErrorRate)
	require.Equal(t, []string{"overflow_operation", "POST /orders"}, items[0].DataWarning.TopLevelOps)
}

func TestGetOperationsPreservesQueryFailure(t *testing.T) {
	expected := errors.New("clickhouse unavailable")
	m := &module{TelemetryStore: serviceTelemetryStore{db: serviceQueryConn{query: func(context.Context, string, ...any) (driver.Rows, error) {
		return nil, expected
	}}}}

	_, err := m.GetTopOperations(context.Background(), valuer.GenerateUUID(), &servicetypesv1.OperationsRequest{Start: "1", End: "2", Service: "api", Limit: 10})
	require.ErrorContains(t, err, expected.Error())
}

func TestGetEntryPointOperationsBuildsScopeQuery(t *testing.T) {
	m := &module{TelemetryStore: serviceTelemetryStore{db: serviceQueryConn{query: func(_ context.Context, query string, args ...any) (driver.Rows, error) {
		require.Contains(t, query, "GLOBAL IN")
		require.Contains(t, query, "parent_span_id != ''")
		require.Contains(t, query, "resource_string_service$$name = @service_name")
		require.Contains(t, query, "mapContains(attributes_number, @filter_0_key)")
		require.NotContains(t, query, "malicious")
		require.GreaterOrEqual(t, len(args), 8)
		return cmock.NewRows([]cmock.ColumnType{
			{Name: "name", Type: "String"}, {Name: "p50", Type: "Float64"}, {Name: "p95", Type: "Float64"}, {Name: "p99", Type: "Float64"}, {Name: "num_calls", Type: "UInt64"}, {Name: "error_count", Type: "UInt64"},
		}, [][]any{{"GET /orders", 10.0, 20.0, 30.0, uint64(4), uint64(1)}}), nil
	}}}}

	items, err := m.GetEntryPointOperations(context.Background(), valuer.GenerateUUID(), &servicetypesv1.OperationsRequest{
		Start: "1000000000", End: "2000000000", Service: "api", Limit: 5,
		Tags: []servicetypesv1.TagFilterItem{{Key: "http.status_code", Operator: "notin", NumberValues: []float64{500}, TagType: "SpanAttribute"}},
	})
	require.NoError(t, err)
	require.Equal(t, []servicetypesv1.OperationItem{{Name: "GET /orders", P50: 10, P95: 20, P99: 30, NumCalls: 4, ErrorCount: 1}}, items)
}

func TestParseServiceTimeRangeRejectsInvalidValues(t *testing.T) {
	for _, testCase := range []struct{ start, end, want string }{
		{"x", "2", "invalid start time"},
		{"1", "x", "invalid end time"},
		{"2", "2", "start must be before end"},
		{"9223372036854775808", "9223372036854775809", "invalid start time"},
	} {
		_, err := parseServiceTimeRange(testCase.start, testCase.end)
		require.ErrorContains(t, err, testCase.want)
	}
}

func TestBuildTagConditionsBindsTagKeysAndValues(t *testing.T) {
	maliciousKey := "service.name'] OR 1=1 --"
	maliciousValue := "prod') OR 1=1 --"
	conditions, args, err := buildTagConditions([]servicetypesv1.TagFilterItem{{
		Key: maliciousKey, Operator: "in", StringValues: []string{maliciousValue}, TagType: "ResourceAttribute",
	}})
	require.NoError(t, err)
	require.NotContains(t, conditions, maliciousKey)
	require.NotContains(t, conditions, maliciousValue)
	require.Contains(t, conditions, "mapContains(resources_string, @filter_0_key)")
	require.Len(t, args, 2)
}

func TestBuildTagConditionsRejectsUnknownTagType(t *testing.T) {
	_, _, err := buildTagConditions([]servicetypesv1.TagFilterItem{{Key: "env", Operator: "in", StringValues: []string{"prod"}, TagType: "scope"}})
	require.ErrorContains(t, err, "unsupported tag type")
}

func TestApplyOpsToItems(t *testing.T) {
	items := []*servicetypesv1.ResponseItem{{ServiceName: "api", DataWarning: servicetypesv1.DataWarning{TopLevelOps: []string{}}}}
	applyOpsToItems(items, map[string][]string{"api": {"overflow_operation", "GET /orders"}})
	require.Equal(t, []string{"overflow_operation", "GET /orders"}, items[0].DataWarning.TopLevelOps)
	require.False(t, strings.Contains(items[0].ServiceName, " "))
}
