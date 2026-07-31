package signoz

import (
	"github.com/SigNoz/signoz/pkg/analytics"
	"github.com/SigNoz/signoz/pkg/assistant"
	"github.com/SigNoz/signoz/pkg/assistant/implassistant"
	"github.com/SigNoz/signoz/pkg/authz"
	"github.com/SigNoz/signoz/pkg/authz/signozauthzapi"
	"github.com/SigNoz/signoz/pkg/factory"
	"github.com/SigNoz/signoz/pkg/flagger"
	"github.com/SigNoz/signoz/pkg/modules/apdex"
	"github.com/SigNoz/signoz/pkg/modules/apdex/implapdex"
	"github.com/SigNoz/signoz/pkg/modules/fields"
	"github.com/SigNoz/signoz/pkg/modules/fields/implfields"
	"github.com/SigNoz/signoz/pkg/modules/metricsexplorer"
	"github.com/SigNoz/signoz/pkg/modules/metricsexplorer/implmetricsexplorer"
	"github.com/SigNoz/signoz/pkg/modules/quickfilter"
	"github.com/SigNoz/signoz/pkg/modules/quickfilter/implquickfilter"
	"github.com/SigNoz/signoz/pkg/modules/savedview"
	"github.com/SigNoz/signoz/pkg/modules/savedview/implsavedview"
	"github.com/SigNoz/signoz/pkg/modules/services"
	"github.com/SigNoz/signoz/pkg/modules/services/implservices"
	"github.com/SigNoz/signoz/pkg/modules/spanpercentile"
	"github.com/SigNoz/signoz/pkg/modules/spanpercentile/implspanpercentile"
	"github.com/SigNoz/signoz/pkg/modules/tracefunnel"
	"github.com/SigNoz/signoz/pkg/modules/tracefunnel/impltracefunnel"
	"github.com/SigNoz/signoz/pkg/querier"
	"github.com/SigNoz/signoz/pkg/types/telemetrytypes"
)

type Handlers struct {
	Assistant       assistant.Handler
	SavedView       savedview.Handler
	Apdex           apdex.Handler
	QuickFilter     quickfilter.Handler
	TraceFunnel     tracefunnel.Handler
	SpanPercentile  spanpercentile.Handler
	Services        services.Handler
	MetricsExplorer metricsexplorer.Handler
	FlaggerHandler  flagger.Handler
	Fields          fields.Handler
	AuthzHandler    authz.Handler
	QuerierHandler  querier.Handler
	RegistryHandler factory.Handler
}

func NewHandlers(
	modules Modules,
	providerSettings factory.ProviderSettings,
	analytics analytics.Analytics,
	querierHandler querier.Handler,
	flaggerService flagger.Flagger,
	telemetryMetadataStore telemetrytypes.MetadataStore,
	authz authz.AuthZ,
	registryHandler factory.Handler,
) Handlers {
	return Handlers{
		Assistant:       implassistant.NewHandler(modules.Assistant),
		SavedView:       implsavedview.NewHandler(modules.SavedView),
		Apdex:           implapdex.NewHandler(modules.Apdex),
		QuickFilter:     implquickfilter.NewHandler(modules.QuickFilter),
		TraceFunnel:     impltracefunnel.NewHandler(modules.TraceFunnel),
		Services:        implservices.NewHandler(modules.Services),
		MetricsExplorer: implmetricsexplorer.NewHandler(modules.MetricsExplorer),
		SpanPercentile:  implspanpercentile.NewHandler(modules.SpanPercentile),
		FlaggerHandler:  flagger.NewHandler(flaggerService),
		Fields:          implfields.NewHandler(providerSettings, telemetryMetadataStore),
		AuthzHandler:    signozauthzapi.NewHandler(authz),
		QuerierHandler:  querierHandler,
		RegistryHandler: registryHandler,
	}
}
