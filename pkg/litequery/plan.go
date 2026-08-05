package litequery

// Plan is the validated, storage-independent form consumed by signal-specific
// compilers in later milestones. It has already selected the signal compiler
// boundary, but has not resolved fields or generated SQL.
type Plan struct {
	Range      TimeRange
	ResultType ResultType
	StepMS     int64
	Queries    []QueryPlan
	Formulas   []Formula
	// TypedFormulas validate the formula language and dependency graph before
	// SQL execution. The executor rebinds this same language to concrete result
	// units and group-by signatures, which are unavailable at planning time.
	TypedFormulas map[string]*TypedFormula
}

type QueryPlan struct {
	Signal Signal
	Query  Query
}

type Planner interface {
	Plan(Request) (Plan, error)
}

type DefaultPlanner struct {
	Limits Limits
}

func (p DefaultPlanner) Plan(request Request) (Plan, error) {
	limits := p.Limits
	if limits == (Limits{}) {
		limits = DefaultLimits()
	}
	if err := Validate(request, limits); err != nil {
		return Plan{}, err
	}
	queries := make([]QueryPlan, 0, len(request.Queries))
	for _, query := range request.Queries {
		queries = append(queries, QueryPlan{
			Signal: query.QuerySignal(),
			Query:  query,
		})
	}
	typedFormulas, err := AnalyzeTypedFormulaSet(request.Formulas, formulaBindingsForQueries(request.Queries))
	if err != nil {
		return Plan{}, err
	}
	return Plan{
		Range:         request.Range,
		ResultType:    request.ResultType,
		StepMS:        request.StepMS,
		Queries:       queries,
		Formulas:      request.Formulas,
		TypedFormulas: typedFormulas,
	}, nil
}

func formulaBindingsForQueries(queries []Query) map[string]FormulaBinding {
	bindings := make(map[string]FormulaBinding, len(queries))
	for _, query := range queries {
		if query == nil {
			continue
		}
		bindings[query.GetCommon().Name] = FormulaBinding{Type: FormulaStaticType{Kind: FormulaValueNumber}}
	}
	return bindings
}
