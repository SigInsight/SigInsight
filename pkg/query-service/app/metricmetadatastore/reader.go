package metricmetadatastore

import (
	"context"
	"fmt"
	"log/slog"
	"math"
	"strconv"

	"github.com/ClickHouse/clickhouse-go/v2"

	"github.com/SigNoz/signoz/pkg/cache"
	errorsV2 "github.com/SigNoz/signoz/pkg/errors"
	"github.com/SigNoz/signoz/pkg/query-service/common"
	"github.com/SigNoz/signoz/pkg/query-service/constants"
	"github.com/SigNoz/signoz/pkg/query-service/interfaces"
	"github.com/SigNoz/signoz/pkg/query-service/model"
	"github.com/SigNoz/signoz/pkg/query-service/model/querytypes"
	"github.com/SigNoz/signoz/pkg/types/ctxtypes"
	"github.com/SigNoz/signoz/pkg/types/instrumentationtypes"
	"github.com/SigNoz/signoz/pkg/types/telemetrytypes"
	"github.com/SigNoz/signoz/pkg/valuer"
)

const (
	signozMetricDBName                = "signoz_metrics"
	signozTSTableNameV4               = "time_series_v4"
	signozTSTableNameV41Day           = "time_series_v4_1day"
	signozUpdatedMetricsMetadataTable = "updated_metadata"
)

type Reader struct {
	db     clickhouse.Conn
	logger *slog.Logger
	cache  cache.Cache
}

var _ interfaces.MetricMetadataReader = (*Reader)(nil)

func New(logger *slog.Logger, db clickhouse.Conn, cache cache.Cache) *Reader {
	return &Reader{db: db, logger: logger, cache: cache}
}

func (r *Reader) GetMetricMetadata(ctx context.Context, orgID valuer.UUID, metricName, serviceName string) (*querytypes.MetricMetadataResponse, error) {
	ctx = ctxtypes.NewContextWithCommentVals(ctx, map[string]string{
		instrumentationtypes.TelemetrySignal:  telemetrytypes.SignalMetrics.StringValue(),
		instrumentationtypes.CodeNamespace:    "metric-metadata-store",
		instrumentationtypes.CodeFunctionName: "GetMetricMetadata",
	})
	metadataMap, err := r.GetUpdatedMetricsMetadata(ctx, orgID, metricName)
	if err != nil {
		r.logger.Error("Error in getting metric cached metadata", errorsV2.Attr(err))
		return nil, fmt.Errorf("error fetching metric metadata: %w", err)
	}

	metadata, ok := metadataMap[metricName]
	if !ok {
		return nil, fmt.Errorf("metric metadata not found: %s", metricName)
	}

	metricType := string(metadata.MetricType)
	temporality := string(metadata.Temporality)
	var leFloat64 []float64
	if metricType == string(querytypes.MetricTypeHistogram) {
		query := fmt.Sprintf(`
			SELECT JSONExtractString(labels, 'le') AS le
			FROM %s.%s
			WHERE metric_name = $1
				AND unix_milli >= $2
				AND type = 'Histogram'
				AND (JSONExtractString(labels, 'service_name') = $3 OR JSONExtractString(labels, 'service.name') = $4)
			GROUP BY le
			ORDER BY le`, signozMetricDBName, signozTSTableNameV41Day)

		rows, err := r.db.Query(ctx, query, metricName, common.PastDayRoundOff(), serviceName, serviceName)
		if err != nil {
			r.logger.Error("Error while querying histogram buckets", errorsV2.Attr(err))
			return nil, fmt.Errorf("error while querying histogram buckets: %s", err)
		}
		defer rows.Close()

		for rows.Next() {
			var leStr string
			if err := rows.Scan(&leStr); err != nil {
				return nil, fmt.Errorf("error while scanning le: %s", err)
			}
			le, err := strconv.ParseFloat(leStr, 64)
			if err != nil || math.IsInf(le, 0) {
				r.logger.Error("Invalid 'le' bucket value", "value", leStr, errorsV2.Attr(err))
				continue
			}
			leFloat64 = append(leFloat64, le)
		}
	}

	return &querytypes.MetricMetadataResponse{
		Delta:       temporality == string(querytypes.Delta),
		Le:          leFloat64,
		Description: metadata.Description,
		Unit:        metadata.Unit,
		Type:        metricType,
		IsMonotonic: metadata.IsMonotonic,
		Temporality: temporality,
	}, nil
}

func (r *Reader) GetUpdatedMetricsMetadata(ctx context.Context, orgID valuer.UUID, metricNames ...string) (map[string]*model.UpdateMetricsMetadata, error) {
	ctx = ctxtypes.NewContextWithCommentVals(ctx, map[string]string{
		instrumentationtypes.TelemetrySignal:  telemetrytypes.SignalMetrics.StringValue(),
		instrumentationtypes.CodeNamespace:    "metric-metadata-store",
		instrumentationtypes.CodeFunctionName: "GetUpdatedMetricsMetadata",
	})
	cachedMetadata := make(map[string]*model.UpdateMetricsMetadata)
	var missingMetrics []string
	for _, metricName := range metricNames {
		metadata := new(model.UpdateMetricsMetadata)
		cacheKey := constants.UpdatedMetricsMetadataCachePrefix + metricName
		if err := r.cache.Get(ctx, orgID, cacheKey, metadata); err == nil {
			cachedMetadata[metricName] = metadata
		} else {
			missingMetrics = append(missingMetrics, metricName)
		}
	}

	var stillMissing []string
	if len(missingMetrics) > 0 {
		query := fmt.Sprintf(`SELECT metric_name, argMax(type, created_at) AS type, argMax(description, created_at) AS description, argMax(temporality, created_at) AS temporality, argMax(is_monotonic, created_at) AS is_monotonic, argMax(unit, created_at) AS unit FROM %s.%s WHERE metric_name IN ({metric_names:Array(String)}) GROUP BY metric_name;`, signozMetricDBName, signozUpdatedMetricsMetadataTable)
		rows, err := r.db.Query(
			context.WithValue(ctx, "clickhouse_max_threads", constants.MetricsExplorerClickhouseThreads),
			query,
			clickhouse.Named("metric_names", missingMetrics),
		)
		if err != nil {
			return cachedMetadata, fmt.Errorf("error querying metrics metadata: %w", err)
		}
		defer rows.Close()

		found := make(map[string]struct{})
		for rows.Next() {
			metadata := new(model.UpdateMetricsMetadata)
			if err := rows.Scan(&metadata.MetricName, &metadata.MetricType, &metadata.Description, &metadata.Temporality, &metadata.IsMonotonic, &metadata.Unit); err != nil {
				return cachedMetadata, fmt.Errorf("error scanning metrics metadata: %w", err)
			}
			r.cacheMetadata(ctx, orgID, metadata, "Failed to store metrics metadata in cache")
			cachedMetadata[metadata.MetricName] = metadata
			found[metadata.MetricName] = struct{}{}
		}
		for _, metricName := range missingMetrics {
			if _, ok := found[metricName]; !ok {
				stillMissing = append(stillMissing, metricName)
			}
		}
	}

	if len(stillMissing) > 0 {
		query := fmt.Sprintf(`SELECT DISTINCT metric_name, type, description, temporality, is_monotonic, unit FROM %s.%s WHERE metric_name IN ({metric_names:Array(String)})`, signozMetricDBName, signozTSTableNameV4)
		rows, err := r.db.Query(
			context.WithValue(ctx, "clickhouse_max_threads", constants.MetricsExplorerClickhouseThreads),
			query,
			clickhouse.Named("metric_names", stillMissing),
		)
		if err != nil {
			return cachedMetadata, fmt.Errorf("error querying time_series_v4 to get metrics metadata: %w", err)
		}
		defer rows.Close()
		for rows.Next() {
			metadata := new(model.UpdateMetricsMetadata)
			if err := rows.Scan(&metadata.MetricName, &metadata.MetricType, &metadata.Description, &metadata.Temporality, &metadata.IsMonotonic, &metadata.Unit); err != nil {
				return cachedMetadata, fmt.Errorf("error scanning fallback metadata: %w", err)
			}
			r.cacheMetadata(ctx, orgID, metadata, "Failed to cache fallback metadata")
			cachedMetadata[metadata.MetricName] = metadata
		}
		if rowsErr := rows.Err(); rowsErr != nil {
			return cachedMetadata, fmt.Errorf("error scanning fallback metadata: %w", rowsErr)
		}
	}

	return cachedMetadata, nil
}

func (r *Reader) cacheMetadata(ctx context.Context, orgID valuer.UUID, metadata *model.UpdateMetricsMetadata, errorMessage string) {
	if err := r.cache.Set(ctx, orgID, constants.UpdatedMetricsMetadataCachePrefix+metadata.MetricName, metadata, 0); err != nil {
		r.logger.Error(errorMessage, "metric_name", metadata.MetricName, errorsV2.Attr(err))
	}
}
