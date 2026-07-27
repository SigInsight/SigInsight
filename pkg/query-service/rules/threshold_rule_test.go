package rules

import (
	"context"
	"fmt"
	"github.com/SigNoz/signoz/pkg/types/timeseriestypes"
	"log/slog"
	"math"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/SigNoz/signoz/pkg/instrumentation/instrumentationtest"
	"github.com/SigNoz/signoz/pkg/query-service/app/clickhouseReader"
	"github.com/SigNoz/signoz/pkg/query-service/common"
	"github.com/SigNoz/signoz/pkg/query-service/model/querytypes"
	"github.com/SigNoz/signoz/pkg/query-service/utils/labels"
	"github.com/SigNoz/signoz/pkg/telemetrystore"
	"github.com/SigNoz/signoz/pkg/telemetrystore/telemetrystoretest"
	qbtypes "github.com/SigNoz/signoz/pkg/types/querybuildertypes/querybuildertypesv5"
	"github.com/SigNoz/signoz/pkg/types/ruletypes"
	"github.com/SigNoz/signoz/pkg/types/telemetrytypes"
	"github.com/SigNoz/signoz/pkg/valuer"
)

type staticV5Querier struct {
	response *qbtypes.QueryRangeResponse
	err      error
}

func (q staticV5Querier) QueryRange(context.Context, valuer.UUID, *qbtypes.QueryRangeRequest) (*qbtypes.QueryRangeResponse, error) {
	return q.response, q.err
}

func (staticV5Querier) QueryRawStream(
	context.Context,
	valuer.UUID,
	*qbtypes.QueryRangeRequest,
	*qbtypes.RawStream,
) {
}

func v5MetricCompositeQuery(metricName string) *ruletypes.CompositeQuery {
	return &ruletypes.CompositeQuery{
		QueryType: querytypes.QueryTypeBuilder,
		Queries: []qbtypes.QueryEnvelope{{
			Type: qbtypes.QueryTypeBuilder,
			Spec: qbtypes.QueryBuilderQuery[qbtypes.MetricAggregation]{
				Name:         "A",
				Signal:       telemetrytypes.SignalMetrics,
				StepInterval: qbtypes.Step{Duration: time.Minute},
				Aggregations: []qbtypes.MetricAggregation{{MetricName: metricName}},
			},
		}},
	}
}

func v5QueryResponse(series ...*qbtypes.TimeSeries) *qbtypes.QueryRangeResponse {
	return &qbtypes.QueryRangeResponse{
		Type: qbtypes.RequestTypeTimeSeries,
		Data: qbtypes.QueryData{
			Results: []any{
				&qbtypes.TimeSeriesData{
					QueryName: "A",
					Aggregations: []*qbtypes.AggregationBucket{
						{Series: series},
					},
				},
			},
		},
	}
}

func v5SeriesFromRows(rows [][]any, splitSeries bool) []*qbtypes.TimeSeries {
	if len(rows) == 0 {
		return nil
	}

	if !splitSeries {
		series := &qbtypes.TimeSeries{Values: make([]*qbtypes.TimeSeriesValue, 0, len(rows))}
		for _, row := range rows {
			series.Values = append(series.Values, &qbtypes.TimeSeriesValue{
				Value:     row[0].(float64),
				Timestamp: row[2].(time.Time).UnixMilli(),
			})
		}
		return []*qbtypes.TimeSeries{series}
	}

	series := make([]*qbtypes.TimeSeries, 0, len(rows))
	for idx, row := range rows {
		series = append(series, &qbtypes.TimeSeries{
			Labels: []*qbtypes.Label{
				{
					Key:   telemetrytypes.TelemetryFieldKey{Name: "series"},
					Value: idx,
				},
			},
			Values: []*qbtypes.TimeSeriesValue{
				{
					Value:     row[0].(float64),
					Timestamp: row[2].(time.Time).UnixMilli(),
				},
			},
		})
	}
	return series
}

func newRuleStateHistoryTestReader() *clickhouseReader.ClickHouseReader {
	telemetryStore := telemetrystoretest.New(telemetrystore.Config{}, &queryMatcherAny{})
	return clickhouseReader.NewReader(
		slog.Default(),
		nil,
		telemetryStore,
		nil,
		time.Second,
		nil,
		nil,
		clickhouseReader.NewOptions("", "", "archiveNamespace"),
	)
}

func TestRuleSeriesFromTimeSeries(t *testing.T) {
	series := ruleSeriesFromTimeSeries(&qbtypes.TimeSeries{
		Labels: []*qbtypes.Label{
			{Key: telemetrytypes.TelemetryFieldKey{Name: "service.name"}, Value: "api"},
			{Key: telemetrytypes.TelemetryFieldKey{Name: "instance"}, Value: 7},
			nil,
		},
		Values: []*qbtypes.TimeSeriesValue{
			{Timestamp: 1000, Value: 1.5},
			{Timestamp: 2000, Value: 2.5, Partial: true},
			nil,
		},
	})

	assert.Equal(t, map[string]string{"service.name": "api", "instance": "7"}, series.Labels)
	assert.Equal(t, []map[string]string{series.Labels}, series.LabelsArray)
	assert.Equal(t, []timeseriestypes.Point{{Timestamp: 1000, Value: 1.5}}, series.Points)
	assert.Empty(t, ruleSeriesFromTimeSeries(nil).Points)
}

func TestThresholdRuleEval(t *testing.T) {
	postableRule := ruletypes.PostableRule{
		AlertName: "Eval test without recovery target",
		AlertType: ruletypes.AlertTypeMetric,
		RuleType:  ruletypes.RuleTypeThreshold,
		Evaluation: &ruletypes.EvaluationEnvelope{ruletypes.RollingEvaluation, ruletypes.RollingWindow{
			EvalWindow: valuer.MustParseTextDuration("5m"),
			Frequency:  valuer.MustParseTextDuration("1m"),
		}},

		RuleCondition: &ruletypes.RuleCondition{
			CompositeQuery: v5MetricCompositeQuery("probe_success"),
			SelectedQuery:  "A",
		},
	}

	logger := instrumentationtest.New().Logger()

	for idx, c := range tcThresholdRuleEvalNoRecoveryTarget {
		postableRule.RuleCondition.Thresholds = &ruletypes.RuleThresholdData{
			Kind: ruletypes.BasicThresholdKind,
			Spec: ruletypes.BasicRuleThresholds{
				{
					TargetValue: &c.target,
					MatchType:   ruletypes.MatchType(c.matchType),
					CompareOp:   ruletypes.CompareOp(c.compareOp),
				},
			},
		}

		rule, err := NewThresholdRule("69", valuer.GenerateUUID(), &postableRule, nil, nil, logger, WithEvalDelay(valuer.MustParseTextDuration("2m")))
		if err != nil {
			assert.NoError(t, err)
		}

		values := c.values
		for i := range values.Points {
			values.Points[i].Timestamp = time.Now().UnixMilli()
		}

		resultVectors, err := rule.Threshold.Eval(c.values, rule.Unit(), ruletypes.EvalData{
			ActiveAlerts: map[uint64]struct{}{},
		})
		assert.NoError(t, err, "Test case %d", idx)

		// Compare result vectors with expected behavior
		if c.expectAlert {
			assert.NotEmpty(t, resultVectors, "Expected alert but got no result vectors for case %d", idx)
			if len(resultVectors) > 0 {
				found := false
				for _, sample := range resultVectors {
					if sample.V == c.expectedAlertSample.Value {
						found = true
						break
					}
				}
				assert.True(t, found, "Expected alert sample value %.2f not found in result vectors for case %d. Got values: %v", c.expectedAlertSample.Value, idx, getVectorValues(resultVectors))
			}
		} else {
			assert.Empty(t, resultVectors, "Expected no alert but got result vectors for case %d", idx)
		}
	}
}

func TestNormalizeLabelName(t *testing.T) {
	cases := []struct {
		labelName string
		expected  string
	}{
		{
			labelName: "label",
			expected:  "label",
		},
		{
			labelName: "label.with.dots",
			expected:  "label_with_dots",
		},
		{
			labelName: "label-with-dashes",
			expected:  "label_with_dashes",
		},
		{
			labelName: "labelwithnospaces",
			expected:  "labelwithnospaces",
		},
		{
			labelName: "label with spaces",
			expected:  "label_with_spaces",
		},
		{
			labelName: "label with spaces and .dots",
			expected:  "label_with_spaces_and__dots",
		},
		{
			labelName: "label with spaces and -dashes",
			expected:  "label_with_spaces_and__dashes",
		},
	}

	for _, c := range cases {
		assert.Equal(t, c.expected, common.NormalizeLabelName(c.labelName))
	}
}

func TestPrepareLinksToLogs(t *testing.T) {
	postableRule := ruletypes.PostableRule{
		AlertName: "Tricky Condition Tests",
		AlertType: ruletypes.AlertTypeLogs,
		RuleType:  ruletypes.RuleTypeThreshold,
		Evaluation: &ruletypes.EvaluationEnvelope{ruletypes.RollingEvaluation, ruletypes.RollingWindow{
			EvalWindow: valuer.MustParseTextDuration("5m"),
			Frequency:  valuer.MustParseTextDuration("1m"),
		}},
		RuleCondition: &ruletypes.RuleCondition{
			CompositeQuery: &ruletypes.CompositeQuery{
				QueryType: querytypes.QueryTypeBuilder,
				Queries: []qbtypes.QueryEnvelope{
					{
						Type: qbtypes.QueryTypeBuilder,
						Spec: qbtypes.QueryBuilderQuery[qbtypes.LogAggregation]{
							Name:         "A",
							StepInterval: qbtypes.Step{Duration: 1 * time.Minute},
							Aggregations: []qbtypes.LogAggregation{
								{
									Expression: "count()",
								},
							},
							Filter: &qbtypes.Filter{
								Expression: "service.name EXISTS",
							},
							Signal: telemetrytypes.SignalLogs,
						},
					},
				},
			},
			CompareOp:     "4", // Not Equals
			MatchType:     "1", // Once
			Target:        &[]float64{0.0}[0],
			SelectedQuery: "A",
		},
		Version: "v5",
	}

	logger := instrumentationtest.New().Logger()
	postableRule.RuleCondition.Thresholds = &ruletypes.RuleThresholdData{
		Kind: ruletypes.BasicThresholdKind,
		Spec: ruletypes.BasicRuleThresholds{
			{
				TargetValue: postableRule.RuleCondition.Target,
				MatchType:   postableRule.RuleCondition.MatchType,
				CompareOp:   postableRule.RuleCondition.CompareOp,
			},
		},
	}
	rule, err := NewThresholdRule("69", valuer.GenerateUUID(), &postableRule, nil, nil, logger, WithEvalDelay(valuer.MustParseTextDuration("2m")))
	if err != nil {
		assert.NoError(t, err)
	}

	ts := time.UnixMilli(1753527163000)

	link := rule.prepareLinksToLogs(context.Background(), ts, labels.Labels{})
	assert.Contains(t, link, "compositeQuery=%257B%2522queryType%2522%253A%2522builder%2522%252C%2522builder%2522%253A%257B%2522queryData%2522%253A%255B%257B%2522queryName%2522%253A%2522A%2522%252C%2522stepInterval%2522%253A60%252C%2522dataSource%2522%253A%2522logs%2522%252C%2522aggregateOperator%2522%253A%2522noop%2522%252C%2522aggregateAttribute%2522%253A%257B%2522key%2522%253A%2522%2522%252C%2522dataType%2522%253A%2522%2522%252C%2522type%2522%253A%2522%2522%252C%2522isColumn%2522%253Afalse%252C%2522isJSON%2522%253Afalse%257D%252C%2522expression%2522%253A%2522A%2522%252C%2522disabled%2522%253Afalse%252C%2522limit%2522%253A0%252C%2522offset%2522%253A0%252C%2522pageSize%2522%253A0%252C%2522ShiftBy%2522%253A0%252C%2522IsAnomaly%2522%253Afalse%252C%2522QueriesUsedInFormula%2522%253Anull%252C%2522filter%2522%253A%257B%2522expression%2522%253A%2522service.name%2BEXISTS%2522%257D%257D%255D%252C%2522queryFormulas%2522%253A%255B%255D%257D%257D&timeRange=%7B%22start%22%3A1753526700000%2C%22end%22%3A1753527000000%2C%22pageSize%22%3A100%7D&startTime=1753526700000&endTime=1753527000000&options=%7B%22maxLines%22%3A0%2C%22format%22%3A%22%22%2C%22selectColumns%22%3Anull%7D")
}

func TestPrepareLinksToTraces(t *testing.T) {
	postableRule := ruletypes.PostableRule{
		AlertName: "Tricky Condition Tests",
		AlertType: ruletypes.AlertTypeTraces,
		RuleType:  ruletypes.RuleTypeThreshold,
		Evaluation: &ruletypes.EvaluationEnvelope{ruletypes.RollingEvaluation, ruletypes.RollingWindow{
			EvalWindow: valuer.MustParseTextDuration("5m"),
			Frequency:  valuer.MustParseTextDuration("1m"),
		}},
		RuleCondition: &ruletypes.RuleCondition{
			CompositeQuery: &ruletypes.CompositeQuery{
				QueryType: querytypes.QueryTypeBuilder,
				Queries: []qbtypes.QueryEnvelope{
					{
						Type: qbtypes.QueryTypeBuilder,
						Spec: qbtypes.QueryBuilderQuery[qbtypes.TraceAggregation]{
							Name:         "A",
							StepInterval: qbtypes.Step{Duration: 1 * time.Minute},
							Aggregations: []qbtypes.TraceAggregation{
								{
									Expression: "count()",
								},
							},
							Filter: &qbtypes.Filter{
								Expression: "service.name EXISTS",
							},
							Signal: telemetrytypes.SignalTraces,
						},
					},
				},
			},
			CompareOp:     "4", // Not Equals
			MatchType:     "1", // Once
			Target:        &[]float64{0.0}[0],
			SelectedQuery: "A",
		},
		Version: "v5",
	}

	logger := instrumentationtest.New().Logger()
	postableRule.RuleCondition.Thresholds = &ruletypes.RuleThresholdData{
		Kind: ruletypes.BasicThresholdKind,
		Spec: ruletypes.BasicRuleThresholds{
			{
				TargetValue: postableRule.RuleCondition.Target,
				MatchType:   postableRule.RuleCondition.MatchType,
				CompareOp:   postableRule.RuleCondition.CompareOp,
			},
		},
	}
	rule, err := NewThresholdRule("69", valuer.GenerateUUID(), &postableRule, nil, nil, logger, WithEvalDelay(valuer.MustParseTextDuration("2m")))
	if err != nil {
		assert.NoError(t, err)
	}

	ts := time.UnixMilli(1753527163000)

	link := rule.prepareLinksToTraces(context.Background(), ts, labels.Labels{})
	assert.Contains(t, link, "compositeQuery=%257B%2522queryType%2522%253A%2522builder%2522%252C%2522builder%2522%253A%257B%2522queryData%2522%253A%255B%257B%2522queryName%2522%253A%2522A%2522%252C%2522stepInterval%2522%253A60%252C%2522dataSource%2522%253A%2522traces%2522%252C%2522aggregateOperator%2522%253A%2522noop%2522%252C%2522aggregateAttribute%2522%253A%257B%2522key%2522%253A%2522%2522%252C%2522dataType%2522%253A%2522%2522%252C%2522type%2522%253A%2522%2522%252C%2522isColumn%2522%253Afalse%252C%2522isJSON%2522%253Afalse%257D%252C%2522expression%2522%253A%2522A%2522%252C%2522disabled%2522%253Afalse%252C%2522limit%2522%253A0%252C%2522offset%2522%253A0%252C%2522pageSize%2522%253A0%252C%2522ShiftBy%2522%253A0%252C%2522IsAnomaly%2522%253Afalse%252C%2522QueriesUsedInFormula%2522%253Anull%252C%2522filter%2522%253A%257B%2522expression%2522%253A%2522service.name%2BEXISTS%2522%257D%257D%255D%252C%2522queryFormulas%2522%253A%255B%255D%257D%257D&timeRange=%7B%22start%22%3A1753526700000000000%2C%22end%22%3A1753527000000000000%2C%22pageSize%22%3A100%7D&startTime=1753526700000000000&endTime=1753527000000000000&options=%7B%22maxLines%22%3A0%2C%22format%22%3A%22%22%2C%22selectColumns%22%3Anull%7D")
}

func TestThresholdRuleLabelNormalization(t *testing.T) {
	postableRule := ruletypes.PostableRule{
		AlertName: "Tricky Condition Tests",
		AlertType: ruletypes.AlertTypeMetric,
		RuleType:  ruletypes.RuleTypeThreshold,
		Evaluation: &ruletypes.EvaluationEnvelope{ruletypes.RollingEvaluation, ruletypes.RollingWindow{
			EvalWindow: valuer.MustParseTextDuration("5m"),
			Frequency:  valuer.MustParseTextDuration("1m"),
		}},
		RuleCondition: &ruletypes.RuleCondition{
			CompositeQuery: v5MetricCompositeQuery("probe_success"),
			SelectedQuery:  "A",
		},
	}

	cases := []struct {
		values      timeseriestypes.Series
		expectAlert bool
		compareOp   string
		matchType   string
		target      float64
	}{
		// Test cases for Equals Always
		{
			values: timeseriestypes.Series{
				Points: []timeseriestypes.Point{
					{Value: 0.0},
					{Value: 0.0},
					{Value: 0.0},
					{Value: 0.0},
					{Value: 0.0},
				},
				Labels: map[string]string{
					"service.name": "frontend",
				},
				LabelsArray: []map[string]string{
					{
						"service.name": "frontend",
					},
				},
			},
			expectAlert: true,
			compareOp:   "3", // Equals
			matchType:   "2", // Always
			target:      0.0,
		},
	}

	logger := instrumentationtest.New().Logger()

	for idx, c := range cases {
		postableRule.RuleCondition.Thresholds = &ruletypes.RuleThresholdData{
			Kind: ruletypes.BasicThresholdKind,
			Spec: ruletypes.BasicRuleThresholds{
				{
					TargetValue: &c.target,
					MatchType:   ruletypes.MatchType(c.matchType),
					CompareOp:   ruletypes.CompareOp(c.compareOp),
				},
			},
		}

		rule, err := NewThresholdRule("69", valuer.GenerateUUID(), &postableRule, nil, nil, logger, WithEvalDelay(valuer.MustParseTextDuration("2m")))
		if err != nil {
			assert.NoError(t, err)
		}

		values := c.values
		for i := range values.Points {
			values.Points[i].Timestamp = time.Now().UnixMilli()
		}

		vector, err := rule.Threshold.Eval(c.values, rule.Unit(), ruletypes.EvalData{})
		assert.NoError(t, err)

		for name, value := range c.values.Labels {
			for _, sample := range vector {
				assert.Equal(t, value, sample.Metric.Get(name))
			}
		}

		// Get result vectors from threshold evaluation
		resultVectors, err := rule.Threshold.Eval(c.values, rule.Unit(), ruletypes.EvalData{})
		assert.NoError(t, err, "Test case %d", idx)

		// Compare result vectors with expected behavior
		if c.expectAlert {
			assert.NotEmpty(t, resultVectors, "Expected alert but got no result vectors for case %d", idx)
			// For this test, we don't have expectedAlertSample, so just verify vectors exist
		} else {
			assert.Empty(t, resultVectors, "Expected no alert but got result vectors for case %d", idx)
		}
	}
}

func TestThresholdRulePrepareQueryRange(t *testing.T) {
	const query = "SELECT 1 >= {{.start_timestamp_ms}} AND 1 <= {{.end_timestamp_ms}}"

	postableRule := ruletypes.PostableRule{
		AlertName: "Test query range",
		AlertType: ruletypes.AlertTypeMetric,
		RuleType:  ruletypes.RuleTypeThreshold,
		Evaluation: &ruletypes.EvaluationEnvelope{Kind: ruletypes.RollingEvaluation, Spec: ruletypes.RollingWindow{
			EvalWindow: valuer.MustParseTextDuration("5m"),
			Frequency:  valuer.MustParseTextDuration("1m"),
		}},
		RuleCondition: &ruletypes.RuleCondition{
			CompositeQuery: &ruletypes.CompositeQuery{
				QueryType: querytypes.QueryTypeClickHouseSQL,
				Queries: []qbtypes.QueryEnvelope{
					{
						Type: qbtypes.QueryTypeClickHouseSQL,
						Spec: qbtypes.ClickHouseQuery{Name: "A", Query: query},
					},
				},
			},
			Thresholds: &ruletypes.RuleThresholdData{
				Kind: ruletypes.BasicThresholdKind,
				Spec: ruletypes.BasicRuleThresholds{{}},
			},
		},
		Version: "v5",
	}

	ts := time.Unix(1717205987, 0)
	cases := []struct {
		name          string
		opts          []RuleOption
		expectedStart uint64
		expectedEnd   uint64
	}{
		{
			name:          "without eval delay",
			expectedStart: 1717205640000,
			expectedEnd:   1717205940000,
		},
		{
			name:          "with eval delay",
			opts:          []RuleOption{WithEvalDelay(valuer.MustParseTextDuration("2m"))},
			expectedStart: 1717205520000,
			expectedEnd:   1717205820000,
		},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			rule, err := NewThresholdRule("69", valuer.GenerateUUID(), &postableRule, nil, nil, instrumentationtest.New().Logger(), c.opts...)
			require.NoError(t, err)

			params, err := rule.prepareQueryRange(context.Background(), ts)
			require.NoError(t, err)
			assert.Equal(t, c.expectedStart, params.Start)
			assert.Equal(t, c.expectedEnd, params.End)
			assert.Equal(t, qbtypes.RequestTypeTimeSeries, params.RequestType)
			require.Len(t, params.CompositeQuery.Queries, 1)
			assert.Equal(t, qbtypes.ClickHouseQuery{Name: "A", Query: query}, params.CompositeQuery.Queries[0].Spec)

			secondParams, err := rule.prepareQueryRange(context.Background(), ts)
			require.NoError(t, err)
			assert.Equal(t, params, secondParams)
		})
	}
}

func TestThresholdRuleUnitCombinations(t *testing.T) {
	postableRule := ruletypes.PostableRule{
		AlertName: "Units test",
		AlertType: ruletypes.AlertTypeMetric,
		RuleType:  ruletypes.RuleTypeThreshold,
		Evaluation: &ruletypes.EvaluationEnvelope{Kind: ruletypes.RollingEvaluation, Spec: ruletypes.RollingWindow{
			EvalWindow: valuer.MustParseTextDuration("5m"),
			Frequency:  valuer.MustParseTextDuration("1m"),
		}},
		RuleCondition: &ruletypes.RuleCondition{
			CompositeQuery: v5MetricCompositeQuery("signoz_calls_total"),
			SelectedQuery:  "A",
		},
		Version: "v5",
	}

	cases := []struct {
		targetUnit   string
		yAxisUnit    string
		values       [][]interface{}
		expectAlerts int
		compareOp    string
		matchType    string
		target       float64
		summaryAny   []string
	}{
		{
			targetUnit: "s",
			yAxisUnit:  "ns",
			values: [][]interface{}{
				{float64(572588400), "attr", time.Now()},                              // 0.57 seconds
				{float64(572386400), "attr", time.Now().Add(1 * time.Second)},         // 0.57 seconds
				{float64(300947400), "attr", time.Now().Add(2 * time.Second)},         // 0.3 seconds
				{float64(299316000), "attr", time.Now().Add(3 * time.Second)},         // 0.3 seconds
				{float64(66640400.00000001), "attr", time.Now().Add(4 * time.Second)}, // 0.06 seconds
			},
			expectAlerts: 0,
			compareOp:    "1", // Above
			matchType:    "1", // Once
			target:       1,   // 1 second
		},
		{
			targetUnit: "ms",
			yAxisUnit:  "ns",
			values: [][]interface{}{
				{float64(572588400), "attr", time.Now()},                              // 572.58 ms
				{float64(572386400), "attr", time.Now().Add(1 * time.Second)},         // 572.38 ms
				{float64(300947400), "attr", time.Now().Add(2 * time.Second)},         // 300.94 ms
				{float64(299316000), "attr", time.Now().Add(3 * time.Second)},         // 299.31 ms
				{float64(66640400.00000001), "attr", time.Now().Add(4 * time.Second)}, // 66.64 ms
			},
			expectAlerts: 4,
			compareOp:    "1", // Above
			matchType:    "1", // Once
			target:       200, // 200 ms
			summaryAny: []string{
				"observed metric value is 299 ms",
				"the observed metric value is 573 ms",
				"the observed metric value is 572 ms",
				"the observed metric value is 301 ms",
			},
		},
		{
			targetUnit: "decgbytes",
			yAxisUnit:  "bytes",
			values: [][]interface{}{
				{float64(2863284053), "attr", time.Now()},                             // 2.86 GB
				{float64(2863388842), "attr", time.Now().Add(1 * time.Second)},        // 2.86 GB
				{float64(300947400), "attr", time.Now().Add(2 * time.Second)},         // 0.3 GB
				{float64(299316000), "attr", time.Now().Add(3 * time.Second)},         // 0.3 GB
				{float64(66640400.00000001), "attr", time.Now().Add(4 * time.Second)}, // 66.64 MB
			},
			expectAlerts: 0,
			compareOp:    "1", // Above
			matchType:    "1", // Once
			target:       200, // 200 GB
		},
		{
			targetUnit: "decgbytes",
			yAxisUnit:  "By",
			values: [][]interface{}{
				{float64(2863284053), "attr", time.Now()},                             // 2.86 GB
				{float64(2863388842), "attr", time.Now().Add(1 * time.Second)},        // 2.86 GB
				{float64(300947400), "attr", time.Now().Add(2 * time.Second)},         // 0.3 GB
				{float64(299316000), "attr", time.Now().Add(3 * time.Second)},         // 0.3 GB
				{float64(66640400.00000001), "attr", time.Now().Add(4 * time.Second)}, // 66.64 MB
			},
			expectAlerts: 0,
			compareOp:    "1", // Above
			matchType:    "1", // Once
			target:       200, // 200 GB
		},
		{
			targetUnit: "h",
			yAxisUnit:  "min",
			values: [][]interface{}{
				{float64(55), "attr", time.Now()},                      // 55 minutes
				{float64(57), "attr", time.Now().Add(1 * time.Minute)}, // 57 minutes
				{float64(30), "attr", time.Now().Add(2 * time.Minute)}, // 30 minutes
				{float64(29), "attr", time.Now().Add(3 * time.Minute)}, // 29 minutes
			},
			expectAlerts: 0,
			compareOp:    "1", // Above
			matchType:    "1", // Once
			target:       1,   // 1 hour
		},
	}

	logger := instrumentationtest.New().Logger()

	for idx, c := range cases {
		postableRule.RuleCondition.CompareOp = ruletypes.CompareOp(c.compareOp)
		postableRule.RuleCondition.MatchType = ruletypes.MatchType(c.matchType)
		postableRule.RuleCondition.Target = &c.target
		postableRule.RuleCondition.CompositeQuery.Unit = c.yAxisUnit
		postableRule.RuleCondition.TargetUnit = c.targetUnit
		postableRule.RuleCondition.Thresholds = &ruletypes.RuleThresholdData{
			Kind: ruletypes.BasicThresholdKind,
			Spec: ruletypes.BasicRuleThresholds{
				{
					Name:        postableRule.AlertName,
					TargetValue: &c.target,
					TargetUnit:  c.targetUnit,
					MatchType:   ruletypes.MatchType(c.matchType),
					CompareOp:   ruletypes.CompareOp(c.compareOp),
				},
			},
		}
		postableRule.Annotations = map[string]string{
			"description": "This alert is fired when the defined metric (current value: {{$value}}) crosses the threshold ({{$threshold}})",
			"summary":     "The rule threshold is set to {{$threshold}}, and the observed metric value is {{$value}}",
		}

		querier := staticV5Querier{
			response: v5QueryResponse(v5SeriesFromRows(c.values, true)...),
		}
		rule, err := NewThresholdRule("69", valuer.GenerateUUID(), &postableRule, newRuleStateHistoryTestReader(), querier, logger)
		if err != nil {
			assert.NoError(t, err)
		}

		alertsFound, err := rule.Eval(context.Background(), time.Now())
		if err != nil {
			assert.NoError(t, err)
		}

		assert.Equal(t, c.expectAlerts, alertsFound, "case %d", idx)
		if c.expectAlerts != 0 {
			foundCount := 0
			for _, item := range rule.Active {
				for _, summary := range c.summaryAny {
					if strings.Contains(item.Annotations.Get("summary"), summary) {
						foundCount++
						break
					}
				}
			}
			assert.Equal(t, c.expectAlerts, foundCount, "case %d", idx)
		}
	}
}

func TestThresholdRuleNoData(t *testing.T) {
	postableRule := ruletypes.PostableRule{
		AlertName: "No data test",
		AlertType: ruletypes.AlertTypeMetric,
		RuleType:  ruletypes.RuleTypeThreshold,
		Evaluation: &ruletypes.EvaluationEnvelope{Kind: ruletypes.RollingEvaluation, Spec: ruletypes.RollingWindow{
			EvalWindow: valuer.MustParseTextDuration("5m"),
			Frequency:  valuer.MustParseTextDuration("1m"),
		}},
		RuleCondition: &ruletypes.RuleCondition{
			CompositeQuery: v5MetricCompositeQuery("signoz_calls_total"),
			SelectedQuery:  "A",
			AlertOnAbsent:  true,
		},
		Version: "v5",
	}

	cases := []struct {
		values       [][]interface{}
		expectNoData bool
	}{
		{
			values:       [][]interface{}{},
			expectNoData: true,
		},
	}

	logger := instrumentationtest.New().Logger()

	for idx, c := range cases {
		var target float64 = 0
		postableRule.RuleCondition.Thresholds = &ruletypes.RuleThresholdData{
			Kind: ruletypes.BasicThresholdKind,
			Spec: ruletypes.BasicRuleThresholds{
				{
					Name:        postableRule.AlertName,
					TargetValue: &target,
					MatchType:   ruletypes.AtleastOnce,
					CompareOp:   ruletypes.ValueIsEq,
				},
			},
		}
		postableRule.Annotations = map[string]string{
			"description": "This alert is fired when the defined metric (current value: {{$value}}) crosses the threshold ({{$threshold}})",
			"summary":     "The rule threshold is set to {{$threshold}}, and the observed metric value is {{$value}}",
		}

		querier := staticV5Querier{
			response: v5QueryResponse(v5SeriesFromRows(c.values, true)...),
		}
		rule, err := NewThresholdRule("69", valuer.GenerateUUID(), &postableRule, newRuleStateHistoryTestReader(), querier, logger)
		if err != nil {
			assert.NoError(t, err)
		}

		alertsFound, err := rule.Eval(context.Background(), time.Now())
		if err != nil {
			assert.NoError(t, err)
		}

		assert.Equal(t, 1, alertsFound, "case %d", idx)
		for _, item := range rule.Active {
			if c.expectNoData {
				assert.True(t, strings.Contains(item.Labels.Get(labels.AlertNameLabel), "[No data]"), "case %d", idx)
			} else {
				assert.False(t, strings.Contains(item.Labels.Get(labels.AlertNameLabel), "[No data]"), "case %d", idx)
			}
		}
	}
}

func TestThresholdRuleTracesLink(t *testing.T) {
	postableRule := ruletypes.PostableRule{
		AlertName: "Traces link test",
		AlertType: ruletypes.AlertTypeTraces,
		RuleType:  ruletypes.RuleTypeThreshold,
		Evaluation: &ruletypes.EvaluationEnvelope{Kind: ruletypes.RollingEvaluation, Spec: ruletypes.RollingWindow{
			EvalWindow: valuer.MustParseTextDuration("5m"),
			Frequency:  valuer.MustParseTextDuration("1m"),
		}},
		RuleCondition: &ruletypes.RuleCondition{
			SelectedQuery: "A",
			CompositeQuery: &ruletypes.CompositeQuery{
				QueryType: querytypes.QueryTypeBuilder,
				Queries: []qbtypes.QueryEnvelope{
					{
						Type: qbtypes.QueryTypeBuilder,
						Spec: qbtypes.QueryBuilderQuery[qbtypes.TraceAggregation]{
							Name:         "A",
							StepInterval: qbtypes.Step{Duration: time.Minute},
							Aggregations: []qbtypes.TraceAggregation{
								{Expression: "p95(durationNano)"},
							},
							Filter: &qbtypes.Filter{
								Expression: "httpMethod = 'GET'",
							},
							Signal: telemetrytypes.SignalTraces,
						},
					},
				},
			},
		},
		Version: "v5",
	}
	logger := instrumentationtest.New().Logger()

	for idx, c := range testCases {
		postableRule.RuleCondition.CompareOp = ruletypes.CompareOp(c.compareOp)
		postableRule.RuleCondition.MatchType = ruletypes.MatchType(c.matchType)
		postableRule.RuleCondition.Target = &c.target
		postableRule.RuleCondition.CompositeQuery.Unit = c.yAxisUnit
		postableRule.RuleCondition.TargetUnit = c.targetUnit
		postableRule.RuleCondition.Thresholds = &ruletypes.RuleThresholdData{
			Kind: ruletypes.BasicThresholdKind,
			Spec: ruletypes.BasicRuleThresholds{
				{
					Name:        postableRule.AlertName,
					TargetValue: &c.target,
					TargetUnit:  c.targetUnit,
					MatchType:   ruletypes.MatchType(c.matchType),
					CompareOp:   ruletypes.CompareOp(c.compareOp),
				},
			},
		}
		postableRule.Annotations = map[string]string{
			"description": "This alert is fired when the defined metric (current value: {{$value}}) crosses the threshold ({{$threshold}})",
			"summary":     "The rule threshold is set to {{$threshold}}, and the observed metric value is {{$value}}",
		}

		querier := staticV5Querier{
			response: v5QueryResponse(v5SeriesFromRows(c.values, true)...),
		}
		rule, err := NewThresholdRule("69", valuer.GenerateUUID(), &postableRule, newRuleStateHistoryTestReader(), querier, logger)
		if err != nil {
			assert.NoError(t, err)
		}

		alertsFound, err := rule.Eval(context.Background(), time.Now())
		if err != nil {
			assert.NoError(t, err)
		}

		if c.expectAlerts == 0 {
			assert.Equal(t, 0, alertsFound, "case %d", idx)
		} else {
			assert.Equal(t, c.expectAlerts, alertsFound, "case %d", idx)
			for _, item := range rule.Active {
				for name, value := range item.Annotations.Map() {
					if name == "related_traces" {
						assert.NotEmpty(t, value, "case %d", idx)
						assert.Contains(t, value, "GET")
					}
				}
			}
		}
	}
}

func TestThresholdRuleLogsLink(t *testing.T) {
	postableRule := ruletypes.PostableRule{
		AlertName: "Logs link test",
		AlertType: ruletypes.AlertTypeLogs,
		RuleType:  ruletypes.RuleTypeThreshold,
		Evaluation: &ruletypes.EvaluationEnvelope{Kind: ruletypes.RollingEvaluation, Spec: ruletypes.RollingWindow{
			EvalWindow: valuer.MustParseTextDuration("5m"),
			Frequency:  valuer.MustParseTextDuration("1m"),
		}},
		RuleCondition: &ruletypes.RuleCondition{
			SelectedQuery: "A",
			CompositeQuery: &ruletypes.CompositeQuery{
				QueryType: querytypes.QueryTypeBuilder,
				Queries: []qbtypes.QueryEnvelope{
					{
						Type: qbtypes.QueryTypeBuilder,
						Spec: qbtypes.QueryBuilderQuery[qbtypes.LogAggregation]{
							Name:         "A",
							StepInterval: qbtypes.Step{Duration: time.Minute},
							Aggregations: []qbtypes.LogAggregation{
								{Expression: "count_distinct(component)"},
							},
							Filter: &qbtypes.Filter{
								Expression: "k8s.container.name = 'testcontainer'",
							},
							Signal: telemetrytypes.SignalLogs,
						},
					},
				},
			},
		},
		Version: "v5",
	}
	logger := instrumentationtest.New().Logger()

	for idx, c := range testCases {
		postableRule.RuleCondition.CompareOp = ruletypes.CompareOp(c.compareOp)
		postableRule.RuleCondition.MatchType = ruletypes.MatchType(c.matchType)
		postableRule.RuleCondition.Target = &c.target
		postableRule.RuleCondition.CompositeQuery.Unit = c.yAxisUnit
		postableRule.RuleCondition.TargetUnit = c.targetUnit
		postableRule.RuleCondition.Thresholds = &ruletypes.RuleThresholdData{
			Kind: ruletypes.BasicThresholdKind,
			Spec: ruletypes.BasicRuleThresholds{
				{
					Name:        postableRule.AlertName,
					TargetValue: &c.target,
					TargetUnit:  c.targetUnit,
					MatchType:   ruletypes.MatchType(c.matchType),
					CompareOp:   ruletypes.CompareOp(c.compareOp),
				},
			},
		}
		postableRule.Annotations = map[string]string{
			"description": "This alert is fired when the defined metric (current value: {{$value}}) crosses the threshold ({{$threshold}})",
			"summary":     "The rule threshold is set to {{$threshold}}, and the observed metric value is {{$value}}",
		}

		querier := staticV5Querier{
			response: v5QueryResponse(v5SeriesFromRows(c.values, true)...),
		}
		rule, err := NewThresholdRule("69", valuer.GenerateUUID(), &postableRule, newRuleStateHistoryTestReader(), querier, logger)
		if err != nil {
			assert.NoError(t, err)
		}

		alertsFound, err := rule.Eval(context.Background(), time.Now())
		if err != nil {
			assert.NoError(t, err)
		}

		if c.expectAlerts == 0 {
			assert.Equal(t, 0, alertsFound, "case %d", idx)
		} else {
			assert.Equal(t, c.expectAlerts, alertsFound, "case %d", idx)
			for _, item := range rule.Active {
				for name, value := range item.Annotations.Map() {
					if name == "related_logs" {
						assert.NotEmpty(t, value, "case %d", idx)
						assert.Contains(t, value, "testcontainer")
					}
				}
			}
		}
	}
}

func TestThresholdRulePreservesTimeShiftFunction(t *testing.T) {
	target := float64(10)
	query := qbtypes.QueryEnvelope{
		Type: qbtypes.QueryTypeBuilder,
		Spec: qbtypes.QueryBuilderQuery[qbtypes.LogAggregation]{
			Name:         "A",
			StepInterval: qbtypes.Step{Duration: time.Minute},
			Signal:       telemetrytypes.SignalLogs,
			Aggregations: []qbtypes.LogAggregation{{Expression: "count_distinct(component)"}},
			Functions: []qbtypes.Function{
				{
					Name: qbtypes.FunctionNameTimeShift,
					Args: []qbtypes.FunctionArg{{Value: float64(10)}},
				},
			},
		},
	}
	postableRule := ruletypes.PostableRule{
		AlertName: "Logs link test",
		AlertType: ruletypes.AlertTypeLogs,
		RuleType:  ruletypes.RuleTypeThreshold,
		Evaluation: &ruletypes.EvaluationEnvelope{Kind: ruletypes.RollingEvaluation, Spec: ruletypes.RollingWindow{
			EvalWindow: valuer.MustParseTextDuration("5m"),
			Frequency:  valuer.MustParseTextDuration("1m"),
		}},
		RuleCondition: &ruletypes.RuleCondition{
			Thresholds: &ruletypes.RuleThresholdData{
				Kind: ruletypes.BasicThresholdKind,
				Spec: ruletypes.BasicRuleThresholds{
					{
						TargetValue: &target,
						CompareOp:   ruletypes.ValueAboveOrEq,
						MatchType:   ruletypes.AtleastOnce,
					},
				},
			},
			CompositeQuery: &ruletypes.CompositeQuery{
				QueryType: querytypes.QueryTypeBuilder,
				Queries:   []qbtypes.QueryEnvelope{query},
			},
		},
		Version: "v5",
	}

	rule, err := NewThresholdRule("69", valuer.GenerateUUID(), &postableRule, nil, nil, instrumentationtest.New().Logger())
	require.NoError(t, err)

	params, err := rule.prepareQueryRange(context.Background(), time.Now())
	require.NoError(t, err)
	require.Len(t, params.CompositeQuery.Queries, 1)
	assert.Equal(t, query, params.CompositeQuery.Queries[0])
}

func TestMultipleThresholdRule(t *testing.T) {
	postableRule := ruletypes.PostableRule{
		AlertName: "Multiple threshold test",
		AlertType: ruletypes.AlertTypeMetric,
		RuleType:  ruletypes.RuleTypeThreshold,
		Evaluation: &ruletypes.EvaluationEnvelope{Kind: ruletypes.RollingEvaluation, Spec: ruletypes.RollingWindow{
			EvalWindow: valuer.MustParseTextDuration("5m"),
			Frequency:  valuer.MustParseTextDuration("1m"),
		}},
		RuleCondition: &ruletypes.RuleCondition{
			CompositeQuery: v5MetricCompositeQuery("signoz_calls_total"),
			SelectedQuery:  "A",
		},
		Version: "v5",
	}
	cases := []struct {
		targetUnit   string
		yAxisUnit    string
		values       [][]interface{}
		expectAlerts int
		compareOp    string
		matchType    string
		target       float64
		secondTarget float64
		summaryAny   []string
	}{
		{
			targetUnit: "s",
			yAxisUnit:  "ns",
			values: [][]interface{}{
				{float64(572588400), "attr", time.Now()},                              // 0.57 seconds
				{float64(572386400), "attr", time.Now().Add(1 * time.Second)},         // 0.57 seconds
				{float64(300947400), "attr", time.Now().Add(2 * time.Second)},         // 0.3 seconds
				{float64(299316000), "attr", time.Now().Add(3 * time.Second)},         // 0.3 seconds
				{float64(66640400.00000001), "attr", time.Now().Add(4 * time.Second)}, // 0.06 seconds
			},
			expectAlerts: 2,
			compareOp:    "1", // Above
			matchType:    "1", // Once
			target:       1,   // 1 second
			secondTarget: .5,
			summaryAny: []string{
				"observed metric value is 573 ms",
				"observed metric value is 572 ms",
			},
		},
		{
			targetUnit: "ms",
			yAxisUnit:  "ns",
			values: [][]interface{}{
				{float64(572588400), "attr", time.Now()},                              // 572.58 ms
				{float64(572386400), "attr", time.Now().Add(1 * time.Second)},         // 572.38 ms
				{float64(300947400), "attr", time.Now().Add(2 * time.Second)},         // 300.94 ms
				{float64(299316000), "attr", time.Now().Add(3 * time.Second)},         // 299.31 ms
				{float64(66640400.00000001), "attr", time.Now().Add(4 * time.Second)}, // 66.64 ms
			},
			expectAlerts: 6,   // Expects 6 values exceed 200ms (572.58, 572.38, 300.94, 299.31) + 2 values exceed 500ms (572.58, 572.38)
			compareOp:    "1", // Above
			matchType:    "1", // Once
			target:       200, // 200 ms
			secondTarget: 500,
			summaryAny: []string{
				"observed metric value is 299 ms",
				"the observed metric value is 573 ms",
				"the observed metric value is 572 ms",
				"the observed metric value is 301 ms",
			},
		},
		{
			targetUnit: "decgbytes",
			yAxisUnit:  "bytes",
			values: [][]interface{}{
				{float64(2863284053), "attr", time.Now()},                             // 2.86 GB
				{float64(2863388842), "attr", time.Now().Add(1 * time.Second)},        // 2.86 GB
				{float64(300947400), "attr", time.Now().Add(2 * time.Second)},         // 0.3 GB
				{float64(299316000), "attr", time.Now().Add(3 * time.Second)},         // 0.3 GB
				{float64(66640400.00000001), "attr", time.Now().Add(4 * time.Second)}, // 66.64 MB
			},
			expectAlerts: 2,
			compareOp:    "1", // Above
			matchType:    "1", // Once
			target:       200, // 200 GB
			secondTarget: 2,   // 2GB
			summaryAny: []string{
				"observed metric value is 2.7 GiB",
				"the observed metric value is 0.3 GB",
			},
		},
	}

	logger := instrumentationtest.New().Logger()

	for idx, c := range cases {
		postableRule.RuleCondition.CompareOp = ruletypes.CompareOp(c.compareOp)
		postableRule.RuleCondition.MatchType = ruletypes.MatchType(c.matchType)
		postableRule.RuleCondition.Target = &c.target
		postableRule.RuleCondition.CompositeQuery.Unit = c.yAxisUnit
		postableRule.RuleCondition.TargetUnit = c.targetUnit
		postableRule.RuleCondition.Thresholds = &ruletypes.RuleThresholdData{
			Kind: ruletypes.BasicThresholdKind,
			Spec: ruletypes.BasicRuleThresholds{
				{
					Name:        "first_threshold",
					TargetValue: &c.target,
					TargetUnit:  c.targetUnit,
					MatchType:   ruletypes.MatchType(c.matchType),
					CompareOp:   ruletypes.CompareOp(c.compareOp),
				},
				{
					Name:        "second_threshold",
					TargetValue: &c.secondTarget,
					TargetUnit:  c.targetUnit,
					MatchType:   ruletypes.MatchType(c.matchType),
					CompareOp:   ruletypes.CompareOp(c.compareOp),
				},
			},
		}
		postableRule.Annotations = map[string]string{
			"description": "This alert is fired when the defined metric (current value: {{$value}}) crosses the threshold ({{$threshold}})",
			"summary":     "The rule threshold is set to {{$threshold}}, and the observed metric value is {{$value}}",
		}

		querier := staticV5Querier{
			response: v5QueryResponse(v5SeriesFromRows(c.values, true)...),
		}
		rule, err := NewThresholdRule("69", valuer.GenerateUUID(), &postableRule, newRuleStateHistoryTestReader(), querier, logger)
		if err != nil {
			assert.NoError(t, err)
		}

		alertsFound, err := rule.Eval(context.Background(), time.Now())
		if err != nil {
			assert.NoError(t, err)
		}

		assert.Equal(t, c.expectAlerts, alertsFound, "case %d", idx)
		if c.expectAlerts != 0 {
			foundCount := 0
			for _, item := range rule.Active {
				for _, summary := range c.summaryAny {
					if strings.Contains(item.Annotations.Get("summary"), summary) {
						foundCount++
						break
					}
				}
			}
			assert.Equal(t, c.expectAlerts, foundCount, "case %d", idx)
		}
	}
}

func TestThresholdRuleEval_BasicCases(t *testing.T) {
	postableRule := ruletypes.PostableRule{
		AlertName: "Eval Recovery Threshold Test",
		AlertType: ruletypes.AlertTypeMetric,
		RuleType:  ruletypes.RuleTypeThreshold,
		Evaluation: &ruletypes.EvaluationEnvelope{Kind: ruletypes.RollingEvaluation, Spec: ruletypes.RollingWindow{
			EvalWindow: valuer.MustParseTextDuration("5m"),
			Frequency:  valuer.MustParseTextDuration("1m"),
		}},
		RuleCondition: &ruletypes.RuleCondition{
			CompositeQuery: v5MetricCompositeQuery("probe_success"),
			SelectedQuery:  "A",
		},
	}

	runEvalTests(t, postableRule, tcThresholdRuleEval)
}

func TestThresholdRuleEval_MatchPlusCompareOps(t *testing.T) {
	postableRule := ruletypes.PostableRule{
		AlertName: "Eval Match Plus Compare Ops Threshold Test",
		AlertType: ruletypes.AlertTypeMetric,
		RuleType:  ruletypes.RuleTypeThreshold,
		Evaluation: &ruletypes.EvaluationEnvelope{Kind: ruletypes.RollingEvaluation, Spec: ruletypes.RollingWindow{
			EvalWindow: valuer.MustParseTextDuration("5m"),
			Frequency:  valuer.MustParseTextDuration("1m"),
		}},
		RuleCondition: &ruletypes.RuleCondition{
			CompositeQuery: v5MetricCompositeQuery("probe_success"),
			SelectedQuery:  "A",
		},
	}

	runEvalTests(t, postableRule, tcThresholdRuleEvalMatchPlusCompareOps)
}

// TestThresholdRuleEval_SendUnmatchedBypassesRecovery tests the case where the sendUnmatched is true and the recovery target is met.
// In this case, the rule should return the first sample as sendUnmatched is supposed to be used in tests and in case of tests
// recovery target is expected to be present. This test make sure this behavior is working as expected.
func TestThresholdRuleEval_SendUnmatchedBypassesRecovery(t *testing.T) {
	target := 10.0
	recovery := 4.0

	postableRule := ruletypes.PostableRule{
		AlertName: "Send unmatched bypass recovery",
		AlertType: ruletypes.AlertTypeMetric,
		RuleType:  ruletypes.RuleTypeThreshold,
		Evaluation: &ruletypes.EvaluationEnvelope{Kind: ruletypes.RollingEvaluation, Spec: ruletypes.RollingWindow{
			EvalWindow: valuer.MustParseTextDuration("5m"),
			Frequency:  valuer.MustParseTextDuration("1m"),
		}},
		RuleCondition: &ruletypes.RuleCondition{
			CompositeQuery: v5MetricCompositeQuery("probe_success"),
			SelectedQuery:  "A",
		},
	}

	postableRule.RuleCondition.Thresholds = &ruletypes.RuleThresholdData{
		Kind: ruletypes.BasicThresholdKind,
		Spec: ruletypes.BasicRuleThresholds{
			{
				Name:           "primary",
				TargetValue:    &target,
				RecoveryTarget: &recovery,
				MatchType:      ruletypes.AtleastOnce,
				CompareOp:      ruletypes.ValueIsAbove,
			},
		},
	}

	logger := instrumentationtest.New().Logger()
	rule, err := NewThresholdRule("69", valuer.GenerateUUID(), &postableRule, nil, nil, logger, WithEvalDelay(valuer.MustParseTextDuration("2m")))
	require.NoError(t, err)

	now := time.Now()
	series := timeseriestypes.Series{
		Points: []timeseriestypes.Point{
			{Timestamp: now.UnixMilli(), Value: 3},
			{Timestamp: now.Add(time.Minute).UnixMilli(), Value: 4},
			{Timestamp: now.Add(2 * time.Minute).UnixMilli(), Value: 5},
		},
		Labels: map[string]string{
			"service.name": "frontend",
		},
		LabelsArray: []map[string]string{
			{
				"service.name": "frontend",
			},
		},
	}

	alertLabels := ruletypes.PrepareSampleLabelsForRule(series.Labels, "primary")
	activeAlerts := map[uint64]struct{}{alertLabels.Hash(): {}}

	resultVectors, err := rule.Threshold.Eval(series, rule.Unit(), ruletypes.EvalData{
		ActiveAlerts:  activeAlerts,
		SendUnmatched: true,
	})
	require.NoError(t, err)
	require.Len(t, resultVectors, 1, "expected unmatched sample to be returned")

	smpl := resultVectors[0]
	assert.Equal(t, float64(3), smpl.V)
	assert.False(t, smpl.IsRecovering, "unmatched path should not mark sample as recovering")
	assert.Equal(t, float64(4), *smpl.RecoveryTarget, "unmatched path should set recovery target")
	assert.InDelta(t, target, smpl.Target, 0.01)
	assert.Equal(t, "primary", smpl.Metric.Get(ruletypes.LabelThresholdName))
}

func intPtr(v int) *int {
	return &v
}

// TestThresholdRuleEval_SendUnmatchedVariants tests the different variants of sendUnmatched behavior.
// It tests the case where sendUnmatched is true, false.
func TestThresholdRuleEval_SendUnmatchedVariants(t *testing.T) {
	target := 10.0
	recovery := 5.0
	postableRule := ruletypes.PostableRule{
		AlertName: "Send unmatched variants",
		AlertType: ruletypes.AlertTypeMetric,
		RuleType:  ruletypes.RuleTypeThreshold,
		Evaluation: &ruletypes.EvaluationEnvelope{Kind: ruletypes.RollingEvaluation, Spec: ruletypes.RollingWindow{
			EvalWindow: valuer.MustParseTextDuration("5m"),
			Frequency:  valuer.MustParseTextDuration("1m"),
		}},
		RuleCondition: &ruletypes.RuleCondition{
			CompositeQuery: v5MetricCompositeQuery("probe_success"),
			SelectedQuery:  "A",
		},
	}

	now := time.Now()

	tests := []recoveryTestCase{
		{
			description: "sendUnmatched returns first valid point",
			values: timeseriestypes.Series{
				Points: []timeseriestypes.Point{
					{Timestamp: now.UnixMilli(), Value: 3},
					{Timestamp: now.Add(time.Minute).UnixMilli(), Value: 4},
				},
				Labels: map[string]string{
					"service.name": "frontend",
				},
				LabelsArray: []map[string]string{
					{
						"service.name": "frontend",
					},
				},
			},
			compareOp:      string(ruletypes.ValueIsAbove),
			matchType:      string(ruletypes.AtleastOnce),
			target:         target,
			recoveryTarget: &recovery,
			thresholdName:  "primary",
			// Since sendUnmatched is true, the rule should return the first valid point
			// even if it doesn't match the rule condition with current target value of 10.0
			sendUnmatched:       true,
			expectSamples:       intPtr(1),
			expectedSampleValue: 3,
		},
		{
			description: "sendUnmatched false suppresses unmatched",
			values: timeseriestypes.Series{
				Points: []timeseriestypes.Point{
					{Timestamp: now.UnixMilli(), Value: 3},
					{Timestamp: now.Add(time.Minute).UnixMilli(), Value: 4},
				},
				Labels: map[string]string{
					"service.name": "frontend",
				},
				LabelsArray: []map[string]string{
					{
						"service.name": "frontend",
					},
				},
			},
			compareOp:      string(ruletypes.ValueIsAbove),
			matchType:      string(ruletypes.AtleastOnce),
			target:         target,
			recoveryTarget: &recovery,
			thresholdName:  "primary",
			// Since sendUnmatched is false, the rule should not return any samples
			sendUnmatched: false,
			expectSamples: intPtr(0),
		},
		{
			description: "sendUnmatched skips NaN and uses next point",
			values: timeseriestypes.Series{
				Points: []timeseriestypes.Point{
					{Timestamp: now.UnixMilli(), Value: math.NaN()},
					{Timestamp: now.Add(time.Minute).UnixMilli(), Value: math.Inf(1)},
					{Timestamp: now.Add(2 * time.Minute).UnixMilli(), Value: 7},
				},
				Labels: map[string]string{
					"service.name": "frontend",
				},
				LabelsArray: []map[string]string{
					{
						"service.name": "frontend",
					},
				},
			},
			compareOp:      string(ruletypes.ValueIsAbove),
			matchType:      string(ruletypes.AtleastOnce),
			target:         target,
			recoveryTarget: &recovery,
			thresholdName:  "primary",
			// Since sendUnmatched is true, the rule should return the first valid point
			// even if it doesn't match the rule condition with current target value of 10.0
			sendUnmatched:       true,
			expectSamples:       intPtr(1),
			expectedSampleValue: 7,
		},
	}

	for _, tc := range tests {
		runEvalTests(t, postableRule, []recoveryTestCase{tc})
	}
}

// TestThresholdRuleEval_RecoveryNotMetSendUnmatchedFalse tests the case where the recovery target is not met and sendUnmatched is false.
// In this case, the rule should not return any samples as no alert is active plus the recovery target is not met.
func TestThresholdRuleEval_RecoveryNotMetSendUnmatchedFalse(t *testing.T) {
	target := 10.0
	recovery := 5.0

	now := time.Now()
	postableRule := ruletypes.PostableRule{
		AlertName: "Recovery not met sendUnmatched false",
		AlertType: ruletypes.AlertTypeMetric,
		RuleType:  ruletypes.RuleTypeThreshold,
		Evaluation: &ruletypes.EvaluationEnvelope{Kind: ruletypes.RollingEvaluation, Spec: ruletypes.RollingWindow{
			EvalWindow: valuer.MustParseTextDuration("5m"),
			Frequency:  valuer.MustParseTextDuration("1m"),
		}},
		RuleCondition: &ruletypes.RuleCondition{
			CompositeQuery: v5MetricCompositeQuery("probe_success"),
			SelectedQuery:  "A",
		},
	}

	tc := recoveryTestCase{
		description: "recovery target present but not met, sendUnmatched false",
		values: timeseriestypes.Series{
			Points: []timeseriestypes.Point{
				{Timestamp: now.UnixMilli(), Value: 3},
				{Timestamp: now.Add(time.Minute).UnixMilli(), Value: 4},
			},
			Labels: map[string]string{
				"service.name": "frontend",
			},
			LabelsArray: []map[string]string{
				{
					"service.name": "frontend",
				},
			},
		},
		compareOp:              string(ruletypes.ValueIsAbove),
		matchType:              string(ruletypes.AtleastOnce),
		target:                 target,
		recoveryTarget:         &recovery,
		thresholdName:          "primary",
		sendUnmatched:          false,
		expectSamples:          intPtr(0),
		activeAlerts:           nil, // will auto-calc
		expectedTarget:         target,
		expectedRecoveryTarget: recovery,
	}

	runEvalTests(t, postableRule, []recoveryTestCase{tc})
}

func runEvalTests(t *testing.T, postableRule ruletypes.PostableRule, testCases []recoveryTestCase) {
	logger := instrumentationtest.New().Logger()
	for _, c := range testCases {
		t.Run(c.description, func(t *testing.T) {
			// Prepare threshold with recovery target
			threshold := ruletypes.BasicRuleThreshold{
				Name:           c.thresholdName,
				TargetValue:    &c.target,
				RecoveryTarget: c.recoveryTarget,
				MatchType:      ruletypes.MatchType(c.matchType),
				CompareOp:      ruletypes.CompareOp(c.compareOp),
			}

			// Build thresholds list
			thresholds := ruletypes.BasicRuleThresholds{threshold}

			// Add additional thresholds if specified
			for _, addThreshold := range c.additionalThresholds {
				thresholds = append(thresholds, ruletypes.BasicRuleThreshold{
					Name:           addThreshold.name,
					TargetValue:    &addThreshold.target,
					RecoveryTarget: addThreshold.recoveryTarget,
					MatchType:      ruletypes.MatchType(addThreshold.matchType),
					CompareOp:      ruletypes.CompareOp(addThreshold.compareOp),
				})
			}

			postableRule.RuleCondition.Thresholds = &ruletypes.RuleThresholdData{
				Kind: ruletypes.BasicThresholdKind,
				Spec: thresholds,
			}

			rule, err := NewThresholdRule("69", valuer.GenerateUUID(), &postableRule, nil, nil, logger, WithEvalDelay(valuer.MustParseTextDuration("2m")))
			if err != nil {
				assert.NoError(t, err)
				return
			}

			values := c.values
			for i := range values.Points {
				values.Points[i].Timestamp = time.Now().UnixMilli()
			}

			// Prepare activeAlerts: if nil, auto-calculate from labels + thresholdName
			activeAlerts := c.activeAlerts
			if activeAlerts == nil {
				sampleLabels := ruletypes.PrepareSampleLabelsForRule(values.Labels, c.thresholdName)
				alertHash := sampleLabels.Hash()
				activeAlerts = map[uint64]struct{}{alertHash: {}}
				// Handle other thresholds
				for _, addThreshold := range c.additionalThresholds {
					sampleLabels := ruletypes.PrepareSampleLabelsForRule(values.Labels, addThreshold.name)
					alertHash := sampleLabels.Hash()
					activeAlerts[alertHash] = struct{}{}
				}
			}

			evalData := ruletypes.EvalData{
				ActiveAlerts:  activeAlerts,
				SendUnmatched: c.sendUnmatched,
			}

			resultVectors, err := rule.Threshold.Eval(values, rule.Unit(), evalData)
			assert.NoError(t, err)

			if c.expectSamples != nil {
				assert.Equal(t, *c.expectSamples, len(resultVectors), "sample count mismatch")
				if *c.expectSamples > 0 {
					assert.InDelta(t, c.expectedSampleValue, resultVectors[0].V, 0.01, "sample value mismatch")
				}
				return
			}

			// Verify results
			if c.expectAlert || c.expectRecovery {
				// Either a new alert fires or recovery happens - both return result vectors
				assert.NotEmpty(t, resultVectors, "Expected alert or recovery but got no result vectors")
				if len(resultVectors) > 0 {
					found := false
					for _, sample := range resultVectors {
						// Check if this is the expected sample
						if sample.V == c.expectedAlertSample.Value {
							found = true
							// Verify IsRecovering flag
							assert.Equal(t, c.expectRecovery, sample.IsRecovering, "IsRecovering flag mismatch")
							// Verify target value
							if c.expectedTarget != 0 || sample.Target != 0 {
								assert.InDelta(t, c.expectedTarget, sample.Target, 0.01, "Target value mismatch")
							}
							if sample.RecoveryTarget != nil {
								assert.InDelta(t, *sample.RecoveryTarget, c.expectedRecoveryTarget, 0.01, "Recovery target value mismatch")
							}
							break
						}
					}
					assert.True(t, found, "Expected alert sample value %.2f not found in result vectors. Got values: %v", c.expectedAlertSample.Value, getVectorValues(resultVectors))
				}
			} else {
				// No alert and no recovery expected - should be empty
				assert.Empty(t, resultVectors, "Expected no alert but got result vectors: %v", resultVectors)
			}
		})
	}
}

// runMultiThresholdEvalTests runs tests for multiple threshold scenarios
// where each threshold can be in a different state (firing, recovering, resolved)
func runMultiThresholdEvalTests(t *testing.T, postableRule ruletypes.PostableRule, testCases []multiThresholdTestCase) {
	logger := instrumentationtest.New().Logger()
	for _, c := range testCases {
		t.Run(c.description, func(t *testing.T) {
			// Prepare primary threshold
			threshold := ruletypes.BasicRuleThreshold{
				Name:           c.thresholdName,
				TargetValue:    &c.target,
				RecoveryTarget: c.recoveryTarget,
				MatchType:      ruletypes.MatchType(c.matchType),
				CompareOp:      ruletypes.CompareOp(c.compareOp),
			}

			// Build thresholds list
			thresholds := ruletypes.BasicRuleThresholds{threshold}

			// Add additional thresholds
			for _, addThreshold := range c.additionalThresholds {
				thresholds = append(thresholds, ruletypes.BasicRuleThreshold{
					Name:           addThreshold.name,
					TargetValue:    &addThreshold.target,
					RecoveryTarget: addThreshold.recoveryTarget,
					MatchType:      ruletypes.MatchType(addThreshold.matchType),
					CompareOp:      ruletypes.CompareOp(addThreshold.compareOp),
				})
			}

			postableRule.RuleCondition.Thresholds = &ruletypes.RuleThresholdData{
				Kind: ruletypes.BasicThresholdKind,
				Spec: thresholds,
			}

			rule, err := NewThresholdRule("69", valuer.GenerateUUID(), &postableRule, nil, nil, logger, WithEvalDelay(valuer.MustParseTextDuration("2m")))
			if err != nil {
				assert.NoError(t, err)
				return
			}

			values := c.values
			for i := range values.Points {
				values.Points[i].Timestamp = time.Now().UnixMilli()
			}

			// Prepare activeAlerts: if nil, auto-calculate from labels + all threshold names
			activeAlerts := c.activeAlerts
			if activeAlerts == nil {
				activeAlerts = make(map[uint64]struct{})
				// Add primary threshold
				sampleLabels := ruletypes.PrepareSampleLabelsForRule(values.Labels, c.thresholdName)
				alertHash := sampleLabels.Hash()
				activeAlerts[alertHash] = struct{}{}
				// Add additional thresholds
				for _, addThreshold := range c.additionalThresholds {
					sampleLabels := ruletypes.PrepareSampleLabelsForRule(values.Labels, addThreshold.name)
					alertHash := sampleLabels.Hash()
					activeAlerts[alertHash] = struct{}{}
				}
			}

			evalData := ruletypes.EvalData{
				ActiveAlerts: activeAlerts,
			}

			resultVectors, err := rule.Threshold.Eval(values, rule.Unit(), evalData)
			assert.NoError(t, err)

			// Validate total sample count
			assert.Equal(t, c.ExpectedSampleCount, len(resultVectors),
				"Expected %d samples but got %d. Sample values: %v",
				c.ExpectedSampleCount, len(resultVectors), getVectorValues(resultVectors))

			// Build a map of threshold name -> sample for easy lookup
			samplesByThreshold := make(map[string]ruletypes.Sample)
			for _, sample := range resultVectors {
				thresholdName := sample.Metric.Get(ruletypes.LabelThresholdName)
				samplesByThreshold[thresholdName] = sample
			}

			// Validate each threshold's expected result
			for thresholdName, expectation := range c.ExpectedResults {
				sample, found := samplesByThreshold[thresholdName]

				if expectation.ShouldReturnSample {
					assert.True(t, found, "Expected sample for threshold '%s' but not found in results", thresholdName)
					if !found {
						continue
					}

					// Validate IsRecovering flag
					assert.Equal(t, expectation.IsRecovering, sample.IsRecovering,
						"Threshold '%s': IsRecovering flag mismatch", thresholdName)

					// Validate sample value
					assert.InDelta(t, expectation.SampleValue, sample.V, 0.01,
						"Threshold '%s': Sample value mismatch", thresholdName)

					// Validate target value
					assert.InDelta(t, expectation.TargetValue, sample.Target, 0.01,
						"Threshold '%s': Target value mismatch", thresholdName)

					// Validate recovery target value
					if expectation.RecoveryValue != nil {
						assert.NotNil(t, sample.RecoveryTarget,
							"Threshold '%s': Expected RecoveryTarget to be set but it was nil", thresholdName)
						if sample.RecoveryTarget != nil {
							assert.InDelta(t, *expectation.RecoveryValue, *sample.RecoveryTarget, 0.01,
								"Threshold '%s': RecoveryTarget value mismatch", thresholdName)
						}
					}
				} else {
					assert.False(t, found, "Expected NO sample for threshold '%s' but found one with value %.2f",
						thresholdName, sample.V)
				}
			}

			// Validate sample order if specified
			if len(c.ExpectedSampleOrder) > 0 {
				assert.Equal(t, len(c.ExpectedSampleOrder), len(resultVectors),
					"Expected sample order length mismatch")
				for i, expectedName := range c.ExpectedSampleOrder {
					if i < len(resultVectors) {
						actualName := resultVectors[i].Metric.Get(ruletypes.LabelThresholdName)
						assert.Equal(t, expectedName, actualName,
							"Sample order mismatch at index %d: expected '%s', got '%s'",
							i, expectedName, actualName)
					}
				}
			}
		})
	}
}

// TestThresholdRuleEval_MultiThreshold tests multiple threshold scenarios
// where each threshold can be in a different state (firing, recovering, resolved)
func TestThresholdRuleEval_MultiThreshold(t *testing.T) {
	postableRule := ruletypes.PostableRule{
		AlertName: "Multi-Threshold Recovery Test",
		AlertType: ruletypes.AlertTypeMetric,
		RuleType:  ruletypes.RuleTypeThreshold,
		Evaluation: &ruletypes.EvaluationEnvelope{Kind: ruletypes.RollingEvaluation, Spec: ruletypes.RollingWindow{
			EvalWindow: valuer.MustParseTextDuration("5m"),
			Frequency:  valuer.MustParseTextDuration("1m"),
		}},
		RuleCondition: &ruletypes.RuleCondition{
			CompositeQuery: v5MetricCompositeQuery("probe_success"),
			SelectedQuery:  "A",
		},
	}

	runMultiThresholdEvalTests(t, postableRule, tcThresholdRuleEvalMultiThreshold)
}

func TestThresholdEval_RequireMinPoints(t *testing.T) {
	postableRule := ruletypes.PostableRule{
		AlertName: "Unit test",
		AlertType: ruletypes.AlertTypeMetric,
		RuleType:  ruletypes.RuleTypeThreshold,
		Evaluation: &ruletypes.EvaluationEnvelope{Kind: ruletypes.RollingEvaluation, Spec: ruletypes.RollingWindow{
			EvalWindow: valuer.MustParseTextDuration("5m"),
			Frequency:  valuer.MustParseTextDuration("1m"),
		}},
		RuleCondition: &ruletypes.RuleCondition{
			CompareOp:      ruletypes.ValueIsAbove,
			MatchType:      ruletypes.AtleastOnce,
			CompositeQuery: v5MetricCompositeQuery("signoz_calls_total"),
			SelectedQuery:  "A",
		},
		Version: "v5",
	}

	cases := []struct {
		description       string
		requireMinPoints  bool
		requiredNumPoints int
		values            [][]any
		target            float64
		expectAlerts      int
	}{
		{
			description:      "AlertCondition=false, RequireMinPoints=false",
			requireMinPoints: false,
			values: [][]any{
				{100.0, "attr", time.Now()},
				{150.0, "attr", time.Now().Add(-1 * time.Minute)},
			},
			target:       200,
			expectAlerts: 0,
		},
		{
			description:      "AlertCondition=true, RequireMinPoints=false",
			requireMinPoints: false,
			values: [][]any{
				{100.0, "attr", time.Now()},
				{150.0, "attr", time.Now().Add(-1 * time.Minute)},
				{250.0, "attr", time.Now().Add(-2 * time.Minute)},
			},
			target:       200,
			expectAlerts: 1,
		},
		{
			description:       "AlertCondition=true, RequireMinPoints=true, NumPoints=more_than_required",
			requireMinPoints:  true,
			requiredNumPoints: 2,
			values: [][]any{
				{100.0, "attr", time.Now()},
				{150.0, "attr", time.Now().Add(-1 * time.Minute)},
				{250.0, "attr", time.Now().Add(-2 * time.Minute)},
			},
			target:       200,
			expectAlerts: 1,
		},
		{
			description:       "AlertCondition=true, RequireMinPoints=true, NumPoints=same_as_required",
			requireMinPoints:  true,
			requiredNumPoints: 3,
			values: [][]any{
				{100.0, "attr", time.Now()},
				{150.0, "attr", time.Now().Add(-1 * time.Minute)},
				{250.0, "attr", time.Now().Add(-2 * time.Minute)},
			},
			target:       200,
			expectAlerts: 1,
		},
		{
			description:       "AlertCondition=true, RequireMinPoints=true, NumPoints=insufficient",
			requireMinPoints:  true,
			requiredNumPoints: 4,
			values: [][]any{
				{100.0, "attr", time.Now()},
				{150.0, "attr", time.Now().Add(-1 * time.Minute)},
				{250.0, "attr", time.Now().Add(-2 * time.Minute)},
			},
			target:       200,
			expectAlerts: 0,
		},
		{
			description:       "AlertCondition=true, RequireMinPoints=true, NumPoints=zero",
			requireMinPoints:  true,
			requiredNumPoints: 4,
			values:            [][]any{},
			target:            200,
			expectAlerts:      0,
		},
	}

	for idx, c := range cases {
		logger := instrumentationtest.New().Logger()
		rc := postableRule.RuleCondition
		rc.Target = &c.target
		rc.RequireMinPoints = c.requireMinPoints
		rc.RequiredNumPoints = c.requiredNumPoints
		rc.Thresholds = &ruletypes.RuleThresholdData{
			Kind: ruletypes.BasicThresholdKind,
			Spec: ruletypes.BasicRuleThresholds{
				{
					Name:        postableRule.AlertName,
					TargetValue: &c.target,
					MatchType:   rc.MatchType,
					CompareOp:   rc.CompareOp,
				},
			},
		}

		querier := staticV5Querier{
			response: v5QueryResponse(v5SeriesFromRows(c.values, false)...),
		}
		rule, err := NewThresholdRule("some-id", valuer.GenerateUUID(), &postableRule, newRuleStateHistoryTestReader(), querier, logger)
		t.Run(fmt.Sprintf("%d %s", idx, c.description), func(t *testing.T) {
			require.NoError(t, err)

			alertsFound, err := rule.Eval(context.Background(), time.Now())
			require.NoError(t, err)

			assert.Equal(t, c.expectAlerts, alertsFound, "case %d", idx)
		})
	}
}
