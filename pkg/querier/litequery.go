package querier

import (
	"context"
	stderrors "errors"
	"fmt"
	"strings"

	"github.com/SigNoz/signoz/pkg/errors"
	"github.com/SigNoz/signoz/pkg/litequery"
	"github.com/SigNoz/signoz/pkg/querier/liteadapter"
	"github.com/SigNoz/signoz/pkg/types/metrictypes"
	qbtypes "github.com/SigNoz/signoz/pkg/types/querybuildertypes/querybuildertypesv5"
)

// queryRangeLite is an opt-in bridge. A capability mismatch is deliberately
// reported to the caller as handled=false, preserving the legacy V5 path until
// every UI request has a lightweight equivalent.
func (q *querier) queryRangeLite(ctx context.Context, request *qbtypes.QueryRangeRequest) (*qbtypes.QueryRangeResponse, bool, error) {
	metadata, err := q.liteMetricMetadata(ctx, request)
	if err != nil {
		return nil, true, err
	}
	requestLite, err := liteadapter.ToLite(request, metadata)
	if err != nil {
		var unsupported *liteadapter.UnsupportedError
		if stderrors.As(err, &unsupported) {
			q.logger.DebugContext(ctx, "routing unsupported lightweight request to legacy querier", "feature", unsupported.Feature)
			return nil, false, nil
		}
		return nil, true, errors.NewInvalidInputf(errors.CodeInvalidInput, "invalid lightweight query: %v", err)
	}
	plan, err := (litequery.DefaultPlanner{}).Plan(requestLite)
	if err != nil {
		return nil, true, liteQueryError(err)
	}
	q.logger.DebugContext(ctx, "executing V5 query with lightweight engine", "queries", len(plan.Queries))
	executor := litequery.Executor{
		Compiler: litequery.NewCompiler(nil),
		Query: func(ctx context.Context, query string, args ...any) (litequery.Rows, error) {
			rows, err := q.telemetryStore.ClickhouseDB().Query(ctx, query, args...)
			if err != nil {
				return nil, err
			}
			return litequery.WrapClickHouseRows(rows), nil
		},
		Config: litequery.ExecutorConfig{MaxConcurrent: 4},
	}
	result, err := executor.Execute(ctx, plan)
	if err != nil {
		return nil, true, liteQueryError(err)
	}
	rowCounts := make([]string, 0, len(result.Queries))
	for _, query := range result.Queries {
		rowCounts = append(rowCounts, fmt.Sprintf("%s=%d", query.Name, len(query.Rows)))
	}
	q.logger.DebugContext(ctx, "lightweight V5 query completed", "queries", len(result.Queries), "result_rows", strings.Join(rowCounts, ","))
	response, err := liteadapter.FromLite(request, result)
	if err != nil {
		return nil, true, liteQueryError(err)
	}
	response.QBEvent = liteQueryEvent(request)
	return response, true, nil
}

func (q *querier) liteMetricMetadata(ctx context.Context, request *qbtypes.QueryRangeRequest) (liteadapter.MetricMetadata, error) {
	names := make([]string, 0)
	for _, envelope := range request.CompositeQuery.Queries {
		if query, ok := envelope.Spec.(qbtypes.QueryBuilderQuery[qbtypes.MetricAggregation]); ok {
			for _, aggregation := range query.Aggregations {
				if aggregation.MetricName != "" && (aggregation.Type == metrictypes.UnspecifiedType || aggregation.Temporality == metrictypes.Unknown) {
					names = append(names, aggregation.MetricName)
				}
			}
		}
	}
	if len(names) == 0 {
		return liteadapter.MetricMetadata{}, nil
	}
	if q.metadataStore == nil {
		return liteadapter.MetricMetadata{}, errors.NewInternalf(errors.CodeInternal, "metric metadata store is unavailable")
	}
	temporalities, types, err := q.metadataStore.FetchTemporalityAndTypeMulti(ctx, request.Start, request.End, names...)
	if err != nil {
		return liteadapter.MetricMetadata{}, errors.NewInternalf(errors.CodeInternal, "failed to fetch metric temporality and type")
	}
	return liteadapter.MetricMetadata{Temporality: temporalities, Types: types}, nil
}

func liteQueryError(err error) error {
	var queryError *litequery.Error
	if stderrors.As(err, &queryError) {
		return errors.NewInvalidInputf(errors.CodeInvalidInput, "invalid lightweight query: %s", queryError.Message)
	}
	return err
}

func liteQueryEvent(request *qbtypes.QueryRangeRequest) *qbtypes.QBEvent {
	event := &qbtypes.QBEvent{Version: "v5", NumberOfQueries: len(request.CompositeQuery.Queries), PanelType: request.RequestType.StringValue()}
	for _, envelope := range request.CompositeQuery.Queries {
		event.QueryType = envelope.Type.StringValue()
		switch spec := envelope.Spec.(type) {
		case qbtypes.QueryBuilderQuery[qbtypes.LogAggregation]:
			event.LogsUsed = true
			event.FilterApplied = spec.Filter != nil && strings.TrimSpace(spec.Filter.Expression) != ""
			event.GroupByApplied = len(spec.GroupBy) != 0
		case qbtypes.QueryBuilderQuery[qbtypes.TraceAggregation]:
			event.TracesUsed = true
			event.FilterApplied = spec.Filter != nil && strings.TrimSpace(spec.Filter.Expression) != ""
			event.GroupByApplied = len(spec.GroupBy) != 0
		case qbtypes.QueryBuilderQuery[qbtypes.MetricAggregation]:
			event.MetricsUsed = true
			event.Source = spec.Source.StringValue()
			event.FilterApplied = spec.Filter != nil && strings.TrimSpace(spec.Filter.Expression) != ""
			event.GroupByApplied = len(spec.GroupBy) != 0
		}
	}
	return event
}
