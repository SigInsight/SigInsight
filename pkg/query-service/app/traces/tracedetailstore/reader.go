package tracedetailstore

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"sort"
	"strconv"
	"strings"
	"time"

	"log/slog"

	"github.com/ClickHouse/clickhouse-go/v2"
	"github.com/pkg/errors"

	"github.com/SigNoz/signoz/pkg/cache"
	errorsV2 "github.com/SigNoz/signoz/pkg/errors"
	"github.com/SigNoz/signoz/pkg/query-service/app/traces/tracedetail"
	"github.com/SigNoz/signoz/pkg/query-service/model"
	"github.com/SigNoz/signoz/pkg/types/ctxtypes"
	"github.com/SigNoz/signoz/pkg/types/instrumentationtypes"
	"github.com/SigNoz/signoz/pkg/types/telemetrytypes"
	"github.com/SigNoz/signoz/pkg/valuer"
)

// Config identifies the ClickHouse tables that hold trace detail data.
type Config struct {
	TraceDB           string
	TraceTableName    string
	TraceSummaryTable string
	FluxInterval      time.Duration
}

// Reader owns the ClickHouse query, cache, and tree assembly used by trace
// waterfall and flamegraph views.
type Reader struct {
	db                         clickhouse.Conn
	logger                     *slog.Logger
	traceDB                    string
	traceTableName             string
	traceSummaryTable          string
	fluxIntervalForTraceDetail time.Duration
	cacheForTraceDetail        cache.Cache
}

func New(
	logger *slog.Logger,
	db clickhouse.Conn,
	cacheForTraceDetail cache.Cache,
	config Config,
) *Reader {
	return &Reader{
		db:                         db,
		logger:                     logger,
		traceDB:                    config.TraceDB,
		traceTableName:             config.TraceTableName,
		traceSummaryTable:          config.TraceSummaryTable,
		fluxIntervalForTraceDetail: config.FluxInterval,
		cacheForTraceDetail:        cacheForTraceDetail,
	}
}

func (r *Reader) getSpansForTrace(ctx context.Context, traceID string, traceDetailsQuery string) ([]model.SpanItemV2, error) {
	ctx = ctxtypes.NewContextWithCommentVals(ctx, map[string]string{
		instrumentationtypes.TelemetrySignal:  telemetrytypes.SignalTraces.StringValue(),
		instrumentationtypes.CodeNamespace:    "clickhouse-reader",
		instrumentationtypes.CodeFunctionName: "GetSpansForTrace",
	})

	var traceSummary model.TraceSummary
	summaryQuery := fmt.Sprintf("SELECT trace_id, min(start) AS start, max(end) AS end, sum(num_spans) AS num_spans FROM %s.%s WHERE trace_id=$1 GROUP BY trace_id", r.traceDB, r.traceSummaryTable)
	err := r.db.QueryRow(ctx, summaryQuery, traceID).Scan(&traceSummary.TraceID, &traceSummary.Start, &traceSummary.End, &traceSummary.NumSpans)
	if err != nil {
		if err == sql.ErrNoRows {
			return []model.SpanItemV2{}, nil
		}
		r.logger.Error("Error in processing trace summary sql query", errorsV2.Attr(err))
		return nil, fmt.Errorf("error in processing trace summary sql query: %w", err)
	}

	var searchScanResponses []model.SpanItemV2
	queryStartTime := time.Now()
	err = r.db.Select(ctx, &searchScanResponses, traceDetailsQuery, traceID, strconv.FormatInt(traceSummary.Start.Unix()-1800, 10), strconv.FormatInt(traceSummary.End.Unix(), 10))
	r.logger.Info(traceDetailsQuery)
	if err != nil {
		r.logger.Error("Error in processing sql query", errorsV2.Attr(err))
		return nil, fmt.Errorf("error in processing trace data sql query: %w", err)
	}
	r.logger.Info("trace details query took: ", "duration", time.Since(queryStartTime), "traceID", traceID)

	return searchScanResponses, nil
}

func (r *Reader) getWaterfallSpansForTraceWithMetadataCache(ctx context.Context, orgID valuer.UUID, traceID string) (*model.GetWaterfallSpansForTraceWithMetadataCache, error) {
	cachedTraceData := new(model.GetWaterfallSpansForTraceWithMetadataCache)
	err := r.cacheForTraceDetail.Get(ctx, orgID, strings.Join([]string{"getWaterfallSpansForTraceWithMetadata", traceID}, "-"), cachedTraceData)
	if err != nil {
		r.logger.Debug("error in retrieving getWaterfallSpansForTraceWithMetadata cache", errorsV2.Attr(err), "traceID", traceID)
		return nil, err
	}

	if time.Since(time.Unix(0, int64(cachedTraceData.EndTime))) < r.fluxIntervalForTraceDetail {
		r.logger.Info("the trace end time falls under the flux interval, skipping getWaterfallSpansForTraceWithMetadata cache", "traceID", traceID)
		return nil, errors.Errorf("the trace end time falls under the flux interval, skipping getWaterfallSpansForTraceWithMetadata cache, traceID: %s", traceID)
	}

	r.logger.Info("cache is successfully hit, applying cache for getWaterfallSpansForTraceWithMetadata", "traceID", traceID)
	return cachedTraceData, nil
}

func (r *Reader) GetWaterfallSpansForTraceWithMetadata(ctx context.Context, orgID valuer.UUID, traceID string, req *model.GetWaterfallSpansForTraceWithMetadataParams) (*model.GetWaterfallSpansForTraceWithMetadataResponse, error) {
	response := new(model.GetWaterfallSpansForTraceWithMetadataResponse)
	var startTime, endTime, durationNano, totalErrorSpans, totalSpans uint64
	var spanIdToSpanNodeMap = map[string]*model.Span{}
	var traceRoots []*model.Span
	var serviceNameToTotalDurationMap = map[string]uint64{}
	var serviceNameIntervalMap = map[string][]tracedetail.Interval{}
	var hasMissingSpans bool

	cachedTraceData, err := r.getWaterfallSpansForTraceWithMetadataCache(ctx, orgID, traceID)
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

		searchScanResponses, err := r.getSpansForTrace(ctx, traceID, fmt.Sprintf("SELECT DISTINCT ON (span_id) timestamp, duration_nano, span_id, trace_id, has_error, kind, service_name, name, links as references, attributes_string, attributes_number, attributes_bool, resources_string, events, status_message, status_code_string, kind_string FROM %s.%s WHERE trace_id=$1 and ts_bucket_start>=$2 and ts_bucket_start<=$3 ORDER BY timestamp ASC, name ASC", r.traceDB, r.traceTableName))
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
				TimeUnixNano:     startTimeUnixNano,
			}

			if startTime == 0 || startTimeUnixNano < startTime {
				startTime = startTimeUnixNano
			}
			if endTime == 0 || (startTimeUnixNano+jsonItem.DurationNano) > endTime {
				endTime = startTimeUnixNano + jsonItem.DurationNano
			}
			if durationNano == 0 || jsonItem.DurationNano > durationNano {
				durationNano = jsonItem.DurationNano
			}
			if jsonItem.HasError {
				totalErrorSpans++
			}

			serviceNameIntervalMap[jsonItem.ServiceName] = append(serviceNameIntervalMap[jsonItem.ServiceName], tracedetail.Interval{StartTime: jsonItem.TimeUnixNano, Duration: jsonItem.DurationNano, Service: jsonItem.ServiceName})
			spanIdToSpanNodeMap[jsonItem.SpanID] = &jsonItem
		}

		for _, spanNode := range spanIdToSpanNodeMap {
			hasParentSpanNode := false
			for _, reference := range spanNode.References {
				if reference.RefType == "CHILD_OF" && reference.SpanId != "" {
					hasParentSpanNode = true
					if parentNode, exists := spanIdToSpanNodeMap[reference.SpanId]; exists {
						parentNode.Children = append(parentNode.Children, spanNode)
					} else {
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

	for _, span := range selectedSpans {
		span.TimeUnixNano /= 1000000
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

func (r *Reader) getFlamegraphSpansForTraceCache(ctx context.Context, orgID valuer.UUID, traceID string) (*model.GetFlamegraphSpansForTraceCache, error) {
	cachedTraceData := new(model.GetFlamegraphSpansForTraceCache)
	err := r.cacheForTraceDetail.Get(ctx, orgID, strings.Join([]string{"getFlamegraphSpansForTrace", traceID}, "-"), cachedTraceData)
	if err != nil {
		r.logger.Debug("error in retrieving getFlamegraphSpansForTrace cache", errorsV2.Attr(err), "traceID", traceID)
		return nil, err
	}

	if time.Since(time.Unix(0, int64(cachedTraceData.EndTime))) < r.fluxIntervalForTraceDetail {
		r.logger.Info("the trace end time falls under the flux interval, skipping getFlamegraphSpansForTrace cache", "traceID", traceID)
		return nil, errors.Errorf("the trace end time falls under the flux interval, skipping getFlamegraphSpansForTrace cache, traceID: %s", traceID)
	}

	r.logger.Info("cache is successfully hit, applying cache for getFlamegraphSpansForTrace", "traceID", traceID)
	return cachedTraceData, nil
}

func (r *Reader) GetFlamegraphSpansForTrace(ctx context.Context, orgID valuer.UUID, traceID string, req *model.GetFlamegraphSpansForTraceParams) (*model.GetFlamegraphSpansForTraceResponse, error) {
	trace := new(model.GetFlamegraphSpansForTraceResponse)
	var startTime, endTime, durationNano uint64
	var spanIdToSpanNodeMap = map[string]*model.FlamegraphSpan{}
	var selectedSpans = [][]*model.FlamegraphSpan{}
	var traceRoots []*model.FlamegraphSpan

	cachedTraceData, err := r.getFlamegraphSpansForTraceCache(ctx, orgID, traceID)
	if err == nil {
		startTime = cachedTraceData.StartTime
		endTime = cachedTraceData.EndTime
		durationNano = cachedTraceData.DurationNano
		selectedSpans = cachedTraceData.SelectedSpans
		traceRoots = cachedTraceData.TraceRoots
	}

	if err != nil {
		r.logger.Info("cache miss for getFlamegraphSpansForTrace", "traceID", traceID)

		searchScanResponses, err := r.getSpansForTrace(ctx, traceID, fmt.Sprintf("SELECT timestamp, duration_nano, span_id, trace_id, has_error,links as references, service_name, name, events FROM %s.%s WHERE trace_id=$1 and ts_bucket_start>=$2 and ts_bucket_start<=$3 ORDER BY timestamp ASC, name ASC", r.traceDB, r.traceTableName))
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

			startTimeUnixNano := uint64(item.TimeUnixNano.UnixNano())
			if startTime == 0 || startTimeUnixNano < startTime {
				startTime = startTimeUnixNano
			}
			if endTime == 0 || (startTimeUnixNano+jsonItem.DurationNano) > endTime {
				endTime = startTimeUnixNano + jsonItem.DurationNano
			}
			if durationNano == 0 || jsonItem.DurationNano > durationNano {
				durationNano = jsonItem.DurationNano
			}

			jsonItem.TimeUnixNano = uint64(item.TimeUnixNano.UnixNano() / 1000000)
			spanIdToSpanNodeMap[jsonItem.SpanID] = &jsonItem
		}

		for _, spanNode := range spanIdToSpanNodeMap {
			hasParentSpanNode := false
			for _, reference := range spanNode.References {
				if reference.RefType == "CHILD_OF" && reference.SpanId != "" {
					hasParentSpanNode = true
					if parentNode, exists := spanIdToSpanNodeMap[reference.SpanId]; exists {
						parentNode.Children = append(parentNode.Children, spanNode)
					} else {
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
