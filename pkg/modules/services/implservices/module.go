package implservices

import (
	"context"
	"fmt"
	"math"
	"strconv"
	"strings"
	"time"

	"github.com/ClickHouse/clickhouse-go/v2"
	"github.com/SigNoz/signoz/pkg/errors"
	"github.com/SigNoz/signoz/pkg/modules/services"
	"github.com/SigNoz/signoz/pkg/telemetrystore"
	"github.com/SigNoz/signoz/pkg/telemetrytraces"
	"github.com/SigNoz/signoz/pkg/types/ctxtypes"
	"github.com/SigNoz/signoz/pkg/types/instrumentationtypes"
	"github.com/SigNoz/signoz/pkg/types/servicetypes/servicetypesv1"
	"github.com/SigNoz/signoz/pkg/types/telemetrytypes"
	"github.com/SigNoz/signoz/pkg/valuer"
)

const traceIndexTable = "signoz_index_v3"

type module struct {
	TelemetryStore telemetrystore.TelemetryStore
}

// NewModule constructs the services module. Services has fixed aggregate
// requirements, so it reads its trace tables directly instead of depending on
// the general-purpose V5 query engine.
func NewModule(ts telemetrystore.TelemetryStore) services.Module {
	return &module{TelemetryStore: ts}
}

// FetchTopLevelOperations returns top-level operations per service using db query.
func (m *module) FetchTopLevelOperations(ctx context.Context, start time.Time, serviceNames []string) (map[string][]string, error) {
	ctx = m.withServicesContext(ctx, "FetchTopLevelOperations")

	query := fmt.Sprintf("SELECT name, serviceName, max(time) AS ts FROM %s.%s WHERE time >= @start", telemetrytraces.DBName, telemetrytraces.TopLevelOperationsTableName)
	args := []any{clickhouse.Named("start", start.UTC())}
	if len(serviceNames) > 0 {
		query += " AND serviceName IN @services"
		args = append(args, clickhouse.Named("services", serviceNames))
	}
	query += " GROUP BY name, serviceName ORDER BY ts DESC LIMIT 5000"

	rows, err := m.TelemetryStore.ClickhouseDB().Query(ctx, query, args...)
	if err != nil {
		return nil, errors.WrapInternalf(err, errors.CodeInternal, "failed to fetch top level operations")
	}
	defer rows.Close()

	operations := make(map[string][]string)
	for rows.Next() {
		var name, serviceName string
		var timestamp time.Time
		if err := rows.Scan(&name, &serviceName, &timestamp); err != nil {
			return nil, errors.WrapInternalf(err, errors.CodeInternal, "failed to scan top level operation")
		}
		if _, ok := operations[serviceName]; !ok {
			operations[serviceName] = []string{"overflow_operation"}
		}
		operations[serviceName] = append(operations[serviceName], name)
	}
	if err := rows.Err(); err != nil {
		return nil, errors.WrapInternalf(err, errors.CodeInternal, "failed to fetch top level operations")
	}
	return operations, nil
}

// Get returns the service overview aggregates for root and entry-point spans.
func (m *module) Get(ctx context.Context, _ valuer.UUID, req *servicetypesv1.Request) ([]*servicetypesv1.ResponseItem, error) {
	ctx = m.withServicesContext(ctx, "Get")
	if req == nil {
		return nil, errors.NewInvalidInputf(errors.CodeInvalidInput, "request is nil")
	}

	timeRange, err := parseServiceTimeRange(req.Start, req.End)
	if err != nil {
		return nil, err
	}
	filters, filterArgs, err := buildTagConditions(req.Tags)
	if err != nil {
		return nil, err
	}

	args := append(timeRange.args(), clickhouse.Named("scope_start", timeRange.start.UTC()))
	args = append(args, filterArgs...)
	query := fmt.Sprintf(`
		SELECT
			resource_string_service$$name AS service_name,
			quantile(0.99)(duration_nano) AS p99,
			avg(duration_nano) AS avg_duration,
			count() AS num_calls,
			countIf(status_code = 2) AS num_errors,
			countIf(toUInt16OrZero(response_status_code) >= 400 AND toUInt16OrZero(response_status_code) < 500) AS num_4xx
		FROM %s.%s
		WHERE timestamp >= @start AND timestamp < @end
			AND ts_bucket_start >= @start_bucket AND ts_bucket_start <= @end_bucket
			AND (parent_span_id = '' OR ((name, resource_string_service$$name) GLOBAL IN (
				SELECT DISTINCT name, serviceName FROM %s.%s WHERE time >= @scope_start
			) AND parent_span_id != ''))%s
		GROUP BY service_name
		ORDER BY service_name`, telemetrytraces.DBName, traceIndexTable, telemetrytraces.DBName, telemetrytraces.TopLevelOperationsTableName, filters)

	rows, err := m.TelemetryStore.ClickhouseDB().Query(ctx, query, args...)
	if err != nil {
		return nil, errors.WrapInternalf(err, errors.CodeInternal, "failed to fetch services")
	}
	defer rows.Close()

	periodSeconds := timeRange.end.Sub(timeRange.start).Seconds()
	items := make([]*servicetypesv1.ResponseItem, 0)
	serviceNames := make([]string, 0)
	for rows.Next() {
		item := &servicetypesv1.ResponseItem{DataWarning: servicetypesv1.DataWarning{TopLevelOps: []string{}}}
		if err := rows.Scan(&item.ServiceName, &item.Percentile99, &item.AvgDuration, &item.NumCalls, &item.NumErrors, &item.Num4XX); err != nil {
			return nil, errors.WrapInternalf(err, errors.CodeInternal, "failed to scan service aggregate")
		}
		if item.NumCalls > 0 {
			item.CallRate = float64(item.NumCalls) / periodSeconds
			item.ErrorRate = float64(item.NumErrors) * 100 / float64(item.NumCalls)
			item.FourXXRate = float64(item.Num4XX) * 100 / float64(item.NumCalls)
		}
		items = append(items, item)
		serviceNames = append(serviceNames, item.ServiceName)
	}
	if err := rows.Err(); err != nil {
		return nil, errors.WrapInternalf(err, errors.CodeInternal, "failed to fetch services")
	}
	if len(items) == 0 {
		return items, nil
	}

	operations, err := m.FetchTopLevelOperations(ctx, timeRange.start, serviceNames)
	if err != nil {
		return nil, err
	}
	applyOpsToItems(items, operations)
	return items, nil
}

// GetTopOperations returns the highest p99 operations for a service.
func (m *module) GetTopOperations(ctx context.Context, _ valuer.UUID, req *servicetypesv1.OperationsRequest) ([]servicetypesv1.OperationItem, error) {
	return m.getOperations(ctx, "GetTopOperations", req, false)
}

// GetEntryPointOperations returns top-level child operations for a service.
func (m *module) GetEntryPointOperations(ctx context.Context, _ valuer.UUID, req *servicetypesv1.OperationsRequest) ([]servicetypesv1.OperationItem, error) {
	return m.getOperations(ctx, "GetEntryPointOperations", req, true)
}

func (m *module) getOperations(ctx context.Context, functionName string, req *servicetypesv1.OperationsRequest, entryPoint bool) ([]servicetypesv1.OperationItem, error) {
	ctx = m.withServicesContext(ctx, functionName)
	if req == nil {
		return nil, errors.NewInvalidInputf(errors.CodeInvalidInput, "request is nil")
	}
	if req.Service == "" {
		return nil, errors.NewInvalidInputf(errors.CodeInvalidInput, "service is required")
	}
	if req.Limit < 1 || req.Limit > 5000 {
		return nil, errors.NewInvalidInputf(errors.CodeInvalidInput, "limit must be between 1 and 5000")
	}

	timeRange, err := parseServiceTimeRange(req.Start, req.End)
	if err != nil {
		return nil, err
	}
	filters, filterArgs, err := buildTagConditions(req.Tags)
	if err != nil {
		return nil, err
	}

	args := append(timeRange.args(), clickhouse.Named("service_name", req.Service))
	if entryPoint {
		args = append(args, clickhouse.Named("scope_start", timeRange.start.UTC()))
	}
	args = append(args, filterArgs...)

	scope := ""
	if entryPoint {
		scope = fmt.Sprintf(` AND ((name, resource_string_service$$name) GLOBAL IN (
			SELECT DISTINCT name, serviceName FROM %s.%s WHERE time >= @scope_start
		) AND parent_span_id != '')`, telemetrytraces.DBName, telemetrytraces.TopLevelOperationsTableName)
	}
	query := fmt.Sprintf(`
		SELECT
			name,
			quantile(0.50)(duration_nano) AS p50,
			quantile(0.95)(duration_nano) AS p95,
			quantile(0.99)(duration_nano) AS p99,
			count() AS num_calls,
			countIf(status_code = 2) AS error_count
		FROM %s.%s
		WHERE timestamp >= @start AND timestamp < @end
			AND ts_bucket_start >= @start_bucket AND ts_bucket_start <= @end_bucket
			AND resource_string_service$$name = @service_name%s%s
		GROUP BY name
		ORDER BY p99 DESC
		LIMIT %d`, telemetrytraces.DBName, traceIndexTable, scope, filters, req.Limit)

	rows, err := m.TelemetryStore.ClickhouseDB().Query(ctx, query, args...)
	if err != nil {
		return nil, errors.WrapInternalf(err, errors.CodeInternal, "failed to fetch service operations")
	}
	defer rows.Close()

	items := make([]servicetypesv1.OperationItem, 0)
	for rows.Next() {
		var item servicetypesv1.OperationItem
		if err := rows.Scan(&item.Name, &item.P50, &item.P95, &item.P99, &item.NumCalls, &item.ErrorCount); err != nil {
			return nil, errors.WrapInternalf(err, errors.CodeInternal, "failed to scan service operation")
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return nil, errors.WrapInternalf(err, errors.CodeInternal, "failed to fetch service operations")
	}
	return items, nil
}

type serviceTimeRange struct {
	start time.Time
	end   time.Time
}

func (r serviceTimeRange) args() []any {
	return []any{
		clickhouse.Named("start", r.start.UTC()),
		clickhouse.Named("end", r.end.UTC()),
		clickhouse.Named("start_bucket", uint64(r.start.Unix())),
		clickhouse.Named("end_bucket", uint64(r.end.Unix())),
	}
}

func parseServiceTimeRange(start, end string) (serviceTimeRange, error) {
	startNS, err := strconv.ParseUint(start, 10, 64)
	if err != nil || startNS > math.MaxInt64 {
		return serviceTimeRange{}, errors.NewInvalidInputf(errors.CodeInvalidInput, "invalid start time")
	}
	endNS, err := strconv.ParseUint(end, 10, 64)
	if err != nil || endNS > math.MaxInt64 {
		return serviceTimeRange{}, errors.NewInvalidInputf(errors.CodeInvalidInput, "invalid end time")
	}
	if startNS >= endNS {
		return serviceTimeRange{}, errors.NewInvalidInputf(errors.CodeInvalidInput, "start must be before end")
	}
	return serviceTimeRange{start: time.Unix(0, int64(startNS)).UTC(), end: time.Unix(0, int64(endNS)).UTC()}, nil
}

func buildTagConditions(tags []servicetypesv1.TagFilterItem) (string, []any, error) {
	if err := validateTagFilterItems(tags); err != nil {
		return "", nil, err
	}
	conditions := make([]string, 0, len(tags))
	args := make([]any, 0, len(tags)*2)
	for index, tag := range tags {
		column, values := tagColumnAndValues(tag)
		if column == "" {
			return "", nil, errors.NewInvalidInputf(errors.CodeInvalidInput, "unsupported tag type: %s", tag.TagType)
		}
		keyParam := fmt.Sprintf("filter_%d_key", index)
		valuesParam := fmt.Sprintf("filter_%d_values", index)
		operator := "IN"
		if strings.EqualFold(tag.Operator, "notin") {
			operator = "NOT IN"
		}
		conditions = append(conditions, fmt.Sprintf("mapContains(%s, @%s) AND %s[@%s] %s @%s", column, keyParam, column, keyParam, operator, valuesParam))
		args = append(args, clickhouse.Named(keyParam, tag.Key), clickhouse.Named(valuesParam, values))
	}
	if len(conditions) == 0 {
		return "", args, nil
	}
	return " AND " + strings.Join(conditions, " AND "), args, nil
}

func tagColumnAndValues(tag servicetypesv1.TagFilterItem) (string, any) {
	tagType := strings.ToLower(strings.ReplaceAll(tag.TagType, "_", ""))
	resource := tagType == "" || tagType == "resource" || tagType == "resourceattribute"
	span := tagType == "span" || tagType == "spanattribute" || tagType == "attribute"
	if !resource && !span {
		return "", nil
	}
	prefix := "resources"
	if span {
		prefix = "attributes"
	}
	if len(tag.StringValues) > 0 {
		return prefix + "_string", tag.StringValues
	}
	if len(tag.NumberValues) > 0 {
		return prefix + "_number", tag.NumberValues
	}
	return prefix + "_bool", tag.BoolValues
}

func (m *module) withServicesContext(ctx context.Context, functionName string) context.Context {
	return ctxtypes.NewContextWithCommentVals(ctx, map[string]string{
		instrumentationtypes.TelemetrySignal:  telemetrytypes.SignalTraces.StringValue(),
		instrumentationtypes.CodeNamespace:    "services",
		instrumentationtypes.CodeFunctionName: functionName,
	})
}
