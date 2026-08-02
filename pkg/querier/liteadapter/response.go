package liteadapter

import (
	"fmt"
	"math"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/SigNoz/signoz/pkg/errors"
	"github.com/SigNoz/signoz/pkg/litequery"
	qbtypes "github.com/SigNoz/signoz/pkg/types/querybuildertypes/querybuildertypesv5"
	"github.com/SigNoz/signoz/pkg/types/telemetrytypes"
)

// FromLite restores the stable V5 response shape used by the frontend. The
// adapter owns labels and column descriptors because they are transport
// concerns, not properties of the lightweight execution core.
func FromLite(request *qbtypes.QueryRangeRequest, result litequery.ExecutionResult) (*qbtypes.QueryRangeResponse, error) {
	if err := ValidateRequestRange(request); err != nil {
		return nil, err
	}
	response := &qbtypes.QueryRangeResponse{
		Type: request.RequestType,
		Meta: qbtypes.ExecStats{
			DurationMS:    uint64(result.Duration.Milliseconds()),
			StepIntervals: make(map[string]uint64, len(result.Queries)),
		},
	}
	for _, query := range result.Queries {
		if request.RequestType == qbtypes.RequestTypeTimeSeries {
			response.Meta.StepIntervals[query.Name] = uint64(stepForQuery(request, query.Name) / 1000)
		}
		var data any
		var err error
		switch request.RequestType {
		case qbtypes.RequestTypeTimeSeries:
			data, err = timeSeries(query)
			if err == nil && request.FormatOptions != nil && request.FormatOptions.FillGaps {
				err = fillTimeSeriesGaps(data.(*qbtypes.TimeSeriesData), int64(request.Start), int64(request.End), stepForQuery(request, query.Name))
			}
		case qbtypes.RequestTypeScalar:
			data, err = scalar(query)
		case qbtypes.RequestTypeRaw, qbtypes.RequestTypeTrace:
			data, err = raw(query)
		default:
			return nil, unsupported("requestType " + request.RequestType.StringValue())
		}
		if err != nil {
			return nil, err
		}
		response.Data.Results = append(response.Data.Results, data)
	}
	if len(result.Warnings) != 0 {
		code := qbtypes.QueryWarningCodeGeneric
		message := "Query completed with warnings"
		for _, query := range result.Queries {
			if query.Truncated {
				code = qbtypes.QueryWarningCodeResultLimit
				message = "Query results were truncated"
				break
			}
		}
		response.Warning = &qbtypes.QueryWarnData{Code: code, Message: message}
		for _, warning := range result.Warnings {
			response.Warning.Warnings = append(response.Warning.Warnings, qbtypes.QueryWarnDataAdditional{Message: warning})
		}
	}
	return response, nil
}

func fillTimeSeriesGaps(data *qbtypes.TimeSeriesData, startMS, endMS, stepMS int64) error {
	if stepMS <= 0 {
		return errors.NewInternalf(errors.CodeInternal, "time series query %q has no positive step for gap filling", data.QueryName)
	}
	if endMS < startMS {
		return errors.NewInternalf(errors.CodeInternal, "time series query %q has an invalid fill range", data.QueryName)
	}
	alignedStart := startMS - startMS%stepMS
	pointCount := (endMS-alignedStart-1)/stepMS + 1
	if pointCount > 11_000 {
		return errors.NewInvalidInputf(errors.CodeInvalidInput, "time series query %q exceeds gap-fill point budget", data.QueryName)
	}
	for _, aggregation := range data.Aggregations {
		if len(aggregation.Series) == 0 {
			aggregation.Series = append(aggregation.Series, &qbtypes.TimeSeries{})
		}
		for _, series := range aggregation.Series {
			existing := make(map[int64]*qbtypes.TimeSeriesValue, len(series.Values))
			for _, value := range series.Values {
				existing[value.Timestamp] = value
			}
			filled := make([]*qbtypes.TimeSeriesValue, 0, pointCount)
			for timestamp := alignedStart; timestamp < endMS; timestamp += stepMS {
				if value := existing[timestamp]; value != nil {
					filled = append(filled, value)
				} else {
					filled = append(filled, &qbtypes.TimeSeriesValue{Timestamp: timestamp, Value: 0})
				}
				if timestamp > math.MaxInt64-stepMS {
					break
				}
			}
			series.Values = filled
		}
	}
	return nil
}

func timeSeries(query litequery.QueryResult) (*qbtypes.TimeSeriesData, error) {
	timestampIndex, valueIndex := -1, -1
	labelIndexes := make([]int, 0)
	for index, column := range query.Columns {
		switch column.Name {
		case "timestamp":
			timestampIndex = index
		case "value":
			valueIndex = index
		default:
			labelIndexes = append(labelIndexes, index)
		}
	}
	if timestampIndex < 0 || valueIndex < 0 {
		return nil, errors.NewInternalf(errors.CodeInternal, "time series query %q has no timestamp/value columns", query.Name)
	}
	seriesByKey := make(map[string]*qbtypes.TimeSeries)
	seriesInOrder := make([]*qbtypes.TimeSeries, 0)
	for _, row := range query.Rows {
		if len(row) != len(query.Columns) {
			return nil, errors.NewInternalf(errors.CodeInternal, "time series query %q returned an invalid row", query.Name)
		}
		timestamp, ok := milliseconds(row[timestampIndex])
		if !ok {
			return nil, errors.NewInternalf(errors.CodeInternal, "time series query %q returned invalid timestamp %T", query.Name, row[timestampIndex])
		}
		value, ok := number(row[valueIndex])
		if !ok {
			return nil, errors.NewInternalf(errors.CodeInternal, "time series query %q returned invalid value %T", query.Name, row[valueIndex])
		}
		if math.IsNaN(value) || math.IsInf(value, 0) {
			continue
		}
		labels := make([]*qbtypes.Label, 0, len(labelIndexes))
		keyValues := make([]any, 0, len(labelIndexes))
		for _, index := range labelIndexes {
			field := query.Columns[index].Field
			if field == nil {
				continue
			}
			labels = append(labels, &qbtypes.Label{Key: fieldKey(*field), Value: row[index]})
			keyValues = append(keyValues, row[index])
		}
		key := responseAlignmentKey(keyValues)
		series := seriesByKey[key]
		if series == nil {
			series = &qbtypes.TimeSeries{Labels: labels}
			seriesByKey[key] = series
			seriesInOrder = append(seriesInOrder, series)
		}
		series.Values = append(series.Values, &qbtypes.TimeSeriesValue{Timestamp: timestamp, Value: value})
	}
	bucket := &qbtypes.AggregationBucket{Index: 0, Alias: "value"}
	for _, series := range seriesInOrder {
		sort.SliceStable(series.Values, func(left, right int) bool {
			return series.Values[left].Timestamp < series.Values[right].Timestamp
		})
		bucket.Series = append(bucket.Series, series)
	}
	return &qbtypes.TimeSeriesData{QueryName: query.Name, Aggregations: []*qbtypes.AggregationBucket{bucket}}, nil
}

func scalar(query litequery.QueryResult) (*qbtypes.ScalarData, error) {
	columns := make([]*qbtypes.ColumnDescriptor, len(query.Columns))
	for index, column := range query.Columns {
		kind := qbtypes.ColumnTypeGroup
		if column.Name == "value" {
			kind = qbtypes.ColumnTypeAggregation
		}
		key := telemetrytypes.TelemetryFieldKey{Name: column.Name}
		if column.Field != nil {
			key = fieldKey(*column.Field)
		}
		columns[index] = &qbtypes.ColumnDescriptor{TelemetryFieldKey: key, QueryName: query.Name, Type: kind}
	}
	data := make([][]any, len(query.Rows))
	for rowIndex, row := range query.Rows {
		if len(row) != len(query.Columns) {
			return nil, errors.NewInternalf(errors.CodeInternal, "scalar query %q returned an invalid row", query.Name)
		}
		data[rowIndex] = append([]any{}, row...)
		for columnIndex, value := range data[rowIndex] {
			if query.Columns[columnIndex].Name != "value" {
				continue
			}
			if number, ok := number(value); ok && (math.IsNaN(number) || math.IsInf(number, 0)) {
				data[rowIndex][columnIndex] = nil
			}
		}
	}
	return &qbtypes.ScalarData{QueryName: query.Name, Columns: columns, Data: data}, nil
}

func raw(query litequery.QueryResult) (*qbtypes.RawData, error) {
	rows := make([]*qbtypes.RawRow, 0, len(query.Rows))
	for _, row := range query.Rows {
		if len(row) != len(query.Columns) {
			return nil, errors.NewInternalf(errors.CodeInternal, "raw query %q returned an invalid row", query.Name)
		}
		result := &qbtypes.RawRow{Data: make(map[string]any, len(row))}
		for index, column := range query.Columns {
			key := column.Name
			if column.Field != nil {
				key = column.Field.Name
			}
			result.Data[key] = row[index]
			if column.Field != nil && column.Field.Name == "timestamp" {
				if timestamp, ok := rawTime(row[index]); ok {
					result.Timestamp = timestamp
				}
			}
		}
		rows = append(rows, result)
	}
	return &qbtypes.RawData{QueryName: query.Name, Rows: rows}, nil
}

func fieldKey(field litequery.FieldRef) telemetrytypes.TelemetryFieldKey {
	key := telemetrytypes.TelemetryFieldKey{Name: field.Name}
	switch field.Context {
	case litequery.FieldContextResource:
		key.FieldContext = telemetrytypes.FieldContextResource
	case litequery.FieldContextAttribute:
		key.FieldContext = telemetrytypes.FieldContextAttribute
	case litequery.FieldContextSpan:
		key.FieldContext = telemetrytypes.FieldContextSpan
	case litequery.FieldContextLog:
		key.FieldContext = telemetrytypes.FieldContextLog
	case litequery.FieldContextBody:
		key.FieldContext = telemetrytypes.FieldContextBody
	case litequery.FieldContextScope:
		key.FieldContext = telemetrytypes.FieldContextScope
	case litequery.FieldContextMetric:
		key.FieldContext = telemetrytypes.FieldContextMetric
	case litequery.FieldContextLabel:
		key.FieldContext = telemetrytypes.FieldContextMetric
	}
	switch field.Type {
	case litequery.ValueTypeString:
		key.FieldDataType = telemetrytypes.FieldDataTypeString
	case litequery.ValueTypeBool:
		key.FieldDataType = telemetrytypes.FieldDataTypeBool
	case litequery.ValueTypeNumber:
		key.FieldDataType = telemetrytypes.FieldDataTypeNumber
	}
	return key
}

func stepForQuery(request *qbtypes.QueryRangeRequest, name string) int64 {
	var enabledStep int64
	for _, envelope := range request.CompositeQuery.Queries {
		switch spec := envelope.Spec.(type) {
		case qbtypes.QueryBuilderQuery[qbtypes.LogAggregation]:
			if !spec.Disabled && spec.Name == name {
				return spec.StepInterval.Milliseconds()
			}
			if !spec.Disabled && enabledStep == 0 {
				enabledStep = spec.StepInterval.Milliseconds()
			}
		case qbtypes.QueryBuilderQuery[qbtypes.TraceAggregation]:
			if !spec.Disabled && spec.Name == name {
				return spec.StepInterval.Milliseconds()
			}
			if !spec.Disabled && enabledStep == 0 {
				enabledStep = spec.StepInterval.Milliseconds()
			}
		case qbtypes.QueryBuilderQuery[qbtypes.MetricAggregation]:
			if !spec.Disabled && spec.Name == name {
				return spec.StepInterval.Milliseconds()
			}
			if !spec.Disabled && enabledStep == 0 {
				enabledStep = spec.StepInterval.Milliseconds()
			}
		}
	}
	return enabledStep
}

func milliseconds(value any) (int64, bool) {
	switch current := value.(type) {
	case int8:
		return int64(current), true
	case int16:
		return int64(current), true
	case int32:
		return int64(current), true
	case int64:
		return current, true
	case int:
		return int64(current), true
	case uint8:
		return int64(current), true
	case uint16:
		return int64(current), true
	case uint32:
		return int64(current), true
	case uint64:
		if current <= math.MaxInt64 {
			return int64(current), true
		}
	case float64:
		if current >= math.MinInt64 && current <= math.MaxInt64 {
			return int64(current), true
		}
	case time.Time:
		return current.UnixMilli(), true
	}
	return 0, false
}

func rawTime(value any) (time.Time, bool) {
	if timestamp, ok := value.(time.Time); ok {
		return timestamp, true
	}
	switch numeric := value.(type) {
	case uint64:
		if numeric <= math.MaxInt64 {
			return time.Unix(0, int64(numeric)), true
		}
	case uint32:
		return time.Unix(0, int64(numeric)), true
	case int64:
		return time.Unix(0, numeric), true
	case int:
		return time.Unix(0, int64(numeric)), true
	case float64:
		if numeric >= 0 && numeric <= math.MaxInt64 {
			return time.Unix(0, int64(numeric)), true
		}
	}
	return time.Time{}, false
}

func responseAlignmentKey(values []any) string {
	var key strings.Builder
	for _, value := range values {
		encoded := fmt.Sprintf("%T:%v", value, value)
		key.WriteString(strconv.Itoa(len(encoded)))
		key.WriteByte(':')
		key.WriteString(encoded)
	}
	return key.String()
}

func number(value any) (float64, bool) {
	switch current := value.(type) {
	case float64:
		return current, true
	case float32:
		return float64(current), true
	case int:
		return float64(current), true
	case int8:
		return float64(current), true
	case int16:
		return float64(current), true
	case int64:
		return float64(current), true
	case int32:
		return float64(current), true
	case uint:
		return float64(current), true
	case uint8:
		return float64(current), true
	case uint16:
		return float64(current), true
	case uint64:
		return float64(current), true
	case uint32:
		return float64(current), true
	case string:
		parsed, err := strconv.ParseFloat(current, 64)
		return parsed, err == nil
	default:
		return 0, false
	}
}
