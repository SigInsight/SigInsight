package ruletypes

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/require"

	qbtypes "github.com/SigNoz/signoz/pkg/types/querybuildertypes/querybuildertypesv5"
)

func TestIsAllQueriesDisabled(t *testing.T) {
	testCases := []struct {
		name           string
		compositeQuery *CompositeQuery
		expected       bool
	}{
		{name: "nil composite query", expected: false},
		{name: "empty query list", compositeQuery: &CompositeQuery{}, expected: false},
		{
			name: "all supported queries disabled",
			compositeQuery: &CompositeQuery{Queries: []qbtypes.QueryEnvelope{
				{Spec: qbtypes.QueryBuilderQuery[qbtypes.MetricAggregation]{Disabled: true}},
				{Spec: qbtypes.QueryBuilderQuery[qbtypes.LogAggregation]{Disabled: true}},
				{Spec: qbtypes.QueryBuilderQuery[qbtypes.TraceAggregation]{Disabled: true}},
				{Spec: qbtypes.QueryBuilderFormula{Disabled: true}},
				{Spec: qbtypes.QueryBuilderJoin{Disabled: true}},
				{Spec: qbtypes.ClickHouseQuery{Disabled: true}},
			}},
			expected: true,
		},
		{
			name: "one query enabled",
			compositeQuery: &CompositeQuery{Queries: []qbtypes.QueryEnvelope{
				{Spec: qbtypes.QueryBuilderFormula{Disabled: false}},
			}},
			expected: false,
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			require.Equal(t, testCase.expected, isAllQueriesDisabled(testCase.compositeQuery))
		})
	}
}

const currentRuleJSON = `{
	"alert":"cpu usage",
	"alertType":"METRIC_BASED_ALERT",
	"ruleType":"threshold_rule",
	"version":"v5",
	"schemaVersion":"v3alpha1",
	"condition":{
		"kind":"numeric",
		"compositeQuery":{"queryType":"builder","queries":[{"type":"builder_query","spec":{"name":"A","signal":"metrics","aggregations":[{"metricName":"cpu_usage","spaceAggregation":"sum"}],"stepInterval":"1m"}}]},
		"selectedQueryName":"A",
		"dataQuality":{"minPoints":2},
		"numeric":{"reduction":"at_least_once","operator":"gt","thresholds":[{"severity":"critical","target":90,"channels":["email"]}]}
	},
	"evaluation":{"kind":"rolling","spec":{"evalWindow":"5m","frequency":"1m"}},
	"labels":{"severity":"critical"},
	"annotations":{"description":"CPU usage is above the configured threshold."},
	"notificationSettings":{"groupBy":[]}
}`

const booleanRuleJSON = `{
	"alert":"high error rate",
	"alertType":"TRACES_BASED_ALERT",
	"ruleType":"threshold_rule",
	"version":"v5",
	"schemaVersion":"v3alpha1",
	"condition":{
		"kind":"boolean",
		"compositeQuery":{"queryType":"builder","queries":[
			{"type":"builder_query","spec":{"name":"A","signal":"traces","aggregations":[{"expression":"count()"}],"stepInterval":"1m"}},
			{"type":"builder_formula","spec":{"name":"F1","expression":"A > 10"}}
		]},
		"selectedQueryName":"F1",
		"dataQuality":{"alertOnNoData":true,"noDataFor":"30s","minPoints":1},
		"boolean":{"policy":"last","severity":"critical","channels":["email"]}
	},
	"evaluation":{"kind":"cumulative","spec":{"period":"1d","frequency":"5m","timezone":"Asia/Shanghai"}},
	"notificationSettings":{"groupBy":["service.name"]}
}`

func TestPostableRuleAcceptsCurrentContract(t *testing.T) {
	var rule PostableRule
	require.NoError(t, json.Unmarshal([]byte(currentRuleJSON), &rule))
	require.Equal(t, CurrentSchemaVersion, rule.SchemaVersion)
	require.Equal(t, RuleType(RuleTypeThreshold), rule.RuleType)
	require.NotNil(t, rule.RuleCondition)
	require.NotNil(t, rule.RuleCondition.Numeric)
	require.NotNil(t, rule.RuleCondition.Thresholds)
	require.NotNil(t, rule.Evaluation)
	require.NotNil(t, rule.NotificationSettings)
}

func TestPostableRuleAcceptsBooleanV3Contract(t *testing.T) {
	var rule PostableRule
	require.NoError(t, json.Unmarshal([]byte(booleanRuleJSON), &rule))
	require.Equal(t, CurrentSchemaVersion, rule.SchemaVersion)
	require.Equal(t, ConditionKindBoolean, rule.RuleCondition.Kind)
	require.Nil(t, rule.RuleCondition.Numeric)
	require.Equal(t, ReductionLast, rule.RuleCondition.Boolean.Policy)
	require.Equal(t, BooleanThresholdKind, rule.RuleCondition.Thresholds.Kind)

	cumulative, ok := rule.Evaluation.Spec.(CumulativeWindow)
	require.True(t, ok)
	require.Equal(t, CumulativePeriodDay, cumulative.Period)
	require.Equal(t, "Asia/Shanghai", cumulative.Timezone)
}

func TestPostableRuleRejectsRetiredContract(t *testing.T) {
	testCases := []struct {
		name    string
		payload string
		errPart string
	}{
		{
			name:    "v2 schema",
			payload: replaceSchemaVersion(currentRuleJSON, `"schemaVersion":"v2alpha1"`),
			errPart: "only schema version",
		},
		{
			name:    "legacy top level fields",
			payload: replaceSchemaVersion(currentRuleJSON, `"schemaVersion":"v3alpha1","evalWindow":"5m"`),
			errPart: "retired alert field",
		},
		{
			name:    "renotify settings",
			payload: replaceNotificationSettings(`"notificationSettings":{"renotify":{"enabled":true,"interval":"1h","alertStates":["firing"]}}`),
			errPart: "retired notification setting",
		},
		{
			name:    "legacy condition thresholds",
			payload: replaceJSONFragment(currentRuleJSON, `"dataQuality":{"minPoints":2},`, `"thresholds":{"kind":"basic","spec":[]},"dataQuality":{"minPoints":2},`),
			errPart: "unsupported alert condition field \"thresholds\"",
		},
		{
			name:    "legacy composite unit",
			payload: replaceJSONFragment(currentRuleJSON, `"compositeQuery":{"queryType":"builder",`, `"compositeQuery":{"queryType":"builder","unit":"ns",`),
			errPart: "unsupported alert composite query field \"unit\"",
		},
		{
			name:    "legacy cumulative schedule",
			payload: replaceJSONFragment(booleanRuleJSON, `"spec":{"period":"1d","frequency":"5m","timezone":"Asia/Shanghai"}`, `"spec":{"schedule":{"type":"daily"},"frequency":"5m","timezone":"Asia/Shanghai"}`),
			errPart: "unsupported evaluation field \"schedule\"",
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			var rule PostableRule
			require.ErrorContains(t, json.Unmarshal([]byte(testCase.payload), &rule), testCase.errPart)
		})
	}
}

func TestPostableRuleNotificationMessageTemplate(t *testing.T) {
	validPayload := replaceNotificationSettings(`"notificationSettings":{"groupBy":[],"messageTemplate":"{{alert.name}} {{value}} {{label.service.name}}"}`)
	var rule PostableRule
	require.NoError(t, json.Unmarshal([]byte(validPayload), &rule))
	require.Equal(t, "{{alert.name}} {{value}} {{label.service.name}}", rule.NotificationSettings.MessageTemplate)

	invalidPayload := replaceNotificationSettings(`"notificationSettings":{"groupBy":[],"messageTemplate":"{{ $value }}"}`)
	require.ErrorContains(t, json.Unmarshal([]byte(invalidPayload), &rule), "unsupported notification message placeholder")

	annotationTemplate := replaceNotificationSettings(`"annotations":{"description":"value is {{$value}}"}`)
	require.ErrorContains(t, json.Unmarshal([]byte(annotationTemplate), &rule), "alert annotation templates are not supported")
}

func replaceSchemaVersion(payload, replacement string) string {
	return replaceJSONFragment(payload, `"schemaVersion":"v3alpha1"`, replacement)
}

func replaceNotificationSettings(replacement string) string {
	return replaceJSONFragment(currentRuleJSON, `"notificationSettings":{"groupBy":[]}`, replacement)
}

func replaceJSONFragment(payload, old, replacement string) string {
	return stringReplace(payload, old, replacement)
}

func stringReplace(payload, old, replacement string) string {
	for i := 0; i+len(old) <= len(payload); i++ {
		if payload[i:i+len(old)] == old {
			return payload[:i] + replacement + payload[i+len(old):]
		}
	}
	return payload
}
