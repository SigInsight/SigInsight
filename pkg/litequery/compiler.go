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
	fields := append([]FieldRef(nil), common.Select...)
	if len(fields) == 0 {
		fields = defaultFields(signal)
	}
	if signal == SignalTraces {
		fields = withTraceRawIdentityFields(fields)
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
	where, whereArgs, err := c.compileWhere(table, signal, common.Filter, plan.Range)
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
	order = appendRawOrderTieBreaker(order, signal, common.Order)
	args = append(args, orderArgs...)
	limit := common.Limit
	if limit == 0 {
		limit = 100
	}
	args = append(args, limit+1)
	suffix := " LIMIT ?"
	if common.Offset != 0 {
		suffix += " OFFSET ?"
		args = append(args, common.Offset)
	}
	return Statement{
		Name:       common.Name,
		SQL:        "SELECT " + strings.Join(selects, ", ") + " FROM " + table + " WHERE " + where + " ORDER BY " + order + suffix,
		Args:       args,
		Columns:    columns,
		Pagination: &Pagination{Limit: limit, Offset: common.Offset},
	}, nil
}

// Trace list rows must retain the identifiers needed to open the trace detail.
// They are transport fields, not user-selected display columns.
func withTraceRawIdentityFields(fields []FieldRef) []FieldRef {
	identityFields := []FieldRef{
		{Name: "timestamp", Context: FieldContextSpan, Type: ValueTypeNumber},
		{Name: "trace_id", Context: FieldContextSpan, Type: ValueTypeString},
		{Name: "span_id", Context: FieldContextSpan, Type: ValueTypeString},
	}
	for _, identity := range identityFields {
		found := false
		for _, field := range fields {
			if field.Name == identity.Name {
				found = true
				break
			}
		}
		if !found {
			fields = append(fields, identity)
		}
	}
	return fields
}

func (c Compiler) compileTraceSummary(table string, plan Plan, common CommonQuery) (Statement, error) {
	where, matchingArgs, err := c.compileWhere(table, SignalTraces, common.Filter, plan.Range)
	if err != nil {
		return Statement{}, err
	}
	serviceField := FieldRef{Name: "service.name", Context: FieldContextResource, Type: ValueTypeString}
	service, err := c.Catalog.Resolve(SignalTraces, serviceField)
	if err != nil {
		return Statement{}, err
	}
	rootRange, rootRangeArgs, err := timeRangeCondition(table, SignalTraces, plan.Range)
	if err != nil {
		return Statement{}, err
	}
	order := "timestamp DESC, trace_id DESC"
	if len(common.Order) != 0 {
		compiledOrder, err := compileTraceSummaryOrder(common.Order)
		if err != nil {
			return Statement{}, err
		}
		order = compiledOrder
	}
	limit := common.Limit
	if limit == 0 {
		limit = 100
	}
	args := append([]any{}, matchingArgs...)
	args = append(args, service.Args...)
	args = append(args, rootRangeArgs...)
	args = append(args, limit+1)
	suffix := " LIMIT ?"
	if common.Offset != 0 {
		suffix += " OFFSET ?"
		args = append(args, common.Offset)
	}
	return Statement{
		Name: common.Name,
		SQL: "WITH __lite_matching_traces AS (SELECT DISTINCT trace_id FROM " + table + " WHERE " + where + "), " +
			"__lite_trace_spans AS (SELECT trace_id, timestamp, span_id, parent_span_id, duration_nano, " + service.SQL + " AS service_name, name FROM " + table +
			" WHERE " + rootRange + " AND trace_id IN (SELECT trace_id FROM __lite_matching_traces)), " +
			"__lite_trace_stats AS (SELECT trace_id, count() AS span_count FROM __lite_trace_spans GROUP BY trace_id), " +
			"__lite_roots AS (SELECT trace_id, " +
			"argMax(timestamp, tuple(parent_span_id = '', duration_nano, timestamp, span_id)) AS root_timestamp, " +
			"argMax(duration_nano, tuple(parent_span_id = '', duration_nano, timestamp, span_id)) AS root_duration_nano, " +
			"argMax(service_name, tuple(parent_span_id = '', duration_nano, timestamp, span_id)) AS root_service_name, " +
			"argMax(name, tuple(parent_span_id = '', duration_nano, timestamp, span_id)) AS root_name FROM __lite_trace_spans GROUP BY trace_id) " +
			"SELECT roots.trace_id AS trace_id, roots.root_timestamp AS timestamp, stats.span_count AS span_count, " +
			"roots.root_duration_nano AS duration_nano, roots.root_service_name AS service_name, roots.root_name AS name " +
			"FROM __lite_trace_stats AS stats INNER JOIN __lite_roots AS roots USING (trace_id) ORDER BY " + order + suffix,
		Args:       args,
		Pagination: &Pagination{Limit: limit, Offset: common.Offset},
		Columns: []ResultColumn{
			{Name: "trace_id", Field: fieldPointer(FieldRef{Name: "trace_id", Context: FieldContextSpan, Type: ValueTypeString})},
			{Name: "timestamp", Field: fieldPointer(FieldRef{Name: "timestamp", Context: FieldContextSpan, Type: ValueTypeNumber})},
			{Name: "span_count", Field: fieldPointer(FieldRef{Name: "span_count", Context: FieldContextSpan, Type: ValueTypeNumber})},
			{Name: "duration_nano", Field: fieldPointer(FieldRef{Name: "duration_nano", Context: FieldContextSpan, Type: ValueTypeNumber})},
			{Name: "service_name", Field: fieldPointer(serviceField)},
			{Name: "name", Field: fieldPointer(FieldRef{Name: "name", Context: FieldContextSpan, Type: ValueTypeString})},
		},
	}, nil
}

func compileTraceSummaryOrder(orders []Order) (string, error) {
	aliases := map[string]string{
		"trace_id":       "trace_id",
		"timestamp":      "timestamp",
		"span_count":     "span_count",
		"duration_nano":  "duration_nano",
		"trace_duration": "duration_nano",
		"service.name":   "service_name",
		"name":           "name",
	}
	parts := make([]string, 0, len(orders))
	for _, order := range orders {
		alias, ok := aliases[order.Field.Name]
		if order.Target != OrderByField || !ok {
			return "", newError(ErrorInvalidRequest, "query.order", "trace summary ordering does not support field %q", order.Field.Name)
		}
		direction := "ASC"
		if order.Direction == SortDescending {
			direction = "DESC"
		}
		parts = append(parts, alias+" "+direction)
	}
	return strings.Join(parts, ", "), nil
}

func fieldPointer(field FieldRef) *FieldRef { return &field }

func (c Compiler) compileAggregate(table string, signal Signal, plan Plan, common CommonQuery, aggregation compiledExpression) (Statement, error) {
	selects := make([]string, 0, len(common.GroupBy)+2)
	columns := make([]ResultColumn, 0, len(common.GroupBy)+2)
	selectArgs := make([]any, 0, len(aggregation.Args)+8)
	groupArgs := make([]any, 0, len(aggregation.Args)+8)
	groupBy := make([]string, 0, len(common.GroupBy)+1)
	if plan.ResultType == ResultTimeSeries {
		bucket, bucketArgs, err := timeBucket(table, signal, plan.StepMS)
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

	where, whereArgs, err := c.compileWhere(table, signal, common.Filter, plan.Range)
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
	resultLimit := uint32(0)
	if common.Limit != 0 {
		query += " LIMIT ?"
		args = append(args, common.Limit+1)
		resultLimit = common.Limit
	}
	return Statement{Name: common.Name, SQL: query, Args: args, Columns: columns, ResultLimit: resultLimit}, nil
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

func (c Compiler) compileWhere(table string, signal Signal, filter FilterNode, timeRange TimeRange) (string, []any, error) {
	timeCondition, args, err := timeRangeCondition(table, signal, timeRange)
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
		if scope, ok := TraceScopeForName(signal, current.Field.Context, current.Field.Name); ok {
			return c.compileTraceScopePredicate(scope, current)
		}
		field, err := c.Catalog.Resolve(signal, current.Field)
		if err != nil {
			return "", nil, err
		}
		return compilePredicate(field, current)
	default:
		return "", nil, newError(ErrorInvalidFilter, "filter", "unsupported filter node %T", node)
	}
}

func (c Compiler) compileTraceScopePredicate(scope TraceScope, predicate Predicate) (string, []any, error) {
	if predicate.Field.Type != ValueTypeBool {
		return "", nil, newError(ErrorInvalidRequest, "filter.field", "trace scope %q has boolean type", predicate.Field.Name)
	}
	if predicate.Op != FilterEqual || predicate.Value.Kind != ValueBool || !predicate.Value.Bool {
		return "", nil, newError(ErrorInvalidFilter, "filter", "trace scope %q only supports = true", predicate.Field.Name)
	}
	switch scope {
	case TraceScopeRoot:
		return "parent_span_id = ''", nil, nil
	case TraceScopeEntrypoint:
		// OTel receiving boundaries have a remote parent and a receiving span
		// kind. Root spans remain a separate scope, so require a parent ID here.
		return "(parent_span_id != '') AND (kind IN (2, 5)) AND (is_remote = 'yes')", nil, nil
	default:
		return "", nil, newError(ErrorInvalidFilter, "filter.field", "unknown trace scope %q", predicate.Field.Name)
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
		values := listValues(predicate.Value)
		placeholders := strings.TrimRight(strings.Repeat("?,", len(values)), ",")
		args := append([]any{}, field.Args...)
		args = append(args, values...)
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
	case FilterNotContains:
		return compileStringPattern(field, "positionCaseInsensitiveUTF8(toString("+field.SQL+"), ?) = 0", predicate.Value.String, false)
	case FilterLike:
		return compileStringPattern(field, "toString("+field.SQL+") LIKE ?", predicate.Value.String, true)
	case FilterNotLike:
		return compileStringPattern(field, "NOT (toString("+field.SQL+") LIKE ?)", predicate.Value.String, false)
	case FilterILike:
		return compileStringPattern(field, "toString("+field.SQL+") ILIKE ?", predicate.Value.String, true)
	case FilterNotILike:
		return compileStringPattern(field, "NOT (toString("+field.SQL+") ILIKE ?)", predicate.Value.String, false)
	case FilterRegexp:
		return compileStringPattern(field, "match(toString("+field.SQL+"), ?)", predicate.Value.String, true)
	case FilterNotRegexp:
		return compileStringPattern(field, "NOT match(toString("+field.SQL+"), ?)", predicate.Value.String, false)
	default:
		return "", nil, newError(ErrorInvalidFilter, "filter.operator", "unsupported filter operator %q", predicate.Op)
	}
}

func compileStringPattern(field ResolvedField, condition, value string, requireExistence bool) (string, []any, error) {
	args := append(append([]any{}, field.Args...), value)
	if field.RequiresExistence && requireExistence {
		return "(" + field.ExistsSQL + ") AND (" + condition + ")", append(append([]any{}, field.ExistsArgs...), args...), nil
	}
	return condition, args, nil
}

func listValues(value Value) []any {
	result := make([]any, 0, len(value.Strings)+len(value.Numbers)+len(value.Bools))
	for _, item := range value.Strings {
		result = append(result, item)
	}
	for _, item := range value.Numbers {
		result = append(result, item)
	}
	for _, item := range value.Bools {
		result = append(result, item)
	}
	return result
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

func timeRangeCondition(table string, signal Signal, timeRange TimeRange) (string, []any, error) {
	switch signal {
	case SignalLogs:
		if timeRange.StartMS > maxLogTimestampMS || timeRange.EndMS > maxLogTimestampMS {
			return "", nil, newError(ErrorInvalidRequest, "range", "log query range exceeds nanosecond timestamp capacity")
		}
		// Always qualify physical timestamps. ClickHouse resolves SELECT aliases
		// in WHERE by default; a time-series result also aliases its millisecond
		// bucket as timestamp, which would otherwise compare milliseconds to the
		// log table's nanosecond timestamp and filter out every row.
		return table + ".timestamp >= toUInt64(?) AND " + table + ".timestamp < toUInt64(?)", []any{timeRange.StartMS * 1_000_000, timeRange.EndMS * 1_000_000}, nil
	case SignalTraces:
		return table + ".timestamp >= fromUnixTimestamp64Milli(?) AND " + table + ".timestamp < fromUnixTimestamp64Milli(?)", []any{timeRange.StartMS, timeRange.EndMS}, nil
	default:
		return "", nil, newError(ErrorUnsupported, "signal", "no time range mapping for %q", signal)
	}
}

func timeBucket(table string, signal Signal, stepMS int64) (string, []any, error) {
	if stepMS <= 0 {
		return "", nil, newError(ErrorInvalidRequest, "stepMs", "time series requests require a positive step")
	}
	switch signal {
	case SignalLogs:
		if stepMS > maxLogTimestampMS {
			return "", nil, newError(ErrorInvalidRequest, "stepMs", "log query step exceeds nanosecond timestamp capacity")
		}
		// Qualifying physical timestamp columns prevents ClickHouse 25.5 from
		// resolving `timestamp` in GROUP BY to the SELECT output alias.
		// The physical column is UInt64 nanoseconds; cast placeholders so the
		// native driver cannot infer signed operands for time arithmetic.
		return "intDiv(" + table + ".timestamp, toUInt64(?)) * toUInt64(?)", []any{stepMS * 1_000_000, stepMS}, nil
	case SignalTraces:
		return "intDiv(toUnixTimestamp64Milli(" + table + ".timestamp), ?) * ?", []any{stepMS, stepMS}, nil
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

func appendRawOrderTieBreaker(order string, signal Signal, orders []Order) string {
	if len(orders) == 0 {
		return order
	}
	name := "id"
	context := FieldContextLog
	if signal == SignalTraces {
		name = "span_id"
		context = FieldContextSpan
	}
	for _, candidate := range orders {
		if candidate.Target == OrderByField && candidate.Field.Context == context && candidate.Field.Name == name {
			return order
		}
	}
	direction := "ASC"
	if orders[len(orders)-1].Direction == SortDescending {
		direction = "DESC"
	}
	return order + ", " + name + " " + direction
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
