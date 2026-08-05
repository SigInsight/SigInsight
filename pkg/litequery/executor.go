package litequery

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/SigNoz/signoz/pkg/errors"
)

// Rows is the small driver boundary used by Executor. ClickHouse adapters can
// forward driver.Rows directly; tests use an in-memory implementation.
type Rows interface {
	Columns() []string
	Next() bool
	Scan(...any) error
	Err() error
	Close() error
}

type QueryFunc func(context.Context, string, ...any) (Rows, error)

type ExecutorConfig struct {
	Timeout       time.Duration
	MaxConcurrent int
	MaxRows       int
}

type Executor struct {
	Compiler Compiler
	Query    QueryFunc
	Config   ExecutorConfig
}

type QueryResult struct {
	Name      string
	Columns   []ResultColumn
	Rows      [][]any
	Warnings  []string
	PageInfo  *PageInfo
	Truncated bool
}

type PageInfo struct {
	Limit       uint32
	Offset      uint32
	Returned    uint32
	HasNextPage bool
}

type ExecutionResult struct {
	Queries  []QueryResult
	Warnings []string
	Duration time.Duration
}

func (e Executor) Execute(ctx context.Context, plan Plan) (ExecutionResult, error) {
	if e.Query == nil {
		return ExecutionResult{}, newError(ErrorInvalidRequest, "executor.query", "query function is required")
	}
	if e.Compiler.Catalog == nil {
		e.Compiler = NewCompiler(nil)
	}
	if e.Config.Timeout <= 0 {
		e.Config.Timeout = 30 * time.Second
	}
	if e.Config.MaxConcurrent <= 0 {
		e.Config.MaxConcurrent = 4
	}
	if e.Config.MaxRows <= 0 {
		e.Config.MaxRows = 250_000
	}
	statements, err := e.Compiler.Compile(plan)
	if err != nil {
		return ExecutionResult{}, err
	}
	started := time.Now()
	queryCtx, cancel := context.WithTimeout(ctx, e.Config.Timeout)
	defer cancel()

	results := make([]QueryResult, len(statements))
	jobs := make(chan int)
	errCh := make(chan error, 1)
	var workers sync.WaitGroup
	workerCount := min(e.Config.MaxConcurrent, len(statements))
	for range workerCount {
		workers.Add(1)
		go func() {
			defer workers.Done()
			for index := range jobs {
				result, queryErr := e.executeStatement(queryCtx, statements[index])
				if queryErr != nil {
					select {
					case errCh <- queryErr:
						cancel()
					default:
					}
					return
				}
				results[index] = result
			}
		}()
	}
	for index := range statements {
		select {
		case jobs <- index:
		case err := <-errCh:
			close(jobs)
			workers.Wait()
			return ExecutionResult{}, normalizeExecutionError(queryCtx, err)
		case <-queryCtx.Done():
			close(jobs)
			workers.Wait()
			return ExecutionResult{}, normalizeExecutionError(queryCtx, queryCtx.Err())
		}
	}
	close(jobs)
	workers.Wait()
	select {
	case err := <-errCh:
		return ExecutionResult{}, normalizeExecutionError(queryCtx, err)
	default:
	}
	if err := queryCtx.Err(); err != nil {
		return ExecutionResult{}, normalizeExecutionError(queryCtx, err)
	}

	warnings := make([]string, 0)
	for _, result := range results {
		warnings = append(warnings, result.Warnings...)
	}
	if len(plan.Formulas) != 0 {
		formulaResults, err := evaluateFormulas(plan, results)
		if err != nil {
			return ExecutionResult{}, err
		}
		results = append(results, formulaResults...)
	}
	return ExecutionResult{Queries: results, Warnings: warnings, Duration: time.Since(started)}, nil
}

func (e Executor) executeStatement(ctx context.Context, statement Statement) (QueryResult, error) {
	if statement.Pagination != nil && (statement.Pagination.Limit == 0 || statement.ResultLimit != 0) {
		return QueryResult{}, newError(ErrorInvalidRequest, "executor.limit", "query %q has conflicting or invalid row-limit semantics", statement.Name)
	}
	rows, err := e.Query(ctx, statement.SQL, statement.Args...)
	if err != nil {
		return QueryResult{}, err
	}
	if rows == nil {
		return QueryResult{}, newError(ErrorInvalidRequest, "executor.rows", "query %q returned nil rows", statement.Name)
	}
	defer rows.Close()
	columns := statement.Columns
	if len(columns) == 0 {
		columns = make([]ResultColumn, len(rows.Columns()))
		for index, name := range rows.Columns() {
			columns[index] = ResultColumn{Name: name}
		}
	}
	result := QueryResult{Name: statement.Name, Columns: columns, Warnings: append([]string{}, statement.Warnings...)}
	for rows.Next() {
		if len(result.Rows) >= e.Config.MaxRows {
			return QueryResult{}, newError(ErrorBudgetExceeded, "executor.rows", "query %q returned more than %d rows", statement.Name, e.Config.MaxRows)
		}
		values := make([]any, len(columns))
		targets := make([]any, len(columns))
		for index := range values {
			targets[index] = &values[index]
		}
		if err := rows.Scan(targets...); err != nil {
			return QueryResult{}, errors.WrapInternalf(err, errors.CodeInternal, "failed to scan query %q", statement.Name)
		}
		result.Rows = append(result.Rows, values)
	}
	if err := rows.Err(); err != nil {
		return QueryResult{}, errors.WrapInternalf(err, errors.CodeInternal, "failed to read query %q", statement.Name)
	}
	if statement.Pagination != nil {
		limit := statement.Pagination.Limit
		hasNextPage := len(result.Rows) > int(limit)
		if hasNextPage {
			result.Rows = result.Rows[:limit]
		}
		result.PageInfo = &PageInfo{
			Limit:       limit,
			Offset:      statement.Pagination.Offset,
			Returned:    uint32(len(result.Rows)),
			HasNextPage: hasNextPage,
		}
	}
	if statement.ResultLimit != 0 && len(result.Rows) > int(statement.ResultLimit) {
		result.Rows = result.Rows[:statement.ResultLimit]
		result.Truncated = true
		result.Warnings = append(result.Warnings, fmt.Sprintf(
			"query %q returned more than %d rows; only the first %d rows are included",
			statement.Name,
			statement.ResultLimit,
			statement.ResultLimit,
		))
	}
	return result, nil
}

func normalizeExecutionError(ctx context.Context, err error) error {
	if errors.Is(err, context.DeadlineExceeded) || errors.Is(ctx.Err(), context.DeadlineExceeded) {
		return newError(ErrorTimeout, "executor.timeout", "query execution exceeded its deadline")
	}
	if errors.Is(err, context.Canceled) {
		return context.Canceled
	}
	if err == nil && errors.Is(ctx.Err(), context.Canceled) {
		return context.Canceled
	}
	return err
}

func evaluateFormulas(plan Plan, queryResults []QueryResult) ([]QueryResult, error) {
	resultByName := make(map[string]QueryResult, len(queryResults))
	for _, result := range queryResults {
		resultByName[result.Name] = result
	}
	// Planning validates the formula language and dependency graph before SQL
	// runs. Result units and group-by signatures only exist after statements
	// have produced rows, so bind the same parsed language to those concrete
	// inputs here. Reusing AnalyzeTypedFormulaSet keeps the two checks on one
	// grammar while preventing a query plan from silently accepting incompatible
	// result schemas.
	programs, err := AnalyzeTypedFormulaSet(plan.Formulas, formulaBindingsForResults(queryResults))
	if err != nil {
		return nil, err
	}
	for name, planned := range plan.TypedFormulas {
		program := programs[name]
		if planned == nil || program == nil || planned.Canonical() != program.Canonical() {
			return nil, newError(ErrorInvalidFormula, "formula.expression", "formula %q no longer matches its validated plan", name)
		}
	}
	formulaByName := make(map[string]QueryResult, len(plan.Formulas))
	pending := append([]Formula{}, plan.Formulas...)
	zeroDefaults := formulaZeroDefaults(plan)
	for len(pending) != 0 {
		progress := false
		remaining := make([]Formula, 0, len(pending))
		for _, formula := range pending {
			program := programs[formula.Name]
			if program == nil {
				return nil, newError(ErrorInvalidFormula, "formula.expression", "formula %q has no typed program", formula.Name)
			}
			if !formulaDependenciesReady(program.References(), resultByName) {
				remaining = append(remaining, formula)
				continue
			}
			keys, values, columns, err := formulaInputs(program.References(), resultByName, zeroDefaults)
			if err != nil {
				return nil, err
			}
			columns = append(columns, ResultColumn{Name: "value", ValueType: program.Type().Kind, Unit: program.Type().Unit})
			formulaResult := QueryResult{Name: formula.Name, Columns: columns}
			for _, key := range keys {
				value, err := program.Evaluate(values[key.text])
				if err != nil {
					return nil, err
				}
				row := append([]any{}, key.values...)
				if value.Missing {
					row = append(row, nil)
				} else if value.Type.Kind == FormulaValueBool {
					row = append(row, value.Bool)
				} else {
					row = append(row, value.Number)
				}
				formulaResult.Rows = append(formulaResult.Rows, row)
			}
			resultByName[formula.Name] = formulaResult
			formulaByName[formula.Name] = formulaResult
			progress = true
		}
		if !progress {
			return nil, newError(ErrorInvalidFormula, "formula.expression", "formula dependencies could not be resolved")
		}
		pending = remaining
	}
	formulas := make([]QueryResult, 0, len(plan.Formulas))
	for _, formula := range plan.Formulas {
		formulas = append(formulas, formulaByName[formula.Name])
	}
	return formulas, nil
}

func formulaDependenciesReady(references []string, results map[string]QueryResult) bool {
	for _, reference := range references {
		if _, ok := results[reference]; !ok {
			return false
		}
	}
	return true
}

type formulaKey struct {
	text   string
	values []any
}

func formulaInputs(references []string, results map[string]QueryResult, zeroDefaults map[string]bool) ([]formulaKey, map[string]map[string]FormulaValue, []ResultColumn, error) {
	keys := make([]formulaKey, 0)
	seen := make(map[string]struct{})
	values := make(map[string]map[string]FormulaValue)
	columns := make([]ResultColumn, 0)
	schemaSet := false
	for _, reference := range references {
		result, ok := results[reference]
		if !ok {
			return nil, nil, nil, newError(ErrorInvalidFormula, "formula.expression", "formula input %q is unavailable", reference)
		}
		if len(result.Columns) == 0 || result.Columns[len(result.Columns)-1].Name != "value" {
			return nil, nil, nil, newError(ErrorInvalidFormula, "formula.expression", "formula input %q has no value column", reference)
		}
		candidate := result.Columns[:len(result.Columns)-1]
		if !schemaSet {
			columns = append([]ResultColumn{}, candidate...)
			schemaSet = true
		} else if !sameResultColumns(columns, candidate) {
			return nil, nil, nil, newError(ErrorInvalidFormula, "formula.expression", "formula inputs must use identical timestamp and group columns")
		}
		valueType := queryResultFormulaType(result)
		for _, row := range result.Rows {
			if len(row) != len(result.Columns) || len(row) == 0 {
				return nil, nil, nil, newError(ErrorInvalidFormula, "formula.expression", "formula input %q returned an invalid row", reference)
			}
			keyValues := append([]any{}, row[:len(row)-1]...)
			key := formulaKey{text: alignmentKey(keyValues), values: keyValues}
			if _, exists := seen[key.text]; !exists {
				seen[key.text] = struct{}{}
				keys = append(keys, key)
			}
			if values[key.text] == nil {
				values[key.text] = make(map[string]FormulaValue)
			}
			value, err := formulaValueFromResult(row[len(row)-1], valueType)
			if err != nil {
				return nil, nil, nil, err
			}
			values[key.text][reference] = value
		}
	}
	for _, key := range keys {
		for _, reference := range references {
			if _, ok := values[key.text][reference]; ok {
				continue
			}
			result := results[reference]
			typ := queryResultFormulaType(result)
			if zeroDefaults[reference] && typ.Kind == FormulaValueNumber {
				values[key.text][reference] = FormulaValue{Type: typ}
			} else {
				values[key.text][reference] = FormulaValue{Type: typ, Missing: true}
			}
		}
	}
	return keys, values, columns, nil
}

func formulaValueFromResult(value any, typ FormulaStaticType) (FormulaValue, error) {
	if value == nil {
		return FormulaValue{Type: typ, Missing: true}, nil
	}
	if typ.Kind == FormulaValueBool {
		boolean, ok := value.(bool)
		if !ok {
			return FormulaValue{}, newError(ErrorInvalidFormula, "formula.expression", "formula input returned %T, want bool", value)
		}
		return FormulaValue{Type: typ, Bool: boolean}, nil
	}
	number, ok := numericValue(value)
	if !ok {
		return FormulaValue{}, newError(ErrorInvalidFormula, "formula.expression", "formula input returned %T, want number", value)
	}
	return checkedFormulaNumber(typ, number), nil
}

func queryResultFormulaType(result QueryResult) FormulaStaticType {
	if len(result.Columns) == 0 {
		return FormulaStaticType{Kind: FormulaValueNumber}
	}
	value := result.Columns[len(result.Columns)-1]
	typ := FormulaStaticType{Kind: value.ValueType, Unit: value.Unit}
	if typ.Kind == "" {
		typ.Kind = FormulaValueNumber
	}
	return typ
}

func formulaBindingsForResults(results []QueryResult) map[string]FormulaBinding {
	bindings := make(map[string]FormulaBinding, len(results))
	for _, result := range results {
		bindings[result.Name] = FormulaBinding{Type: queryResultFormulaType(result), SeriesSignature: formulaResultSeriesSignature(result)}
	}
	return bindings
}

func formulaResultSeriesSignature(result QueryResult) string {
	if len(result.Columns) < 2 {
		return ""
	}
	var key strings.Builder
	for _, column := range result.Columns[:len(result.Columns)-1] {
		key.WriteString(column.Name)
		key.WriteByte(':')
		if column.Field != nil {
			key.WriteString(string(column.Field.Context))
			key.WriteByte('.')
			key.WriteString(column.Field.Name)
		}
		key.WriteByte(';')
	}
	return key.String()
}

func formulaZeroDefaults(plan Plan) map[string]bool {
	defaults := make(map[string]bool, len(plan.Queries))
	for _, queryPlan := range plan.Queries {
		name := queryPlan.Query.GetCommon().Name
		switch query := queryPlan.Query.(type) {
		case LogQuery:
			defaults[name] = query.Aggregation == LogAggregateCount || query.Aggregation == LogAggregateSum
		case TraceQuery:
			defaults[name] = query.Aggregation == TraceAggregateCount
		case MetricQuery:
			defaults[name] = aggregationDefaultsMissingToZero(query.Aggregation.TimeAggregation)
		case MeterQuery:
			defaults[name] = aggregationDefaultsMissingToZero(query.Aggregation.TimeAggregation)
		}
	}
	return defaults
}

func aggregationDefaultsMissingToZero(aggregation TimeAggregation) bool {
	switch aggregation {
	case TimeAggregateCount, TimeAggregateSum, TimeAggregateRate, TimeAggregateIncrease:
		return true
	default:
		return false
	}
}

func sameResultColumns(left, right []ResultColumn) bool {
	if len(left) != len(right) {
		return false
	}
	for index := range left {
		if left[index].Name != right[index].Name {
			return false
		}
		if (left[index].Field == nil) != (right[index].Field == nil) {
			return false
		}
		if left[index].Field != nil && *left[index].Field != *right[index].Field {
			return false
		}
	}
	return true
}

func alignmentKey(values []any) string {
	var key strings.Builder
	for _, value := range values {
		encoded := fmt.Sprintf("%T:%v", value, value)
		key.WriteString(strconv.Itoa(len(encoded)))
		key.WriteByte(':')
		key.WriteString(encoded)
	}
	return key.String()
}

func numericValue(value any) (float64, bool) {
	switch current := value.(type) {
	case float64:
		return current, true
	case float32:
		return float64(current), true
	case int64:
		return float64(current), true
	case int32:
		return float64(current), true
	case int16:
		return float64(current), true
	case int8:
		return float64(current), true
	case int:
		return float64(current), true
	case uint64:
		return float64(current), true
	case uint32:
		return float64(current), true
	case uint16:
		return float64(current), true
	case uint8:
		return float64(current), true
	case uint:
		return float64(current), true
	default:
		return 0, false
	}
}
