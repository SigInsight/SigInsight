package ruletypes

import (
	"encoding/json"
	"fmt"
	"net/url"
	"strings"
	"time"

	"github.com/SigNoz/signoz/pkg/errors"
	"github.com/SigNoz/signoz/pkg/query-service/model"
	"github.com/SigNoz/signoz/pkg/query-service/model/querytypes"
	"github.com/SigNoz/signoz/pkg/query-service/utils/labels"
	qbtypes "github.com/SigNoz/signoz/pkg/types/querybuildertypes/querybuildertypesv5"
	"github.com/SigNoz/signoz/pkg/types/timeseriestypes"
	"github.com/SigNoz/signoz/pkg/valuer"
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

// ConditionKind distinguishes the two outputs the lightweight query engine
// can safely hand to an alert evaluator. It is deliberately not inferred from
// a query name or a numeric sentinel value.
type ConditionKind string

const (
	ConditionKindNumeric ConditionKind = "numeric"
	ConditionKindBoolean ConditionKind = "boolean"
)

// Reduction is the bounded window reduction available to a numeric condition.
// Its wire values are descriptive so the v3 contract no longer leaks the old
// numeric MatchType enum.
type Reduction string

const (
	ReductionAtLeastOnce Reduction = "at_least_once"
	ReductionAllTheTime  Reduction = "all_the_time"
	ReductionAverage     Reduction = "average"
	ReductionTotal       Reduction = "total"
	ReductionLast        Reduction = "last"
)

// NumericOperator is intentionally the small Formula-compatible comparison
// set. Outside-bounds is expressed as a boolean formula instead.
type NumericOperator string

const (
	NumericOperatorEqual              NumericOperator = "eq"
	NumericOperatorNotEqual           NumericOperator = "neq"
	NumericOperatorGreaterThan        NumericOperator = "gt"
	NumericOperatorGreaterThanOrEqual NumericOperator = "gte"
	NumericOperatorLessThan           NumericOperator = "lt"
	NumericOperatorLessThanOrEqual    NumericOperator = "lte"
)

type DataQualityPolicy struct {
	AlertOnNoData bool                `json:"alertOnNoData,omitempty"`
	NoDataFor     valuer.TextDuration `json:"noDataFor,omitempty"`
	MinPoints     int                 `json:"minPoints,omitempty"`
}

func (policy DataQualityPolicy) Validate() error {
	if policy.MinPoints < 0 {
		return errors.NewInvalidInputf(errors.CodeInvalidInput, "minPoints must not be negative")
	}
	if policy.AlertOnNoData && !policy.NoDataFor.IsPositive() {
		return errors.NewInvalidInputf(errors.CodeInvalidInput, "noDataFor must be greater than zero when no-data alerting is enabled")
	}
	if !policy.AlertOnNoData && policy.NoDataFor.IsPositive() {
		return errors.NewInvalidInputf(errors.CodeInvalidInput, "noDataFor requires alertOnNoData")
	}
	return nil
}

type NumericThreshold struct {
	Severity       string   `json:"severity"`
	Target         *float64 `json:"target"`
	TargetUnit     string   `json:"targetUnit,omitempty"`
	RecoveryTarget *float64 `json:"recoveryTarget,omitempty"`
	Channels       []string `json:"channels"`
}

type NumericThresholdCondition struct {
	Reduction  Reduction          `json:"reduction"`
	Operator   NumericOperator    `json:"operator"`
	Thresholds []NumericThreshold `json:"thresholds"`
}

func (condition NumericThresholdCondition) Validate() error {
	if _, ok := condition.Reduction.legacy(); !ok {
		return errors.NewInvalidInputf(errors.CodeInvalidInput, "unsupported numeric reduction %q", condition.Reduction)
	}
	if _, ok := condition.Operator.legacy(); !ok {
		return errors.NewInvalidInputf(errors.CodeInvalidInput, "unsupported numeric operator %q", condition.Operator)
	}
	if len(condition.Thresholds) == 0 || len(condition.Thresholds) > 3 {
		return errors.NewInvalidInputf(errors.CodeInvalidInput, "numeric condition must have between one and three thresholds")
	}
	seen := make(map[string]struct{}, len(condition.Thresholds))
	for _, threshold := range condition.Thresholds {
		severity := strings.ToLower(strings.TrimSpace(threshold.Severity))
		if severity == "" {
			return errors.NewInvalidInputf(errors.CodeInvalidInput, "threshold severity is required")
		}
		if _, exists := seen[severity]; exists {
			return errors.NewInvalidInputf(errors.CodeInvalidInput, "duplicate threshold severity %q", threshold.Severity)
		}
		seen[severity] = struct{}{}
		if threshold.Target == nil {
			return errors.NewInvalidInputf(errors.CodeInvalidInput, "threshold target is required")
		}
	}
	return nil
}

type BooleanCondition struct {
	Policy   Reduction `json:"policy"`
	Severity string    `json:"severity"`
	Channels []string  `json:"channels"`
}

func (condition BooleanCondition) Validate() error {
	if condition.Policy != ReductionAtLeastOnce && condition.Policy != ReductionAllTheTime && condition.Policy != ReductionLast {
		return errors.NewInvalidInputf(errors.CodeInvalidInput, "boolean policy must be at_least_once, all_the_time, or last")
	}
	if strings.TrimSpace(condition.Severity) == "" {
		return errors.NewInvalidInputf(errors.CodeInvalidInput, "boolean severity is required")
	}
	return nil
}

func (reduction Reduction) legacy() (MatchType, bool) {
	switch reduction {
	case ReductionAtLeastOnce:
		return AtleastOnce, true
	case ReductionAllTheTime:
		return AllTheTimes, true
	case ReductionAverage:
		return OnAverage, true
	case ReductionTotal:
		return InTotal, true
	case ReductionLast:
		return Last, true
	default:
		return MatchTypeNone, false
	}
}

func (operator NumericOperator) legacy() (CompareOp, bool) {
	switch operator {
	case NumericOperatorEqual:
		return ValueIsEq, true
	case NumericOperatorNotEqual:
		return ValueIsNotEq, true
	case NumericOperatorGreaterThan:
		return ValueIsAbove, true
	case NumericOperatorGreaterThanOrEqual:
		return ValueAboveOrEq, true
	case NumericOperatorLessThan:
		return ValueIsBelow, true
	case NumericOperatorLessThanOrEqual:
		return ValueBelowOrEq, true
	default:
		return CompareOpNone, false
	}
}

func reductionFromLegacy(match MatchType) (Reduction, bool) {
	switch match {
	case AtleastOnce:
		return ReductionAtLeastOnce, true
	case AllTheTimes:
		return ReductionAllTheTime, true
	case OnAverage:
		return ReductionAverage, true
	case InTotal:
		return ReductionTotal, true
	case Last:
		return ReductionLast, true
	default:
		return "", false
	}
}

func numericOperatorFromLegacy(operator CompareOp) (NumericOperator, bool) {
	switch operator {
	case ValueIsEq:
		return NumericOperatorEqual, true
	case ValueIsNotEq:
		return NumericOperatorNotEqual, true
	case ValueIsAbove:
		return NumericOperatorGreaterThan, true
	case ValueAboveOrEq:
		return NumericOperatorGreaterThanOrEqual, true
	case ValueIsBelow:
		return NumericOperatorLessThan, true
	case ValueBelowOrEq:
		return NumericOperatorLessThanOrEqual, true
	default:
		return "", false
	}
}

type RuleCondition struct {
	Kind           ConditionKind              `json:"kind"`
	CompositeQuery *CompositeQuery            `json:"compositeQuery"`
	SelectedQuery  string                     `json:"selectedQueryName"`
	DataQuality    DataQualityPolicy          `json:"dataQuality,omitempty"`
	Numeric        *NumericThresholdCondition `json:"numeric,omitempty"`
	Boolean        *BooleanCondition          `json:"boolean,omitempty"`

	// The fields below are the existing threshold-state-machine adapter. They
	// are intentionally never serialized and are populated only after a v3
	// condition has validated. Keeping the wire contract separate lets the
	// evaluator evolve without retaining the retired condition JSON.
	CompareOp         CompareOp          `json:"-"`
	Target            *float64           `json:"-"`
	AlertOnAbsent     bool               `json:"-"`
	AbsentFor         uint64             `json:"-"`
	MatchType         MatchType          `json:"-"`
	TargetUnit        string             `json:"-"`
	RequireMinPoints  bool               `json:"-"`
	RequiredNumPoints int                `json:"-"`
	Thresholds        *RuleThresholdData `json:"-"`
}

func (rc *RuleCondition) UnmarshalJSON(data []byte) error {
	var fields map[string]json.RawMessage
	if err := json.Unmarshal(data, &fields); err != nil {
		return err
	}
	allowed := map[string]struct{}{
		"kind": {}, "compositeQuery": {}, "selectedQueryName": {}, "dataQuality": {}, "numeric": {}, "boolean": {},
	}
	for field := range fields {
		if _, ok := allowed[field]; !ok {
			return errors.NewInvalidInputf(errors.CodeInvalidInput, "unsupported alert condition field %q", field)
		}
	}
	type wireCondition struct {
		Kind           ConditionKind              `json:"kind"`
		CompositeQuery *CompositeQuery            `json:"compositeQuery"`
		SelectedQuery  string                     `json:"selectedQueryName"`
		DataQuality    DataQualityPolicy          `json:"dataQuality,omitempty"`
		Numeric        *NumericThresholdCondition `json:"numeric,omitempty"`
		Boolean        *BooleanCondition          `json:"boolean,omitempty"`
	}
	var decoded wireCondition
	if err := json.Unmarshal(data, &decoded); err != nil {
		return err
	}
	*rc = RuleCondition{
		Kind:           decoded.Kind,
		CompositeQuery: decoded.CompositeQuery,
		SelectedQuery:  decoded.SelectedQuery,
		DataQuality:    decoded.DataQuality,
		Numeric:        decoded.Numeric,
		Boolean:        decoded.Boolean,
	}
	if err := rc.Validate(); err != nil {
		return err
	}
	rc.populateRuntimeFields()
	return nil
}

// MarshalJSON only emits the v3 condition union. The fallback is for Go code
// that constructs the existing evaluator adapter directly (mostly focused
// state-machine tests); it serializes that in-memory state as v3 rather than
// reintroducing any retired input shape.
func (rc RuleCondition) MarshalJSON() ([]byte, error) {
	type wireCondition struct {
		Kind           ConditionKind              `json:"kind"`
		CompositeQuery *CompositeQuery            `json:"compositeQuery"`
		SelectedQuery  string                     `json:"selectedQueryName"`
		DataQuality    DataQualityPolicy          `json:"dataQuality,omitempty"`
		Numeric        *NumericThresholdCondition `json:"numeric,omitempty"`
		Boolean        *BooleanCondition          `json:"boolean,omitempty"`
	}
	if rc.Kind != "" {
		return json.Marshal(wireCondition{
			Kind: rc.Kind, CompositeQuery: rc.CompositeQuery, SelectedQuery: rc.SelectedQuery,
			DataQuality: rc.DataQuality, Numeric: rc.Numeric, Boolean: rc.Boolean,
		})
	}
	if rc.Thresholds == nil {
		return json.Marshal(wireCondition{CompositeQuery: rc.CompositeQuery, SelectedQuery: rc.SelectedQuery})
	}
	legacy, ok := rc.Thresholds.Spec.(BasicRuleThresholds)
	if !ok || len(legacy) == 0 {
		return nil, errors.NewInvalidInputf(errors.CodeInvalidInput, "cannot serialize invalid in-memory threshold condition")
	}
	reduction, ok := reductionFromLegacy(legacy[0].MatchType)
	if !ok {
		return nil, errors.NewInvalidInputf(errors.CodeInvalidInput, "cannot serialize unsupported legacy match type %q", legacy[0].MatchType)
	}
	operator, ok := numericOperatorFromLegacy(legacy[0].CompareOp)
	if !ok {
		return nil, errors.NewInvalidInputf(errors.CodeInvalidInput, "cannot serialize unsupported legacy compare op %q", legacy[0].CompareOp)
	}
	thresholds := make([]NumericThreshold, 0, len(legacy))
	for _, threshold := range legacy {
		thresholds = append(thresholds, NumericThreshold{
			Severity: threshold.Name, Target: threshold.TargetValue, TargetUnit: threshold.TargetUnit,
			RecoveryTarget: threshold.RecoveryTarget, Channels: threshold.Channels,
		})
	}
	policy := DataQualityPolicy{AlertOnNoData: rc.AlertOnAbsent, MinPoints: rc.RequiredNumPoints}
	if rc.AlertOnAbsent && rc.AbsentFor > 0 {
		policy.NoDataFor = valuer.MustParseTextDuration((time.Duration(rc.AbsentFor) * time.Minute).String())
	}
	return json.Marshal(wireCondition{
		Kind: ConditionKindNumeric, CompositeQuery: rc.CompositeQuery, SelectedQuery: rc.SelectedQuery,
		DataQuality: policy,
		Numeric:     &NumericThresholdCondition{Reduction: reduction, Operator: operator, Thresholds: thresholds},
	})
}

func (rc *RuleCondition) Validate() error {
	if rc == nil || rc.CompositeQuery == nil || len(rc.CompositeQuery.Queries) == 0 {
		return errors.NewInvalidInputf(errors.CodeInvalidInput, "rule condition requires a composite query")
	}
	if rc.Kind == "" {
		// Go callers which construct a RuleCondition directly still use the
		// unexported execution adapter. Public JSON always supplies a v3 kind.
		if rc.Thresholds != nil {
			return nil
		}
		return errors.NewInvalidInputf(errors.CodeInvalidInput, "rule condition kind is required")
	}
	if strings.TrimSpace(rc.SelectedQuery) == "" {
		return errors.NewInvalidInputf(errors.CodeInvalidInput, "selectedQueryName is required")
	}
	if err := rc.DataQuality.Validate(); err != nil {
		return err
	}
	switch rc.Kind {
	case ConditionKindNumeric:
		if rc.Numeric == nil || rc.Boolean != nil {
			return errors.NewInvalidInputf(errors.CodeInvalidInput, "numeric condition requires numeric details only")
		}
		return rc.Numeric.Validate()
	case ConditionKindBoolean:
		if rc.Boolean == nil || rc.Numeric != nil {
			return errors.NewInvalidInputf(errors.CodeInvalidInput, "boolean condition requires boolean details only")
		}
		return rc.Boolean.Validate()
	default:
		return errors.NewInvalidInputf(errors.CodeInvalidInput, "unsupported alert condition kind %q", rc.Kind)
	}
}

func (rc *RuleCondition) populateRuntimeFields() {
	rc.AlertOnAbsent = rc.DataQuality.AlertOnNoData
	rc.RequiredNumPoints = rc.DataQuality.MinPoints
	rc.RequireMinPoints = rc.DataQuality.MinPoints > 0
	rc.AbsentFor = uint64(rc.DataQuality.NoDataFor.Duration() / time.Minute)
	if rc.Kind == ConditionKindBoolean && rc.Boolean != nil {
		rc.Thresholds = &RuleThresholdData{Kind: BooleanThresholdKind, Spec: BooleanRuleThreshold{
			Policy: rc.Boolean.Policy, Severity: rc.Boolean.Severity, Channels: rc.Boolean.Channels,
		}}
		return
	}
	if rc.Kind != ConditionKindNumeric || rc.Numeric == nil {
		return
	}
	matchType, _ := rc.Numeric.Reduction.legacy()
	compareOp, _ := rc.Numeric.Operator.legacy()
	rc.MatchType = matchType
	rc.CompareOp = compareOp
	thresholds := make(BasicRuleThresholds, 0, len(rc.Numeric.Thresholds))
	for _, threshold := range rc.Numeric.Thresholds {
		thresholds = append(thresholds, BasicRuleThreshold{
			Name: threshold.Severity, TargetValue: threshold.Target, TargetUnit: threshold.TargetUnit,
			RecoveryTarget: threshold.RecoveryTarget, MatchType: matchType, CompareOp: compareOp, Channels: threshold.Channels,
		})
	}
	rc.Thresholds = &RuleThresholdData{Kind: BasicThresholdKind, Spec: thresholds}
}

// CompositeQuery is the alert-rule query entity. Query execution is V5-only,
// while the remaining metadata controls alert formatting and presentation.
type CompositeQuery struct {
	Queries     []qbtypes.QueryEnvelope `json:"queries"`
	PanelType   querytypes.PanelType    `json:"panelType"`
	QueryType   querytypes.QueryType    `json:"queryType"`
	ResultUnit  string                  `json:"resultUnit,omitempty"`
	DisplayUnit string                  `json:"displayUnit,omitempty"`
	FillGaps    bool                    `json:"fillGaps,omitempty"`
}

func (c *CompositeQuery) UnmarshalJSON(data []byte) error {
	var fields map[string]json.RawMessage
	if err := json.Unmarshal(data, &fields); err != nil {
		return err
	}

	validFields := map[string]struct{}{
		"queries":     {},
		"panelType":   {},
		"queryType":   {},
		"resultUnit":  {},
		"displayUnit": {},
		"fillGaps":    {},
	}
	for field := range fields {
		if _, ok := validFields[field]; !ok {
			return errors.Newf(errors.TypeInvalidInput, errors.CodeInvalidInput, "unsupported alert composite query field %q", field)
		}
	}

	type alias CompositeQuery
	var decoded alias
	if err := json.Unmarshal(data, &decoded); err != nil {
		return err
	}

	*c = CompositeQuery(decoded)
	if c.DisplayUnit == "" {
		c.DisplayUnit = c.ResultUnit
	}
	return nil
}

func (c *CompositeQuery) EffectiveResultUnit() string {
	if c == nil {
		return ""
	}
	return c.ResultUnit
}

func (rc *RuleCondition) GetSelectedQueryName() string {
	if rc == nil {
		return ""
	}
	return rc.SelectedQuery
}

func (rc *RuleCondition) IsValid() bool {
	return rc.Validate() == nil
}

// ShouldEval checks if the further series should be evaluated at all for alerts.
func (rc *RuleCondition) ShouldEval(series *timeseriestypes.Series) bool {
	if rc == nil {
		return true
	}
	if rc.DataQuality.MinPoints > 0 {
		return len(series.Points) >= rc.DataQuality.MinPoints
	}
	return !rc.RequireMinPoints || len(series.Points) >= rc.RequiredNumPoints
}

// NoDataAfter is the duration before a missing-data alert is eligible. V3
// preserves sub-minute values; the legacy adapter is only a fallback for
// direct in-process callers that have not been serialized as a rule yet.
func (rc *RuleCondition) NoDataAfter() time.Duration {
	if rc == nil {
		return 0
	}
	if rc.DataQuality.AlertOnNoData {
		return rc.DataQuality.NoDataFor.Duration()
	}
	return time.Duration(rc.AbsentFor) * time.Minute
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
