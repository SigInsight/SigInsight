// Package schemareadiness verifies the ClickHouse contract owned by the
// current Collector release before SigInsight creates any query consumers.
package schemareadiness

import (
	"context"
	"fmt"
	"sort"
	"strings"

	"github.com/SigNoz/signoz/pkg/errors"
	"github.com/SigNoz/signoz/pkg/telemetrystore"
)

type requirement struct {
	database string
	table    string
	columns  []string
}

const (
	logsDB      = "siginsight_logs"
	tracesDB    = "siginsight_traces"
	metricsDB   = "siginsight_metrics"
	meterDB     = "siginsight_meter"
	analyticsDB = "siginsight_analytics"
)

var required = []requirement{
	{logsDB, "logs", []string{"timestamp", "trace_id", "span_id", "body", "attributes_string", "attributes_number", "attributes_bool", "resources_string", "resource_fingerprint", "_retention_days", "_retention_days_cold"}},
	{logsDB, "resource_sets", []string{"labels", "fingerprint", "seen_at_ts_bucket_start"}},
	{logsDB, "field_values", []string{"tag_key", "tag_type", "tag_data_type", "string_value", "number_value"}},
	{logsDB, "logs_attribute_keys", []string{"name", "datatype"}},
	{logsDB, "logs_resource_keys", []string{"name", "datatype"}},
	{logsDB, "usage", nil},

	{tracesDB, "spans", []string{"timestamp", "trace_id", "span_id", "parent_span_id", "name", "kind", "kind_string", "duration_nano", "status_code", "status_message", "status_code_string", "attributes_string", "attributes_number", "attributes_bool", "resources_string", "events", "links", "response_status_code", "http_method", "http_host", "http_url", "db_name", "db_operation", "has_error", "is_remote", "service_name", "http_route", "messaging_system", "messaging_operation", "db_system", "rpc_system", "rpc_service", "rpc_method", "peer_service", "service_name_present", "http_route_present", "messaging_system_present", "messaging_operation_present", "db_system_present", "rpc_system_present", "rpc_service_present", "rpc_method_present", "peer_service_present"}},
	{tracesDB, "resource_sets", []string{"labels", "fingerprint", "seen_at_ts_bucket_start"}},
	{tracesDB, "field_values", []string{"tag_key", "tag_type", "tag_data_type", "string_value", "number_value"}},
	{tracesDB, "span_attributes_keys", []string{"tagKey", "tagType", "dataType"}},
	{tracesDB, "exceptions", []string{"serviceName", "traceID", "spanID", "timestamp"}},
	{tracesDB, "operations", []string{"serviceName", "name", "time"}},
	{tracesDB, "root_operations_mv", nil},
	{tracesDB, "sub_root_operations_mv", nil},
	{tracesDB, "trace_summary", []string{"trace_id", "start", "end", "num_spans"}},
	{tracesDB, "trace_summary_from_spans_mv", nil},
	{tracesDB, "service_edges", []string{"timestamp", "src", "dest"}},
	{tracesDB, "service_edges_service_calls_mv", nil},
	{tracesDB, "service_edges_db_calls_mv", nil},
	{tracesDB, "service_edges_messaging_calls_mv", nil},
	{tracesDB, "usage", nil},

	{metricsDB, "metric_points", []string{"env", "temporality", "metric_name", "fingerprint", "unix_milli", "value", "flags"}},
	{metricsDB, "metric_rollup_5m", []string{"metric_name", "fingerprint", "unix_milli", "count", "sum"}},
	{metricsDB, "metric_rollup_5m_mv", nil},
	{metricsDB, "metric_rollup_30m", []string{"metric_name", "fingerprint", "unix_milli", "count", "sum"}},
	{metricsDB, "metric_rollup_30m_mv", nil},
	{metricsDB, "metric_series", []string{"env", "temporality", "metric_name", "type", "is_monotonic", "fingerprint", "resource_attrs", "labels"}},
	{metricsDB, "metric_series_6h", []string{"metric_name", "fingerprint", "unix_milli"}},
	{metricsDB, "metric_series_6h_mv", nil},
	{metricsDB, "metric_series_1d", []string{"metric_name", "fingerprint", "unix_milli"}},
	{metricsDB, "metric_series_1d_mv", nil},
	{metricsDB, "metric_series_1w", []string{"metric_name", "fingerprint", "unix_milli"}},
	{metricsDB, "metric_series_1w_mv", nil},
	{metricsDB, "metadata", []string{"metric_name", "type", "temporality"}},
	{metricsDB, "exp_hist", []string{"metric_name", "fingerprint", "unix_milli"}},
	{metricsDB, "usage", nil},

	{meterDB, "meter_points", []string{"temporality", "metric_name", "type", "fingerprint", "unix_milli", "value"}},
	{meterDB, "meter_rollup_1d", []string{"temporality", "metric_name", "fingerprint", "unix_milli", "count", "sum"}},
	{meterDB, "meter_rollup_1d_mv", nil},
	{analyticsDB, "rule_state_history", []string{"rule_id", "unix_milli", "value", "state"}},
}

var legacy = map[string][]string{
	logsDB:      {"logs_v2", "logs_v2_resource", "tag_attributes_v2"},
	tracesDB:    {"span_index_v3", "traces_v3_resource", "tag_attributes_v2", "error_index_v2", "top_level_operations", "dependency_graph_minutes_v2", "root_operations", "sub_root_operations", "trace_summary_mv"},
	metricsDB:   {"samples_v4", "samples_v4_agg_5m", "samples_v4_agg_30m", "time_series_v4", "time_series_v4_6hrs", "time_series_v4_1day", "time_series_v4_1week"},
	meterDB:     {"samples", "samples_agg_1d", "samples_agg_1d_mv"},
	analyticsDB: {"rule_state_history_v0"},
}

// Validate rejects a ClickHouse instance that does not match the canonical
// Collector schema. It intentionally has no old-schema fallback: M16 is a
// destructive release boundary and stale data must be recreated by Collector.
func Validate(ctx context.Context, store telemetrystore.TelemetryStore) error {
	rows, err := store.ClickhouseDB().Query(ctx, `SELECT database, name FROM system.tables WHERE database IN (?, ?, ?, ?, ?)`, logsDB, tracesDB, metricsDB, meterDB, analyticsDB)
	if err != nil {
		return errors.WrapInternalf(err, errors.CodeInternal, "read ClickHouse schema tables")
	}
	defer rows.Close()

	tables := make(map[string]struct{})
	for rows.Next() {
		var database string
		var table string
		if err := rows.Scan(&database, &table); err != nil {
			return errors.WrapInternalf(err, errors.CodeInternal, "scan ClickHouse schema table")
		}
		tables[database+"."+table] = struct{}{}
	}
	if err := rows.Err(); err != nil {
		return errors.WrapInternalf(err, errors.CodeInternal, "read ClickHouse schema tables")
	}

	missingTables := make([]string, 0)
	for _, item := range required {
		key := item.database + "." + item.table
		if _, ok := tables[key]; !ok {
			missingTables = append(missingTables, key)
		}
	}
	if len(missingTables) > 0 {
		sort.Strings(missingTables)
		return errors.NewInternalf(errors.CodeInternal, "canonical ClickHouse schema is missing required tables: %s", strings.Join(missingTables, ", "))
	}

	legacyTables := make([]string, 0)
	for database, names := range legacy {
		for _, name := range names {
			if _, ok := tables[database+"."+name]; ok {
				legacyTables = append(legacyTables, database+"."+name)
			}
		}
	}
	if len(legacyTables) > 0 {
		sort.Strings(legacyTables)
		return errors.NewInternalf(errors.CodeInternal, "legacy ClickHouse schema objects remain after canonical cutover: %s", strings.Join(legacyTables, ", "))
	}

	columnRows, err := store.ClickhouseDB().Query(ctx, `SELECT database, table, name FROM system.columns WHERE database IN (?, ?, ?, ?, ?)`, logsDB, tracesDB, metricsDB, meterDB, analyticsDB)
	if err != nil {
		return errors.WrapInternalf(err, errors.CodeInternal, "read ClickHouse schema columns")
	}
	defer columnRows.Close()

	columns := make(map[string]map[string]struct{})
	for columnRows.Next() {
		var database string
		var table string
		var column string
		if err := columnRows.Scan(&database, &table, &column); err != nil {
			return errors.WrapInternalf(err, errors.CodeInternal, "scan ClickHouse schema column")
		}
		key := database + "." + table
		if columns[key] == nil {
			columns[key] = make(map[string]struct{})
		}
		columns[key][column] = struct{}{}
	}
	if err := columnRows.Err(); err != nil {
		return errors.WrapInternalf(err, errors.CodeInternal, "read ClickHouse schema columns")
	}

	missingColumns := make([]string, 0)
	for _, item := range required {
		key := item.database + "." + item.table
		for _, column := range item.columns {
			if _, ok := columns[key][column]; !ok {
				missingColumns = append(missingColumns, fmt.Sprintf("%s.%s", key, column))
			}
		}
	}
	if len(missingColumns) > 0 {
		sort.Strings(missingColumns)
		return errors.NewInternalf(errors.CodeInternal, "canonical ClickHouse schema is missing required columns: %s", strings.Join(missingColumns, ", "))
	}

	return nil
}
