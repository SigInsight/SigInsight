package metricmetadatastore

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

	"github.com/SigNoz/signoz/pkg/query-service/constants"
	"github.com/SigNoz/signoz/pkg/query-service/model"
	"github.com/SigNoz/signoz/pkg/query-service/model/querytypes"
	"github.com/SigNoz/signoz/pkg/types/cachetypes"
	"github.com/SigNoz/signoz/pkg/valuer"
)

type queryOnlyConn struct {
	clickhouse.Conn
	query func(context.Context, string, ...any) (driver.Rows, error)
}

func (c queryOnlyConn) Query(ctx context.Context, query string, args ...any) (driver.Rows, error) {
	return c.query(ctx, query, args...)
}

type rowsWithError struct {
	driver.Rows
	err error
}

func (r rowsWithError) Err() error { return r.err }

var metricMetadataColumns = []cmock.ColumnType{
	{Name: "metric_name", Type: "String"},
	{Name: "type", Type: "String"},
	{Name: "description", Type: "String"},
	{Name: "temporality", Type: "String"},
	{Name: "is_monotonic", Type: "Bool"},
	{Name: "unit", Type: "String"},
}

type fakeCache struct {
	data map[string][]byte
}

func (c *fakeCache) Set(_ context.Context, _ valuer.UUID, key string, value cachetypes.Cacheable, _ time.Duration) error {
	data, err := value.MarshalBinary()
	if err != nil {
		return err
	}
	c.data[key] = data
	return nil
}

func (c *fakeCache) Get(_ context.Context, _ valuer.UUID, key string, dest cachetypes.Cacheable) error {
	data, ok := c.data[key]
	if !ok {
		return errors.New("cache miss")
	}
	return dest.UnmarshalBinary(data)
}

func (c *fakeCache) Delete(_ context.Context, _ valuer.UUID, key string) {
	delete(c.data, key)
}

func (c *fakeCache) DeleteMany(_ context.Context, _ valuer.UUID, keys []string) {
	for _, key := range keys {
		delete(c.data, key)
	}
}

func TestGetMetricMetadataUsesSharedMetadataCache(t *testing.T) {
	ctx := context.Background()
	orgID := valuer.GenerateUUID()
	cache := &fakeCache{data: map[string][]byte{}}
	metadata := &model.UpdateMetricsMetadata{
		MetricName:  "request.count",
		MetricType:  querytypes.MetricTypeGauge,
		Description: "Requests handled",
		Unit:        "requests",
		Temporality: querytypes.Cumulative,
		IsMonotonic: true,
	}
	require.NoError(t, cache.Set(
		ctx,
		orgID,
		constants.UpdatedMetricsMetadataCachePrefix+metadata.MetricName,
		metadata,
		0,
	))

	reader := New(slog.New(slog.NewTextHandler(io.Discard, nil)), nil, cache)
	result, err := reader.GetMetricMetadata(ctx, orgID, metadata.MetricName, "api")

	require.NoError(t, err)
	require.Equal(t, string(querytypes.MetricTypeGauge), result.Type)
	require.Equal(t, metadata.Description, result.Description)
	require.Equal(t, metadata.Unit, result.Unit)
	require.Equal(t, string(querytypes.Cumulative), result.Temporality)
	require.True(t, result.IsMonotonic)
	require.False(t, result.Delta)
	require.Empty(t, result.Le)
}

func TestGetUpdatedMetricsMetadataCachesFallbackResult(t *testing.T) {
	ctx := context.Background()
	orgID := valuer.GenerateUUID()
	cache := &fakeCache{data: map[string][]byte{}}
	queries := 0
	reader := New(slog.New(slog.NewTextHandler(io.Discard, nil)), queryOnlyConn{
		query: func(_ context.Context, query string, _ ...any) (driver.Rows, error) {
			queries++
			switch {
			case strings.Contains(query, siginsightUpdatedMetricsMetadataTable):
				return cmock.NewRows(metricMetadataColumns, nil), nil
			case strings.Contains(query, siginsightTSTableNameV4):
				return cmock.NewRows(metricMetadataColumns, [][]any{{"request.count", "Gauge", "Requests", "Cumulative", true, "requests"}}), nil
			default:
				t.Fatalf("unexpected query: %s", query)
				return nil, nil
			}
		},
	}, cache)

	metadata, apiErr := reader.GetUpdatedMetricsMetadata(ctx, orgID, "request.count")
	require.Nil(t, apiErr)
	require.Equal(t, querytypes.MetricTypeGauge, metadata["request.count"].MetricType)
	require.Equal(t, 2, queries)

	metadata, apiErr = reader.GetUpdatedMetricsMetadata(ctx, orgID, "request.count")
	require.Nil(t, apiErr)
	require.Equal(t, "Requests", metadata["request.count"].Description)
	require.Equal(t, 2, queries, "cached fallback metadata must avoid another ClickHouse query")
}

func TestGetUpdatedMetricsMetadataDoesNotFallbackWhenMetadataTableHasResult(t *testing.T) {
	ctx := context.Background()
	orgID := valuer.GenerateUUID()
	cache := &fakeCache{data: map[string][]byte{}}
	reader := New(slog.New(slog.NewTextHandler(io.Discard, nil)), queryOnlyConn{
		query: func(_ context.Context, query string, _ ...any) (driver.Rows, error) {
			require.Contains(t, query, siginsightUpdatedMetricsMetadataTable)
			return cmock.NewRows(metricMetadataColumns, [][]any{{"system.cpu", "Gauge", "CPU", "Cumulative", false, "percent"}}), nil
		},
	}, cache)

	metadata, apiErr := reader.GetUpdatedMetricsMetadata(ctx, orgID, "system.cpu")
	require.Nil(t, apiErr)
	require.Equal(t, "CPU", metadata["system.cpu"].Description)
}

func TestGetUpdatedMetricsMetadataBindsMetricNames(t *testing.T) {
	ctx := context.Background()
	orgID := valuer.GenerateUUID()
	cache := &fakeCache{data: map[string][]byte{}}
	metricName := "request'count"
	queries := 0
	reader := New(slog.New(slog.NewTextHandler(io.Discard, nil)), queryOnlyConn{
		query: func(_ context.Context, query string, args ...any) (driver.Rows, error) {
			queries++
			require.Contains(t, query, "IN ({metric_names:Array(String)})")
			require.NotContains(t, query, metricName)
			require.Len(t, args, 1)
			named, ok := args[0].(driver.NamedValue)
			require.True(t, ok)
			require.Equal(t, "metric_names", named.Name)
			require.Equal(t, []string{metricName}, named.Value)

			if strings.Contains(query, siginsightUpdatedMetricsMetadataTable) {
				return cmock.NewRows(metricMetadataColumns, nil), nil
			}
			return cmock.NewRows(metricMetadataColumns, [][]any{{metricName, "Gauge", "Requests", "Cumulative", true, "requests"}}), nil
		},
	}, cache)

	metadata, apiErr := reader.GetUpdatedMetricsMetadata(ctx, orgID, metricName)
	require.Nil(t, apiErr)
	require.Equal(t, 2, queries)
	require.Equal(t, metricName, metadata[metricName].MetricName)
}

func TestGetUpdatedMetricsMetadataReturnsFallbackRowsError(t *testing.T) {
	ctx := context.Background()
	orgID := valuer.GenerateUUID()
	cache := &fakeCache{data: map[string][]byte{}}
	fallbackErr := errors.New("fallback stream interrupted")
	reader := New(slog.New(slog.NewTextHandler(io.Discard, nil)), queryOnlyConn{
		query: func(_ context.Context, query string, _ ...any) (driver.Rows, error) {
			if strings.Contains(query, siginsightUpdatedMetricsMetadataTable) {
				return cmock.NewRows(metricMetadataColumns, nil), nil
			}
			return rowsWithError{Rows: cmock.NewRows(metricMetadataColumns, nil), err: fallbackErr}, nil
		},
	}, cache)

	metadata, err := reader.GetUpdatedMetricsMetadata(ctx, orgID, "request.count")
	require.Empty(t, metadata)
	require.ErrorIs(t, err, fallbackErr)
	require.ErrorContains(t, err, "error scanning fallback metadata")
}
