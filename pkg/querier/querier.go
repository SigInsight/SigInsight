package querier

import (
	"context"
	"log/slog"

	"github.com/SigNoz/signoz/pkg/errors"
	"github.com/SigNoz/signoz/pkg/factory"
	"github.com/SigNoz/signoz/pkg/telemetrystore"
	qbtypes "github.com/SigNoz/signoz/pkg/types/querybuildertypes/querybuildertypesv5"
	"github.com/SigNoz/signoz/pkg/types/telemetrytypes"
	"github.com/SigNoz/signoz/pkg/valuer"
)

// querier is the V5 HTTP compatibility boundary for the lightweight engine.
// It deliberately owns neither legacy statement builders nor result caches.
type querier struct {
	logger         *slog.Logger
	telemetryStore telemetrystore.TelemetryStore
	metadataStore  telemetrytypes.MetadataStore
}

var _ Querier = (*querier)(nil)

func New(
	settings factory.ProviderSettings,
	telemetryStore telemetrystore.TelemetryStore,
	metadataStore telemetrytypes.MetadataStore,
) *querier {
	querierSettings := factory.NewScopedProviderSettings(settings, "github.com/SigNoz/signoz/pkg/querier")
	return &querier{
		logger:         querierSettings.Logger(),
		telemetryStore: telemetryStore,
		metadataStore:  metadataStore,
	}
}

func (q *querier) QueryRange(ctx context.Context, _ valuer.UUID, req *qbtypes.QueryRangeRequest) (*qbtypes.QueryRangeResponse, error) {
	if req == nil {
		return nil, errors.NewInvalidInputf(errors.CodeInvalidInput, "V5 request is required")
	}
	return q.queryRangeLite(ctx, req)
}
