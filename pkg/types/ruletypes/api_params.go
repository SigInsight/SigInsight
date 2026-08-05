package ruletypes

import (
	"encoding/json"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/prometheus/alertmanager/config"

	signozError "github.com/SigNoz/signoz/pkg/errors"
	"github.com/SigNoz/signoz/pkg/query-service/model"
	"github.com/SigNoz/signoz/pkg/types/alertmanagertypes"
	"github.com/SigNoz/signoz/pkg/units"
	"github.com/SigNoz/signoz/pkg/valuer"
)

type AlertType string

const (
	AlertTypeMetric     AlertType = "METRIC_BASED_ALERT"
	AlertTypeTraces     AlertType = "TRACES_BASED_ALERT"
	AlertTypeLogs       AlertType = "LOGS_BASED_ALERT"
	AlertTypeExceptions AlertType = "EXCEPTIONS_BASED_ALERT"
)

const CurrentSchemaVersion = "v3alpha1"

// PostableRule is used to create alerting rule from HTTP api
type PostableRule struct {
	AlertName string    `json:"alert,omitempty"`
	AlertType AlertType `json:"alertType,omitempty"`
	RuleType  RuleType  `json:"ruleType,omitempty"`

	RuleCondition *RuleCondition    `json:"condition,omitempty"`
	Labels        map[string]string `json:"labels,omitempty"`
	Annotations   map[string]string `json:"annotations,omitempty"`

	Disabled bool `json:"disabled"`

	// Source captures the source url where rule has been created
	Source string `json:"source,omitempty"`

	Version string `json:"version,omitempty"`

	Evaluation    *EvaluationEnvelope `yaml:"evaluation,omitempty" json:"evaluation,omitempty"`
	SchemaVersion string              `json:"schemaVersion,omitempty"`

	NotificationSettings *NotificationSettings `json:"notificationSettings,omitempty"`
}

type NotificationSettings struct {
	GroupBy []string `json:"groupBy,omitempty"`
	// NewGroupEvalDelay is the grace period for new series to be excluded from alerts evaluation
	NewGroupEvalDelay valuer.TextDuration `json:"newGroupEvalDelay,omitzero"`
}

func (ns *NotificationSettings) GetAlertManagerNotificationConfig() alertmanagertypes.NotificationConfig {
	return alertmanagertypes.NewNotificationConfig(ns.GroupBy, 0, 0)
}

func (r *PostableRule) GetRuleChannels() ([]string, error) {
	threshold, err := r.RuleCondition.Thresholds.GetRuleThreshold()
	if err != nil {
		return nil, err
	}
	channels := make([]string, 0)
	seen := make(map[string]struct{})
	for _, receiver := range threshold.GetRuleReceivers() {
		for _, channel := range receiver.Channels {
			if _, exists := seen[channel]; exists {
				continue
			}
			seen[channel] = struct{}{}
			channels = append(channels, channel)
		}
	}
	return channels, nil
}

// EvalWindow derives the current evaluation window for related-data links.
// The window is part of Evaluation and is intentionally not a second JSON field.
func (r *PostableRule) EvalWindow() valuer.TextDuration {
	if r.Evaluation == nil {
		return valuer.TextDuration{}
	}
	evaluation, err := r.Evaluation.GetEvaluation()
	if err != nil {
		return valuer.TextDuration{}
	}
	now := time.Now()
	start, end := evaluation.NextWindowFor(now)
	return valuer.MustParseTextDuration(end.Sub(start).String())
}

func (r *PostableRule) GetInhibitRules(ruleId string) ([]config.InhibitRule, error) {
	threshold, err := r.RuleCondition.Thresholds.GetRuleThreshold()
	if err != nil {
		return nil, err
	}
	var groups []string
	if r.NotificationSettings != nil {
		for k := range r.NotificationSettings.GetAlertManagerNotificationConfig().NotificationGroup {
			groups = append(groups, string(k))
		}
	}
	receivers := threshold.GetRuleReceivers()
	var inhibitRules []config.InhibitRule
	for i := 0; i < len(receivers)-1; i++ {
		rule := config.InhibitRule{
			SourceMatchers: config.Matchers{
				{
					Name:  LabelThresholdName,
					Value: receivers[i].Name,
				},
				{
					Name:  LabelRuleId,
					Value: ruleId,
				},
			},
			TargetMatchers: config.Matchers{
				{
					Name:  LabelThresholdName,
					Value: receivers[i+1].Name,
				},
				{
					Name:  LabelRuleId,
					Value: ruleId,
				},
			},
			Equal: groups,
		}
		inhibitRules = append(inhibitRules, rule)
	}
	return inhibitRules, nil
}

func (r *PostableRule) UnmarshalJSON(bytes []byte) error {
	type Alias PostableRule
	var raw map[string]json.RawMessage
	if err := json.Unmarshal(bytes, &raw); err != nil {
		return signozError.NewInvalidInputf(signozError.CodeInvalidInput, "failed to parse json: %v", err)
	}
	for _, field := range []string{"description", "evalWindow", "frequency", "preferredChannels"} {
		if _, ok := raw[field]; ok {
			return signozError.NewInvalidInputf(signozError.CodeInvalidInput, "retired alert field %q is not supported", field)
		}
	}
	var version struct {
		SchemaVersion string `json:"schemaVersion"`
	}
	if err := json.Unmarshal(bytes, &version); err != nil {
		return signozError.NewInvalidInputf(signozError.CodeInvalidInput, "failed to parse schema version: %v", err)
	}
	if version.SchemaVersion != CurrentSchemaVersion {
		return signozError.NewInvalidInputf(signozError.CodeInvalidInput, "only schema version %q is supported, got %q", CurrentSchemaVersion, version.SchemaVersion)
	}

	var decoded Alias
	if err := json.Unmarshal(bytes, &decoded); err != nil {
		return signozError.NewInvalidInputf(signozError.CodeInvalidInput, "failed to parse json: %v", err)
	}
	*r = PostableRule(decoded)
	normalizeRuleUnits(r)
	return r.validate()
}

func (ns *NotificationSettings) UnmarshalJSON(data []byte) error {
	type Alias NotificationSettings
	var raw map[string]json.RawMessage
	if err := json.Unmarshal(data, &raw); err != nil {
		return err
	}
	if _, ok := raw["renotify"]; ok {
		return signozError.NewInvalidInputf(signozError.CodeInvalidInput, "retired notification setting %q is not supported", "renotify")
	}
	var decoded Alias
	if err := json.Unmarshal(data, &decoded); err != nil {
		return err
	}
	*ns = NotificationSettings(decoded)
	return nil
}

func isValidLabelName(ln string) bool {
	if len(ln) == 0 {
		return false
	}
	for i, b := range ln {
		if !((b >= 'a' && b <= 'z') || (b >= 'A' && b <= 'Z') || b == '_' || b == '.' || (b >= '0' && b <= '9' && i > 0)) {
			return false
		}
	}
	return true
}

func isValidLabelValue(v string) bool {
	return utf8.ValidString(v)
}

func isAllQueriesDisabled(compositeQuery *CompositeQuery) bool {
	if compositeQuery == nil || len(compositeQuery.Queries) == 0 {
		return false
	}
	for idx := range compositeQuery.Queries {
		if !compositeQuery.Queries[idx].IsDisabled() {
			return false
		}
	}
	return true
}

func (r *PostableRule) validate() error {

	var errs []error

	if r.RuleCondition == nil {
		// will get panic if we try to access CompositeQuery, so return here
		return signozError.NewInvalidInputf(signozError.CodeInvalidInput, "rule condition is required")
	}
	if r.RuleCondition.CompositeQuery == nil {
		errs = append(errs, signozError.NewInvalidInputf(signozError.CodeInvalidInput, "composite query is required"))
	}

	if r.Version != "v5" {
		errs = append(errs, signozError.NewInvalidInputf(signozError.CodeInvalidInput, "only version v5 is supported, got %q", r.Version))
	}
	if r.SchemaVersion != CurrentSchemaVersion {
		errs = append(errs, signozError.NewInvalidInputf(signozError.CodeInvalidInput, "only schema version %q is supported, got %q", CurrentSchemaVersion, r.SchemaVersion))
	}

	if isAllQueriesDisabled(r.RuleCondition.CompositeQuery) {
		errs = append(errs, signozError.NewInvalidInputf(signozError.CodeInvalidInput, "all queries are disabled in rule condition"))
	}

	if r.RuleCondition.CompositeQuery != nil {
		query := r.RuleCondition.CompositeQuery
		resultUnit := query.EffectiveResultUnit()
		if resultUnit == "" && query.DisplayUnit != "" {
			errs = append(errs, signozError.NewInvalidInputf(signozError.CodeInvalidInput, "display unit requires a result unit"))
		}
		if inferredUnit := inferredRuleResultUnit(r); inferredUnit != "" && resultUnit != inferredUnit {
			errs = append(errs, signozError.NewInvalidInputf(signozError.CodeInvalidInput, "result unit %q is invalid for the selected query; expected %q", resultUnit, inferredUnit))
		}
		if resultUnit != "" && query.DisplayUnit != "" && !units.AreCompatible(units.Unit(resultUnit), units.Unit(query.DisplayUnit)) {
			errs = append(errs, signozError.NewInvalidInputf(signozError.CodeInvalidInput, "display unit %q is incompatible with result unit %q", query.DisplayUnit, resultUnit))
		}
		if r.RuleCondition.Thresholds != nil {
			if thresholds, ok := r.RuleCondition.Thresholds.Spec.(BasicRuleThresholds); ok {
				for _, threshold := range thresholds {
					if resultUnit == "" && threshold.TargetUnit != "" {
						errs = append(errs, signozError.NewInvalidInputf(signozError.CodeInvalidInput, "target unit requires a result unit"))
					}
					if resultUnit != "" && threshold.TargetUnit != "" && !units.AreCompatible(units.Unit(resultUnit), units.Unit(threshold.TargetUnit)) {
						errs = append(errs, signozError.NewInvalidInputf(signozError.CodeInvalidInput, "target unit %q is incompatible with result unit %q", threshold.TargetUnit, resultUnit))
					}
				}
			}
		}
	}

	for k, v := range r.Labels {
		if !isValidLabelName(k) {
			errs = append(errs, signozError.NewInvalidInputf(signozError.CodeInvalidInput, "invalid label name: %s", k))
		}

		if !isValidLabelValue(v) {
			errs = append(errs, signozError.NewInvalidInputf(signozError.CodeInvalidInput, "invalid label value: %s", v))
		}
		if strings.Contains(v, "{{") || strings.Contains(v, "}}") {
			errs = append(errs, signozError.NewInvalidInputf(signozError.CodeInvalidInput, "alert label templates are not supported"))
		}
	}

	for k, v := range r.Annotations {
		if !isValidLabelName(k) {
			errs = append(errs, signozError.NewInvalidInputf(signozError.CodeInvalidInput, "invalid annotation name: %s", k))
		}
		if strings.Contains(v, "{{") || strings.Contains(v, "}}") {
			errs = append(errs, signozError.NewInvalidInputf(signozError.CodeInvalidInput, "alert annotation templates are not supported"))
		}
	}
	return signozError.Join(errs...)
}

// GettableRules has info for all stored rules.
type GettableRules struct {
	Rules []*GettableRule `json:"rules"`
}

// GettableRule has info for an alerting rules.
type GettableRule struct {
	Id    string           `json:"id"`
	State model.AlertState `json:"state"`
	PostableRule
	CreatedAt *time.Time `json:"createAt"`
	CreatedBy *string    `json:"createBy"`
	UpdatedAt *time.Time `json:"updateAt"`
	UpdatedBy *string    `json:"updateBy"`
}
