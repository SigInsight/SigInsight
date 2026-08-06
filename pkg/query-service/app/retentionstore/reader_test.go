package retentionstore

import (
	"context"
	"database/sql"
	"errors"
	"io"
	"log/slog"
	"testing"
	"time"

	cmock "github.com/srikanthccv/ClickHouse-go-mock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/uptrace/bun"
	"github.com/uptrace/bun/dialect/sqlitedialect"
	_ "modernc.org/sqlite"

	errorsV2 "github.com/SigNoz/signoz/pkg/errors"
	"github.com/SigNoz/signoz/pkg/query-service/constants"
	"github.com/SigNoz/signoz/pkg/query-service/model"
	"github.com/SigNoz/signoz/pkg/sqlstore"
	"github.com/SigNoz/signoz/pkg/types"
	"github.com/SigNoz/signoz/pkg/valuer"
)

type retentionTestSQLStore struct {
	sqlstore.SQLStore
	db *bun.DB
}

func (s *retentionTestSQLStore) BunDB() *bun.DB { return s.db }

func newRetentionTestReader(t *testing.T) (*Reader, *bun.DB) {
	t.Helper()
	database, err := sql.Open("sqlite", ":memory:")
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, database.Close()) })

	db := bun.NewDB(database, sqlitedialect.New())
	t.Cleanup(func() { require.NoError(t, db.Close()) })
	_, err = db.NewCreateTable().Model((*types.TTLSetting)(nil)).Exec(context.Background())
	require.NoError(t, err)

	reader := New(slog.New(slog.NewTextHandler(io.Discard, nil)), &retentionTestSQLStore{db: db}, nil, Config{})
	return reader, db
}

func insertTTLStatus(t *testing.T, db *bun.DB, orgID, tableName, status string, updatedAt time.Time) {
	t.Helper()
	setting := &types.TTLSetting{
		Identifiable:   types.Identifiable{ID: valuer.GenerateUUID()},
		TransactionID:  valuer.GenerateUUID().StringValue(),
		TableName:      tableName,
		TTL:            7200,
		ColdStorageTTL: -1,
		Status:         status,
		OrgID:          orgID,
	}
	setting.CreatedAt = updatedAt
	setting.UpdatedAt = updatedAt
	_, err := db.NewInsert().Model(setting).Exec(context.Background())
	require.NoError(t, err)
}

func TestGetLocalTableNameIsCanonicalIdentity(t *testing.T) {
	assert := assert.New(t)

	assert.Equal("siginsight_traces.spans", getLocalTableName("siginsight_traces.spans"))
	assert.Equal("spans", getLocalTableName("spans"))
	assert.Equal(
		[]string{"siginsight_logs.logs", "siginsight_traces.spans"},
		getLocalTableNameArray([]string{"siginsight_logs.logs", "siginsight_traces.spans"}),
	)
}

func TestBuildMultiIfExpression(t *testing.T) {
	reader := &Reader{logger: slog.New(slog.NewTextHandler(io.Discard, nil))}
	rules := []model.CustomRetentionRule{
		{
			TTLDays: 3,
			Filters: []model.FilterCondition{
				{Key: "service.name", Values: []string{"api", "worker"}},
				{Key: "deployment.environment", Values: []string{"prod"}},
			},
		},
	}

	require.Equal(
		t,
		"multiIf(resources_string['service.name'] IN ('api', 'worker') AND resources_string['deployment.environment'] IN ('prod'), 3, 15)",
		reader.buildMultiIfExpression(rules, 15, false),
	)
	require.Equal(
		t,
		"multiIf(JSONExtractString(labels, 'service.name') IN ('api', 'worker') AND JSONExtractString(labels, 'deployment.environment') IN ('prod'), 3, 15)",
		reader.buildMultiIfExpression(rules, 15, true),
	)
}

func TestGetTTLQueryStatus(t *testing.T) {
	ctx := context.Background()
	const orgID = "test-org"

	t.Run("returns empty when no persisted TTL request exists", func(t *testing.T) {
		reader, _ := newRetentionTestReader(t)
		status, apiErr := reader.getTTLQueryStatus(ctx, orgID, []string{"siginsight_traces.spans"})
		require.Nil(t, apiErr)
		require.Empty(t, status)
	})

	t.Run("reports only a recent pending request as pending", func(t *testing.T) {
		reader, db := newRetentionTestReader(t)
		insertTTLStatus(t, db, orgID, "siginsight_traces.spans", constants.StatusPending, time.Now().Add(-30*time.Minute))
		status, apiErr := reader.getTTLQueryStatus(ctx, orgID, []string{"siginsight_traces.spans"})
		require.Nil(t, apiErr)
		require.Equal(t, constants.StatusPending, status)
	})

	t.Run("does not let a stale pending request block future work", func(t *testing.T) {
		reader, db := newRetentionTestReader(t)
		insertTTLStatus(t, db, orgID, "siginsight_traces.spans", constants.StatusPending, time.Now().Add(-time.Hour))
		status, apiErr := reader.getTTLQueryStatus(ctx, orgID, []string{"siginsight_traces.spans"})
		require.Nil(t, apiErr)
		require.Equal(t, constants.StatusSuccess, status)
	})

	t.Run("retains failed state", func(t *testing.T) {
		reader, db := newRetentionTestReader(t)
		insertTTLStatus(t, db, orgID, "siginsight_traces.spans", constants.StatusFailed, time.Now())
		status, apiErr := reader.getTTLQueryStatus(ctx, orgID, []string{"siginsight_traces.spans"})
		require.Nil(t, apiErr)
		require.Equal(t, constants.StatusFailed, status)
	})
}

func TestAsyncTTLContextSurvivesRequestCancellation(t *testing.T) {
	ctx, cancel := context.WithCancel(context.WithValue(context.Background(), "request-id", "ttl-request"))
	cancel()

	operationCtx := asyncTTLContext(ctx)
	require.NoError(t, operationCtx.Err())
	require.Equal(t, "ttl-request", operationCtx.Value("request-id"))
}

func TestGetTTLMissingClickHouseTableReturnsAPIError(t *testing.T) {
	reader, _ := newRetentionTestReader(t)
	mock, err := cmock.NewClickHouseNative(nil)
	require.NoError(t, err)
	mock.ExpectSelect("SELECT engine_full FROM system.tables WHERE name='metric_points' AND database='siginsight_metrics'").WillReturnRows(
		cmock.NewRows([]cmock.ColumnType{{Name: "engine_full", Type: "String"}}, nil),
	)
	reader.db = mock

	response, apiErr := reader.GetTTL(context.Background(), "test-org", &model.GetTTLParams{Type: constants.MetricsTTL})
	require.Nil(t, response)
	require.NotNil(t, apiErr)
	require.ErrorContains(t, apiErr, "metrics table metric_points is missing")
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestGetTTLWithoutPersistedRequestUsesUnknownExpectedValues(t *testing.T) {
	reader, _ := newRetentionTestReader(t)
	mock, err := cmock.NewClickHouseNative(nil)
	require.NoError(t, err)
	mock.ExpectSelect("SELECT engine_full FROM system.tables WHERE name='metric_points' AND database='siginsight_metrics'").WillReturnRows(
		cmock.NewRows([]cmock.ColumnType{{Name: "engine_full", Type: "String"}}, [][]any{{"MergeTree() TTL toDateTime(unix_milli) + toIntervalSecond(7200) DELETE"}}),
	)
	reader.db = mock

	response, apiErr := reader.GetTTL(context.Background(), "test-org", &model.GetTTLParams{Type: constants.MetricsTTL})
	require.Nil(t, apiErr)
	require.Equal(t, -1, response.ExpectedMetricsTime)
	require.Equal(t, -1, response.ExpectedMetricsMoveTime)
	require.Empty(t, response.Status)
	require.Equal(t, 2, response.MetricsTime)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestGetDisksPreservesClickHouseError(t *testing.T) {
	reader, _ := newRetentionTestReader(t)
	mock, err := cmock.NewClickHouseNative(nil)
	require.NoError(t, err)
	expected := errors.New("ClickHouse unavailable")
	mock.ExpectSelect("SELECT name,type FROM system.disks").WillReturnError(expected)
	reader.db = mock

	response, err := reader.GetDisks(context.Background())
	require.Nil(t, response)
	require.ErrorIs(t, err, expected)
	require.ErrorContains(t, err, "get ClickHouse disks")
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestTTLMethodsRejectUnknownType(t *testing.T) {
	reader, _ := newRetentionTestReader(t)

	getResponse, getErr := reader.GetTTL(context.Background(), "test-org", &model.GetTTLParams{Type: "logs"})
	require.Nil(t, getResponse)
	require.True(t, errorsV2.Ast(getErr, errorsV2.TypeInvalidInput))

	setResponse, setErr := reader.SetTTL(context.Background(), "test-org", &model.TTLParams{Type: "logs"})
	require.Nil(t, setResponse)
	require.True(t, errorsV2.Ast(setErr, errorsV2.TypeInvalidInput))
}
