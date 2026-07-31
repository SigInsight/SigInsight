package litequery

import (
	"errors"
	"reflect"
	"strings"
	"testing"
)

func TestDefaultCatalogResolvesSemanticFields(t *testing.T) {
	tests := []struct {
		name   string
		signal Signal
		field  FieldRef
		want   ResolvedField
	}{
		{
			name: "log resource string", signal: SignalLogs,
			field: FieldRef{Name: "service.name", Context: FieldContextResource, Type: ValueTypeString},
			want:  ResolvedField{SQL: "resources_string[?]", Args: []any{"service.name"}, ExistsSQL: "mapContains(resources_string, ?)", ExistsArgs: []any{"service.name"}, RequiresExistence: true},
		},
		{
			name: "trace number attribute", signal: SignalTraces,
			field: FieldRef{Name: "http.request.size", Context: FieldContextAttribute, Type: ValueTypeNumber},
			want:  ResolvedField{SQL: "attributes_number[?]", Args: []any{"http.request.size"}, ExistsSQL: "mapContains(attributes_number, ?)", ExistsArgs: []any{"http.request.size"}, RequiresExistence: true},
		},
		{
			name: "trace materialized resource string", signal: SignalTraces,
			field: FieldRef{Name: "service.name", Context: FieldContextResource, Type: ValueTypeString},
			want:  ResolvedField{SQL: "`resource_string_service$$name`", ExistsSQL: "`resource_string_service$$name_exists`", RequiresExistence: true},
		},
		{
			name: "trace materialized attribute string", signal: SignalTraces,
			field: FieldRef{Name: "http.route", Context: FieldContextAttribute, Type: ValueTypeString},
			want:  ResolvedField{SQL: "`attribute_string_http$$route`", ExistsSQL: "`attribute_string_http$$route_exists`", RequiresExistence: true},
		},
		{
			name: "log json body path", signal: SignalLogs,
			field: FieldRef{Name: "request.id", Context: FieldContextBody, Type: ValueTypeString},
			want:  ResolvedField{SQL: "JSON_VALUE(body, ?)", Args: []any{"$.request.id"}, ExistsSQL: "JSON_VALUE(body, ?) IS NOT NULL", ExistsArgs: []any{"$.request.id"}, RequiresExistence: true},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got, err := (DefaultCatalog{}).Resolve(test.signal, test.field)
			if err != nil {
				t.Fatalf("Resolve() error = %v", err)
			}
			if !reflect.DeepEqual(got, test.want) {
				t.Fatalf("Resolve() = %#v, want %#v", got, test.want)
			}
		})
	}
}

func TestCompilerCompilesLogRawWithJSONAndMapParameters(t *testing.T) {
	statement := compileOne(t, Request{
		Range: TimeRange{StartMS: 1_000, EndMS: 2_000}, ResultType: ResultRaw,
		Queries: []Query{LogQuery{
			Common: CommonQuery{
				Name: "logs",
				Select: []FieldRef{
					{Name: "timestamp", Context: FieldContextLog, Type: ValueTypeNumber},
					{Name: "request.id", Context: FieldContextBody, Type: ValueTypeString},
				},
				Filter: Predicate{Field: FieldRef{Name: "service.name", Context: FieldContextResource, Type: ValueTypeString}, Op: FilterEqual, Value: Value{Kind: ValueString, String: "api"}},
				Limit:  25,
			},
			Aggregation: LogAggregateCount,
		}},
	})
	wantSQL := "SELECT timestamp AS field_0, JSON_VALUE(body, ?) AS field_1 FROM signoz_logs.logs_v2 WHERE (signoz_logs.logs_v2.timestamp >= toUInt64(?) AND signoz_logs.logs_v2.timestamp < toUInt64(?)) AND ((mapContains(resources_string, ?)) AND (resources_string[?] = ?)) ORDER BY timestamp DESC, id DESC LIMIT ?"
	wantArgs := []any{"$.request.id", int64(1_000_000_000), int64(2_000_000_000), "service.name", "service.name", "api", uint32(25)}
	assertStatement(t, statement, wantSQL, wantArgs)
}

func TestCompilerCompilesTraceTimeSeriesWithCorrectArgumentOrder(t *testing.T) {
	statement := compileOne(t, Request{
		Range: TimeRange{StartMS: 1_000, EndMS: 61_000}, ResultType: ResultTimeSeries, StepMS: 1_000,
		Queries: []Query{TraceQuery{
			Common:      CommonQuery{Name: "latency", GroupBy: []FieldRef{{Name: "service.name", Context: FieldContextResource, Type: ValueTypeString}}},
			Aggregation: TraceAggregateDurationP95,
		}},
	})
	wantSQL := "SELECT intDiv(toUnixTimestamp64Milli(signoz_index_v3.timestamp), ?) * ? AS timestamp, `resource_string_service$$name` AS group_0, quantile(0.95)(duration_nano) AS value FROM signoz_traces.signoz_index_v3 WHERE signoz_traces.signoz_index_v3.timestamp >= fromUnixTimestamp64Milli(?) AND signoz_traces.signoz_index_v3.timestamp < fromUnixTimestamp64Milli(?) GROUP BY intDiv(toUnixTimestamp64Milli(signoz_index_v3.timestamp), ?) * ?, `resource_string_service$$name` ORDER BY timestamp ASC"
	wantArgs := []any{int64(1_000), int64(1_000), int64(1_000), int64(61_000), int64(1_000), int64(1_000)}
	assertStatement(t, statement, wantSQL, wantArgs)
}

func TestCompilerCompilesTraceMaterializedFilterAndGroupBy(t *testing.T) {
	statement := compileOne(t, Request{
		Range: TimeRange{StartMS: 1_000, EndMS: 61_000}, ResultType: ResultTimeSeries, StepMS: 1_000,
		Queries: []Query{TraceQuery{
			Common: CommonQuery{
				Name:    "traces",
				GroupBy: []FieldRef{{Name: "http.route", Context: FieldContextAttribute, Type: ValueTypeString}},
				Filter: Predicate{
					Field: FieldRef{Name: "service.name", Context: FieldContextResource, Type: ValueTypeString},
					Op:    FilterEqual,
					Value: Value{Kind: ValueString, String: "api"},
				},
			},
			Aggregation: TraceAggregateCount,
		}},
	})
	wantSQL := "SELECT intDiv(toUnixTimestamp64Milli(signoz_index_v3.timestamp), ?) * ? AS timestamp, `attribute_string_http$$route` AS group_0, count() AS value FROM signoz_traces.signoz_index_v3 WHERE (signoz_traces.signoz_index_v3.timestamp >= fromUnixTimestamp64Milli(?) AND signoz_traces.signoz_index_v3.timestamp < fromUnixTimestamp64Milli(?)) AND ((`resource_string_service$$name_exists`) AND (`resource_string_service$$name` = ?)) GROUP BY intDiv(toUnixTimestamp64Milli(signoz_index_v3.timestamp), ?) * ?, `attribute_string_http$$route` ORDER BY timestamp ASC"
	wantArgs := []any{int64(1_000), int64(1_000), int64(1_000), int64(61_000), "api", int64(1_000), int64(1_000)}
	assertStatement(t, statement, wantSQL, wantArgs)
}

func TestDefaultCatalogMaterializedManifestHasTrustedTraceColumns(t *testing.T) {
	if len(defaultMaterializedFields) != 9 {
		t.Fatalf("materialized field count = %d, want 9", len(defaultMaterializedFields))
	}
	for key, field := range defaultMaterializedFields {
		if key.Signal != SignalTraces || key.Type != ValueTypeString {
			t.Fatalf("manifest key = %#v, want string trace field", key)
		}
		if !strings.HasPrefix(field.Column, string(key.Context)+"_string_") || !strings.HasSuffix(field.ExistsColumn, "_exists") {
			t.Fatalf("manifest field = %#v has invalid trusted column names", field)
		}
	}
}

func TestDefaultCatalogResolvesEveryMaterializedManifestField(t *testing.T) {
	tests := []struct {
		field        FieldRef
		column       string
		existsColumn string
	}{
		{FieldRef{Name: "service.name", Context: FieldContextResource, Type: ValueTypeString}, "resource_string_service$$name", "resource_string_service$$name_exists"},
		{FieldRef{Name: "http.route", Context: FieldContextAttribute, Type: ValueTypeString}, "attribute_string_http$$route", "attribute_string_http$$route_exists"},
		{FieldRef{Name: "messaging.system", Context: FieldContextAttribute, Type: ValueTypeString}, "attribute_string_messaging$$system", "attribute_string_messaging$$system_exists"},
		{FieldRef{Name: "messaging.operation", Context: FieldContextAttribute, Type: ValueTypeString}, "attribute_string_messaging$$operation", "attribute_string_messaging$$operation_exists"},
		{FieldRef{Name: "db.system", Context: FieldContextAttribute, Type: ValueTypeString}, "attribute_string_db$$system", "attribute_string_db$$system_exists"},
		{FieldRef{Name: "rpc.system", Context: FieldContextAttribute, Type: ValueTypeString}, "attribute_string_rpc$$system", "attribute_string_rpc$$system_exists"},
		{FieldRef{Name: "rpc.service", Context: FieldContextAttribute, Type: ValueTypeString}, "attribute_string_rpc$$service", "attribute_string_rpc$$service_exists"},
		{FieldRef{Name: "rpc.method", Context: FieldContextAttribute, Type: ValueTypeString}, "attribute_string_rpc$$method", "attribute_string_rpc$$method_exists"},
		{FieldRef{Name: "peer.service", Context: FieldContextAttribute, Type: ValueTypeString}, "attribute_string_peer$$service", "attribute_string_peer$$service_exists"},
	}

	for _, test := range tests {
		t.Run(test.field.Name, func(t *testing.T) {
			got, err := (DefaultCatalog{}).Resolve(SignalTraces, test.field)
			if err != nil {
				t.Fatalf("Resolve() error = %v", err)
			}
			want := ResolvedField{
				SQL:               quoteIdentifier(test.column),
				ExistsSQL:         quoteIdentifier(test.existsColumn),
				RequiresExistence: true,
			}
			if !reflect.DeepEqual(got, want) {
				t.Fatalf("Resolve() = %#v, want %#v", got, want)
			}
		})
	}
}

func TestCompilerQualifiesLogTimeBucketForClickHouseAliasResolution(t *testing.T) {
	statement := compileOne(t, Request{
		Range: TimeRange{StartMS: 1_000, EndMS: 61_000}, ResultType: ResultTimeSeries, StepMS: 1_000,
		Queries: []Query{LogQuery{Common: CommonQuery{Name: "logs"}, Aggregation: LogAggregateCount}},
	})
	wantSQL := "SELECT intDiv(signoz_logs.logs_v2.timestamp, toUInt64(?)) * toUInt64(?) AS timestamp, count() AS value FROM signoz_logs.logs_v2 WHERE signoz_logs.logs_v2.timestamp >= toUInt64(?) AND signoz_logs.logs_v2.timestamp < toUInt64(?) GROUP BY intDiv(signoz_logs.logs_v2.timestamp, toUInt64(?)) * toUInt64(?) ORDER BY timestamp ASC"
	wantArgs := []any{int64(1_000_000_000), int64(1_000), int64(1_000_000_000), int64(61_000_000_000), int64(1_000_000_000), int64(1_000)}
	assertStatement(t, statement, wantSQL, wantArgs)
}

func TestCompilerCompilesLogAggregationPredicate(t *testing.T) {
	statement := compileOne(t, Request{
		Range: TimeRange{StartMS: 1, EndMS: 2}, ResultType: ResultScalar,
		Queries: []Query{LogQuery{
			Common: CommonQuery{
				Name:      "bytes",
				Predicate: &AggregationPredicate{Operator: CompareGreaterThan, Value: 10},
			},
			Aggregation: LogAggregateSum,
			Field:       FieldRef{Name: "http.response.size", Context: FieldContextAttribute, Type: ValueTypeNumber},
		}},
	})
	wantSQL := "SELECT sum(attributes_number[?]) AS value FROM signoz_logs.logs_v2 WHERE signoz_logs.logs_v2.timestamp >= toUInt64(?) AND signoz_logs.logs_v2.timestamp < toUInt64(?) HAVING sum(attributes_number[?]) > ? ORDER BY value DESC"
	wantArgs := []any{"http.response.size", int64(1_000_000), int64(2_000_000), "http.response.size", float64(10)}
	assertStatement(t, statement, wantSQL, wantArgs)
}

func TestCompilerCompilesTraceSummary(t *testing.T) {
	statement := compileOne(t, Request{
		Range: TimeRange{StartMS: 1, EndMS: 2}, ResultType: ResultTrace,
		Queries: []Query{TraceQuery{
			Common:      CommonQuery{Name: "traces", Limit: 20},
			Aggregation: TraceAggregateCount,
		}},
	})
	wantSQL := "SELECT trace_id, max(timestamp) AS timestamp, count() AS span_count, sum(duration_nano) AS duration_nano FROM signoz_traces.signoz_index_v3 WHERE signoz_traces.signoz_index_v3.timestamp >= fromUnixTimestamp64Milli(?) AND signoz_traces.signoz_index_v3.timestamp < fromUnixTimestamp64Milli(?) GROUP BY trace_id ORDER BY timestamp DESC, trace_id DESC LIMIT ?"
	wantArgs := []any{int64(1), int64(2), uint32(20)}
	assertStatement(t, statement, wantSQL, wantArgs)
}

func TestCompilerDoesNotEmbedDynamicMapKeysOrJSONPaths(t *testing.T) {
	malicious := "service.name'] OR 1=1 --"
	statement := compileOne(t, Request{
		Range: TimeRange{StartMS: 1, EndMS: 2}, ResultType: ResultRaw,
		Queries: []Query{LogQuery{
			Common: CommonQuery{
				Name:   "logs",
				Select: []FieldRef{{Name: malicious, Context: FieldContextBody, Type: ValueTypeString}},
				Filter: Predicate{Field: FieldRef{Name: malicious, Context: FieldContextResource, Type: ValueTypeString}, Op: FilterExists, Value: Value{Kind: ValueNone}},
			},
			Aggregation: LogAggregateCount,
		}},
	})
	if strings.Contains(statement.SQL, malicious) {
		t.Fatalf("SQL contains dynamic input: %s", statement.SQL)
	}
	if !reflect.DeepEqual(statement.Args, []any{"$." + malicious, int64(1_000_000), int64(2_000_000), malicious, uint32(100)}) {
		t.Fatalf("Args = %#v", statement.Args)
	}
}

func TestCompilerRejectsCursorUntilScannerExists(t *testing.T) {
	request := Request{
		Range: TimeRange{StartMS: 1, EndMS: 2}, ResultType: ResultRaw,
		Queries: []Query{LogQuery{Common: CommonQuery{Name: "logs", Cursor: "opaque"}, Aggregation: LogAggregateCount}},
	}
	plan, err := (DefaultPlanner{}).Plan(request)
	if err != nil {
		t.Fatalf("Plan() error = %v", err)
	}
	_, err = NewCompiler(nil).Compile(plan)
	var queryErr *Error
	if !errors.As(err, &queryErr) || queryErr.Code != ErrorUnsupported {
		t.Fatalf("Compile() error = %v, want unsupported error", err)
	}
}

func TestCompilerCompilesTypedLiveLogCursor(t *testing.T) {
	statement := compileOne(t, Request{
		Range: TimeRange{StartMS: 1, EndMS: 2}, ResultType: ResultRaw,
		Queries: []Query{LogQuery{Common: CommonQuery{
			Name:  "logs",
			After: &RawLogCursor{TimestampNS: 1_500_000, ID: "last"},
		}, Aggregation: LogAggregateCount}},
	})
	wantSQL := "SELECT timestamp AS field_0, id AS field_1, severity_text AS field_2, body AS field_3, trace_id AS field_4, span_id AS field_5 FROM signoz_logs.logs_v2 WHERE (signoz_logs.logs_v2.timestamp >= toUInt64(?) AND signoz_logs.logs_v2.timestamp < toUInt64(?)) AND ((timestamp > toUInt64(?)) OR (timestamp = toUInt64(?) AND id > ?)) ORDER BY timestamp ASC, id ASC LIMIT ?"
	wantArgs := []any{int64(1_000_000), int64(2_000_000), uint64(1_500_000), uint64(1_500_000), "last", uint32(100)}
	assertStatement(t, statement, wantSQL, wantArgs)
}

func TestCompilerCompilesMetricRateWithTwoAggregationStages(t *testing.T) {
	service := FieldRef{Name: "service.name", Context: FieldContextLabel, Type: ValueTypeString}
	statement := compileOne(t, Request{
		Range: TimeRange{StartMS: 1_000, EndMS: 61_000}, ResultType: ResultTimeSeries, StepMS: 1_000,
		Queries: []Query{MetricQuery{Common: CommonQuery{
			Name: "requests", GroupBy: []FieldRef{service},
			Filter: Predicate{Field: service, Op: FilterEqual, Value: Value{Kind: ValueString, String: "api"}},
		}, Aggregation: MetricAggregation{
			MetricName: "http.server.request.count", Type: MetricSum, Temporality: TemporalityCumulative,
			TimeAggregation: TimeAggregateRate, SpaceAggregation: SpaceAggregateSum,
		}}},
	})
	for _, fragment := range []string{
		"__lite_series AS", "__lite_bucketed AS", "__lite_temporal AS",
		"PARTITION BY fingerprint ORDER BY timestamp", "sum(per_series_value) AS value",
	} {
		if !strings.Contains(statement.SQL, fragment) {
			t.Fatalf("SQL does not contain %q:\n%s", fragment, statement.SQL)
		}
	}
	if strings.Contains(statement.SQL, "service.name") || strings.Contains(statement.SQL, "api") {
		t.Fatalf("SQL contains dynamic metric input: %s", statement.SQL)
	}
	wantArgs := []any{
		"service.name", "http.server.request.count", "Sum", "Cumulative", false,
		"service.name", "service.name", "api", "service.name", int64(1_000), int64(1_000),
		"http.server.request.count", "Cumulative", int64(1_000), int64(61_000), int64(1_000), int64(1_000),
	}
	if !reflect.DeepEqual(statement.Args, wantArgs) {
		t.Fatalf("Args = %#v, want %#v\nSQL: %s", statement.Args, wantArgs, statement.SQL)
	}
}

func TestCompilerCompilesExplicitHistogramAndRejectsAmbiguousName(t *testing.T) {
	request := Request{
		Range: TimeRange{StartMS: 1_000, EndMS: 61_000}, ResultType: ResultTimeSeries, StepMS: 1_000,
		Queries: []Query{MetricQuery{Common: CommonQuery{Name: "latency"}, Aggregation: MetricAggregation{
			MetricName: "http.server.duration.bucket", Type: MetricHistogram, Temporality: TemporalityUnspecified,
			TimeAggregation: TimeAggregateCount, SpaceAggregation: SpaceAggregateP95,
		}}},
	}
	statement := compileOne(t, request)
	for _, fragment := range []string{"series.le", "quantileExactWeighted", "__lite_histogram_weights AS"} {
		if !strings.Contains(statement.SQL, fragment) {
			t.Fatalf("SQL does not contain %q:\n%s", fragment, statement.SQL)
		}
	}
	metric := request.Queries[0].(MetricQuery)
	metric.Aggregation.MetricName = "http.server.duration"
	request.Queries = []Query{metric}
	plan, err := (DefaultPlanner{}).Plan(request)
	if err != nil {
		t.Fatalf("Plan() error = %v", err)
	}
	_, err = NewCompiler(nil).Compile(plan)
	var queryErr *Error
	if !errors.As(err, &queryErr) || queryErr.Code != ErrorInvalidAggregation {
		t.Fatalf("Compile() error = %v, want invalid aggregation", err)
	}
}

func TestCompilerCompilesMeterWithoutMetricMetadataJoin(t *testing.T) {
	statement := compileOne(t, Request{
		Range: TimeRange{StartMS: 1_000, EndMS: 2_000}, ResultType: ResultTimeSeries, StepMS: 1_000,
		Queries: []Query{MeterQuery{Common: CommonQuery{Name: "cost", GroupBy: []FieldRef{{Name: "service.name", Context: FieldContextLabel, Type: ValueTypeString}}}, Aggregation: MetricAggregation{
			MetricName: "signoz.meter.log.size", Type: MetricSum, Temporality: TemporalityDelta,
			TimeAggregation: TimeAggregateSum, SpaceAggregation: SpaceAggregateSum,
		}}},
	})
	if !strings.Contains(statement.SQL, "signoz_meter.samples") || strings.Contains(statement.SQL, "time_series_v4") {
		t.Fatalf("unexpected meter source SQL: %s", statement.SQL)
	}
}

func TestCompilerCompilesScalarCumulativeIncreaseOverPointOrder(t *testing.T) {
	statement := compileOne(t, Request{
		Range: TimeRange{StartMS: 1_000, EndMS: 61_000}, ResultType: ResultScalar,
		Queries: []Query{MetricQuery{Common: CommonQuery{Name: "requests"}, Aggregation: MetricAggregation{
			MetricName: "http.server.request.count", Type: MetricSum, Temporality: TemporalityCumulative,
			TimeAggregation: TimeAggregateIncrease, SpaceAggregation: SpaceAggregateSum,
		}}},
	})
	for _, fragment := range []string{"points.unix_milli AS timestamp", "PARTITION BY fingerprint ORDER BY timestamp", "sum(per_series_value) AS value"} {
		if !strings.Contains(statement.SQL, fragment) {
			t.Fatalf("scalar counter SQL does not contain %q:\n%s", fragment, statement.SQL)
		}
	}
}

func compileOne(t *testing.T, request Request) Statement {
	t.Helper()
	plan, err := (DefaultPlanner{}).Plan(request)
	if err != nil {
		t.Fatalf("Plan() error = %v", err)
	}
	statements, err := NewCompiler(nil).Compile(plan)
	if err != nil {
		t.Fatalf("Compile() error = %v", err)
	}
	if len(statements) != 1 {
		t.Fatalf("Compile() statements = %d, want 1", len(statements))
	}
	return statements[0]
}

func assertStatement(t *testing.T, statement Statement, wantSQL string, wantArgs []any) {
	t.Helper()
	if statement.SQL != wantSQL {
		t.Fatalf("SQL =\n%s\nwant\n%s", statement.SQL, wantSQL)
	}
	if !reflect.DeepEqual(statement.Args, wantArgs) {
		t.Fatalf("Args = %#v, want %#v", statement.Args, wantArgs)
	}
}
