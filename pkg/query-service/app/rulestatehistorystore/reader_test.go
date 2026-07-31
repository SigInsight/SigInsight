package rulestatehistorystore

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"strings"
	"testing"
	"time"

	"github.com/ClickHouse/clickhouse-go/v2"
	"github.com/ClickHouse/clickhouse-go/v2/lib/driver"
	"github.com/stretchr/testify/require"

	chErrors "github.com/SigNoz/signoz/pkg/query-service/errors"
	"github.com/SigNoz/signoz/pkg/query-service/model"
)

type ruleHistoryQueryConn struct {
	clickhouse.Conn
	query func(context.Context, string, ...any) (driver.Rows, error)
}

func (c ruleHistoryQueryConn) Query(ctx context.Context, query string, args ...any) (driver.Rows, error) {
	return c.query(ctx, query, args...)
}

type ruleHistorySelectConn struct {
	clickhouse.Conn
	selectRows func(context.Context, any, string, ...any) error
}

func (c ruleHistorySelectConn) Select(ctx context.Context, dest any, query string, args ...any) error {
	return c.selectRows(ctx, dest, query, args...)
}

func TestReadRowMapsTimestampValueAndLabels(t *testing.T) {
	timestamp := time.Unix(100, 0)
	value := 42.5
	service := "api"

	groupBy, labels, labelsArray, point := readRow(
		[]interface{}{&timestamp, &value, &service},
		[]string{"ts", "value", "service.name"},
		1,
	)

	require.Equal(t, []string{"api"}, groupBy)
	require.Equal(t, map[string]string{"service.name": "api"}, labels)
	require.Equal(t, []map[string]string{{"service.name": "api"}}, labelsArray)
	require.NotNil(t, point)
	require.Equal(t, timestamp.UnixMilli(), point.Timestamp)
	require.Equal(t, value, point.Value)
}

func TestReadRowTreatsAdditionalNumberAsLabel(t *testing.T) {
	value := 42.5
	shard := uint64(7)

	groupBy, labels, _, point := readRow(
		[]interface{}{&value, &shard},
		[]string{"value", "shard"},
		2,
	)

	require.Equal(t, []string{"7"}, groupBy)
	require.Equal(t, map[string]string{"shard": "7"}, labels)
	require.NotNil(t, point)
	require.Equal(t, value, point.Value)
}

func TestPersonalisedErrorMapsClickHouseLimits(t *testing.T) {
	require.NoError(t, getPersonalisedError(nil))
	require.ErrorIs(t, getPersonalisedError(errors.New("clickhouse code: 307")), chErrors.ErrResourceBytesLimitExceeded)
	require.ErrorIs(t, getPersonalisedError(errors.New("clickhouse code: 159")), chErrors.ErrResourceTimeLimitExceeded)
}

func TestPersonalisedErrorPreservesUnknownError(t *testing.T) {
	expected := errors.New("query failed")
	require.Same(t, expected, getPersonalisedError(expected))
}

func TestBuildRuleStateHistoryConditionsBindsUserInput(t *testing.T) {
	ruleID := "rule' OR 1=1 --"
	state := "fir'ing"
	conditions, args, err := buildRuleStateHistoryConditions(ruleID, &model.QueryRuleStateHistory{
		Start: 100,
		End:   200,
		State: state,
	})

	require.NoError(t, err)
	require.Equal(t, "rule_id = ? AND unix_milli >= ? AND unix_milli < ? AND state = ?", conditions)
	require.Equal(t, []any{ruleID, int64(100), int64(200), state}, args)
	for _, value := range []string{ruleID, state} {
		require.False(t, strings.Contains(conditions, value))
	}
}

func TestRuleStateHistoryOrderOnlyAllowsKnownDirections(t *testing.T) {
	order, err := ruleStateHistoryOrder(" DESC ")
	require.NoError(t, err)
	require.Equal(t, "DESC", order)

	_, err = ruleStateHistoryOrder("desc; DROP TABLE rule_state_history_v0")
	require.EqualError(t, err, "order must be asc or desc")
}

func TestGetLastSavedRuleStateHistorySelectsOnlyModelColumns(t *testing.T) {
	var capturedQuery string
	reader := New(slog.New(slog.NewTextHandler(io.Discard, nil)), ruleHistorySelectConn{
		selectRows: func(_ context.Context, _ any, query string, _ ...any) error {
			capturedQuery = query
			return nil
		},
	}, DefaultConfig())

	_, err := reader.GetLastSavedRuleStateHistory(context.Background(), "rule-id")

	require.NoError(t, err)
	require.Contains(t, capturedQuery, "SELECT "+ruleStateHistorySelectColumns+" FROM")
	require.NotContains(t, capturedQuery, "SELECT *")
	require.NotContains(t, capturedQuery, "_retention_days")
}

func TestGetTriggersByIntervalPreservesQueryError(t *testing.T) {
	expected := errors.New("clickhouse unavailable")
	ruleID := "rule' OR 1=1 --"
	var capturedQuery string
	var capturedArgs []any
	reader := New(slog.New(slog.NewTextHandler(io.Discard, nil)), ruleHistoryQueryConn{
		query: func(_ context.Context, query string, args ...any) (driver.Rows, error) {
			capturedQuery = query
			capturedArgs = args
			return nil, expected
		},
	}, DefaultConfig())

	_, err := reader.GetTriggersByInterval(context.Background(), ruleID, &model.QueryRuleStateHistory{Start: 1, End: 2})

	require.ErrorIs(t, err, expected)
	require.ErrorContains(t, err, "query rule state history time series")
	require.NotContains(t, capturedQuery, ruleID)
	require.Contains(t, capturedQuery, "rule_id = ?")
	require.Equal(t, []any{ruleID, model.StateFiring.String(), int64(1), int64(2)}, capturedArgs)
}
