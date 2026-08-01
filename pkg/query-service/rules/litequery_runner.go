package rules

import (
	"context"
	stderrors "errors"
	"fmt"

	"github.com/SigNoz/signoz/pkg/errors"
	"github.com/SigNoz/signoz/pkg/litequery"
	"github.com/SigNoz/signoz/pkg/querier/liteadapter"
	"github.com/SigNoz/signoz/pkg/telemetrystore"
	"github.com/SigNoz/signoz/pkg/types/metrictypes"
	qbtypes "github.com/SigNoz/signoz/pkg/types/querybuildertypes/querybuildertypesv5"
	"github.com/SigNoz/signoz/pkg/types/telemetrytypes"
	"github.com/SigNoz/signoz/pkg/valuer"
)

// QueryRunner is the query boundary used by threshold rules. Keeping it local
// to rules prevents rule scheduling from depending on the legacy Querier.
type QueryRunner interface {
	Execute(context.Context, valuer.UUID, *qbtypes.QueryRangeRequest) (*qbtypes.QueryRangeResponse, error)
}

type liteQueryRunner struct {
	store    telemetrystore.TelemetryStore
	metadata telemetrytypes.MetadataStore
}

// NewLiteQueryRunner executes the restricted alert subset through Lite. Rule
// definitions retain their V5 persistence shape, but unsupported definitions
// receive a stable validation error rather than falling back to legacy SQL.
func NewLiteQueryRunner(store telemetrystore.TelemetryStore, metadata telemetrytypes.MetadataStore) QueryRunner {
	return &liteQueryRunner{store: store, metadata: metadata}
}

func (r *liteQueryRunner) Execute(ctx context.Context, _ valuer.UUID, request *qbtypes.QueryRangeRequest) (*qbtypes.QueryRangeResponse, error) {
	metadata, err := r.metricMetadata(ctx, request)
	if err != nil {
		return nil, err
	}
	liteRequest, err := liteadapter.ToLite(request, metadata)
	if err != nil {
		var unsupported *liteadapter.UnsupportedError
		if stderrors.As(err, &unsupported) {
			return nil, errors.NewInvalidInputf(errors.CodeInvalidInput, "unsupported threshold query: %s", unsupported.Feature)
		}
		return nil, errors.NewInvalidInputf(errors.CodeInvalidInput, "invalid threshold query: %v", err)
	}
	plan, err := (litequery.DefaultPlanner{}).Plan(liteRequest)
	if err != nil {
		return nil, errors.NewInvalidInputf(errors.CodeInvalidInput, "invalid threshold query: %v", err)
	}
	executor := litequery.Executor{
		Compiler: litequery.NewCompiler(nil),
		Query: func(ctx context.Context, query string, args ...any) (litequery.Rows, error) {
			rows, err := r.store.ClickhouseDB().Query(ctx, query, args...)
			if err != nil {
				return nil, err
			}
			return litequery.WrapClickHouseRows(rows), nil
		},
		Config: litequery.ExecutorConfig{MaxConcurrent: 4},
	}
	result, err := executor.Execute(ctx, plan)
	if err != nil {
		return nil, fmt.Errorf("execute threshold query: %w", err)
	}
	response, err := liteadapter.FromLite(request, result)
	if err != nil {
		return nil, fmt.Errorf("adapt threshold query result: %w", err)
	}
	return response, nil
}

func (r *liteQueryRunner) metricMetadata(ctx context.Context, request *qbtypes.QueryRangeRequest) (liteadapter.MetricMetadata, error) {
	if request == nil {
		return liteadapter.MetricMetadata{}, errors.NewInvalidInputf(errors.CodeInvalidInput, "threshold query is required")
	}
	names := make([]string, 0)
	for _, envelope := range request.CompositeQuery.Queries {
		query, ok := envelope.Spec.(qbtypes.QueryBuilderQuery[qbtypes.MetricAggregation])
		if !ok {
			continue
		}
		for _, aggregation := range query.Aggregations {
			if aggregation.MetricName != "" && (aggregation.Type == metrictypes.UnspecifiedType || aggregation.Temporality == metrictypes.Unknown) {
				names = append(names, aggregation.MetricName)
			}
		}
	}
	if len(names) == 0 {
		return liteadapter.MetricMetadata{}, nil
	}
	if r.metadata == nil {
		return liteadapter.MetricMetadata{}, errors.NewInternalf(errors.CodeInternal, "metric metadata store is unavailable")
	}
	temporalities, types, err := r.metadata.FetchTemporalityAndTypeMulti(ctx, request.Start, request.End, names...)
	if err != nil {
		return liteadapter.MetricMetadata{}, errors.NewInternalf(errors.CodeInternal, "failed to fetch metric temporality and type")
	}
	return liteadapter.MetricMetadata{Temporality: temporalities, Types: types}, nil
}
