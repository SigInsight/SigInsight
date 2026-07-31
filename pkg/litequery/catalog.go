package litequery

import (
	"strings"
)

// Catalog resolves semantic fields to trusted ClickHouse expressions. SQL
// templates are owned by this package; request values are represented only by
// positional arguments.
type Catalog interface {
	Table(Signal) (string, error)
	Resolve(Signal, FieldRef) (ResolvedField, error)
	MetricSource(Signal) (MetricSource, error)
}

// MetricSource describes the trusted physical tables used by metric compilers.
// Metrics keep series identity separately from samples; Meter stores labels with
// the points and therefore leaves SeriesTable empty.
type MetricSource struct {
	PointsTable string
	SeriesTable string
}

type ResolvedField struct {
	SQL               string
	Args              []any
	ExistsSQL         string
	ExistsArgs        []any
	RequiresExistence bool
}

type DefaultCatalog struct{}

func (DefaultCatalog) Table(signal Signal) (string, error) {
	switch signal {
	case SignalLogs:
		return "signoz_logs.logs_v2", nil
	case SignalTraces:
		return "signoz_traces.signoz_index_v3", nil
	case SignalMetrics:
		return "signoz_metrics.samples_v4", nil
	case SignalMeter:
		return "signoz_meter.samples", nil
	default:
		return "", newError(ErrorUnsupported, "signal", "no ClickHouse table is configured for %q", signal)
	}
}

func (DefaultCatalog) MetricSource(signal Signal) (MetricSource, error) {
	switch signal {
	case SignalMetrics:
		return MetricSource{
			PointsTable: "signoz_metrics.samples_v4",
			SeriesTable: "signoz_metrics.time_series_v4",
		}, nil
	case SignalMeter:
		return MetricSource{PointsTable: "signoz_meter.samples"}, nil
	default:
		return MetricSource{}, newError(ErrorUnsupported, "signal", "no metric source is configured for %q", signal)
	}
}

func (DefaultCatalog) Resolve(signal Signal, field FieldRef) (ResolvedField, error) {
	if err := validateField(field, "field"); err != nil {
		return ResolvedField{}, err
	}
	switch signal {
	case SignalLogs:
		return resolveLogField(field)
	case SignalTraces:
		return resolveTraceField(field)
	default:
		return ResolvedField{}, newError(ErrorUnsupported, "signal", "no field catalog is configured for %q", signal)
	}
}

func resolveLogField(field FieldRef) (ResolvedField, error) {
	if field.Context == FieldContextResource {
		return resolveMapField("resources_string", field)
	}
	if field.Context == FieldContextAttribute {
		return resolveTypedMapField("attributes", field)
	}
	if field.Context == FieldContextScope {
		switch field.Name {
		case "name", "scope.name", "scope_name":
			return staticField("scope_name", field.Type), nil
		case "version", "scope.version", "scope_version":
			return staticField("scope_version", field.Type), nil
		default:
			return resolveMapField("scope_string", field)
		}
	}
	if field.Context == FieldContextBody {
		if field.Type != ValueTypeString {
			return ResolvedField{}, newError(ErrorUnsupported, "field.type", "body JSON paths currently support string values only")
		}
		if field.Name == "body" {
			return staticField("body", field.Type), nil
		}
		path := field.Name
		if !strings.HasPrefix(path, "$") {
			path = "$." + path
		}
		return ResolvedField{
			SQL:               "JSON_VALUE(body, ?)",
			Args:              []any{path},
			ExistsSQL:         "JSON_VALUE(body, ?) IS NOT NULL",
			ExistsArgs:        []any{path},
			RequiresExistence: true,
		}, nil
	}
	if field.Context != FieldContextLog {
		return ResolvedField{}, newError(ErrorUnsupported, "field.context", "log field context %q is unsupported", field.Context)
	}
	if valueType, ok := logIntrinsicFields[field.Name]; ok {
		if valueType != field.Type {
			return ResolvedField{}, newError(ErrorInvalidRequest, "field.type", "field %q has type %q", field.Name, valueType)
		}
		return staticField(field.Name, field.Type), nil
	}
	return ResolvedField{}, newError(ErrorUnsupported, "field.name", "log field %q is not in the schema catalog", field.Name)
}

func resolveTraceField(field FieldRef) (ResolvedField, error) {
	if field.Context == FieldContextResource {
		return resolveMapField("resources_string", field)
	}
	if field.Context == FieldContextAttribute {
		return resolveTypedMapField("attributes", field)
	}
	if field.Context != FieldContextSpan {
		return ResolvedField{}, newError(ErrorUnsupported, "field.context", "trace field context %q is unsupported", field.Context)
	}
	if valueType, ok := traceIntrinsicFields[field.Name]; ok {
		if valueType != field.Type {
			return ResolvedField{}, newError(ErrorInvalidRequest, "field.type", "field %q has type %q", field.Name, valueType)
		}
		return staticField(field.Name, field.Type), nil
	}
	return ResolvedField{}, newError(ErrorUnsupported, "field.name", "trace field %q is not in the schema catalog", field.Name)
}

func resolveTypedMapField(prefix string, field FieldRef) (ResolvedField, error) {
	switch field.Type {
	case ValueTypeString:
		return resolveMapField(prefix+"_string", field)
	case ValueTypeNumber:
		return resolveMapField(prefix+"_number", field)
	case ValueTypeBool:
		return resolveMapField(prefix+"_bool", field)
	default:
		return ResolvedField{}, newError(ErrorInvalidRequest, "field.type", "unsupported map value type %q", field.Type)
	}
}

func resolveMapField(column string, field FieldRef) (ResolvedField, error) {
	return ResolvedField{
		SQL:               column + "[?]",
		Args:              []any{field.Name},
		ExistsSQL:         "mapContains(" + column + ", ?)",
		ExistsArgs:        []any{field.Name},
		RequiresExistence: true,
	}, nil
}

func staticField(column string, fieldType ValueType) ResolvedField {
	exists := column + " != 0"
	if fieldType == ValueTypeString {
		exists = column + " != ''"
	}
	return ResolvedField{SQL: column, ExistsSQL: exists}
}

var logIntrinsicFields = map[string]ValueType{
	"timestamp":          ValueTypeNumber,
	"observed_timestamp": ValueTypeNumber,
	"id":                 ValueTypeString,
	"trace_id":           ValueTypeString,
	"span_id":            ValueTypeString,
	"trace_flags":        ValueTypeNumber,
	"severity_text":      ValueTypeString,
	"severity_number":    ValueTypeNumber,
	"body":               ValueTypeString,
	"scope_name":         ValueTypeString,
	"scope_version":      ValueTypeString,
}

var traceIntrinsicFields = map[string]ValueType{
	"timestamp":            ValueTypeNumber,
	"trace_id":             ValueTypeString,
	"span_id":              ValueTypeString,
	"trace_state":          ValueTypeString,
	"parent_span_id":       ValueTypeString,
	"flags":                ValueTypeNumber,
	"name":                 ValueTypeString,
	"kind":                 ValueTypeNumber,
	"kind_string":          ValueTypeString,
	"duration_nano":        ValueTypeNumber,
	"status_code":          ValueTypeNumber,
	"status_message":       ValueTypeString,
	"status_code_string":   ValueTypeString,
	"response_status_code": ValueTypeString,
	"external_http_url":    ValueTypeString,
	"http_url":             ValueTypeString,
	"external_http_method": ValueTypeString,
	"http_method":          ValueTypeString,
	"http_host":            ValueTypeString,
	"db_name":              ValueTypeString,
	"db_operation":         ValueTypeString,
	"has_error":            ValueTypeBool,
	"is_remote":            ValueTypeString,
}
