package rules

import (
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/SigNoz/signoz/pkg/alertmanager"
	alertmanagermock "github.com/SigNoz/signoz/pkg/alertmanager/alertmanagertest"
	"github.com/SigNoz/signoz/pkg/cache"
	"github.com/SigNoz/signoz/pkg/cache/cachetest"
	"github.com/SigNoz/signoz/pkg/instrumentation/instrumentationtest"
	"github.com/SigNoz/signoz/pkg/query-service/app/rulestatehistorystore"
	"github.com/SigNoz/signoz/pkg/sqlstore"
	"github.com/SigNoz/signoz/pkg/sqlstore/sqlstoretest"
	"github.com/SigNoz/signoz/pkg/telemetrylogs"
	"github.com/SigNoz/signoz/pkg/telemetrymetadata"
	"github.com/SigNoz/signoz/pkg/telemetrymeter"
	"github.com/SigNoz/signoz/pkg/telemetrymetrics"
	"github.com/SigNoz/signoz/pkg/telemetrystore"
	"github.com/SigNoz/signoz/pkg/telemetrystore/telemetrystoretest"
	"github.com/SigNoz/signoz/pkg/telemetrytraces"
	"github.com/stretchr/testify/require"
)

type queryMatcherAny struct{}

func (m *queryMatcherAny) Match(x string, y string) error {
	return nil
}

// TestManagerOptions provides options for customizing the test manager creation.
type TestManagerOptions struct {
	// QueryRunner replaces storage execution in manager behavior tests. Lite's
	// compiler has direct tests; manager tests use this seam for rule semantics.
	QueryRunner QueryRunner

	// AlertmanagerHook is a function that will be called with the Alertmanager mock
	// after it's created but before it's used. This allows customizing the mock behavior.
	AlertmanagerHook func(alertmanager.Alertmanager)

	// SqlStoreHook is a function that will be called with the SQLStore mock
	// after it's created but before it's used. This allows customizing the mock behavior.
	SqlStoreHook func(sqlstore.SQLStore)

	// TelemetryStoreHook is a function that will be called with the TelemetryStore mock
	// after it's created but before it's used. This allows customizing the mock behavior.
	TelemetryStoreHook func(telemetrystore.TelemetryStore)
}

// NewTestManager creates a Manager instance for testing purposes.
// It sets up all the necessary mocks and dependencies required for testing.
// Options can be provided to customize the manager behavior. If nil, default options are used.
func NewTestManager(t *testing.T, testOpts *TestManagerOptions) *Manager {
	// mocking the alertmanager + capturing the triggered test alerts
	fAlert := alertmanagermock.NewMockAlertmanager(t)

	// Call the Alertmanager hook if provided
	if testOpts != nil && testOpts.AlertmanagerHook != nil {
		testOpts.AlertmanagerHook(fAlert)
	}

	cacheObj, err := cachetest.New(cache.Config{
		Provider: "memory",
		Memory: cache.Memory{
			NumCounters: 1000,
			MaxCost:     1 << 20,
		},
	})
	require.NoError(t, err)

	// Create SQLStore mock
	sqlStore := sqlstoretest.New(sqlstore.Config{Provider: "sqlite"}, sqlmock.QueryMatcherRegexp)

	// Call the SqlStore hook if provided
	if testOpts != nil && testOpts.SqlStoreHook != nil {
		testOpts.SqlStoreHook(sqlStore)
	}

	// Create TelemetryStore mock
	telemetryStore := telemetrystoretest.New(telemetrystore.Config{}, &queryMatcherAny{})

	// Call the TelemetryStore hook if provided
	if testOpts != nil && testOpts.TelemetryStoreHook != nil {
		testOpts.TelemetryStoreHook(telemetryStore)
	}

	providerSettings := instrumentationtest.New().ToProviderSettings()
	reader := rulestatehistorystore.New(
		instrumentationtest.New().Logger(),
		telemetryStore.ClickhouseDB(),
		rulestatehistorystore.DefaultConfig(),
	)

	metadataStore := telemetrymetadata.NewTelemetryMetaStore(
		providerSettings,
		telemetryStore,
		telemetrytraces.DBName,
		telemetrytraces.FieldValuesTableName,
		telemetrytraces.SpanAttributesKeysTblName,
		telemetrytraces.SpansTableName,
		telemetrymetrics.DBName,
		telemetrymetrics.MetricMetadataTableName,
		telemetrymeter.DBName,
		telemetrymeter.MeterRollup1dTableName,
		telemetrylogs.DBName,
		telemetrylogs.LogsTableName,
		telemetrylogs.FieldValuesTableName,
		telemetrylogs.LogAttributeKeysTblName,
		telemetrylogs.LogResourceKeysTblName,
	)

	var queryRunner QueryRunner
	if testOpts != nil {
		queryRunner = testOpts.QueryRunner
	}
	if queryRunner == nil {
		queryRunner = NewLiteQueryRunner(telemetryStore, metadataStore)
	}

	mgrOpts := &ManagerOptions{
		Logger:       instrumentationtest.New().Logger(),
		Cache:        cacheObj,
		Alertmanager: fAlert,
		QueryRunner:  queryRunner,
		Reader:       reader,
		SqlStore:     sqlStore, // SQLStore needed for SendAlerts to query organizations
	}

	mgr, err := NewManager(mgrOpts)
	require.NoError(t, err)

	return mgr
}
