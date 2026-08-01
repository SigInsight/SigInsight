package implspanpercentile

import (
	"context"
	"fmt"
	"math"
	"sort"
	"strings"
	"time"

	"github.com/ClickHouse/clickhouse-go/v2"
	"github.com/SigNoz/signoz/pkg/errors"
	"github.com/SigNoz/signoz/pkg/modules/spanpercentile"
	"github.com/SigNoz/signoz/pkg/telemetrystore"
	"github.com/SigNoz/signoz/pkg/telemetrytraces"
	"github.com/SigNoz/signoz/pkg/types/ctxtypes"
	"github.com/SigNoz/signoz/pkg/types/instrumentationtypes"
	"github.com/SigNoz/signoz/pkg/types/spanpercentiletypes"
	"github.com/SigNoz/signoz/pkg/valuer"
)

const traceIndexTable = "span_index_v3"

type module struct {
	telemetryStore telemetrystore.TelemetryStore
}

func NewModule(telemetryStore telemetrystore.TelemetryStore) spanpercentile.Module {
	return &module{telemetryStore: telemetryStore}
}

// GetSpanPercentile returns percentile context for one service operation. This
// workflow has one fixed aggregate shape, so it reads the trace index directly
// instead of building a V5 scalar request.
func (m *module) GetSpanPercentile(ctx context.Context, _ valuer.UUID, _ valuer.UUID, req *spanpercentiletypes.SpanPercentileRequest) (*spanpercentiletypes.SpanPercentileResponse, error) {
	ctx = ctxtypes.NewContextWithCommentVals(ctx, map[string]string{
		instrumentationtypes.CodeNamespace:    "spanpercentile",
		instrumentationtypes.CodeFunctionName: "GetSpanPercentile",
	})
	if req == nil {
		return nil, errors.New(errors.TypeInvalidInput, errors.CodeInvalidInput, "request is nil")
	}
	if err := req.Validate(); err != nil {
		return nil, err
	}
	if req.Start > math.MaxInt64 || req.End > math.MaxInt64 {
		return nil, errors.New(errors.TypeInvalidInput, errors.CodeInvalidInput, "span percentile time is out of range")
	}

	attributes, attributeArgs := resourceAttributeConditions(req.ResourceAttributes)
	start := time.UnixMilli(int64(req.Start)).UTC()
	end := time.UnixMilli(int64(req.End)).UTC()
	args := []any{
		clickhouse.Named("start", start),
		clickhouse.Named("end", end),
		clickhouse.Named("start_bucket", uint64(start.Unix())),
		clickhouse.Named("end_bucket", uint64(end.Unix())),
		clickhouse.Named("service_name", req.ServiceName),
		clickhouse.Named("span_name", req.Name),
		clickhouse.Named("duration_nano", uint64(req.DurationNano)),
	}
	args = append(args, attributeArgs...)

	query := fmt.Sprintf(`
		SELECT
			quantile(0.50)(duration_nano) AS p50,
			quantile(0.90)(duration_nano) AS p90,
			quantile(0.99)(duration_nano) AS p99,
			(100.0 * countIf(duration_nano <= @duration_nano)) / count() AS percentile_position,
			count() AS span_count
		FROM %s.%s
		WHERE timestamp >= @start AND timestamp < @end
			AND ts_bucket_start >= @start_bucket AND ts_bucket_start <= @end_bucket
			AND resource_string_service$$name = @service_name
			AND name = @span_name%s`, telemetrytraces.DBName, traceIndexTable, attributes)

	rows, err := m.telemetryStore.ClickhouseDB().Query(ctx, query, args...)
	if err != nil {
		return nil, errors.WrapInternalf(err, errors.CodeInternal, "failed to fetch span percentile")
	}
	defer rows.Close()
	if !rows.Next() {
		if err := rows.Err(); err != nil {
			return nil, errors.WrapInternalf(err, errors.CodeInternal, "failed to fetch span percentile")
		}
		return nil, noMatchingSpansError()
	}

	var p50, p90, p99, position float64
	var spanCount uint64
	if err := rows.Scan(&p50, &p90, &p99, &position, &spanCount); err != nil {
		return nil, errors.WrapInternalf(err, errors.CodeInternal, "failed to scan span percentile")
	}
	if err := rows.Err(); err != nil {
		return nil, errors.WrapInternalf(err, errors.CodeInternal, "failed to fetch span percentile")
	}
	if spanCount == 0 {
		return nil, noMatchingSpansError()
	}

	description := fmt.Sprintf("slower than %.1f%% of spans", position)
	if position < 50 {
		description = fmt.Sprintf("faster than %.1f%% of spans", 100-position)
	}
	return &spanpercentiletypes.SpanPercentileResponse{
		Percentiles: spanpercentiletypes.PercentileStats{P50: p50, P90: p90, P99: p99},
		Position:    spanpercentiletypes.PercentilePosition{Percentile: position, Description: description},
	}, nil
}

func resourceAttributeConditions(attributes map[string]string) (string, []any) {
	keys := make([]string, 0, len(attributes))
	for key := range attributes {
		keys = append(keys, key)
	}
	sort.Strings(keys)

	conditions := make([]string, 0, len(keys))
	args := make([]any, 0, len(keys)*2)
	for index, key := range keys {
		keyParam := fmt.Sprintf("resource_%d_key", index)
		valueParam := fmt.Sprintf("resource_%d_value", index)
		conditions = append(conditions, fmt.Sprintf("mapContains(resources_string, @%s) AND resources_string[@%s] = @%s", keyParam, keyParam, valueParam))
		args = append(args, clickhouse.Named(keyParam, key), clickhouse.Named(valueParam, attributes[key]))
	}
	if len(conditions) == 0 {
		return "", args
	}
	return " AND " + strings.Join(conditions, " AND "), args
}

func noMatchingSpansError() error {
	return errors.New(errors.TypeNotFound, errors.CodeNotFound, "no spans found matching the specified criteria")
}
