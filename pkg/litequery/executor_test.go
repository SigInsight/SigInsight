package litequery

import (
	"context"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/SigNoz/signoz/pkg/errors"
)

func TestExecutorScansResultsAndEvaluatesFormula(t *testing.T) {
	plan, err := (DefaultPlanner{}).Plan(Request{
		Range: TimeRange{StartMS: 1, EndMS: 2}, ResultType: ResultScalar,
		Queries: []Query{
			LogQuery{Common: CommonQuery{Name: "A"}, Aggregation: LogAggregateCount},
			LogQuery{Common: CommonQuery{Name: "B"}, Aggregation: LogAggregateCount},
		},
		Formulas: []Formula{{Name: "F", Expression: "A + B * 2"}},
	})
	if err != nil {
		t.Fatalf("Plan() error = %v", err)
	}
	var lock sync.Mutex
	values := []float64{2, 3}
	executor := Executor{Config: ExecutorConfig{MaxConcurrent: 1}, Query: func(context.Context, string, ...any) (Rows, error) {
		lock.Lock()
		defer lock.Unlock()
		value := values[0]
		values = values[1:]
		return &fakeRows{columns: []string{"value"}, data: [][]any{{value}}}, nil
	}}
	result, err := executor.Execute(context.Background(), plan)
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
	if len(result.Queries) != 3 || result.Queries[2].Name != "F" {
		t.Fatalf("Queries = %#v", result.Queries)
	}
	if got := result.Queries[2].Rows[0][0]; got != float64(8) {
		t.Fatalf("formula result = %#v, want 8", got)
	}
}

func TestExecutorResolvesForwardFormulaDependencies(t *testing.T) {
	plan, err := (DefaultPlanner{}).Plan(Request{
		Range: TimeRange{StartMS: 1, EndMS: 2}, ResultType: ResultScalar,
		Queries: []Query{
			LogQuery{Common: CommonQuery{Name: "A"}, Aggregation: LogAggregateCount},
			LogQuery{Common: CommonQuery{Name: "B"}, Aggregation: LogAggregateCount},
		},
		Formulas: []Formula{{Name: "F", Expression: "A + G"}, {Name: "G", Expression: "B + 1"}},
	})
	if err != nil {
		t.Fatalf("Plan() error = %v", err)
	}
	values := []float64{2, 3}
	executor := Executor{Config: ExecutorConfig{MaxConcurrent: 1}, Query: func(context.Context, string, ...any) (Rows, error) {
		value := values[0]
		values = values[1:]
		return &fakeRows{columns: []string{"value"}, data: [][]any{{value}}}, nil
	}}
	result, err := executor.Execute(context.Background(), plan)
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
	if result.Queries[2].Name != "F" || result.Queries[3].Name != "G" {
		t.Fatalf("formula result order = %q, %q, want F, G", result.Queries[2].Name, result.Queries[3].Name)
	}
	if got := result.Queries[2].Rows[0][0]; got != float64(6) {
		t.Fatalf("forward formula result = %#v, want 6", got)
	}
}

func TestEvaluateFormulasUsesCollisionSafeAlignmentAndReportsMissingInputs(t *testing.T) {
	groupA := FieldRef{Name: "first", Context: FieldContextAttribute, Type: ValueTypeString}
	groupB := FieldRef{Name: "second", Context: FieldContextAttribute, Type: ValueTypeString}
	columns := []ResultColumn{{Name: "group_0", Field: &groupA}, {Name: "group_1", Field: &groupB}, {Name: "value"}}
	plan := Plan{Formulas: []Formula{{Name: "F", Expression: "A + B"}}}
	formulaResults, warnings, err := evaluateFormulas(plan, []QueryResult{
		{Name: "A", Columns: columns, Rows: [][]any{{"a b", "c", float64(1)}, {"a", "b c", float64(2)}}},
		{Name: "B", Columns: columns, Rows: [][]any{{"a b", "c", float64(10)}}},
	})
	if err != nil {
		t.Fatalf("evaluateFormulas() error = %v", err)
	}
	if len(formulaResults) != 1 || len(formulaResults[0].Rows) != 2 {
		t.Fatalf("formula results = %#v, want two independently aligned rows", formulaResults)
	}
	if formulaResults[0].Rows[0][2] != float64(11) || formulaResults[0].Rows[1][2] != float64(2) {
		t.Fatalf("formula rows = %#v, want 11 and missing-as-zero 2", formulaResults[0].Rows)
	}
	if len(warnings) != 1 || !strings.Contains(warnings[0], "missing aligned value") {
		t.Fatalf("warnings = %#v, want missing alignment warning", warnings)
	}
}

func TestEvaluateFormulasRejectsDifferentGroupSchemas(t *testing.T) {
	service := FieldRef{Name: "service.name", Context: FieldContextResource, Type: ValueTypeString}
	host := FieldRef{Name: "host.name", Context: FieldContextResource, Type: ValueTypeString}
	_, _, err := evaluateFormulas(Plan{Formulas: []Formula{{Name: "F", Expression: "A + B"}}}, []QueryResult{
		{Name: "A", Columns: []ResultColumn{{Name: "group_0", Field: &service}, {Name: "value"}}, Rows: [][]any{{"api", float64(1)}}},
		{Name: "B", Columns: []ResultColumn{{Name: "group_0", Field: &host}, {Name: "value"}}, Rows: [][]any{{"api", float64(2)}}},
	})
	var queryErr *Error
	if !errors.As(err, &queryErr) || queryErr.Code != ErrorInvalidFormula {
		t.Fatalf("evaluateFormulas() error = %v, want invalid formula", err)
	}
}

func TestExecutorClosesRowsWhenScanFails(t *testing.T) {
	plan := testLogPlan(t)
	rows := &fakeRows{columns: []string{"value"}, data: [][]any{{1}}, scanErr: errors.New(errors.TypeInternal, errors.CodeInternal, "scan failed")}
	_, err := (Executor{Query: func(context.Context, string, ...any) (Rows, error) { return rows, nil }}).Execute(context.Background(), plan)
	if err == nil || !rows.closed {
		t.Fatalf("Execute() error = %v, rows closed = %v", err, rows.closed)
	}
}

func TestExecutorEnforcesStatementRowBudgetAndClosesRows(t *testing.T) {
	plan := testLogPlan(t)
	rows := &fakeRows{columns: []string{"value"}, data: [][]any{{1}, {2}, {3}}}
	_, err := (Executor{
		Config: ExecutorConfig{MaxRows: 2},
		Query:  func(context.Context, string, ...any) (Rows, error) { return rows, nil },
	}).Execute(context.Background(), plan)
	var queryErr *Error
	if !errors.As(err, &queryErr) || queryErr.Code != ErrorBudgetExceeded || !rows.closed {
		t.Fatalf("Execute() error = %v, rows closed = %v; want budget error and closed rows", err, rows.closed)
	}
}

func TestExecutorTrimsOverflowProbeAndAddsWarning(t *testing.T) {
	rows := &fakeRows{columns: []string{"value"}, data: [][]any{{1}, {2}, {3}}}
	result, err := (Executor{
		Config: ExecutorConfig{MaxRows: 10},
		Query:  func(context.Context, string, ...any) (Rows, error) { return rows, nil },
	}).executeStatement(context.Background(), Statement{
		Name: "A", SQL: "SELECT value", Columns: []ResultColumn{{Name: "value"}}, ResultLimit: 2,
	})
	if err != nil {
		t.Fatalf("executeStatement() error = %v", err)
	}
	if len(result.Rows) != 2 || !result.Truncated || len(result.Warnings) != 1 {
		t.Fatalf("executeStatement() result = %#v, want two rows and a truncation warning", result)
	}
}

func TestExecutorPropagatesTimeoutToQueryer(t *testing.T) {
	plan := testLogPlan(t)
	_, err := (Executor{
		Config: ExecutorConfig{Timeout: time.Millisecond},
		Query: func(ctx context.Context, _ string, _ ...any) (Rows, error) {
			<-ctx.Done()
			return nil, ctx.Err()
		},
	}).Execute(context.Background(), plan)
	var queryErr *Error
	if !errors.As(err, &queryErr) || queryErr.Code != ErrorTimeout || queryErr.Field != "executor.timeout" {
		t.Fatalf("Execute() error = %v, want timeout domain error", err)
	}
}

func TestExecutorClassifiesDriverErrorAfterDeadlineAsTimeout(t *testing.T) {
	plan := testLogPlan(t)
	_, err := (Executor{
		Config: ExecutorConfig{Timeout: time.Millisecond},
		Query: func(ctx context.Context, _ string, _ ...any) (Rows, error) {
			<-ctx.Done()
			return nil, errors.New(errors.TypeInternal, errors.CodeInternal, "driver query interrupted")
		},
	}).Execute(context.Background(), plan)
	var queryErr *Error
	if !errors.As(err, &queryErr) || queryErr.Code != ErrorTimeout {
		t.Fatalf("Execute() error = %v, want timeout domain error", err)
	}
}

func TestExecutorHonorsCallerCancellation(t *testing.T) {
	plan := testLogPlan(t)
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	_, err := (Executor{Query: func(ctx context.Context, _ string, _ ...any) (Rows, error) { return nil, ctx.Err() }}).Execute(ctx, plan)
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("Execute() error = %v, want context cancellation", err)
	}
}

func testLogPlan(t *testing.T) Plan {
	t.Helper()
	plan, err := (DefaultPlanner{}).Plan(Request{
		Range: TimeRange{StartMS: 1, EndMS: 2}, ResultType: ResultScalar,
		Queries: []Query{LogQuery{Common: CommonQuery{Name: "A"}, Aggregation: LogAggregateCount}},
	})
	if err != nil {
		t.Fatalf("Plan() error = %v", err)
	}
	return plan
}

type fakeRows struct {
	columns []string
	data    [][]any
	index   int
	scanErr error
	closed  bool
}

func (r *fakeRows) Columns() []string { return r.columns }

func (r *fakeRows) Next() bool {
	if r.index >= len(r.data) {
		return false
	}
	r.index++
	return true
}

func (r *fakeRows) Scan(destinations ...any) error {
	if r.scanErr != nil {
		return r.scanErr
	}
	for index, value := range r.data[r.index-1] {
		*destinations[index].(*any) = value
	}
	return nil
}

func (r *fakeRows) Err() error   { return nil }
func (r *fakeRows) Close() error { r.closed = true; return nil }
