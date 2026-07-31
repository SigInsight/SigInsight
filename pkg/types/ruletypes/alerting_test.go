package ruletypes

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/SigNoz/signoz/pkg/query-service/model/querytypes"
	qbtypes "github.com/SigNoz/signoz/pkg/types/querybuildertypes/querybuildertypesv5"
)

func TestRuleConditionGetSelectedQueryName(t *testing.T) {
	testCases := []struct {
		name      string
		condition *RuleCondition
		expected  string
	}{
		{name: "nil condition"},
		{name: "missing selected query", condition: &RuleCondition{CompositeQuery: v5CompositeQuery("F1")}},
		{name: "explicit selected query", condition: &RuleCondition{CompositeQuery: v5CompositeQuery("A"), SelectedQuery: "A"}, expected: "A"},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			assert.Equal(t, testCase.expected, testCase.condition.GetSelectedQueryName())
		})
	}
}

func TestRuleConditionIsValidRequiresV5Queries(t *testing.T) {
	require.False(t, (*RuleCondition)(nil).IsValid())

	condition := &RuleCondition{
		CompositeQuery: &CompositeQuery{QueryType: querytypes.QueryTypeClickHouseSQL},
	}

	require.False(t, condition.IsValid())

	condition.CompositeQuery = v5CompositeQuery("A")
	require.True(t, condition.IsValid())
}

func TestCompositeQueryRejectsLegacyFields(t *testing.T) {
	for _, field := range []string{"builderQueries", "promQueries", "chQueries"} {
		t.Run(field, func(t *testing.T) {
			var compositeQuery CompositeQuery
			err := json.Unmarshal([]byte(`{"queries":[],"`+field+`":{}}`), &compositeQuery)
			require.ErrorContains(t, err, "unsupported alert composite query field")
		})
	}
}

func TestCompositeQueryMigratesLegacyUnitOnUnmarshal(t *testing.T) {
	var compositeQuery CompositeQuery
	require.NoError(t, json.Unmarshal([]byte(`{"queries":[],"unit":"ns"}`), &compositeQuery))
	require.Equal(t, "ns", compositeQuery.ResultUnit)
	require.Equal(t, "ns", compositeQuery.DisplayUnit)

	encoded, err := json.Marshal(compositeQuery)
	require.NoError(t, err)
	require.NotContains(t, string(encoded), `"unit"`)
	require.Contains(t, string(encoded), `"resultUnit":"ns"`)
	require.Contains(t, string(encoded), `"displayUnit":"ns"`)
}

func TestNormalizeRuleUnitsCorrectsLogCount(t *testing.T) {
	target := 1.0
	rule := &PostableRule{
		AlertType: AlertTypeLogs,
		RuleCondition: &RuleCondition{
			SelectedQuery: "A",
			CompositeQuery: &CompositeQuery{
				Unit: "s",
				Queries: []qbtypes.QueryEnvelope{{
					Type: qbtypes.QueryTypeBuilder,
					Spec: qbtypes.QueryBuilderQuery[qbtypes.LogAggregation]{
						Name:         "A",
						Aggregations: []qbtypes.LogAggregation{{Expression: "count()"}},
					},
				}},
			},
			Thresholds: &RuleThresholdData{
				Kind: BasicThresholdKind,
				Spec: BasicRuleThresholds{{TargetValue: &target, TargetUnit: "s"}},
			},
		},
	}

	normalizeRuleUnits(rule)

	require.Equal(t, countResultUnit, rule.RuleCondition.CompositeQuery.ResultUnit)
	require.Equal(t, countResultUnit, rule.RuleCondition.CompositeQuery.DisplayUnit)
	thresholds := rule.RuleCondition.Thresholds.Spec.(BasicRuleThresholds)
	require.Equal(t, countResultUnit, thresholds[0].TargetUnit)
}

func TestValidateRejectsNewLogCountWithTimeUnits(t *testing.T) {
	target := 1.0
	rule := &PostableRule{
		Version:   "v5",
		AlertType: AlertTypeLogs,
		RuleCondition: &RuleCondition{
			SelectedQuery: "A",
			CompositeQuery: &CompositeQuery{
				ResultUnit:  "s",
				DisplayUnit: "s",
				Queries: []qbtypes.QueryEnvelope{{
					Type: qbtypes.QueryTypeBuilder,
					Spec: qbtypes.QueryBuilderQuery[qbtypes.LogAggregation]{
						Name:         "A",
						Aggregations: []qbtypes.LogAggregation{{Expression: "count()"}},
					},
				}},
			},
			Thresholds: &RuleThresholdData{
				Kind: BasicThresholdKind,
				Spec: BasicRuleThresholds{{
					Name:        "critical",
					TargetValue: &target,
					TargetUnit:  "s",
					MatchType:   AtleastOnce,
					CompareOp:   ValueIsAbove,
				}},
			},
		},
	}

	require.ErrorContains(t, rule.validate(), `expected "{count}"`)
}

func TestNormalizeRuleUnitsDefaultsTargetToResultUnit(t *testing.T) {
	target := 2.0
	rule := &PostableRule{
		RuleCondition: &RuleCondition{
			CompositeQuery: &CompositeQuery{ResultUnit: "ns"},
			Thresholds: &RuleThresholdData{
				Kind: BasicThresholdKind,
				Spec: BasicRuleThresholds{{TargetValue: &target}},
			},
		},
	}

	normalizeRuleUnits(rule)
	thresholds := rule.RuleCondition.Thresholds.Spec.(BasicRuleThresholds)
	require.Equal(t, "ns", thresholds[0].TargetUnit)
}

func TestValidateRejectsUnitsWithoutResultUnit(t *testing.T) {
	target := 1.0
	rule := &PostableRule{
		Version:       "v5",
		SchemaVersion: "v2alpha1",
		RuleCondition: &RuleCondition{
			CompositeQuery: &CompositeQuery{DisplayUnit: "s"},
			Thresholds: &RuleThresholdData{
				Kind: BasicThresholdKind,
				Spec: BasicRuleThresholds{{TargetValue: &target, TargetUnit: "s"}},
			},
		},
	}

	err := rule.validate()
	require.ErrorContains(t, err, "display unit requires a result unit")
	require.ErrorContains(t, err, "target unit requires a result unit")
}

func v5CompositeQuery(name string) *CompositeQuery {
	return &CompositeQuery{
		QueryType: querytypes.QueryTypeClickHouseSQL,
		Queries: []qbtypes.QueryEnvelope{{
			Type: qbtypes.QueryTypeClickHouseSQL,
			Spec: qbtypes.ClickHouseQuery{Name: name, Query: "SELECT 1"},
		}},
	}
}
