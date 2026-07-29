package storageconfig

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestDefaultSharedTraceTablesStayConsistent(t *testing.T) {
	config := Default()

	require.Equal(t, config.Trace.Database, config.Retention.TraceDB)
	require.Equal(t, config.Trace.IndexTable, config.Retention.TraceTable)
	require.Equal(t, config.Trace.ErrorTable, config.Retention.ErrorTable)
	require.Equal(t, config.Trace.DependencyGraphTable, config.Retention.DependencyGraphTable)
	require.Equal(t, config.Trace.SummaryTable, config.Retention.TraceSummaryTable)
}
