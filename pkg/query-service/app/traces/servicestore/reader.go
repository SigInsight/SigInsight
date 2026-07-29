package servicestore

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/ClickHouse/clickhouse-go/v2"

	errorsV2 "github.com/SigNoz/signoz/pkg/errors"
	"github.com/SigNoz/signoz/pkg/query-service/interfaces"
	"github.com/SigNoz/signoz/pkg/query-service/model"
	"github.com/SigNoz/signoz/pkg/types/ctxtypes"
	"github.com/SigNoz/signoz/pkg/types/instrumentationtypes"
	"github.com/SigNoz/signoz/pkg/types/telemetrytypes"
)

type Config struct {
	TraceDB                 string
	DependencyGraphTable    string
	TopLevelOperationsTable string
}

type Reader struct {
	db                      clickhouse.Conn
	logger                  *slog.Logger
	traceDB                 string
	dependencyGraphTable    string
	topLevelOperationsTable string
}

var _ interfaces.ServiceReader = (*Reader)(nil)

func New(logger *slog.Logger, db clickhouse.Conn, config Config) *Reader {
	return &Reader{
		db:                      db,
		logger:                  logger,
		traceDB:                 config.TraceDB,
		dependencyGraphTable:    config.DependencyGraphTable,
		topLevelOperationsTable: config.TopLevelOperationsTable,
	}
}

func (r *Reader) GetTopLevelOperations(ctx context.Context, start, end time.Time, services []string) (*map[string][]string, error) {
	ctx = withTraceQueryMetadata(ctx, "GetTopLevelOperations")
	start = start.In(time.UTC)

	operations := map[string][]string{}
	// This table stores only the most recent instance of each operation, so end
	// cannot be used to bound the query.
	query := fmt.Sprintf(`SELECT name, serviceName, max(time) as ts FROM %s.%s WHERE time >= @start`, r.traceDB, r.topLevelOperationsTable)
	if len(services) > 0 {
		query += ` AND serviceName IN @services`
	}
	query += ` GROUP BY name, serviceName ORDER BY ts DESC LIMIT 5000`

	rows, err := r.db.Query(ctx, query, clickhouse.Named("start", start), clickhouse.Named("services", services))
	if err != nil {
		r.logger.Error("Error in processing sql query", errorsV2.Attr(err))
		return nil, fmt.Errorf("query top-level operations: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var name, serviceName string
		var timestamp time.Time
		if err := rows.Scan(&name, &serviceName, &timestamp); err != nil {
			return nil, fmt.Errorf("scan top-level operations: %w", err)
		}
		if _, ok := operations[serviceName]; !ok {
			operations[serviceName] = []string{"overflow_operation"}
		}
		operations[serviceName] = append(operations[serviceName], name)
	}
	return &operations, nil
}

func (r *Reader) GetDependencyGraph(ctx context.Context, queryParams *model.GetServicesParams) (*[]model.ServiceMapDependencyResponseItem, error) {
	ctx = withTraceQueryMetadata(ctx, "GetDependencyGraph")
	response := []model.ServiceMapDependencyResponseItem{}
	args := []interface{}{
		clickhouse.Named("start", uint64(queryParams.Start.Unix())),
		clickhouse.Named("end", uint64(queryParams.End.Unix())),
		clickhouse.Named("duration", uint64(queryParams.End.Unix()-queryParams.Start.Unix())),
	}

	query := fmt.Sprintf(`
		WITH
			quantilesMergeState(0.5, 0.75, 0.9, 0.95, 0.99)(duration_quantiles_state) AS duration_quantiles_state,
			finalizeAggregation(duration_quantiles_state) AS result
		SELECT
			src as parent,
			dest as child,
			result[1] AS p50,
			result[2] AS p75,
			result[3] AS p90,
			result[4] AS p95,
			result[5] AS p99,
			sum(total_count) as callCount,
			sum(total_count)/ @duration AS callRate,
			sum(error_count)/sum(total_count) * 100 as errorRate
		FROM %s.%s
		WHERE toUInt64(toDateTime(timestamp)) >= @start AND toUInt64(toDateTime(timestamp)) <= @end`,
		r.traceDB, r.dependencyGraphTable,
	)
	query += " GROUP BY src, dest;"

	r.logger.Debug("GetDependencyGraph query", "query", query, "args", args)
	if err := r.db.Select(ctx, &response, query, args...); err != nil {
		r.logger.Error("Error in processing sql query", errorsV2.Attr(err))
		return nil, fmt.Errorf("error in processing sql query %w", err)
	}

	return &response, nil
}

func withTraceQueryMetadata(ctx context.Context, functionName string) context.Context {
	return ctxtypes.NewContextWithCommentVals(ctx, map[string]string{
		instrumentationtypes.TelemetrySignal:  telemetrytypes.SignalTraces.StringValue(),
		instrumentationtypes.CodeNamespace:    "clickhouse-reader",
		instrumentationtypes.CodeFunctionName: functionName,
	})
}
