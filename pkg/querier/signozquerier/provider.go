package signozquerier

import (
	"context"

	"github.com/SigNoz/signoz/pkg/factory"
	"github.com/SigNoz/signoz/pkg/querier"
	"github.com/SigNoz/signoz/pkg/telemetrylogs"
	"github.com/SigNoz/signoz/pkg/telemetrymetadata"
	"github.com/SigNoz/signoz/pkg/telemetrymeter"
	"github.com/SigNoz/signoz/pkg/telemetrymetrics"
	"github.com/SigNoz/signoz/pkg/telemetrystore"
	"github.com/SigNoz/signoz/pkg/telemetrytraces"
)

// NewFactory creates a new factory for the signoz querier provider
func NewFactory(telemetryStore telemetrystore.TelemetryStore) factory.ProviderFactory[querier.Querier, querier.Config] {
	return factory.NewProviderFactory(
		factory.MustNewName("siginsight"),
		func(
			ctx context.Context,
			settings factory.ProviderSettings,
			cfg querier.Config,
		) (querier.Querier, error) {
			return newProvider(ctx, settings, cfg, telemetryStore)
		},
	)
}

func newProvider(
	_ context.Context,
	settings factory.ProviderSettings,
	_ querier.Config,
	telemetryStore telemetrystore.TelemetryStore,
) (querier.Querier, error) {

	// Create telemetry metadata store
	telemetryMetadataStore := telemetrymetadata.NewTelemetryMetaStore(
		settings,
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

	return querier.New(settings, telemetryStore, telemetryMetadataStore), nil
}
