package interfaces

import (
	"context"
	"github.com/SigNoz/signoz/pkg/types/timeseriestypes"
	"time"

	"github.com/SigNoz/signoz/pkg/query-service/model"
	"github.com/SigNoz/signoz/pkg/query-service/model/metrics_explorer"
	"github.com/SigNoz/signoz/pkg/query-service/model/querytypes"
	"github.com/SigNoz/signoz/pkg/query-service/querycache"
	"github.com/SigNoz/signoz/pkg/valuer"
)

type Reader interface {
	GetTopLevelOperations(ctx context.Context, start, end time.Time, services []string) (*map[string][]string, *model.ApiError)
	GetUsage(ctx context.Context, query *model.GetUsageParams) (*[]model.UsageItem, error)
	GetDependencyGraph(ctx context.Context, query *model.GetServicesParams) (*[]model.ServiceMapDependencyResponseItem, error)

	GetTTL(ctx context.Context, orgID string, ttlParams *model.GetTTLParams) (*model.GetTTLResponseItem, *model.ApiError)
	GetCustomRetentionTTL(ctx context.Context, orgID string) (*model.GetCustomRetentionTTLResponse, error)

	// GetDisks returns a list of disks configured in the underlying DB. It is supported by
	// clickhouse only.
	GetDisks(ctx context.Context) (*[]model.DiskItem, *model.ApiError)
	GetTraceAggregateAttributes(ctx context.Context, req *querytypes.AggregateAttributeRequest) (*querytypes.AggregateAttributeResponse, error)
	GetTraceAttributeKeys(ctx context.Context, req *querytypes.FilterAttributeKeyRequest) (*querytypes.FilterAttributeKeyResponse, error)
	GetTraceAttributeValues(ctx context.Context, req *querytypes.FilterAttributeValueRequest) (*querytypes.FilterAttributeValueResponse, error)
	GetSpanAttributeKeysByNames(ctx context.Context, names []string) (map[string]querytypes.AttributeKey, error)

	ListErrors(ctx context.Context, params *model.ListErrorsParams) (*[]model.Error, *model.ApiError)
	CountErrors(ctx context.Context, params *model.CountErrorsParams) (uint64, *model.ApiError)
	GetErrorFromErrorID(ctx context.Context, params *model.GetErrorParams) (*model.ErrorWithSpan, *model.ApiError)
	GetErrorFromGroupID(ctx context.Context, params *model.GetErrorParams) (*model.ErrorWithSpan, *model.ApiError)
	GetNextPrevErrorIDs(ctx context.Context, params *model.GetErrorParams) (*model.NextPrevErrorIDs, *model.ApiError)

	// Search Interfaces
	SearchTraces(ctx context.Context, params *model.SearchTracesParams) (*[]model.SearchSpansResult, error)
	GetWaterfallSpansForTraceWithMetadata(ctx context.Context, orgID valuer.UUID, traceID string, req *model.GetWaterfallSpansForTraceWithMetadataParams) (*model.GetWaterfallSpansForTraceWithMetadataResponse, error)
	GetFlamegraphSpansForTrace(ctx context.Context, orgID valuer.UUID, traceID string, req *model.GetFlamegraphSpansForTraceParams) (*model.GetFlamegraphSpansForTraceResponse, error)

	// Setter Interfaces
	SetTTL(ctx context.Context, orgID string, ttlParams *model.TTLParams) (*model.SetTTLResponseItem, *model.ApiError)
	SetTTLV2(ctx context.Context, orgID string, params *model.CustomRetentionTTLParams) (*model.CustomRetentionTTLResponse, error)

	GetMetricAggregateAttributes(ctx context.Context, orgID valuer.UUID, req *querytypes.AggregateAttributeRequest, skipSignozMetrics bool) (*querytypes.AggregateAttributeResponse, error)
	GetMeterAggregateAttributes(ctx context.Context, orgID valuer.UUID, req *querytypes.AggregateAttributeRequest) (*querytypes.AggregateAttributeResponse, error)
	GetMetricAttributeKeys(ctx context.Context, req *querytypes.FilterAttributeKeyRequest) (*querytypes.FilterAttributeKeyResponse, error)
	GetMeterAttributeKeys(ctx context.Context, req *querytypes.FilterAttributeKeyRequest) (*querytypes.FilterAttributeKeyResponse, error)
	GetMetricAttributeValues(ctx context.Context, req *querytypes.FilterAttributeValueRequest) (*querytypes.FilterAttributeValueResponse, error)

	// Returns `MetricStatus` for latest received metric among `metricNames`. Useful for status calculations
	GetLatestReceivedMetric(
		ctx context.Context, metricNames []string, labelValues map[string]string,
	) (*model.MetricStatus, *model.ApiError)

	// SQL result helpers used by specialized metric and log endpoints.
	GetTimeSeriesResult(ctx context.Context, query string) ([]*timeseriestypes.Series, error)
	GetListResult(ctx context.Context, query string) ([]*timeseriestypes.Row, error)
	// Logs
	GetLogAttributeKeys(ctx context.Context, req *querytypes.FilterAttributeKeyRequest) (*querytypes.FilterAttributeKeyResponse, error)
	GetLogAttributeValues(ctx context.Context, req *querytypes.FilterAttributeValueRequest) (*querytypes.FilterAttributeValueResponse, error)
	GetLogAggregateAttributes(ctx context.Context, req *querytypes.AggregateAttributeRequest) (*querytypes.AggregateAttributeResponse, error)
	GetQBFilterSuggestionsForLogs(
		ctx context.Context,
		req *querytypes.QBFilterSuggestionsRequest,
	) (*querytypes.QBFilterSuggestionsResponse, *model.ApiError)

	CheckClickHouse(ctx context.Context) error

	GetMetricMetadata(context.Context, valuer.UUID, string, string) (*querytypes.MetricMetadataResponse, error)

	AddRuleStateHistory(ctx context.Context, ruleStateHistory []model.RuleStateHistory) error
	GetOverallStateTransitions(ctx context.Context, ruleID string, params *model.QueryRuleStateHistory) ([]model.ReleStateItem, error)
	ReadRuleStateHistoryByRuleID(ctx context.Context, ruleID string, params *model.QueryRuleStateHistory) (*model.RuleStateTimeline, error)
	GetTotalTriggers(ctx context.Context, ruleID string, params *model.QueryRuleStateHistory) (uint64, error)
	GetTriggersByInterval(ctx context.Context, ruleID string, params *model.QueryRuleStateHistory) (*timeseriestypes.Series, error)
	GetAvgResolutionTime(ctx context.Context, ruleID string, params *model.QueryRuleStateHistory) (float64, error)
	GetAvgResolutionTimeByInterval(ctx context.Context, ruleID string, params *model.QueryRuleStateHistory) (*timeseriestypes.Series, error)
	ReadRuleStateHistoryTopContributorsByRuleID(ctx context.Context, ruleID string, params *model.QueryRuleStateHistory) ([]model.RuleStateHistoryContributor, error)
	GetLastSavedRuleStateHistory(ctx context.Context, ruleID string) ([]model.RuleStateHistory, error)

	GetMinAndMaxTimestampForTraceID(ctx context.Context, traceID []string) (int64, int64, error)

	// Query Progress tracking helpers.
	ReportQueryStartForProgressTracking(queryId string) (reportQueryFinished func(), apiErr *model.ApiError)
	SubscribeToQueryProgress(queryId string) (<-chan model.QueryProgress, func(), *model.ApiError)

	GetCountOfThings(ctx context.Context, query string) (uint64, error)
	GetActiveHostsFromMetricMetadata(ctx context.Context, metricNames []string, hostNameAttr string, sinceUnixMilli int64) (map[string]bool, error)

	GetMetricsExistenceAndEarliestTime(ctx context.Context, metricNames []string) (uint64, uint64, error)

	//trace
	GetTraceFields(ctx context.Context) (*model.GetFieldsResponse, *model.ApiError)
	UpdateTraceField(ctx context.Context, field *model.UpdateField) *model.ApiError

	GetAllMetricFilterAttributeValues(ctx context.Context, req *metrics_explorer.FilterValueRequest) ([]string, *model.ApiError)
	GetAllMetricFilterUnits(ctx context.Context, req *metrics_explorer.FilterValueRequest) ([]string, *model.ApiError)
	GetAllMetricFilterTypes(ctx context.Context, req *metrics_explorer.FilterValueRequest) ([]string, *model.ApiError)
	GetAllMetricFilterAttributeKeys(ctx context.Context, req *metrics_explorer.FilterKeyRequest) (*[]querytypes.AttributeKey, *model.ApiError)

	GetMetricsDataPoints(ctx context.Context, metricName string) (uint64, *model.ApiError)
	GetMetricsLastReceived(ctx context.Context, metricName string) (int64, *model.ApiError)
	GetTotalTimeSeriesForMetricName(ctx context.Context, metricName string) (uint64, *model.ApiError)
	GetActiveTimeSeriesForMetricName(ctx context.Context, metricName string, duration time.Duration) (uint64, *model.ApiError)
	GetAttributesForMetricName(ctx context.Context, metricName string, start, end *int64, set *querytypes.FilterSet) (*[]metrics_explorer.Attribute, *model.ApiError)

	ListSummaryMetrics(ctx context.Context, orgID valuer.UUID, req *metrics_explorer.SummaryListMetricsRequest) (*metrics_explorer.SummaryListMetricsResponse, *model.ApiError)

	GetMetricsTimeSeriesPercentage(ctx context.Context, request *metrics_explorer.TreeMapMetricsRequest) (*[]metrics_explorer.TreeMapResponseItem, *model.ApiError)
	GetMetricsSamplesPercentage(ctx context.Context, req *metrics_explorer.TreeMapMetricsRequest) (*[]metrics_explorer.TreeMapResponseItem, *model.ApiError)

	GetNameSimilarity(ctx context.Context, req *metrics_explorer.RelatedMetricsRequest) (map[string]metrics_explorer.RelatedMetricsScore, *model.ApiError)
	GetAttributeSimilarity(ctx context.Context, req *metrics_explorer.RelatedMetricsRequest) (map[string]metrics_explorer.RelatedMetricsScore, *model.ApiError)

	GetMetricsAllResourceAttributes(ctx context.Context, start int64, end int64) (map[string]uint64, *model.ApiError)
	GetInspectMetricsFingerprints(ctx context.Context, attributes []string, req *metrics_explorer.InspectMetricsRequest) ([]string, *model.ApiError)
	GetInspectMetrics(ctx context.Context, req *metrics_explorer.InspectMetricsRequest, fingerprints []string) (*metrics_explorer.InspectMetricsResponse, *model.ApiError)

	UpdateMetricsMetadata(ctx context.Context, orgID valuer.UUID, req *model.UpdateMetricsMetadata) *model.ApiError
	GetUpdatedMetricsMetadata(ctx context.Context, orgID valuer.UUID, metricNames ...string) (map[string]*model.UpdateMetricsMetadata, *model.ApiError)

	CheckForLabelsInMetric(ctx context.Context, metricName string, labels []string) (bool, *model.ApiError)
	GetNormalizedStatus(ctx context.Context, orgID valuer.UUID, metricNames []string) (map[string]bool, error)
}

type QueryCache interface {
	FindMissingTimeRanges(orgID valuer.UUID, start, end int64, step int64, cacheKey string) []querycache.MissInterval
	FindMissingTimeRangesV2(orgID valuer.UUID, start, end int64, step int64, cacheKey string) []querycache.MissInterval
	MergeWithCachedSeriesData(orgID valuer.UUID, cacheKey string, newData []querycache.CachedSeriesData) []querycache.CachedSeriesData
	StoreSeriesInCache(orgID valuer.UUID, cacheKey string, series []querycache.CachedSeriesData)
	MergeWithCachedSeriesDataV2(orgID valuer.UUID, cacheKey string, series []querycache.CachedSeriesData) []querycache.CachedSeriesData
}
