//go:build integration

package litequery

import (
	"context"
	"fmt"
	"os"
	"reflect"
	"testing"
	"time"

	"github.com/ClickHouse/clickhouse-go/v2"
	"github.com/ClickHouse/clickhouse-go/v2/lib/driver"
)

func TestCompilerExecutesOnCurrentClickHouseSchema(t *testing.T) {
	dsn := os.Getenv("LITEQUERY_CLICKHOUSE_DSN")
	if dsn == "" {
		t.Skip("set LITEQUERY_CLICKHOUSE_DSN to run ClickHouse compiler integration")
	}
	options, err := clickhouse.ParseDSN(dsn)
	if err != nil {
		t.Fatalf("ParseDSN() error = %v", err)
	}
	conn, err := clickhouse.Open(options)
	if err != nil {
		t.Fatalf("Open() error = %v", err)
	}
	defer conn.Close()
	ctx := context.Background()
	now := time.Now().UnixMilli()
	seedMetricCompilerData(t, conn, now)
	traceIDs := seedTraceSummaryData(t, conn, now)

	requests := []Request{
		{
			Range: TimeRange{StartMS: 1_000, EndMS: 2_000}, ResultType: ResultRaw,
			Queries: []Query{LogQuery{Common: CommonQuery{
				Name: "logs",
				Select: []FieldRef{
					{Name: "body", Context: FieldContextBody, Type: ValueTypeString},
					{Name: "request.id", Context: FieldContextBody, Type: ValueTypeString},
				},
				Filter: Predicate{Field: FieldRef{Name: "service.name", Context: FieldContextResource, Type: ValueTypeString}, Op: FilterEqual, Value: Value{Kind: ValueString, String: "api"}},
			}, Aggregation: LogAggregateCount}},
		},
		{
			Range: TimeRange{StartMS: 1_000, EndMS: 61_000}, ResultType: ResultTimeSeries, StepMS: 1_000,
			Queries: []Query{LogQuery{Common: CommonQuery{
				Name: "numeric_log_attribute",
				Filter: Predicate{
					Field: FieldRef{Name: "thread.id", Context: FieldContextAttribute, Type: ValueTypeNumber},
					Op:    FilterEqual,
					Value: Value{Kind: ValueNumber, Number: 65},
				},
			}, Aggregation: LogAggregateCount}},
		},
		{
			Range: TimeRange{StartMS: 1_000, EndMS: 61_000}, ResultType: ResultTimeSeries, StepMS: 1_000,
			Queries: []Query{TraceQuery{Common: CommonQuery{
				Name:    "traces",
				GroupBy: []FieldRef{{Name: "service.name", Context: FieldContextResource, Type: ValueTypeString}},
			}, Aggregation: TraceAggregateDurationP95}},
		},
		{
			Range: TimeRange{StartMS: now - 3_000, EndMS: now + 1_000}, ResultType: ResultTimeSeries, StepMS: 1_000,
			Queries: []Query{TraceQuery{Common: CommonQuery{
				Name: "trace_quick_filters",
				Filter: LogicalFilter{Operator: BooleanAnd, Items: []FilterNode{
					Predicate{Field: FieldRef{Name: "name", Context: FieldContextSpan, Type: ValueTypeString}, Op: FilterNotIn, Value: Value{Kind: ValueStringList, Strings: []string{"excluded-operation"}}},
					Predicate{Field: FieldRef{Name: "has_error", Context: FieldContextSpan, Type: ValueTypeBool}, Op: FilterNotIn, Value: Value{Kind: ValueBoolList, Bools: []bool{true}}},
				}},
			}, Aggregation: TraceAggregateCount}},
		},
		{
			Range: TimeRange{StartMS: now - 3_000, EndMS: now + 1_000}, ResultType: ResultTimeSeries, StepMS: 1_000,
			Queries: []Query{TraceQuery{Common: CommonQuery{
				Name: "trace_pattern_filters",
				Filter: LogicalFilter{Operator: BooleanAnd, Items: []FilterNode{
					Predicate{Field: FieldRef{Name: "http.route", Context: FieldContextAttribute, Type: ValueTypeString}, Op: FilterILike, Value: Value{Kind: ValueString, String: "/MATCHED-%"}},
					Predicate{Field: FieldRef{Name: "name", Context: FieldContextSpan, Type: ValueTypeString}, Op: FilterRegexp, Value: Value{Kind: ValueString, String: "^(child|orphan)-operation$"}},
					Predicate{Field: FieldRef{Name: "service.name", Context: FieldContextResource, Type: ValueTypeString}, Op: FilterNotLike, Value: Value{Kind: ValueString, String: "excluded-%"}},
				}},
			}, Aggregation: TraceAggregateCount}},
		},
		{
			Range: TimeRange{StartMS: now - 3_000, EndMS: now + 1_000}, ResultType: ResultTimeSeries, StepMS: 1_000,
			Queries: []Query{MetricQuery{Common: CommonQuery{
				Name: "requests", GroupBy: []FieldRef{{Name: "service.name", Context: FieldContextLabel, Type: ValueTypeString}},
			}, Aggregation: MetricAggregation{
				MetricName: "http.server.request.count", Type: MetricSum, Temporality: TemporalityCumulative,
				TimeAggregation: TimeAggregateRate, SpaceAggregation: SpaceAggregateSum,
			}}},
		},
		{
			Range: TimeRange{StartMS: now - 3_000, EndMS: now + 1_000}, ResultType: ResultTimeSeries, StepMS: 1_000,
			Queries: []Query{MetricQuery{Common: CommonQuery{
				Name: "semantic_gauge", GroupBy: []FieldRef{{Name: "service.name", Context: FieldContextLabel, Type: ValueTypeString}},
			}, Aggregation: MetricAggregation{
				MetricName: "http.server.request.count", Type: MetricGauge, Temporality: TemporalityUnspecified,
				TimeAggregation: TimeAggregateLatest, SpaceAggregation: SpaceAggregateAvg,
			}}},
		},
		{
			Range: TimeRange{StartMS: now - 3_000, EndMS: now + 1_000}, ResultType: ResultTimeSeries, StepMS: 1_000,
			Queries: []Query{MetricQuery{Common: CommonQuery{Name: "latency"}, Aggregation: MetricAggregation{
				MetricName: "http.server.duration.bucket", Type: MetricHistogram, Temporality: TemporalityCumulative,
				TimeAggregation: TimeAggregateCount, SpaceAggregation: SpaceAggregateP95,
			}}},
		},
		{
			Range: TimeRange{StartMS: now - 3_000, EndMS: now + 1_000}, ResultType: ResultTimeSeries, StepMS: 1_000,
			Queries: []Query{MetricQuery{Common: CommonQuery{Name: "latency_delta"}, Aggregation: MetricAggregation{
				MetricName: "http.server.delta.duration.bucket", Type: MetricHistogram, Temporality: TemporalityDelta,
				TimeAggregation: TimeAggregateCount, SpaceAggregation: SpaceAggregateP95,
			}}},
		},
		{
			Range: TimeRange{StartMS: now - 3_000, EndMS: now + 1_000}, ResultType: ResultTimeSeries, StepMS: 1_000,
			Queries: []Query{MeterQuery{Common: CommonQuery{Name: "meter", GroupBy: []FieldRef{{Name: "service.name", Context: FieldContextLabel, Type: ValueTypeString}}}, Aggregation: MetricAggregation{
				MetricName: "signoz.meter.log.size", Type: MetricSum, Temporality: TemporalityDelta,
				TimeAggregation: TimeAggregateSum, SpaceAggregation: SpaceAggregateSum,
			}}},
		},
	}

	for _, request := range requests {
		plan, err := (DefaultPlanner{}).Plan(request)
		if err != nil {
			t.Fatalf("Plan() error = %v", err)
		}
		statements, err := NewCompiler(nil).Compile(plan)
		if err != nil {
			t.Fatalf("Compile() error = %v", err)
		}
		for _, statement := range statements {
			rows, err := conn.Query(context.Background(), statement.SQL, statement.Args...)
			if err != nil {
				t.Fatalf("Query(%s) error = %v\nSQL: %s\nArgs: %#v", statement.Name, err, statement.SQL, statement.Args)
			}
			if statement.Name == "requests" || statement.Name == "semantic_gauge" || statement.Name == "latency" || statement.Name == "latency_delta" || statement.Name == "meter" || statement.Name == "trace_pattern_filters" {
				assertPositiveMetricRows(t, statement.Name, rows)
			}
			if err := rows.Err(); err != nil {
				t.Fatalf("Rows(%s) error = %v", statement.Name, err)
			}
			if err := rows.Close(); err != nil {
				t.Fatalf("Close(%s) error = %v", statement.Name, err)
			}
		}
	}

	plan, err := (DefaultPlanner{}).Plan(Request{
		Range: TimeRange{StartMS: now - 3_000, EndMS: now + 1_000}, ResultType: ResultTimeSeries, StepMS: 1_000,
		Queries: []Query{
			MeterQuery{Common: CommonQuery{Name: "A", GroupBy: []FieldRef{{Name: "service.name", Context: FieldContextLabel, Type: ValueTypeString}}}, Aggregation: MetricAggregation{MetricName: "signoz.meter.log.size", Type: MetricSum, Temporality: TemporalityDelta, TimeAggregation: TimeAggregateSum, SpaceAggregation: SpaceAggregateSum}},
			MeterQuery{Common: CommonQuery{Name: "B", GroupBy: []FieldRef{{Name: "service.name", Context: FieldContextLabel, Type: ValueTypeString}}}, Aggregation: MetricAggregation{MetricName: "signoz.meter.log.size", Type: MetricSum, Temporality: TemporalityDelta, TimeAggregation: TimeAggregateSum, SpaceAggregation: SpaceAggregateSum}},
		},
		Formulas: []Formula{{Name: "F", Expression: "A + B"}},
	})
	if err != nil {
		t.Fatalf("Plan executor formula error = %v", err)
	}
	result, err := (Executor{Query: func(ctx context.Context, query string, args ...any) (Rows, error) {
		rows, err := conn.Query(ctx, query, args...)
		if err != nil {
			return nil, err
		}
		return &integrationClickHouseRows{Rows: rows}, nil
	}}).Execute(ctx, plan)
	if err != nil {
		t.Fatalf("Executor.Execute() error = %v", err)
	}
	if len(result.Queries) != 3 || len(result.Queries[2].Rows) == 0 {
		t.Fatalf("Executor result = %#v", result)
	}

	tracePlan, err := (DefaultPlanner{}).Plan(Request{
		Range: TimeRange{StartMS: now - 3_000, EndMS: now + 1_000}, ResultType: ResultTrace,
		Queries: []Query{TraceQuery{Common: CommonQuery{
			Name:   "trace_summary",
			Filter: Predicate{Field: FieldRef{Name: "http.route", Context: FieldContextAttribute, Type: ValueTypeString}, Op: FilterEqual, Value: Value{Kind: ValueString, String: "/matched-child"}},
		}, Aggregation: TraceAggregateCount}},
	})
	if err != nil {
		t.Fatalf("Plan trace summary error = %v", err)
	}
	traceResult, err := (Executor{Query: func(ctx context.Context, query string, args ...any) (Rows, error) {
		rows, err := conn.Query(ctx, query, args...)
		if err != nil {
			return nil, err
		}
		return WrapClickHouseRows(rows), nil
	}}).Execute(ctx, tracePlan)
	if err != nil {
		t.Fatalf("Execute trace summary error = %v", err)
	}
	if len(traceResult.Queries) != 1 || len(traceResult.Queries[0].Rows) != 2 {
		t.Fatalf("trace summary result = %#v", traceResult)
	}
	rowsByTraceID := make(map[string][]any, 2)
	for _, row := range traceResult.Queries[0].Rows {
		rowsByTraceID[fmt.Sprint(row[0])] = row
	}
	rooted := rowsByTraceID[traceIDs[0]]
	if rooted == nil || rooted[2] != uint64(2) || rooted[3] != uint64(10_000_000) || fmt.Sprint(rooted[4]) != "root-service" || fmt.Sprint(rooted[5]) != "root-operation" {
		t.Fatalf("rooted trace summary row = %#v", rooted)
	}
	orphan := rowsByTraceID[traceIDs[1]]
	if orphan == nil || orphan[2] != uint64(1) || orphan[3] != uint64(4_000_000) || fmt.Sprint(orphan[4]) != "orphan-service" || fmt.Sprint(orphan[5]) != "orphan-operation" {
		t.Fatalf("orphan trace summary row = %#v", orphan)
	}

	// Verify Root and Entrypoint remain distinct: Entrypoint follows the OTel
	// receiving-boundary semantics encoded directly on the span row.
	for _, test := range []struct {
		name       string
		field      string
		wantSpanID string
	}{
		{name: "root", field: "isRoot", wantSpanID: "root-span"},
		{name: "entrypoint", field: "isEntryPoint", wantSpanID: "child-span"},
	} {
		t.Run("trace_detail_"+test.name, func(t *testing.T) {
			scopePlan, err := (DefaultPlanner{}).Plan(Request{
				Range: TimeRange{StartMS: now - 3_000, EndMS: now + 1_000}, ResultType: ResultRaw,
				Queries: []Query{TraceQuery{Common: CommonQuery{
					Name:   "trace_detail_scope",
					Select: []FieldRef{{Name: "span_id", Context: FieldContextSpan, Type: ValueTypeString}},
					Filter: LogicalFilter{Operator: BooleanAnd, Items: []FilterNode{
						Predicate{Field: FieldRef{Name: "trace_id", Context: FieldContextSpan, Type: ValueTypeString}, Op: FilterEqual, Value: Value{Kind: ValueString, String: traceIDs[0]}},
						Predicate{Field: FieldRef{Name: test.field, Context: FieldContextSpan, Type: ValueTypeBool}, Op: FilterEqual, Value: Value{Kind: ValueBool, Bool: true}},
					}},
				}, Aggregation: TraceAggregateCount}},
			})
			if err != nil {
				t.Fatalf("Plan() error = %v", err)
			}
			result, err := (Executor{Query: func(ctx context.Context, query string, args ...any) (Rows, error) {
				rows, err := conn.Query(ctx, query, args...)
				if err != nil {
					return nil, err
				}
				return WrapClickHouseRows(rows), nil
			}}).Execute(ctx, scopePlan)
			if err != nil {
				t.Fatalf("Execute() error = %v", err)
			}
			if len(result.Queries) != 1 || len(result.Queries[0].Rows) != 1 || fmt.Sprint(result.Queries[0].Rows[0][0]) != test.wantSpanID {
				t.Fatalf("scope result = %#v, want span %q", result, test.wantSpanID)
			}
		})
	}

	overflowPlan, err := (DefaultPlanner{}).Plan(Request{
		Range: TimeRange{StartMS: now - 3_000, EndMS: now + 1_000}, ResultType: ResultRaw,
		Queries: []Query{TraceQuery{Common: CommonQuery{
			Name:  "trace_limit_probe",
			Limit: 1,
			Filter: Predicate{
				Field: FieldRef{Name: "trace_id", Context: FieldContextSpan, Type: ValueTypeString},
				Op:    FilterEqual,
				Value: Value{Kind: ValueString, String: traceIDs[0]},
			},
		}, Aggregation: TraceAggregateCount}},
	})
	if err != nil {
		t.Fatalf("Plan overflow probe error = %v", err)
	}
	overflowResult, err := (Executor{Query: func(ctx context.Context, query string, args ...any) (Rows, error) {
		rows, err := conn.Query(ctx, query, args...)
		if err != nil {
			return nil, err
		}
		return WrapClickHouseRows(rows), nil
	}}).Execute(ctx, overflowPlan)
	if err != nil {
		t.Fatalf("Execute overflow probe error = %v", err)
	}
	if len(overflowResult.Queries) != 1 || len(overflowResult.Queries[0].Rows) != 1 || !overflowResult.Queries[0].Truncated || len(overflowResult.Warnings) != 1 {
		t.Fatalf("overflow result = %#v, want one retained row and truncation warning", overflowResult)
	}
}

// clickHouseRows is a test-side driver adapter. The lightweight executor stays
// independent of ClickHouse's concrete Scan requirements, while production
// adapters can make the same conversion at the infrastructure boundary.
type integrationClickHouseRows struct{ driver.Rows }

func (r *integrationClickHouseRows) Scan(destinations ...any) error {
	types := r.ColumnTypes()
	values := make([]any, len(types))
	for index, columnType := range types {
		values[index] = reflect.New(columnType.ScanType()).Interface()
	}
	if err := r.Rows.Scan(values...); err != nil {
		return err
	}
	for index, value := range values {
		*destinations[index].(*any) = reflect.ValueOf(value).Elem().Interface()
	}
	return nil
}

func assertPositiveMetricRows(t *testing.T, name string, rows interface {
	Next() bool
	Scan(...any) error
}) {
	t.Helper()
	foundPositiveValue := false
	for rows.Next() {
		var timestamp int64
		if name == "trace_pattern_filters" {
			var value uint64
			if err := rows.Scan(&timestamp, &value); err != nil {
				t.Fatalf("Scan(%s) error = %v", name, err)
			}
			if value > 0 {
				foundPositiveValue = true
			}
			continue
		}
		var value float64
		if name == "latency" || name == "latency_delta" {
			if err := rows.Scan(&timestamp, &value); err != nil {
				t.Fatalf("Scan(%s) error = %v", name, err)
			}
		} else {
			var group string
			if err := rows.Scan(&timestamp, &group, &value); err != nil {
				t.Fatalf("Scan(%s) error = %v", name, err)
			}
		}
		if value > 0 {
			foundPositiveValue = true
		}
	}
	if !foundPositiveValue {
		t.Fatalf("Query(%s) returned no positive metric value", name)
	}
}

func seedMetricCompilerData(t *testing.T, conn clickhouse.Conn, now int64) {
	t.Helper()
	ctx := context.Background()
	series := "INSERT INTO siginsight_metrics.time_series_v4 (temporality, metric_name, type, fingerprint, unix_milli, labels, __normalized) VALUES (?, ?, ?, ?, ?, ?, ?)"
	for _, row := range [][]any{
		{"Cumulative", "http.server.request.count", "Sum", uint64(101), now - 2_000, `{"service.name":"api"}`, false},
		{"Cumulative", "http.server.duration.bucket", "Histogram", uint64(201), now - 2_000, `{"le":"10"}`, false},
		{"Cumulative", "http.server.duration.bucket", "Histogram", uint64(202), now - 2_000, `{"le":"+Inf"}`, false},
		{"Delta", "http.server.delta.duration.bucket", "Histogram", uint64(203), now - 2_000, `{"le":"10"}`, false},
		{"Delta", "http.server.delta.duration.bucket", "Histogram", uint64(204), now - 2_000, `{"le":"+Inf"}`, false},
	} {
		if err := conn.Exec(ctx, series, row...); err != nil {
			t.Fatalf("insert metric series error = %v", err)
		}
	}
	points := "INSERT INTO siginsight_metrics.samples_v4 (temporality, metric_name, fingerprint, unix_milli, value, inserted_at_unix_milli) VALUES (?, ?, ?, ?, ?, ?)"
	for _, row := range [][]any{
		{"Cumulative", "http.server.request.count", uint64(101), now - 2_000, 10.0, now - 2_000},
		{"Cumulative", "http.server.request.count", uint64(101), now - 1_000, 15.0, now - 1_000},
		{"Cumulative", "http.server.duration.bucket", uint64(201), now - 2_000, 3.0, now - 2_000},
		{"Cumulative", "http.server.duration.bucket", uint64(201), now - 1_000, 5.0, now - 1_000},
		{"Cumulative", "http.server.duration.bucket", uint64(202), now - 2_000, 5.0, now - 2_000},
		{"Cumulative", "http.server.duration.bucket", uint64(202), now - 1_000, 8.0, now - 1_000},
		{"Delta", "http.server.delta.duration.bucket", uint64(203), now - 2_000, 3.0, now - 2_000},
		{"Delta", "http.server.delta.duration.bucket", uint64(203), now - 1_000, 2.0, now - 1_000},
		{"Delta", "http.server.delta.duration.bucket", uint64(204), now - 2_000, 5.0, now - 2_000},
		{"Delta", "http.server.delta.duration.bucket", uint64(204), now - 1_000, 3.0, now - 1_000},
	} {
		if err := conn.Exec(ctx, points, row...); err != nil {
			t.Fatalf("insert metric point error = %v", err)
		}
	}
	meter := "INSERT INTO siginsight_meter.samples (temporality, metric_name, type, labels, fingerprint, unix_milli, value) VALUES (?, ?, ?, ?, ?, ?, ?)"
	for _, row := range [][]any{
		{"Delta", "signoz.meter.log.size", "Sum", `{"service.name":"api"}`, uint64(301), now - 2_000, 4.0},
		{"Delta", "signoz.meter.log.size", "Sum", `{"service.name":"api"}`, uint64(301), now - 1_000, 6.0},
	} {
		if err := conn.Exec(ctx, meter, row...); err != nil {
			t.Fatalf("insert meter point error = %v", err)
		}
	}
	var seriesCount, pointsCount uint64
	if err := conn.QueryRow(ctx, "SELECT count() FROM siginsight_metrics.time_series_v4 WHERE metric_name = ?", "http.server.request.count").Scan(&seriesCount); err != nil {
		t.Fatalf("count metric series error = %v", err)
	}
	if err := conn.QueryRow(ctx, "SELECT count() FROM siginsight_metrics.samples_v4 WHERE metric_name = ?", "http.server.request.count").Scan(&pointsCount); err != nil {
		t.Fatalf("count metric points error = %v", err)
	}
	if seriesCount < 1 || pointsCount < 2 {
		t.Fatalf("seeded metric data counts = series:%d points:%d", seriesCount, pointsCount)
	}
}

func seedTraceSummaryData(t *testing.T, conn clickhouse.Conn, now int64) [2]string {
	t.Helper()
	traceID := fmt.Sprintf("%032x", now)
	orphanTraceID := fmt.Sprintf("%032x", now+1)
	insert := "INSERT INTO siginsight_traces.span_index_v3 " +
		"(timestamp, trace_id, span_id, parent_span_id, name, duration_nano, kind, is_remote, resources_string, attributes_string) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)"
	rows := [][]any{
		{time.UnixMilli(now - 2_000), traceID, "root-span", "", "root-operation", uint64(10_000_000), int8(2), "no", map[string]string{"service.name": "root-service"}, map[string]string{}},
		{time.UnixMilli(now - 1_500), traceID, "child-span", "root-span", "child-operation", uint64(3_000_000), int8(2), "yes", map[string]string{"service.name": "child-service"}, map[string]string{"http.route": "/matched-child"}},
		{time.UnixMilli(now - 1_000), orphanTraceID, "orphan-span", "missing-parent", "orphan-operation", uint64(4_000_000), int8(1), "no", map[string]string{"service.name": "orphan-service"}, map[string]string{"http.route": "/matched-child"}},
	}
	for _, row := range rows {
		if err := conn.Exec(context.Background(), insert, row...); err != nil {
			t.Fatalf("insert trace summary row error = %v", err)
		}
	}
	return [2]string{traceID, orphanTraceID}
}
