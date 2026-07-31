package litequery

import (
	"math"
	"strings"
)

type Limits struct {
	MaxQueries       int
	MaxFormulas      int
	MaxFilterDepth   int
	MaxFilterNodes   int
	MaxGroupByFields int
	MaxRawLimit      uint32
	MaxSeriesPoints  int64
}

func DefaultLimits() Limits {
	return Limits{
		MaxQueries:       8,
		MaxFormulas:      4,
		MaxFilterDepth:   8,
		MaxFilterNodes:   64,
		MaxGroupByFields: 4,
		MaxRawLimit:      1000,
		MaxSeriesPoints:  11000,
	}
}

func Validate(req Request, limits Limits) error {
	if err := validateLimits(limits); err != nil {
		return err
	}
	if req.Range.StartMS < 0 || req.Range.EndMS <= req.Range.StartMS {
		return newError(ErrorInvalidRequest, "range", "end must be after start")
	}
	if !req.ResultType.valid() {
		return newError(ErrorInvalidRequest, "resultType", "unsupported result type %q", req.ResultType)
	}
	if len(req.Queries) == 0 {
		return newError(ErrorInvalidRequest, "queries", "at least one query is required")
	}
	if len(req.Queries) > limits.MaxQueries {
		return newError(ErrorBudgetExceeded, "queries", "request contains more than %d queries", limits.MaxQueries)
	}
	if len(req.Formulas) > limits.MaxFormulas {
		return newError(ErrorBudgetExceeded, "formulas", "request contains more than %d formulas", limits.MaxFormulas)
	}
	if req.ResultType == ResultTimeSeries {
		if req.StepMS <= 0 {
			return newError(ErrorInvalidRequest, "stepMs", "time series requests require a positive step")
		}
		points := (req.Range.EndMS - req.Range.StartMS) / req.StepMS
		if points > limits.MaxSeriesPoints {
			return newError(ErrorBudgetExceeded, "stepMs", "time series contains more than %d points", limits.MaxSeriesPoints)
		}
	} else if req.StepMS != 0 {
		return newError(ErrorInvalidRequest, "stepMs", "step is only valid for time series requests")
	}

	names := make(map[string]struct{}, len(req.Queries)+len(req.Formulas))
	for _, query := range req.Queries {
		if query == nil {
			return newError(ErrorInvalidRequest, "queries", "query must not be nil")
		}
		common := query.GetCommon()
		if err := validateCommon(common, req.ResultType, limits); err != nil {
			return err
		}
		if err := addName(names, common.Name, "queries"); err != nil {
			return err
		}
		if err := validateQuery(query); err != nil {
			return err
		}
		if err := validateResultType(query, req.ResultType); err != nil {
			return err
		}
	}
	if err := validateFormulas(req.Formulas, names); err != nil {
		return err
	}
	return nil
}

func validateLimits(limits Limits) error {
	if limits.MaxQueries < 1 || limits.MaxFormulas < 0 || limits.MaxFilterDepth < 1 ||
		limits.MaxFilterNodes < 1 || limits.MaxGroupByFields < 0 || limits.MaxRawLimit < 1 || limits.MaxSeriesPoints < 1 {
		return newError(ErrorInvalidRequest, "limits", "all limits must be positive except max formulas and max group by fields")
	}
	return nil
}

func validateCommon(common CommonQuery, resultType ResultType, limits Limits) error {
	if !validName(common.Name) {
		return newError(ErrorInvalidRequest, "query.name", "query name must start with a letter and contain only letters, digits, or underscores")
	}
	if len(common.GroupBy) > limits.MaxGroupByFields {
		return newError(ErrorBudgetExceeded, "query.groupBy", "query contains more than %d group-by fields", limits.MaxGroupByFields)
	}
	for _, field := range common.Select {
		if err := validateField(field, "query.select"); err != nil {
			return err
		}
	}
	for _, field := range common.GroupBy {
		if err := validateField(field, "query.groupBy"); err != nil {
			return err
		}
	}
	for _, order := range common.Order {
		if order.Direction != SortAscending && order.Direction != SortDescending {
			return newError(ErrorInvalidRequest, "query.order.direction", "unsupported sort direction %q", order.Direction)
		}
		switch order.Target {
		case OrderByField:
			if err := validateField(order.Field, "query.order.field"); err != nil {
				return err
			}
		case OrderByAggregation:
			if !resultType.isAggregate() {
				return newError(ErrorInvalidRequest, "query.order", "aggregation order requires an aggregate result")
			}
		default:
			return newError(ErrorInvalidRequest, "query.order.target", "unsupported order target %q", order.Target)
		}
	}
	if common.Limit > limits.MaxRawLimit {
		return newError(ErrorBudgetExceeded, "query.limit", "limit exceeds %d", limits.MaxRawLimit)
	}
	if common.Cursor != "" && resultType != ResultRaw && resultType != ResultTrace {
		return newError(ErrorInvalidRequest, "query.cursor", "cursor is only valid for raw and trace results")
	}
	if common.Predicate != nil {
		if !resultType.isAggregate() {
			return newError(ErrorInvalidRequest, "query.predicate", "aggregation predicate requires an aggregate result")
		}
		if !common.Predicate.Operator.valid() {
			return newError(ErrorInvalidRequest, "query.predicate.operator", "unsupported aggregation predicate operator %q", common.Predicate.Operator)
		}
		if !finite(common.Predicate.Value) {
			return newError(ErrorInvalidRequest, "query.predicate.value", "aggregation predicate value must be finite")
		}
	}
	depth, nodes := 0, 0
	return validateFilter(common.Filter, &depth, &nodes, limits)
}

func validateField(field FieldRef, fieldName string) error {
	if strings.TrimSpace(field.Name) == "" {
		return newError(ErrorInvalidRequest, fieldName+".name", "field name is required")
	}
	if !field.Context.valid() {
		return newError(ErrorInvalidRequest, fieldName+".context", "unsupported field context %q", field.Context)
	}
	if !field.Type.valid() {
		return newError(ErrorInvalidRequest, fieldName+".type", "unsupported field type %q", field.Type)
	}
	return nil
}

func validateQuery(query Query) error {
	switch current := query.(type) {
	case LogQuery:
		return validateLogQuery(current)
	case TraceQuery:
		return validateTraceQuery(current)
	case MetricQuery:
		return validateMetricQuery(current.Aggregation, false)
	case MeterQuery:
		return validateMetricQuery(current.Aggregation, true)
	default:
		return newError(ErrorUnsupported, "query", "unsupported query implementation %T", query)
	}
}

func validateLogQuery(query LogQuery) error {
	switch query.Aggregation {
	case LogAggregateCount:
		return nil
	case LogAggregateSum, LogAggregateAvg, LogAggregateMin, LogAggregateMax:
		return validateField(query.Field, "query.field")
	default:
		return newError(ErrorInvalidAggregation, "query.aggregation", "unsupported log aggregation %q", query.Aggregation)
	}
}

func validateTraceQuery(query TraceQuery) error {
	switch query.Aggregation {
	case TraceAggregateCount, TraceAggregateDurationAvg, TraceAggregateDurationP50,
		TraceAggregateDurationP90, TraceAggregateDurationP95, TraceAggregateDurationP99:
		return nil
	default:
		return newError(ErrorInvalidAggregation, "query.aggregation", "unsupported trace aggregation %q", query.Aggregation)
	}
}

func validateMetricQuery(aggregation MetricAggregation, meter bool) error {
	if strings.TrimSpace(aggregation.MetricName) == "" {
		return newError(ErrorInvalidAggregation, "query.metricName", "metric name is required")
	}
	if aggregation.Temporality != TemporalityUnspecified && aggregation.Temporality != TemporalityDelta && aggregation.Temporality != TemporalityCumulative {
		return newError(ErrorInvalidAggregation, "query.temporality", "unsupported temporality %q", aggregation.Temporality)
	}
	if meter && aggregation.Type != MetricSum {
		return newError(ErrorInvalidAggregation, "query.type", "meter only supports sum metrics")
	}
	switch aggregation.Type {
	case MetricGauge:
		if aggregation.Temporality != TemporalityUnspecified {
			return newError(ErrorInvalidAggregation, "query.temporality", "gauge metrics do not use temporality")
		}
		if aggregation.TimeAggregation != TimeAggregateLatest && aggregation.TimeAggregation != TimeAggregateAvg && aggregation.TimeAggregation != TimeAggregateMin && aggregation.TimeAggregation != TimeAggregateMax {
			return newError(ErrorInvalidAggregation, "query.timeAggregation", "unsupported gauge time aggregation %q", aggregation.TimeAggregation)
		}
		if !isBasicSpaceAggregation(aggregation.SpaceAggregation) {
			return newError(ErrorInvalidAggregation, "query.spaceAggregation", "unsupported gauge space aggregation %q", aggregation.SpaceAggregation)
		}
	case MetricSum:
		if aggregation.TimeAggregation != TimeAggregateSum && aggregation.TimeAggregation != TimeAggregateRate && aggregation.TimeAggregation != TimeAggregateIncrease && aggregation.TimeAggregation != TimeAggregateCount && aggregation.TimeAggregation != TimeAggregateAvg {
			return newError(ErrorInvalidAggregation, "query.timeAggregation", "unsupported sum time aggregation %q", aggregation.TimeAggregation)
		}
		if !isBasicSpaceAggregation(aggregation.SpaceAggregation) {
			return newError(ErrorInvalidAggregation, "query.spaceAggregation", "unsupported sum space aggregation %q", aggregation.SpaceAggregation)
		}
	case MetricHistogram:
		if aggregation.Temporality != TemporalityUnspecified {
			return newError(ErrorInvalidAggregation, "query.temporality", "histogram metrics do not use temporality")
		}
		if aggregation.TimeAggregation != TimeAggregateCount && aggregation.TimeAggregation != TimeAggregateSum && aggregation.TimeAggregation != TimeAggregateAvg {
			return newError(ErrorInvalidAggregation, "query.timeAggregation", "unsupported histogram time aggregation %q", aggregation.TimeAggregation)
		}
		if !isHistogramSpaceAggregation(aggregation.SpaceAggregation) {
			return newError(ErrorInvalidAggregation, "query.spaceAggregation", "unsupported histogram space aggregation %q", aggregation.SpaceAggregation)
		}
	default:
		return newError(ErrorUnsupported, "query.type", "unsupported metric type %q", aggregation.Type)
	}
	return nil
}

func isBasicSpaceAggregation(aggregation SpaceAggregation) bool {
	return aggregation == SpaceAggregateSum || aggregation == SpaceAggregateAvg || aggregation == SpaceAggregateMin || aggregation == SpaceAggregateMax || aggregation == SpaceAggregateCount
}

func isHistogramSpaceAggregation(aggregation SpaceAggregation) bool {
	return isBasicSpaceAggregation(aggregation) || aggregation == SpaceAggregateP50 || aggregation == SpaceAggregateP90 || aggregation == SpaceAggregateP95 || aggregation == SpaceAggregateP99
}

func validateResultType(query Query, resultType ResultType) error {
	switch query.QuerySignal() {
	case SignalLogs:
		if resultType == ResultTrace {
			return newError(ErrorInvalidRequest, "resultType", "trace result is not valid for logs")
		}
	case SignalTraces:
		if resultType != ResultRaw && resultType != ResultTrace && resultType != ResultTimeSeries && resultType != ResultScalar {
			return newError(ErrorInvalidRequest, "resultType", "unsupported trace result type")
		}
	case SignalMetrics, SignalMeter:
		if resultType != ResultTimeSeries && resultType != ResultScalar {
			return newError(ErrorInvalidRequest, "resultType", "metrics and meter require time series or scalar results")
		}
	default:
		return newError(ErrorUnsupported, "query.signal", "unsupported signal %q", query.QuerySignal())
	}
	return nil
}

func addName(names map[string]struct{}, name, field string) error {
	if _, exists := names[name]; exists {
		return newError(ErrorInvalidRequest, field, "duplicate query or formula name %q", name)
	}
	names[name] = struct{}{}
	return nil
}

func validName(name string) bool {
	if name == "" {
		return false
	}
	for index, r := range name {
		isLetter := (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z')
		if index == 0 && !isLetter {
			return false
		}
		if index > 0 && !isLetter && (r < '0' || r > '9') && r != '_' {
			return false
		}
	}
	return true
}

func (r ResultType) isAggregate() bool {
	return r == ResultTimeSeries || r == ResultScalar
}

func finite(value float64) bool {
	return !math.IsInf(value, 0) && !math.IsNaN(value)
}
