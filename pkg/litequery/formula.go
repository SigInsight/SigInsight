package litequery

// validateFormulas shares the typed parser used by the executor. At request
// validation time all source queries are numeric aggregate values; units and
// result-column signatures are enriched by later planner/executor stages.
func validateFormulas(formulas []Formula, queryNames map[string]struct{}) error {
	bindings := make(map[string]FormulaBinding, len(queryNames))
	for name := range queryNames {
		bindings[name] = FormulaBinding{Type: FormulaStaticType{Kind: FormulaValueNumber}}
	}
	_, err := AnalyzeTypedFormulaSet(formulas, bindings)
	return err
}
