package metricsexplorerstore

import (
	"context"
	"fmt"

	"github.com/ClickHouse/clickhouse-go/v2/lib/driver"

	errorsV2 "github.com/SigNoz/signoz/pkg/errors"
	"github.com/SigNoz/signoz/pkg/query-service/constants"
	"github.com/SigNoz/signoz/pkg/query-service/model/metrics_explorer"
	"github.com/SigNoz/signoz/pkg/types/ctxtypes"
	"github.com/SigNoz/signoz/pkg/types/instrumentationtypes"
	"github.com/SigNoz/signoz/pkg/types/telemetrytypes"
)

// Metric unit and type suggestions share the metadata-table facet query shape.
func (r *Reader) GetAllMetricFilterUnits(ctx context.Context, req *metrics_explorer.FilterValueRequest) ([]string, error) {
	return r.getMetricMetadataFacet(ctx, req, "unit", "GetAllMetricFilterUnits")
}

func (r *Reader) GetAllMetricFilterTypes(ctx context.Context, req *metrics_explorer.FilterValueRequest) ([]string, error) {
	return r.getMetricMetadataFacet(ctx, req, "type", "GetAllMetricFilterTypes")
}

func (r *Reader) getMetricMetadataFacet(ctx context.Context, req *metrics_explorer.FilterValueRequest, column, operation string) ([]string, error) {
	ctx = ctxtypes.NewContextWithCommentVals(ctx, map[string]string{
		instrumentationtypes.TelemetrySignal:  telemetrytypes.SignalMetrics.StringValue(),
		instrumentationtypes.CodeNamespace:    "clickhouse-reader",
		instrumentationtypes.CodeFunctionName: operation,
	})
	query := fmt.Sprintf("SELECT DISTINCT %s FROM %s.%s WHERE %s ILIKE $1 AND %s IS NOT NULL ORDER BY %s", column, signozMetricDBName, signozTSTableNameV41Day, column, column, column)
	if req.Limit != 0 {
		query += fmt.Sprintf(" LIMIT %d;", req.Limit)
	}
	valueCtx := context.WithValue(ctx, "clickhouse_max_threads", constants.MetricsExplorerClickhouseThreads)
	rows, err := r.db.Query(valueCtx, query, fmt.Sprintf("%%%s%%", req.SearchText))
	if err != nil {
		r.logger.Error("Error while executing query", errorsV2.Attr(err))
		return nil, fmt.Errorf("query metric filter %ss: %w", column, err)
	}
	defer rows.Close()
	return scanMetricFacet(rows, column)
}

func scanMetricFacet(rows driver.Rows, column string) ([]string, error) {
	var response []string
	for rows.Next() {
		var value string
		if err := rows.Scan(&value); err != nil {
			return nil, fmt.Errorf("scan metric filter %s: %w", column, err)
		}
		response = append(response, value)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate metric filter %ss: %w", column, err)
	}
	return response, nil
}
