// Package liteadapter translates the public V5 query DTO at the application
// boundary. The lightweight core deliberately does not import HTTP or V5
// request types.
package liteadapter

import (
	"fmt"
	"strings"

	"github.com/SigNoz/signoz/pkg/errors"
	"github.com/SigNoz/signoz/pkg/litequery"
	"github.com/SigNoz/signoz/pkg/types/metrictypes"
	qbtypes "github.com/SigNoz/signoz/pkg/types/querybuildertypes/querybuildertypesv5"
	"github.com/SigNoz/signoz/pkg/types/telemetrytypes"
)

// UnsupportedError says that a valid V5 request lies outside the deliberately
// small lightweight capability set. It is returned as a stable capability
// error at the HTTP boundary and must never be treated as a partial conversion.
type UnsupportedError struct{ Feature string }

func (e *UnsupportedError) Error() string {
	return "lightweight query does not support " + e.Feature
}

func unsupported(feature string) error { return &UnsupportedError{Feature: feature} }

// MetricMetadata contains the schema information that V5 resolves before
// adapting a request. It is intentionally data, not a store dependency, so
// this package remains deterministic in tests.
type MetricMetadata struct {
	Temporality map[string]metrictypes.Temporality
	Types       map[string]metrictypes.Type
	FieldKeys   map[string][]*telemetrytypes.TelemetryFieldKey
}

// ToLite converts the supported V5 subset to a storage-independent request.
// Unsupported features return UnsupportedError rather than being ignored.
func ToLite(request *qbtypes.QueryRangeRequest, metadata MetricMetadata) (litequery.Request, error) {
	if request == nil {
		return litequery.Request{}, errors.NewInvalidInputf(errors.CodeInvalidInput, "V5 request is required")
	}
	resultType, err := resultTypeFromV5(request.RequestType)
	if err != nil {
		return litequery.Request{}, err
	}
	if request.Start > uint64(^uint64(0)>>1) || request.End > uint64(^uint64(0)>>1) {
		return litequery.Request{}, errors.NewInvalidInputf(errors.CodeInvalidInput, "query range exceeds supported millisecond range")
	}
	result := litequery.Request{
		Range:      litequery.TimeRange{StartMS: int64(request.Start), EndMS: int64(request.End)},
		ResultType: resultType,
	}
	if request.FormatOptions != nil {
		if request.FormatOptions.FillGaps {
			return litequery.Request{}, unsupported("formatOptions.fillGaps")
		}
		result.Format.FillGaps = request.FormatOptions.FillGaps
	}

	var stepMS int64
	for _, envelope := range request.CompositeQuery.Queries {
		switch envelope.Type {
		case qbtypes.QueryTypeBuilder:
			query, step, disabled, err := builderToLite(envelope.Spec, resultType, metadata)
			if err != nil {
				return litequery.Request{}, err
			}
			if disabled {
				continue
			}
			if resultType == litequery.ResultTimeSeries {
				if step <= 0 {
					return litequery.Request{}, unsupported("builder query without stepInterval")
				}
				if stepMS == 0 {
					stepMS = step
				} else if stepMS != step {
					return litequery.Request{}, unsupported("mixed stepInterval values")
				}
			}
			result.Queries = append(result.Queries, query)
		case qbtypes.QueryTypeFormula:
			formula, disabled, err := formulaToLite(envelope.Spec)
			if err != nil {
				return litequery.Request{}, err
			}
			if !disabled {
				result.Formulas = append(result.Formulas, formula)
			}
		default:
			return litequery.Request{}, unsupported("V5 query type " + envelope.Type.StringValue())
		}
	}
	if len(result.Queries) == 0 {
		return litequery.Request{}, unsupported("request without enabled builder queries")
	}
	result.StepMS = stepMS
	return result, nil
}

func resultTypeFromV5(kind qbtypes.RequestType) (litequery.ResultType, error) {
	switch kind {
	case qbtypes.RequestTypeRaw:
		return litequery.ResultRaw, nil
	case qbtypes.RequestTypeTrace:
		return litequery.ResultTrace, nil
	case qbtypes.RequestTypeTimeSeries:
		return litequery.ResultTimeSeries, nil
	case qbtypes.RequestTypeScalar:
		return litequery.ResultScalar, nil
	default:
		return "", unsupported("requestType " + kind.StringValue())
	}
}

func builderToLite(spec any, resultType litequery.ResultType, metadata MetricMetadata) (litequery.Query, int64, bool, error) {
	switch query := spec.(type) {
	case qbtypes.QueryBuilderQuery[qbtypes.LogAggregation]:
		common, err := commonToLite(query.Name, query.Filter, query.SelectFields, query.GroupBy, query.Order, query.Limit, query.Offset, query.Cursor, query.LimitBy, query.Having, query.SecondaryAggregations, query.Functions, litequery.SignalLogs, metadata)
		if err != nil {
			return nil, 0, false, err
		}
		if len(query.Aggregations) != 1 && resultType != litequery.ResultRaw {
			return nil, 0, false, unsupported("multiple or missing log aggregations")
		}
		aggregation := litequery.LogAggregateCount
		var field litequery.FieldRef
		if len(query.Aggregations) == 1 {
			aggregation, field, err = logAggregation(query.Aggregations[0], metadata)
			if err != nil {
				return nil, 0, false, err
			}
		}
		return litequery.LogQuery{Common: common, Aggregation: aggregation, Field: field}, query.StepInterval.Milliseconds(), query.Disabled, nil
	case qbtypes.QueryBuilderQuery[qbtypes.TraceAggregation]:
		common, err := commonToLite(query.Name, query.Filter, query.SelectFields, query.GroupBy, query.Order, query.Limit, query.Offset, query.Cursor, query.LimitBy, query.Having, query.SecondaryAggregations, query.Functions, litequery.SignalTraces, metadata)
		if err != nil {
			return nil, 0, false, err
		}
		if len(query.Aggregations) != 1 && resultType != litequery.ResultRaw && resultType != litequery.ResultTrace {
			return nil, 0, false, unsupported("multiple or missing trace aggregations")
		}
		aggregation := litequery.TraceAggregateCount
		if len(query.Aggregations) == 1 {
			aggregation, err = traceAggregation(query.Aggregations[0])
			if err != nil {
				return nil, 0, false, err
			}
		}
		return litequery.TraceQuery{Common: common, Aggregation: aggregation}, query.StepInterval.Milliseconds(), query.Disabled, nil
	case qbtypes.QueryBuilderQuery[qbtypes.MetricAggregation]:
		common, err := commonToLite(query.Name, query.Filter, query.SelectFields, query.GroupBy, query.Order, query.Limit, query.Offset, query.Cursor, query.LimitBy, query.Having, query.SecondaryAggregations, query.Functions, litequery.SignalMetrics, metadata)
		if err != nil {
			return nil, 0, false, err
		}
		if len(query.Aggregations) != 1 {
			return nil, 0, false, unsupported("multiple or missing metric aggregations")
		}
		aggregation, err := metricAggregation(query.Aggregations[0], metadata)
		if err != nil {
			return nil, 0, false, err
		}
		if query.Source == telemetrytypes.SourceMeter {
			return litequery.MeterQuery{Common: common, Aggregation: aggregation}, query.StepInterval.Milliseconds(), query.Disabled, nil
		}
		return litequery.MetricQuery{Common: common, Aggregation: aggregation}, query.StepInterval.Milliseconds(), query.Disabled, nil
	default:
		return nil, 0, false, unsupported(fmt.Sprintf("builder spec %T", spec))
	}
}

func commonToLite(name string, filter *qbtypes.Filter, selectFields []telemetrytypes.TelemetryFieldKey, groupBy []qbtypes.GroupByKey, order []qbtypes.OrderBy, limit, offset int, cursor string, limitBy *qbtypes.LimitBy, having *qbtypes.Having, secondary []qbtypes.SecondaryAggregation, functions []qbtypes.Function, signal litequery.Signal, metadata MetricMetadata) (litequery.CommonQuery, error) {
	if limit < 0 || offset < 0 {
		return litequery.CommonQuery{}, errors.NewInvalidInputf(errors.CodeInvalidInput, "query limit and offset must not be negative")
	}
	if limitBy != nil || len(secondary) != 0 || (having != nil && strings.TrimSpace(having.Expression) != "") {
		return litequery.CommonQuery{}, unsupported("limitBy, secondary aggregation, or arbitrary having")
	}
	if len(functions) != 0 {
		return litequery.CommonQuery{}, unsupported("post-processing functions")
	}
	if cursor != "" {
		return litequery.CommonQuery{}, unsupported("cursor pagination")
	}
	common := litequery.CommonQuery{Name: name, Limit: uint32(limit), Offset: uint32(offset), Cursor: cursor}
	for _, field := range selectFields {
		converted, err := fieldToLite(field, signal, litequery.ValueTypeString, metadata)
		if err != nil {
			return litequery.CommonQuery{}, err
		}
		common.Select = append(common.Select, converted)
	}
	for _, field := range groupBy {
		converted, err := fieldToLite(field.TelemetryFieldKey, signal, litequery.ValueTypeString, metadata)
		if err != nil {
			return litequery.CommonQuery{}, err
		}
		common.GroupBy = append(common.GroupBy, converted)
	}
	for _, item := range order {
		converted, err := fieldToLite(item.Key.TelemetryFieldKey, signal, litequery.ValueTypeString, metadata)
		if err != nil {
			return litequery.CommonQuery{}, err
		}
		target := litequery.OrderByField
		if item.Key.Name == "" || item.Key.Name == "value" || strings.HasPrefix(item.Key.Name, "__result_") {
			target = litequery.OrderByAggregation
		}
		direction := litequery.SortAscending
		if item.Direction == qbtypes.OrderDirectionDesc {
			direction = litequery.SortDescending
		}
		common.Order = append(common.Order, litequery.Order{Target: target, Field: converted, Direction: direction})
	}
	if filter != nil && strings.TrimSpace(filter.Expression) != "" {
		parsed, err := parseFilter(filter.Expression, signal, metadata)
		if err != nil {
			return litequery.CommonQuery{}, err
		}
		common.Filter = parsed
	}
	return common, nil
}

func logAggregation(aggregation qbtypes.LogAggregation, metadata MetricMetadata) (litequery.LogAggregation, litequery.FieldRef, error) {
	expression := strings.ToLower(strings.TrimSpace(aggregation.Expression))
	if expression == "count()" {
		return litequery.LogAggregateCount, litequery.FieldRef{}, nil
	}
	for _, candidate := range []struct {
		prefix string
		kind   litequery.LogAggregation
	}{
		{"sum(", litequery.LogAggregateSum}, {"avg(", litequery.LogAggregateAvg}, {"min(", litequery.LogAggregateMin}, {"max(", litequery.LogAggregateMax},
	} {
		if strings.HasPrefix(expression, candidate.prefix) && strings.HasSuffix(expression, ")") {
			name := strings.TrimSpace(aggregation.Expression[len(candidate.prefix) : len(aggregation.Expression)-1])
			field, err := textFieldToLite(name, litequery.SignalLogs, litequery.ValueTypeNumber, metadata)
			return candidate.kind, field, err
		}
	}
	return "", litequery.FieldRef{}, unsupported("log aggregation " + aggregation.Expression)
}

func traceAggregation(aggregation qbtypes.TraceAggregation) (litequery.TraceAggregation, error) {
	switch strings.ToLower(strings.ReplaceAll(strings.TrimSpace(aggregation.Expression), " ", "")) {
	case "count()":
		return litequery.TraceAggregateCount, nil
	case "avg(duration_nano)":
		return litequery.TraceAggregateDurationAvg, nil
	case "p50(duration_nano)", "quantile(0.5)(duration_nano)":
		return litequery.TraceAggregateDurationP50, nil
	case "p90(duration_nano)", "quantile(0.9)(duration_nano)":
		return litequery.TraceAggregateDurationP90, nil
	case "p95(duration_nano)", "quantile(0.95)(duration_nano)":
		return litequery.TraceAggregateDurationP95, nil
	case "p99(duration_nano)", "quantile(0.99)(duration_nano)":
		return litequery.TraceAggregateDurationP99, nil
	default:
		return "", unsupported("trace aggregation " + aggregation.Expression)
	}
}

func metricAggregation(aggregation qbtypes.MetricAggregation, metadata MetricMetadata) (litequery.MetricAggregation, error) {
	metricType := aggregation.Type
	temporality := aggregation.Temporality
	if metricType == metrictypes.UnspecifiedType && metadata.Types != nil {
		metricType = metadata.Types[aggregation.MetricName]
	}
	if temporality == metrictypes.Unknown && metadata.Temporality != nil {
		temporality = metadata.Temporality[aggregation.MetricName]
	}
	typeValue := litequery.MetricType(strings.ToLower(metricType.StringValue()))
	if typeValue != litequery.MetricGauge && typeValue != litequery.MetricSum && typeValue != litequery.MetricHistogram {
		return litequery.MetricAggregation{}, unsupported("metric type for " + aggregation.MetricName)
	}
	temporalityValue := litequery.Temporality(strings.ToLower(temporality.StringValue()))
	if temporalityValue == "" {
		temporalityValue = litequery.TemporalityUnspecified
	}
	// Metadata intentionally exposes non-monotonic OTLP sums as gauges. Once
	// normalized to a gauge, the original sum temporality is no longer part of
	// the query contract.
	if typeValue == litequery.MetricGauge {
		temporalityValue = litequery.TemporalityUnspecified
	}
	// Histogram points carry their aggregation semantics in the bucket value.
	// V5 metadata can still report the instrument temporality, but the
	// lightweight query contract deliberately does not accept it for histograms.
	if typeValue == litequery.MetricHistogram {
		temporalityValue = litequery.TemporalityUnspecified
	}
	timeAggregation := litequery.TimeAggregation(aggregation.TimeAggregation.StringValue())
	spaceAggregation := litequery.SpaceAggregation(aggregation.SpaceAggregation.StringValue())
	// The V5 editor historically serializes a histogram percentile as the
	// timeAggregation (for example p95) while the lightweight IR models the
	// percentile at the histogram reduction phase. Normalize that wire-level
	// spelling at the boundary so the core remains small and unambiguous.
	if typeValue == litequery.MetricHistogram {
		switch timeAggregation {
		case litequery.TimeAggregation("p50"), litequery.TimeAggregation("p90"), litequery.TimeAggregation("p95"), litequery.TimeAggregation("p99"):
			spaceAggregation = litequery.SpaceAggregation(timeAggregation)
			timeAggregation = litequery.TimeAggregateCount
		}
	}
	return litequery.MetricAggregation{
		MetricName: aggregation.MetricName, Type: typeValue, Temporality: temporalityValue,
		TimeAggregation:  timeAggregation,
		SpaceAggregation: spaceAggregation,
	}, nil
}

func formulaToLite(spec any) (litequery.Formula, bool, error) {
	formula, ok := spec.(qbtypes.QueryBuilderFormula)
	if !ok {
		return litequery.Formula{}, false, unsupported(fmt.Sprintf("formula spec %T", spec))
	}
	if len(formula.Order) != 0 || formula.Limit != 0 || (formula.Having != nil && strings.TrimSpace(formula.Having.Expression) != "") || len(formula.Functions) != 0 {
		return litequery.Formula{}, false, unsupported("formula ordering, limit, having, or functions")
	}
	return litequery.Formula{Name: formula.Name, Expression: formula.Expression}, formula.Disabled, nil
}
