package clickhouseReader

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"math"
	"math/rand"
	"reflect"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/uptrace/bun"

	"github.com/SigNoz/signoz/pkg/prometheus"
	"github.com/SigNoz/signoz/pkg/query-service/model/metrics_explorer"
	"github.com/SigNoz/signoz/pkg/sqlstore"
	"github.com/SigNoz/signoz/pkg/telemetrystore"
	"github.com/SigNoz/signoz/pkg/types"
	"github.com/SigNoz/signoz/pkg/types/ctxtypes"
	"github.com/SigNoz/signoz/pkg/types/instrumentationtypes"
	"github.com/SigNoz/signoz/pkg/types/telemetrytypes"
	"github.com/SigNoz/signoz/pkg/types/timeseriestypes"
	"github.com/SigNoz/signoz/pkg/valuer"

	"github.com/google/uuid"
	"github.com/pkg/errors"

	errorsV2 "github.com/SigNoz/signoz/pkg/errors"

	"github.com/ClickHouse/clickhouse-go/v2"
	"github.com/ClickHouse/clickhouse-go/v2/lib/driver"

	"github.com/SigNoz/signoz/pkg/cache"

	"log/slog"

	"github.com/SigNoz/signoz/pkg/query-service/app/traces/tracedetail"
	"github.com/SigNoz/signoz/pkg/query-service/common"
	"github.com/SigNoz/signoz/pkg/query-service/constants"

	chErrors "github.com/SigNoz/signoz/pkg/query-service/errors"
	"github.com/SigNoz/signoz/pkg/query-service/model"
	"github.com/SigNoz/signoz/pkg/query-service/model/querytypes"
	"github.com/SigNoz/signoz/pkg/query-service/utils"
)

const (
	primaryNamespace          = "clickhouse"
	archiveNamespace          = "clickhouse-archive"
	signozTraceDBName         = "signoz_traces"
	signozHistoryDBName       = "signoz_analytics"
	ruleStateHistoryTableName = "rule_state_history_v0"
	signozDurationMVTable     = "durationSort"
	signozUsageExplorerTable  = "usage_explorer"
	signozSpansTable          = "signoz_spans"
	signozErrorIndexTable     = "signoz_error_index_v2"
	signozTraceTableName      = "signoz_index_v2"
	signozTraceLocalTableName = "signoz_index_v2"
	signozMetricDBName        = "signoz_metrics"
	signozMetadataDbName      = "signoz_metadata"
	signozMeterDBName         = "signoz_meter"
	signozMeterSamplesName    = "samples_agg_1d"

	signozSampleLocalTableName = "samples_v4"
	signozSampleTableName      = "samples_v4"

	signozSamplesAgg5mLocalTableName = "samples_v4_agg_5m"
	signozSamplesAgg5mTableName      = "samples_v4_agg_5m"

	signozSamplesAgg30mLocalTableName = "samples_v4_agg_30m"
	signozSamplesAgg30mTableName      = "samples_v4_agg_30m"

	signozExpHistLocalTableName = "exp_hist"
	signozExpHistTableName      = "exp_hist"

	signozTSLocalTableNameV4 = "time_series_v4"
	signozTSTableNameV4      = "time_series_v4"

	signozTSLocalTableNameV46Hrs = "time_series_v4_6hrs"
	signozTSTableNameV46Hrs      = "time_series_v4_6hrs"

	signozTSLocalTableNameV41Day = "time_series_v4_1day"
	signozTSTableNameV41Day      = "time_series_v4_1day"

	signozTSLocalTableNameV41Week = "time_series_v4_1week"
	signozTSTableNameV41Week      = "time_series_v4_1week"

	signozTableAttributesMetadata      = "attributes_metadata"
	signozLocalTableAttributesMetadata = "attributes_metadata"

	signozUpdatedMetricsMetadataLocalTable = "updated_metadata"
	signozUpdatedMetricsMetadataTable      = "updated_metadata"
	minTimespanForProgressiveSearch        = time.Hour
	minTimespanForProgressiveSearchMargin  = time.Minute
	maxProgressiveSteps                    = 4
	charset                                = "abcdefghijklmnopqrstuvwxyz" +
		"ABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	NANOSECOND = 1000000000
)

var (
	ErrNoOperationsTable            = errors.New("no operations table supplied")
	ErrNoIndexTable                 = errors.New("no index table supplied")
	ErrStartTimeRequired            = errors.New("start time is required for search queries")
	seededRand           *rand.Rand = rand.New(
		rand.NewSource(time.Now().UnixNano()))
)

// SpanWriter for reading spans from ClickHouse
type ClickHouseReader struct {
	db                      clickhouse.Conn
	prometheus              prometheus.Prometheus
	sqlDB                   sqlstore.SQLStore
	TraceDB                 string
	durationTable           string
	indexTable              string
	errorTable              string
	SpansTable              string
	spanAttributeTableV2    string
	spanAttributesKeysTable string
	dependencyGraphTable    string
	topLevelOperationsTable string
	logsDB                  string
	logsAttributeKeys       string
	logsResourceKeys        string
	logsTagAttributeTableV2 string
	logger                  *slog.Logger

	logsTableV2              string
	logsLocalTableV2         string
	logsResourceTableV2      string
	logsResourceLocalTableV2 string

	liveTailRefreshSeconds int

	logsTableName      string
	logsLocalTableName string

	traceTableName       string
	traceLocalTableName  string
	traceResourceTableV3 string
	traceSummaryTable    string

	fluxIntervalForTraceDetail time.Duration
	cache                      cache.Cache
	cacheForTraceDetail        cache.Cache
	metadataDB                 string
	metadataTable              string
}

// NewTraceReader returns a TraceReader for the database
func NewReader(
	logger *slog.Logger,
	sqlDB sqlstore.SQLStore,
	telemetryStore telemetrystore.TelemetryStore,
	prometheus prometheus.Prometheus,
	fluxIntervalForTraceDetail time.Duration,
	cacheForTraceDetail cache.Cache,
	cache cache.Cache,
	options *Options,
) *ClickHouseReader {
	if options == nil {
		options = NewOptions(primaryNamespace, archiveNamespace)
	}

	logsTableName := options.primary.LogsTableV2
	logsLocalTableName := options.primary.LogsLocalTableV2
	traceTableName := options.primary.TraceIndexTableV3
	traceLocalTableName := options.primary.TraceLocalTableNameV3

	return &ClickHouseReader{
		db:                         telemetryStore.ClickhouseDB(),
		logger:                     logger,
		prometheus:                 prometheus,
		sqlDB:                      sqlDB,
		TraceDB:                    options.primary.TraceDB,
		indexTable:                 options.primary.IndexTable,
		errorTable:                 options.primary.ErrorTable,
		durationTable:              options.primary.DurationTable,
		SpansTable:                 options.primary.SpansTable,
		spanAttributeTableV2:       options.primary.SpanAttributeTableV2,
		spanAttributesKeysTable:    options.primary.SpanAttributeKeysTable,
		dependencyGraphTable:       options.primary.DependencyGraphTable,
		topLevelOperationsTable:    options.primary.TopLevelOperationsTable,
		logsDB:                     options.primary.LogsDB,
		logsAttributeKeys:          options.primary.LogsAttributeKeysTable,
		logsResourceKeys:           options.primary.LogsResourceKeysTable,
		logsTagAttributeTableV2:    options.primary.LogsTagAttributeTableV2,
		liveTailRefreshSeconds:     options.primary.LiveTailRefreshSeconds,
		logsTableV2:                options.primary.LogsTableV2,
		logsLocalTableV2:           options.primary.LogsLocalTableV2,
		logsResourceTableV2:        options.primary.LogsResourceTableV2,
		logsResourceLocalTableV2:   options.primary.LogsResourceLocalTableV2,
		logsTableName:              logsTableName,
		logsLocalTableName:         logsLocalTableName,
		traceLocalTableName:        traceLocalTableName,
		traceTableName:             traceTableName,
		traceResourceTableV3:       options.primary.TraceResourceTableV3,
		traceSummaryTable:          options.primary.TraceSummaryTable,
		fluxIntervalForTraceDetail: fluxIntervalForTraceDetail,
		cache:                      cache,
		cacheForTraceDetail:        cacheForTraceDetail,
		metadataDB:                 options.primary.MetadataDB,
		metadataTable:              options.primary.MetadataTable,
	}
}

func (r *ClickHouseReader) GetTopLevelOperations(ctx context.Context, start, end time.Time, services []string) (*map[string][]string, *model.ApiError) {
	ctx = ctxtypes.NewContextWithCommentVals(ctx, map[string]string{
		instrumentationtypes.TelemetrySignal:  telemetrytypes.SignalTraces.StringValue(),
		instrumentationtypes.CodeNamespace:    "clickhouse-reader",
		instrumentationtypes.CodeFunctionName: "GetTopLevelOperations",
	})

	start = start.In(time.UTC)

	// The `top_level_operations` that have `time` >= start
	operations := map[string][]string{}
	// We can't use the `end` because the `top_level_operations` table has the most recent instances of the operations
	// We can only use the `start` time to filter the operations
	query := fmt.Sprintf(`SELECT name, serviceName, max(time) as ts FROM %s.%s WHERE time >= @start`, r.TraceDB, r.topLevelOperationsTable)
	if len(services) > 0 {
		query += ` AND serviceName IN @services`
	}
	query += ` GROUP BY name, serviceName ORDER BY ts DESC LIMIT 5000`

	rows, err := r.db.Query(ctx, query, clickhouse.Named("start", start), clickhouse.Named("services", services))

	if err != nil {
		r.logger.Error("Error in processing sql query", errorsV2.Attr(err))
		return nil, &model.ApiError{Typ: model.ErrorExec, Err: fmt.Errorf("error in processing sql query")}
	}

	defer rows.Close()
	for rows.Next() {
		var name, serviceName string
		var t time.Time
		if err := rows.Scan(&name, &serviceName, &t); err != nil {
			return nil, &model.ApiError{Typ: model.ErrorInternal, Err: fmt.Errorf("error in reading data")}
		}
		if _, ok := operations[serviceName]; !ok {
			operations[serviceName] = []string{"overflow_operation"}
		}
		operations[serviceName] = append(operations[serviceName], name)
	}
	return &operations, nil
}

func createTagQueryFromTagQueryParams(queryParams []model.TagQueryParam) []model.TagQuery {
	tags := []model.TagQuery{}
	for _, tag := range queryParams {
		if len(tag.StringValues) > 0 {
			tags = append(tags, model.NewTagQueryString(tag))
		}
		if len(tag.NumberValues) > 0 {
			tags = append(tags, model.NewTagQueryNumber(tag))
		}
		if len(tag.BoolValues) > 0 {
			tags = append(tags, model.NewTagQueryBool(tag))
		}
	}
	return tags
}

func StringWithCharset(length int, charset string) string {
	b := make([]byte, length)
	for i := range b {
		b[i] = charset[seededRand.Intn(len(charset))]
	}
	return string(b)
}

func String(length int) string {
	return StringWithCharset(length, charset)
}

func buildQueryWithTagParams(_ context.Context, tags []model.TagQuery) (string, []interface{}, *model.ApiError) {
	query := ""
	var args []interface{}
	for _, item := range tags {
		var subQuery string
		var argsSubQuery []interface{}
		tagMapType := item.GetTagMapColumn()
		switch item.GetOperator() {
		case model.EqualOperator:
			subQuery, argsSubQuery = addArithmeticOperator(item, tagMapType, "=")
		case model.NotEqualOperator:
			subQuery, argsSubQuery = addArithmeticOperator(item, tagMapType, "!=")
		case model.LessThanOperator:
			subQuery, argsSubQuery = addArithmeticOperator(item, tagMapType, "<")
		case model.GreaterThanOperator:
			subQuery, argsSubQuery = addArithmeticOperator(item, tagMapType, ">")
		case model.InOperator:
			subQuery, argsSubQuery = addInOperator(item, tagMapType, false)
		case model.NotInOperator:
			subQuery, argsSubQuery = addInOperator(item, tagMapType, true)
		case model.LessThanEqualOperator:
			subQuery, argsSubQuery = addArithmeticOperator(item, tagMapType, "<=")
		case model.GreaterThanEqualOperator:
			subQuery, argsSubQuery = addArithmeticOperator(item, tagMapType, ">=")
		case model.ContainsOperator:
			subQuery, argsSubQuery = addContainsOperator(item, tagMapType, false)
		case model.NotContainsOperator:
			subQuery, argsSubQuery = addContainsOperator(item, tagMapType, true)
		case model.StartsWithOperator:
			subQuery, argsSubQuery = addStartsWithOperator(item, tagMapType, false)
		case model.NotStartsWithOperator:
			subQuery, argsSubQuery = addStartsWithOperator(item, tagMapType, true)
		case model.ExistsOperator:
			subQuery, argsSubQuery = addExistsOperator(item, tagMapType, false)
		case model.NotExistsOperator:
			subQuery, argsSubQuery = addExistsOperator(item, tagMapType, true)
		default:
			return "", nil, &model.ApiError{Typ: model.ErrorExec, Err: fmt.Errorf("filter operator %s not supported", item.GetOperator())}
		}
		query += subQuery
		args = append(args, argsSubQuery...)
	}
	return query, args, nil
}

func addInOperator(item model.TagQuery, tagMapType string, not bool) (string, []interface{}) {
	values := item.GetValues()
	args := []interface{}{}
	notStr := ""
	if not {
		notStr = "NOT"
	}
	tagValuePair := []string{}
	for _, value := range values {
		tagKey := "inTagKey" + String(5)
		tagValue := "inTagValue" + String(5)
		tagValuePair = append(tagValuePair, fmt.Sprintf("%s[@%s] = @%s", tagMapType, tagKey, tagValue))
		args = append(args, clickhouse.Named(tagKey, item.GetKey()))
		args = append(args, clickhouse.Named(tagValue, value))
	}
	return fmt.Sprintf(" AND %s (%s)", notStr, strings.Join(tagValuePair, " OR ")), args
}

func addContainsOperator(item model.TagQuery, tagMapType string, not bool) (string, []interface{}) {
	values := item.GetValues()
	args := []interface{}{}
	notStr := ""
	if not {
		notStr = "NOT"
	}
	tagValuePair := []string{}
	for _, value := range values {
		tagKey := "containsTagKey" + String(5)
		tagValue := "containsTagValue" + String(5)
		tagValuePair = append(tagValuePair, fmt.Sprintf("%s[@%s] ILIKE @%s", tagMapType, tagKey, tagValue))
		args = append(args, clickhouse.Named(tagKey, item.GetKey()))
		args = append(args, clickhouse.Named(tagValue, "%"+fmt.Sprintf("%v", value)+"%"))
	}
	return fmt.Sprintf(" AND %s (%s)", notStr, strings.Join(tagValuePair, " OR ")), args
}

func addStartsWithOperator(item model.TagQuery, tagMapType string, not bool) (string, []interface{}) {
	values := item.GetValues()
	args := []interface{}{}
	notStr := ""
	if not {
		notStr = "NOT"
	}
	tagValuePair := []string{}
	for _, value := range values {
		tagKey := "startsWithTagKey" + String(5)
		tagValue := "startsWithTagValue" + String(5)
		tagValuePair = append(tagValuePair, fmt.Sprintf("%s[@%s] ILIKE @%s", tagMapType, tagKey, tagValue))
		args = append(args, clickhouse.Named(tagKey, item.GetKey()))
		args = append(args, clickhouse.Named(tagValue, "%"+fmt.Sprintf("%v", value)+"%"))
	}
	return fmt.Sprintf(" AND %s (%s)", notStr, strings.Join(tagValuePair, " OR ")), args
}

func addArithmeticOperator(item model.TagQuery, tagMapType string, operator string) (string, []interface{}) {
	values := item.GetValues()
	args := []interface{}{}
	tagValuePair := []string{}
	for _, value := range values {
		tagKey := "arithmeticTagKey" + String(5)
		tagValue := "arithmeticTagValue" + String(5)
		tagValuePair = append(tagValuePair, fmt.Sprintf("%s[@%s] %s @%s", tagMapType, tagKey, operator, tagValue))
		args = append(args, clickhouse.Named(tagKey, item.GetKey()))
		args = append(args, clickhouse.Named(tagValue, value))
	}
	return fmt.Sprintf(" AND (%s)", strings.Join(tagValuePair, " OR ")), args
}

func addExistsOperator(item model.TagQuery, tagMapType string, not bool) (string, []interface{}) {
	values := item.GetValues()
	notStr := ""
	if not {
		notStr = "NOT"
	}
	args := []interface{}{}
	tagOperatorPair := []string{}
	for range values {
		tagKey := "existsTagKey" + String(5)
		tagOperatorPair = append(tagOperatorPair, fmt.Sprintf("mapContains(%s, @%s)", tagMapType, tagKey))
		args = append(args, clickhouse.Named(tagKey, item.GetKey()))
	}
	return fmt.Sprintf(" AND %s (%s)", notStr, strings.Join(tagOperatorPair, " OR ")), args
}

func (r *ClickHouseReader) GetSpansForTrace(ctx context.Context, traceID string, traceDetailsQuery string) ([]model.SpanItemV2, *model.ApiError) {

	ctx = ctxtypes.NewContextWithCommentVals(ctx, map[string]string{
		instrumentationtypes.TelemetrySignal:  telemetrytypes.SignalTraces.StringValue(),
		instrumentationtypes.CodeNamespace:    "clickhouse-reader",
		instrumentationtypes.CodeFunctionName: "GetSpansForTrace",
	})

	var traceSummary model.TraceSummary
	summaryQuery := fmt.Sprintf("SELECT trace_id, min(start) AS start, max(end) AS end, sum(num_spans) AS num_spans FROM %s.%s WHERE trace_id=$1 GROUP BY trace_id", r.TraceDB, r.traceSummaryTable)
	err := r.db.QueryRow(ctx, summaryQuery, traceID).Scan(&traceSummary.TraceID, &traceSummary.Start, &traceSummary.End, &traceSummary.NumSpans)
	if err != nil {
		if err == sql.ErrNoRows {
			return []model.SpanItemV2{}, nil
		}
		r.logger.Error("Error in processing trace summary sql query", errorsV2.Attr(err))
		return nil, model.ExecutionError(fmt.Errorf("error in processing trace summary sql query: %w", err))
	}

	var searchScanResponses []model.SpanItemV2
	queryStartTime := time.Now()
	err = r.db.Select(ctx, &searchScanResponses, traceDetailsQuery, traceID, strconv.FormatInt(traceSummary.Start.Unix()-1800, 10), strconv.FormatInt(traceSummary.End.Unix(), 10))
	r.logger.Info(traceDetailsQuery)
	if err != nil {
		r.logger.Error("Error in processing sql query", errorsV2.Attr(err))
		return nil, model.ExecutionError(fmt.Errorf("error in processing trace data sql query: %w", err))
	}
	r.logger.Info("trace details query took: ", "duration", time.Since(queryStartTime), "traceID", traceID)

	return searchScanResponses, nil
}

func (r *ClickHouseReader) GetWaterfallSpansForTraceWithMetadataCache(ctx context.Context, orgID valuer.UUID, traceID string) (*model.GetWaterfallSpansForTraceWithMetadataCache, error) {
	cachedTraceData := new(model.GetWaterfallSpansForTraceWithMetadataCache)
	err := r.cacheForTraceDetail.Get(ctx, orgID, strings.Join([]string{"getWaterfallSpansForTraceWithMetadata", traceID}, "-"), cachedTraceData)
	if err != nil {
		r.logger.Debug("error in retrieving getWaterfallSpansForTraceWithMetadata cache", errorsV2.Attr(err), "traceID", traceID)
		return nil, err
	}

	if time.Since(time.UnixMilli(int64(cachedTraceData.EndTime))) < r.fluxIntervalForTraceDetail {
		r.logger.Info("the trace end time falls under the flux interval, skipping getWaterfallSpansForTraceWithMetadata cache", "traceID", traceID)
		return nil, errors.Errorf("the trace end time falls under the flux interval, skipping getWaterfallSpansForTraceWithMetadata cache, traceID: %s", traceID)
	}

	r.logger.Info("cache is successfully hit, applying cache for getWaterfallSpansForTraceWithMetadata", "traceID", traceID)
	return cachedTraceData, nil
}

func (r *ClickHouseReader) GetWaterfallSpansForTraceWithMetadata(ctx context.Context, orgID valuer.UUID, traceID string, req *model.GetWaterfallSpansForTraceWithMetadataParams) (*model.GetWaterfallSpansForTraceWithMetadataResponse, error) {
	response := new(model.GetWaterfallSpansForTraceWithMetadataResponse)
	var startTime, endTime, durationNano, totalErrorSpans, totalSpans uint64
	var spanIdToSpanNodeMap = map[string]*model.Span{}
	var traceRoots []*model.Span
	var serviceNameToTotalDurationMap = map[string]uint64{}
	var serviceNameIntervalMap = map[string][]tracedetail.Interval{}
	var hasMissingSpans bool

	cachedTraceData, err := r.GetWaterfallSpansForTraceWithMetadataCache(ctx, orgID, traceID)
	if err == nil {
		startTime = cachedTraceData.StartTime
		endTime = cachedTraceData.EndTime
		durationNano = cachedTraceData.DurationNano
		spanIdToSpanNodeMap = cachedTraceData.SpanIdToSpanNodeMap
		serviceNameToTotalDurationMap = cachedTraceData.ServiceNameToTotalDurationMap
		traceRoots = cachedTraceData.TraceRoots
		totalSpans = cachedTraceData.TotalSpans
		totalErrorSpans = cachedTraceData.TotalErrorSpans
		hasMissingSpans = cachedTraceData.HasMissingSpans
	}

	if err != nil {
		r.logger.Info("cache miss for getWaterfallSpansForTraceWithMetadata", "traceID", traceID)

		searchScanResponses, err := r.GetSpansForTrace(ctx, traceID, fmt.Sprintf("SELECT DISTINCT ON (span_id) timestamp, duration_nano, span_id, trace_id, has_error, kind, resource_string_service$$name, name, links as references, attributes_string, attributes_number, attributes_bool, resources_string, events, status_message, status_code_string, kind_string FROM %s.%s WHERE trace_id=$1 and ts_bucket_start>=$2 and ts_bucket_start<=$3 ORDER BY timestamp ASC, name ASC", r.TraceDB, r.traceTableName))
		if err != nil {
			return nil, err
		}
		if len(searchScanResponses) == 0 {
			return response, nil
		}
		totalSpans = uint64(len(searchScanResponses))
		processingBeforeCache := time.Now()
		for _, item := range searchScanResponses {
			ref := []model.OtelSpanRef{}
			err := json.Unmarshal([]byte(item.References), &ref)
			if err != nil {
				r.logger.Error("getWaterfallSpansForTraceWithMetadata: error unmarshalling references", errorsV2.Attr(err), "traceID", traceID)
				return nil, errorsV2.Newf(errorsV2.TypeInvalidInput, errorsV2.CodeInvalidInput, "getWaterfallSpansForTraceWithMetadata: error unmarshalling references %s", err.Error())
			}

			// merge attributes_number and attributes_bool to attributes_string
			for k, v := range item.Attributes_bool {
				item.Attributes_string[k] = fmt.Sprintf("%v", v)
			}
			for k, v := range item.Attributes_number {
				item.Attributes_string[k] = strconv.FormatFloat(v, 'f', -1, 64)
			}
			for k, v := range item.Resources_string {
				item.Attributes_string[k] = v
			}

			events := make([]model.Event, 0)
			for _, event := range item.Events {
				var eventMap model.Event
				err = json.Unmarshal([]byte(event), &eventMap)
				if err != nil {
					r.logger.Error("Error unmarshalling events", errorsV2.Attr(err))
					return nil, errorsV2.Newf(errorsV2.TypeInternal, errorsV2.CodeInternal, "getWaterfallSpansForTraceWithMetadata: error in unmarshalling events %s", err.Error())
				}
				events = append(events, eventMap)
			}

			startTimeUnixNano := uint64(item.TimeUnixNano.UnixNano())

			jsonItem := model.Span{
				SpanID:           item.SpanID,
				TraceID:          item.TraceID,
				ServiceName:      item.ServiceName,
				Name:             item.Name,
				Kind:             int32(item.Kind),
				DurationNano:     item.DurationNano,
				HasError:         item.HasError,
				StatusMessage:    item.StatusMessage,
				StatusCodeString: item.StatusCodeString,
				SpanKind:         item.SpanKind,
				References:       ref,
				Events:           events,
				TagMap:           item.Attributes_string,
				Children:         make([]*model.Span, 0),
				TimeUnixNano:     startTimeUnixNano, // Store nanoseconds temporarily
			}

			// metadata calculation
			if startTime == 0 || startTimeUnixNano < startTime {
				startTime = startTimeUnixNano
			}
			if endTime == 0 || (startTimeUnixNano+jsonItem.DurationNano) > endTime {
				endTime = (startTimeUnixNano + jsonItem.DurationNano)
			}
			if durationNano == 0 || jsonItem.DurationNano > durationNano {
				durationNano = jsonItem.DurationNano
			}

			if jsonItem.HasError {
				totalErrorSpans = totalErrorSpans + 1
			}

			// collect the intervals for service for execution time calculation
			serviceNameIntervalMap[jsonItem.ServiceName] =
				append(serviceNameIntervalMap[jsonItem.ServiceName], tracedetail.Interval{StartTime: jsonItem.TimeUnixNano, Duration: jsonItem.DurationNano, Service: jsonItem.ServiceName})

			// append to the span node map
			spanIdToSpanNodeMap[jsonItem.SpanID] = &jsonItem
		}

		// traverse through the map and append each node to the children array of the parent node
		// and add the missing spans
		for _, spanNode := range spanIdToSpanNodeMap {
			hasParentSpanNode := false
			for _, reference := range spanNode.References {
				if reference.RefType == "CHILD_OF" && reference.SpanId != "" {
					hasParentSpanNode = true

					if parentNode, exists := spanIdToSpanNodeMap[reference.SpanId]; exists {
						parentNode.Children = append(parentNode.Children, spanNode)
					} else {
						// insert the missing span
						missingSpan := model.Span{
							SpanID:           reference.SpanId,
							TraceID:          spanNode.TraceID,
							ServiceName:      "",
							Name:             "Missing Span",
							TimeUnixNano:     spanNode.TimeUnixNano,
							Kind:             0,
							DurationNano:     spanNode.DurationNano,
							HasError:         false,
							StatusMessage:    "",
							StatusCodeString: "",
							SpanKind:         "",
							Events:           make([]model.Event, 0),
							Children:         make([]*model.Span, 0),
						}
						missingSpan.Children = append(missingSpan.Children, spanNode)
						spanIdToSpanNodeMap[missingSpan.SpanID] = &missingSpan
						traceRoots = append(traceRoots, &missingSpan)
						hasMissingSpans = true
					}
				}
			}
			if !hasParentSpanNode && !tracedetail.ContainsWaterfallSpan(traceRoots, spanNode) {
				traceRoots = append(traceRoots, spanNode)
			}
		}

		// sort the trace roots to add missing spans at the right order
		sort.Slice(traceRoots, func(i, j int) bool {
			if traceRoots[i].TimeUnixNano == traceRoots[j].TimeUnixNano {
				return traceRoots[i].Name < traceRoots[j].Name
			}
			return traceRoots[i].TimeUnixNano < traceRoots[j].TimeUnixNano
		})

		serviceNameToTotalDurationMap = tracedetail.CalculateServiceTime(serviceNameIntervalMap)

		traceCache := model.GetWaterfallSpansForTraceWithMetadataCache{
			StartTime:                     startTime,
			EndTime:                       endTime,
			DurationNano:                  durationNano,
			TotalSpans:                    totalSpans,
			TotalErrorSpans:               totalErrorSpans,
			SpanIdToSpanNodeMap:           spanIdToSpanNodeMap,
			ServiceNameToTotalDurationMap: serviceNameToTotalDurationMap,
			TraceRoots:                    traceRoots,
			HasMissingSpans:               hasMissingSpans,
		}

		r.logger.Info("getWaterfallSpansForTraceWithMetadata: processing pre cache", "duration", time.Since(processingBeforeCache), "traceID", traceID)
		cacheErr := r.cacheForTraceDetail.Set(ctx, orgID, strings.Join([]string{"getWaterfallSpansForTraceWithMetadata", traceID}, "-"), &traceCache, time.Minute*5)
		if cacheErr != nil {
			r.logger.Debug("failed to store cache for getWaterfallSpansForTraceWithMetadata", "traceID", traceID, errorsV2.Attr(err))
		}
	}

	processingPostCache := time.Now()
	selectedSpans, uncollapsedSpans, rootServiceName, rootServiceEntryPoint := tracedetail.GetSelectedSpans(req.UncollapsedSpans, req.SelectedSpanID, traceRoots, spanIdToSpanNodeMap, req.IsSelectedSpanIDUnCollapsed)
	r.logger.Info("getWaterfallSpansForTraceWithMetadata: processing post cache", "duration", time.Since(processingPostCache), "traceID", traceID)

	// convert start timestamp to millis because right now frontend is expecting it in millis
	for _, span := range selectedSpans {
		span.TimeUnixNano = span.TimeUnixNano / 1000000
	}

	for serviceName, totalDuration := range serviceNameToTotalDurationMap {
		serviceNameToTotalDurationMap[serviceName] = totalDuration / 1000000
	}

	response.Spans = selectedSpans
	response.UncollapsedSpans = uncollapsedSpans
	response.StartTimestampMillis = startTime / 1000000
	response.EndTimestampMillis = endTime / 1000000
	response.TotalSpansCount = totalSpans
	response.TotalErrorSpansCount = totalErrorSpans
	response.RootServiceName = rootServiceName
	response.RootServiceEntryPoint = rootServiceEntryPoint
	response.ServiceNameToTotalDurationMap = serviceNameToTotalDurationMap
	response.HasMissingSpans = hasMissingSpans
	return response, nil
}

func (r *ClickHouseReader) GetFlamegraphSpansForTraceCache(ctx context.Context, orgID valuer.UUID, traceID string) (*model.GetFlamegraphSpansForTraceCache, error) {
	cachedTraceData := new(model.GetFlamegraphSpansForTraceCache)
	err := r.cacheForTraceDetail.Get(ctx, orgID, strings.Join([]string{"getFlamegraphSpansForTrace", traceID}, "-"), cachedTraceData)
	if err != nil {
		r.logger.Debug("error in retrieving getFlamegraphSpansForTrace cache", errorsV2.Attr(err), "traceID", traceID)
		return nil, err
	}

	if time.Since(time.UnixMilli(int64(cachedTraceData.EndTime))) < r.fluxIntervalForTraceDetail {
		r.logger.Info("the trace end time falls under the flux interval, skipping getFlamegraphSpansForTrace cache", "traceID", traceID)
		return nil, errors.Errorf("the trace end time falls under the flux interval, skipping getFlamegraphSpansForTrace cache, traceID: %s", traceID)
	}

	r.logger.Info("cache is successfully hit, applying cache for getFlamegraphSpansForTrace", "traceID", traceID)
	return cachedTraceData, nil
}

func (r *ClickHouseReader) GetFlamegraphSpansForTrace(ctx context.Context, orgID valuer.UUID, traceID string, req *model.GetFlamegraphSpansForTraceParams) (*model.GetFlamegraphSpansForTraceResponse, error) {
	trace := new(model.GetFlamegraphSpansForTraceResponse)
	var startTime, endTime, durationNano uint64
	var spanIdToSpanNodeMap = map[string]*model.FlamegraphSpan{}
	// map[traceID][level]span
	var selectedSpans = [][]*model.FlamegraphSpan{}
	var traceRoots []*model.FlamegraphSpan

	// get the trace tree from cache!
	cachedTraceData, err := r.GetFlamegraphSpansForTraceCache(ctx, orgID, traceID)

	if err == nil {
		startTime = cachedTraceData.StartTime
		endTime = cachedTraceData.EndTime
		durationNano = cachedTraceData.DurationNano
		selectedSpans = cachedTraceData.SelectedSpans
		traceRoots = cachedTraceData.TraceRoots
	}

	if err != nil {
		r.logger.Info("cache miss for getFlamegraphSpansForTrace", "traceID", traceID)

		searchScanResponses, err := r.GetSpansForTrace(ctx, traceID, fmt.Sprintf("SELECT timestamp, duration_nano, span_id, trace_id, has_error,links as references, resource_string_service$$name, name, events FROM %s.%s WHERE trace_id=$1 and ts_bucket_start>=$2 and ts_bucket_start<=$3 ORDER BY timestamp ASC, name ASC", r.TraceDB, r.traceTableName))
		if err != nil {
			return nil, err
		}
		if len(searchScanResponses) == 0 {
			return trace, nil
		}

		processingBeforeCache := time.Now()
		for _, item := range searchScanResponses {
			ref := []model.OtelSpanRef{}
			err := json.Unmarshal([]byte(item.References), &ref)
			if err != nil {
				r.logger.Error("Error unmarshalling references", errorsV2.Attr(err))
				return nil, errorsV2.Newf(errorsV2.TypeInternal, errorsV2.CodeInternal, "getFlamegraphSpansForTrace: error in unmarshalling references %s", err.Error())
			}

			events := make([]model.Event, 0)
			for _, event := range item.Events {
				var eventMap model.Event
				err = json.Unmarshal([]byte(event), &eventMap)
				if err != nil {
					r.logger.Error("Error unmarshalling events", errorsV2.Attr(err))
					return nil, errorsV2.Newf(errorsV2.TypeInternal, errorsV2.CodeInternal, "getFlamegraphSpansForTrace: error in unmarshalling events %s", err.Error())
				}
				events = append(events, eventMap)
			}

			jsonItem := model.FlamegraphSpan{
				SpanID:       item.SpanID,
				TraceID:      item.TraceID,
				ServiceName:  item.ServiceName,
				Name:         item.Name,
				DurationNano: item.DurationNano,
				HasError:     item.HasError,
				References:   ref,
				Events:       events,
				Children:     make([]*model.FlamegraphSpan, 0),
			}

			// metadata calculation
			startTimeUnixNano := uint64(item.TimeUnixNano.UnixNano())
			if startTime == 0 || startTimeUnixNano < startTime {
				startTime = startTimeUnixNano
			}
			if endTime == 0 || (startTimeUnixNano+jsonItem.DurationNano) > endTime {
				endTime = (startTimeUnixNano + jsonItem.DurationNano)
			}
			if durationNano == 0 || jsonItem.DurationNano > durationNano {
				durationNano = jsonItem.DurationNano
			}

			jsonItem.TimeUnixNano = uint64(item.TimeUnixNano.UnixNano() / 1000000)
			spanIdToSpanNodeMap[jsonItem.SpanID] = &jsonItem
		}

		// traverse through the map and append each node to the children array of the parent node
		// and add missing spans
		for _, spanNode := range spanIdToSpanNodeMap {
			hasParentSpanNode := false
			for _, reference := range spanNode.References {
				if reference.RefType == "CHILD_OF" && reference.SpanId != "" {
					hasParentSpanNode = true
					if parentNode, exists := spanIdToSpanNodeMap[reference.SpanId]; exists {
						parentNode.Children = append(parentNode.Children, spanNode)
					} else {
						// insert the missing spans
						missingSpan := model.FlamegraphSpan{
							SpanID:       reference.SpanId,
							TraceID:      spanNode.TraceID,
							ServiceName:  "",
							Name:         "Missing Span",
							TimeUnixNano: spanNode.TimeUnixNano,
							DurationNano: spanNode.DurationNano,
							HasError:     false,
							Events:       make([]model.Event, 0),
							Children:     make([]*model.FlamegraphSpan, 0),
						}
						missingSpan.Children = append(missingSpan.Children, spanNode)
						spanIdToSpanNodeMap[missingSpan.SpanID] = &missingSpan
						traceRoots = append(traceRoots, &missingSpan)
					}
				}
			}
			if !hasParentSpanNode && !tracedetail.ContainsFlamegraphSpan(traceRoots, spanNode) {
				traceRoots = append(traceRoots, spanNode)
			}
		}

		selectedSpans = tracedetail.GetSelectedSpansForFlamegraph(traceRoots, spanIdToSpanNodeMap)
		traceCache := model.GetFlamegraphSpansForTraceCache{
			StartTime:     startTime,
			EndTime:       endTime,
			DurationNano:  durationNano,
			SelectedSpans: selectedSpans,
			TraceRoots:    traceRoots,
		}

		r.logger.Info("getFlamegraphSpansForTrace: processing pre cache", "duration", time.Since(processingBeforeCache), "traceID", traceID)
		cacheErr := r.cacheForTraceDetail.Set(ctx, orgID, strings.Join([]string{"getFlamegraphSpansForTrace", traceID}, "-"), &traceCache, time.Minute*5)
		if cacheErr != nil {
			r.logger.Debug("failed to store cache for getFlamegraphSpansForTrace", "traceID", traceID, errorsV2.Attr(err))
		}
	}

	processingPostCache := time.Now()
	selectedSpansForRequest := tracedetail.GetSelectedSpansForFlamegraphForRequest(req.SelectedSpanID, selectedSpans, startTime, endTime)
	r.logger.Info("getFlamegraphSpansForTrace: processing post cache", "duration", time.Since(processingPostCache), "traceID", traceID)

	trace.Spans = selectedSpansForRequest
	trace.StartTimestampMillis = startTime / 1000000
	trace.EndTimestampMillis = endTime / 1000000
	return trace, nil
}

func (r *ClickHouseReader) GetDependencyGraph(ctx context.Context, queryParams *model.GetServicesParams) (*[]model.ServiceMapDependencyResponseItem, error) {

	ctx = ctxtypes.NewContextWithCommentVals(ctx, map[string]string{
		instrumentationtypes.TelemetrySignal:  telemetrytypes.SignalTraces.StringValue(),
		instrumentationtypes.CodeNamespace:    "clickhouse-reader",
		instrumentationtypes.CodeFunctionName: "GetDependencyGraph",
	})
	response := []model.ServiceMapDependencyResponseItem{}

	args := []interface{}{}
	args = append(args,
		clickhouse.Named("start", uint64(queryParams.Start.Unix())),
		clickhouse.Named("end", uint64(queryParams.End.Unix())),
		clickhouse.Named("duration", uint64(queryParams.End.Unix()-queryParams.Start.Unix())),
	)

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
		r.TraceDB, r.dependencyGraphTable,
	)

	query += " GROUP BY src, dest;"

	r.logger.Debug("GetDependencyGraph query", "query", query, "args", args)

	err := r.db.Select(ctx, &response, query, args...)

	if err != nil {
		r.logger.Error("Error in processing sql query", errorsV2.Attr(err))
		return nil, fmt.Errorf("error in processing sql query %w", err)
	}

	return &response, nil
}

// getLocalTableName normalizes legacy distributed table names for single-node ClickHouse.
func getLocalTableName(tableName string) string {
	tableNameSplit := strings.SplitN(tableName, ".", 2)
	if len(tableNameSplit) != 2 {
		return tableName
	}

	return tableNameSplit[0] + "." + strings.TrimPrefix(tableNameSplit[1], "distributed_")
}

func (r *ClickHouseReader) setTTLTraces(ctx context.Context, orgID string, params *model.TTLParams) (*model.SetTTLResponseItem, *model.ApiError) {
	ctx = ctxtypes.NewContextWithCommentVals(ctx, map[string]string{
		instrumentationtypes.TelemetrySignal:  telemetrytypes.SignalTraces.StringValue(),
		instrumentationtypes.CodeNamespace:    "clickhouse-reader",
		instrumentationtypes.CodeFunctionName: "setTTLTraces",
	})
	// uuid is used as transaction id
	uuidWithHyphen := uuid.New()
	uuid := strings.Replace(uuidWithHyphen.String(), "-", "", -1)
	tableNames := []string{
		r.TraceDB + "." + r.traceTableName,
		r.TraceDB + "." + r.traceResourceTableV3,
		r.TraceDB + "." + signozErrorIndexTable,
		r.TraceDB + "." + signozUsageExplorerTable,
		r.TraceDB + "." + defaultDependencyGraphTable,
		r.TraceDB + "." + r.traceSummaryTable,
		r.TraceDB + "." + r.spanAttributesKeysTable,
	}

	coldStorageDuration := -1
	if len(params.ColdStorageVolume) > 0 {
		coldStorageDuration = int(params.ToColdStorageDuration)
	}

	// check if there is existing things to be done
	for _, tableName := range tableNames {
		statusItem, apiErr := r.checkTTLStatusItem(ctx, orgID, tableName)
		if apiErr != nil {
			return nil, &model.ApiError{Typ: model.ErrorExec, Err: fmt.Errorf("error in processing ttl_status check sql query")}
		}
		if statusItem.Status == constants.StatusPending {
			return nil, &model.ApiError{Typ: model.ErrorConflict, Err: fmt.Errorf("TTL is already running")}
		}
	}

	// TTL query
	ttlV2 := "ALTER TABLE %s MODIFY TTL toDateTime(%s) + INTERVAL %v SECOND DELETE"
	ttlV2ColdStorage := ", toDateTime(%s) + INTERVAL %v SECOND TO VOLUME '%s'"

	// TTL query for resource table
	ttlV2Resource := "ALTER TABLE %s MODIFY TTL toDateTime(seen_at_ts_bucket_start) + toIntervalSecond(1800) + INTERVAL %v SECOND DELETE"
	ttlTracesV2ResourceColdStorage := ", toDateTime(seen_at_ts_bucket_start) + toIntervalSecond(1800) + INTERVAL %v SECOND TO VOLUME '%s'"

	for _, distributedTableName := range tableNames {
		go func(distributedTableName string) {
			tableName := getLocalTableName(distributedTableName)

			// for trace summary table, we need to use end instead of timestamp
			timestamp := "timestamp"
			if strings.HasSuffix(distributedTableName, r.traceSummaryTable) {
				timestamp = "end"
			}

			ttl := types.TTLSetting{
				Identifiable: types.Identifiable{
					ID: valuer.GenerateUUID(),
				},
				TimeAuditable: types.TimeAuditable{
					CreatedAt: time.Now(),
					UpdatedAt: time.Now(),
				},
				TransactionID:  uuid,
				TableName:      tableName,
				TTL:            int(params.DelDuration),
				Status:         constants.StatusPending,
				ColdStorageTTL: coldStorageDuration,
				OrgID:          orgID,
			}
			_, dbErr := r.
				sqlDB.
				BunDB().
				NewInsert().
				Model(&ttl).
				Exec(ctx)
			if dbErr != nil {
				r.logger.Error("error in inserting to ttl_status table", errorsV2.Attr(dbErr))
				return
			}

			req := fmt.Sprintf(ttlV2, tableName, timestamp, params.DelDuration)
			if strings.HasSuffix(distributedTableName, r.traceResourceTableV3) {
				req = fmt.Sprintf(ttlV2Resource, tableName, params.DelDuration)
			}

			if len(params.ColdStorageVolume) > 0 && !strings.HasSuffix(distributedTableName, r.spanAttributesKeysTable) {
				if strings.HasSuffix(distributedTableName, r.traceResourceTableV3) {
					req += fmt.Sprintf(ttlTracesV2ResourceColdStorage, params.ToColdStorageDuration, params.ColdStorageVolume)
				} else {
					req += fmt.Sprintf(ttlV2ColdStorage, timestamp, params.ToColdStorageDuration, params.ColdStorageVolume)
				}
			}
			err := r.setColdStorage(context.Background(), tableName, params.ColdStorageVolume)
			if err != nil {
				r.logger.Error("Error in setting cold storage", errorsV2.Attr(err))
				statusItem, apiErr := r.checkTTLStatusItem(ctx, orgID, tableName)
				if apiErr == nil {
					_, dbErr := r.
						sqlDB.
						BunDB().
						NewUpdate().
						Model(new(types.TTLSetting)).
						Set("updated_at = ?", time.Now()).
						Set("status = ?", constants.StatusFailed).
						Where("id = ?", statusItem.ID.StringValue()).
						Exec(ctx)
					if dbErr != nil {
						r.logger.Error("Error in processing ttl_status update sql query", errorsV2.Attr(dbErr))
						return
					}
				}
				return
			}
			req += " SETTINGS materialize_ttl_after_modify=0;"
			r.logger.Error(" ExecutingTTL request: ", "request", req)
			statusItem, _ := r.checkTTLStatusItem(ctx, orgID, tableName)
			if err := r.db.Exec(ctx, req); err != nil {
				r.logger.Error("Error in executing set TTL query", errorsV2.Attr(err))
				_, dbErr := r.
					sqlDB.
					BunDB().
					NewUpdate().
					Model(new(types.TTLSetting)).
					Set("updated_at = ?", time.Now()).
					Set("status = ?", constants.StatusFailed).
					Where("id = ?", statusItem.ID.StringValue()).
					Exec(ctx)
				if dbErr != nil {
					r.logger.Error("Error in processing ttl_status update sql query", errorsV2.Attr(dbErr))
					return
				}
				return
			}
			_, dbErr = r.
				sqlDB.
				BunDB().
				NewUpdate().
				Model(new(types.TTLSetting)).
				Set("updated_at = ?", time.Now()).
				Set("status = ?", constants.StatusSuccess).
				Where("id = ?", statusItem.ID.StringValue()).
				Exec(ctx)
			if dbErr != nil {
				r.logger.Error("Error in processing ttl_status update sql query", errorsV2.Attr(dbErr))
				return
			}
		}(distributedTableName)
	}
	return &model.SetTTLResponseItem{Message: "move ttl has been successfully set up"}, nil
}

func (r *ClickHouseReader) SetCustomRetentionTTL(ctx context.Context, orgID string, params *model.CustomRetentionTTLParams) (*model.CustomRetentionTTLResponse, error) {

	ctx = ctxtypes.NewContextWithCommentVals(ctx, map[string]string{
		instrumentationtypes.TelemetrySignal:  telemetrytypes.SignalLogs.StringValue(),
		instrumentationtypes.CodeNamespace:    "clickhouse-reader",
		instrumentationtypes.CodeFunctionName: "SetCustomRetentionTTL",
	})

	// Keep only latest 100 transactions/requests
	r.deleteTtlTransactions(ctx, orgID, 100)

	uuidWithHyphen := valuer.GenerateUUID()
	uuid := strings.Replace(uuidWithHyphen.String(), "-", "", -1)

	if params.Type != constants.LogsTTL {
		return nil, errorsV2.Newf(errorsV2.TypeInternal, errorsV2.CodeInternal, "custom retention TTL only supported for logs")
	}

	// Validate TTL conditions
	if err := r.validateTTLConditions(ctx, params.TTLConditions); err != nil {
		return nil, err
	}

	// Calculate cold storage duration
	coldStorageDuration := -1
	if len(params.ColdStorageVolume) > 0 && params.ToColdStorageDurationDays > 0 {
		coldStorageDuration = int(params.ToColdStorageDurationDays) // Already in days
	}

	tableNames := []string{
		r.logsDB + "." + r.logsLocalTableV2,
		r.logsDB + "." + r.logsResourceLocalTableV2,
		getLocalTableName(r.logsDB + "." + r.logsAttributeKeys),
		getLocalTableName(r.logsDB + "." + r.logsResourceKeys),
	}
	distributedTableNames := []string{
		r.logsDB + "." + r.logsTableV2,
		r.logsDB + "." + r.logsResourceTableV2,
	}

	for _, tableName := range tableNames {
		statusItem, apiErr := r.checkCustomRetentionTTLStatusItem(ctx, orgID, tableName)
		if apiErr != nil {
			return nil, errorsV2.Newf(errorsV2.TypeInternal, errorsV2.CodeInternal, "error in processing custom_retention_ttl_status check sql query")
		}
		if statusItem.Status == constants.StatusPending {
			return nil, errorsV2.Newf(errorsV2.TypeInternal, errorsV2.CodeInternal, "custom retention TTL is already running")
		}
	}

	multiIfExpr := r.buildMultiIfExpression(params.TTLConditions, params.DefaultTTLDays, false)
	resourceMultiIfExpr := r.buildMultiIfExpression(params.TTLConditions, params.DefaultTTLDays, true)

	ttlPayload := make(map[string][]string)

	queries := []string{
		fmt.Sprintf(`ALTER TABLE %s MODIFY COLUMN _retention_days UInt16 DEFAULT %s`,
			tableNames[0], multiIfExpr),
		// for distributed table
		fmt.Sprintf(`ALTER TABLE %s MODIFY COLUMN _retention_days UInt16 DEFAULT %s`,
			distributedTableNames[0], multiIfExpr),
	}

	if len(params.ColdStorageVolume) > 0 && coldStorageDuration > 0 {
		queries = append(queries, fmt.Sprintf(`ALTER TABLE %s MODIFY COLUMN _retention_days_cold UInt16 DEFAULT %d`,
			tableNames[0], coldStorageDuration))
		// for distributed table
		queries = append(queries, fmt.Sprintf(`ALTER TABLE %s MODIFY COLUMN _retention_days_cold UInt16 DEFAULT %d`,
			distributedTableNames[0], coldStorageDuration))

		queries = append(queries, fmt.Sprintf(`ALTER TABLE %s MODIFY TTL toDateTime(timestamp / 1000000000) + toIntervalDay(_retention_days) DELETE, toDateTime(timestamp / 1000000000) + toIntervalDay(_retention_days_cold) TO VOLUME '%s' SETTINGS materialize_ttl_after_modify=0`,
			tableNames[0], params.ColdStorageVolume))
	}

	ttlPayload[tableNames[0]] = queries

	resourceQueries := []string{
		fmt.Sprintf(`ALTER TABLE %s MODIFY COLUMN _retention_days UInt16 DEFAULT %s`,
			tableNames[1], resourceMultiIfExpr),
		// for distributed table
		fmt.Sprintf(`ALTER TABLE %s MODIFY COLUMN _retention_days UInt16 DEFAULT %s`,
			distributedTableNames[1], resourceMultiIfExpr),
	}

	if len(params.ColdStorageVolume) > 0 && coldStorageDuration > 0 {
		resourceQueries = append(resourceQueries, fmt.Sprintf(`ALTER TABLE %s MODIFY COLUMN _retention_days_cold UInt16 DEFAULT %d`,
			tableNames[1], coldStorageDuration))
		// for distributed table
		resourceQueries = append(resourceQueries, fmt.Sprintf(`ALTER TABLE %s MODIFY COLUMN _retention_days_cold UInt16 DEFAULT %d`,
			distributedTableNames[1], coldStorageDuration))
		resourceQueries = append(resourceQueries, fmt.Sprintf(`ALTER TABLE %s MODIFY TTL toDateTime(seen_at_ts_bucket_start) + toIntervalSecond(1800) + toIntervalDay(_retention_days) DELETE, toDateTime(seen_at_ts_bucket_start) + toIntervalSecond(1800) + toIntervalDay(_retention_days_cold) TO VOLUME '%s' SETTINGS materialize_ttl_after_modify=0`,
			tableNames[1], params.ColdStorageVolume))
	}

	ttlPayload[tableNames[1]] = resourceQueries

	// NOTE: Since logs support custom rule based retention, that makes it difficult to identify which attributes, resource keys
	// we need to keep, hence choosing MAX for safe side and not to create any complex solution for this.
	maxRetentionTTL := params.DefaultTTLDays
	for _, rule := range params.TTLConditions {
		maxRetentionTTL = max(maxRetentionTTL, rule.TTLDays)
	}

	ttlPayload[tableNames[2]] = []string{
		fmt.Sprintf("ALTER TABLE %s MODIFY TTL timestamp + toIntervalDay(%d) DELETE SETTINGS materialize_ttl_after_modify=0",
			tableNames[2], maxRetentionTTL),
	}

	ttlPayload[tableNames[3]] = []string{
		fmt.Sprintf("ALTER TABLE %s MODIFY TTL timestamp + toIntervalDay(%d) DELETE SETTINGS materialize_ttl_after_modify=0",
			tableNames[3], maxRetentionTTL),
	}

	ttlConditionsJSON, err := json.Marshal(params.TTLConditions)
	if err != nil {
		return nil, errorsV2.Wrapf(err, errorsV2.TypeInternal, errorsV2.CodeInternal, "error marshalling TTL condition")
	}

	for tableName, queries := range ttlPayload {
		customTTL := types.TTLSetting{
			Identifiable: types.Identifiable{
				ID: valuer.GenerateUUID(),
			},
			TimeAuditable: types.TimeAuditable{
				CreatedAt: time.Now(),
				UpdatedAt: time.Now(),
			},
			TransactionID:  uuid,
			TableName:      tableName,
			TTL:            params.DefaultTTLDays,
			Condition:      string(ttlConditionsJSON),
			Status:         constants.StatusPending,
			ColdStorageTTL: coldStorageDuration,
			OrgID:          orgID,
		}

		// Insert TTL setting record
		_, dbErr := r.sqlDB.BunDB().NewInsert().Model(&customTTL).Exec(ctx)
		if dbErr != nil {
			r.logger.Error("error in inserting to custom_retention_ttl_settings table", errorsV2.Attr(dbErr))
			return nil, errorsV2.Wrapf(dbErr, errorsV2.TypeInternal, errorsV2.CodeInternal, "error inserting TTL settings")
		}

		if len(params.ColdStorageVolume) > 0 && coldStorageDuration > 0 {
			err := r.setColdStorage(ctx, tableName, params.ColdStorageVolume)
			if err != nil {
				r.logger.Error("error in setting cold storage", errorsV2.Attr(err))
				r.updateCustomRetentionTTLStatus(ctx, orgID, tableName, constants.StatusFailed)
				return nil, errorsV2.Wrapf(err.Err, errorsV2.TypeInternal, errorsV2.CodeInternal, "error setting cold storage for table %s", tableName)
			}
		}

		for i, query := range queries {
			r.logger.Debug("Executing custom retention TTL request: ", "request", query, "step", i+1)
			if err := r.db.Exec(ctx, query); err != nil {
				r.logger.Error("error while setting custom retention ttl", errorsV2.Attr(err))
				r.updateCustomRetentionTTLStatus(ctx, orgID, tableName, constants.StatusFailed)
				return nil, errorsV2.Wrapf(err, errorsV2.TypeInternal, errorsV2.CodeInternal, "error setting custom retention TTL for table %s, query: %s", tableName, query)
			}
		}

		r.updateCustomRetentionTTLStatus(ctx, orgID, tableName, constants.StatusSuccess)
	}

	return &model.CustomRetentionTTLResponse{
		Message: "custom retention TTL has been successfully set up",
	}, nil
}

// New method to build multiIf expressions with support for multiple AND conditions
func (r *ClickHouseReader) buildMultiIfExpression(ttlConditions []model.CustomRetentionRule, defaultTTLDays int, isResourceTable bool) string {
	var conditions []string

	for i, rule := range ttlConditions {
		r.logger.Debug("Processing rule", "ruleIndex", i, "ttlDays", rule.TTLDays, "conditionsCount", len(rule.Filters))

		if len(rule.Filters) == 0 {
			r.logger.Warn("Rule has no filters, skipping", "ruleIndex", i)
			continue
		}

		// Build AND conditions for this rule
		var andConditions []string
		for j, condition := range rule.Filters {
			r.logger.Debug("Processing condition", "ruleIndex", i, "conditionIndex", j, "key", condition.Key, "values", condition.Values)

			// This should not happen as validation should catch it
			if len(condition.Values) == 0 {
				r.logger.Error("Condition has no values - this should have been caught in validation", "ruleIndex", i, "conditionIndex", j)
				continue
			}

			// Properly quote values for IN clause
			quotedValues := make([]string, len(condition.Values))
			for k, v := range condition.Values {
				quotedValues[k] = fmt.Sprintf("'%s'", v)
			}

			var conditionExpr string
			if isResourceTable {
				// For resource table, use JSONExtractString
				conditionExpr = fmt.Sprintf(
					"JSONExtractString(labels, '%s') IN (%s)",
					condition.Key,
					strings.Join(quotedValues, ", "),
				)
			} else {
				// For main logs table, use resources_string
				conditionExpr = fmt.Sprintf(
					"resources_string['%s'] IN (%s)",
					condition.Key,
					strings.Join(quotedValues, ", "),
				)
			}
			andConditions = append(andConditions, conditionExpr)
		}

		if len(andConditions) > 0 {
			// Join all conditions with AND
			fullCondition := strings.Join(andConditions, " AND ")
			conditionWithTTL := fmt.Sprintf("%s, %d", fullCondition, rule.TTLDays)
			r.logger.Debug("Adding condition to multiIf", "condition", conditionWithTTL)
			conditions = append(conditions, conditionWithTTL)
		}
	}

	// Handle case where no valid conditions were found
	if len(conditions) == 0 {
		r.logger.Info("No valid conditions found, returning default TTL", "defaultTTLDays", defaultTTLDays)
		return fmt.Sprintf("%d", defaultTTLDays)
	}

	result := fmt.Sprintf(
		"multiIf(%s, %d)",
		strings.Join(conditions, ", "),
		defaultTTLDays,
	)

	r.logger.Debug("Final multiIf expression", "expression", result)
	return result
}

func (r *ClickHouseReader) GetCustomRetentionTTL(ctx context.Context, orgID string) (*model.GetCustomRetentionTTLResponse, error) {
	response := &model.GetCustomRetentionTTLResponse{}
	customTTL := new(types.TTLSetting)
	err := r.sqlDB.BunDB().NewSelect().
		Model(customTTL).
		Where("org_id = ?", orgID).
		Where("table_name = ?", r.logsDB+"."+r.logsLocalTableV2).
		OrderExpr("created_at DESC").
		Limit(1).
		Scan(ctx)
	if err != nil && err != sql.ErrNoRows {
		r.logger.Error("Error in processing sql query", errorsV2.Attr(err))
		return nil, errorsV2.Newf(errorsV2.TypeInternal, errorsV2.CodeInternal, "error in processing get custom ttl query")
	}
	if err == sql.ErrNoRows {
		response.DefaultTTLDays = 15
		response.TTLConditions = []model.CustomRetentionRule{}
		response.Status = constants.StatusSuccess
		response.ColdStorageTTLDays = -1
		return response, nil
	}

	var ttlConditions []model.CustomRetentionRule
	if customTTL.Condition != "" {
		if err := json.Unmarshal([]byte(customTTL.Condition), &ttlConditions); err != nil {
			r.logger.Error("Error parsing TTL conditions", errorsV2.Attr(err))
			ttlConditions = []model.CustomRetentionRule{}
		}
	}

	response.DefaultTTLDays = customTTL.TTL
	response.TTLConditions = ttlConditions
	response.Status = customTTL.Status
	response.ColdStorageTTLDays = customTTL.ColdStorageTTL

	return response, nil
}

func (r *ClickHouseReader) checkCustomRetentionTTLStatusItem(ctx context.Context, orgID string, tableName string) (*types.TTLSetting, error) {
	ttl := new(types.TTLSetting)
	err := r.sqlDB.BunDB().NewSelect().
		Model(ttl).
		Where("table_name = ?", tableName).
		Where("org_id = ?", orgID).
		OrderExpr("created_at DESC").
		Limit(1).
		Scan(ctx)

	if err != nil && err != sql.ErrNoRows {
		r.logger.Error("Error in processing sql query", errorsV2.Attr(err))
		return ttl, errorsV2.Newf(errorsV2.TypeInternal, errorsV2.CodeInternal, "error in processing custom_retention_ttl_status check sql query")
	}

	return ttl, nil
}

func (r *ClickHouseReader) updateCustomRetentionTTLStatus(ctx context.Context, orgID, tableName, status string) {
	statusItem, apiErr := r.checkCustomRetentionTTLStatusItem(ctx, orgID, tableName)
	if apiErr == nil && statusItem != nil {
		_, dbErr := r.sqlDB.BunDB().NewUpdate().
			Model(new(types.TTLSetting)).
			Set("updated_at = ?", time.Now()).
			Set("status = ?", status).
			Where("id = ?", statusItem.ID.StringValue()).
			Exec(ctx)
		if dbErr != nil {
			r.logger.Error("Error in processing custom_retention_ttl_status update sql query", errorsV2.Attr(dbErr))
		}
	}
}

// Enhanced validation function with duplicate detection and efficient key validation
func (r *ClickHouseReader) validateTTLConditions(ctx context.Context, ttlConditions []model.CustomRetentionRule) error {
	ctx = ctxtypes.NewContextWithCommentVals(ctx, map[string]string{
		instrumentationtypes.CodeNamespace:    "clickhouse-reader",
		instrumentationtypes.CodeFunctionName: "validateTTLConditions",
	})
	if len(ttlConditions) == 0 {
		return nil
	}

	// Collect all unique keys and detect duplicates
	var allKeys []string
	keySet := make(map[string]struct{})
	conditionSignatures := make(map[string]bool)

	for i, rule := range ttlConditions {
		if len(rule.Filters) == 0 {
			return errorsV2.Newf(errorsV2.TypeInternal, errorsV2.CodeInternal, "rule at index %d has no filters", i)
		}

		// Create a signature for this rule's conditions to detect duplicates
		var conditionKeys []string
		var conditionValues []string

		for j, condition := range rule.Filters {
			if len(condition.Values) == 0 {
				return errorsV2.Newf(errorsV2.TypeInternal, errorsV2.CodeInternal, "condition at rule %d, condition %d has no values", i, j)
			}

			// Collect unique keys
			if _, exists := keySet[condition.Key]; !exists {
				allKeys = append(allKeys, condition.Key)
				keySet[condition.Key] = struct{}{}
			}

			// Build signature for duplicate detection
			conditionKeys = append(conditionKeys, condition.Key)
			conditionValues = append(conditionValues, strings.Join(condition.Values, ","))
		}

		// Create signature by sorting keys and values to handle order-independent comparison
		sort.Strings(conditionKeys)
		sort.Strings(conditionValues)
		signature := strings.Join(conditionKeys, "|") + ":" + strings.Join(conditionValues, "|")

		if conditionSignatures[signature] {
			return errorsV2.Newf(errorsV2.TypeInternal, errorsV2.CodeInternal, "duplicate rule detected at index %d: rules with identical conditions are not allowed", i)
		}
		conditionSignatures[signature] = true
	}

	if len(allKeys) == 0 {
		return nil
	}

	// Create placeholders for IN query
	placeholders := make([]string, len(allKeys))
	for i := range allKeys {
		placeholders[i] = "?"
	}

	// Efficient validation using IN query
	query := fmt.Sprintf("SELECT name FROM %s.%s WHERE name IN (%s)",
		r.logsDB, r.logsResourceKeys, strings.Join(placeholders, ", "))

	// Convert keys to interface{} for query parameters
	params := make([]interface{}, len(allKeys))
	for i, key := range allKeys {
		params[i] = key
	}

	rows, err := r.db.Query(ctx, query, params...)
	if err != nil {
		return errorsV2.Wrapf(err, errorsV2.TypeInternal, errorsV2.CodeInternal, "failed to validate resource keys")
	}
	defer rows.Close()

	// Collect valid keys
	validKeys := make(map[string]struct{})
	for rows.Next() {
		var name string
		if err := rows.Scan(&name); err != nil {
			return errorsV2.Wrapf(err, errorsV2.TypeInternal, errorsV2.CodeInternal, "failed to scan resource keys")
		}
		validKeys[name] = struct{}{}
	}

	// Find invalid keys
	var invalidKeys []string
	for _, key := range allKeys {
		if _, exists := validKeys[key]; !exists {
			invalidKeys = append(invalidKeys, key)
		}
	}

	if len(invalidKeys) > 0 {
		return errorsV2.Newf(errorsV2.TypeInternal, errorsV2.CodeInternal, "invalid resource keys found: %v. Please check logs_resource_keys table for valid keys", invalidKeys)
	}

	return nil
}

// SetTTL sets the TTL for traces or metrics tables.
// This is an async API which creates goroutines to set TTL.
// Status of TTL update is tracked with ttl_status table in sqlite db.
func (r *ClickHouseReader) SetTTL(ctx context.Context, orgID string, params *model.TTLParams) (*model.SetTTLResponseItem, *model.ApiError) {
	// Keep only latest 100 transactions/requests
	r.deleteTtlTransactions(ctx, orgID, 100)

	switch params.Type {
	case constants.TraceTTL:
		return r.setTTLTraces(ctx, orgID, params)
	case constants.MetricsTTL:
		return r.setTTLMetrics(ctx, orgID, params)
	default:
		return nil, &model.ApiError{Typ: model.ErrorExec, Err: fmt.Errorf("error while setting ttl. ttl type should be <metrics|traces>, got %v", params.Type)}
	}

}

func (r *ClickHouseReader) setTTLMetrics(ctx context.Context, orgID string, params *model.TTLParams) (*model.SetTTLResponseItem, *model.ApiError) {
	ctx = ctxtypes.NewContextWithCommentVals(ctx, map[string]string{
		instrumentationtypes.TelemetrySignal:  telemetrytypes.SignalMetrics.StringValue(),
		instrumentationtypes.CodeNamespace:    "clickhouse-reader",
		instrumentationtypes.CodeFunctionName: "setTTLMetrics",
	})
	// uuid is used as transaction id
	uuidWithHyphen := uuid.New()
	uuid := strings.Replace(uuidWithHyphen.String(), "-", "", -1)

	coldStorageDuration := -1
	if len(params.ColdStorageVolume) > 0 {
		coldStorageDuration = int(params.ToColdStorageDuration)
	}
	tableNames := []string{
		signozMetricDBName + "." + signozSampleLocalTableName,
		signozMetricDBName + "." + signozSamplesAgg5mLocalTableName,
		signozMetricDBName + "." + signozSamplesAgg30mLocalTableName,
		signozMetricDBName + "." + signozExpHistLocalTableName,
		signozMetricDBName + "." + signozTSLocalTableNameV4,
		signozMetricDBName + "." + signozTSLocalTableNameV46Hrs,
		signozMetricDBName + "." + signozTSLocalTableNameV41Day,
		signozMetricDBName + "." + signozTSLocalTableNameV41Week,
	}
	for _, tableName := range tableNames {
		statusItem, apiErr := r.checkTTLStatusItem(ctx, orgID, tableName)
		if apiErr != nil {
			return nil, &model.ApiError{Typ: model.ErrorExec, Err: fmt.Errorf("error in processing ttl_status check sql query")}
		}
		if statusItem.Status == constants.StatusPending {
			return nil, &model.ApiError{Typ: model.ErrorConflict, Err: fmt.Errorf("TTL is already running")}
		}
	}
	metricTTL := func(tableName string) {
		ttl := types.TTLSetting{
			Identifiable: types.Identifiable{
				ID: valuer.GenerateUUID(),
			},
			TimeAuditable: types.TimeAuditable{
				CreatedAt: time.Now(),
				UpdatedAt: time.Now(),
			},
			TransactionID:  uuid,
			TableName:      tableName,
			TTL:            int(params.DelDuration),
			Status:         constants.StatusPending,
			ColdStorageTTL: coldStorageDuration,
			OrgID:          orgID,
		}
		_, dbErr := r.
			sqlDB.
			BunDB().
			NewInsert().
			Model(&ttl).
			Exec(ctx)
		if dbErr != nil {
			r.logger.Error("error in inserting to ttl_status table", errorsV2.Attr(dbErr))
			return
		}
		timeColumn := "timestamp_ms"
		if strings.Contains(tableName, "v4") || strings.Contains(tableName, "exp_hist") {
			timeColumn = "unix_milli"
		}

		req := fmt.Sprintf(
			"ALTER TABLE %v MODIFY TTL toDateTime(toUInt32(%s / 1000), 'UTC') + "+
				"INTERVAL %v SECOND DELETE", tableName, timeColumn, params.DelDuration)
		if len(params.ColdStorageVolume) > 0 {
			req += fmt.Sprintf(", toDateTime(toUInt32(%s / 1000), 'UTC')"+
				" + INTERVAL %v SECOND TO VOLUME '%s'",
				timeColumn, params.ToColdStorageDuration, params.ColdStorageVolume)
		}
		err := r.setColdStorage(context.Background(), tableName, params.ColdStorageVolume)
		if err != nil {
			r.logger.Error("Error in setting cold storage", errorsV2.Attr(err))
			statusItem, apiErr := r.checkTTLStatusItem(ctx, orgID, tableName)
			if apiErr == nil {
				_, dbErr := r.
					sqlDB.
					BunDB().
					NewUpdate().
					Model(new(types.TTLSetting)).
					Set("updated_at = ?", time.Now()).
					Set("status = ?", constants.StatusFailed).
					Where("id = ?", statusItem.ID.StringValue()).
					Exec(ctx)
				if dbErr != nil {
					r.logger.Error("Error in processing ttl_status update sql query", errorsV2.Attr(dbErr))
					return
				}
			}
			return
		}
		req += " SETTINGS materialize_ttl_after_modify=0"
		r.logger.Info("Executing TTL request: ", "request", req)
		statusItem, _ := r.checkTTLStatusItem(ctx, orgID, tableName)
		if err := r.db.Exec(ctx, req); err != nil {
			r.logger.Error("error while setting ttl.", errorsV2.Attr(err))
			_, dbErr := r.
				sqlDB.
				BunDB().
				NewUpdate().
				Model(new(types.TTLSetting)).
				Set("updated_at = ?", time.Now()).
				Set("status = ?", constants.StatusFailed).
				Where("id = ?", statusItem.ID.StringValue()).
				Exec(ctx)
			if dbErr != nil {
				r.logger.Error("Error in processing ttl_status update sql query", errorsV2.Attr(dbErr))
				return
			}
			return
		}
		_, dbErr = r.
			sqlDB.
			BunDB().
			NewUpdate().
			Model(new(types.TTLSetting)).
			Set("updated_at = ?", time.Now()).
			Set("status = ?", constants.StatusSuccess).
			Where("id = ?", statusItem.ID.StringValue()).
			Exec(ctx)
		if dbErr != nil {
			r.logger.Error("Error in processing ttl_status update sql query", errorsV2.Attr(dbErr))
			return
		}
	}
	for _, tableName := range tableNames {
		go metricTTL(tableName)
	}
	return &model.SetTTLResponseItem{Message: "move ttl has been successfully set up"}, nil
}

func (r *ClickHouseReader) deleteTtlTransactions(ctx context.Context, orgID string, numberOfTransactionsStore int) {
	limitTransactions := []string{}
	err := r.
		sqlDB.
		BunDB().
		NewSelect().
		Column("transaction_id").
		Model(new(types.TTLSetting)).
		Where("org_id = ?", orgID).
		Group("transaction_id").
		OrderExpr("MAX(created_at) DESC").
		Limit(numberOfTransactionsStore).
		Scan(ctx, &limitTransactions)

	if err != nil {
		r.logger.Error("Error in processing ttl_status delete sql query", errorsV2.Attr(err))
	}

	_, err = r.
		sqlDB.
		BunDB().
		NewDelete().
		Model(new(types.TTLSetting)).
		Where("transaction_id NOT IN (?)", bun.In(limitTransactions)).
		Exec(ctx)
	if err != nil {
		r.logger.Error("Error in processing ttl_status delete sql query", errorsV2.Attr(err))
	}
}

// checkTTLStatusItem checks if ttl_status table has an entry for the given table name
func (r *ClickHouseReader) checkTTLStatusItem(ctx context.Context, orgID string, tableName string) (*types.TTLSetting, *model.ApiError) {
	r.logger.Info("checkTTLStatusItem query", "tableName", tableName)
	ttl := new(types.TTLSetting)
	err := r.
		sqlDB.
		BunDB().
		NewSelect().
		Model(ttl).
		Where("table_name = ?", tableName).
		Where("org_id = ?", orgID).
		OrderExpr("created_at DESC").
		Limit(1).
		Scan(ctx)
	if err != nil && err != sql.ErrNoRows {
		r.logger.Error("Error in processing sql query", errorsV2.Attr(err))
		return ttl, &model.ApiError{Typ: model.ErrorExec, Err: fmt.Errorf("error in processing ttl_status check sql query")}
	}
	return ttl, nil
}

// getTTLQueryStatus fetches ttl_status table status from DB
func (r *ClickHouseReader) getTTLQueryStatus(ctx context.Context, orgID string, tableNameArray []string) (string, *model.ApiError) {
	failFlag := false
	status := constants.StatusSuccess
	for _, tableName := range tableNameArray {
		statusItem, apiErr := r.checkTTLStatusItem(ctx, orgID, tableName)
		emptyStatusStruct := new(types.TTLSetting)
		if statusItem == emptyStatusStruct {
			return "", nil
		}
		if apiErr != nil {
			return "", &model.ApiError{Typ: model.ErrorExec, Err: fmt.Errorf("error in processing ttl_status check sql query")}
		}
		if statusItem.Status == constants.StatusPending && statusItem.UpdatedAt.Unix()-time.Now().Unix() < 3600 {
			status = constants.StatusPending
			return status, nil
		}
		if statusItem.Status == constants.StatusFailed {
			failFlag = true
		}
	}
	if failFlag {
		status = constants.StatusFailed
	}

	return status, nil
}

func (r *ClickHouseReader) setColdStorage(ctx context.Context, tableName string, coldStorageVolume string) *model.ApiError {

	ctx = ctxtypes.NewContextWithCommentVals(ctx, map[string]string{
		instrumentationtypes.CodeNamespace:    "clickhouse-reader",
		instrumentationtypes.CodeFunctionName: "setColdStorage",
	})
	// Set the storage policy for the required table. If it is already set, then setting it again
	// will not a problem.
	if len(coldStorageVolume) > 0 {
		policyReq := fmt.Sprintf("ALTER TABLE %s MODIFY SETTING storage_policy='tiered'", tableName)

		r.logger.Info("Executing Storage policy request: ", "request", policyReq)
		if err := r.db.Exec(ctx, policyReq); err != nil {
			r.logger.Error("error while setting storage policy", errorsV2.Attr(err))
			return &model.ApiError{Typ: model.ErrorExec, Err: fmt.Errorf("error while setting storage policy. Err=%v", err)}
		}
	}
	return nil
}

// GetDisks returns a list of disks {name, type} configured in clickhouse DB.
func (r *ClickHouseReader) GetDisks(ctx context.Context) (*[]model.DiskItem, *model.ApiError) {
	ctx = ctxtypes.NewContextWithCommentVals(ctx, map[string]string{
		instrumentationtypes.CodeNamespace:    "clickhouse-reader",
		instrumentationtypes.CodeFunctionName: "GetDisks",
	})
	diskItems := []model.DiskItem{}

	query := "SELECT name,type FROM system.disks"
	if err := r.db.Select(ctx, &diskItems, query); err != nil {
		r.logger.Error("Error in processing sql query", errorsV2.Attr(err))
		return nil, &model.ApiError{Typ: model.ErrorExec, Err: fmt.Errorf("error while getting disks. Err=%v", err)}
	}

	return &diskItems, nil
}

func getLocalTableNameArray(tableNames []string) []string {
	localTableNames := make([]string, 0, len(tableNames))
	for _, tableName := range tableNames {
		localTableNames = append(localTableNames, getLocalTableName(tableName))
	}

	return localTableNames
}

// GetTTL returns current ttl, expected ttl and past setTTL status for metrics/traces.
func (r *ClickHouseReader) GetTTL(ctx context.Context, orgID string, ttlParams *model.GetTTLParams) (*model.GetTTLResponseItem, *model.ApiError) {

	ctx = ctxtypes.NewContextWithCommentVals(ctx, map[string]string{
		instrumentationtypes.CodeNamespace:    "clickhouse-reader",
		instrumentationtypes.CodeFunctionName: "GetTTL",
	})
	parseTTL := func(queryResp string) (int, int) {

		r.logger.Info("Parsing TTL from: ", "queryResp", queryResp)
		deleteTTLExp := regexp.MustCompile(`toIntervalSecond\(([0-9]*)\)`)
		moveTTLExp := regexp.MustCompile(`toIntervalSecond\(([0-9]*)\) TO VOLUME`)

		var delTTL, moveTTL int = -1, -1

		m := deleteTTLExp.FindStringSubmatch(queryResp)
		if len(m) > 1 {
			seconds_int, err := strconv.Atoi(m[1])
			if err != nil {
				return -1, -1
			}
			delTTL = seconds_int / 3600
		}

		m = moveTTLExp.FindStringSubmatch(queryResp)
		if len(m) > 1 {
			seconds_int, err := strconv.Atoi(m[1])
			if err != nil {
				return -1, -1
			}
			moveTTL = seconds_int / 3600
		}

		return delTTL, moveTTL
	}

	getMetricsTTL := func() (*model.DBResponseTTL, *model.ApiError) {
		var dbResp []model.DBResponseTTL

		query := fmt.Sprintf("SELECT engine_full FROM system.tables WHERE name='%v'", signozSampleLocalTableName)

		err := r.db.Select(ctx, &dbResp, query)

		if err != nil {
			r.logger.Error("error while getting ttl", errorsV2.Attr(err))
			return nil, &model.ApiError{Typ: model.ErrorExec, Err: fmt.Errorf("error while getting ttl. Err=%v", err)}
		}
		if len(dbResp) == 0 {
			return nil, nil
		} else {
			return &dbResp[0], nil
		}
	}

	getTracesTTL := func() (*model.DBResponseTTL, *model.ApiError) {
		var dbResp []model.DBResponseTTL

		query := fmt.Sprintf("SELECT engine_full FROM system.tables WHERE name='%v' AND database='%v'", r.traceLocalTableName, signozTraceDBName)

		err := r.db.Select(ctx, &dbResp, query)

		if err != nil {
			r.logger.Error("error while getting ttl", errorsV2.Attr(err))
			return nil, &model.ApiError{Typ: model.ErrorExec, Err: fmt.Errorf("error while getting ttl. Err=%v", err)}
		}
		if len(dbResp) == 0 {
			return nil, nil
		} else {
			return &dbResp[0], nil
		}
	}

	switch ttlParams.Type {
	case constants.TraceTTL:
		tableNameArray := []string{
			r.TraceDB + "." + r.traceTableName,
			r.TraceDB + "." + r.traceResourceTableV3,
			r.TraceDB + "." + signozErrorIndexTable,
			r.TraceDB + "." + signozUsageExplorerTable,
			r.TraceDB + "." + defaultDependencyGraphTable,
			r.TraceDB + "." + r.traceSummaryTable,
		}
		tableNameArray = getLocalTableNameArray(tableNameArray)
		status, apiErr := r.getTTLQueryStatus(ctx, orgID, tableNameArray)
		if apiErr != nil {
			return nil, apiErr
		}
		dbResp, apiErr := getTracesTTL()
		if apiErr != nil {
			return nil, apiErr
		}
		ttlQuery, apiErr := r.checkTTLStatusItem(ctx, orgID, tableNameArray[0])
		if apiErr != nil {
			return nil, apiErr
		}
		ttlQuery.TTL = ttlQuery.TTL / 3600 // convert to hours
		if ttlQuery.ColdStorageTTL != -1 {
			ttlQuery.ColdStorageTTL = ttlQuery.ColdStorageTTL / 3600 // convert to hours
		}

		delTTL, moveTTL := parseTTL(dbResp.EngineFull)
		return &model.GetTTLResponseItem{TracesTime: delTTL, TracesMoveTime: moveTTL, ExpectedTracesTime: ttlQuery.TTL, ExpectedTracesMoveTime: ttlQuery.ColdStorageTTL, Status: status}, nil

	case constants.MetricsTTL:
		tableNameArray := []string{signozMetricDBName + "." + signozSampleTableName}
		tableNameArray = getLocalTableNameArray(tableNameArray)
		status, apiErr := r.getTTLQueryStatus(ctx, orgID, tableNameArray)
		if apiErr != nil {
			return nil, apiErr
		}
		dbResp, apiErr := getMetricsTTL()
		if apiErr != nil {
			return nil, apiErr
		}
		ttlQuery, apiErr := r.checkTTLStatusItem(ctx, orgID, tableNameArray[0])
		if apiErr != nil {
			return nil, apiErr
		}
		ttlQuery.TTL = ttlQuery.TTL / 3600 // convert to hours
		if ttlQuery.ColdStorageTTL != -1 {
			ttlQuery.ColdStorageTTL = ttlQuery.ColdStorageTTL / 3600 // convert to hours
		}

		delTTL, moveTTL := parseTTL(dbResp.EngineFull)
		return &model.GetTTLResponseItem{MetricsTime: delTTL, MetricsMoveTime: moveTTL, ExpectedMetricsTime: ttlQuery.TTL, ExpectedMetricsMoveTime: ttlQuery.ColdStorageTTL, Status: status}, nil

	default:
		return nil, &model.ApiError{Typ: model.ErrorExec, Err: fmt.Errorf("error while getting ttl. ttl type should be metrics|traces, got %v",
			ttlParams.Type)}
	}

}

func (r *ClickHouseReader) ListErrors(ctx context.Context, queryParams *model.ListErrorsParams) (*[]model.Error, *model.ApiError) {

	ctx = ctxtypes.NewContextWithCommentVals(ctx, map[string]string{
		instrumentationtypes.TelemetrySignal:  telemetrytypes.SignalTraces.StringValue(),
		instrumentationtypes.CodeNamespace:    "clickhouse-reader",
		instrumentationtypes.CodeFunctionName: "ListErrors",
	})
	var getErrorResponses []model.Error

	query := "SELECT any(exceptionMessage) as exceptionMessage, count() AS exceptionCount, min(timestamp) as firstSeen, max(timestamp) as lastSeen, groupID"
	if len(queryParams.ServiceName) != 0 {
		query = query + ", serviceName"
	} else {
		query = query + ", any(serviceName) as serviceName"
	}
	if len(queryParams.ExceptionType) != 0 {
		query = query + ", exceptionType"
	} else {
		query = query + ", any(exceptionType) as exceptionType"
	}
	query += fmt.Sprintf(" FROM %s.%s WHERE timestamp >= @timestampL AND timestamp <= @timestampU", r.TraceDB, r.errorTable)
	args := []interface{}{clickhouse.Named("timestampL", strconv.FormatInt(queryParams.Start.UnixNano(), 10)), clickhouse.Named("timestampU", strconv.FormatInt(queryParams.End.UnixNano(), 10))}

	if len(queryParams.ServiceName) != 0 {
		query = query + " AND serviceName ilike @serviceName"
		args = append(args, clickhouse.Named("serviceName", "%"+queryParams.ServiceName+"%"))
	}
	if len(queryParams.ExceptionType) != 0 {
		query = query + " AND exceptionType ilike @exceptionType"
		args = append(args, clickhouse.Named("exceptionType", "%"+queryParams.ExceptionType+"%"))
	}

	// create TagQuery from TagQueryParams
	tags := createTagQueryFromTagQueryParams(queryParams.Tags)
	subQuery, argsSubQuery, errStatus := buildQueryWithTagParams(ctx, tags)
	query += subQuery
	args = append(args, argsSubQuery...)

	if errStatus != nil {
		r.logger.Error("Error in processing tags", errorsV2.Attr(errStatus))
		return nil, errStatus
	}
	query = query + " GROUP BY groupID"
	if len(queryParams.ServiceName) != 0 {
		query = query + ", serviceName"
	}
	if len(queryParams.ExceptionType) != 0 {
		query = query + ", exceptionType"
	}
	if len(queryParams.OrderParam) != 0 {
		if queryParams.Order == constants.Descending {
			query = query + " ORDER BY " + queryParams.OrderParam + " DESC"
		} else if queryParams.Order == constants.Ascending {
			query = query + " ORDER BY " + queryParams.OrderParam + " ASC"
		}
	}
	if queryParams.Limit > 0 {
		query = query + " LIMIT @limit"
		args = append(args, clickhouse.Named("limit", queryParams.Limit))
	}

	if queryParams.Offset > 0 {
		query = query + " OFFSET @offset"
		args = append(args, clickhouse.Named("offset", queryParams.Offset))
	}

	err := r.db.Select(ctx, &getErrorResponses, query, args...)
	r.logger.Info(query)

	if err != nil {
		r.logger.Error("Error in processing sql query", errorsV2.Attr(err))
		return nil, &model.ApiError{Typ: model.ErrorExec, Err: fmt.Errorf("error in processing sql query")}
	}

	return &getErrorResponses, nil
}

func (r *ClickHouseReader) CountErrors(ctx context.Context, queryParams *model.CountErrorsParams) (uint64, *model.ApiError) {

	ctx = ctxtypes.NewContextWithCommentVals(ctx, map[string]string{
		instrumentationtypes.TelemetrySignal:  telemetrytypes.SignalTraces.StringValue(),
		instrumentationtypes.CodeNamespace:    "clickhouse-reader",
		instrumentationtypes.CodeFunctionName: "CountErrors",
	})

	var errorCount uint64

	query := fmt.Sprintf("SELECT count(distinct(groupID)) FROM %s.%s WHERE timestamp >= @timestampL AND timestamp <= @timestampU", r.TraceDB, r.errorTable)
	args := []interface{}{clickhouse.Named("timestampL", strconv.FormatInt(queryParams.Start.UnixNano(), 10)), clickhouse.Named("timestampU", strconv.FormatInt(queryParams.End.UnixNano(), 10))}
	if len(queryParams.ServiceName) != 0 {
		query = query + " AND serviceName ilike @serviceName"
		args = append(args, clickhouse.Named("serviceName", "%"+queryParams.ServiceName+"%"))
	}
	if len(queryParams.ExceptionType) != 0 {
		query = query + " AND exceptionType ilike @exceptionType"
		args = append(args, clickhouse.Named("exceptionType", "%"+queryParams.ExceptionType+"%"))
	}

	// create TagQuery from TagQueryParams
	tags := createTagQueryFromTagQueryParams(queryParams.Tags)
	subQuery, argsSubQuery, errStatus := buildQueryWithTagParams(ctx, tags)
	query += subQuery
	args = append(args, argsSubQuery...)

	if errStatus != nil {
		r.logger.Error("Error in processing tags", errorsV2.Attr(errStatus))
		return 0, errStatus
	}

	err := r.db.QueryRow(ctx, query, args...).Scan(&errorCount)
	r.logger.Info(query)

	if err != nil {
		r.logger.Error("Error in processing sql query", errorsV2.Attr(err))
		return 0, &model.ApiError{Typ: model.ErrorExec, Err: fmt.Errorf("error in processing sql query")}
	}

	return errorCount, nil
}

func (r *ClickHouseReader) GetErrorFromErrorID(ctx context.Context, queryParams *model.GetErrorParams) (*model.ErrorWithSpan, *model.ApiError) {

	ctx = ctxtypes.NewContextWithCommentVals(ctx, map[string]string{
		instrumentationtypes.TelemetrySignal:  telemetrytypes.SignalTraces.StringValue(),
		instrumentationtypes.CodeNamespace:    "clickhouse-reader",
		instrumentationtypes.CodeFunctionName: "GetErrorFromErrorID",
	})
	if queryParams.ErrorID == "" {
		r.logger.Error("errorId missing from params")
		return nil, &model.ApiError{Typ: model.ErrorBadData, Err: fmt.Errorf("ErrorID missing from params")}
	}
	var getErrorWithSpanReponse []model.ErrorWithSpan

	query := fmt.Sprintf("SELECT errorID, exceptionType, exceptionStacktrace, exceptionEscaped, exceptionMessage, timestamp, spanID, traceID, serviceName, groupID FROM %s.%s WHERE timestamp = @timestamp AND groupID = @groupID AND errorID = @errorID LIMIT 1", r.TraceDB, r.errorTable)
	args := []interface{}{clickhouse.Named("errorID", queryParams.ErrorID), clickhouse.Named("groupID", queryParams.GroupID), clickhouse.Named("timestamp", strconv.FormatInt(queryParams.Timestamp.UnixNano(), 10))}

	err := r.db.Select(ctx, &getErrorWithSpanReponse, query, args...)
	r.logger.Info(query)

	if err != nil {
		r.logger.Error("Error in processing sql query", errorsV2.Attr(err))
		return nil, &model.ApiError{Typ: model.ErrorExec, Err: fmt.Errorf("error in processing sql query")}
	}

	if len(getErrorWithSpanReponse) > 0 {
		return &getErrorWithSpanReponse[0], nil
	} else {
		return nil, &model.ApiError{Typ: model.ErrorNotFound, Err: fmt.Errorf("Error/Exception not found")}
	}

}

func (r *ClickHouseReader) GetErrorFromGroupID(ctx context.Context, queryParams *model.GetErrorParams) (*model.ErrorWithSpan, *model.ApiError) {

	ctx = ctxtypes.NewContextWithCommentVals(ctx, map[string]string{
		instrumentationtypes.TelemetrySignal:  telemetrytypes.SignalTraces.StringValue(),
		instrumentationtypes.CodeNamespace:    "clickhouse-reader",
		instrumentationtypes.CodeFunctionName: "GetErrorFromGroupID",
	})
	var getErrorWithSpanReponse []model.ErrorWithSpan

	query := fmt.Sprintf("SELECT errorID, exceptionType, exceptionStacktrace, exceptionEscaped, exceptionMessage, timestamp, spanID, traceID, serviceName, groupID FROM %s.%s WHERE timestamp = @timestamp AND groupID = @groupID LIMIT 1", r.TraceDB, r.errorTable)
	args := []interface{}{clickhouse.Named("groupID", queryParams.GroupID), clickhouse.Named("timestamp", strconv.FormatInt(queryParams.Timestamp.UnixNano(), 10))}

	err := r.db.Select(ctx, &getErrorWithSpanReponse, query, args...)

	r.logger.Info(query)

	if err != nil {
		r.logger.Error("Error in processing sql query", errorsV2.Attr(err))
		return nil, &model.ApiError{Typ: model.ErrorExec, Err: fmt.Errorf("error in processing sql query")}
	}

	if len(getErrorWithSpanReponse) > 0 {
		return &getErrorWithSpanReponse[0], nil
	} else {
		return nil, &model.ApiError{Typ: model.ErrorNotFound, Err: fmt.Errorf("Error/Exception not found")}
	}

}

func (r *ClickHouseReader) GetNextPrevErrorIDs(ctx context.Context, queryParams *model.GetErrorParams) (*model.NextPrevErrorIDs, *model.ApiError) {

	if queryParams.ErrorID == "" {
		r.logger.Error("errorId missing from params")
		return nil, &model.ApiError{Typ: model.ErrorBadData, Err: fmt.Errorf("ErrorID missing from params")}
	}
	var apiErr *model.ApiError
	getNextPrevErrorIDsResponse := model.NextPrevErrorIDs{
		GroupID: queryParams.GroupID,
	}
	getNextPrevErrorIDsResponse.NextErrorID, getNextPrevErrorIDsResponse.NextTimestamp, apiErr = r.getNextErrorID(ctx, queryParams)
	if apiErr != nil {
		r.logger.Error("Unable to get next error ID due to err: ", errorsV2.Attr(apiErr))
		return nil, apiErr
	}
	getNextPrevErrorIDsResponse.PrevErrorID, getNextPrevErrorIDsResponse.PrevTimestamp, apiErr = r.getPrevErrorID(ctx, queryParams)
	if apiErr != nil {
		r.logger.Error("Unable to get prev error ID due to err: ", errorsV2.Attr(apiErr))
		return nil, apiErr
	}
	return &getNextPrevErrorIDsResponse, nil

}

func (r *ClickHouseReader) getNextErrorID(ctx context.Context, queryParams *model.GetErrorParams) (string, time.Time, *model.ApiError) {

	ctx = ctxtypes.NewContextWithCommentVals(ctx, map[string]string{
		instrumentationtypes.TelemetrySignal:  telemetrytypes.SignalTraces.StringValue(),
		instrumentationtypes.CodeNamespace:    "clickhouse-reader",
		instrumentationtypes.CodeFunctionName: "getNextErrorID",
	})
	var getNextErrorIDReponse []model.NextPrevErrorIDsDBResponse

	query := fmt.Sprintf("SELECT errorID as nextErrorID, timestamp as nextTimestamp FROM %s.%s WHERE groupID = @groupID AND timestamp >= @timestamp AND errorID != @errorID ORDER BY timestamp ASC LIMIT 2", r.TraceDB, r.errorTable)
	args := []interface{}{clickhouse.Named("errorID", queryParams.ErrorID), clickhouse.Named("groupID", queryParams.GroupID), clickhouse.Named("timestamp", strconv.FormatInt(queryParams.Timestamp.UnixNano(), 10))}

	err := r.db.Select(ctx, &getNextErrorIDReponse, query, args...)

	r.logger.Info(query)

	if err != nil {
		r.logger.Error("Error in processing sql query", errorsV2.Attr(err))
		return "", time.Time{}, &model.ApiError{Typ: model.ErrorExec, Err: fmt.Errorf("error in processing sql query")}
	}
	if len(getNextErrorIDReponse) == 0 {
		r.logger.Info("NextErrorID not found")
		return "", time.Time{}, nil
	} else if len(getNextErrorIDReponse) == 1 {
		r.logger.Info("NextErrorID found")
		return getNextErrorIDReponse[0].NextErrorID, getNextErrorIDReponse[0].NextTimestamp, nil
	} else {
		if getNextErrorIDReponse[0].Timestamp.UnixNano() == getNextErrorIDReponse[1].Timestamp.UnixNano() {
			var getNextErrorIDReponse []model.NextPrevErrorIDsDBResponse

			query := fmt.Sprintf("SELECT errorID as nextErrorID, timestamp as nextTimestamp FROM %s.%s WHERE groupID = @groupID AND timestamp = @timestamp AND errorID > @errorID ORDER BY errorID ASC LIMIT 1", r.TraceDB, r.errorTable)
			args := []interface{}{clickhouse.Named("errorID", queryParams.ErrorID), clickhouse.Named("groupID", queryParams.GroupID), clickhouse.Named("timestamp", strconv.FormatInt(queryParams.Timestamp.UnixNano(), 10))}

			err := r.db.Select(ctx, &getNextErrorIDReponse, query, args...)

			r.logger.Info(query)

			if err != nil {
				r.logger.Error("Error in processing sql query", errorsV2.Attr(err))
				return "", time.Time{}, &model.ApiError{Typ: model.ErrorExec, Err: fmt.Errorf("error in processing sql query")}
			}
			if len(getNextErrorIDReponse) == 0 {
				var getNextErrorIDReponse []model.NextPrevErrorIDsDBResponse

				query := fmt.Sprintf("SELECT errorID as nextErrorID, timestamp as nextTimestamp FROM %s.%s WHERE groupID = @groupID AND timestamp > @timestamp ORDER BY timestamp ASC LIMIT 1", r.TraceDB, r.errorTable)
				args := []interface{}{clickhouse.Named("errorID", queryParams.ErrorID), clickhouse.Named("groupID", queryParams.GroupID), clickhouse.Named("timestamp", strconv.FormatInt(queryParams.Timestamp.UnixNano(), 10))}

				err := r.db.Select(ctx, &getNextErrorIDReponse, query, args...)

				r.logger.Info(query)

				if err != nil {
					r.logger.Error("Error in processing sql query", errorsV2.Attr(err))
					return "", time.Time{}, &model.ApiError{Typ: model.ErrorExec, Err: fmt.Errorf("error in processing sql query")}
				}

				if len(getNextErrorIDReponse) == 0 {
					r.logger.Info("NextErrorID not found")
					return "", time.Time{}, nil
				} else {
					r.logger.Info("NextErrorID found")
					return getNextErrorIDReponse[0].NextErrorID, getNextErrorIDReponse[0].NextTimestamp, nil
				}
			} else {
				r.logger.Info("NextErrorID found")
				return getNextErrorIDReponse[0].NextErrorID, getNextErrorIDReponse[0].NextTimestamp, nil
			}
		} else {
			r.logger.Info("NextErrorID found")
			return getNextErrorIDReponse[0].NextErrorID, getNextErrorIDReponse[0].NextTimestamp, nil
		}
	}
}

func (r *ClickHouseReader) getPrevErrorID(ctx context.Context, queryParams *model.GetErrorParams) (string, time.Time, *model.ApiError) {

	ctx = ctxtypes.NewContextWithCommentVals(ctx, map[string]string{
		instrumentationtypes.TelemetrySignal:  telemetrytypes.SignalTraces.StringValue(),
		instrumentationtypes.CodeNamespace:    "clickhouse-reader",
		instrumentationtypes.CodeFunctionName: "getPrevErrorID",
	})
	var getPrevErrorIDReponse []model.NextPrevErrorIDsDBResponse

	query := fmt.Sprintf("SELECT errorID as prevErrorID, timestamp as prevTimestamp FROM %s.%s WHERE groupID = @groupID AND timestamp <= @timestamp AND errorID != @errorID ORDER BY timestamp DESC LIMIT 2", r.TraceDB, r.errorTable)
	args := []interface{}{clickhouse.Named("errorID", queryParams.ErrorID), clickhouse.Named("groupID", queryParams.GroupID), clickhouse.Named("timestamp", strconv.FormatInt(queryParams.Timestamp.UnixNano(), 10))}

	err := r.db.Select(ctx, &getPrevErrorIDReponse, query, args...)

	r.logger.Info(query)

	if err != nil {
		r.logger.Error("Error in processing sql query", errorsV2.Attr(err))
		return "", time.Time{}, &model.ApiError{Typ: model.ErrorExec, Err: fmt.Errorf("error in processing sql query")}
	}
	if len(getPrevErrorIDReponse) == 0 {
		r.logger.Info("PrevErrorID not found")
		return "", time.Time{}, nil
	} else if len(getPrevErrorIDReponse) == 1 {
		r.logger.Info("PrevErrorID found")
		return getPrevErrorIDReponse[0].PrevErrorID, getPrevErrorIDReponse[0].PrevTimestamp, nil
	} else {
		if getPrevErrorIDReponse[0].Timestamp.UnixNano() == getPrevErrorIDReponse[1].Timestamp.UnixNano() {
			var getPrevErrorIDReponse []model.NextPrevErrorIDsDBResponse

			query := fmt.Sprintf("SELECT errorID as prevErrorID, timestamp as prevTimestamp FROM %s.%s WHERE groupID = @groupID AND timestamp = @timestamp AND errorID < @errorID ORDER BY errorID DESC LIMIT 1", r.TraceDB, r.errorTable)
			args := []interface{}{clickhouse.Named("errorID", queryParams.ErrorID), clickhouse.Named("groupID", queryParams.GroupID), clickhouse.Named("timestamp", strconv.FormatInt(queryParams.Timestamp.UnixNano(), 10))}

			err := r.db.Select(ctx, &getPrevErrorIDReponse, query, args...)

			r.logger.Info(query)

			if err != nil {
				r.logger.Error("Error in processing sql query", errorsV2.Attr(err))
				return "", time.Time{}, &model.ApiError{Typ: model.ErrorExec, Err: fmt.Errorf("error in processing sql query")}
			}
			if len(getPrevErrorIDReponse) == 0 {
				var getPrevErrorIDReponse []model.NextPrevErrorIDsDBResponse

				query := fmt.Sprintf("SELECT errorID as prevErrorID, timestamp as prevTimestamp FROM %s.%s WHERE groupID = @groupID AND timestamp < @timestamp ORDER BY timestamp DESC LIMIT 1", r.TraceDB, r.errorTable)
				args := []interface{}{clickhouse.Named("errorID", queryParams.ErrorID), clickhouse.Named("groupID", queryParams.GroupID), clickhouse.Named("timestamp", strconv.FormatInt(queryParams.Timestamp.UnixNano(), 10))}

				err := r.db.Select(ctx, &getPrevErrorIDReponse, query, args...)

				r.logger.Info(query)

				if err != nil {
					r.logger.Error("Error in processing sql query", errorsV2.Attr(err))
					return "", time.Time{}, &model.ApiError{Typ: model.ErrorExec, Err: fmt.Errorf("error in processing sql query")}
				}

				if len(getPrevErrorIDReponse) == 0 {
					r.logger.Info("PrevErrorID not found")
					return "", time.Time{}, nil
				} else {
					r.logger.Info("PrevErrorID found")
					return getPrevErrorIDReponse[0].PrevErrorID, getPrevErrorIDReponse[0].PrevTimestamp, nil
				}
			} else {
				r.logger.Info("PrevErrorID found")
				return getPrevErrorIDReponse[0].PrevErrorID, getPrevErrorIDReponse[0].PrevTimestamp, nil
			}
		} else {
			r.logger.Info("PrevErrorID found")
			return getPrevErrorIDReponse[0].PrevErrorID, getPrevErrorIDReponse[0].PrevTimestamp, nil
		}
	}
}

func (r *ClickHouseReader) GetMetricAggregateAttributes(ctx context.Context, orgID valuer.UUID, req *querytypes.AggregateAttributeRequest, skipSignozMetrics bool) (*querytypes.AggregateAttributeResponse, error) {
	ctx = ctxtypes.NewContextWithCommentVals(ctx, map[string]string{
		instrumentationtypes.TelemetrySignal:  telemetrytypes.SignalMetrics.StringValue(),
		instrumentationtypes.CodeNamespace:    "clickhouse-reader",
		instrumentationtypes.CodeFunctionName: "GetMetricAggregateAttributes",
	})
	var response querytypes.AggregateAttributeResponse
	normalized := true
	if constants.IsDotMetricsEnabled {
		normalized = false
	}

	// Query all relevant metric names from time_series_v4, but leave metadata retrieval to cache/db
	query := fmt.Sprintf(
		`SELECT DISTINCT metric_name 
		 FROM %s.%s 
		 WHERE metric_name ILIKE $1 AND __normalized = $2`,
		signozMetricDBName, signozTSTableNameV41Day)

	if req.Limit != 0 {
		query = query + fmt.Sprintf(" LIMIT %d;", req.Limit)
	}

	rows, err := r.db.Query(ctx, query, fmt.Sprintf("%%%s%%", req.SearchText), normalized)
	if err != nil {
		r.logger.Error("Error while querying metric names", errorsV2.Attr(err))
		return nil, fmt.Errorf("error while executing metric name query: %s", err.Error())
	}
	defer rows.Close()

	var metricNames []string
	for rows.Next() {
		var name string
		if err := rows.Scan(&name); err != nil {
			return nil, fmt.Errorf("error while scanning metric name: %s", err.Error())
		}
		if skipSignozMetrics && strings.HasPrefix(name, "signoz") {
			continue
		}
		metricNames = append(metricNames, name)
	}

	if len(metricNames) == 0 {
		return &response, nil
	}

	// Get all metadata in one shot
	metadataMap, apiError := r.GetUpdatedMetricsMetadata(ctx, orgID, metricNames...)
	if apiError != nil {
		return &response, fmt.Errorf("error getting updated metrics metadata: %s", apiError.Error())
	}

	seen := make(map[string]struct{})
	for _, name := range metricNames {
		metadata := metadataMap[name]

		typ := string(metadata.MetricType)
		temporality := string(metadata.Temporality)
		isMonotonic := metadata.IsMonotonic

		// Non-monotonic cumulative sums are treated as gauges
		if typ == "Sum" && !isMonotonic && temporality == string(querytypes.Cumulative) {
			typ = "Gauge"
		}

		// unlike traces/logs `tag`/`resource` type, the `Type` will be metric type
		key := querytypes.AttributeKey{
			Key:      name,
			DataType: querytypes.AttributeKeyDataTypeFloat64,
			Type:     querytypes.AttributeKeyType(typ),
			IsColumn: true,
		}

		if _, ok := seen[name+typ]; ok {
			continue
		}
		seen[name+typ] = struct{}{}
		response.AttributeKeys = append(response.AttributeKeys, key)
	}

	return &response, nil
}

func (r *ClickHouseReader) GetMetricMetadata(ctx context.Context, orgID valuer.UUID, metricName, serviceName string) (*querytypes.MetricMetadataResponse, error) {

	ctx = ctxtypes.NewContextWithCommentVals(ctx, map[string]string{
		instrumentationtypes.TelemetrySignal:  telemetrytypes.SignalMetrics.StringValue(),
		instrumentationtypes.CodeNamespace:    "clickhouse-reader",
		instrumentationtypes.CodeFunctionName: "GetMetricMetadata",
	})
	unixMilli := common.PastDayRoundOff()

	// 1. Fetch metadata from cache/db using unified function
	metadataMap, apiError := r.GetUpdatedMetricsMetadata(ctx, orgID, metricName)
	if apiError != nil {
		r.logger.Error("Error in getting metric cached metadata", errorsV2.Attr(apiError))
		return nil, fmt.Errorf("error fetching metric metadata: %s", apiError.Err.Error())
	}

	// Defaults in case metadata is not found
	var (
		deltaExists bool
		isMonotonic bool
		temporality string
		description string
		metricType  string
		unit        string
	)

	metadata, ok := metadataMap[metricName]
	if !ok {
		return nil, fmt.Errorf("metric metadata not found: %s", metricName)
	}

	metricType = string(metadata.MetricType)
	temporality = string(metadata.Temporality)
	isMonotonic = metadata.IsMonotonic
	description = metadata.Description
	unit = metadata.Unit

	if temporality == string(querytypes.Delta) {
		deltaExists = true
	}
	// 2. Only for Histograms, get `le` buckets
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

		rows, err := r.db.Query(ctx, query, metricName, unixMilli, serviceName, serviceName)
		if err != nil {
			r.logger.Error("Error while querying histogram buckets", errorsV2.Attr(err))
			return nil, fmt.Errorf("error while querying histogram buckets: %s", err.Error())
		}
		defer rows.Close()

		for rows.Next() {
			var leStr string
			if err := rows.Scan(&leStr); err != nil {
				return nil, fmt.Errorf("error while scanning le: %s", err.Error())
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
		Delta:       deltaExists,
		Le:          leFloat64,
		Description: description,
		Unit:        unit,
		Type:        metricType,
		IsMonotonic: isMonotonic,
		Temporality: temporality,
	}, nil
}

// GetTimeSeriesResult runs a query and returns time series rows.
func readRow(vars []interface{}, columnNames []string, countOfNumberCols int) ([]string, map[string]string, []map[string]string, *timeseriestypes.Point) {
	// Each row will have a value and a timestamp, and an optional list of label values
	// example: {Timestamp: ..., Value: ...}
	// The timestamp may also not present in some cases where the time series is reduced to single value
	var point timeseriestypes.Point

	// groupBy is a container to hold label values for the current point
	// example: ["frontend", "/fetch"]
	var groupBy []string

	var groupAttributesArray []map[string]string
	// groupAttributes is a container to hold the key-value pairs for the current
	// metric point.
	// example: {"serviceName": "frontend", "operation": "/fetch"}
	groupAttributes := make(map[string]string)

	isValidPoint := false

	for idx, v := range vars {
		colName := columnNames[idx]
		switch v := v.(type) {
		case *string:
			// special case for returning all labels in metrics datasource
			if colName == "fullLabels" {
				var metric map[string]string
				err := json.Unmarshal([]byte(*v), &metric)
				if err != nil {
					slog.Error("unexpected error encountered", errorsV2.Attr(err))
				}
				for key, val := range metric {
					groupBy = append(groupBy, val)
					if _, ok := groupAttributes[key]; !ok {
						groupAttributesArray = append(groupAttributesArray, map[string]string{key: val})
					}
					groupAttributes[key] = val
				}
			} else {
				groupBy = append(groupBy, *v)
				if _, ok := groupAttributes[colName]; !ok {
					groupAttributesArray = append(groupAttributesArray, map[string]string{colName: *v})
				}
				groupAttributes[colName] = *v
			}
		case *time.Time:
			point.Timestamp = v.UnixMilli()
		case *float64, *float32:
			if _, ok := constants.ReservedColumnTargetAliases[colName]; ok || countOfNumberCols == 1 {
				isValidPoint = true
				point.Value = float64(reflect.ValueOf(v).Elem().Float())
			} else {
				val := strconv.FormatFloat(reflect.ValueOf(v).Elem().Float(), 'f', -1, 64)
				groupBy = append(groupBy, val)
				if _, ok := groupAttributes[colName]; !ok {
					groupAttributesArray = append(groupAttributesArray, map[string]string{colName: val})
				}
				groupAttributes[colName] = val
			}
		case **float64, **float32:
			val := reflect.ValueOf(v)
			if val.IsValid() && !val.IsNil() && !val.Elem().IsNil() {
				value := reflect.ValueOf(v).Elem().Elem().Float()
				if _, ok := constants.ReservedColumnTargetAliases[colName]; ok || countOfNumberCols == 1 {
					isValidPoint = true
					point.Value = value
				} else {
					val := strconv.FormatFloat(value, 'f', -1, 64)
					groupBy = append(groupBy, val)
					if _, ok := groupAttributes[colName]; !ok {
						groupAttributesArray = append(groupAttributesArray, map[string]string{colName: val})
					}
					groupAttributes[colName] = val
				}
			}
		case *uint, *uint8, *uint64, *uint16, *uint32:
			if _, ok := constants.ReservedColumnTargetAliases[colName]; ok || countOfNumberCols == 1 {
				isValidPoint = true
				point.Value = float64(reflect.ValueOf(v).Elem().Uint())
			} else {
				groupBy = append(groupBy, fmt.Sprintf("%v", reflect.ValueOf(v).Elem().Uint()))
				if _, ok := groupAttributes[colName]; !ok {
					groupAttributesArray = append(groupAttributesArray, map[string]string{colName: fmt.Sprintf("%v", reflect.ValueOf(v).Elem().Uint())})
				}
				groupAttributes[colName] = fmt.Sprintf("%v", reflect.ValueOf(v).Elem().Uint())
			}
		case **uint, **uint8, **uint64, **uint16, **uint32:
			val := reflect.ValueOf(v)
			if val.IsValid() && !val.IsNil() && !val.Elem().IsNil() {
				value := reflect.ValueOf(v).Elem().Elem().Uint()
				if _, ok := constants.ReservedColumnTargetAliases[colName]; ok || countOfNumberCols == 1 {
					isValidPoint = true
					point.Value = float64(value)
				} else {
					groupBy = append(groupBy, fmt.Sprintf("%v", value))
					if _, ok := groupAttributes[colName]; !ok {
						groupAttributesArray = append(groupAttributesArray, map[string]string{colName: fmt.Sprintf("%v", value)})
					}
					groupAttributes[colName] = fmt.Sprintf("%v", value)
				}
			}
		case *int, *int8, *int16, *int32, *int64:
			if _, ok := constants.ReservedColumnTargetAliases[colName]; ok || countOfNumberCols == 1 {
				isValidPoint = true
				point.Value = float64(reflect.ValueOf(v).Elem().Int())
			} else {
				groupBy = append(groupBy, fmt.Sprintf("%v", reflect.ValueOf(v).Elem().Int()))
				if _, ok := groupAttributes[colName]; !ok {
					groupAttributesArray = append(groupAttributesArray, map[string]string{colName: fmt.Sprintf("%v", reflect.ValueOf(v).Elem().Int())})
				}
				groupAttributes[colName] = fmt.Sprintf("%v", reflect.ValueOf(v).Elem().Int())
			}
		case **int, **int8, **int16, **int32, **int64:
			val := reflect.ValueOf(v)
			if val.IsValid() && !val.IsNil() && !val.Elem().IsNil() {
				value := reflect.ValueOf(v).Elem().Elem().Int()
				if _, ok := constants.ReservedColumnTargetAliases[colName]; ok || countOfNumberCols == 1 {
					isValidPoint = true
					point.Value = float64(value)
				} else {
					groupBy = append(groupBy, fmt.Sprintf("%v", value))
					if _, ok := groupAttributes[colName]; !ok {
						groupAttributesArray = append(groupAttributesArray, map[string]string{colName: fmt.Sprintf("%v", value)})
					}
					groupAttributes[colName] = fmt.Sprintf("%v", value)
				}
			}
		case *bool:
			groupBy = append(groupBy, fmt.Sprintf("%v", *v))
			if _, ok := groupAttributes[colName]; !ok {
				groupAttributesArray = append(groupAttributesArray, map[string]string{colName: fmt.Sprintf("%v", *v)})
			}
			groupAttributes[colName] = fmt.Sprintf("%v", *v)

		default:
			slog.Error("unsupported var type found in query builder query result", "v", v, "colName", colName)
		}
	}
	if isValidPoint {
		return groupBy, groupAttributes, groupAttributesArray, &point
	}
	return groupBy, groupAttributes, groupAttributesArray, nil
}

func readRowsForTimeSeriesResult(rows driver.Rows, vars []interface{}, columnNames []string, countOfNumberCols int) ([]*timeseriestypes.Series, error) {
	// when groupBy is applied, each combination of cartesian product
	// of attribute values is a separate series. Each item in seriesToPoints
	// represent a unique series where the key is sorted attribute values joined
	// by "," and the value is the list of points for that series

	// For instance, group by (serviceName, operation)
	// with two services and three operations in each will result in (maximum of) 6 series
	// ("frontend", "order") x ("/fetch", "/fetch/{Id}", "/order")
	//
	// ("frontend", "/fetch")
	// ("frontend", "/fetch/{Id}")
	// ("frontend", "/order")
	// ("order", "/fetch")
	// ("order", "/fetch/{Id}")
	// ("order", "/order")
	seriesToPoints := make(map[string][]timeseriestypes.Point)
	var keys []string
	// seriesToAttrs is a mapping of key to a map of attribute key to attribute value
	// for each series. This is used to populate the series' attributes
	// For instance, for the above example, the seriesToAttrs will be
	// {
	//   "frontend,/fetch": {"serviceName": "frontend", "operation": "/fetch"},
	//   "frontend,/fetch/{Id}": {"serviceName": "frontend", "operation": "/fetch/{Id}"},
	//   "frontend,/order": {"serviceName": "frontend", "operation": "/order"},
	//   "order,/fetch": {"serviceName": "order", "operation": "/fetch"},
	//   "order,/fetch/{Id}": {"serviceName": "order", "operation": "/fetch/{Id}"},
	//   "order,/order": {"serviceName": "order", "operation": "/order"},
	// }
	seriesToAttrs := make(map[string]map[string]string)
	labelsArray := make(map[string][]map[string]string)
	for rows.Next() {
		if err := rows.Scan(vars...); err != nil {
			return nil, err
		}
		groupBy, groupAttributes, groupAttributesArray, metricPoint := readRow(vars, columnNames, countOfNumberCols)
		// skip the point if the value is NaN or Inf
		// are they ever useful enough to be returned?
		if metricPoint != nil && (math.IsNaN(metricPoint.Value) || math.IsInf(metricPoint.Value, 0)) {
			continue
		}
		sort.Strings(groupBy)
		key := strings.Join(groupBy, "")
		if _, exists := seriesToAttrs[key]; !exists {
			keys = append(keys, key)
		}
		seriesToAttrs[key] = groupAttributes
		labelsArray[key] = groupAttributesArray
		if metricPoint != nil {
			seriesToPoints[key] = append(seriesToPoints[key], *metricPoint)
		}
	}

	var seriesList []*timeseriestypes.Series
	for _, key := range keys {
		points := seriesToPoints[key]
		series := timeseriestypes.Series{Labels: seriesToAttrs[key], Points: points, LabelsArray: labelsArray[key]}
		seriesList = append(seriesList, &series)
	}
	return seriesList, getPersonalisedError(rows.Err())
}

// GetTimeSeriesResult runs the query and returns list of time series
func (r *ClickHouseReader) GetTimeSeriesResult(ctx context.Context, query string) ([]*timeseriestypes.Series, error) {
	ctx = ctxtypes.NewContextWithCommentVals(ctx, map[string]string{
		instrumentationtypes.CodeNamespace:    "clickhouse-reader",
		instrumentationtypes.CodeFunctionName: "GetTimeSeriesResult",
	})
	rows, err := r.db.Query(ctx, query)

	if err != nil {
		r.logger.Error("error while reading time series result", errorsV2.Attr(err))
		return nil, errors.New(err.Error())
	}
	defer rows.Close()

	var (
		columnTypes = rows.ColumnTypes()
		columnNames = rows.Columns()
		vars        = make([]interface{}, len(columnTypes))
	)
	var countOfNumberCols int

	for i := range columnTypes {
		vars[i] = reflect.New(columnTypes[i].ScanType()).Interface()
		switch columnTypes[i].ScanType().Kind() {
		case reflect.Float32,
			reflect.Float64,
			reflect.Uint,
			reflect.Uint8,
			reflect.Uint16,
			reflect.Uint32,
			reflect.Uint64,
			reflect.Int,
			reflect.Int8,
			reflect.Int16,
			reflect.Int32,
			reflect.Int64:
			countOfNumberCols++
		}
	}

	return readRowsForTimeSeriesResult(rows, vars, columnNames, countOfNumberCols)
}

// GetListResult runs the query and returns list of rows
func (r *ClickHouseReader) GetListResult(ctx context.Context, query string) ([]*timeseriestypes.Row, error) {
	ctx = ctxtypes.NewContextWithCommentVals(ctx, map[string]string{
		instrumentationtypes.CodeNamespace:    "clickhouse-reader",
		instrumentationtypes.CodeFunctionName: "GetListResult",
	})
	rows, err := r.db.Query(ctx, query)
	if err != nil {
		r.logger.Error("error while reading time series result", errorsV2.Attr(err))
		return nil, errors.New(err.Error())
	}

	defer rows.Close()

	var (
		columnTypes = rows.ColumnTypes()
		columnNames = rows.Columns()
	)

	var rowList []*timeseriestypes.Row

	for rows.Next() {
		var vars = make([]interface{}, len(columnTypes))
		for i := range columnTypes {
			vars[i] = reflect.New(columnTypes[i].ScanType()).Interface()
		}
		if err := rows.Scan(vars...); err != nil {
			return nil, err
		}
		row := map[string]interface{}{}
		var t time.Time
		for idx, v := range vars {
			if columnNames[idx] == "timestamp" {
				switch v := v.(type) {
				case *uint64:
					t = time.Unix(0, int64(*v))
				case *time.Time:
					t = *v
				}
			} else if columnNames[idx] == "timestamp_datetime" {
				t = *v.(*time.Time)
			} else if columnNames[idx] == "events" {
				var events []map[string]interface{}
				eventsFromDB, ok := v.(*[]string)
				if !ok {
					continue
				}
				for _, event := range *eventsFromDB {
					var eventMap map[string]interface{}
					json.Unmarshal([]byte(event), &eventMap)
					events = append(events, eventMap)
				}
				row[columnNames[idx]] = events
			} else {
				row[columnNames[idx]] = v
			}
		}

		rowList = append(rowList, &timeseriestypes.Row{Timestamp: t, Data: row})
	}

	return rowList, getPersonalisedError(rows.Err())

}

// GetHostMetricsExistenceAndEarliestTime returns (count, minFirstReportedUnixMilli, error) for the given host metric names
// from metadata. When count is 0, minFirstReportedUnixMilli is 0.
func getPersonalisedError(err error) error {
	if err == nil {
		return nil
	}
	slog.Error("error while reading result", errorsV2.Attr(err))
	if strings.Contains(err.Error(), "code: 307") {
		return chErrors.ErrResourceBytesLimitExceeded
	}

	if strings.Contains(err.Error(), "code: 159") {
		return chErrors.ErrResourceTimeLimitExceeded
	}
	return err
}

func (r *ClickHouseReader) CheckClickHouse(ctx context.Context) error {
	ctx = ctxtypes.NewContextWithCommentVals(ctx, map[string]string{
		instrumentationtypes.CodeNamespace:    "clickhouse-reader",
		instrumentationtypes.CodeFunctionName: "CheckClickHouse",
	})
	rows, err := r.db.Query(ctx, "SELECT 1")
	if err != nil {
		return err
	}
	defer rows.Close()

	return nil
}

func (r *ClickHouseReader) AddRuleStateHistory(ctx context.Context, ruleStateHistory []model.RuleStateHistory) error {
	ctx = ctxtypes.NewContextWithCommentVals(ctx, map[string]string{
		instrumentationtypes.CodeNamespace:    "clickhouse-reader",
		instrumentationtypes.CodeFunctionName: "AddRuleStateHistory",
	})
	var statement driver.Batch
	var err error

	defer func() {
		if statement != nil {
			statement.Abort()
		}
	}()

	statement, err = r.db.PrepareBatch(ctx, fmt.Sprintf("INSERT INTO %s.%s (rule_id, rule_name, overall_state, overall_state_changed, state, state_changed, unix_milli, labels, fingerprint, value) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)",
		signozHistoryDBName, ruleStateHistoryTableName))

	if err != nil {
		return err
	}

	for _, history := range ruleStateHistory {
		err = statement.Append(history.RuleID, history.RuleName, history.OverallState, history.OverallStateChanged, history.State, history.StateChanged, history.UnixMilli, history.Labels, history.Fingerprint, history.Value)
		if err != nil {
			return err
		}
	}

	err = statement.Send()
	if err != nil {
		return err
	}
	return nil
}

func (r *ClickHouseReader) GetLastSavedRuleStateHistory(ctx context.Context, ruleID string) ([]model.RuleStateHistory, error) {
	ctx = ctxtypes.NewContextWithCommentVals(ctx, map[string]string{
		instrumentationtypes.CodeNamespace:    "clickhouse-reader",
		instrumentationtypes.CodeFunctionName: "GetLastSavedRuleStateHistory",
	})
	query := fmt.Sprintf("SELECT * FROM %s.%s WHERE rule_id = '%s' AND state_changed = true ORDER BY unix_milli DESC LIMIT 1 BY fingerprint",
		signozHistoryDBName, ruleStateHistoryTableName, ruleID)

	history := []model.RuleStateHistory{}
	err := r.db.Select(ctx, &history, query)
	if err != nil {
		return nil, err
	}
	return history, nil
}

func (r *ClickHouseReader) ReadRuleStateHistoryByRuleID(
	ctx context.Context, ruleID string, params *model.QueryRuleStateHistory) (*model.RuleStateTimeline, error) {

	ctx = ctxtypes.NewContextWithCommentVals(ctx, map[string]string{
		instrumentationtypes.CodeNamespace:    "clickhouse-reader",
		instrumentationtypes.CodeFunctionName: "ReadRuleStateHistoryByRuleID",
	})
	var conditions []string

	conditions = append(conditions, fmt.Sprintf("rule_id = '%s'", ruleID))

	conditions = append(conditions, fmt.Sprintf("unix_milli >= %d AND unix_milli < %d", params.Start, params.End))

	if params.State != "" {
		conditions = append(conditions, fmt.Sprintf("state = '%s'", params.State))
	}

	if params.Filters != nil && len(params.Filters.Items) != 0 {
		for _, item := range params.Filters.Items {
			toFormat := item.Value
			op := querytypes.FilterOperator(strings.ToLower(strings.TrimSpace(string(item.Operator))))
			if op == querytypes.FilterOperatorContains || op == querytypes.FilterOperatorNotContains {
				toFormat = fmt.Sprintf("%%%s%%", toFormat)
			}
			fmtVal := utils.ClickHouseFormattedValue(toFormat)
			switch op {
			case querytypes.FilterOperatorEqual:
				conditions = append(conditions, fmt.Sprintf("JSONExtractString(labels, '%s') = %s", item.Key.Key, fmtVal))
			case querytypes.FilterOperatorNotEqual:
				conditions = append(conditions, fmt.Sprintf("JSONExtractString(labels, '%s') != %s", item.Key.Key, fmtVal))
			case querytypes.FilterOperatorIn:
				conditions = append(conditions, fmt.Sprintf("JSONExtractString(labels, '%s') IN %s", item.Key.Key, fmtVal))
			case querytypes.FilterOperatorNotIn:
				conditions = append(conditions, fmt.Sprintf("JSONExtractString(labels, '%s') NOT IN %s", item.Key.Key, fmtVal))
			case querytypes.FilterOperatorLike:
				conditions = append(conditions, fmt.Sprintf("like(JSONExtractString(labels, '%s'), %s)", item.Key.Key, fmtVal))
			case querytypes.FilterOperatorNotLike:
				conditions = append(conditions, fmt.Sprintf("notLike(JSONExtractString(labels, '%s'), %s)", item.Key.Key, fmtVal))
			case querytypes.FilterOperatorRegex:
				conditions = append(conditions, fmt.Sprintf("match(JSONExtractString(labels, '%s'), %s)", item.Key.Key, fmtVal))
			case querytypes.FilterOperatorNotRegex:
				conditions = append(conditions, fmt.Sprintf("not match(JSONExtractString(labels, '%s'), %s)", item.Key.Key, fmtVal))
			case querytypes.FilterOperatorGreaterThan:
				conditions = append(conditions, fmt.Sprintf("JSONExtractString(labels, '%s') > %s", item.Key.Key, fmtVal))
			case querytypes.FilterOperatorGreaterThanOrEq:
				conditions = append(conditions, fmt.Sprintf("JSONExtractString(labels, '%s') >= %s", item.Key.Key, fmtVal))
			case querytypes.FilterOperatorLessThan:
				conditions = append(conditions, fmt.Sprintf("JSONExtractString(labels, '%s') < %s", item.Key.Key, fmtVal))
			case querytypes.FilterOperatorLessThanOrEq:
				conditions = append(conditions, fmt.Sprintf("JSONExtractString(labels, '%s') <= %s", item.Key.Key, fmtVal))
			case querytypes.FilterOperatorContains:
				conditions = append(conditions, fmt.Sprintf("like(JSONExtractString(labels, '%s'), %s)", item.Key.Key, fmtVal))
			case querytypes.FilterOperatorNotContains:
				conditions = append(conditions, fmt.Sprintf("notLike(JSONExtractString(labels, '%s'), %s)", item.Key.Key, fmtVal))
			case querytypes.FilterOperatorExists:
				conditions = append(conditions, fmt.Sprintf("has(JSONExtractKeys(labels), '%s')", item.Key.Key))
			case querytypes.FilterOperatorNotExists:
				conditions = append(conditions, fmt.Sprintf("not has(JSONExtractKeys(labels), '%s')", item.Key.Key))
			default:
				return nil, fmt.Errorf("unsupported filter operator")
			}
		}
	}
	whereClause := strings.Join(conditions, " AND ")

	query := fmt.Sprintf("SELECT * FROM %s.%s WHERE %s ORDER BY unix_milli %s LIMIT %d OFFSET %d",
		signozHistoryDBName, ruleStateHistoryTableName, whereClause, params.Order, params.Limit, params.Offset)

	history := []model.RuleStateHistory{}
	r.logger.Debug("rule state history query", "query", query)
	err := r.db.Select(ctx, &history, query)
	if err != nil {
		r.logger.Error("Error while reading rule state history", errorsV2.Attr(err))
		return nil, err
	}

	var total uint64
	r.logger.Debug("rule state history total query", "query", fmt.Sprintf("SELECT count(*) FROM %s.%s WHERE %s",
		signozHistoryDBName, ruleStateHistoryTableName, whereClause))
	err = r.db.QueryRow(ctx, fmt.Sprintf("SELECT count(*) FROM %s.%s WHERE %s",
		signozHistoryDBName, ruleStateHistoryTableName, whereClause)).Scan(&total)
	if err != nil {
		return nil, err
	}

	labelsQuery := fmt.Sprintf("SELECT DISTINCT labels FROM %s.%s WHERE rule_id = $1",
		signozHistoryDBName, ruleStateHistoryTableName)
	rows, err := r.db.Query(ctx, labelsQuery, ruleID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	labelsMap := make(map[string][]string)
	for rows.Next() {
		var rawLabel string
		err = rows.Scan(&rawLabel)
		if err != nil {
			return nil, err
		}
		label := map[string]string{}
		err = json.Unmarshal([]byte(rawLabel), &label)
		if err != nil {
			return nil, err
		}
		for k, v := range label {
			labelsMap[k] = append(labelsMap[k], v)
		}
	}

	timeline := &model.RuleStateTimeline{
		Items:  history,
		Total:  total,
		Labels: labelsMap,
	}

	return timeline, nil
}

func (r *ClickHouseReader) ReadRuleStateHistoryTopContributorsByRuleID(
	ctx context.Context, ruleID string, params *model.QueryRuleStateHistory) ([]model.RuleStateHistoryContributor, error) {
	ctx = ctxtypes.NewContextWithCommentVals(ctx, map[string]string{
		instrumentationtypes.CodeNamespace:    "clickhouse-reader",
		instrumentationtypes.CodeFunctionName: "ReadRuleStateHistoryTopContributorsByRuleID",
	})
	query := fmt.Sprintf(`SELECT
		fingerprint,
		any(labels) as labels,
		count(*) as count
	FROM %s.%s
	WHERE rule_id = '%s' AND (state_changed = true) AND (state = '%s') AND unix_milli >= %d AND unix_milli <= %d
	GROUP BY fingerprint
	HAVING labels != '{}'
	ORDER BY count DESC`,
		signozHistoryDBName, ruleStateHistoryTableName, ruleID, model.StateFiring.String(), params.Start, params.End)

	r.logger.Debug("rule state history top contributors query", "query", query)
	contributors := []model.RuleStateHistoryContributor{}
	err := r.db.Select(ctx, &contributors, query)
	if err != nil {
		r.logger.Error("Error while reading rule state history", errorsV2.Attr(err))
		return nil, err
	}

	return contributors, nil
}

func (r *ClickHouseReader) GetOverallStateTransitions(ctx context.Context, ruleID string, params *model.QueryRuleStateHistory) ([]model.ReleStateItem, error) {

	ctx = ctxtypes.NewContextWithCommentVals(ctx, map[string]string{
		instrumentationtypes.CodeNamespace:    "clickhouse-reader",
		instrumentationtypes.CodeFunctionName: "GetOverallStateTransitions",
	})
	tmpl := `WITH firing_events AS (
    SELECT
        rule_id,
        state,
        unix_milli AS firing_time
    FROM %s.%s
    WHERE overall_state = '` + model.StateFiring.String() + `' 
      AND overall_state_changed = true
      AND rule_id IN ('%s')
	  AND unix_milli >= %d AND unix_milli <= %d
),
resolution_events AS (
    SELECT
        rule_id,
        state,
        unix_milli AS resolution_time
    FROM %s.%s
    WHERE overall_state = '` + model.StateInactive.String() + `' 
      AND overall_state_changed = true
      AND rule_id IN ('%s')
	  AND unix_milli >= %d AND unix_milli <= %d
),
matched_events AS (
    SELECT
        f.rule_id,
        f.state,
        f.firing_time,
        MIN(r.resolution_time) AS resolution_time
    FROM firing_events f
    LEFT JOIN resolution_events r
        ON f.rule_id = r.rule_id
    WHERE r.resolution_time > f.firing_time
    GROUP BY f.rule_id, f.state, f.firing_time
)
SELECT *
FROM matched_events
ORDER BY firing_time ASC;`

	query := fmt.Sprintf(tmpl,
		signozHistoryDBName, ruleStateHistoryTableName, ruleID, params.Start, params.End,
		signozHistoryDBName, ruleStateHistoryTableName, ruleID, params.Start, params.End)

	r.logger.Debug("overall state transitions query", "query", query)

	transitions := []model.RuleStateTransition{}
	err := r.db.Select(ctx, &transitions, query)
	if err != nil {
		return nil, err
	}

	stateItems := []model.ReleStateItem{}

	for idx, item := range transitions {
		start := item.FiringTime
		end := item.ResolutionTime
		stateItems = append(stateItems, model.ReleStateItem{
			State: item.State,
			Start: start,
			End:   end,
		})
		if idx < len(transitions)-1 {
			nextStart := transitions[idx+1].FiringTime
			if nextStart > end {
				stateItems = append(stateItems, model.ReleStateItem{
					State: model.StateInactive,
					Start: end,
					End:   nextStart,
				})
			}
		}
	}

	// fetch the most recent overall_state from the table
	var state model.AlertState
	stateQuery := fmt.Sprintf("SELECT state FROM %s.%s WHERE rule_id = '%s' AND unix_milli <= %d ORDER BY unix_milli DESC LIMIT 1",
		signozHistoryDBName, ruleStateHistoryTableName, ruleID, params.End)
	if err := r.db.QueryRow(ctx, stateQuery).Scan(&state); err != nil {
		if err != sql.ErrNoRows {
			return nil, err
		}
		state = model.StateInactive
	}

	if len(transitions) == 0 {
		// no transitions found, it is either firing or inactive for whole time range
		stateItems = append(stateItems, model.ReleStateItem{
			State: state,
			Start: params.Start,
			End:   params.End,
		})
	} else {
		// there were some transitions, we need to add the last state at the end
		if state == model.StateInactive {
			stateItems = append(stateItems, model.ReleStateItem{
				State: model.StateInactive,
				Start: transitions[len(transitions)-1].ResolutionTime,
				End:   params.End,
			})
		} else {
			// fetch the most recent firing event from the table in the given time range
			var firingTime int64
			firingQuery := fmt.Sprintf(`
			SELECT
				unix_milli
			FROM %s.%s
			WHERE rule_id = '%s' AND overall_state_changed = true AND overall_state = '%s' AND unix_milli <= %d
			ORDER BY unix_milli DESC LIMIT 1`, signozHistoryDBName, ruleStateHistoryTableName, ruleID, model.StateFiring.String(), params.End)
			if err := r.db.QueryRow(ctx, firingQuery).Scan(&firingTime); err != nil {
				return nil, err
			}
			stateItems = append(stateItems, model.ReleStateItem{
				State: model.StateInactive,
				Start: transitions[len(transitions)-1].ResolutionTime,
				End:   firingTime,
			})
			stateItems = append(stateItems, model.ReleStateItem{
				State: model.StateFiring,
				Start: firingTime,
				End:   params.End,
			})
		}
	}
	return stateItems, nil
}

func (r *ClickHouseReader) GetAvgResolutionTime(ctx context.Context, ruleID string, params *model.QueryRuleStateHistory) (float64, error) {

	ctx = ctxtypes.NewContextWithCommentVals(ctx, map[string]string{
		instrumentationtypes.CodeNamespace:    "clickhouse-reader",
		instrumentationtypes.CodeFunctionName: "GetAvgResolutionTime",
	})
	tmpl := `
WITH firing_events AS (
    SELECT
        rule_id,
        state,
        unix_milli AS firing_time
    FROM %s.%s
    WHERE overall_state = '` + model.StateFiring.String() + `' 
      AND overall_state_changed = true
      AND rule_id IN ('%s')
	  AND unix_milli >= %d AND unix_milli <= %d
),
resolution_events AS (
    SELECT
        rule_id,
        state,
        unix_milli AS resolution_time
    FROM %s.%s
    WHERE overall_state = '` + model.StateInactive.String() + `' 
      AND overall_state_changed = true
      AND rule_id IN ('%s')
	  AND unix_milli >= %d AND unix_milli <= %d
),
matched_events AS (
    SELECT
        f.rule_id,
        f.state,
        f.firing_time,
        MIN(r.resolution_time) AS resolution_time
    FROM firing_events f
    LEFT JOIN resolution_events r
        ON f.rule_id = r.rule_id
    WHERE r.resolution_time > f.firing_time
    GROUP BY f.rule_id, f.state, f.firing_time
)
SELECT AVG(resolution_time - firing_time) / 1000 AS avg_resolution_time
FROM matched_events;
`

	query := fmt.Sprintf(tmpl,
		signozHistoryDBName, ruleStateHistoryTableName, ruleID, params.Start, params.End,
		signozHistoryDBName, ruleStateHistoryTableName, ruleID, params.Start, params.End)

	r.logger.Debug("avg resolution time query", "query", query)
	var avgResolutionTime float64
	err := r.db.QueryRow(ctx, query).Scan(&avgResolutionTime)
	if err != nil {
		return 0, err
	}

	return avgResolutionTime, nil
}

func (r *ClickHouseReader) GetAvgResolutionTimeByInterval(ctx context.Context, ruleID string, params *model.QueryRuleStateHistory) (*timeseriestypes.Series, error) {

	step := common.MinAllowedStepInterval(params.Start, params.End)

	tmpl := `
WITH firing_events AS (
    SELECT
        rule_id,
        state,
        unix_milli AS firing_time
    FROM %s.%s
    WHERE overall_state = '` + model.StateFiring.String() + `' 
      AND overall_state_changed = true
      AND rule_id IN ('%s')
	  AND unix_milli >= %d AND unix_milli <= %d
),
resolution_events AS (
    SELECT
        rule_id,
        state,
        unix_milli AS resolution_time
    FROM %s.%s
    WHERE overall_state = '` + model.StateInactive.String() + `' 
      AND overall_state_changed = true
      AND rule_id IN ('%s')
	  AND unix_milli >= %d AND unix_milli <= %d
),
matched_events AS (
    SELECT
        f.rule_id,
        f.state,
        f.firing_time,
        MIN(r.resolution_time) AS resolution_time
    FROM firing_events f
    LEFT JOIN resolution_events r
        ON f.rule_id = r.rule_id
    WHERE r.resolution_time > f.firing_time
    GROUP BY f.rule_id, f.state, f.firing_time
)
SELECT toStartOfInterval(toDateTime(firing_time / 1000), INTERVAL %d SECOND) AS ts, AVG(resolution_time - firing_time) / 1000 AS avg_resolution_time
FROM matched_events
GROUP BY ts
ORDER BY ts ASC;`

	query := fmt.Sprintf(tmpl,
		signozHistoryDBName, ruleStateHistoryTableName, ruleID, params.Start, params.End,
		signozHistoryDBName, ruleStateHistoryTableName, ruleID, params.Start, params.End, step)

	r.logger.Debug("avg resolution time by interval query", "query", query)
	result, err := r.GetTimeSeriesResult(ctx, query)
	if err != nil || len(result) == 0 {
		return nil, err
	}

	return result[0], nil
}

func (r *ClickHouseReader) GetTotalTriggers(ctx context.Context, ruleID string, params *model.QueryRuleStateHistory) (uint64, error) {
	ctx = ctxtypes.NewContextWithCommentVals(ctx, map[string]string{
		instrumentationtypes.CodeNamespace:    "clickhouse-reader",
		instrumentationtypes.CodeFunctionName: "GetTotalTriggers",
	})
	query := fmt.Sprintf("SELECT count(*) FROM %s.%s WHERE rule_id = '%s' AND (state_changed = true) AND (state = '%s') AND unix_milli >= %d AND unix_milli <= %d",
		signozHistoryDBName, ruleStateHistoryTableName, ruleID, model.StateFiring.String(), params.Start, params.End)

	var totalTriggers uint64

	err := r.db.QueryRow(ctx, query).Scan(&totalTriggers)
	if err != nil {
		return 0, err
	}

	return totalTriggers, nil
}

func (r *ClickHouseReader) GetTriggersByInterval(ctx context.Context, ruleID string, params *model.QueryRuleStateHistory) (*timeseriestypes.Series, error) {
	step := common.MinAllowedStepInterval(params.Start, params.End)

	query := fmt.Sprintf("SELECT count(*), toStartOfInterval(toDateTime(intDiv(unix_milli, 1000)), INTERVAL %d SECOND) as ts FROM %s.%s WHERE rule_id = '%s' AND (state_changed = true) AND (state = '%s') AND unix_milli >= %d AND unix_milli <= %d GROUP BY ts ORDER BY ts ASC",
		step, signozHistoryDBName, ruleStateHistoryTableName, ruleID, model.StateFiring.String(), params.Start, params.End)

	result, err := r.GetTimeSeriesResult(ctx, query)
	if err != nil || len(result) == 0 {
		return nil, err
	}

	return result[0], nil
}

func (r *ClickHouseReader) GetAllMetricFilterAttributeKeys(ctx context.Context, req *metrics_explorer.FilterKeyRequest) (*[]querytypes.AttributeKey, *model.ApiError) {
	ctx = ctxtypes.NewContextWithCommentVals(ctx, map[string]string{
		instrumentationtypes.TelemetrySignal:  telemetrytypes.SignalMetrics.StringValue(),
		instrumentationtypes.CodeNamespace:    "clickhouse-reader",
		instrumentationtypes.CodeFunctionName: "GetAllMetricFilterAttributeKeys",
	})
	var rows driver.Rows
	var response []querytypes.AttributeKey
	normalized := true
	if constants.IsDotMetricsEnabled {
		normalized = false
	}
	query := fmt.Sprintf("SELECT arrayJoin(tagKeys) AS distinctTagKey FROM (SELECT JSONExtractKeys(labels) AS tagKeys FROM %s.%s WHERE unix_milli >= $1 and __normalized = $2 GROUP BY tagKeys) WHERE distinctTagKey ILIKE $3 AND distinctTagKey NOT LIKE '\\_\\_%%' GROUP BY distinctTagKey", signozMetricDBName, signozTSTableNameV41Day)
	if req.Limit != 0 {
		query = query + fmt.Sprintf(" LIMIT %d;", req.Limit)
	}
	valueCtx := context.WithValue(ctx, "clickhouse_max_threads", constants.MetricsExplorerClickhouseThreads)
	rows, err := r.db.Query(valueCtx, query, common.PastDayRoundOff(), normalized, fmt.Sprintf("%%%s%%", req.SearchText)) //only showing past day data
	if err != nil {
		r.logger.Error("Error while executing query", errorsV2.Attr(err))
		return nil, &model.ApiError{Typ: "ClickHouseError", Err: err}
	}

	var attributeKey string
	for rows.Next() {
		if err := rows.Scan(&attributeKey); err != nil {
			return nil, &model.ApiError{Typ: "ClickHouseError", Err: err}
		}
		key := querytypes.AttributeKey{
			Key:      attributeKey,
			DataType: querytypes.AttributeKeyDataTypeString, // https://github.com/OpenObservability/OpenMetrics/blob/main/proto/openmetrics_data_model.proto#L64-L72.
			Type:     querytypes.AttributeKeyTypeTag,
			IsColumn: false,
		}
		response = append(response, key)
	}
	if err := rows.Err(); err != nil {
		return nil, &model.ApiError{Typ: "ClickHouseError", Err: err}
	}
	return &response, nil
}

func (r *ClickHouseReader) GetAllMetricFilterAttributeValues(ctx context.Context, req *metrics_explorer.FilterValueRequest) ([]string, *model.ApiError) {
	ctx = ctxtypes.NewContextWithCommentVals(ctx, map[string]string{
		instrumentationtypes.TelemetrySignal:  telemetrytypes.SignalMetrics.StringValue(),
		instrumentationtypes.CodeNamespace:    "clickhouse-reader",
		instrumentationtypes.CodeFunctionName: "GetAllMetricFilterAttributeValues",
	})
	var query string
	var err error
	var rows driver.Rows
	var attributeValues []string
	normalized := true
	if constants.IsDotMetricsEnabled {
		normalized = false
	}

	query = fmt.Sprintf("SELECT JSONExtractString(labels, $1) AS tagValue FROM %s.%s WHERE JSONExtractString(labels, $2) ILIKE $3 AND unix_milli >= $4 AND __normalized = $5 GROUP BY tagValue", signozMetricDBName, signozTSTableNameV41Day)
	if req.Limit != 0 {
		query = query + fmt.Sprintf(" LIMIT %d;", req.Limit)
	}
	valueCtx := context.WithValue(ctx, "clickhouse_max_threads", constants.MetricsExplorerClickhouseThreads)
	rows, err = r.db.Query(valueCtx, query, req.FilterKey, req.FilterKey, fmt.Sprintf("%%%s%%", req.SearchText), common.PastDayRoundOff(), normalized) //only showing past day data

	if err != nil {
		r.logger.Error("Error while executing query", errorsV2.Attr(err))
		return nil, &model.ApiError{Typ: "ClickHouseError", Err: err}
	}
	defer rows.Close()

	var atrributeValue string
	for rows.Next() {
		if err := rows.Scan(&atrributeValue); err != nil {
			return nil, &model.ApiError{Typ: "ClickHouseError", Err: err}
		}
		attributeValues = append(attributeValues, atrributeValue)
	}
	if err := rows.Err(); err != nil {
		return nil, &model.ApiError{Typ: "ClickHouseError", Err: err}
	}
	return attributeValues, nil
}

func (r *ClickHouseReader) GetAllMetricFilterUnits(ctx context.Context, req *metrics_explorer.FilterValueRequest) ([]string, *model.ApiError) {
	ctx = ctxtypes.NewContextWithCommentVals(ctx, map[string]string{
		instrumentationtypes.TelemetrySignal:  telemetrytypes.SignalMetrics.StringValue(),
		instrumentationtypes.CodeNamespace:    "clickhouse-reader",
		instrumentationtypes.CodeFunctionName: "GetAllMetricFilterUnits",
	})
	var rows driver.Rows
	var response []string
	query := fmt.Sprintf("SELECT DISTINCT unit FROM %s.%s WHERE unit ILIKE $1 AND unit IS NOT NULL ORDER BY unit", signozMetricDBName, signozTSTableNameV41Day)
	if req.Limit != 0 {
		query = query + fmt.Sprintf(" LIMIT %d;", req.Limit)
	}

	valueCtx := context.WithValue(ctx, "clickhouse_max_threads", constants.MetricsExplorerClickhouseThreads)
	rows, err := r.db.Query(valueCtx, query, fmt.Sprintf("%%%s%%", req.SearchText))
	if err != nil {
		r.logger.Error("Error while executing query", errorsV2.Attr(err))
		return nil, &model.ApiError{Typ: "ClickHouseError", Err: err}
	}

	var attributeKey string
	for rows.Next() {
		if err := rows.Scan(&attributeKey); err != nil {
			return nil, &model.ApiError{Typ: "ClickHouseError", Err: err}
		}
		response = append(response, attributeKey)
	}
	if err := rows.Err(); err != nil {
		return nil, &model.ApiError{Typ: "ClickHouseError", Err: err}
	}
	return response, nil
}
func (r *ClickHouseReader) GetAllMetricFilterTypes(ctx context.Context, req *metrics_explorer.FilterValueRequest) ([]string, *model.ApiError) {
	ctx = ctxtypes.NewContextWithCommentVals(ctx, map[string]string{
		instrumentationtypes.TelemetrySignal:  telemetrytypes.SignalMetrics.StringValue(),
		instrumentationtypes.CodeNamespace:    "clickhouse-reader",
		instrumentationtypes.CodeFunctionName: "GetAllMetricFilterTypes",
	})
	var rows driver.Rows
	var response []string
	query := fmt.Sprintf("SELECT DISTINCT type FROM %s.%s WHERE type ILIKE $1 AND type IS NOT NULL ORDER BY type", signozMetricDBName, signozTSTableNameV41Day)
	if req.Limit != 0 {
		query = query + fmt.Sprintf(" LIMIT %d;", req.Limit)
	}
	valueCtx := context.WithValue(ctx, "clickhouse_max_threads", constants.MetricsExplorerClickhouseThreads)
	rows, err := r.db.Query(valueCtx, query, fmt.Sprintf("%%%s%%", req.SearchText))
	if err != nil {
		r.logger.Error("Error while executing query", errorsV2.Attr(err))
		return nil, &model.ApiError{Typ: "ClickHouseError", Err: err}
	}

	var attributeKey string
	for rows.Next() {
		if err := rows.Scan(&attributeKey); err != nil {
			return nil, &model.ApiError{Typ: "ClickHouseError", Err: err}
		}
		response = append(response, attributeKey)
	}
	if err := rows.Err(); err != nil {
		return nil, &model.ApiError{Typ: "ClickHouseError", Err: err}
	}
	return response, nil
}

func (r *ClickHouseReader) GetAttributesForMetricName(ctx context.Context, metricName string, start, end *int64, filters *querytypes.FilterSet) (*[]metrics_explorer.Attribute, *model.ApiError) {
	ctx = ctxtypes.NewContextWithCommentVals(ctx, map[string]string{
		instrumentationtypes.TelemetrySignal:  telemetrytypes.SignalMetrics.StringValue(),
		instrumentationtypes.CodeNamespace:    "clickhouse-reader",
		instrumentationtypes.CodeFunctionName: "GetAttributesForMetricName",
	})
	whereClause := ""
	if filters != nil {
		conditions, _ := utils.BuildFilterConditions(filters, "t")
		if conditions != nil {
			whereClause = "AND " + strings.Join(conditions, " AND ")
		}
	}
	normalized := true
	if constants.IsDotMetricsEnabled {
		normalized = false
	}

	const baseQueryTemplate = `
SELECT 
    kv.1 AS key,
    arrayMap(x -> trim(BOTH '"' FROM x), groupUniqArray(1000)(kv.2)) AS values,
    length(groupUniqArray(10000)(kv.2)) AS valueCount
FROM %s.%s
ARRAY JOIN arrayFilter(x -> NOT startsWith(x.1, '__'), JSONExtractKeysAndValuesRaw(labels)) AS kv
WHERE metric_name = ? AND __normalized=? %s`

	var args []interface{}
	args = append(args, metricName)
	tableName := signozTSTableNameV41Week

	args = append(args, normalized)

	if start != nil && end != nil {
		st, en, tsTable, _ := utils.WhichTSTableToUse(*start, *end)
		*start, *end, tableName = st, en, tsTable
		args = append(args, *start, *end)
	} else if start == nil && end == nil {
		tableName = signozTSTableNameV41Week
	}

	query := fmt.Sprintf(baseQueryTemplate, signozMetricDBName, tableName, whereClause)

	if start != nil && end != nil {
		query += " AND unix_milli BETWEEN ? AND ?"
	}

	query += "\nGROUP BY kv.1\nORDER BY valueCount DESC;"

	valueCtx := context.WithValue(ctx, "clickhouse_max_threads", constants.MetricsExplorerClickhouseThreads)
	rows, err := r.db.Query(valueCtx, query, args...)
	if err != nil {
		return nil, &model.ApiError{Typ: "ClickHouseError", Err: err}
	}
	defer rows.Close()

	var attributesList []metrics_explorer.Attribute
	for rows.Next() {
		var attr metrics_explorer.Attribute
		if err := rows.Scan(&attr.Key, &attr.Value, &attr.ValueCount); err != nil {
			return nil, &model.ApiError{Typ: "ClickHouseError", Err: err}
		}
		attributesList = append(attributesList, attr)
	}

	if err := rows.Err(); err != nil {
		return nil, &model.ApiError{Typ: "ClickHouseError", Err: err}
	}

	return &attributesList, nil
}

func (r *ClickHouseReader) GetNameSimilarity(ctx context.Context, req *metrics_explorer.RelatedMetricsRequest) (map[string]metrics_explorer.RelatedMetricsScore, *model.ApiError) {
	ctx = ctxtypes.NewContextWithCommentVals(ctx, map[string]string{
		instrumentationtypes.TelemetrySignal:  telemetrytypes.SignalMetrics.StringValue(),
		instrumentationtypes.CodeNamespace:    "clickhouse-reader",
		instrumentationtypes.CodeFunctionName: "GetNameSimilarity",
	})
	start, end, tsTable, _ := utils.WhichTSTableToUse(req.Start, req.End)

	normalized := true
	if constants.IsDotMetricsEnabled {
		normalized = false
	}

	query := fmt.Sprintf(`
		SELECT 
			metric_name,
			any(type) as type,
		    any(temporality) as temporality,
		    any(is_monotonic) as monotonic,
			1 - (levenshteinDistance(?, metric_name) / greatest(NULLIF(length(?), 0), NULLIF(length(metric_name), 0))) AS name_similarity
		FROM %s.%s
		WHERE metric_name != ?
		  AND unix_milli BETWEEN ? AND ?
		 AND NOT startsWith(metric_name, 'signoz')
		AND __normalized = ?
		GROUP BY metric_name
		ORDER BY name_similarity DESC
		LIMIT 30;`,
		signozMetricDBName, tsTable)

	valueCtx := context.WithValue(ctx, "clickhouse_max_threads", constants.MetricsExplorerClickhouseThreads)
	rows, err := r.db.Query(valueCtx, query, req.CurrentMetricName, req.CurrentMetricName, req.CurrentMetricName, start, end, normalized)
	if err != nil {
		return nil, &model.ApiError{Typ: "ClickHouseError", Err: err}
	}
	defer rows.Close()

	result := make(map[string]metrics_explorer.RelatedMetricsScore)
	for rows.Next() {
		var metric string
		var sim float64
		var metricType querytypes.MetricType
		var temporality querytypes.Temporality
		var isMonotonic bool
		if err := rows.Scan(&metric, &metricType, &temporality, &isMonotonic, &sim); err != nil {
			return nil, &model.ApiError{Typ: "ClickHouseError", Err: err}
		}
		result[metric] = metrics_explorer.RelatedMetricsScore{
			NameSimilarity: sim,
			MetricType:     metricType,
			Temporality:    temporality,
			IsMonotonic:    isMonotonic,
		}
	}

	return result, nil
}

func (r *ClickHouseReader) GetAttributeSimilarity(ctx context.Context, req *metrics_explorer.RelatedMetricsRequest) (map[string]metrics_explorer.RelatedMetricsScore, *model.ApiError) {
	ctx = ctxtypes.NewContextWithCommentVals(ctx, map[string]string{
		instrumentationtypes.TelemetrySignal:  telemetrytypes.SignalMetrics.StringValue(),
		instrumentationtypes.CodeNamespace:    "clickhouse-reader",
		instrumentationtypes.CodeFunctionName: "GetAttributeSimilarity",
	})
	start, end, tsTable, _ := utils.WhichTSTableToUse(req.Start, req.End)

	normalized := true
	if constants.IsDotMetricsEnabled {
		normalized = false
	}

	// Get target labels
	extractedLabelsQuery := fmt.Sprintf(`
		SELECT 
			kv.1 AS label_key,
			topK(10)(JSONExtractString(kv.2)) AS label_values
		FROM %s.%s
		ARRAY JOIN JSONExtractKeysAndValuesRaw(labels) AS kv
		WHERE metric_name = ?
		  AND unix_milli between ? and ?
		  AND NOT startsWith(kv.1, '__')
		AND NOT startsWith(metric_name, 'signoz_')
		AND __normalized = ?
		GROUP BY label_key
		LIMIT 50`, signozMetricDBName, tsTable)

	valueCtx := context.WithValue(ctx, "clickhouse_max_threads", constants.MetricsExplorerClickhouseThreads)
	rows, err := r.db.Query(valueCtx, extractedLabelsQuery, req.CurrentMetricName, start, end, normalized)
	if err != nil {
		return nil, &model.ApiError{Typ: "ClickHouseError", Err: err}
	}
	defer rows.Close()

	var targetKeys []string
	var targetValues []string
	for rows.Next() {
		var key string
		var value []string
		if err := rows.Scan(&key, &value); err != nil {
			return nil, &model.ApiError{Typ: "ClickHouseError", Err: err}
		}
		targetKeys = append(targetKeys, key)
		targetValues = append(targetValues, value...)
	}

	targetKeysList := "'" + strings.Join(targetKeys, "', '") + "'"
	targetValuesList := "'" + strings.Join(targetValues, "', '") + "'"

	var priorityList []string
	for _, f := range req.Filters.Items {
		if f.Operator == querytypes.FilterOperatorEqual {
			priorityList = append(priorityList, fmt.Sprintf("tuple('%s', '%s')", f.Key.Key, f.Value))
		}
	}
	priorityListString := strings.Join(priorityList, ", ")

	candidateLabelsQuery := fmt.Sprintf(`
		WITH 
			arrayDistinct([%s]) AS filter_keys,     
			arrayDistinct([%s]) AS filter_values,
			[%s] AS priority_pairs_input,
			%d AS priority_multiplier
		SELECT 
			metric_name,
			any(type) as type,
			any(temporality) as temporality,
			any(is_monotonic) as monotonic,
			SUM(
				arraySum(
					kv -> if(has(filter_keys, kv.1) AND has(filter_values, kv.2), 1, 0),
					JSONExtractKeysAndValues(labels, 'String')
				)
			)::UInt64 AS raw_match_count,
			SUM(
				arraySum(
					kv ->
						if(
							arrayExists(pr -> pr.1 = kv.1 AND pr.2 = kv.2, priority_pairs_input),
							priority_multiplier,
							0
						),
					JSONExtractKeysAndValues(labels, 'String')
				)
			)::UInt64 AS weighted_match_count,
		toJSONString(
			arrayDistinct(
				arrayFlatten(
					groupArray(
						arrayFilter(
							kv -> arrayExists(pr -> pr.1 = kv.1 AND pr.2 = kv.2, priority_pairs_input),
							JSONExtractKeysAndValues(labels, 'String')
						)
					)
				)
			)
		) AS priority_pairs
		FROM %s.%s
		WHERE rand() %% 100 < 10
		AND unix_milli between ? and ?
		AND NOT startsWith(metric_name, 'signoz_')
		AND __normalized = ?
		GROUP BY metric_name
		ORDER BY weighted_match_count DESC, raw_match_count DESC
		LIMIT 30
		`,
		targetKeysList, targetValuesList, priorityListString, 2,
		signozMetricDBName, tsTable)

	rows, err = r.db.Query(valueCtx, candidateLabelsQuery, start, end, normalized)
	if err != nil {
		return nil, &model.ApiError{Typ: "ClickHouseError", Err: err}
	}
	defer rows.Close()

	result := make(map[string]metrics_explorer.RelatedMetricsScore)
	attributeMap := make(map[string]uint64)

	for rows.Next() {
		var metric string
		var metricType querytypes.MetricType
		var temporality querytypes.Temporality
		var isMonotonic bool
		var weightedMatchCount, rawMatchCount uint64
		var priorityPairsJSON string

		if err := rows.Scan(&metric, &metricType, &temporality, &isMonotonic, &rawMatchCount, &weightedMatchCount, &priorityPairsJSON); err != nil {
			return nil, &model.ApiError{Typ: "ClickHouseError", Err: err}
		}

		attributeMap[metric] = weightedMatchCount + (rawMatchCount)/10
		var priorityPairs [][]string
		if err := json.Unmarshal([]byte(priorityPairsJSON), &priorityPairs); err != nil {
			priorityPairs = [][]string{}
		}

		result[metric] = metrics_explorer.RelatedMetricsScore{
			AttributeSimilarity: float64(attributeMap[metric]), // Will be normalized later
			Filters:             priorityPairs,
			MetricType:          metricType,
			Temporality:         temporality,
			IsMonotonic:         isMonotonic,
		}
	}

	if err := rows.Err(); err != nil {
		return nil, &model.ApiError{Typ: "ClickHouseError", Err: err}
	}

	// Normalize the attribute similarity scores
	normalizeMap := utils.NormalizeMap(attributeMap)
	for metric := range result {
		if score, exists := normalizeMap[metric]; exists {
			metricScore := result[metric]
			metricScore.AttributeSimilarity = score
			result[metric] = metricScore
		}
	}

	return result, nil
}

func (r *ClickHouseReader) GetMetricsAllResourceAttributes(ctx context.Context, start int64, end int64) (map[string]uint64, *model.ApiError) {
	ctx = ctxtypes.NewContextWithCommentVals(ctx, map[string]string{
		instrumentationtypes.TelemetrySignal:  telemetrytypes.SignalMetrics.StringValue(),
		instrumentationtypes.CodeNamespace:    "clickhouse-reader",
		instrumentationtypes.CodeFunctionName: "GetMetricsAllResourceAttributes",
	})
	start, end, attTable, _ := utils.WhichAttributesTableToUse(start, end)
	query := fmt.Sprintf(`SELECT 
    key, 
    count(distinct value) AS distinct_value_count
FROM (
    SELECT key, value
    FROM %s.%s
    ARRAY JOIN 
        arrayConcat(mapKeys(resource_attributes)) AS key,
        arrayConcat(mapValues(resource_attributes)) AS value
    WHERE unix_milli between ? and ?
) 
GROUP BY key
ORDER BY distinct_value_count DESC;`, signozMetadataDbName, attTable)
	valueCtx := context.WithValue(ctx, "clickhouse_max_threads", constants.MetricsExplorerClickhouseThreads)
	rows, err := r.db.Query(valueCtx, query, start, end)
	if err != nil {
		return nil, &model.ApiError{Typ: "ClickHouseError", Err: err}
	}
	attributes := make(map[string]uint64)
	for rows.Next() {
		var attrs string
		var uniqCount uint64

		if err := rows.Scan(&attrs, &uniqCount); err != nil {
			return nil, &model.ApiError{Typ: "ClickHouseError", Err: err}
		}
		attributes[attrs] = uniqCount
	}
	if err := rows.Err(); err != nil {
		return nil, &model.ApiError{Typ: "ClickHouseError", Err: err}
	}
	return attributes, nil
}

func (r *ClickHouseReader) GetInspectMetrics(ctx context.Context, req *metrics_explorer.InspectMetricsRequest, fingerprints []string) (*metrics_explorer.InspectMetricsResponse, *model.ApiError) {
	ctx = ctxtypes.NewContextWithCommentVals(ctx, map[string]string{
		instrumentationtypes.TelemetrySignal:  telemetrytypes.SignalMetrics.StringValue(),
		instrumentationtypes.CodeNamespace:    "clickhouse-reader",
		instrumentationtypes.CodeFunctionName: "GetInspectMetrics",
	})
	start, end, _, localTsTable := utils.WhichTSTableToUse(req.Start, req.End)
	fingerprintsString := strings.Join(fingerprints, ",")
	query := fmt.Sprintf(`SELECT
                fingerprint,
                labels,
                unix_milli,
                value as per_series_value
        FROM
                signoz_metrics.samples_v4
        INNER JOIN (
                SELECT DISTINCT
                        fingerprint,
                        labels
                FROM
                        %s.%s
                WHERE
                        fingerprint in (%s)
                        AND unix_milli >= ?
                        AND unix_milli < ?) as filtered_time_series
                USING fingerprint
        WHERE
                metric_name  = ?
                AND unix_milli >= ?
                AND unix_milli < ?
                ORDER BY fingerprint DESC, unix_milli DESC`, signozMetricDBName, localTsTable, fingerprintsString)
	valueCtx := context.WithValue(ctx, "clickhouse_max_threads", constants.MetricsExplorerClickhouseThreads)
	rows, err := r.db.Query(valueCtx, query, start, end, req.MetricName, start, end)
	if err != nil {
		return nil, &model.ApiError{Typ: "ClickHouseError", Err: err}
	}
	defer rows.Close()

	seriesMap := make(map[uint64]*timeseriestypes.Series)

	for rows.Next() {
		var fingerprint uint64
		var labelsJSON string
		var unixMilli int64
		var perSeriesValue float64

		if err := rows.Scan(&fingerprint, &labelsJSON, &unixMilli, &perSeriesValue); err != nil {
			return nil, &model.ApiError{Typ: "ClickHouseError", Err: err}
		}

		var labelsMap map[string]string
		if err := json.Unmarshal([]byte(labelsJSON), &labelsMap); err != nil {
			return nil, &model.ApiError{Typ: "JsonUnmarshalError", Err: err}
		}

		// Filter out keys starting with "__"
		filteredLabelsMap := make(map[string]string)
		for k, v := range labelsMap {
			if !strings.HasPrefix(k, "__") {
				filteredLabelsMap[k] = v
			}
		}

		var labelsArray []map[string]string
		for k, v := range filteredLabelsMap {
			labelsArray = append(labelsArray, map[string]string{k: v})
		}

		// Check if we already have a Series for this fingerprint.
		series, exists := seriesMap[fingerprint]
		if !exists {
			series = &timeseriestypes.Series{
				Labels:      filteredLabelsMap,
				LabelsArray: labelsArray,
				Points:      []timeseriestypes.Point{},
			}
			seriesMap[fingerprint] = series
		}

		series.Points = append(series.Points, timeseriestypes.Point{
			Timestamp: unixMilli,
			Value:     perSeriesValue,
		})
	}

	if err = rows.Err(); err != nil {
		return nil, &model.ApiError{Typ: "ClickHouseError", Err: err}
	}

	var seriesList []timeseriestypes.Series
	for _, s := range seriesMap {
		seriesList = append(seriesList, *s)
	}

	return &metrics_explorer.InspectMetricsResponse{
		Series: &seriesList,
	}, nil
}

func (r *ClickHouseReader) GetInspectMetricsFingerprints(ctx context.Context, attributes []string, req *metrics_explorer.InspectMetricsRequest) ([]string, *model.ApiError) {
	ctx = ctxtypes.NewContextWithCommentVals(ctx, map[string]string{
		instrumentationtypes.TelemetrySignal:  telemetrytypes.SignalMetrics.StringValue(),
		instrumentationtypes.CodeNamespace:    "clickhouse-reader",
		instrumentationtypes.CodeFunctionName: "GetInspectMetricsFingerprints",
	})
	// Build dynamic key selections and JSON extracts
	var jsonExtracts []string
	var groupBys []string

	for i, attr := range attributes {
		keyAlias := fmt.Sprintf("key%d", i+1)
		jsonExtracts = append(jsonExtracts, fmt.Sprintf("JSONExtractString(labels, '%s') AS %s", attr, keyAlias))
		groupBys = append(groupBys, keyAlias)
	}

	conditions, _ := utils.BuildFilterConditions(&req.Filters, "")
	whereClause := ""
	if len(conditions) > 0 {
		whereClause = "AND " + strings.Join(conditions, " AND ")
	}

	start, end, tsTable, _ := utils.WhichTSTableToUse(req.Start, req.End)
	query := fmt.Sprintf(`
        SELECT 
    arrayDistinct(groupArray(toString(fingerprint))) AS fingerprints
FROM
(
    SELECT 
        metric_name, labels, fingerprint,
        %s
    FROM %s.%s
    WHERE metric_name = ?
      AND unix_milli BETWEEN ? AND ?
    %s
)
GROUP BY %s
ORDER BY length(fingerprints) DESC, rand()
LIMIT 40`, // added rand to get diff value every time we run this query
		strings.Join(jsonExtracts, ", "),
		signozMetricDBName, tsTable,
		whereClause,
		strings.Join(groupBys, ", "))
	valueCtx := context.WithValue(ctx, "clickhouse_max_threads", constants.MetricsExplorerClickhouseThreads)
	rows, err := r.db.Query(valueCtx, query,
		req.MetricName,
		start,
		end,
	)
	if err != nil {
		return nil, &model.ApiError{Typ: model.ErrorExec, Err: err}
	}
	defer rows.Close()

	var fingerprints []string
	for rows.Next() {
		// Create dynamic scanning based on number of attributes
		var batch []string

		if err := rows.Scan(&batch); err != nil {
			return nil, &model.ApiError{Typ: model.ErrorExec, Err: err}
		}

		remaining := 40 - len(fingerprints)
		if remaining <= 0 {
			break
		}

		// if this batch would overshoot, only take as many as we need
		if len(batch) > remaining {
			fingerprints = append(fingerprints, batch[:remaining]...)
			break
		}

		// otherwise take the whole batch and keep going
		fingerprints = append(fingerprints, batch...)

	}

	if err := rows.Err(); err != nil {
		return nil, &model.ApiError{Typ: model.ErrorExec, Err: err}
	}

	return fingerprints, nil
}

func (r *ClickHouseReader) CheckForLabelsInMetric(ctx context.Context, metricName string, labels []string) (bool, *model.ApiError) {
	ctx = ctxtypes.NewContextWithCommentVals(ctx, map[string]string{
		instrumentationtypes.TelemetrySignal:  telemetrytypes.SignalMetrics.StringValue(),
		instrumentationtypes.CodeNamespace:    "clickhouse-reader",
		instrumentationtypes.CodeFunctionName: "CheckForLabelsInMetric",
	})
	if len(labels) == 0 {
		return true, nil
	}

	conditions := "metric_name = ?"
	for range labels {
		conditions += " AND JSONHas(labels, ?) = 1"
	}

	query := fmt.Sprintf(`
        SELECT count(*) > 0 as has_le
        FROM %s.%s
        WHERE %s
        LIMIT 1`, signozMetricDBName, signozTSTableNameV41Day, conditions)

	args := make([]interface{}, 0, len(labels)+1)
	args = append(args, metricName)
	for _, label := range labels {
		args = append(args, label)
	}

	var hasLE bool
	valueCtx := context.WithValue(ctx, "clickhouse_max_threads", constants.MetricsExplorerClickhouseThreads)
	err := r.db.QueryRow(valueCtx, query, args...).Scan(&hasLE)
	if err != nil {
		return false, &model.ApiError{
			Typ: "ClickHouseError",
			Err: fmt.Errorf("error checking summary labels: %v", err),
		}
	}
	return hasLE, nil
}

func (r *ClickHouseReader) GetUpdatedMetricsMetadata(ctx context.Context, orgID valuer.UUID, metricNames ...string) (map[string]*model.UpdateMetricsMetadata, *model.ApiError) {
	ctx = ctxtypes.NewContextWithCommentVals(ctx, map[string]string{
		instrumentationtypes.TelemetrySignal:  telemetrytypes.SignalMetrics.StringValue(),
		instrumentationtypes.CodeNamespace:    "clickhouse-reader",
		instrumentationtypes.CodeFunctionName: "GetUpdatedMetricsMetadata",
	})
	cachedMetadata := make(map[string]*model.UpdateMetricsMetadata)
	var missingMetrics []string

	// 1. Try cache
	for _, metricName := range metricNames {
		metadata := new(model.UpdateMetricsMetadata)
		cacheKey := constants.UpdatedMetricsMetadataCachePrefix + metricName
		err := r.cache.Get(ctx, orgID, cacheKey, metadata)
		if err == nil {
			cachedMetadata[metricName] = metadata
		} else {
			missingMetrics = append(missingMetrics, metricName)
		}
	}

	// 2. Try updated_metrics_metadata table
	var stillMissing []string
	if len(missingMetrics) > 0 {
		metricList := "'" + strings.Join(missingMetrics, "', '") + "'"
		query := fmt.Sprintf(`SELECT 
						metric_name,
						argMax(type, created_at) AS type,
						argMax(description, created_at) AS description,
						argMax(temporality, created_at) AS temporality,
						argMax(is_monotonic, created_at) AS is_monotonic,
						argMax(unit, created_at) AS unit
					FROM %s.%s 
					WHERE metric_name IN (%s)
					GROUP BY metric_name;`,
			signozMetricDBName,
			signozUpdatedMetricsMetadataTable,
			metricList)

		valueCtx := context.WithValue(ctx, "clickhouse_max_threads", constants.MetricsExplorerClickhouseThreads)
		rows, err := r.db.Query(valueCtx, query)
		if err != nil {
			return cachedMetadata, &model.ApiError{Typ: "ClickhouseErr", Err: fmt.Errorf("error querying metrics metadata: %v", err)}
		}
		defer rows.Close()

		found := make(map[string]struct{})
		for rows.Next() {
			metadata := new(model.UpdateMetricsMetadata)
			if err := rows.Scan(
				&metadata.MetricName,
				&metadata.MetricType,
				&metadata.Description,
				&metadata.Temporality,
				&metadata.IsMonotonic,
				&metadata.Unit,
			); err != nil {
				return cachedMetadata, &model.ApiError{Typ: "ClickhouseErr", Err: fmt.Errorf("error scanning metrics metadata: %v", err)}
			}

			cacheKey := constants.UpdatedMetricsMetadataCachePrefix + metadata.MetricName
			if cacheErr := r.cache.Set(ctx, orgID, cacheKey, metadata, 0); cacheErr != nil {
				r.logger.Error("Failed to store metrics metadata in cache", "metric_name", metadata.MetricName, errorsV2.Attr(cacheErr))
			}
			cachedMetadata[metadata.MetricName] = metadata
			found[metadata.MetricName] = struct{}{}
		}

		// Determine which metrics are still missing
		for _, m := range missingMetrics {
			if _, ok := found[m]; !ok {
				stillMissing = append(stillMissing, m)
			}
		}
	}

	// 3. Fallback: Try time_series_v4_1week table
	if len(stillMissing) > 0 {
		metricList := "'" + strings.Join(stillMissing, "', '") + "'"
		query := fmt.Sprintf(`SELECT DISTINCT metric_name, type, description, temporality, is_monotonic, unit
			FROM %s.%s 
			WHERE metric_name IN (%s)`, signozMetricDBName, signozTSTableNameV4, metricList)
		valueCtx := context.WithValue(ctx, "clickhouse_max_threads", constants.MetricsExplorerClickhouseThreads)
		rows, err := r.db.Query(valueCtx, query)
		if err != nil {
			return cachedMetadata, &model.ApiError{Typ: "ClickhouseErr", Err: fmt.Errorf("error querying time_series_v4 to get metrics metadata: %v", err)}
		}
		defer rows.Close()
		for rows.Next() {
			metadata := new(model.UpdateMetricsMetadata)
			if err := rows.Scan(
				&metadata.MetricName,
				&metadata.MetricType,
				&metadata.Description,
				&metadata.Temporality,
				&metadata.IsMonotonic,
				&metadata.Unit,
			); err != nil {
				return cachedMetadata, &model.ApiError{Typ: "ClickhouseErr", Err: fmt.Errorf("error scanning fallback metadata: %v", err)}
			}

			cacheKey := constants.UpdatedMetricsMetadataCachePrefix + metadata.MetricName
			if cacheErr := r.cache.Set(ctx, orgID, cacheKey, metadata, 0); cacheErr != nil {
				r.logger.Error("Failed to cache fallback metadata", "metric_name", metadata.MetricName, errorsV2.Attr(cacheErr))
			}
			cachedMetadata[metadata.MetricName] = metadata
		}
		if rows.Err() != nil {
			return cachedMetadata, &model.ApiError{Typ: "ClickhouseErr", Err: fmt.Errorf("error scanning fallback metadata: %v", err)}
		}
	}
	return cachedMetadata, nil
}
