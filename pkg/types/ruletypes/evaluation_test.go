package ruletypes

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/SigNoz/signoz/pkg/valuer"
)

func TestRollingWindowValidateRequiresPositiveCompatibleFrequency(t *testing.T) {
	tests := []struct {
		name  string
		value RollingWindow
		valid bool
	}{
		{"valid", RollingWindow{EvalWindow: valuer.MustParseTextDuration("5m"), Frequency: valuer.MustParseTextDuration("1m")}, true},
		{"zero window", RollingWindow{Frequency: valuer.MustParseTextDuration("1m")}, false},
		{"zero frequency", RollingWindow{EvalWindow: valuer.MustParseTextDuration("5m")}, false},
		{"frequency exceeds window", RollingWindow{EvalWindow: valuer.MustParseTextDuration("1m"), Frequency: valuer.MustParseTextDuration("5m")}, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := tt.value.Validate(); (err == nil) != tt.valid {
				t.Fatalf("Validate() error = %v, valid = %v", err, tt.valid)
			}
		})
	}
}

func TestFixedCumulativeWindowUsesLocalPeriodBoundaries(t *testing.T) {
	tests := []struct {
		name     string
		period   string
		timezone string
		current  time.Time
		start    time.Time
	}{
		{
			name: "hour starts at local hour", period: "1h", timezone: "Asia/Shanghai",
			current: time.Date(2025, 3, 15, 6, 45, 0, 0, time.UTC),
			start:   time.Date(2025, 3, 15, 6, 0, 0, 0, time.UTC),
		},
		{
			name: "day starts at local midnight", period: "1d", timezone: "Asia/Shanghai",
			current: time.Date(2025, 3, 15, 6, 45, 0, 0, time.UTC),
			start:   time.Date(2025, 3, 14, 16, 0, 0, 0, time.UTC),
		},
		{
			name: "week starts Monday local midnight", period: "7d", timezone: "UTC",
			current: time.Date(2025, 3, 16, 12, 0, 0, 0, time.UTC),
			start:   time.Date(2025, 3, 10, 0, 0, 0, 0, time.UTC),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			window := CumulativeWindow{Period: CumulativePeriod(tt.period), Frequency: valuer.MustParseTextDuration("1m"), Timezone: tt.timezone}
			if err := window.Validate(); err != nil {
				t.Fatalf("Validate() error = %v", err)
			}
			start, end := window.NextWindowFor(tt.current)
			if !start.Equal(tt.start) || !end.Equal(tt.current) {
				t.Fatalf("NextWindowFor() = (%v, %v), want (%v, %v)", start, end, tt.start, tt.current)
			}
		})
	}
}

func TestFixedCumulativeWindowPreservesDSTCalendarBoundaries(t *testing.T) {
	window := CumulativeWindow{Period: CumulativePeriodDay, Frequency: valuer.MustParseTextDuration("1m"), Timezone: "America/New_York"}
	for _, tt := range []struct {
		name    string
		current time.Time
		start   time.Time
	}{
		{
			name: "spring forward", current: time.Date(2025, 3, 9, 7, 30, 0, 0, time.UTC),
			start: time.Date(2025, 3, 9, 5, 0, 0, 0, time.UTC),
		},
		{
			name: "fall back", current: time.Date(2025, 11, 2, 6, 30, 0, 0, time.UTC),
			start: time.Date(2025, 11, 2, 4, 0, 0, 0, time.UTC),
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			start, _ := window.NextWindowFor(tt.current)
			if !start.Equal(tt.start) {
				t.Fatalf("NextWindowFor() start = %v, want %v", start, tt.start)
			}
		})
	}
}

func TestCumulativeWindowRejectsUnsupportedPeriodsAndEvaluationScheduleJSON(t *testing.T) {
	for _, period := range []string{"30m", "1month", "0s"} {
		t.Run(period, func(t *testing.T) {
			window := CumulativeWindow{Period: CumulativePeriod(period), Frequency: valuer.MustParseTextDuration("1m"), Timezone: "UTC"}
			if err := window.Validate(); err == nil {
				t.Fatal("Validate() error = nil, want rejected period")
			}
		})
	}

	var evaluation EvaluationEnvelope
	err := json.Unmarshal([]byte(`{"kind":"cumulative","spec":{"schedule":{"type":"daily"},"frequency":"1m","timezone":"UTC"}}`), &evaluation)
	if err == nil {
		t.Fatal("Unmarshal() error = nil, want retired schedule rejection")
	}
}

func TestEvaluationEnvelopeDecodesOnlyFixedV3Shapes(t *testing.T) {
	for _, payload := range []string{
		`{"kind":"rolling","spec":{"evalWindow":"5m","frequency":"1m"}}`,
		`{"kind":"cumulative","spec":{"period":"7d","frequency":"5m","timezone":"Asia/Shanghai"}}`,
	} {
		var evaluation EvaluationEnvelope
		if err := json.Unmarshal([]byte(payload), &evaluation); err != nil {
			t.Fatalf("Unmarshal(%s) error = %v", payload, err)
		}
		if _, err := evaluation.GetEvaluation(); err != nil {
			t.Fatalf("GetEvaluation() error = %v", err)
		}
	}
}
