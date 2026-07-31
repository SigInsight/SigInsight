package litequery

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"
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
	if got := result.Queries[len(result.Queries)-1].Rows[0][0]; got != float64(6) {
		t.Fatalf("forward formula result = %#v, want 6", got)
	}
}

func TestExecutorClosesRowsWhenScanFails(t *testing.T) {
	plan := testLogPlan(t)
	rows := &fakeRows{columns: []string{"value"}, data: [][]any{{1}}, scanErr: errors.New("scan failed")}
	_, err := (Executor{Query: func(context.Context, string, ...any) (Rows, error) { return rows, nil }}).Execute(context.Background(), plan)
	if err == nil || !rows.closed {
		t.Fatalf("Execute() error = %v, rows closed = %v", err, rows.closed)
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
	if !errors.As(err, &queryErr) || queryErr.Field != "executor.timeout" {
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
