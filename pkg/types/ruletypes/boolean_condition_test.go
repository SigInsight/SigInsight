package ruletypes

import (
	"testing"
	"time"

	"github.com/SigNoz/signoz/pkg/types/timeseriestypes"
	"github.com/SigNoz/signoz/pkg/valuer"
)

func TestBooleanRuleThresholdEvaluatesTypedBooleanPoints(t *testing.T) {
	trueValue, falseValue := true, false
	series := timeseriestypes.Series{
		Labels: map[string]string{"service": "api"},
		Points: []timeseriestypes.Point{
			{Timestamp: 1, BoolValue: &falseValue},
			{Timestamp: 2, BoolValue: &trueValue},
		},
	}
	tests := []struct {
		policy  Reduction
		matches bool
	}{
		{policy: ReductionAtLeastOnce, matches: true},
		{policy: ReductionAllTheTime, matches: false},
		{policy: ReductionLast, matches: true},
	}
	for _, tt := range tests {
		t.Run(string(tt.policy), func(t *testing.T) {
			threshold := BooleanRuleThreshold{Policy: tt.policy, Severity: "critical"}
			result, err := threshold.Eval(series, "", EvalData{})
			if err != nil {
				t.Fatalf("Eval() error = %v", err)
			}
			if tt.matches && (len(result) != 1 || result[0].BoolValue == nil || !*result[0].BoolValue) {
				t.Fatalf("Eval() = %#v, want a true boolean sample", result)
			}
			if !tt.matches && len(result) != 0 {
				t.Fatalf("Eval() = %#v, want no matching sample", result)
			}
		})
	}
}

func TestRuleConditionNoDataAfterPreservesSubMinuteV3Duration(t *testing.T) {
	condition := RuleCondition{DataQuality: DataQualityPolicy{
		AlertOnNoData: true,
		NoDataFor:     valuer.MustParseTextDuration("30s"),
	}}
	if got := condition.NoDataAfter(); got != 30*time.Second {
		t.Fatalf("NoDataAfter() = %v, want 30s", got)
	}
}

func TestBooleanRuleThresholdKeepsFalseDistinctForTestPreview(t *testing.T) {
	falseValue := false
	threshold := BooleanRuleThreshold{Policy: ReductionLast, Severity: "warning"}
	result, err := threshold.Eval(timeseriestypes.Series{Points: []timeseriestypes.Point{{Timestamp: 1, BoolValue: &falseValue}}}, "", EvalData{SendUnmatched: true})
	if err != nil {
		t.Fatalf("Eval() error = %v", err)
	}
	if len(result) != 1 || result[0].BoolValue == nil || *result[0].BoolValue || result[0].V != 0 {
		t.Fatalf("Eval() = %#v, want explicit false preview", result)
	}
}
