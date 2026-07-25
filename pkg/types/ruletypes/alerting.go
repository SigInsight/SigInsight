package ruletypes

import (
	"encoding/json"
	"fmt"
	"github.com/SigNoz/signoz/pkg/types/timeseriestypes"
	"net/url"
	"strings"
	"time"

	"github.com/SigNoz/signoz/pkg/query-service/model"
	"github.com/SigNoz/signoz/pkg/query-service/model/querytypes"
	"github.com/SigNoz/signoz/pkg/query-service/utils/labels"
	qbtypes "github.com/SigNoz/signoz/pkg/types/querybuildertypes/querybuildertypesv5"
)

// this file contains common structs and methods used by
// rule engine

const (
	// how long before re-sending the alert
	ResolvedRetention = 15 * time.Minute
	TestAlertPostFix  = "_TEST_ALERT"
)

type RuleType string

const (
	RuleTypeThreshold = "threshold_rule"
	RuleTypeProm      = "promql_rule"
)

type RuleHealth string

const (
	HealthUnknown RuleHealth = "unknown"
	HealthGood    RuleHealth = "ok"
	HealthBad     RuleHealth = "err"
)

type Alert struct {
	State model.AlertState

	Labels      labels.BaseLabels
	Annotations labels.BaseLabels

	QueryResultLables labels.BaseLabels

	GeneratorURL string

	// list of preferred receivers, e.g. slack
	Receivers []string

	Value      float64
	ActiveAt   time.Time
	FiredAt    time.Time
	ResolvedAt time.Time
	LastSentAt time.Time
	ValidUntil time.Time

	Missing      bool
	IsRecovering bool
}

func (a *Alert) NeedsSending(ts time.Time, resendDelay time.Duration) bool {
	if a.State == model.StatePending {
		return false
	}

	// if an alert has been resolved since the last send, resend it
	if a.ResolvedAt.After(a.LastSentAt) {
		return true
	}

	return a.LastSentAt.Add(resendDelay).Before(ts)
}

type NamedAlert struct {
	Name string
	*Alert
}

type CompareOp string

const (
	CompareOpNone      CompareOp = "0"
	ValueIsAbove       CompareOp = "1"
	ValueIsBelow       CompareOp = "2"
	ValueIsEq          CompareOp = "3"
	ValueIsNotEq       CompareOp = "4"
	ValueAboveOrEq     CompareOp = "5"
	ValueBelowOrEq     CompareOp = "6"
	ValueOutsideBounds CompareOp = "7"
)

type MatchType string

const (
	MatchTypeNone MatchType = "0"
	AtleastOnce   MatchType = "1"
	AllTheTimes   MatchType = "2"
	OnAverage     MatchType = "3"
	InTotal       MatchType = "4"
	Last          MatchType = "5"
)

type RuleCondition struct {
	CompositeQuery    *CompositeQuery    `json:"compositeQuery,omitempty"`
	CompareOp         CompareOp          `json:"op,omitempty"`
	Target            *float64           `json:"target,omitempty"`
	AlertOnAbsent     bool               `json:"alertOnAbsent,omitempty"`
	AbsentFor         uint64             `json:"absentFor,omitempty"`
	MatchType         MatchType          `json:"matchType,omitempty"`
	TargetUnit        string             `json:"targetUnit,omitempty"`
	Algorithm         string             `json:"algorithm,omitempty"`
	Seasonality       string             `json:"seasonality,omitempty"`
	SelectedQuery     string             `json:"selectedQueryName,omitempty"`
	RequireMinPoints  bool               `json:"requireMinPoints,omitempty"`
	RequiredNumPoints int                `json:"requiredNumPoints,omitempty"`
	Thresholds        *RuleThresholdData `json:"thresholds,omitempty"`
}

// CompositeQuery is the alert-rule query entity. Query execution is V5-only,
// while the remaining metadata controls alert formatting and presentation.
type CompositeQuery struct {
	Queries   []qbtypes.QueryEnvelope `json:"queries"`
	PanelType querytypes.PanelType    `json:"panelType"`
	QueryType querytypes.QueryType    `json:"queryType"`
	Unit      string                  `json:"unit,omitempty"`
	FillGaps  bool                    `json:"fillGaps,omitempty"`
}

func (c *CompositeQuery) UnmarshalJSON(data []byte) error {
	var fields map[string]json.RawMessage
	if err := json.Unmarshal(data, &fields); err != nil {
		return err
	}

	validFields := map[string]struct{}{
		"queries":   {},
		"panelType": {},
		"queryType": {},
		"unit":      {},
		"fillGaps":  {},
	}
	for field := range fields {
		if _, ok := validFields[field]; !ok {
			return fmt.Errorf("unsupported alert composite query field %q", field)
		}
	}

	type alias CompositeQuery
	return json.Unmarshal(data, (*alias)(c))
}

func (rc *RuleCondition) GetSelectedQueryName() string {
	if rc == nil {
		return ""
	}
	return rc.SelectedQuery
}

func (rc *RuleCondition) IsValid() bool {
	if rc == nil || rc.CompositeQuery == nil || len(rc.CompositeQuery.Queries) == 0 {
		return false
	}

	if rc.QueryType() == querytypes.QueryTypeBuilder {
		if rc.Thresholds == nil {
			return false
		}
	}
	return true
}

// ShouldEval checks if the further series should be evaluated at all for alerts.
func (rc *RuleCondition) ShouldEval(series *timeseriestypes.Series) bool {
	if rc == nil {
		return true
	}
	return !rc.RequireMinPoints || len(series.Points) >= rc.RequiredNumPoints
}

// QueryType is a shorthand method to get query type
func (rc *RuleCondition) QueryType() querytypes.QueryType {
	if rc.CompositeQuery != nil {
		return rc.CompositeQuery.QueryType
	}
	return querytypes.QueryTypeUnknown
}

// String is useful in printing rule condition in logs
func (rc *RuleCondition) String() string {
	if rc == nil {
		return ""
	}
	data, _ := json.Marshal(*rc)
	return string(data)
}

// PrepareRuleGeneratorURL creates an appropriate url for the rule. The URL is
// sent in Slack messages as well as to other systems and allows backtracking
// to the rule definition from the third party systems.
func PrepareRuleGeneratorURL(ruleId string, source string) string {
	if source == "" {
		return source
	}

	// check if source is a valid url
	parsedSource, err := url.Parse(source)
	if err != nil {
		return ""
	}
	// since we capture window.location when a new rule is created
	// we end up with rulesource host:port/alerts/new. in this case
	// we want to replace new with rule id parameter

	hasNew := strings.LastIndex(source, "new")
	if hasNew > -1 {
		ruleURL := fmt.Sprintf("%sedit?ruleId=%s", source[0:hasNew], ruleId)
		return ruleURL
	}

	// The source contains the encoded query, start and end time
	// and other parameters. We don't want to include them in the generator URL
	// mainly to keep the URL short and lower the alert body contents
	// The generator URL with /alerts/edit?ruleId= is enough
	if parsedSource.Port() != "" {
		return fmt.Sprintf("%s://%s:%s/alerts/edit?ruleId=%s", parsedSource.Scheme, parsedSource.Hostname(), parsedSource.Port(), ruleId)
	}
	return fmt.Sprintf("%s://%s/alerts/edit?ruleId=%s", parsedSource.Scheme, parsedSource.Hostname(), ruleId)
}
