package interfaces

import (
	"context"
	"time"

	"github.com/SigNoz/signoz/pkg/query-service/model"
	"github.com/SigNoz/signoz/pkg/query-service/model/metrics_explorer"
	"github.com/SigNoz/signoz/pkg/query-service/model/querytypes"
	"github.com/SigNoz/signoz/pkg/types/timeseriestypes"
	"github.com/SigNoz/signoz/pkg/valuer"
)

type Reader interface {
	GetTopLevelOperations(ctx context.Context, start, end time.Time, services []string) (*map[string][]string, *model.ApiError)
	GetDependencyGraph(ctx context.Context, query *model.GetServicesParams) (*[]model.ServiceMapDependencyResponseItem, error)

	GetTTL(ctx context.Context, orgID string, ttlParams *model.GetTTLParams) (*model.GetTTLResponseItem, *model.ApiError)
	GetCustomRetentionTTL(ctx context.Context, orgID string) (*model.GetCustomRetentionTTLResponse, error)

	// GetDisks returns a list of disks configured in the underlying DB. It is supported by
	// clickhouse only.
	GetDisks(ctx context.Context) (*[]model.DiskItem, *model.ApiError)

	ListErrors(ctx context.Context, params *model.ListErrorsParams) (*[]model.Error, *model.ApiError)
	CountErrors(ctx context.Context, params *model.CountErrorsParams) (uint64, *model.ApiError)
	GetErrorFromErrorID(ctx context.Context, params *model.GetErrorParams) (*model.ErrorWithSpan, *model.ApiError)
	GetErrorFromGroupID(ctx context.Context, params *model.GetErrorParams) (*model.ErrorWithSpan, *model.ApiError)
	GetNextPrevErrorIDs(ctx context.Context, params *model.GetErrorParams) (*model.NextPrevErrorIDs, *model.ApiError)

	// Trace detail interfaces.
	GetWaterfallSpansForTraceWithMetadata(ctx context.Context, orgID valuer.UUID, traceID string, req *model.GetWaterfallSpansForTraceWithMetadataParams) (*model.GetWaterfallSpansForTraceWithMetadataResponse, error)
	GetFlamegraphSpansForTrace(ctx context.Context, orgID valuer.UUID, traceID string, req *model.GetFlamegraphSpansForTraceParams) (*model.GetFlamegraphSpansForTraceResponse, error)

	// Setter Interfaces
	SetTTL(ctx context.Context, orgID string, ttlParams *model.TTLParams) (*model.SetTTLResponseItem, *model.ApiError)
	SetCustomRetentionTTL(ctx context.Context, orgID string, params *model.CustomRetentionTTLParams) (*model.CustomRetentionTTLResponse, error)

	GetMetricAggregateAttributes(ctx context.Context, orgID valuer.UUID, req *querytypes.AggregateAttributeRequest, skipSignozMetrics bool) (*querytypes.AggregateAttributeResponse, error)

	// SQL result helpers used by specialized metric and log endpoints.
	GetTimeSeriesResult(ctx context.Context, query string) ([]*timeseriestypes.Series, error)
	GetListResult(ctx context.Context, query string) ([]*timeseriestypes.Row, error)
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

	GetAllMetricFilterAttributeValues(ctx context.Context, req *metrics_explorer.FilterValueRequest) ([]string, *model.ApiError)
	GetAllMetricFilterUnits(ctx context.Context, req *metrics_explorer.FilterValueRequest) ([]string, *model.ApiError)
	GetAllMetricFilterTypes(ctx context.Context, req *metrics_explorer.FilterValueRequest) ([]string, *model.ApiError)
	GetAllMetricFilterAttributeKeys(ctx context.Context, req *metrics_explorer.FilterKeyRequest) (*[]querytypes.AttributeKey, *model.ApiError)

	GetAttributesForMetricName(ctx context.Context, metricName string, start, end *int64, set *querytypes.FilterSet) (*[]metrics_explorer.Attribute, *model.ApiError)

	GetNameSimilarity(ctx context.Context, req *metrics_explorer.RelatedMetricsRequest) (map[string]metrics_explorer.RelatedMetricsScore, *model.ApiError)
	GetAttributeSimilarity(ctx context.Context, req *metrics_explorer.RelatedMetricsRequest) (map[string]metrics_explorer.RelatedMetricsScore, *model.ApiError)

	GetMetricsAllResourceAttributes(ctx context.Context, start int64, end int64) (map[string]uint64, *model.ApiError)
	GetInspectMetricsFingerprints(ctx context.Context, attributes []string, req *metrics_explorer.InspectMetricsRequest) ([]string, *model.ApiError)
	GetInspectMetrics(ctx context.Context, req *metrics_explorer.InspectMetricsRequest, fingerprints []string) (*metrics_explorer.InspectMetricsResponse, *model.ApiError)

	GetUpdatedMetricsMetadata(ctx context.Context, orgID valuer.UUID, metricNames ...string) (map[string]*model.UpdateMetricsMetadata, *model.ApiError)

	CheckForLabelsInMetric(ctx context.Context, metricName string, labels []string) (bool, *model.ApiError)
}
