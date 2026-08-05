package querier

import (
	"context"
	"encoding/json"
	"net/http"

	"github.com/SigNoz/signoz/pkg/analytics"
	"github.com/SigNoz/signoz/pkg/factory"
	"github.com/SigNoz/signoz/pkg/http/render"
	"github.com/SigNoz/signoz/pkg/types/authtypes"
	"github.com/SigNoz/signoz/pkg/types/ctxtypes"
	"github.com/SigNoz/signoz/pkg/types/instrumentationtypes"
	qbtypes "github.com/SigNoz/signoz/pkg/types/querybuildertypes/querybuildertypesv5"
	"github.com/SigNoz/signoz/pkg/valuer"
)

type handler struct {
	set       factory.ProviderSettings
	analytics analytics.Analytics
	querier   Querier
}

func NewHandler(set factory.ProviderSettings, querier Querier, analytics analytics.Analytics) Handler {
	return &handler{set: set, querier: querier, analytics: analytics}
}

func (handler *handler) QueryRange(rw http.ResponseWriter, req *http.Request) {
	ctx := req.Context()
	ctx = ctxtypes.NewContextWithCommentVals(ctx, map[string]string{
		instrumentationtypes.CodeNamespace:    "querier",
		instrumentationtypes.CodeFunctionName: "QueryRange",
	})

	claims, err := authtypes.ClaimsFromContext(ctx)
	if err != nil {
		render.Error(rw, err)
		return
	}

	var queryRangeRequest qbtypes.QueryRangeRequest
	if err := json.NewDecoder(req.Body).Decode(&queryRangeRequest); err != nil {
		render.Error(rw, err)
		return
	}

	// Validate the query request
	if err := queryRangeRequest.Validate(); err != nil {
		render.Error(rw, err)
		return
	}

	orgID, err := valuer.NewUUID(claims.OrgID)
	if err != nil {
		render.Error(rw, err)
		return
	}

	queryRangeResponse, err := handler.querier.QueryRange(ctx, orgID, &queryRangeRequest)
	if err != nil {
		render.Error(rw, err)
		return
	}

	handler.logEvent(req.Context(), req.Header.Get("Referer"), queryRangeResponse.QBEvent)

	render.Success(rw, http.StatusOK, queryRangeResponse)
}

func (handler *handler) logEvent(ctx context.Context, referrer string, event *qbtypes.QBEvent) {
	claims, err := authtypes.ClaimsFromContext(ctx)
	if err != nil {
		return
	}

	if !(event.LogsUsed || event.MetricsUsed || event.TracesUsed) {
		return
	}

	properties := map[string]any{
		"version":           event.Version,
		"logs_used":         event.LogsUsed,
		"traces_used":       event.TracesUsed,
		"metrics_used":      event.MetricsUsed,
		"source":            event.Source,
		"filter_applied":    event.FilterApplied,
		"group_by_applied":  event.GroupByApplied,
		"query_type":        event.QueryType,
		"panel_type":        event.PanelType,
		"number_of_queries": event.NumberOfQueries,
	}

	if referrer == "" {
		return
	}

	comments := ctxtypes.CommentFromContext(ctx).Map()
	for key, value := range comments {
		properties[key] = value
	}

	if !event.HasData {
		handler.analytics.TrackUser(ctx, claims.OrgID, claims.UserID, "Telemetry Query Returned Empty", properties)
		return
	}

	handler.analytics.TrackUser(ctx, claims.OrgID, claims.UserID, "Telemetry Query Returned Results", properties)
}
