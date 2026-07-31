package implspanpercentile

import (
	"context"
	"errors"
	"testing"

	"github.com/ClickHouse/clickhouse-go/v2"
	"github.com/ClickHouse/clickhouse-go/v2/lib/driver"
	cmock "github.com/srikanthccv/ClickHouse-go-mock"
	"github.com/stretchr/testify/require"

	"github.com/SigNoz/signoz/pkg/telemetrystore"
	"github.com/SigNoz/signoz/pkg/types/spanpercentiletypes"
	"github.com/SigNoz/signoz/pkg/valuer"
)

type percentileQueryConn struct {
	clickhouse.Conn
	query func(context.Context, string, ...any) (driver.Rows, error)
}

func (c percentileQueryConn) Query(ctx context.Context, query string, args ...any) (driver.Rows, error) {
	return c.query(ctx, query, args...)
}

type percentileTelemetryStore struct{ db clickhouse.Conn }

var _ telemetrystore.TelemetryStore = percentileTelemetryStore{}

func (s percentileTelemetryStore) ClickhouseDB() clickhouse.Conn { return s.db }

func TestGetSpanPercentileUsesDedicatedBoundQuery(t *testing.T) {
	maliciousName := "GET /orders') OR 1=1 --"
	maliciousAttribute := "environment') OR 1=1 --"
	m := &module{telemetryStore: percentileTelemetryStore{db: percentileQueryConn{query: func(_ context.Context, query string, args ...any) (driver.Rows, error) {
		require.Contains(t, query, "FROM siginsight_traces.span_index_v3")
		require.Contains(t, query, "quantile(0.90)(duration_nano)")
		require.Contains(t, query, "countIf(duration_nano <= @duration_nano)")
		require.Contains(t, query, "mapContains(resources_string, @resource_0_key)")
		require.NotContains(t, query, maliciousName)
		require.NotContains(t, query, maliciousAttribute)
		require.Len(t, args, 9)
		return cmock.NewRows([]cmock.ColumnType{
			{Name: "p50", Type: "Float64"}, {Name: "p90", Type: "Float64"}, {Name: "p99", Type: "Float64"},
			{Name: "percentile_position", Type: "Float64"}, {Name: "span_count", Type: "UInt64"},
		}, [][]any{{10.0, 20.0, 30.0, 25.0, uint64(4)}}), nil
	}}}}

	response, err := m.GetSpanPercentile(context.Background(), valuer.GenerateUUID(), valuer.GenerateUUID(), &spanpercentiletypes.SpanPercentileRequest{
		DurationNano: 15, Name: maliciousName, ServiceName: "checkout",
		ResourceAttributes: map[string]string{maliciousAttribute: "production') OR 1=1 --"},
		Start:              1_700_000_000_000, End: 1_700_000_060_000,
	})
	require.NoError(t, err)
	require.Equal(t, 10.0, response.Percentiles.P50)
	require.Equal(t, 25.0, response.Position.Percentile)
	require.Equal(t, "faster than 75.0% of spans", response.Position.Description)
}

func TestGetSpanPercentileReturnsNotFoundForNoRows(t *testing.T) {
	m := &module{telemetryStore: percentileTelemetryStore{db: percentileQueryConn{query: func(context.Context, string, ...any) (driver.Rows, error) {
		return cmock.NewRows([]cmock.ColumnType{{Name: "p50", Type: "Float64"}}, nil), nil
	}}}}
	_, err := m.GetSpanPercentile(context.Background(), valuer.GenerateUUID(), valuer.GenerateUUID(), validPercentileRequest())
	require.ErrorContains(t, err, "no spans found matching")
}

func TestGetSpanPercentileReturnsNotFoundForZeroCount(t *testing.T) {
	m := &module{telemetryStore: percentileTelemetryStore{db: percentileQueryConn{query: func(context.Context, string, ...any) (driver.Rows, error) {
		return cmock.NewRows([]cmock.ColumnType{
			{Name: "p50", Type: "Float64"}, {Name: "p90", Type: "Float64"}, {Name: "p99", Type: "Float64"}, {Name: "percentile_position", Type: "Float64"}, {Name: "span_count", Type: "UInt64"},
		}, [][]any{{0.0, 0.0, 0.0, 0.0, uint64(0)}}), nil
	}}}}
	_, err := m.GetSpanPercentile(context.Background(), valuer.GenerateUUID(), valuer.GenerateUUID(), validPercentileRequest())
	require.ErrorContains(t, err, "no spans found matching")
}

func TestGetSpanPercentilePreservesQueryFailure(t *testing.T) {
	expected := errors.New("clickhouse unavailable")
	m := &module{telemetryStore: percentileTelemetryStore{db: percentileQueryConn{query: func(context.Context, string, ...any) (driver.Rows, error) {
		return nil, expected
	}}}}
	_, err := m.GetSpanPercentile(context.Background(), valuer.GenerateUUID(), valuer.GenerateUUID(), validPercentileRequest())
	require.ErrorContains(t, err, expected.Error())
}

func TestGetSpanPercentileValidatesNilAndOutOfRangeRequests(t *testing.T) {
	m := &module{}
	_, err := m.GetSpanPercentile(context.Background(), valuer.GenerateUUID(), valuer.GenerateUUID(), nil)
	require.ErrorContains(t, err, "request is nil")

	request := validPercentileRequest()
	request.Start = 1 << 63
	request.End = request.Start + 1
	_, err = m.GetSpanPercentile(context.Background(), valuer.GenerateUUID(), valuer.GenerateUUID(), request)
	require.ErrorContains(t, err, "out of range")
}

func TestResourceAttributeConditionsSortAndBind(t *testing.T) {
	conditions, args := resourceAttributeConditions(map[string]string{"z": "last", "a": "first"})
	require.Equal(t, " AND mapContains(resources_string, @resource_0_key) AND resources_string[@resource_0_key] = @resource_0_value AND mapContains(resources_string, @resource_1_key) AND resources_string[@resource_1_key] = @resource_1_value", conditions)
	require.Len(t, args, 4)
}

func validPercentileRequest() *spanpercentiletypes.SpanPercentileRequest {
	return &spanpercentiletypes.SpanPercentileRequest{DurationNano: 10, Name: "GET /orders", ServiceName: "checkout", Start: 1_700_000_000_000, End: 1_700_000_060_000}
}
