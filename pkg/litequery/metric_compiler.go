package litequery

import (
	"fmt"
	"strings"
)

// compileMetric keeps the two aggregation phases explicit. The first CTE owns
// one value per fingerprint and bucket; the final SELECT only aggregates those
// per-series values across the requested dimensions.
func (c Compiler) compileMetric(plan Plan, aggregation MetricAggregation, common CommonQuery, signal Signal) (Statement, error) {
	if common.Cursor != "" {
		return Statement{}, newError(ErrorUnsupported, "query.cursor", "cursor compilation is not available until result scanning is implemented")
	}
	source, err := c.Catalog.MetricSource(signal)
	if err != nil {
		return Statement{}, err
	}
	if plan.ResultType != ResultTimeSeries && plan.ResultType != ResultScalar {
		return Statement{}, newError(ErrorInvalidRequest, "resultType", "metrics and meter require time series or scalar results")
	}
	if signal == SignalMeter && aggregation.Type != MetricSum {
		return Statement{}, newError(ErrorInvalidAggregation, "query.type", "meter only supports sum metrics")
	}

	if aggregation.Type == MetricHistogram {
		return c.compileHistogram(plan, source, aggregation, common)
	}
	return c.compileNumericMetric(plan, source, aggregation, common, signal)
}

func (c Compiler) compileNumericMetric(plan Plan, source MetricSource, aggregation MetricAggregation, common CommonQuery, signal Signal) (Statement, error) {
	series, seriesArgs, groupColumns, err := c.compileMetricSeries(source, aggregation, common, signal, false)
	if err != nil {
		return Statement{}, err
	}
	bucket, bucketArgs, err := metricBucket(plan)
	if err != nil {
		return Statement{}, err
	}
	metricName := aggregation.MetricName
	temporality := physicalTemporality(aggregation)
	groupNames := resultGroupNames(len(groupColumns))
	groupSelect := prefixedColumns("series", groupNames)

	pointWhere := "points.metric_name = ? AND lower(points.temporality) = lower(?) AND points.unix_milli >= ? AND points.unix_milli < ?"
	pointArgs := []any{metricName, temporality, plan.Range.StartMS, plan.Range.EndMS}
	perSeries, requiresWindow, err := numericTemporalExpression(aggregation, plan)
	if err != nil {
		return Statement{}, err
	}
	usesTimestamp := plan.ResultType == ResultTimeSeries || requiresWindow
	groupBy := append([]string{"points.fingerprint"}, groupNames...)
	if plan.ResultType == ResultTimeSeries {
		groupBy = append(groupBy, bucket)
	} else if requiresWindow {
		groupBy = append(groupBy, "points.unix_milli")
	}

	bucketSelect := ""
	if plan.ResultType == ResultTimeSeries {
		bucketSelect = bucket + " AS timestamp, "
	} else if requiresWindow {
		bucketSelect = "points.unix_milli AS timestamp, "
	}
	seriesValue := perSeries
	if requiresWindow {
		base := rawCounterAggregation(aggregation)
		if base == "" {
			return Statement{}, newError(ErrorInvalidAggregation, "query.timeAggregation", "unsupported counter aggregation %q", aggregation.TimeAggregation)
		}
		seriesValue = "bucket_value"
		perSeries = "bucket_value"
		if aggregation.Temporality == TemporalityCumulative {
			previousValue := "lagInFrame(bucket_value, 1) OVER counter_window"
			previousTime := "lagInFrame(timestamp, 1) OVER counter_window"
			delta := "if(bucket_value < " + previousValue + ", bucket_value, bucket_value - " + previousValue + ")"
			if aggregation.TimeAggregation == TimeAggregateRate {
				perSeries = "if(row_number() OVER counter_window = 1, NULL, " + delta + " / ((timestamp - " + previousTime + ") / 1000.0))"
			} else {
				perSeries = "if(row_number() OVER counter_window = 1, NULL, " + delta + ")"
			}
		} else if aggregation.TimeAggregation == TimeAggregateRate {
			if plan.ResultType == ResultTimeSeries {
				perSeries = "bucket_value / (? / 1000.0)"
			} else {
				perSeries = "bucket_value / ((? - ?) / 1000.0)"
			}
		}
		seriesValue = base
	}

	var ctes []string
	ctes = append(ctes, "__lite_series AS ("+series+")")
	baseSelect := "SELECT points.fingerprint"
	if plan.ResultType == ResultTimeSeries {
		baseSelect += ", " + strings.TrimSuffix(bucketSelect, ", ")
	} else if requiresWindow {
		baseSelect += ", " + strings.TrimSuffix(bucketSelect, ", ")
	}
	if len(groupSelect) != 0 {
		baseSelect += ", " + strings.Join(groupSelect, ", ")
	}
	baseSelect += ", " + seriesValue + " AS bucket_value"
	base := baseSelect + " FROM " + source.PointsTable + " AS points INNER JOIN __lite_series AS series ON points.fingerprint = series.fingerprint WHERE " + pointWhere + " GROUP BY " + strings.Join(groupBy, ", ")

	args := append([]any{}, seriesArgs...)
	if plan.ResultType == ResultTimeSeries {
		args = append(args, bucketArgs...)
	}
	args = append(args, pointArgs...)
	if plan.ResultType == ResultTimeSeries {
		args = append(args, bucketArgs...)
	}
	if requiresWindow {
		ctes = append(ctes, "__lite_bucketed AS ("+base+")")
		windowSelect := "SELECT fingerprint"
		if usesTimestamp {
			windowSelect += ", timestamp"
		}
		if len(groupNames) != 0 {
			windowSelect += ", " + strings.Join(groupNames, ", ")
		}
		windowSelect += ", " + perSeries + " AS per_series_value FROM __lite_bucketed"
		partition := "PARTITION BY fingerprint"
		if usesTimestamp {
			partition += " ORDER BY timestamp"
		}
		windowSelect += " WINDOW counter_window AS (" + partition + ")"
		ctes = append(ctes, "__lite_temporal AS ("+windowSelect+")")
		if aggregation.Temporality != TemporalityCumulative && aggregation.TimeAggregation == TimeAggregateRate {
			if plan.ResultType == ResultTimeSeries {
				args = append(args, plan.StepMS)
			} else {
				args = append(args, plan.Range.EndMS, plan.Range.StartMS)
			}
		}
	} else {
		ctes = append(ctes, "__lite_temporal AS ("+strings.Replace(base, " AS bucket_value", " AS per_series_value", 1)+")")
	}

	space, err := spaceAggregation(aggregation.SpaceAggregation, "per_series_value")
	if err != nil {
		return Statement{}, err
	}
	selects := make([]string, 0, len(groupNames)+2)
	columns := make([]ResultColumn, 0, len(groupNames)+2)
	if plan.ResultType == ResultTimeSeries {
		selects = append(selects, "timestamp")
		columns = append(columns, ResultColumn{Name: "timestamp"})
	}
	for index, name := range groupNames {
		field := common.GroupBy[index]
		selects = append(selects, name)
		columns = append(columns, ResultColumn{Name: name, Field: &field})
	}
	selects = append(selects, space+" AS value")
	columns = append(columns, ResultColumn{Name: "value"})
	groupFinal := append([]string{}, groupNames...)
	if plan.ResultType == ResultTimeSeries {
		groupFinal = append([]string{"timestamp"}, groupFinal...)
	}
	query := "WITH " + strings.Join(ctes, ", ") + " SELECT " + strings.Join(selects, ", ") + " FROM __lite_temporal"
	if len(groupFinal) != 0 {
		query += " GROUP BY " + strings.Join(groupFinal, ", ")
	}
	if common.Predicate != nil {
		query += " HAVING " + space + " " + comparisonSQL(common.Predicate.Operator) + " ?"
		args = append(args, common.Predicate.Value)
	}
	order, err := metricOrder(common, groupNames, plan.ResultType)
	if err != nil {
		return Statement{}, err
	}
	query += " ORDER BY " + order
	if common.Limit != 0 {
		query += " LIMIT ?"
		args = append(args, common.Limit)
	}
	return Statement{Name: common.Name, SQL: query, Args: args, Columns: columns}, nil
}

func (c Compiler) compileHistogram(plan Plan, source MetricSource, aggregation MetricAggregation, common CommonQuery) (Statement, error) {
	if source.SeriesTable == "" {
		return Statement{}, newError(ErrorUnsupported, "query.type", "histograms are not supported by meter")
	}
	if !strings.HasSuffix(aggregation.MetricName, ".bucket") {
		return Statement{}, newError(ErrorInvalidAggregation, "query.metricName", "explicit histogram metric names must end in .bucket")
	}
	quantile, err := histogramQuantile(aggregation.SpaceAggregation)
	if err != nil {
		return Statement{}, err
	}
	series, seriesArgs, groupColumns, err := c.compileMetricSeries(source, aggregation, common, SignalMetrics, true)
	if err != nil {
		return Statement{}, err
	}
	bucket, bucketArgs, err := metricBucket(plan)
	if err != nil {
		return Statement{}, err
	}
	groupNames := resultGroupNames(len(groupColumns))
	groupSelect := prefixedColumns("series", append(groupNames, "le"))
	pointWhere := "points.metric_name = ? AND lower(points.temporality) = lower(?) AND points.unix_milli >= ? AND points.unix_milli < ?"
	args := append([]any{}, seriesArgs...)
	if plan.ResultType == ResultTimeSeries {
		args = append(args, bucketArgs...)
	}
	args = append(args, aggregation.MetricName, physicalTemporality(aggregation), plan.Range.StartMS, plan.Range.EndMS)
	groupBy := append([]string{"points.fingerprint"}, append(groupNames, "le")...)
	bucketSelect := ""
	if plan.ResultType == ResultTimeSeries {
		bucketSelect = bucket + " AS timestamp, "
		groupBy = append(groupBy, bucket)
		args = append(args, bucketArgs...)
	}
	// Histograms are stored as cumulative buckets. The compact engine uses the
	// requested bucket value directly for a scalar range, and a bucket delta for
	// time series. The executor later owns gap filling and final formatting.
	perSeries := "argMax(points.value, points.unix_milli)"
	if physicalTemporality(aggregation) == "Cumulative" && plan.ResultType == ResultTimeSeries {
		perSeries = "if(row_number() OVER histogram_window = 1, NULL, if(bucket_value < lagInFrame(bucket_value, 1) OVER histogram_window, bucket_value, bucket_value - lagInFrame(bucket_value, 1) OVER histogram_window))"
	}
	base := "SELECT points.fingerprint, " + bucketSelect
	if len(groupSelect) != 0 {
		base += strings.Join(groupSelect, ", ") + ", "
	}
	base += "argMax(points.value, points.unix_milli) AS bucket_value FROM " + source.PointsTable + " AS points INNER JOIN __lite_series AS series ON points.fingerprint = series.fingerprint WHERE " + pointWhere + " GROUP BY " + strings.Join(groupBy, ", ")
	ctes := []string{"__lite_series AS (" + series + ")", "__lite_bucketed AS (" + base + ")"}
	if plan.ResultType == ResultTimeSeries {
		window := "SELECT fingerprint, timestamp"
		if len(groupNames) != 0 {
			window += ", " + strings.Join(groupNames, ", ")
		}
		window += ", le, " + perSeries + " AS per_series_value FROM __lite_bucketed WINDOW histogram_window AS (PARTITION BY fingerprint ORDER BY timestamp)"
		ctes = append(ctes, "__lite_temporal AS ("+window+")")
	} else {
		ctes = append(ctes, "__lite_temporal AS (SELECT fingerprint"+optionalColumns(groupNames)+", le, bucket_value AS per_series_value FROM __lite_bucketed)")
	}
	spatialGroups := append([]string{}, groupNames...)
	if plan.ResultType == ResultTimeSeries {
		spatialGroups = append([]string{"timestamp"}, spatialGroups...)
	}
	intermediate := "__lite_histogram AS (SELECT " + optionalSelectColumns(spatialGroups) + "le, sum(per_series_value) AS bucket_value FROM __lite_temporal WHERE per_series_value IS NOT NULL GROUP BY " + strings.Join(append(spatialGroups, "le"), ", ") + ")"
	ctes = append(ctes, intermediate)
	weightPartition := ""
	if len(spatialGroups) != 0 {
		weightPartition = "PARTITION BY " + strings.Join(spatialGroups, ", ") + " "
	}
	upperBound := "if(le = '+Inf', 1e308, toFloat64(le))"
	weights := "__lite_histogram_weights AS (SELECT " + optionalSelectColumns(spatialGroups) + upperBound + " AS upper_bound, greatest(bucket_value - lagInFrame(bucket_value, 1, 0) OVER bucket_window, 0) AS bucket_weight FROM __lite_histogram WINDOW bucket_window AS (" + weightPartition + "ORDER BY " + upperBound + "))"
	ctes = append(ctes, weights)
	selects := make([]string, 0, len(groupNames)+2)
	columns := make([]ResultColumn, 0, len(groupNames)+2)
	if plan.ResultType == ResultTimeSeries {
		selects = append(selects, "timestamp")
		columns = append(columns, ResultColumn{Name: "timestamp"})
	}
	for index, name := range groupNames {
		field := common.GroupBy[index]
		selects = append(selects, name)
		columns = append(columns, ResultColumn{Name: name, Field: &field})
	}
	selects = append(selects, fmt.Sprintf("quantileExactWeighted(%.2f)(upper_bound, toUInt64(bucket_weight)) AS value", quantile))
	columns = append(columns, ResultColumn{Name: "value"})
	query := "WITH " + strings.Join(ctes, ", ") + " SELECT " + strings.Join(selects, ", ") + " FROM __lite_histogram_weights"
	if len(spatialGroups) != 0 {
		query += " GROUP BY " + strings.Join(spatialGroups, ", ")
	}
	order, err := metricOrder(common, groupNames, plan.ResultType)
	if err != nil {
		return Statement{}, err
	}
	query += " ORDER BY " + order
	if common.Limit != 0 {
		query += " LIMIT ?"
		args = append(args, common.Limit)
	}
	return Statement{Name: common.Name, SQL: query, Args: args, Columns: columns}, nil
}

func (c Compiler) compileMetricSeries(source MetricSource, aggregation MetricAggregation, common CommonQuery, signal Signal, histogram bool) (string, []any, []ResultColumn, error) {
	if source.SeriesTable == "" {
		return c.compileMeterSeries(source, aggregation, common)
	}
	fields := append([]FieldRef{}, common.GroupBy...)
	if histogram {
		fields = append(fields, FieldRef{Name: "le", Context: FieldContextLabel, Type: ValueTypeString})
	}
	selects := []string{"fingerprint"}
	groups := []string{"fingerprint"}
	selectArgs := make([]any, 0)
	groupArgs := make([]any, 0)
	columns := make([]ResultColumn, 0, len(common.GroupBy))
	for index, field := range fields {
		resolved, err := resolveMetricField(field, signal)
		if err != nil {
			return "", nil, nil, err
		}
		alias := "le"
		if index < len(common.GroupBy) {
			alias = fmt.Sprintf("group_%d", index)
			copy := field
			columns = append(columns, ResultColumn{Name: alias, Field: &copy})
		}
		selects = append(selects, resolved.SQL+" AS "+alias)
		groups = append(groups, resolved.SQL)
		selectArgs = append(selectArgs, resolved.Args...)
		groupArgs = append(groupArgs, resolved.Args...)
	}
	where, whereArgs, err := compileMetricFilter(common.Filter, signal)
	if err != nil {
		return "", nil, nil, err
	}
	args := append(selectArgs, aggregation.MetricName, metricTypeName(aggregation.Type), physicalTemporality(aggregation), true)
	args = append(args, whereArgs...)
	args = append(args, groupArgs...)
	query := "SELECT " + strings.Join(selects, ", ") + " FROM " + source.SeriesTable + " WHERE metric_name = ? AND type = ? AND lower(temporality) = lower(?) AND __normalized = ?"
	if where != "" {
		query += " AND (" + where + ")"
	}
	query += " GROUP BY " + strings.Join(groups, ", ")
	return query, args, columns, nil
}

func (c Compiler) compileMeterSeries(source MetricSource, aggregation MetricAggregation, common CommonQuery) (string, []any, []ResultColumn, error) {
	fields := common.GroupBy
	selects := []string{"fingerprint"}
	groups := []string{"fingerprint"}
	selectArgs := make([]any, 0)
	groupArgs := make([]any, 0)
	columns := make([]ResultColumn, 0, len(fields))
	for index, field := range fields {
		resolved, err := resolveMetricField(field, SignalMeter)
		if err != nil {
			return "", nil, nil, err
		}
		alias := fmt.Sprintf("group_%d", index)
		selects = append(selects, resolved.SQL+" AS "+alias)
		groups = append(groups, resolved.SQL)
		selectArgs = append(selectArgs, resolved.Args...)
		groupArgs = append(groupArgs, resolved.Args...)
		copy := field
		columns = append(columns, ResultColumn{Name: alias, Field: &copy})
	}
	where, whereArgs, err := compileMetricFilter(common.Filter, SignalMeter)
	if err != nil {
		return "", nil, nil, err
	}
	args := append(selectArgs, aggregation.MetricName, physicalTemporality(aggregation))
	args = append(args, whereArgs...)
	args = append(args, groupArgs...)
	query := "SELECT " + strings.Join(selects, ", ") + " FROM " + source.PointsTable + " WHERE metric_name = ? AND lower(temporality) = lower(?)"
	if where != "" {
		query += " AND (" + where + ")"
	}
	query += " GROUP BY " + strings.Join(groups, ", ")
	return query, args, columns, nil
}

func resolveMetricField(field FieldRef, signal Signal) (ResolvedField, error) {
	if err := validateField(field, "field"); err != nil {
		return ResolvedField{}, err
	}
	switch field.Context {
	case FieldContextLabel:
		if field.Type != ValueTypeString {
			return ResolvedField{}, newError(ErrorInvalidRequest, "field.type", "metric labels are strings")
		}
		return ResolvedField{SQL: "JSONExtractString(labels, ?)", Args: []any{field.Name}, ExistsSQL: "JSONHas(labels, ?)", ExistsArgs: []any{field.Name}, RequiresExistence: true}, nil
	case FieldContextAttribute:
		if signal != SignalMetrics || field.Type != ValueTypeString {
			return ResolvedField{}, newError(ErrorUnsupported, "field.context", "only string point attributes are supported for metrics")
		}
		return resolveMapField("attrs", field)
	case FieldContextResource:
		if signal != SignalMetrics || field.Type != ValueTypeString {
			return ResolvedField{}, newError(ErrorUnsupported, "field.context", "only string resource attributes are supported for metrics")
		}
		return resolveMapField("resource_attrs", field)
	case FieldContextScope:
		if signal != SignalMetrics || field.Type != ValueTypeString {
			return ResolvedField{}, newError(ErrorUnsupported, "field.context", "only string scope attributes are supported for metrics")
		}
		return resolveMapField("scope_attrs", field)
	case FieldContextMetric:
		if field.Type != ValueTypeString {
			return ResolvedField{}, newError(ErrorInvalidRequest, "field.type", "metric intrinsic fields are strings")
		}
		switch field.Name {
		case "metric_name", "temporality", "type", "unit", "description":
			return staticField(field.Name, field.Type), nil
		default:
			return ResolvedField{}, newError(ErrorUnsupported, "field.name", "metric field %q is not in the schema catalog", field.Name)
		}
	default:
		return ResolvedField{}, newError(ErrorUnsupported, "field.context", "metric field context %q is unsupported", field.Context)
	}
}

func compileMetricFilter(node FilterNode, signal Signal) (string, []any, error) {
	if node == nil {
		return "", nil, nil
	}
	switch current := node.(type) {
	case LogicalFilter:
		parts := make([]string, 0, len(current.Items))
		args := make([]any, 0)
		separator := " AND "
		if current.Operator == BooleanOr {
			separator = " OR "
		}
		for _, item := range current.Items {
			part, itemArgs, err := compileMetricFilter(item, signal)
			if err != nil {
				return "", nil, err
			}
			parts = append(parts, "("+part+")")
			args = append(args, itemArgs...)
		}
		return strings.Join(parts, separator), args, nil
	case Predicate:
		field, err := resolveMetricField(current.Field, signal)
		if err != nil {
			return "", nil, err
		}
		return compilePredicate(field, current)
	default:
		return "", nil, newError(ErrorInvalidFilter, "filter", "unsupported filter node %T", node)
	}
}

func metricBucket(plan Plan) (string, []any, error) {
	if plan.ResultType == ResultScalar {
		return "", nil, nil
	}
	if plan.StepMS <= 0 {
		return "", nil, newError(ErrorInvalidRequest, "stepMs", "time series requests require a positive step")
	}
	return "intDiv(points.unix_milli, ?) * ?", []any{plan.StepMS, plan.StepMS}, nil
}

func numericTemporalExpression(aggregation MetricAggregation, plan Plan) (string, bool, error) {
	switch aggregation.TimeAggregation {
	case TimeAggregateLatest:
		return "argMax(points.value, points.unix_milli)", false, nil
	case TimeAggregateSum:
		return "sum(points.value)", false, nil
	case TimeAggregateAvg:
		return "avg(points.value)", false, nil
	case TimeAggregateMin:
		return "min(points.value)", false, nil
	case TimeAggregateMax:
		return "max(points.value)", false, nil
	case TimeAggregateCount:
		return "count(points.value)", false, nil
	case TimeAggregateRate, TimeAggregateIncrease:
		return "", true, nil
	default:
		return "", false, newError(ErrorInvalidAggregation, "query.timeAggregation", "unsupported time aggregation %q", aggregation.TimeAggregation)
	}
}

func rawCounterAggregation(aggregation MetricAggregation) string {
	if aggregation.Temporality == TemporalityCumulative {
		return "argMax(points.value, points.unix_milli)"
	}
	return "sum(points.value)"
}

func spaceAggregation(aggregation SpaceAggregation, value string) (string, error) {
	switch aggregation {
	case SpaceAggregateSum:
		return "sum(" + value + ")", nil
	case SpaceAggregateAvg:
		return "avg(" + value + ")", nil
	case SpaceAggregateMin:
		return "min(" + value + ")", nil
	case SpaceAggregateMax:
		return "max(" + value + ")", nil
	case SpaceAggregateCount:
		return "count(" + value + ")", nil
	default:
		return "", newError(ErrorInvalidAggregation, "query.spaceAggregation", "unsupported space aggregation %q", aggregation)
	}
}

func histogramQuantile(aggregation SpaceAggregation) (float64, error) {
	switch aggregation {
	case SpaceAggregateP50:
		return 0.50, nil
	case SpaceAggregateP90:
		return 0.90, nil
	case SpaceAggregateP95:
		return 0.95, nil
	case SpaceAggregateP99:
		return 0.99, nil
	default:
		return 0, newError(ErrorInvalidAggregation, "query.spaceAggregation", "histogram requires p50, p90, p95, or p99")
	}
}

func metricTypeName(metricType MetricType) string {
	switch metricType {
	case MetricGauge:
		return "Gauge"
	case MetricSum:
		return "Sum"
	case MetricHistogram:
		return "Histogram"
	default:
		return ""
	}
}

func physicalTemporality(aggregation MetricAggregation) string {
	if aggregation.Type == MetricHistogram && aggregation.Temporality == TemporalityUnspecified {
		return "Cumulative"
	}
	switch aggregation.Temporality {
	case TemporalityDelta:
		return "Delta"
	case TemporalityCumulative:
		return "Cumulative"
	default:
		return "Unspecified"
	}
}

func metricOrder(common CommonQuery, groupNames []string, resultType ResultType) (string, error) {
	if len(common.Order) == 0 {
		if resultType == ResultTimeSeries {
			return "timestamp ASC", nil
		}
		return "value DESC", nil
	}
	parts := make([]string, 0, len(common.Order))
	for _, order := range common.Order {
		direction := "ASC"
		if order.Direction == SortDescending {
			direction = "DESC"
		}
		switch order.Target {
		case OrderByAggregation:
			parts = append(parts, "value "+direction)
		case OrderByField:
			found := -1
			for index, field := range common.GroupBy {
				if field == order.Field {
					found = index
					break
				}
			}
			if found < 0 {
				return "", newError(ErrorInvalidRequest, "query.order", "metric field ordering requires a groupBy field")
			}
			parts = append(parts, groupNames[found]+" "+direction)
		default:
			return "", newError(ErrorInvalidRequest, "query.order.target", "unsupported order target %q", order.Target)
		}
	}
	return strings.Join(parts, ", "), nil
}

func resultGroupNames(length int) []string {
	names := make([]string, length)
	for index := range names {
		names[index] = fmt.Sprintf("group_%d", index)
	}
	return names
}

func prefixedColumns(prefix string, names []string) []string {
	columns := make([]string, len(names))
	for index, name := range names {
		columns[index] = prefix + "." + name
	}
	return columns
}

func optionalColumns(columns []string) string {
	if len(columns) == 0 {
		return ""
	}
	return ", " + strings.Join(columns, ", ")
}

func optionalSelectColumns(columns []string) string {
	if len(columns) == 0 {
		return ""
	}
	return strings.Join(columns, ", ") + ", "
}
