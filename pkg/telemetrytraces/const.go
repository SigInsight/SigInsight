package telemetrytraces

import "github.com/SigNoz/signoz/pkg/types/telemetrytypes"

var (
	IntrinsicFields = map[string]telemetrytypes.TelemetryFieldKey{
		"trace_id": {
			Name:          "trace_id",
			Signal:        telemetrytypes.SignalTraces,
			FieldContext:  telemetrytypes.FieldContextSpan,
			FieldDataType: telemetrytypes.FieldDataTypeString,
		},
		"span_id": {
			Name:          "span_id",
			Signal:        telemetrytypes.SignalTraces,
			FieldContext:  telemetrytypes.FieldContextSpan,
			FieldDataType: telemetrytypes.FieldDataTypeString,
		},
		"trace_state": {
			Name:          "trace_state",
			Signal:        telemetrytypes.SignalTraces,
			FieldContext:  telemetrytypes.FieldContextSpan,
			FieldDataType: telemetrytypes.FieldDataTypeString,
		},
		"parent_span_id": {
			Name:          "parent_span_id",
			Signal:        telemetrytypes.SignalTraces,
			FieldContext:  telemetrytypes.FieldContextSpan,
			FieldDataType: telemetrytypes.FieldDataTypeString,
		},
		"flags": {
			Name:          "flags",
			Signal:        telemetrytypes.SignalTraces,
			FieldContext:  telemetrytypes.FieldContextSpan,
			FieldDataType: telemetrytypes.FieldDataTypeNumber,
		},
		"name": {
			Name:          "name",
			Description:   "Name of the span",
			Signal:        telemetrytypes.SignalTraces,
			FieldContext:  telemetrytypes.FieldContextSpan,
			FieldDataType: telemetrytypes.FieldDataTypeString,
		},
		"kind_string": {
			Name:          "kind_string",
			Description:   "Span kind enum (string). Known values are ['Client', 'Server', 'Internal', 'Producer', 'Consumer']. Learn more [here](https://opentelemetry.io/docs/concepts/signals/traces/#span-kind)",
			Signal:        telemetrytypes.SignalTraces,
			FieldContext:  telemetrytypes.FieldContextSpan,
			FieldDataType: telemetrytypes.FieldDataTypeString,
		},
		"duration_nano": {
			Name:          "duration_nano",
			Description:   "Span duration",
			Unit:          "ns",
			Signal:        telemetrytypes.SignalTraces,
			FieldContext:  telemetrytypes.FieldContextSpan,
			FieldDataType: telemetrytypes.FieldDataTypeNumber,
		},
		"status_code": {
			Name:          "status_code",
			Description:   "Span status code enum (number). Use `status_code_string` instead. Learn more [here](https://opentelemetry.io/docs/concepts/signals/traces/#span-status)",
			Signal:        telemetrytypes.SignalTraces,
			FieldContext:  telemetrytypes.FieldContextSpan,
			FieldDataType: telemetrytypes.FieldDataTypeNumber,
		},
		"status_message": {
			Name:          "status_message",
			Description:   "Span status message. Learn more [here](https://opentelemetry.io/docs/concepts/signals/traces/#span-status)",
			Signal:        telemetrytypes.SignalTraces,
			FieldContext:  telemetrytypes.FieldContextSpan,
			FieldDataType: telemetrytypes.FieldDataTypeString,
		},
		"status_code_string": {
			Name:          "status_code_string",
			Description:   "Span status code enum (string).  Learn more [here](https://opentelemetry.io/docs/concepts/signals/traces/#span-status)",
			Signal:        telemetrytypes.SignalTraces,
			FieldContext:  telemetrytypes.FieldContextSpan,
			FieldDataType: telemetrytypes.FieldDataTypeString,
		},
		"timestamp": {
			Name:          "timestamp",
			Signal:        telemetrytypes.SignalTraces,
			FieldContext:  telemetrytypes.FieldContextSpan,
			FieldDataType: telemetrytypes.FieldDataTypeNumber,
		},
	}

	CalculatedFields = map[string]telemetrytypes.TelemetryFieldKey{
		"http.route": {
			Name:          "http.route",
			Description:   "The HTTP route of the request.",
			Signal:        telemetrytypes.SignalTraces,
			FieldContext:  telemetrytypes.FieldContextSpan,
			FieldDataType: telemetrytypes.FieldDataTypeString,
		},
		"rpc.method": {
			Name:          "rpc.method",
			Description:   "The RPC method of the request.",
			Signal:        telemetrytypes.SignalTraces,
			FieldContext:  telemetrytypes.FieldContextSpan,
			FieldDataType: telemetrytypes.FieldDataTypeString,
		},
		"response_status_code": {
			Name:          "response_status_code",
			Description:   "Derived response status code from the HTTP/RPC status code attributes. Learn more [here](https://signoz.io/docs/traces-management/guides/derived-fields-spans/#response_status_code)",
			Signal:        telemetrytypes.SignalTraces,
			FieldContext:  telemetrytypes.FieldContextSpan,
			FieldDataType: telemetrytypes.FieldDataTypeString,
		},
		"external_http_url": {
			Name:          "external_http_url",
			Description:   "The hostname of the external HTTP URL. Learn more [here](https://signoz.io/docs/traces-management/guides/derived-fields-spans/#external_http_url)",
			Signal:        telemetrytypes.SignalTraces,
			FieldContext:  telemetrytypes.FieldContextSpan,
			FieldDataType: telemetrytypes.FieldDataTypeString,
		},
		"http_url": {
			Name:          "http_url",
			Description:   "HTTP URL of the request. Learn more [here](https://signoz.io/docs/traces-management/guides/derived-fields-spans/#http_url)",
			Signal:        telemetrytypes.SignalTraces,
			FieldContext:  telemetrytypes.FieldContextSpan,
			FieldDataType: telemetrytypes.FieldDataTypeString,
		},
		"external_http_method": {
			Name:          "external_http_method",
			Description:   "HTTP request method of client spans. Learn more [here](https://signoz.io/docs/traces-management/guides/derived-fields-spans/#external_http_method)",
			Signal:        telemetrytypes.SignalTraces,
			FieldContext:  telemetrytypes.FieldContextSpan,
			FieldDataType: telemetrytypes.FieldDataTypeString,
		},
		"http_method": {
			Name:          "http_method",
			Description:   "The HTTP request method. Learn more [here](https://signoz.io/docs/traces-management/guides/derived-fields-spans/#http_method)",
			Signal:        telemetrytypes.SignalTraces,
			FieldContext:  telemetrytypes.FieldContextSpan,
			FieldDataType: telemetrytypes.FieldDataTypeString,
		},
		"http_host": {
			Name:          "http_host",
			Description:   "The HTTP host or server address. Learn more [here](https://signoz.io/docs/traces-management/guides/derived-fields-spans/#http_host)",
			Signal:        telemetrytypes.SignalTraces,
			FieldContext:  telemetrytypes.FieldContextSpan,
			FieldDataType: telemetrytypes.FieldDataTypeString,
		},
		"db_name": {
			Name:          "db_name",
			Description:   "The database name or namespace. Learn more [here](https://signoz.io/docs/traces-management/guides/derived-fields-spans/#db_name)",
			Signal:        telemetrytypes.SignalTraces,
			FieldContext:  telemetrytypes.FieldContextSpan,
			FieldDataType: telemetrytypes.FieldDataTypeString,
		},
		"db_operation": {
			Name:          "db_operation",
			Description:   "The database operation being performed. Learn more [here](https://signoz.io/docs/traces-management/guides/derived-fields-spans/#db_operation)",
			Signal:        telemetrytypes.SignalTraces,
			FieldContext:  telemetrytypes.FieldContextSpan,
			FieldDataType: telemetrytypes.FieldDataTypeString,
		},
		"has_error": {
			Name:          "has_error",
			Description:   "Whether the span has an error. Learn more [here](https://signoz.io/docs/traces-management/guides/derived-fields-spans/#has_error)",
			Signal:        telemetrytypes.SignalTraces,
			FieldContext:  telemetrytypes.FieldContextSpan,
			FieldDataType: telemetrytypes.FieldDataTypeBool,
		},
		"is_remote": {
			Name:          "is_remote",
			Description:   "Whether the span is remote. Learn more [here](https://signoz.io/docs/traces-management/guides/derived-fields-spans/#is_remote)",
			Signal:        telemetrytypes.SignalTraces,
			FieldContext:  telemetrytypes.FieldContextSpan,
			FieldDataType: telemetrytypes.FieldDataTypeString,
		},
	}

	SpanSearchScopeRoot       = "isroot"
	SpanSearchScopeEntryPoint = "isentrypoint"

	DefaultFields = map[string]telemetrytypes.TelemetryFieldKey{
		"timestamp": {
			Name:          "timestamp",
			Signal:        telemetrytypes.SignalTraces,
			FieldContext:  telemetrytypes.FieldContextSpan,
			FieldDataType: telemetrytypes.FieldDataTypeNumber,
		},
		"span_id": {
			Name:          "span_id",
			Signal:        telemetrytypes.SignalTraces,
			FieldContext:  telemetrytypes.FieldContextSpan,
			FieldDataType: telemetrytypes.FieldDataTypeString,
		},
		"trace_id": {
			Name:          "trace_id",
			Signal:        telemetrytypes.SignalTraces,
			FieldContext:  telemetrytypes.FieldContextSpan,
			FieldDataType: telemetrytypes.FieldDataTypeString,
		},
		"name": {
			Name:          "name",
			Signal:        telemetrytypes.SignalTraces,
			FieldContext:  telemetrytypes.FieldContextSpan,
			FieldDataType: telemetrytypes.FieldDataTypeString,
		},
		"service.name": {
			Name:          "service.name",
			Signal:        telemetrytypes.SignalTraces,
			FieldContext:  telemetrytypes.FieldContextResource,
			FieldDataType: telemetrytypes.FieldDataTypeString,
			Materialized:  true,
		},
		"duration_nano": {
			Name:          "duration_nano",
			Signal:        telemetrytypes.SignalTraces,
			FieldContext:  telemetrytypes.FieldContextSpan,
			FieldDataType: telemetrytypes.FieldDataTypeNumber,
		},
		"response_status_code": {
			Name:          "response_status_code",
			Signal:        telemetrytypes.SignalTraces,
			FieldContext:  telemetrytypes.FieldContextSpan,
			FieldDataType: telemetrytypes.FieldDataTypeString,
		},
	}
)
