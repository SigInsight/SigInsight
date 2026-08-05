package ruletypes

import (
	"encoding/json"
	"time"

	"github.com/SigNoz/signoz/pkg/errors"
	"github.com/SigNoz/signoz/pkg/valuer"
)

type EvaluationKind struct {
	valuer.String
}

var (
	RollingEvaluation    = EvaluationKind{valuer.NewString("rolling")}
	CumulativeEvaluation = EvaluationKind{valuer.NewString("cumulative")}
)

type Evaluation interface {
	NextWindowFor(curr time.Time) (time.Time, time.Time)
	GetFrequency() valuer.TextDuration
}

type RollingWindow struct {
	EvalWindow valuer.TextDuration `json:"evalWindow"`
	Frequency  valuer.TextDuration `json:"frequency"`
}

func (rollingWindow RollingWindow) Validate() error {
	if !rollingWindow.EvalWindow.IsPositive() {
		return errors.NewInvalidInputf(errors.CodeInvalidInput, "evalWindow must be greater than zero")
	}
	if !rollingWindow.Frequency.IsPositive() {
		return errors.NewInvalidInputf(errors.CodeInvalidInput, "frequency must be greater than zero")
	}
	if rollingWindow.Frequency.Duration() > rollingWindow.EvalWindow.Duration() {
		return errors.NewInvalidInputf(errors.CodeInvalidInput, "frequency must not exceed evalWindow")
	}
	return nil
}

func (rollingWindow RollingWindow) NextWindowFor(curr time.Time) (time.Time, time.Time) {
	return curr.Add(-rollingWindow.EvalWindow.Duration()), curr
}

func (rollingWindow RollingWindow) GetFrequency() valuer.TextDuration {
	return rollingWindow.Frequency
}

type CumulativeWindow struct {
	// Period is deliberately limited to fixed calendar boundaries. It is not a
	// duration measured back from now: 1d and 7d begin at local midnight, so
	// they retain their correct meaning across DST changes.
	Period    CumulativePeriod    `json:"period"`
	Frequency valuer.TextDuration `json:"frequency"`
	Timezone  string              `json:"timezone"`
}

// CumulativePeriod is a calendar-boundary selector, not a generic duration.
// Keeping it separate from TextDuration makes unsupported periods such as a
// month impossible to reinterpret as an arbitrary number of hours.
type CumulativePeriod string

const (
	CumulativePeriodHour CumulativePeriod = "1h"
	CumulativePeriodDay  CumulativePeriod = "1d"
	CumulativePeriodWeek CumulativePeriod = "7d"
)

func (period CumulativePeriod) duration() time.Duration {
	switch period {
	case CumulativePeriodHour:
		return time.Hour
	case CumulativePeriodDay:
		return 24 * time.Hour
	case CumulativePeriodWeek:
		return 7 * 24 * time.Hour
	default:
		return 0
	}
}

func (cumulativeWindow CumulativeWindow) Validate() error {
	switch cumulativeWindow.Period {
	case CumulativePeriodHour, CumulativePeriodDay, CumulativePeriodWeek:
	default:
		return errors.NewInvalidInputf(errors.CodeInvalidInput, "cumulative period must be one of 1h, 1d, or 7d")
	}
	if cumulativeWindow.Timezone == "" {
		return errors.NewInvalidInputf(errors.CodeInvalidInput, "timezone is required")
	}
	if _, err := time.LoadLocation(cumulativeWindow.Timezone); err != nil {
		return errors.NewInvalidInputf(errors.CodeInvalidInput, "timezone is invalid")
	}
	if !cumulativeWindow.Frequency.IsPositive() {
		return errors.NewInvalidInputf(errors.CodeInvalidInput, "frequency must be greater than zero")
	}
	if cumulativeWindow.Frequency.Duration() > cumulativeWindow.Period.duration() {
		return errors.NewInvalidInputf(errors.CodeInvalidInput, "frequency must not exceed cumulative period")
	}
	return nil
}

func (cumulativeWindow CumulativeWindow) NextWindowFor(curr time.Time) (time.Time, time.Time) {
	loc, err := time.LoadLocation(cumulativeWindow.Timezone)
	if err != nil {
		loc = time.UTC
	}
	local := curr.In(loc)
	var start time.Time
	switch cumulativeWindow.Period {
	case CumulativePeriodHour:
		start = time.Date(local.Year(), local.Month(), local.Day(), local.Hour(), 0, 0, 0, loc)
	case CumulativePeriodDay:
		start = time.Date(local.Year(), local.Month(), local.Day(), 0, 0, 0, 0, loc)
	case CumulativePeriodWeek:
		daysSinceMonday := (int(local.Weekday()) + 6) % 7
		day := local.AddDate(0, 0, -daysSinceMonday)
		start = time.Date(day.Year(), day.Month(), day.Day(), 0, 0, 0, 0, loc)
	default:
		start = local
	}
	return start.UTC(), curr.UTC()
}

func (cumulativeWindow CumulativeWindow) GetFrequency() valuer.TextDuration {
	return cumulativeWindow.Frequency
}

type EvaluationEnvelope struct {
	Kind EvaluationKind `json:"kind"`
	Spec any            `json:"spec"`
}

func (e *EvaluationEnvelope) UnmarshalJSON(data []byte) error {
	var raw map[string]json.RawMessage
	if err := json.Unmarshal(data, &raw); err != nil {
		return errors.NewInvalidInputf(errors.CodeInvalidInput, "failed to unmarshal evaluation: %v", err)
	}
	if err := json.Unmarshal(raw["kind"], &e.Kind); err != nil {
		return errors.NewInvalidInputf(errors.CodeInvalidInput, "failed to unmarshal evaluation kind: %v", err)
	}
	switch e.Kind {
	case RollingEvaluation:
		var rollingWindow RollingWindow
		if err := unmarshalEvaluationSpec(raw["spec"], &rollingWindow, "evalWindow", "frequency"); err != nil {
			return errors.NewInvalidInputf(errors.CodeInvalidInput, "failed to unmarshal rolling window: %v", err)
		}
		err := rollingWindow.Validate()
		if err != nil {
			return errors.NewInvalidInputf(errors.CodeInvalidInput, "failed to validate rolling window: %v", err)
		}
		e.Spec = rollingWindow
	case CumulativeEvaluation:
		var cumulativeWindow CumulativeWindow
		if err := unmarshalEvaluationSpec(raw["spec"], &cumulativeWindow, "period", "frequency", "timezone"); err != nil {
			return errors.NewInvalidInputf(errors.CodeInvalidInput, "failed to unmarshal cumulative window: %v", err)
		}
		err := cumulativeWindow.Validate()
		if err != nil {
			return errors.NewInvalidInputf(errors.CodeInvalidInput, "failed to validate cumulative window: %v", err)
		}
		e.Spec = cumulativeWindow

	default:
		return errors.NewInvalidInputf(errors.CodeUnsupported, "unknown evaluation kind")
	}

	return nil
}

func unmarshalEvaluationSpec(raw json.RawMessage, target any, allowed ...string) error {
	if len(raw) == 0 {
		return errors.NewInvalidInputf(errors.CodeInvalidInput, "evaluation spec is required")
	}
	var values map[string]json.RawMessage
	if err := json.Unmarshal(raw, &values); err != nil {
		return err
	}
	allowedSet := make(map[string]struct{}, len(allowed))
	for _, field := range allowed {
		allowedSet[field] = struct{}{}
	}
	for field := range values {
		if _, ok := allowedSet[field]; !ok {
			return errors.NewInvalidInputf(errors.CodeInvalidInput, "unsupported evaluation field %q", field)
		}
	}
	return json.Unmarshal(raw, target)
}

func (e *EvaluationEnvelope) GetEvaluation() (Evaluation, error) {
	if e == nil {
		return nil, errors.NewInvalidInputf(errors.CodeInvalidInput, "evaluation is required")
	}

	switch e.Kind {
	case RollingEvaluation:
		if rolling, ok := e.Spec.(RollingWindow); ok {
			return rolling, nil
		}
	case CumulativeEvaluation:
		if cumulative, ok := e.Spec.(CumulativeWindow); ok {
			return cumulative, nil
		}
	default:
		return nil, errors.NewInvalidInputf(errors.CodeUnsupported, "unknown evaluation kind")
	}
	return nil, errors.NewInvalidInputf(errors.CodeUnsupported, "unknown evaluation kind")
}
