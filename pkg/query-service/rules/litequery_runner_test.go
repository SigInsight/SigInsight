package rules

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/SigNoz/signoz/pkg/telemetrystore"
	"github.com/SigNoz/signoz/pkg/telemetrystore/telemetrystoretest"
	qbtypes "github.com/SigNoz/signoz/pkg/types/querybuildertypes/querybuildertypesv5"
	"github.com/SigNoz/signoz/pkg/types/telemetrytypes"
	"github.com/SigNoz/signoz/pkg/valuer"
	cmock "github.com/srikanthccv/ClickHouse-go-mock"
	"github.com/stretchr/testify/require"
)

func TestLiteQueryRunnerExecutesSupportedLogRule(t *testing.T) {
	store := telemetrystoretest.New(telemetrystore.Config{}, &queryMatcherAny{})
	store.Mock().ExpectQuery(".*").WithArgs(
		int64(60_000_000_000), int64(60_000), int64(1_000_000_000), int64(61_000_000_000), int64(60_000_000_000), int64(60_000),
	).WillReturnRows(cmock.NewRows([]cmock.ColumnType{
		{Name: "timestamp", Type: "UInt64"},
		{Name: "value", Type: "Float64"},
	}, [][]any{{uint64(60_000_000_000), float64(3)}}))

	request := &qbtypes.QueryRangeRequest{
		Start: 1_000, End: 61_000, RequestType: qbtypes.RequestTypeTimeSeries,
		CompositeQuery: qbtypes.CompositeQuery{Queries: []qbtypes.QueryEnvelope{{
			Type: qbtypes.QueryTypeBuilder,
			Spec: qbtypes.QueryBuilderQuery[qbtypes.LogAggregation]{
				Name: "A", Signal: telemetrytypes.SignalLogs, StepInterval: qbtypes.Step{Duration: time.Minute},
				Aggregations: []qbtypes.LogAggregation{{Expression: "count()"}},
			},
		}}},
	}
	response, err := NewLiteQueryRunner(store, nil).Execute(context.Background(), valuer.GenerateUUID(), request)
	require.NoError(t, err)
	data := response.Data.Results[0].(*qbtypes.TimeSeriesData)
	require.Len(t, data.Aggregations, 1)
	require.Equal(t, float64(3), data.Aggregations[0].Series[0].Values[0].Value)
	require.NoError(t, store.Mock().ExpectationsWereMet())
}

func TestLiteQueryRunnerRejectsAdvancedRuleWithoutLegacyFallback(t *testing.T) {
	request := &qbtypes.QueryRangeRequest{
		Start: 1, End: 2, RequestType: qbtypes.RequestTypeScalar,
		CompositeQuery: qbtypes.CompositeQuery{Queries: []qbtypes.QueryEnvelope{{
			Type: qbtypes.QueryTypeClickHouseSQL,
			Spec: qbtypes.ClickHouseQuery{Name: "A", Query: "SELECT 1"},
		}}},
	}
	_, err := NewLiteQueryRunner(nil, nil).Execute(context.Background(), valuer.GenerateUUID(), request)
	require.Error(t, err)
	require.True(t, strings.Contains(err.Error(), "unsupported threshold query"))
}
