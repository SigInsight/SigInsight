package querybuildertypesv5

import "github.com/SigNoz/signoz/pkg/valuer"

type QueryType struct {
	valuer.String
}

var (
	QueryTypeUnknown = QueryType{valuer.NewString("unknown")}
	QueryTypeBuilder = QueryType{valuer.NewString("builder_query")}
	QueryTypeFormula = QueryType{valuer.NewString("builder_formula")}
)

// Enum returns the acceptable values for QueryType.
func (QueryType) Enum() []any {
	return []any{
		QueryTypeBuilder,
		QueryTypeFormula,
	}
}
