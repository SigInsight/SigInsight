package transition

var legacyTraceFieldNames = map[string]string{
	"traceID":            "trace_id",
	"spanID":             "span_id",
	"parentSpanID":       "parent_span_id",
	"spanKind":           "kind_string",
	"durationNano":       "duration_nano",
	"statusCode":         "status_code",
	"statusMessage":      "status_message",
	"statusCodeString":   "status_code_string",
	"responseStatusCode": "response_status_code",
	"externalHttpUrl":    "external_http_url",
	"httpUrl":            "http_url",
	"externalHttpMethod": "external_http_method",
	"httpMethod":         "http_method",
	"httpHost":           "http_host",
	"dbName":             "db_name",
	"dbOperation":        "db_operation",
	"hasError":           "has_error",
	"isRemote":           "is_remote",
	"serviceName":        "service.name",
	"httpRoute":          "http.route",
	"msgSystem":          "messaging.system",
	"msgOperation":       "messaging.operation",
	"dbSystem":           "db.system",
	"rpcSystem":          "rpc.system",
	"rpcService":         "rpc.service",
	"rpcMethod":          "rpc.method",
	"peerService":        "peer.service",
}

func canonicalTraceFieldName(name string) string {
	if canonical, ok := legacyTraceFieldNames[name]; ok {
		return canonical
	}
	return name
}
