package ruletypes

import (
	"regexp"
	"strings"

	qbtypes "github.com/SigNoz/signoz/pkg/types/querybuildertypes/querybuildertypesv5"
	"github.com/SigNoz/signoz/pkg/units"
)

const countResultUnit = "{count}"

var countAggregationPattern = regexp.MustCompile(`(?i)^count(?:_distinct|if)?\s*\(`)

func isSelectedLogCountQuery(condition *RuleCondition) bool {
	if condition == nil || condition.CompositeQuery == nil {
		return false
	}
	selectedQuery := condition.SelectedQuery
	for _, envelope := range condition.CompositeQuery.Queries {
		if envelope.Type != qbtypes.QueryTypeBuilder {
			continue
		}
		spec, ok := envelope.Spec.(qbtypes.QueryBuilderQuery[qbtypes.LogAggregation])
		if !ok || (selectedQuery != "" && spec.Name != selectedQuery) || len(spec.Aggregations) == 0 {
			continue
		}
		return countAggregationPattern.MatchString(strings.TrimSpace(spec.Aggregations[0].Expression))
	}
	return false
}

func isSelectedExceptionCountQuery(rule *PostableRule) bool {
	if rule == nil || rule.AlertType != AlertTypeExceptions || rule.RuleCondition == nil || rule.RuleCondition.CompositeQuery == nil {
		return false
	}
	selectedQuery := rule.RuleCondition.SelectedQuery
	for _, envelope := range rule.RuleCondition.CompositeQuery.Queries {
		if envelope.Type != qbtypes.QueryTypeClickHouseSQL {
			continue
		}
		spec, ok := envelope.Spec.(qbtypes.ClickHouseQuery)
		if !ok || (selectedQuery != "" && spec.Name != selectedQuery) {
			continue
		}
		return strings.Contains(strings.ToLower(spec.Query), "count(")
	}
	return false
}

func inferredRuleResultUnit(rule *PostableRule) string {
	if rule == nil {
		return ""
	}
	if isSelectedLogCountQuery(rule.RuleCondition) || isSelectedExceptionCountQuery(rule) {
		return countResultUnit
	}
	return ""
}

func normalizeRuleUnits(rule *PostableRule) {
	if rule == nil || rule.RuleCondition == nil || rule.RuleCondition.CompositeQuery == nil {
		return
	}
	query := rule.RuleCondition.CompositeQuery
	if query.DisplayUnit == "" {
		query.DisplayUnit = query.ResultUnit
	}

	inferredUnit := inferredRuleResultUnit(rule)
	if inferredUnit != "" && query.ResultUnit == "" {
		query.ResultUnit = inferredUnit
		query.DisplayUnit = inferredUnit
	}

	if query.ResultUnit != "" && !units.AreCompatible(units.Unit(query.ResultUnit), units.Unit(query.DisplayUnit)) {
		return
	}

	if rule.RuleCondition.Numeric != nil {
		for index := range rule.RuleCondition.Numeric.Thresholds {
			if rule.RuleCondition.Numeric.Thresholds[index].TargetUnit == "" && query.ResultUnit != "" {
				rule.RuleCondition.Numeric.Thresholds[index].TargetUnit = query.ResultUnit
			}
		}
		rule.RuleCondition.populateRuntimeFields()
		return
	}

	// The direct adapter is retained until the state-machine tests are moved to
	// v3 construction. It is not a JSON compatibility path.
	if rule.RuleCondition.Thresholds != nil {
		if thresholds, ok := rule.RuleCondition.Thresholds.Spec.(BasicRuleThresholds); ok {
			for index := range thresholds {
				if thresholds[index].TargetUnit == "" && query.ResultUnit != "" {
					thresholds[index].TargetUnit = query.ResultUnit
				}
			}
			rule.RuleCondition.Thresholds.Spec = thresholds
		}
	}
}
