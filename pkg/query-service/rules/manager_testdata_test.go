package rules

import (
	"math"
	"time"

	"github.com/SigNoz/signoz/pkg/query-service/model/querytypes"
	"github.com/SigNoz/signoz/pkg/types/metrictypes"
	qbtypes "github.com/SigNoz/signoz/pkg/types/querybuildertypes/querybuildertypesv5"
	ruletypes "github.com/SigNoz/signoz/pkg/types/ruletypes"
	"github.com/SigNoz/signoz/pkg/types/telemetrytypes"
	"github.com/SigNoz/signoz/pkg/valuer"
)

// ThresholdRuleTestCase defines test case structure for threshold rule test notifications
type ThresholdRuleTestCase struct {
	Name         string
	Values       [][]interface{}
	ExpectAlerts int
	ExpectValue  float64
}

// ThresholdRuleAtLeastOnceValueAbove creates a PostableRule for threshold rule test notifications
func ThresholdRuleAtLeastOnceValueAbove(target float64, recovery *float64) ruletypes.PostableRule {
	return ruletypes.PostableRule{
		AlertName: "test-alert",
		AlertType: ruletypes.AlertTypeMetric,
		RuleType:  ruletypes.RuleTypeThreshold,
		Evaluation: &ruletypes.EvaluationEnvelope{Kind: ruletypes.RollingEvaluation, Spec: ruletypes.RollingWindow{
			EvalWindow: valuer.MustParseTextDuration("5m"),
			Frequency:  valuer.MustParseTextDuration("1m"),
		}},
		Labels: map[string]string{
			"service.name": "frontend",
		},
		Annotations: map[string]string{
			"description": "The configured alert condition was met.",
		},
		Version: "v5",
		SchemaVersion: ruletypes.CurrentSchemaVersion,
		RuleCondition: &ruletypes.RuleCondition{
			MatchType:     ruletypes.AtleastOnce,
			CompareOp:     ruletypes.ValueIsAbove,
			Target:        &target,
			SelectedQuery: "A",
			CompositeQuery: &ruletypes.CompositeQuery{
				QueryType: querytypes.QueryTypeBuilder,
				Queries: []qbtypes.QueryEnvelope{
					{
						Type: qbtypes.QueryTypeBuilder,
						Spec: qbtypes.QueryBuilderQuery[qbtypes.MetricAggregation]{
							Name:         "A",
							StepInterval: qbtypes.Step{Duration: 60 * time.Second},
							Signal:       telemetrytypes.SignalMetrics,

							Aggregations: []qbtypes.MetricAggregation{
								{
									MetricName:       "probe_success",
									TimeAggregation:  metrictypes.TimeAggregationAvg,
									SpaceAggregation: metrictypes.SpaceAggregationAvg,
								},
							},
						},
					},
				},
			},
			Thresholds: &ruletypes.RuleThresholdData{
				Kind: ruletypes.BasicThresholdKind,
				Spec: ruletypes.BasicRuleThresholds{
					{
						Name:           "primary",
						TargetValue:    &target,
						RecoveryTarget: recovery,
						MatchType:      ruletypes.AtleastOnce,
						CompareOp:      ruletypes.ValueIsAbove,
					},
				},
			},
		},
		NotificationSettings: &ruletypes.NotificationSettings{},
	}
}

var (
	// TcTestNotiSendUnmatchedThresholdRule contains test cases for threshold rule test notifications
	TcTestNotiSendUnmatchedThresholdRule = []ThresholdRuleTestCase{
		{
			Name: "return first valid point in case of test notification",
			Values: [][]interface{}{
				{float64(3), "attr", time.Now()},
				{float64(4), "attr", time.Now().Add(1 * time.Minute)},
			},
			ExpectAlerts: 1,
			ExpectValue:  3,
		},
		{
			Name:         "No data in DB so no alerts fired",
			Values:       [][]interface{}{},
			ExpectAlerts: 0,
		},
		{
			Name: "return first valid point in case of test notification skips NaN and Inf",
			Values: [][]interface{}{
				{math.NaN(), "attr", time.Now()},
				{math.Inf(1), "attr", time.Now().Add(1 * time.Minute)},
				{float64(7), "attr", time.Now().Add(2 * time.Minute)},
			},
			ExpectAlerts: 1,
			ExpectValue:  7,
		},
		{
			Name: "If found matching alert with given target value, return the alerting value rather than first valid point",
			Values: [][]interface{}{
				{float64(1), "attr", time.Now()},
				{float64(2), "attr", time.Now().Add(1 * time.Minute)},
				{float64(3), "attr", time.Now().Add(2 * time.Minute)},
				{float64(12), "attr", time.Now().Add(3 * time.Minute)},
			},
			ExpectAlerts: 1,
			ExpectValue:  12,
		},
	}
)
