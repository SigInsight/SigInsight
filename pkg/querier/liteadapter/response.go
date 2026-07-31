package liteadapter

import (
	"fmt"
	"math"
	"strconv"
	"time"

	"github.com/SigNoz/signoz/pkg/litequery"
	qbtypes "github.com/SigNoz/signoz/pkg/types/querybuildertypes/querybuildertypesv5"
	"github.com/SigNoz/signoz/pkg/types/telemetrytypes"
)

// FromLite restores the stable V5 response shape used by the frontend. The
// adapter owns labels and column descriptors because they are transport
// concerns, not properties of the lightweight execution core.
func FromLite(request *qbtypes.QueryRangeRequest, result litequery.ExecutionResult) (*qbtypes.QueryRangeResponse, error) {
	if request == nil {
		return nil, fmt.Errorf("V5 request is required")
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
		response.Warning = &qbtypes.QueryWarnData{Message: "Encountered warnings"}
		for _, warning := range result.Warnings {
			response.Warning.Warnings = append(response.Warning.Warnings, qbtypes.QueryWarnDataAdditional{Message: warning})
		}
	}
	return response, nil
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
		return nil, fmt.Errorf("time series query %q has no timestamp/value columns", query.Name)
	}
	seriesByKey := make(map[string]*qbtypes.TimeSeries)
	for _, row := range query.Rows {
		if len(row) != len(query.Columns) {
			return nil, fmt.Errorf("time series query %q returned an invalid row", query.Name)
		}
		timestamp, ok := milliseconds(row[timestampIndex])
		if !ok {
			return nil, fmt.Errorf("time series query %q returned invalid timestamp %T", query.Name, row[timestampIndex])
		}
		value, ok := number(row[valueIndex])
		if !ok || math.IsNaN(value) || math.IsInf(value, 0) {
			continue
		}
		labels := make([]*qbtypes.Label, 0, len(labelIndexes))
		key := ""
		for _, index := range labelIndexes {
			field := query.Columns[index].Field
			if field == nil {
				continue
			}
			labels = append(labels, &qbtypes.Label{Key: fieldKey(*field), Value: row[index]})
			key += query.Columns[index].Name + "=" + fmt.Sprint(row[index]) + ","
		}
		series := seriesByKey[key]
		if series == nil {
			series = &qbtypes.TimeSeries{Labels: labels}
			seriesByKey[key] = series
		}
		series.Values = append(series.Values, &qbtypes.TimeSeriesValue{Timestamp: timestamp, Value: value})
	}
	bucket := &qbtypes.AggregationBucket{Index: 0, Alias: "value"}
	for _, series := range seriesByKey {
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
	return &qbtypes.ScalarData{QueryName: query.Name, Columns: columns, Data: query.Rows}, nil
}

func raw(query litequery.QueryResult) (*qbtypes.RawData, error) {
	rows := make([]*qbtypes.RawRow, 0, len(query.Rows))
	for _, row := range query.Rows {
		if len(row) != len(query.Columns) {
			return nil, fmt.Errorf("raw query %q returned an invalid row", query.Name)
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
	for _, envelope := range request.CompositeQuery.Queries {
		switch spec := envelope.Spec.(type) {
		case qbtypes.QueryBuilderQuery[qbtypes.LogAggregation]:
			if spec.Name == name {
				return spec.StepInterval.Milliseconds()
			}
		case qbtypes.QueryBuilderQuery[qbtypes.TraceAggregation]:
			if spec.Name == name {
				return spec.StepInterval.Milliseconds()
			}
		case qbtypes.QueryBuilderQuery[qbtypes.MetricAggregation]:
			if spec.Name == name {
				return spec.StepInterval.Milliseconds()
			}
		}
	}
	return request.StepIntervalForQuery(name)
}

func milliseconds(value any) (int64, bool) {
	switch current := value.(type) {
	case int64:
		return current, true
	case int:
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
	if numeric, ok := number(value); ok {
		return time.Unix(0, int64(numeric)), true
	}
	return time.Time{}, false
}

func number(value any) (float64, bool) {
	switch current := value.(type) {
	case float64:
		return current, true
	case float32:
		return float64(current), true
	case int:
		return float64(current), true
	case int64:
		return float64(current), true
	case int32:
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
