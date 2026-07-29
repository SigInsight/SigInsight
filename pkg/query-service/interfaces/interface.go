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

// ServiceReader serves service topology and operation metadata.
type ServiceReader interface {
	GetTopLevelOperations(ctx context.Context, start, end time.Time, services []string) (*map[string][]string, error)
	GetDependencyGraph(ctx context.Context, query *model.GetServicesParams) (*[]model.ServiceMapDependencyResponseItem, error)
}

// RetentionReader manages ClickHouse retention policies and configured disks.
type RetentionReader interface {
	GetTTL(ctx context.Context, orgID string, ttlParams *model.GetTTLParams) (*model.GetTTLResponseItem, error)
	GetCustomRetentionTTL(ctx context.Context, orgID string) (*model.GetCustomRetentionTTLResponse, error)
	GetDisks(ctx context.Context) (*[]model.DiskItem, error)
	SetTTL(ctx context.Context, orgID string, ttlParams *model.TTLParams) (*model.SetTTLResponseItem, error)
	SetCustomRetentionTTL(ctx context.Context, orgID string, params *model.CustomRetentionTTLParams) (*model.CustomRetentionTTLResponse, error)
}

// ExceptionReader serves the exceptions explorer.
type ExceptionReader interface {
	ListErrors(ctx context.Context, params *model.ListErrorsParams) (*[]model.Error, error)
	CountErrors(ctx context.Context, params *model.CountErrorsParams) (uint64, error)
	GetErrorFromErrorID(ctx context.Context, params *model.GetErrorParams) (*model.ErrorWithSpan, error)
	GetErrorFromGroupID(ctx context.Context, params *model.GetErrorParams) (*model.ErrorWithSpan, error)
	GetNextPrevErrorIDs(ctx context.Context, params *model.GetErrorParams) (*model.NextPrevErrorIDs, error)
}

// TraceDetailReader serves paginated waterfall and flamegraph trace views.
type TraceDetailReader interface {
	GetWaterfallSpansForTraceWithMetadata(ctx context.Context, orgID valuer.UUID, traceID string, req *model.GetWaterfallSpansForTraceWithMetadataParams) (*model.GetWaterfallSpansForTraceWithMetadataResponse, error)
	GetFlamegraphSpansForTrace(ctx context.Context, orgID valuer.UUID, traceID string, req *model.GetFlamegraphSpansForTraceParams) (*model.GetFlamegraphSpansForTraceResponse, error)
}

// ClickHouseHealthReader verifies storage availability for health endpoints.
type ClickHouseHealthReader interface {
	CheckClickHouse(ctx context.Context) error
}

// MetricMetadataReader serves the legacy metric metadata endpoint.
type MetricMetadataReader interface {
	GetMetricMetadata(context.Context, valuer.UUID, string, string) (*querytypes.MetricMetadataResponse, error)
}

// RuleStateHistoryQueryReader serves alert history views.
type RuleStateHistoryQueryReader interface {
	GetOverallStateTransitions(ctx context.Context, ruleID string, params *model.QueryRuleStateHistory) ([]model.ReleStateItem, error)
	ReadRuleStateHistoryByRuleID(ctx context.Context, ruleID string, params *model.QueryRuleStateHistory) (*model.RuleStateTimeline, error)
	GetTotalTriggers(ctx context.Context, ruleID string, params *model.QueryRuleStateHistory) (uint64, error)
	GetTriggersByInterval(ctx context.Context, ruleID string, params *model.QueryRuleStateHistory) (*timeseriestypes.Series, error)
	GetAvgResolutionTime(ctx context.Context, ruleID string, params *model.QueryRuleStateHistory) (float64, error)
	GetAvgResolutionTimeByInterval(ctx context.Context, ruleID string, params *model.QueryRuleStateHistory) (*timeseriestypes.Series, error)
	ReadRuleStateHistoryTopContributorsByRuleID(ctx context.Context, ruleID string, params *model.QueryRuleStateHistory) ([]model.RuleStateHistoryContributor, error)
}

// RuleStateHistoryReader is the persistence boundary used while evaluating
// alert rules. Rule evaluation itself is handled by the v5 querier.
type RuleStateHistoryReader interface {
	AddRuleStateHistory(ctx context.Context, ruleStateHistory []model.RuleStateHistory) error
	GetLastSavedRuleStateHistory(ctx context.Context, ruleID string) ([]model.RuleStateHistory, error)
}

// TraceFunnelQueryReader executes the dynamically shaped SQL produced by the
// trace-funnel query builder. Its results are intentionally not exposed to
// other query-service consumers.
type TraceFunnelQueryReader interface {
	ExecuteTraceFunnelQuery(ctx context.Context, query string) ([]*timeseriestypes.Row, error)
}

// MetricsExplorerReader contains the metadata and inspection queries used by
// the metrics explorer. Keeping it separate prevents that service from taking
// a dependency on unrelated trace, retention, and alert-history queries.
type MetricsExplorerReader interface {
	GetMetricAggregateAttributes(ctx context.Context, orgID valuer.UUID, req *querytypes.AggregateAttributeRequest, skipSignozMetrics bool) (*querytypes.AggregateAttributeResponse, error)
	GetAllMetricFilterAttributeValues(ctx context.Context, req *metrics_explorer.FilterValueRequest) ([]string, error)
	GetAllMetricFilterUnits(ctx context.Context, req *metrics_explorer.FilterValueRequest) ([]string, error)
	GetAllMetricFilterTypes(ctx context.Context, req *metrics_explorer.FilterValueRequest) ([]string, error)
	GetAllMetricFilterAttributeKeys(ctx context.Context, req *metrics_explorer.FilterKeyRequest) (*[]querytypes.AttributeKey, error)
	GetAttributesForMetricName(ctx context.Context, metricName string, start, end *int64, set *querytypes.FilterSet) (*[]metrics_explorer.Attribute, error)
	GetNameSimilarity(ctx context.Context, req *metrics_explorer.RelatedMetricsRequest) (map[string]metrics_explorer.RelatedMetricsScore, error)
	GetAttributeSimilarity(ctx context.Context, req *metrics_explorer.RelatedMetricsRequest) (map[string]metrics_explorer.RelatedMetricsScore, error)
	GetMetricsAllResourceAttributes(ctx context.Context, start int64, end int64) (map[string]uint64, error)
	GetInspectMetricsFingerprints(ctx context.Context, attributes []string, req *metrics_explorer.InspectMetricsRequest) ([]string, error)
	GetInspectMetrics(ctx context.Context, req *metrics_explorer.InspectMetricsRequest, fingerprints []string) (*metrics_explorer.InspectMetricsResponse, error)
}
