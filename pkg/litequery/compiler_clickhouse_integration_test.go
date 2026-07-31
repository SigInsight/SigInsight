//go:build integration

package litequery

import (
	"context"
	"os"
	"testing"

	"github.com/ClickHouse/clickhouse-go/v2"
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
			Queries: []Query{TraceQuery{Common: CommonQuery{
				Name:    "traces",
				GroupBy: []FieldRef{{Name: "service.name", Context: FieldContextResource, Type: ValueTypeString}},
			}, Aggregation: TraceAggregateDurationP95}},
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
			if err := rows.Close(); err != nil {
				t.Fatalf("Close(%s) error = %v", statement.Name, err)
			}
		}
	}
}
