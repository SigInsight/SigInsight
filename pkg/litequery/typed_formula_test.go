package litequery

import (
	"errors"
	"testing"
)

func TestAnalyzeTypedFormulaCanonicalizesBooleanPrecedence(t *testing.T) {
	formula, err := AnalyzeTypedFormula(
		"(A > 10 or B < 5) and not C = 0",
		map[string]FormulaBinding{
			"A": {Type: FormulaStaticType{Kind: FormulaValueNumber}},
			"B": {Type: FormulaStaticType{Kind: FormulaValueNumber}},
			"C": {Type: FormulaStaticType{Kind: FormulaValueNumber}},
		},
	)
	if err != nil {
		t.Fatalf("AnalyzeTypedFormula() error = %v", err)
	}
	if got, want := formula.Canonical(), "(A > 10 OR B < 5) AND NOT (C = 0)"; got != want {
		t.Fatalf("Canonical() = %q, want %q", got, want)
	}
	if got, want := formula.Type(), (FormulaStaticType{Kind: FormulaValueBool}); got != want {
		t.Fatalf("Type() = %#v, want %#v", got, want)
	}
	if got, want := formula.References(), []string{"A", "B", "C"}; len(got) != len(want) || got[0] != want[0] || got[1] != want[1] || got[2] != want[2] {
		t.Fatalf("References() = %#v, want %#v", got, want)
	}
}

func TestAnalyzeTypedFormulaRejectsUnsupportedAliasesAndChainedComparison(t *testing.T) {
	bindings := map[string]FormulaBinding{
		"A": {Type: FormulaStaticType{Kind: FormulaValueNumber}},
		"B": {Type: FormulaStaticType{Kind: FormulaValueNumber}},
		"C": {Type: FormulaStaticType{Kind: FormulaValueNumber}},
	}
	for _, expression := range []string{
		"A == B",
		"A && B",
		"!A",
		"A < B < C",
		"A > 1 AND B",
	} {
		t.Run(expression, func(t *testing.T) {
			_, err := AnalyzeTypedFormula(expression, bindings)
			var queryErr *Error
			if !errors.As(err, &queryErr) || queryErr.Code != ErrorInvalidFormula {
				t.Fatalf("AnalyzeTypedFormula(%q) error = %v, want invalid formula", expression, err)
			}
		})
	}
}

func TestAnalyzeTypedFormulaChecksUnitsAndSeriesSignature(t *testing.T) {
	t.Run("compatible duration units retain left unit", func(t *testing.T) {
		formula, err := AnalyzeTypedFormula("A + B", map[string]FormulaBinding{
			"A": {Type: FormulaStaticType{Kind: FormulaValueNumber, Unit: "ms"}, SeriesSignature: "timestamp,service"},
			"B": {Type: FormulaStaticType{Kind: FormulaValueNumber, Unit: "s"}, SeriesSignature: "timestamp,service"},
		})
		if err != nil {
			t.Fatalf("AnalyzeTypedFormula() error = %v", err)
		}
		if got, want := formula.Type(), (FormulaStaticType{Kind: FormulaValueNumber, Unit: "ms"}); got != want {
			t.Fatalf("Type() = %#v, want %#v", got, want)
		}
		if got, want := formula.SeriesSignature(), "timestamp,service"; got != want {
			t.Fatalf("SeriesSignature() = %q, want %q", got, want)
		}
	})

	for _, test := range []struct {
		name       string
		expression string
		bindings   map[string]FormulaBinding
	}{
		{
			name:       "different dimensions cannot add",
			expression: "A + B",
			bindings: map[string]FormulaBinding{
				"A": {Type: FormulaStaticType{Kind: FormulaValueNumber, Unit: "ms"}},
				"B": {Type: FormulaStaticType{Kind: FormulaValueNumber, Unit: "bytes"}},
			},
		},
		{
			name:       "two unitful values cannot multiply",
			expression: "A * B",
			bindings: map[string]FormulaBinding{
				"A": {Type: FormulaStaticType{Kind: FormulaValueNumber, Unit: "ms"}},
				"B": {Type: FormulaStaticType{Kind: FormulaValueNumber, Unit: "s"}},
			},
		},
		{
			name:       "unitful divisor must be compatible",
			expression: "A / B",
			bindings: map[string]FormulaBinding{
				"A": {Type: FormulaStaticType{Kind: FormulaValueNumber, Unit: "bytes"}},
				"B": {Type: FormulaStaticType{Kind: FormulaValueNumber, Unit: "s"}},
			},
		},
		{
			name:       "series signatures must align",
			expression: "A - B",
			bindings: map[string]FormulaBinding{
				"A": {Type: FormulaStaticType{Kind: FormulaValueNumber}, SeriesSignature: "timestamp,service"},
				"B": {Type: FormulaStaticType{Kind: FormulaValueNumber}, SeriesSignature: "timestamp,host"},
			},
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			_, err := AnalyzeTypedFormula(test.expression, test.bindings)
			var queryErr *Error
			if !errors.As(err, &queryErr) || queryErr.Code != ErrorInvalidFormula {
				t.Fatalf("AnalyzeTypedFormula() error = %v, want invalid formula", err)
			}
		})
	}
}

func TestAnalyzeTypedFormulaSetSupportsForwardReferencesAndRejectsCycles(t *testing.T) {
	bindings := map[string]FormulaBinding{
		"A": {Type: FormulaStaticType{Kind: FormulaValueNumber}, SeriesSignature: "timestamp"},
	}
	formulas, err := AnalyzeTypedFormulaSet([]Formula{
		{Name: "F1", Expression: "F2 > 5"},
		{Name: "F2", Expression: "A * 2"},
	}, bindings)
	if err != nil {
		t.Fatalf("AnalyzeTypedFormulaSet() error = %v", err)
	}
	if got, want := formulas["F1"].Type(), (FormulaStaticType{Kind: FormulaValueBool}); got != want {
		t.Fatalf("F1 Type() = %#v, want %#v", got, want)
	}

	_, err = AnalyzeTypedFormulaSet([]Formula{
		{Name: "F1", Expression: "F2 + A"},
		{Name: "F2", Expression: "F1 + A"},
	}, bindings)
	var queryErr *Error
	if !errors.As(err, &queryErr) || queryErr.Code != ErrorInvalidFormula {
		t.Fatalf("AnalyzeTypedFormulaSet(cycle) error = %v, want invalid formula", err)
	}
}

func TestTypedFormulaEvaluateConvertsUnitsAndPropagatesMissing(t *testing.T) {
	t.Run("converts compatible units before addition", func(t *testing.T) {
		formula, err := AnalyzeTypedFormula("A + B", map[string]FormulaBinding{
			"A": {Type: FormulaStaticType{Kind: FormulaValueNumber, Unit: "ms"}},
			"B": {Type: FormulaStaticType{Kind: FormulaValueNumber, Unit: "s"}},
		})
		if err != nil {
			t.Fatalf("AnalyzeTypedFormula() error = %v", err)
		}
		value, err := formula.Evaluate(map[string]FormulaValue{
			"A": {Type: FormulaStaticType{Kind: FormulaValueNumber, Unit: "ms"}, Number: 1_000},
			"B": {Type: FormulaStaticType{Kind: FormulaValueNumber, Unit: "s"}, Number: 1},
		})
		if err != nil {
			t.Fatalf("Evaluate() error = %v", err)
		}
		if value.Missing || value.Number != 2_000 || value.Type.Unit != "ms" {
			t.Fatalf("Evaluate() = %#v, want 2000ms", value)
		}
	})

	t.Run("boolean logic does not short circuit missing", func(t *testing.T) {
		formula, err := AnalyzeTypedFormula("A > 0 OR B > 0", map[string]FormulaBinding{
			"A": {Type: FormulaStaticType{Kind: FormulaValueNumber}},
			"B": {Type: FormulaStaticType{Kind: FormulaValueNumber}},
		})
		if err != nil {
			t.Fatalf("AnalyzeTypedFormula() error = %v", err)
		}
		value, err := formula.Evaluate(map[string]FormulaValue{
			"A": {Type: FormulaStaticType{Kind: FormulaValueNumber}, Number: 1},
			"B": {Type: FormulaStaticType{Kind: FormulaValueNumber}, Missing: true},
		})
		if err != nil {
			t.Fatalf("Evaluate() error = %v", err)
		}
		if !value.Missing || value.Type.Kind != FormulaValueBool {
			t.Fatalf("Evaluate() = %#v, want missing bool", value)
		}
	})

	t.Run("division by runtime zero is missing", func(t *testing.T) {
		formula, err := AnalyzeTypedFormula("A / B", map[string]FormulaBinding{
			"A": {Type: FormulaStaticType{Kind: FormulaValueNumber}},
			"B": {Type: FormulaStaticType{Kind: FormulaValueNumber}},
		})
		if err != nil {
			t.Fatalf("AnalyzeTypedFormula() error = %v", err)
		}
		value, err := formula.Evaluate(map[string]FormulaValue{
			"A": {Type: FormulaStaticType{Kind: FormulaValueNumber}, Number: 5},
			"B": {Type: FormulaStaticType{Kind: FormulaValueNumber}, Number: 0},
		})
		if err != nil {
			t.Fatalf("Evaluate() error = %v", err)
		}
		if !value.Missing || value.Type.Kind != FormulaValueNumber {
			t.Fatalf("Evaluate() = %#v, want missing number", value)
		}
	})
}

func FuzzAnalyzeTypedFormula(f *testing.F) {
	for _, expression := range []string{
		"A + B * 2",
		"(A > 10 OR B < 5) AND NOT C = 0",
		"A / 0",
		"A < B < C",
		"A && B",
	} {
		f.Add(expression)
	}
	bindings := map[string]FormulaBinding{
		"A": {Type: FormulaStaticType{Kind: FormulaValueNumber, Unit: "ms"}, SeriesSignature: "timestamp"},
		"B": {Type: FormulaStaticType{Kind: FormulaValueNumber, Unit: "s"}, SeriesSignature: "timestamp"},
		"C": {Type: FormulaStaticType{Kind: FormulaValueNumber}, SeriesSignature: "timestamp"},
	}
	f.Fuzz(func(t *testing.T, expression string) {
		formula, err := AnalyzeTypedFormula(expression, bindings)
		if err == nil {
			_, _ = formula.Evaluate(map[string]FormulaValue{
				"A": {Type: FormulaStaticType{Kind: FormulaValueNumber, Unit: "ms"}, Number: 1},
				"B": {Type: FormulaStaticType{Kind: FormulaValueNumber, Unit: "s"}, Number: 1},
				"C": {Type: FormulaStaticType{Kind: FormulaValueNumber}, Number: 1},
			})
		}
	})
}
