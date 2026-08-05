package metricsexplorerstore

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"strings"
	"testing"

	"github.com/ClickHouse/clickhouse-go/v2"
	"github.com/ClickHouse/clickhouse-go/v2/lib/driver"
	cmock "github.com/srikanthccv/ClickHouse-go-mock"
	"github.com/stretchr/testify/require"

	"github.com/SigNoz/signoz/pkg/query-service/model"
	"github.com/SigNoz/signoz/pkg/query-service/model/metrics_explorer"
	"github.com/SigNoz/signoz/pkg/query-service/model/querytypes"
	"github.com/SigNoz/signoz/pkg/valuer"
)

type aggregateQueryConn struct {
	clickhouse.Conn
	query func(context.Context, string, ...any) (driver.Rows, error)
}

func TestGetAllMetricFilterAttributeValuesPreservesQueryError(t *testing.T) {
	expected := errors.New("clickhouse unavailable")
	reader := New(slog.New(slog.NewTextHandler(io.Discard, nil)), aggregateQueryConn{
		query: func(context.Context, string, ...any) (driver.Rows, error) {
			return nil, expected
		},
	}, nil)

	_, err := reader.GetAllMetricFilterAttributeValues(context.Background(), &metrics_explorer.FilterValueRequest{})

	require.ErrorIs(t, err, expected)
	require.ErrorContains(t, err, "query metric filter attribute values")
}

func TestGetNameSimilarityPreservesScanError(t *testing.T) {
	reader := New(slog.New(slog.NewTextHandler(io.Discard, nil)), aggregateQueryConn{
		query: func(context.Context, string, ...any) (driver.Rows, error) {
			return cmock.NewRows([]cmock.ColumnType{{Name: "metric_name", Type: "String"}}, [][]any{{nil}}), nil
		},
	}, nil)

	_, err := reader.GetNameSimilarity(context.Background(), &metrics_explorer.RelatedMetricsRequest{})

	require.Error(t, err)
	require.ErrorContains(t, err, "scan metric name similarity")
}

func (c aggregateQueryConn) Query(ctx context.Context, query string, args ...any) (driver.Rows, error) {
	return c.query(ctx, query, args...)
}

type aggregateMetadataReader struct {
	requested []string
	metadata  map[string]*model.MetricMetadata
}

func (r *aggregateMetadataReader) GetMetricsMetadata(_ context.Context, _ valuer.UUID, metricNames ...string) (map[string]*model.MetricMetadata, error) {
	r.requested = append(r.requested, metricNames...)
	return r.metadata, nil
}

func TestGetMetricAggregateAttributesUsesMetadataBoundary(t *testing.T) {
	metadata := &aggregateMetadataReader{metadata: map[string]*model.MetricMetadata{
		"request.count": {
			MetricName:  "request.count",
			MetricType:  querytypes.MetricTypeSum,
			Temporality: querytypes.Cumulative,
			IsMonotonic: false,
		},
	}}
	reader := New(slog.New(slog.NewTextHandler(io.Discard, nil)), aggregateQueryConn{
		query: func(_ context.Context, _ string, _ ...any) (driver.Rows, error) {
			return cmock.NewRows([]cmock.ColumnType{{Name: "metric_name", Type: "String"}}, [][]any{{"request.count"}, {"signoz.internal"}}), nil
		},
	}, metadata)

	response, err := reader.GetMetricAggregateAttributes(
		context.Background(),
		valuer.GenerateUUID(),
		&querytypes.AggregateAttributeRequest{SearchText: "request"},
		true,
	)
	require.NoError(t, err)
	require.Equal(t, []string{"request.count"}, metadata.requested)
	require.Equal(t, []querytypes.AttributeKey{{
		Key:      "request.count",
		DataType: querytypes.AttributeKeyDataTypeFloat64,
		Type:     querytypes.AttributeKeyType(querytypes.MetricTypeGauge),
		IsColumn: true,
	}}, response.AttributeKeys)
}

func TestGetMetricAggregateAttributesRejectsMissingMetadata(t *testing.T) {
	reader := New(slog.New(slog.NewTextHandler(io.Discard, nil)), aggregateQueryConn{
		query: func(_ context.Context, _ string, _ ...any) (driver.Rows, error) {
			return cmock.NewRows([]cmock.ColumnType{{Name: "metric_name", Type: "String"}}, [][]any{{"request.count"}}), nil
		},
	}, &aggregateMetadataReader{metadata: map[string]*model.MetricMetadata{}})

	response, err := reader.GetMetricAggregateAttributes(
		context.Background(),
		valuer.GenerateUUID(),
		&querytypes.AggregateAttributeRequest{},
		false,
	)
	require.ErrorContains(t, err, "metric metadata not found: request.count")
	require.Empty(t, response.AttributeKeys)
}

func TestBuildMetricFilterConditionsBindsKeysAndValues(t *testing.T) {
	maliciousKey := "service.name') OR 1=1 --"
	maliciousValue := "prod'); DROP TABLE metrics --"
	conditions, args, err := buildMetricFilterConditions(&querytypes.FilterSet{Items: []querytypes.FilterItem{
		{
			Key:      querytypes.AttributeKey{Key: maliciousKey},
			Operator: querytypes.FilterOperatorEqual,
			Value:    maliciousValue,
		},
	}}, "")

	require.NoError(t, err)
	require.Equal(t, []string{"JSONExtractString(labels, ?) = ?"}, conditions)
	require.Equal(t, []any{maliciousKey, maliciousValue}, args)
	require.NotContains(t, strings.Join(conditions, " "), maliciousKey)
	require.NotContains(t, strings.Join(conditions, " "), maliciousValue)
}

func TestGetAttributeSimilarityBindsPriorityFilters(t *testing.T) {
	maliciousKey := "service.name') OR 1=1 --"
	maliciousValue := "prod'); DROP TABLE metrics --"
	queryCount := 0
	reader := New(slog.New(slog.NewTextHandler(io.Discard, nil)), aggregateQueryConn{
		query: func(_ context.Context, query string, args ...any) (driver.Rows, error) {
			queryCount++
			if queryCount == 1 {
				return cmock.NewRows([]cmock.ColumnType{
					{Name: "label_key", Type: "String"},
					{Name: "label_values", Type: "Array(String)"},
				}, nil), nil
			}

			require.NotContains(t, query, maliciousKey)
			require.NotContains(t, query, maliciousValue)
			require.Contains(t, query, "arrayDistinct(?) AS filter_keys")
			require.Len(t, args, 7)
			priorityPairs, ok := args[2].(clickhouse.ArraySet)
			require.True(t, ok)
			require.Equal(t, clickhouse.ArraySet{
				clickhouse.GroupSet{Value: []any{maliciousKey, maliciousValue}},
			}, priorityPairs)

			return cmock.NewRows([]cmock.ColumnType{
				{Name: "metric_name", Type: "String"},
				{Name: "type", Type: "String"},
				{Name: "temporality", Type: "String"},
				{Name: "monotonic", Type: "Bool"},
				{Name: "raw_match_count", Type: "UInt64"},
				{Name: "weighted_match_count", Type: "UInt64"},
				{Name: "priority_pairs", Type: "String"},
			}, nil), nil
		},
	}, nil)

	_, err := reader.GetAttributeSimilarity(context.Background(), &metrics_explorer.RelatedMetricsRequest{
		CurrentMetricName: "request.count",
		Filters: querytypes.FilterSet{Items: []querytypes.FilterItem{{
			Key:      querytypes.AttributeKey{Key: maliciousKey},
			Operator: querytypes.FilterOperatorEqual,
			Value:    maliciousValue,
		}}},
	})

	require.NoError(t, err)
	require.Equal(t, 2, queryCount)
}

func TestGetInspectMetricsFingerprintsBindsAttributesAndFilters(t *testing.T) {
	maliciousAttribute := "region') OR 1=1 --"
	maliciousValue := "prod'); DROP TABLE metrics --"
	reader := New(slog.New(slog.NewTextHandler(io.Discard, nil)), aggregateQueryConn{
		query: func(_ context.Context, query string, args ...any) (driver.Rows, error) {
			require.NotContains(t, query, maliciousAttribute)
			require.NotContains(t, query, maliciousValue)
			require.Contains(t, query, "JSONExtractString(labels, ?) AS key1")
			require.Equal(t, maliciousAttribute, args[0])
			require.Equal(t, maliciousAttribute, args[4])
			require.Equal(t, maliciousValue, args[5])
			return cmock.NewRows([]cmock.ColumnType{{Name: "fingerprints", Type: "Array(String)"}}, nil), nil
		},
	}, nil)

	result, err := reader.GetInspectMetricsFingerprints(context.Background(), []string{maliciousAttribute}, &metrics_explorer.InspectMetricsRequest{
		MetricName: "request.count",
		Filters: querytypes.FilterSet{Items: []querytypes.FilterItem{{
			Key:      querytypes.AttributeKey{Key: maliciousAttribute},
			Operator: querytypes.FilterOperatorEqual,
			Value:    maliciousValue,
		}}},
	})

	require.NoError(t, err)
	require.Empty(t, result)
}
