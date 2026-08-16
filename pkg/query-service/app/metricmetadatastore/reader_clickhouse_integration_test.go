//go:build integration

package metricmetadatastore

import (
	"context"
	"io"
	"log/slog"
	"os"
	"testing"

	"github.com/ClickHouse/clickhouse-go/v2"

	"github.com/SigNoz/signoz/pkg/valuer"
)

func TestGetMetricsMetadataBindsArrayOnClickHouse(t *testing.T) {
	dsn := os.Getenv("SIGINSIGHT_CLICKHOUSE_INTEGRATION_DSN")
	if dsn == "" {
		t.Skip("set SIGINSIGHT_CLICKHOUSE_INTEGRATION_DSN to run ClickHouse metadata integration")
	}
	options, err := clickhouse.ParseDSN(dsn)
	if err != nil {
		t.Fatalf("ParseDSN() error = %v", err)
	}
	conn, err := clickhouse.Open(options)
	if err != nil {
		t.Fatalf("Open() error = %v", err)
	}
	defer conn.Close()

	reader := New(
		slog.New(slog.NewTextHandler(io.Discard, nil)),
		conn,
		&fakeCache{data: map[string][]byte{}},
	)
	metadata, err := reader.GetMetricsMetadata(
		context.Background(),
		valuer.GenerateUUID(),
		"__missing_metric_binding_probe__",
	)
	if err != nil {
		t.Fatalf("GetMetricsMetadata() error = %v", err)
	}
	if len(metadata) != 0 {
		t.Fatalf("metadata = %#v, want no rows for probe metric", metadata)
	}
}
