package litequery

import (
	"context"
	"fmt"
	"math"
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
	Name     string
	Columns  []ResultColumn
	Rows     [][]any
	Warnings []string
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
		formulaResults, formulaWarnings, err := evaluateFormulas(plan, results)
		if err != nil {
			return ExecutionResult{}, err
		}
		results = append(results, formulaResults...)
		warnings = append(warnings, formulaWarnings...)
	}
	return ExecutionResult{Queries: results, Warnings: warnings, Duration: time.Since(started)}, nil
}

func (e Executor) executeStatement(ctx context.Context, statement Statement) (QueryResult, error) {
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

func evaluateFormulas(plan Plan, queryResults []QueryResult) ([]QueryResult, []string, error) {
	resultByName := make(map[string]QueryResult, len(queryResults))
	for _, result := range queryResults {
		resultByName[result.Name] = result
	}
	formulaByName := make(map[string]QueryResult, len(plan.Formulas))
	warnings := make([]string, 0)
	pending := append([]Formula{}, plan.Formulas...)
	for len(pending) != 0 {
		progress := false
		remaining := make([]Formula, 0, len(pending))
		for _, formula := range pending {
			tokens, err := tokenizeFormula(formula.Expression)
			if err != nil {
				return nil, nil, err
			}
			if !formulaDependenciesReady(tokens, resultByName) {
				remaining = append(remaining, formula)
				continue
			}
			keys, values, columns, missing, err := formulaInputs(tokens, resultByName)
			if err != nil {
				return nil, nil, err
			}
			if missing {
				warnings = append(warnings, "formula "+formula.Name+" substituted zero for a missing aligned value")
			}
			formulaResult := QueryResult{Name: formula.Name, Columns: columns}
			for _, key := range keys {
				value, err := evaluateFormulaTokens(tokens, values[key.text])
				if err != nil {
					return nil, nil, err
				}
				row := append([]any{}, key.values...)
				row = append(row, value)
				formulaResult.Rows = append(formulaResult.Rows, row)
			}
			resultByName[formula.Name] = formulaResult
			formulaByName[formula.Name] = formulaResult
			progress = true
		}
		if !progress {
			return nil, nil, newError(ErrorInvalidFormula, "formula.expression", "formula dependencies could not be resolved")
		}
		pending = remaining
	}
	formulas := make([]QueryResult, 0, len(plan.Formulas))
	for _, formula := range plan.Formulas {
		formulas = append(formulas, formulaByName[formula.Name])
	}
	return formulas, warnings, nil
}

func formulaDependenciesReady(tokens []formulaToken, results map[string]QueryResult) bool {
	for _, token := range tokens {
		if token.kind == formulaIdentifier {
			if _, ok := results[token.value]; !ok {
				return false
			}
		}
	}
	return true
}

type formulaKey struct {
	text   string
	values []any
}

func formulaInputs(tokens []formulaToken, results map[string]QueryResult) ([]formulaKey, map[string]map[string]float64, []ResultColumn, bool, error) {
	keys := make([]formulaKey, 0)
	seen := make(map[string]struct{})
	values := make(map[string]map[string]float64)
	columns := []ResultColumn{{Name: "value"}}
	schemaSet := false
	missing := false
	dependencies := make([]string, 0)
	dependencySeen := make(map[string]struct{})
	for _, token := range tokens {
		if token.kind != formulaIdentifier {
			continue
		}
		result, ok := results[token.value]
		if !ok {
			continue
		}
		if _, ok := dependencySeen[token.value]; ok {
			continue
		}
		dependencySeen[token.value] = struct{}{}
		dependencies = append(dependencies, token.value)
		if len(result.Columns) == 0 || result.Columns[len(result.Columns)-1].Name != "value" {
			return nil, nil, nil, false, newError(ErrorInvalidFormula, "formula.expression", "formula input %q has no value column", token.value)
		}
		candidate := result.Columns[:len(result.Columns)-1]
		if !schemaSet {
			columns = append([]ResultColumn{}, candidate...)
			columns = append(columns, ResultColumn{Name: "value"})
			schemaSet = true
		} else if !sameResultColumns(columns[:len(columns)-1], candidate) {
			return nil, nil, nil, false, newError(ErrorInvalidFormula, "formula.expression", "formula inputs must use identical timestamp and group columns")
		}
		for _, row := range result.Rows {
			if len(row) != len(result.Columns) || len(row) == 0 {
				missing = true
				continue
			}
			keyValues := append([]any{}, row[:len(row)-1]...)
			key := formulaKey{text: alignmentKey(keyValues), values: keyValues}
			if _, exists := seen[key.text]; !exists {
				seen[key.text] = struct{}{}
				keys = append(keys, key)
			}
			if values[key.text] == nil {
				values[key.text] = make(map[string]float64)
			}
			value, ok := numericValue(row[len(row)-1])
			if !ok {
				missing = true
				continue
			}
			values[key.text][token.value] = value
		}
	}
	for _, key := range keys {
		for _, dependency := range dependencies {
			if _, ok := values[key.text][dependency]; !ok {
				missing = true
			}
		}
	}
	return keys, values, columns, missing, nil
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

func evaluateFormulaTokens(tokens []formulaToken, values map[string]float64) (float64, error) {
	position := 0
	value, err := parseFormulaExpression(tokens, &position, values)
	if err != nil || position != len(tokens) {
		if err != nil {
			return 0, err
		}
		return 0, newError(ErrorInvalidFormula, "formula.expression", "could not consume formula expression")
	}
	return value, nil
}

func parseFormulaExpression(tokens []formulaToken, position *int, values map[string]float64) (float64, error) {
	value, err := parseFormulaTerm(tokens, position, values)
	if err != nil {
		return 0, err
	}
	for *position < len(tokens) && tokens[*position].kind == formulaOperator && (tokens[*position].value == "+" || tokens[*position].value == "-") {
		op := tokens[*position].value
		*position++
		right, err := parseFormulaTerm(tokens, position, values)
		if err != nil {
			return 0, err
		}
		if op == "+" {
			value += right
		} else {
			value -= right
		}
	}
	return value, nil
}

func parseFormulaTerm(tokens []formulaToken, position *int, values map[string]float64) (float64, error) {
	value, err := parseFormulaFactor(tokens, position, values)
	if err != nil {
		return 0, err
	}
	for *position < len(tokens) && tokens[*position].kind == formulaOperator && (tokens[*position].value == "*" || tokens[*position].value == "/") {
		op := tokens[*position].value
		*position++
		right, err := parseFormulaFactor(tokens, position, values)
		if err != nil {
			return 0, err
		}
		if op == "*" {
			value *= right
		} else if right == 0 {
			value = math.NaN()
		} else {
			value /= right
		}
	}
	return value, nil
}

func parseFormulaFactor(tokens []formulaToken, position *int, values map[string]float64) (float64, error) {
	if *position >= len(tokens) {
		return 0, newError(ErrorInvalidFormula, "formula.expression", "missing formula operand")
	}
	token := tokens[*position]
	*position++
	switch token.kind {
	case formulaNumber:
		return strconv.ParseFloat(token.value, 64)
	case formulaIdentifier:
		return values[token.value], nil
	case formulaLeftParen:
		value, err := parseFormulaExpression(tokens, position, values)
		if err != nil {
			return 0, err
		}
		if *position >= len(tokens) || tokens[*position].kind != formulaRightParen {
			return 0, newError(ErrorInvalidFormula, "formula.expression", "missing closing parenthesis")
		}
		*position++
		return value, nil
	default:
		return 0, newError(ErrorInvalidFormula, "formula.expression", "unexpected token %q", token.value)
	}
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

func (k formulaKey) String() string { return strings.TrimSpace(k.text) }
