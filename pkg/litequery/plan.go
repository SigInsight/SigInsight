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
	return Plan{
		Range:      request.Range,
		ResultType: request.ResultType,
		StepMS:     request.StepMS,
		Queries:    queries,
		Formulas:   request.Formulas,
	}, nil
}
