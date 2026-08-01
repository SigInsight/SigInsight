package litequery

import (
	"fmt"
	"strings"
)

type Compiler struct {
	Catalog Catalog
}

func NewCompiler(catalog Catalog) Compiler {
	if catalog == nil {
		catalog = DefaultCatalog{}
	}
	return Compiler{Catalog: catalog}
}

func (c Compiler) Compile(plan Plan) ([]Statement, error) {
	statements := make([]Statement, 0, len(plan.Queries))
	for _, queryPlan := range plan.Queries {
		var (
			statement Statement
			err       error
		)
		switch query := queryPlan.Query.(type) {
		case LogQuery:
			statement, err = c.compileLogs(plan, query)
		case TraceQuery:
			statement, err = c.compileTraces(plan, query)
		case MetricQuery:
			statement, err = c.compileMetric(plan, query.Aggregation, query.Common, SignalMetrics)
		case MeterQuery:
			statement, err = c.compileMetric(plan, query.Aggregation, query.Common, SignalMeter)
		default:
			err = newError(ErrorUnsupported, "query", "compiler does not support %T", queryPlan.Query)
		}
		if err != nil {
			return nil, err
		}
		statements = append(statements, statement)
	}
	return statements, nil
}

func (c Compiler) compileLogs(plan Plan, query LogQuery) (Statement, error) {
	table, err := c.Catalog.Table(SignalLogs)
	if err != nil {
		return Statement{}, err
	}
	common := query.GetCommon()
	if plan.ResultType == ResultRaw {
		return c.compileRaw(table, SignalLogs, plan, common)
	}
	aggregation, err := c.logAggregation(query)
	if err != nil {
		return Statement{}, err
	}
	return c.compileAggregate(table, SignalLogs, plan, common, aggregation)
}

func (c Compiler) compileTraces(plan Plan, query TraceQuery) (Statement, error) {
	table, err := c.Catalog.Table(SignalTraces)
	if err != nil {
		return Statement{}, err
	}
	common := query.GetCommon()
	switch plan.ResultType {
	case ResultRaw:
		return c.compileRaw(table, SignalTraces, plan, common)
	case ResultTrace:
		return c.compileTraceSummary(table, plan, common)
	default:
		aggregation, err := c.traceAggregation(query)
		if err != nil {
			return Statement{}, err
		}
		return c.compileAggregate(table, SignalTraces, plan, common, aggregation)
	}
}

type compiledExpression struct {
	SQL  string
	Args []any
}

func (c Compiler) compileRaw(table string, signal Signal, plan Plan, common CommonQuery) (Statement, error) {
	if common.Cursor != "" {
		return Statement{}, newError(ErrorUnsupported, "query.cursor", "cursor compilation is not available until result scanning is implemented")
	}
	if common.After != nil && signal != SignalLogs {
		return Statement{}, newError(ErrorUnsupported, "query.after", "typed raw cursors are only supported for logs")
	}
	fields := common.Select
	if len(fields) == 0 {
		fields = defaultFields(signal)
	}
	selects := make([]string, 0, len(fields))
	columns := make([]ResultColumn, 0, len(fields))
	args := make([]any, 0)
	for _, field := range fields {
		resolved, err := c.Catalog.Resolve(signal, field)
		if err != nil {
			return Statement{}, err
		}
		alias := fmt.Sprintf("field_%d", len(columns))
		selects = append(selects, resolved.SQL+" AS "+alias)
		fieldCopy := field
		columns = append(columns, ResultColumn{Name: alias, Field: &fieldCopy})
		args = append(args, resolved.Args...)
	}
	where, whereArgs, err := c.compileWhere(signal, common.Filter, plan.Range)
	if err != nil {
		return Statement{}, err
	}
	args = append(args, whereArgs...)
	if common.After != nil {
		where = "(" + where + ") AND ((timestamp > toUInt64(?)) OR (timestamp = toUInt64(?) AND id > ?))"
		args = append(args, common.After.TimestampNS, common.After.TimestampNS, common.After.ID)
	}
	defaultOrder := defaultRawOrder(signal)
	if common.After != nil {
		defaultOrder = "timestamp ASC, id ASC"
	}
	order, orderArgs, err := c.compileOrder(signal, common.Order, false, defaultOrder)
	if err != nil {
		return Statement{}, err
	}
	args = append(args, orderArgs...)
	limit := common.Limit
	if limit == 0 {
		limit = 100
	}
	args = append(args, limit)
	suffix := " LIMIT ?"
	if common.Offset != 0 {
		suffix += " OFFSET ?"
		args = append(args, common.Offset)
	}
	return Statement{
		Name:    common.Name,
		SQL:     "SELECT " + strings.Join(selects, ", ") + " FROM " + table + " WHERE " + where + " ORDER BY " + order + suffix,
		Args:    args,
		Columns: columns,
	}, nil
}

func (c Compiler) compileTraceSummary(table string, plan Plan, common CommonQuery) (Statement, error) {
	where, args, err := c.compileWhere(SignalTraces, common.Filter, plan.Range)
	if err != nil {
		return Statement{}, err
	}
	order := "timestamp DESC, trace_id DESC"
	if len(common.Order) != 0 {
		compiledOrder, orderArgs, err := c.compileOrder(SignalTraces, common.Order, false, order)
		if err != nil {
			return Statement{}, err
		}
		order = compiledOrder
		args = append(args, orderArgs...)
	}
	limit := common.Limit
	if limit == 0 {
		limit = 100
	}
	args = append(args, limit)
	suffix := " LIMIT ?"
	if common.Offset != 0 {
		suffix += " OFFSET ?"
		args = append(args, common.Offset)
	}
	return Statement{
		Name: common.Name,
		SQL: "SELECT trace_id, max(timestamp) AS timestamp, count() AS span_count, sum(duration_nano) AS duration_nano " +
			"FROM " + table + " WHERE " + where + " GROUP BY trace_id ORDER BY " + order + suffix,
		Args: args,
		Columns: []ResultColumn{
			{Name: "trace_id"},
			{Name: "timestamp"},
			{Name: "span_count"},
			{Name: "duration_nano"},
		},
	}, nil
}

func (c Compiler) compileAggregate(table string, signal Signal, plan Plan, common CommonQuery, aggregation compiledExpression) (Statement, error) {
	selects := make([]string, 0, len(common.GroupBy)+2)
	columns := make([]ResultColumn, 0, len(common.GroupBy)+2)
	selectArgs := make([]any, 0, len(aggregation.Args)+8)
	groupArgs := make([]any, 0, len(aggregation.Args)+8)
	groupBy := make([]string, 0, len(common.GroupBy)+1)
	if plan.ResultType == ResultTimeSeries {
		bucket, bucketArgs, err := timeBucket(signal, plan.StepMS)
		if err != nil {
			return Statement{}, err
		}
		selects = append(selects, bucket+" AS timestamp")
		columns = append(columns, ResultColumn{Name: "timestamp"})
		selectArgs = append(selectArgs, bucketArgs...)
		groupBy = append(groupBy, bucket)
		groupArgs = append(groupArgs, bucketArgs...)
	}
	for index, field := range common.GroupBy {
		resolved, err := c.Catalog.Resolve(signal, field)
		if err != nil {
			return Statement{}, err
		}
		alias := fmt.Sprintf("group_%d", index)
		selects = append(selects, resolved.SQL+" AS "+alias)
		fieldCopy := field
		columns = append(columns, ResultColumn{Name: alias, Field: &fieldCopy})
		selectArgs = append(selectArgs, resolved.Args...)
		groupBy = append(groupBy, resolved.SQL)
		groupArgs = append(groupArgs, resolved.Args...)
	}
	selects = append(selects, aggregation.SQL+" AS value")
	columns = append(columns, ResultColumn{Name: "value"})
	selectArgs = append(selectArgs, aggregation.Args...)

	where, whereArgs, err := c.compileWhere(signal, common.Filter, plan.Range)
	if err != nil {
		return Statement{}, err
	}
	args := append(selectArgs, whereArgs...)

	query := "SELECT " + strings.Join(selects, ", ") + " FROM " + table + " WHERE " + where
	if len(groupBy) != 0 {
		query += " GROUP BY " + strings.Join(groupBy, ", ")
		args = append(args, groupArgs...)
	}
	if common.Predicate != nil {
		query += " HAVING " + aggregation.SQL + " " + comparisonSQL(common.Predicate.Operator) + " ?"
		args = append(args, aggregation.Args...)
		args = append(args, common.Predicate.Value)
	}
	defaultOrder := "value DESC"
	if plan.ResultType == ResultTimeSeries {
		defaultOrder = "timestamp ASC"
	}
	order, orderArgs, err := c.compileOrder(signal, common.Order, true, defaultOrder)
	if err != nil {
		return Statement{}, err
	}
	query += " ORDER BY " + order
	args = append(args, orderArgs...)
	if common.Limit != 0 {
		query += " LIMIT ?"
		args = append(args, common.Limit)
	}
	return Statement{Name: common.Name, SQL: query, Args: args, Columns: columns}, nil
}

func (c Compiler) logAggregation(query LogQuery) (compiledExpression, error) {
	switch query.Aggregation {
	case LogAggregateCount:
		return compiledExpression{SQL: "count()"}, nil
	case LogAggregateSum, LogAggregateAvg, LogAggregateMin, LogAggregateMax:
		field, err := c.Catalog.Resolve(SignalLogs, query.Field)
		if err != nil {
			return compiledExpression{}, err
		}
		return compiledExpression{SQL: string(query.Aggregation) + "(" + field.SQL + ")", Args: field.Args}, nil
	default:
		return compiledExpression{}, newError(ErrorInvalidAggregation, "query.aggregation", "unsupported log aggregation %q", query.Aggregation)
	}
}

func (c Compiler) traceAggregation(query TraceQuery) (compiledExpression, error) {
	switch query.Aggregation {
	case TraceAggregateCount:
		return compiledExpression{SQL: "count()"}, nil
	case TraceAggregateDurationAvg:
		return compiledExpression{SQL: "avg(duration_nano)"}, nil
	case TraceAggregateDurationP50:
		return compiledExpression{SQL: "quantile(0.5)(duration_nano)"}, nil
	case TraceAggregateDurationP90:
		return compiledExpression{SQL: "quantile(0.9)(duration_nano)"}, nil
	case TraceAggregateDurationP95:
		return compiledExpression{SQL: "quantile(0.95)(duration_nano)"}, nil
	case TraceAggregateDurationP99:
		return compiledExpression{SQL: "quantile(0.99)(duration_nano)"}, nil
	default:
		return compiledExpression{}, newError(ErrorInvalidAggregation, "query.aggregation", "unsupported trace aggregation %q", query.Aggregation)
	}
}

func (c Compiler) compileWhere(signal Signal, filter FilterNode, timeRange TimeRange) (string, []any, error) {
	timeCondition, args, err := timeRangeCondition(signal, timeRange)
	if err != nil {
		return "", nil, err
	}
	if filter == nil {
		return timeCondition, args, nil
	}
	condition, filterArgs, err := c.compileFilter(signal, filter)
	if err != nil {
		return "", nil, err
	}
	return "(" + timeCondition + ") AND (" + condition + ")", append(args, filterArgs...), nil
}

func (c Compiler) compileFilter(signal Signal, node FilterNode) (string, []any, error) {
	switch current := node.(type) {
	case LogicalFilter:
		parts := make([]string, 0, len(current.Items))
		args := make([]any, 0)
		separator := " AND "
		if current.Operator == BooleanOr {
			separator = " OR "
		}
		for _, item := range current.Items {
			part, partArgs, err := c.compileFilter(signal, item)
			if err != nil {
				return "", nil, err
			}
			parts = append(parts, "("+part+")")
			args = append(args, partArgs...)
		}
		return strings.Join(parts, separator), args, nil
	case Predicate:
		field, err := c.Catalog.Resolve(signal, current.Field)
		if err != nil {
			return "", nil, err
		}
		return compilePredicate(field, current)
	default:
		return "", nil, newError(ErrorInvalidFilter, "filter", "unsupported filter node %T", node)
	}
}

func compilePredicate(field ResolvedField, predicate Predicate) (string, []any, error) {
	switch predicate.Op {
	case FilterEqual, FilterNotEqual, FilterGreaterThan, FilterGreaterEq, FilterLessThan, FilterLessEq:
		valueSQL := field.ComparisonValueSQL
		if valueSQL == "" {
			valueSQL = "?"
		}
		condition := field.SQL + " " + filterSQL(predicate.Op) + " " + valueSQL
		args := append(append([]any{}, field.Args...), scalarValue(predicate.Value))
		if field.RequiresExistence && predicate.Op != FilterNotEqual {
			return "(" + field.ExistsSQL + ") AND (" + condition + ")", append(append([]any{}, field.ExistsArgs...), args...), nil
		}
		return condition, args, nil
	case FilterIn, FilterNotIn:
		placeholders := strings.TrimRight(strings.Repeat("?,", len(predicate.Value.Strings)), ",")
		args := append([]any{}, field.Args...)
		for _, value := range predicate.Value.Strings {
			args = append(args, value)
		}
		operator := "IN"
		if predicate.Op == FilterNotIn {
			operator = "NOT IN"
		}
		condition := field.SQL + " " + operator + " (" + placeholders + ")"
		if field.RequiresExistence && predicate.Op == FilterIn {
			return "(" + field.ExistsSQL + ") AND (" + condition + ")", append(append([]any{}, field.ExistsArgs...), args...), nil
		}
		return condition, args, nil
	case FilterExists:
		return field.ExistsSQL, append([]any{}, field.ExistsArgs...), nil
	case FilterNotExists:
		return "NOT (" + field.ExistsSQL + ")", append([]any{}, field.ExistsArgs...), nil
	case FilterContains:
		condition := "positionCaseInsensitiveUTF8(toString(" + field.SQL + "), ?) > 0"
		args := append(append([]any{}, field.Args...), predicate.Value.String)
		if field.RequiresExistence {
			return "(" + field.ExistsSQL + ") AND (" + condition + ")", append(append([]any{}, field.ExistsArgs...), args...), nil
		}
		return condition, args, nil
	default:
		return "", nil, newError(ErrorInvalidFilter, "filter.operator", "unsupported filter operator %q", predicate.Op)
	}
}

func (c Compiler) compileOrder(signal Signal, orders []Order, aggregationAllowed bool, fallback string) (string, []any, error) {
	if len(orders) == 0 {
		return fallback, nil, nil
	}
	parts := make([]string, 0, len(orders))
	args := make([]any, 0)
	for _, order := range orders {
		direction := "ASC"
		if order.Direction == SortDescending {
			direction = "DESC"
		}
		switch order.Target {
		case OrderByAggregation:
			if !aggregationAllowed {
				return "", nil, newError(ErrorInvalidRequest, "query.order", "aggregation order is not valid for raw results")
			}
			parts = append(parts, "value "+direction)
		case OrderByField:
			field, err := c.Catalog.Resolve(signal, order.Field)
			if err != nil {
				return "", nil, err
			}
			parts = append(parts, field.SQL+" "+direction)
			args = append(args, field.Args...)
		default:
			return "", nil, newError(ErrorInvalidRequest, "query.order.target", "unsupported order target %q", order.Target)
		}
	}
	return strings.Join(parts, ", "), args, nil
}

func timeRangeCondition(signal Signal, timeRange TimeRange) (string, []any, error) {
	switch signal {
	case SignalLogs:
		// Always qualify physical timestamps. ClickHouse resolves SELECT aliases
		// in WHERE by default; a time-series result also aliases its millisecond
		// bucket as timestamp, which would otherwise compare milliseconds to the
		// log table's nanosecond timestamp and filter out every row.
		return "siginsight_logs.logs_v2.timestamp >= toUInt64(?) AND siginsight_logs.logs_v2.timestamp < toUInt64(?)", []any{timeRange.StartMS * 1_000_000, timeRange.EndMS * 1_000_000}, nil
	case SignalTraces:
		return "siginsight_traces.span_index_v3.timestamp >= fromUnixTimestamp64Milli(?) AND siginsight_traces.span_index_v3.timestamp < fromUnixTimestamp64Milli(?)", []any{timeRange.StartMS, timeRange.EndMS}, nil
	default:
		return "", nil, newError(ErrorUnsupported, "signal", "no time range mapping for %q", signal)
	}
}

func timeBucket(signal Signal, stepMS int64) (string, []any, error) {
	if stepMS <= 0 {
		return "", nil, newError(ErrorInvalidRequest, "stepMs", "time series requests require a positive step")
	}
	switch signal {
	case SignalLogs:
		// Qualifying physical timestamp columns prevents ClickHouse 25.5 from
		// resolving `timestamp` in GROUP BY to the SELECT output alias.
		// The physical column is UInt64 nanoseconds; cast placeholders so the
		// native driver cannot infer signed operands for time arithmetic.
		return "intDiv(siginsight_logs.logs_v2.timestamp, toUInt64(?)) * toUInt64(?)", []any{stepMS * 1_000_000, stepMS}, nil
	case SignalTraces:
		return "intDiv(toUnixTimestamp64Milli(span_index_v3.timestamp), ?) * ?", []any{stepMS, stepMS}, nil
	default:
		return "", nil, newError(ErrorUnsupported, "signal", "no time bucket mapping for %q", signal)
	}
}

func defaultFields(signal Signal) []FieldRef {
	if signal == SignalLogs {
		return []FieldRef{
			{Name: "timestamp", Context: FieldContextLog, Type: ValueTypeNumber},
			{Name: "id", Context: FieldContextLog, Type: ValueTypeString},
			{Name: "severity_text", Context: FieldContextLog, Type: ValueTypeString},
			{Name: "body", Context: FieldContextBody, Type: ValueTypeString},
			{Name: "trace_id", Context: FieldContextLog, Type: ValueTypeString},
			{Name: "span_id", Context: FieldContextLog, Type: ValueTypeString},
		}
	}
	return []FieldRef{
		{Name: "timestamp", Context: FieldContextSpan, Type: ValueTypeNumber},
		{Name: "trace_id", Context: FieldContextSpan, Type: ValueTypeString},
		{Name: "span_id", Context: FieldContextSpan, Type: ValueTypeString},
		{Name: "name", Context: FieldContextSpan, Type: ValueTypeString},
		{Name: "duration_nano", Context: FieldContextSpan, Type: ValueTypeNumber},
		{Name: "service.name", Context: FieldContextResource, Type: ValueTypeString},
	}
}

func defaultRawOrder(signal Signal) string {
	if signal == SignalLogs {
		return "timestamp DESC, id DESC"
	}
	return "timestamp DESC, span_id DESC"
}

func filterSQL(operator FilterOperator) string {
	switch operator {
	case FilterEqual:
		return "="
	case FilterNotEqual:
		return "!="
	case FilterGreaterThan:
		return ">"
	case FilterGreaterEq:
		return ">="
	case FilterLessThan:
		return "<"
	case FilterLessEq:
		return "<="
	default:
		return ""
	}
}

func comparisonSQL(operator ComparisonOperator) string {
	return filterSQL(FilterOperator(operator))
}

func scalarValue(value Value) any {
	switch value.Kind {
	case ValueString:
		return value.String
	case ValueNumber:
		return value.Number
	case ValueBool:
		return value.Bool
	default:
		return nil
	}
}
