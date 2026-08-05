package constants

import (
	"maps"
	"os"
	"regexp"
	"strconv"

	"github.com/SigNoz/signoz/pkg/query-service/model"
	"github.com/SigNoz/signoz/pkg/query-service/model/querytypes"
	"github.com/SigNoz/signoz/pkg/valuer"
)

const (
	HTTPHostPort    = "0.0.0.0:8080" // Address to serve http (query service)
	PrivateHostPort = "0.0.0.0:8085" // Address to server internal services like alert manager
	OpAmpWsEndpoint = "0.0.0.0:4320" // address for opamp websocket
)

const MaxAllowedPointsInTimeSeries = 300

const TraceTTL = "traces"
const MetricsTTL = "metrics"
const LogsTTL = "logs"

const SpanSearchScopeRoot = "isroot"
const SpanSearchScopeEntryPoint = "isentrypoint"
const OrderBySpanCount = "span_count"

var MetricsExplorerClickhouseThreads = GetOrDefaultEnvInt("METRICS_EXPLORER_CLICKHOUSE_THREADS", 8)
var MetricsMetadataCachePrefix = GetOrDefaultEnv("METRICS_METADATA_CACHE_KEY", "METRICS_METADATA")

const NormalizedMetricsMapCacheKey = "NORMALIZED_METRICS_MAP_CACHE_KEY"
const NormalizedMetricsMapQueryThreads = 10

var NormalizedMetricsMapRegex = regexp.MustCompile(`[^a-zA-Z0-9]`)
var NormalizedMetricsMapQuantileRegex = regexp.MustCompile(`(?i)([._-]?quantile.*)$`)

func GetEvalDelay() valuer.TextDuration {
	evalDelayStr := GetOrDefaultEnv("RULES_EVAL_DELAY", "2m")
	evalDelayDuration, err := valuer.ParseTextDuration(evalDelayStr)
	if err != nil {
		return valuer.TextDuration{}
	}
	return evalDelayDuration
}

const (
	TraceID                        = "traceID"
	ServiceName                    = "serviceName"
	HttpRoute                      = "httpRoute"
	HttpHost                       = "httpHost"
	HttpUrl                        = "httpUrl"
	HttpMethod                     = "httpMethod"
	OperationDB                    = "name"
	OperationRequest               = "operation"
	Status                         = "status"
	Duration                       = "duration"
	DBName                         = "dbName"
	DBOperation                    = "dbOperation"
	DBSystem                       = "dbSystem"
	MsgSystem                      = "msgSystem"
	MsgOperation                   = "msgOperation"
	Timestamp                      = "timestamp"
	RPCMethod                      = "rpcMethod"
	ResponseStatusCode             = "responseStatusCode"
	Descending                     = "descending"
	Ascending                      = "ascending"
	StatusPending                  = "pending"
	StatusFailed                   = "failed"
	StatusSuccess                  = "success"
	ExceptionType                  = "exceptionType"
	ExceptionCount                 = "exceptionCount"
	LastSeen                       = "lastSeen"
	FirstSeen                      = "firstSeen"
	Attributes                     = "attributes"
	Resources                      = "resources"
	Static                         = "static"
	DefaultLogSkipIndexType        = "bloom_filter(0.01)"
	DefaultLogSkipIndexGranularity = 64
)

var GroupByColMap = map[string]struct{}{
	ServiceName:        {},
	HttpHost:           {},
	HttpRoute:          {},
	HttpUrl:            {},
	HttpMethod:         {},
	OperationDB:        {},
	DBName:             {},
	DBOperation:        {},
	DBSystem:           {},
	MsgOperation:       {},
	MsgSystem:          {},
	RPCMethod:          {},
	ResponseStatusCode: {},
}

const (
	SIGINSIGHT_METRIC_DBNAME                       = "siginsight_metrics"
	SIGINSIGHT_SAMPLES_V4_LOCAL_TABLENAME          = "samples_v4"
	SIGINSIGHT_SAMPLES_V4_TABLENAME                = "samples_v4"
	SIGINSIGHT_SAMPLES_V4_AGG_5M_TABLENAME         = "samples_v4_agg_5m"
	SIGINSIGHT_SAMPLES_V4_AGG_30M_TABLENAME        = "samples_v4_agg_30m"
	SIGINSIGHT_EXP_HISTOGRAM_TABLENAME             = "exp_hist"
	SIGINSIGHT_EXP_HISTOGRAM_LOCAL_TABLENAME       = "exp_hist"
	SIGINSIGHT_TRACE_DBNAME                        = "siginsight_traces"
	SIGINSIGHT_SPAN_INDEX_TABLENAME                = "span_index_v2"
	SIGINSIGHT_SPAN_INDEX_V3                       = "span_index_v3"
	SIGINSIGHT_SPAN_INDEX_LOCAL_TABLENAME          = "span_index_v2"
	SIGINSIGHT_SPAN_INDEX_V3_LOCAL_TABLENAME       = "span_index_v3"
	SIGINSIGHT_TIMESERIES_v4_LOCAL_TABLENAME       = "time_series_v4"
	SIGINSIGHT_TIMESERIES_V4_TABLENAME             = "time_series_v4"
	SIGINSIGHT_TIMESERIES_v4_6HRS_LOCAL_TABLENAME  = "time_series_v4_6hrs"
	SIGINSIGHT_TIMESERIES_v4_1DAY_LOCAL_TABLENAME  = "time_series_v4_1day"
	SIGINSIGHT_TIMESERIES_v4_1WEEK_LOCAL_TABLENAME = "time_series_v4_1week"
	SIGINSIGHT_TIMESERIES_v4_1DAY_TABLENAME        = "time_series_v4_1day"
	SIGINSIGHT_TOP_LEVEL_OPERATIONS_TABLENAME      = "top_level_operations"
	SIGINSIGHT_TIMESERIES_v4_TABLENAME             = "time_series_v4"
	SIGINSIGHT_TIMESERIES_v4_1WEEK_TABLENAME       = "time_series_v4_1week"
	SIGINSIGHT_TIMESERIES_v4_6HRS_TABLENAME        = "time_series_v4_6hrs"
	SIGINSIGHT_ATTRIBUTES_METADATA_TABLENAME       = "attributes_metadata"
	SIGINSIGHT_ATTRIBUTES_METADATA_LOCAL_TABLENAME = "attributes_metadata"
	SIGINSIGHT_METADATA_TABLENAME                  = "metadata"
	SIGINSIGHT_METADATA_LOCAL_TABLENAME            = "metadata"
)

// alert related constants
const (
	AlertTimeFormat = "2006-01-02 15:04:05"
)

func GetOrDefaultEnv(key string, fallback string) string {
	v := os.Getenv(key)
	if len(v) == 0 {
		return fallback
	}
	return v
}

func GetOrDefaultEnvInt(key string, fallback int) int {
	v := os.Getenv(key)
	if len(v) == 0 {
		return fallback
	}
	intVal, err := strconv.Atoi(v)
	if err != nil {
		return fallback
	}
	return intVal
}

const (
	STRING                = "String"
	UINT32                = "UInt32"
	LOWCARDINALITY_STRING = "LowCardinality(String)"
	INT32                 = "Int32"
	UINT8                 = "Uint8"
)

var StaticSelectedLogFields = []model.Field{
	{
		Name:     "timestamp",
		DataType: UINT32,
		Type:     Static,
	},
	{
		Name:     "id",
		DataType: STRING,
		Type:     Static,
	},
	{
		Name:     "severity_text",
		DataType: LOWCARDINALITY_STRING,
		Type:     Static,
	},
	{
		Name:     "severity_number",
		DataType: UINT8,
		Type:     Static,
	},
	{
		Name:     "trace_flags",
		DataType: UINT32,
		Type:     Static,
	},
	{
		Name:     "trace_id",
		DataType: STRING,
		Type:     Static,
	},
	{
		Name:     "span_id",
		DataType: STRING,
		Type:     Static,
	},
}

const (
	LogsSQLSelect = "SELECT " +
		"timestamp, id, trace_id, span_id, trace_flags, severity_text, severity_number, scope_name, scope_version, body," +
		"CAST((attributes_string_key, attributes_string_value), 'Map(String, String)') as  attributes_string," +
		"CAST((attributes_int64_key, attributes_int64_value), 'Map(String, Int64)') as  attributes_int64," +
		"CAST((attributes_float64_key, attributes_float64_value), 'Map(String, Float64)') as  attributes_float64," +
		"CAST((attributes_bool_key, attributes_bool_value), 'Map(String, Bool)') as  attributes_bool," +
		"CAST((resources_string_key, resources_string_value), 'Map(String, String)') as resources_string," +
		"CAST((scope_string_key, scope_string_value), 'Map(String, String)') as scope "
	LogsSQLSelectV2 = "SELECT " +
		"timestamp, id, trace_id, span_id, trace_flags, severity_text, severity_number, scope_name, scope_version, body, " +
		"attributes_string, " +
		"attributes_number, " +
		"attributes_bool, " +
		"resources_string, " +
		"scope_string "
	TracesExplorerViewSQLSelectWithSubQuery = "(SELECT traceID, durationNano, " +
		"serviceName, name FROM %s.%s WHERE parentSpanID = '' AND %s ORDER BY durationNano DESC LIMIT 1 BY traceID"
	TracesExplorerViewSQLSelectBeforeSubQuery = "SELECT subQuery.serviceName as `subQuery.serviceName`, subQuery.name as `subQuery.name`, count() AS " +
		"span_count, subQuery.durationNano as `subQuery.durationNano`, subQuery.traceID FROM " +
		"(SELECT traceID AS dist_traceID, timestamp, ts_bucket_start FROM %s.%s WHERE %s%s) as dist_table " +
		"INNER JOIN ( SELECT * FROM "
	TracesExplorerViewSQLSelectAfterSubQuery = " AS inner_subquery ) AS subQuery ON dist_table.dist_traceID = subQuery.traceID " +
		"GROUP BY subQuery.traceID, subQuery.durationNano, subQuery.name, subQuery.serviceName ORDER BY subQuery.durationNano desc LIMIT 1 BY subQuery.traceID "
	TracesExplorerSpanCountWithSubQuery  = "(SELECT trace_id, count() as span_count FROM %s.%s WHERE %s %s GROUP BY trace_id ORDER BY span_count DESC LIMIT 1 BY trace_id"
	TraceExplorerSpanCountBeforeSubQuery = "SELECT serviceName, name, subQuery.span_count as span_count, durationNano, trace_id as traceID from %s.%s GLOBAL INNER JOIN ( SELECT * FROM "
	TraceExplorerSpanCountAfterSubQuery  = "AS inner_subquery ) AS subQuery ON %s.%s.trace_id = subQuery.trace_id WHERE parent_span_id = '' AND %s ORDER BY subQuery.span_count DESC"
)

// ReservedColumnTargetAliases identifies result value from a user
// written clickhouse query. The column alias indcate which value is
// to be considered as final result (or target)
var ReservedColumnTargetAliases = map[string]struct{}{
	"__result": {},
	"__value":  {},
	"result":   {},
	"res":      {},
	"value":    {},
}

// The datatype present here doesn't represent the actual datatype of column in the logs table.

var StaticFieldsLogsV3 = map[string]querytypes.AttributeKey{
	"timestamp": {},
	"id":        {},
	"trace_id": {
		Key:      "trace_id",
		DataType: querytypes.AttributeKeyDataTypeString,
		Type:     querytypes.AttributeKeyTypeUnspecified,
		IsColumn: true,
	},
	"span_id": {
		Key:      "span_id",
		DataType: querytypes.AttributeKeyDataTypeString,
		Type:     querytypes.AttributeKeyTypeUnspecified,
		IsColumn: true,
	},
	"trace_flags": {
		Key:      "trace_flags",
		DataType: querytypes.AttributeKeyDataTypeInt64,
		Type:     querytypes.AttributeKeyTypeUnspecified,
		IsColumn: true,
	},
	"severity_text": {
		Key:      "severity_text",
		DataType: querytypes.AttributeKeyDataTypeString,
		Type:     querytypes.AttributeKeyTypeUnspecified,
		IsColumn: true,
	},
	"severity_number": {
		Key:      "severity_number",
		DataType: querytypes.AttributeKeyDataTypeInt64,
		Type:     querytypes.AttributeKeyTypeUnspecified,
		IsColumn: true,
	},
	"body": {
		Key:      "body",
		DataType: querytypes.AttributeKeyDataTypeString,
		Type:     querytypes.AttributeKeyTypeUnspecified,
		IsColumn: true,
	},
	"__attrs": {
		Key:      "__attrs",
		DataType: querytypes.AttributeKeyDataTypeString,
		Type:     querytypes.AttributeKeyTypeUnspecified,
		IsColumn: true,
	},
	"scope_name": {
		Key:      "scope_name",
		DataType: querytypes.AttributeKeyDataTypeString,
		Type:     querytypes.AttributeKeyTypeUnspecified,
		IsColumn: true,
	},
	"scope_version": {
		Key:      "scope_version",
		DataType: querytypes.AttributeKeyDataTypeString,
		Type:     querytypes.AttributeKeyTypeUnspecified,
		IsColumn: true,
	},
}

// SigNozOrderByValue is a retained query-language token, not a product
// configuration or storage identifier.
const SigNozOrderByValue = "#SIGNOZ_VALUE"

const TIMESTAMP = "timestamp"

const FirstQueryGraphLimit = "first_query_graph_limit"
const SecondQueryGraphLimit = "second_query_graph_limit"

const DefaultFilterSuggestionsAttributesLimit = 50
const MaxFilterSuggestionsAttributesLimit = 100
const DefaultFilterSuggestionsExamplesLimit = 2
const MaxFilterSuggestionsExamplesLimit = 10

var SpanRenderLimitStr = GetOrDefaultEnv("SPAN_RENDER_LIMIT", "2500")
var MaxSpansInTraceStr = GetOrDefaultEnv("MAX_SPANS_IN_TRACE", "250000")

var NewStaticFieldsTraces = map[string]querytypes.AttributeKey{
	"timestamp": {},
	"trace_id": {
		Key:      "trace_id",
		DataType: querytypes.AttributeKeyDataTypeString,
		IsColumn: true,
	},
	"span_id": {
		Key:      "span_id",
		DataType: querytypes.AttributeKeyDataTypeString,
		IsColumn: true,
	},
	"trace_state": {
		Key:      "trace_state",
		DataType: querytypes.AttributeKeyDataTypeString,
		IsColumn: true,
	},
	"parent_span_id": {
		Key:      "parent_span_id",
		DataType: querytypes.AttributeKeyDataTypeString,
		IsColumn: true,
	},
	"flags": {
		Key:      "flags",
		DataType: querytypes.AttributeKeyDataTypeInt64,
		IsColumn: true,
	},
	"name": {
		Key:      "name",
		DataType: querytypes.AttributeKeyDataTypeString,
		IsColumn: true,
	},
	"kind_string": {
		Key:      "kind_string",
		DataType: querytypes.AttributeKeyDataTypeString,
		IsColumn: true,
	},
	"duration_nano": {
		Key:      "duration_nano",
		DataType: querytypes.AttributeKeyDataTypeFloat64,
		IsColumn: true,
	},
	"status_code": {
		Key:      "status_code",
		DataType: querytypes.AttributeKeyDataTypeFloat64,
		IsColumn: true,
	},
	"status_message": {
		Key:      "status_message",
		DataType: querytypes.AttributeKeyDataTypeString,
		IsColumn: true,
	},
	"status_code_string": {
		Key:      "status_code_string",
		DataType: querytypes.AttributeKeyDataTypeString,
		IsColumn: true,
	},

	// new support for composite attributes
	"response_status_code": {
		Key:      "response_status_code",
		DataType: querytypes.AttributeKeyDataTypeString,
		IsColumn: true,
	},
	"external_http_url": {
		Key:      "external_http_url",
		DataType: querytypes.AttributeKeyDataTypeString,
		IsColumn: true,
	},
	"http_url": {
		Key:      "http_url",
		DataType: querytypes.AttributeKeyDataTypeString,
		IsColumn: true,
	},
	"external_http_method": {
		Key:      "external_http_method",
		DataType: querytypes.AttributeKeyDataTypeString,
		IsColumn: true,
	},
	"http_method": {
		Key:      "http_method",
		DataType: querytypes.AttributeKeyDataTypeString,
		IsColumn: true,
	},
	"http_host": {
		Key:      "http_host",
		DataType: querytypes.AttributeKeyDataTypeString,
		IsColumn: true,
	},
	"db_name": {
		Key:      "db_name",
		DataType: querytypes.AttributeKeyDataTypeString,
		IsColumn: true,
	},
	"db_operation": {
		Key:      "db_operation",
		DataType: querytypes.AttributeKeyDataTypeString,
		IsColumn: true,
	},
	"has_error": {
		Key:      "has_error",
		DataType: querytypes.AttributeKeyDataTypeBool,
		IsColumn: true,
	},
	"is_remote": {
		Key:      "is_remote",
		DataType: querytypes.AttributeKeyDataTypeString,
		IsColumn: true,
	},

	// these are just added so that we don't use the aliased columns
	"resource_string_service$$name": {
		Key:      "resource_string_service$$name",
		DataType: querytypes.AttributeKeyDataTypeString,
		IsColumn: true,
	},
	"attribute_string_http$$route": {
		Key:      "attribute_string_http$$route",
		DataType: querytypes.AttributeKeyDataTypeString,
		IsColumn: true,
	},
	"attribute_string_messaging$$system": {
		Key:      "attribute_string_messaging$$system",
		DataType: querytypes.AttributeKeyDataTypeString,
		IsColumn: true,
	},
	"attribute_string_messaging$$operation": {
		Key:      "attribute_string_messaging$$operation",
		DataType: querytypes.AttributeKeyDataTypeString,
		IsColumn: true,
	},
	"attribute_string_db$$system": {
		Key:      "attribute_string_db$$system",
		DataType: querytypes.AttributeKeyDataTypeString,
		IsColumn: true,
	},
	"attribute_string_rpc$$system": {
		Key:      "attribute_string_rpc$$system",
		DataType: querytypes.AttributeKeyDataTypeString,
		IsColumn: true,
	},
	"attribute_string_rpc$$service": {
		Key:      "attribute_string_rpc$$service",
		DataType: querytypes.AttributeKeyDataTypeString,
		IsColumn: true,
	},
	"attribute_string_rpc$$method": {
		Key:      "attribute_string_rpc$$method",
		DataType: querytypes.AttributeKeyDataTypeString,
		IsColumn: true,
	},
	"attribute_string_peer$$service": {
		Key:      "attribute_string_peer$$service",
		DataType: querytypes.AttributeKeyDataTypeString,
		IsColumn: true,
	},
}

var StaticFieldsTraces = map[string]querytypes.AttributeKey{}

var IsDotMetricsEnabled = false
var MaxJSONFlatteningDepth = 1

func init() {
	StaticFieldsTraces = maps.Clone(NewStaticFieldsTraces)
	if GetOrDefaultEnv(DotMetricsEnabled, "true") == "true" {
		IsDotMetricsEnabled = true
	}

	// set max flattening depth
	depth, err := strconv.Atoi(GetOrDefaultEnv(maxJSONFlatteningDepth, "1"))
	if err == nil {
		MaxJSONFlatteningDepth = depth
	}
}

const TRACE_V4_MAX_PAGINATION_LIMIT = 10000

const MaxResultRowsForCHQuery = 1_000_000

var ChDataTypeMap = map[string]string{
	"string":  "String",
	"bool":    "Bool",
	"int64":   "Float64",
	"float64": "Float64",
}

var MaterializedDataTypeMap = map[string]string{
	"string":  "string",
	"bool":    "bool",
	"int64":   "number",
	"float64": "number",
}

const InspectMetricsMaxTimeDiff = 1800000

const DotMetricsEnabled = "DOT_METRICS_ENABLED"
const maxJSONFlatteningDepth = "MAX_JSON_FLATTENING_DEPTH"
