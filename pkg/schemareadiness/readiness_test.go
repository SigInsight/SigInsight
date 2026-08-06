package schemareadiness

import (
	"context"
	"regexp"
	"testing"

	"github.com/SigNoz/signoz/pkg/errors"
	"github.com/SigNoz/signoz/pkg/telemetrystore"
	"github.com/SigNoz/signoz/pkg/telemetrystore/telemetrystoretest"
	cmock "github.com/srikanthccv/ClickHouse-go-mock"
	"github.com/stretchr/testify/require"
)

type regexMatcher struct{}

func (regexMatcher) Match(expectedSQL, actualSQL string) error {
	matched, err := regexp.MatchString(expectedSQL, actualSQL)
	if err != nil {
		return err
	}
	if !matched {
		return errors.NewInternalf(errors.CodeInternal, "query %q does not match %q", actualSQL, expectedSQL)
	}
	return nil
}

func expectCompleteSchema(t *testing.T, store *telemetrystoretest.Provider, omitTable, omitColumn string, includeLegacy bool) {
	t.Helper()
	mock := store.Mock()
	tableRows := make([][]any, 0, len(required)+1)
	columnRows := make([][]any, 0)
	for _, item := range required {
		key := item.database + "." + item.table
		if key != omitTable {
			tableRows = append(tableRows, []any{item.database, item.table})
		}
		for _, column := range item.columns {
			if key+"."+column != omitColumn {
				columnRows = append(columnRows, []any{item.database, item.table, column})
			}
		}
	}
	if includeLegacy {
		tableRows = append(tableRows, []any{logsDB, "logs_v2"})
	}

	mock.ExpectQuery(`SELECT database, name FROM system\.tables`).WithArgs(logsDB, tracesDB, metricsDB, meterDB, analyticsDB).WillReturnRows(cmock.NewRows([]cmock.ColumnType{
		{Name: "database", Type: "String"},
		{Name: "name", Type: "String"},
	}, tableRows))
	if omitTable != "" || includeLegacy {
		return
	}
	mock.ExpectQuery(`SELECT database, table, name FROM system\.columns`).WithArgs(logsDB, tracesDB, metricsDB, meterDB, analyticsDB).WillReturnRows(cmock.NewRows([]cmock.ColumnType{
		{Name: "database", Type: "String"},
		{Name: "table", Type: "String"},
		{Name: "name", Type: "String"},
	}, columnRows))
}

func TestValidateAcceptsCanonicalSchema(t *testing.T) {
	store := telemetrystoretest.New(telemetrystore.Config{}, regexMatcher{})
	expectCompleteSchema(t, store, "", "", false)

	require.NoError(t, Validate(context.Background(), store))
	require.NoError(t, store.Mock().ExpectationsWereMet())
}

func TestValidateRejectsMissingTable(t *testing.T) {
	store := telemetrystoretest.New(telemetrystore.Config{}, regexMatcher{})
	expectCompleteSchema(t, store, tracesDB+".spans", "", false)

	err := Validate(context.Background(), store)
	require.ErrorContains(t, err, "siginsight_traces.spans")
	require.NoError(t, store.Mock().ExpectationsWereMet())
}

func TestValidateRejectsLegacyObjectsBeforeColumnRead(t *testing.T) {
	store := telemetrystoretest.New(telemetrystore.Config{}, regexMatcher{})
	mock := store.Mock()
	tableRows := make([][]any, 0, len(required)+1)
	for _, item := range required {
		tableRows = append(tableRows, []any{item.database, item.table})
	}
	tableRows = append(tableRows, []any{logsDB, "logs_v2"})
	mock.ExpectQuery(`SELECT database, name FROM system\.tables`).WithArgs(logsDB, tracesDB, metricsDB, meterDB, analyticsDB).WillReturnRows(cmock.NewRows([]cmock.ColumnType{
		{Name: "database", Type: "String"},
		{Name: "name", Type: "String"},
	}, tableRows))

	err := Validate(context.Background(), store)
	require.ErrorContains(t, err, "legacy ClickHouse schema objects remain")
	require.ErrorContains(t, err, "siginsight_logs.logs_v2")
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestValidateRejectsMissingColumn(t *testing.T) {
	store := telemetrystoretest.New(telemetrystore.Config{}, regexMatcher{})
	expectCompleteSchema(t, store, "", tracesDB+".spans.service_name_present", false)

	err := Validate(context.Background(), store)
	require.ErrorContains(t, err, "siginsight_traces.spans.service_name_present")
	require.NoError(t, store.Mock().ExpectationsWereMet())
}

func TestCanonicalTraceAndMetricRetentionUseTableTTL(t *testing.T) {
	for _, item := range required {
		if item.database != tracesDB && item.database != metricsDB {
			continue
		}
		for _, column := range item.columns {
			require.NotEqual(t, "_retention_days", column)
			require.NotEqual(t, "_retention_days_cold", column)
		}
	}
}
