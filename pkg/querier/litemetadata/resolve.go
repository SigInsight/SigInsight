// Package litemetadata owns the shared metadata lookup performed before a V5
// request enters the deterministic lightweight adapter.
package litemetadata

import (
	"context"

	"github.com/SigNoz/signoz/pkg/errors"
	"github.com/SigNoz/signoz/pkg/querier/liteadapter"
	"github.com/SigNoz/signoz/pkg/types/metrictypes"
	qbtypes "github.com/SigNoz/signoz/pkg/types/querybuildertypes/querybuildertypesv5"
	"github.com/SigNoz/signoz/pkg/types/telemetrytypes"
)

// Resolve batches metric and telemetry-field metadata for both interactive
// queries and threshold rules. Keeping this bridge shared prevents the two V5
// entry points from developing different field-resolution semantics.
func Resolve(ctx context.Context, store telemetrytypes.MetadataStore, request *qbtypes.QueryRangeRequest) (liteadapter.MetricMetadata, error) {
	if err := liteadapter.ValidateRequestRange(request); err != nil {
		return liteadapter.MetricMetadata{}, err
	}

	names := unresolvedMetricNames(request)
	fieldSelectors := liteadapter.FieldKeySelectors(request)
	if len(names) == 0 && len(fieldSelectors) == 0 {
		return liteadapter.MetricMetadata{}, nil
	}
	if store == nil {
		return liteadapter.MetricMetadata{}, errors.NewInternalf(errors.CodeInternal, "telemetry metadata store is unavailable")
	}

	metadata := liteadapter.MetricMetadata{}
	if len(names) != 0 {
		temporalities, types, err := store.FetchTemporalityAndTypeMulti(ctx, request.Start, request.End, names...)
		if err != nil {
			return liteadapter.MetricMetadata{}, errors.WrapInternalf(err, errors.CodeInternal, "failed to fetch metric temporality and type")
		}
		metadata.Temporality = temporalities
		metadata.Types = types
	}
	if len(fieldSelectors) != 0 {
		fieldKeys, _, err := store.GetKeysMulti(ctx, fieldSelectors)
		if err != nil {
			return liteadapter.MetricMetadata{}, errors.WrapInternalf(err, errors.CodeInternal, "failed to resolve telemetry fields")
		}
		metadata.FieldKeys = fieldKeys
	}
	return metadata, nil
}

func unresolvedMetricNames(request *qbtypes.QueryRangeRequest) []string {
	names := make([]string, 0)
	seen := make(map[string]struct{})
	for _, envelope := range request.CompositeQuery.Queries {
		query, ok := envelope.Spec.(qbtypes.QueryBuilderQuery[qbtypes.MetricAggregation])
		if !ok || query.Disabled {
			continue
		}
		for _, aggregation := range query.Aggregations {
			needsType := aggregation.Type == metrictypes.UnspecifiedType
			needsTemporality := (aggregation.Type == metrictypes.SumType || aggregation.Type == metrictypes.HistogramType) && aggregation.Temporality == metrictypes.Unknown
			if aggregation.MetricName == "" || (!needsType && !needsTemporality) {
				continue
			}
			if _, ok := seen[aggregation.MetricName]; ok {
				continue
			}
			seen[aggregation.MetricName] = struct{}{}
			names = append(names, aggregation.MetricName)
		}
	}
	return names
}
