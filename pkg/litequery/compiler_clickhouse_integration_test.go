//go:build integration

package litequery

import (
	"context"
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
				MetricName: "http.server.duration.bucket", Type: MetricHistogram, Temporality: TemporalityUnspecified,
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
			if statement.Name == "requests" || statement.Name == "semantic_gauge" || statement.Name == "latency" || statement.Name == "meter" {
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
		var value float64
		if name == "latency" {
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
