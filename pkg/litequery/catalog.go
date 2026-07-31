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

// materializedFieldKey identifies a semantic field with a Collector-owned
// physical fast path. The manifest is intentionally static: a schema migration
// must update it and its cross-repository verification before a new column can
// affect generated SQL.
type materializedFieldKey struct {
	Signal  Signal
	Context FieldContext
	Name    string
	Type    ValueType
}

type materializedField struct {
	Column       string
	ExistsColumn string
}

var defaultMaterializedFields = map[materializedFieldKey]materializedField{
	{Signal: SignalTraces, Context: FieldContextResource, Name: "service.name", Type: ValueTypeString}: {
		Column: "resource_string_service$$name", ExistsColumn: "resource_string_service$$name_exists",
	},
	{Signal: SignalTraces, Context: FieldContextAttribute, Name: "http.route", Type: ValueTypeString}: {
		Column: "attribute_string_http$$route", ExistsColumn: "attribute_string_http$$route_exists",
	},
	{Signal: SignalTraces, Context: FieldContextAttribute, Name: "messaging.system", Type: ValueTypeString}: {
		Column: "attribute_string_messaging$$system", ExistsColumn: "attribute_string_messaging$$system_exists",
	},
	{Signal: SignalTraces, Context: FieldContextAttribute, Name: "messaging.operation", Type: ValueTypeString}: {
		Column: "attribute_string_messaging$$operation", ExistsColumn: "attribute_string_messaging$$operation_exists",
	},
	{Signal: SignalTraces, Context: FieldContextAttribute, Name: "db.system", Type: ValueTypeString}: {
		Column: "attribute_string_db$$system", ExistsColumn: "attribute_string_db$$system_exists",
	},
	{Signal: SignalTraces, Context: FieldContextAttribute, Name: "rpc.system", Type: ValueTypeString}: {
		Column: "attribute_string_rpc$$system", ExistsColumn: "attribute_string_rpc$$system_exists",
	},
	{Signal: SignalTraces, Context: FieldContextAttribute, Name: "rpc.service", Type: ValueTypeString}: {
		Column: "attribute_string_rpc$$service", ExistsColumn: "attribute_string_rpc$$service_exists",
	},
	{Signal: SignalTraces, Context: FieldContextAttribute, Name: "rpc.method", Type: ValueTypeString}: {
		Column: "attribute_string_rpc$$method", ExistsColumn: "attribute_string_rpc$$method_exists",
	},
	{Signal: SignalTraces, Context: FieldContextAttribute, Name: "peer.service", Type: ValueTypeString}: {
		Column: "attribute_string_peer$$service", ExistsColumn: "attribute_string_peer$$service_exists",
	},
}

func (DefaultCatalog) Table(signal Signal) (string, error) {
	switch signal {
	case SignalLogs:
		return "siginsight_logs.logs_v2", nil
	case SignalTraces:
		return "siginsight_traces.span_index_v3", nil
	case SignalMetrics:
		return "siginsight_metrics.samples_v4", nil
	case SignalMeter:
		return "siginsight_meter.samples", nil
	default:
		return "", newError(ErrorUnsupported, "signal", "no ClickHouse table is configured for %q", signal)
	}
}

func (DefaultCatalog) MetricSource(signal Signal) (MetricSource, error) {
	switch signal {
	case SignalMetrics:
		return MetricSource{
			PointsTable: "siginsight_metrics.samples_v4",
			SeriesTable: "siginsight_metrics.time_series_v4",
		}, nil
	case SignalMeter:
		return MetricSource{PointsTable: "siginsight_meter.samples"}, nil
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
	if materialized, ok := defaultMaterializedFields[materializedFieldKey{
		Signal: SignalTraces, Context: field.Context, Name: field.Name, Type: field.Type,
	}]; ok {
		return ResolvedField{
			SQL:               quoteIdentifier(materialized.Column),
			ExistsSQL:         quoteIdentifier(materialized.ExistsColumn),
			RequiresExistence: true,
		}, nil
	}
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

func quoteIdentifier(identifier string) string {
	return "`" + strings.ReplaceAll(identifier, "`", "``") + "`"
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

// IntrinsicFieldType returns the schema type for fields whose type is fixed by
// the Logs or Traces table contract. V5 callers may omit fieldDataType for
// these fields, so the adapter uses this catalog-owned mapping before applying
// its generic fallback for dynamic fields.
func IntrinsicFieldType(signal Signal, context FieldContext, name string) (ValueType, bool) {
	switch signal {
	case SignalLogs:
		if context == FieldContextLog {
			valueType, ok := logIntrinsicFields[name]
			return valueType, ok
		}
	case SignalTraces:
		if context == FieldContextSpan {
			valueType, ok := traceIntrinsicFields[name]
			return valueType, ok
		}
	}
	return "", false
}
