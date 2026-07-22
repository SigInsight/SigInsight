package telemetrymetadata

import otelcollectorconst "github.com/SigInsight/OtelCollector/constants"

const (
	DBName                           = "signoz_metadata"
	AttributesMetadataTableName      = "distributed_attributes_metadata"
	AttributesMetadataLocalTableName = "attributes_metadata"
	PathTypesTableName               = otelcollectorconst.DistributedPathTypesTable
	// Column Evolution table stores promoted paths as (signal, column_name, field_context, field_name); see siginsight-otel-collector metadata_migrations.
	PromotedPathsTableName = "distributed_column_evolution_metadata"
	SkipIndexTableName     = "system.data_skipping_indices"
)
