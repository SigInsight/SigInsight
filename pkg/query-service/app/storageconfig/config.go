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
	UsageExplorerTable     string
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
			IndexTable:              "span_index_v3",
			SummaryTable:            "trace_summary",
			ErrorTable:              "error_index_v2",
			DependencyGraphTable:    "dependency_graph_minutes_v2",
			TopLevelOperationsTable: "top_level_operations",
		},
		Retention: Retention{
			TraceDB:                "siginsight_traces",
			TraceTable:             "span_index_v3",
			TraceLocalTable:        "span_index_v3",
			TraceResourceTable:     "traces_v3_resource",
			ErrorTable:             "error_index_v2",
			UsageExplorerTable:     "usage_explorer",
			DependencyGraphTable:   "dependency_graph_minutes_v2",
			TraceSummaryTable:      "trace_summary",
			SpanAttributeKeysTable: "span_attributes_keys",
			LogsDB:                 "siginsight_logs",
			LogsTable:              "logs_v2",
			LogsLocalTable:         "logs_v2",
			LogsResourceTable:      "logs_v2_resource",
			LogsResourceLocalTable: "logs_v2_resource",
			LogsAttributeKeysTable: "logs_attribute_keys",
			LogsResourceKeysTable:  "logs_resource_keys",
		},
	}
}
