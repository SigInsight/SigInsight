package model

import (
	"database/sql/driver"
	"encoding/json"
	"fmt"
	"github.com/SigNoz/signoz/pkg/types/timeseriestypes"

	"github.com/pkg/errors"
)

// AlertState denotes the state of an active alert.
type AlertState int

// The enum values are ordered by priority (lowest to highest).
// When determining overall rule state, higher numeric values take precedence.
const (
	StateInactive AlertState = iota
	StatePending
	StateRecovering
	StateFiring
	StateNoData
	StateDisabled
)

func (s AlertState) String() string {
	switch s {
	case StateInactive:
		return "inactive"
	case StatePending:
		return "pending"
	case StateFiring:
		return "firing"
	case StateNoData:
		return "nodata"
	case StateDisabled:
		return "disabled"
	case StateRecovering:
		return "recovering"
	}
	panic(errors.Errorf("unknown alert state: %d", s))
}

func (s AlertState) MarshalJSON() ([]byte, error) {
	return json.Marshal(s.String())
}

func (s *AlertState) UnmarshalJSON(b []byte) error {
	var v interface{}
	if err := json.Unmarshal(b, &v); err != nil {
		return err
	}
	switch value := v.(type) {
	case string:
		switch value {
		case "inactive":
			*s = StateInactive
		case "pending":
			*s = StatePending
		case "firing":
			*s = StateFiring
		case "nodata":
			*s = StateNoData
		case "disabled":
			*s = StateDisabled
		case "recovering":
			*s = StateRecovering
		default:
			*s = StateInactive
		}
		return nil
	default:
		return errors.New("invalid alert state")
	}
}

func (s *AlertState) Scan(value interface{}) error {
	v, ok := value.(string)
	if !ok {
		return errors.New("invalid alert state")
	}
	switch v {
	case "inactive":
		*s = StateInactive
	case "pending":
		*s = StatePending
	case "firing":
		*s = StateFiring
	case "nodata":
		*s = StateNoData
	case "disabled":
		*s = StateDisabled
	case "recovering":
		*s = StateRecovering
	}
	return nil
}

func (s *AlertState) Value() (driver.Value, error) {
	return s.String(), nil
}

type LabelsString string

func (l *LabelsString) MarshalJSON() ([]byte, error) {
	lbls := make(map[string]string)
	err := json.Unmarshal([]byte(*l), &lbls)
	if err != nil {
		return nil, err
	}
	return json.Marshal(lbls)
}

func (l *LabelsString) Scan(src interface{}) error {
	if data, ok := src.(string); ok {
		*l = LabelsString(data)
	}
	return nil
}

func (l LabelsString) String() string {
	return string(l)
}

type RuleStateTimeline struct {
	Items []RuleStateHistory `json:"items"`
	Total uint64             `json:"total"`
}

type RuleStateHistory struct {
	RuleID   string `json:"ruleID" ch:"rule_id"`
	RuleName string `json:"ruleName" ch:"rule_name"`
	// One of ["normal", "firing"]
	OverallState        AlertState `json:"overallState" ch:"overall_state"`
	OverallStateChanged bool       `json:"overallStateChanged" ch:"overall_state_changed"`
	// One of ["normal", "firing", "no_data", "muted"]
	State        AlertState   `json:"state" ch:"state"`
	StateChanged bool         `json:"stateChanged" ch:"state_changed"`
	UnixMilli    int64        `json:"unixMilli" ch:"unix_milli"`
	Labels       LabelsString `json:"labels" ch:"labels"`
	Fingerprint  uint64       `json:"fingerprint" ch:"fingerprint"`
	Value        float64      `json:"value" ch:"value"`

	RelatedTracesLink string `json:"relatedTracesLink"`
	RelatedLogsLink   string `json:"relatedLogsLink"`
}

type QueryRuleStateHistory struct {
	Start  int64  `json:"start"`
	End    int64  `json:"end"`
	State  string `json:"state"`
	Offset int64  `json:"offset"`
	Limit  int64  `json:"limit"`
	Order  string `json:"order"`
}

func (r *QueryRuleStateHistory) Validate() error {
	if r.Start == 0 || r.End == 0 {
		return fmt.Errorf("start and end are required")
	}
	if r.Offset < 0 || r.Limit < 0 {
		return fmt.Errorf("offset and limit must be greater than 0")
	}
	if r.Order != "asc" && r.Order != "desc" {
		return fmt.Errorf("order must be asc or desc")
	}
	return nil
}

type RuleStateHistoryContributor struct {
	Fingerprint       uint64       `json:"fingerprint" ch:"fingerprint"`
	Labels            LabelsString `json:"labels" ch:"labels"`
	Count             uint64       `json:"count" ch:"count"`
	RelatedTracesLink string       `json:"relatedTracesLink"`
	RelatedLogsLink   string       `json:"relatedLogsLink"`
}

type RuleStateTransition struct {
	RuleID         string     `json:"ruleID" ch:"rule_id"`
	State          AlertState `json:"state" ch:"state"`
	FiringTime     int64      `json:"firingTime" ch:"firing_time"`
	ResolutionTime int64      `json:"resolutionTime" ch:"resolution_time"`
}

type ReleStateItem struct {
	State AlertState `json:"state"`
	Start int64      `json:"start"`
	End   int64      `json:"end"`
}

type Stats struct {
	TotalCurrentTriggers           uint64                  `json:"totalCurrentTriggers"`
	TotalPastTriggers              uint64                  `json:"totalPastTriggers"`
	CurrentTriggersSeries          *timeseriestypes.Series `json:"currentTriggersSeries"`
	PastTriggersSeries             *timeseriestypes.Series `json:"pastTriggersSeries"`
	CurrentAvgResolutionTime       string                  `json:"currentAvgResolutionTime"`
	PastAvgResolutionTime          string                  `json:"pastAvgResolutionTime"`
	CurrentAvgResolutionTimeSeries *timeseriestypes.Series `json:"currentAvgResolutionTimeSeries"`
	PastAvgResolutionTimeSeries    *timeseriestypes.Series `json:"pastAvgResolutionTimeSeries"`
}
