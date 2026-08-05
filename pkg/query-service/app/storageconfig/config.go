// Package storageconfig defines the ClickHouse tables used by query-service
// stores. It is intentionally limited to active storage contracts.
package storageconfig

type Trace struct {
	Database                string
	IndexTable              string
	SummaryTable            string
	ErrorTable              string
	DependencyGraphTable    string
	TopLevelOperationsTable string
}

type Retention struct {
	TraceDB                string
	TraceTable             string
	TraceLocalTable        string
	TraceResourceTable     string
	ErrorTable             string
	DependencyGraphTable   string
	TraceSummaryTable      string
	SpanAttributeKeysTable string
	LogsDB                 string
	LogsTable              string
	LogsLocalTable         string
	LogsResourceTable      string
	LogsResourceLocalTable string
	LogsAttributeKeysTable string
	LogsResourceKeysTable  string
}

type Config struct {
	Trace     Trace
	Retention Retention
}

func Default() Config {
	return Config{
		Trace: Trace{
			Database:                "siginsight_traces",
			IndexTable:              "spans",
			SummaryTable:            "trace_summary",
			ErrorTable:              "exceptions",
			DependencyGraphTable:    "service_edges",
			TopLevelOperationsTable: "operations",
		},
		Retention: Retention{
			TraceDB:                "siginsight_traces",
			TraceTable:             "spans",
			TraceLocalTable:        "spans",
			TraceResourceTable:     "resource_sets",
			ErrorTable:             "exceptions",
			DependencyGraphTable:   "service_edges",
			TraceSummaryTable:      "trace_summary",
			SpanAttributeKeysTable: "span_attributes_keys",
			LogsDB:                 "siginsight_logs",
			LogsTable:              "logs",
			LogsLocalTable:         "logs",
			LogsResourceTable:      "resource_sets",
			LogsResourceLocalTable: "resource_sets",
			LogsAttributeKeysTable: "logs_attribute_keys",
			LogsResourceKeysTable:  "logs_resource_keys",
		},
	}
}
