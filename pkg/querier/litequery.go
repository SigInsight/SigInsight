package querier

import (
	"context"
	"fmt"
	"log/slog"
	"strings"

	"github.com/SigNoz/signoz/pkg/errors"
	"github.com/SigNoz/signoz/pkg/litequery"
	"github.com/SigNoz/signoz/pkg/querier/liteadapter"
	"github.com/SigNoz/signoz/pkg/querier/litemetadata"
	qbtypes "github.com/SigNoz/signoz/pkg/types/querybuildertypes/querybuildertypesv5"
)

// queryRangeLite executes the constrained V5 contract. Requests outside that
// contract are rejected at the boundary; there is no legacy V5 executor.
func (q *querier) queryRangeLite(ctx context.Context, request *qbtypes.QueryRangeRequest) (*qbtypes.QueryRangeResponse, error) {
	if err := liteadapter.ValidateRequestRange(request); err != nil {
		return nil, err
	}
	metadata, err := q.liteMetadata(ctx, request)
	if err != nil {
		return nil, err
	}
	requestLite, err := liteadapter.ToLite(request, metadata)
	if err != nil {
		var unsupported *liteadapter.UnsupportedError
		if errors.As(err, &unsupported) {
			return nil, errors.NewInvalidInputf(
				errors.CodeInvalidInput,
				"unsupported lightweight query capability: %s",
				unsupported.Feature,
			)
		}
		return nil, errors.NewInvalidInputf(errors.CodeInvalidInput, "invalid lightweight query: %v", err)
	}
	plan, err := (litequery.DefaultPlanner{}).Plan(requestLite)
	if err != nil {
		return nil, liteQueryError(err)
	}
	q.logger.DebugContext(ctx, "executing V5 query with lightweight engine", slog.Int("queries", len(plan.Queries)))
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
		return nil, liteQueryError(err)
	}
	rowCounts := make([]string, 0, len(result.Queries))
	for _, query := range result.Queries {
		rowCounts = append(rowCounts, fmt.Sprintf("%s=%d", query.Name, len(query.Rows)))
	}
	q.logger.DebugContext(ctx, "lightweight V5 query completed", slog.Int("queries", len(result.Queries)), slog.String("result_rows", strings.Join(rowCounts, ",")))
	response, err := liteadapter.FromLite(request, result)
	if err != nil {
		return nil, liteQueryError(err)
	}
	response.QBEvent = liteQueryEvent(request)
	response.QBEvent.HasData = liteResultHasData(result)
	return response, nil
}

func liteResultHasData(result litequery.ExecutionResult) bool {
	for _, query := range result.Queries {
		if len(query.Rows) != 0 {
			return true
		}
	}
	return false
}

func (q *querier) liteMetadata(ctx context.Context, request *qbtypes.QueryRangeRequest) (liteadapter.MetricMetadata, error) {
	return litemetadata.Resolve(ctx, q.metadataStore, request)
}

func liteQueryError(err error) error {
	var queryError *litequery.Error
	if errors.As(err, &queryError) {
		if queryError.Code == litequery.ErrorTimeout {
			return errors.NewTimeoutf(errors.CodeTimeout, "lightweight query execution timed out")
		}
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
