package rules

import (
	"context"
	"database/sql"
	"encoding/json"
	"strconv"
	"testing"
	"time"

	"github.com/SigNoz/signoz/pkg/alertmanager"
	alertmanagermock "github.com/SigNoz/signoz/pkg/alertmanager/alertmanagertest"
	"github.com/SigNoz/signoz/pkg/query-service/model"
	"github.com/SigNoz/signoz/pkg/query-service/model/querytypes"
	"github.com/SigNoz/signoz/pkg/sqlstore"
	"github.com/SigNoz/signoz/pkg/sqlstore/sqlstoretest"
	"github.com/SigNoz/signoz/pkg/types/alertmanagertypes"
	"github.com/SigNoz/signoz/pkg/types/metrictypes"
	qbtypes "github.com/SigNoz/signoz/pkg/types/querybuildertypes/querybuildertypesv5"
	"github.com/SigNoz/signoz/pkg/types/ruletypes"
	"github.com/SigNoz/signoz/pkg/types/telemetrytypes"
	"github.com/SigNoz/signoz/pkg/valuer"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

type queryRunnerFunc func(context.Context, valuer.UUID, *qbtypes.QueryRangeRequest) (*qbtypes.QueryRangeResponse, error)

func (f queryRunnerFunc) Execute(ctx context.Context, orgID valuer.UUID, request *qbtypes.QueryRangeRequest) (*qbtypes.QueryRangeResponse, error) {
	return f(ctx, orgID, request)
}

func responseFromTestValues(values [][]interface{}) *qbtypes.QueryRangeResponse {
	points := make([]*qbtypes.TimeSeriesValue, 0, len(values))
	for _, row := range values {
		value, _ := row[0].(float64)
		timestamp, _ := row[2].(time.Time)
		points = append(points, &qbtypes.TimeSeriesValue{Timestamp: timestamp.UnixMilli(), Value: value})
	}
	return &qbtypes.QueryRangeResponse{
		Type: qbtypes.RequestTypeTimeSeries,
		Data: qbtypes.QueryData{Results: []any{&qbtypes.TimeSeriesData{
			QueryName: "A",
			Aggregations: []*qbtypes.AggregationBucket{{
				Series: []*qbtypes.TimeSeries{{Values: points}},
			}},
		}}},
	}
}

func TestManager_TestNotification_SendUnmatched_ThresholdRule(t *testing.T) {
	target := 10.0
	recovery := 5.0

	for _, tc := range TcTestNotiSendUnmatchedThresholdRule {
		t.Run(tc.Name, func(t *testing.T) {
			rule := ThresholdRuleAtLeastOnceValueAbove(target, &recovery)
			rule.NotificationSettings.MessageTemplate = "{{alert.name}}|{{severity}}|{{value}}|{{threshold}}|{{label.service.name}}"

			// Marshal rule to JSON as TestNotification expects
			ruleBytes, err := json.Marshal(rule)
			require.NoError(t, err)

			orgID := valuer.GenerateUUID()

			// for saving temp alerts that are triggered via TestNotification
			triggeredTestAlerts := []map[*alertmanagertypes.PostableAlert][]string{}

			// Create manager using test factory with hooks
			mgr := NewTestManager(t, &TestManagerOptions{
				QueryRunner: queryRunnerFunc(func(_ context.Context, _ valuer.UUID, _ *qbtypes.QueryRangeRequest) (*qbtypes.QueryRangeResponse, error) {
					return responseFromTestValues(tc.Values), nil
				}),
				AlertmanagerHook: func(am alertmanager.Alertmanager) {
					mockAM := am.(*alertmanagermock.MockAlertmanager)
					// mock set notification config
					mockAM.On("SetNotificationConfig", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(nil)
					// for saving temp alerts that are triggered via TestNotification
					if tc.ExpectAlerts > 0 {
						mockAM.On("TestAlert", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Run(func(args mock.Arguments) {
							triggeredTestAlerts = append(triggeredTestAlerts, args.Get(3).(map[*alertmanagertypes.PostableAlert][]string))
						}).Return(nil).Times(tc.ExpectAlerts)
					}
				},
				SqlStoreHook: func(store sqlstore.SQLStore) {
					mockStore := store.(*sqlstoretest.Provider)
					// Mock the organizations query that SendAlerts makes
					// Bun generates: SELECT id FROM organizations LIMIT 1 (or SELECT "id" FROM "organizations" LIMIT 1)
					orgRows := mockStore.Mock().NewRows([]string{"id"}).AddRow(orgID.StringValue())
					// Match bun's generated query pattern - bun may quote identifiers
					mockStore.Mock().ExpectQuery("SELECT (.+) FROM (.+)organizations(.+) LIMIT (.+)").WillReturnRows(orgRows)
				},
			})

			preview, apiErr := mgr.TestNotification(context.Background(), orgID, string(ruleBytes))
			if apiErr != nil {
				t.Logf("TestNotification error: %v, type: %s", apiErr.Err, apiErr.Typ)
			}
			require.Nil(t, apiErr)
			assert.Equal(t, tc.ExpectAlerts, preview.AlertCount)
			assert.NotZero(t, preview.EvaluatedAt)

			if tc.ExpectAlerts > 0 {
				// check if the alert has been triggered
				require.Len(t, triggeredTestAlerts, 1)
				var gotAlerts []*alertmanagertypes.PostableAlert
				for a := range triggeredTestAlerts[0] {
					gotAlerts = append(gotAlerts, a)
				}
				require.Len(t, gotAlerts, tc.ExpectAlerts)
				// check if the alert has triggered with correct threshold value
				if tc.ExpectValue != 0 {
					assert.Equal(t, strconv.FormatFloat(tc.ExpectValue, 'f', -1, 64), gotAlerts[0].Annotations["value"])
					assert.Equal(t, "test-alert_TEST_ALERT|primary|"+strconv.FormatFloat(tc.ExpectValue, 'f', -1, 64)+"|10|frontend", gotAlerts[0].Annotations["description"])
					assert.Equal(t, gotAlerts[0].Annotations["description"], gotAlerts[0].Annotations["summary"])
					assert.Equal(t, gotAlerts[0].Annotations["description"], gotAlerts[0].Annotations["message"])
				}
			} else {
				// check if no alerts have been triggered
				assert.Empty(t, triggeredTestAlerts)
			}
		})
	}
}

func TestManager_TestNotificationV3NoDataPreviewAndNotification(t *testing.T) {
	target := 10.0
	rule := ruletypes.PostableRule{
		AlertName:     "no data preview",
		AlertType:     ruletypes.AlertTypeMetric,
		RuleType:      ruletypes.RuleTypeThreshold,
		SchemaVersion: ruletypes.CurrentSchemaVersion,
		Version:       "v5",
		Evaluation: &ruletypes.EvaluationEnvelope{
			Kind: ruletypes.RollingEvaluation,
			Spec: ruletypes.RollingWindow{
				EvalWindow: valuer.MustParseTextDuration("5m"),
				Frequency:  valuer.MustParseTextDuration("1m"),
			},
		},
		RuleCondition: &ruletypes.RuleCondition{
			Kind:          ruletypes.ConditionKindNumeric,
			SelectedQuery: "A",
			DataQuality: ruletypes.DataQualityPolicy{
				AlertOnNoData: true,
				NoDataFor:     valuer.MustParseTextDuration("30s"),
				MinPoints:     1,
			},
			CompositeQuery: &ruletypes.CompositeQuery{
				QueryType: querytypes.QueryTypeBuilder,
				Queries: []qbtypes.QueryEnvelope{{
					Type: qbtypes.QueryTypeBuilder,
					Spec: qbtypes.QueryBuilderQuery[qbtypes.MetricAggregation]{
						Name:         "A",
						Signal:       telemetrytypes.SignalMetrics,
						StepInterval: qbtypes.Step{Duration: time.Minute},
						Aggregations: []qbtypes.MetricAggregation{{
							MetricName:       "no_data_metric",
							TimeAggregation:  metrictypes.TimeAggregationSum,
							SpaceAggregation: metrictypes.SpaceAggregationSum,
						}},
					},
				}},
			},
			Numeric: &ruletypes.NumericThresholdCondition{
				Reduction: ruletypes.ReductionAtLeastOnce,
				Operator:  ruletypes.NumericOperatorGreaterThan,
				Thresholds: []ruletypes.NumericThreshold{{
					Severity: "critical",
					Target:   &target,
					Channels: []string{"local-webhook"},
				}, {
					Severity: "warning",
					Target:   &target,
					Channels: []string{"local-webhook", "secondary-webhook"},
				}},
			},
		},
		NotificationSettings: &ruletypes.NotificationSettings{},
	}

	// TestNotification receives JSON at the API boundary. A round trip here
	// ensures the v3 decoder populates the evaluator-only runtime fields too.
	ruleBytes, err := json.Marshal(rule)
	require.NoError(t, err)

	orgID := valuer.GenerateUUID()
	var delivered map[*alertmanagertypes.PostableAlert][]string
	mgr := NewTestManager(t, &TestManagerOptions{
		QueryRunner: queryRunnerFunc(func(_ context.Context, _ valuer.UUID, _ *qbtypes.QueryRangeRequest) (*qbtypes.QueryRangeResponse, error) {
			return responseFromTestValues(nil), nil
		}),
		AlertmanagerHook: func(am alertmanager.Alertmanager) {
			mockAM := am.(*alertmanagermock.MockAlertmanager)
			mockAM.On("SetNotificationConfig", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(nil)
			mockAM.On("TestAlert", mock.Anything, mock.Anything, mock.Anything, mock.Anything).
				Run(func(args mock.Arguments) {
					delivered = args.Get(3).(map[*alertmanagertypes.PostableAlert][]string)
				}).
				Return(nil).
				Once()
		},
		SqlStoreHook: func(store sqlstore.SQLStore) {
			mockStore := store.(*sqlstoretest.Provider)
			orgRows := mockStore.Mock().NewRows([]string{"id"}).AddRow(orgID.StringValue())
			mockStore.Mock().ExpectQuery("SELECT (.+) FROM (.+)organizations(.+) LIMIT (.+)").WillReturnRows(orgRows)
		},
	})

	preview, apiErr := mgr.TestNotification(context.Background(), orgID, string(ruleBytes))
	require.Nil(t, apiErr)
	require.Equal(t, 1, preview.AlertCount)
	require.Equal(t, model.StateNoData, preview.State)
	require.NotZero(t, preview.EvaluatedAt)
	require.Len(t, delivered, 1)

	for alert, receivers := range delivered {
		require.Equal(t, []string{"local-webhook", "secondary-webhook"}, receivers)
		require.Equal(t, "true", alert.Alert.Labels["nodata"])
		require.Equal(t, "true", alert.Alert.Labels["testalert"])
	}
}

func TestManager_TestNotificationReturnsDeliveryFailure(t *testing.T) {
	target := 10.0
	ruleBytes, err := json.Marshal(ThresholdRuleAtLeastOnceValueAbove(target, nil))
	require.NoError(t, err)

	orgID := valuer.GenerateUUID()
	mgr := NewTestManager(t, &TestManagerOptions{
		QueryRunner: queryRunnerFunc(func(_ context.Context, _ valuer.UUID, _ *qbtypes.QueryRangeRequest) (*qbtypes.QueryRangeResponse, error) {
			return responseFromTestValues([][]interface{}{{float64(20), "", time.Now()}}), nil
		}),
		AlertmanagerHook: func(am alertmanager.Alertmanager) {
			mockAM := am.(*alertmanagermock.MockAlertmanager)
			mockAM.On("SetNotificationConfig", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(nil)
			mockAM.On("TestAlert", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(sql.ErrConnDone).Once()
		},
		SqlStoreHook: func(store sqlstore.SQLStore) {
			mockStore := store.(*sqlstoretest.Provider)
			orgRows := mockStore.Mock().NewRows([]string{"id"}).AddRow(orgID.StringValue())
			mockStore.Mock().ExpectQuery("SELECT (.+) FROM (.+)organizations(.+) LIMIT (.+)").WillReturnRows(orgRows)
		},
	})

	_, apiErr := mgr.TestNotification(context.Background(), orgID, string(ruleBytes))
	require.NotNil(t, apiErr)
	require.ErrorContains(t, apiErr.Err, "test notification delivery failed")
	require.ErrorIs(t, apiErr.Err, sql.ErrConnDone)
}
