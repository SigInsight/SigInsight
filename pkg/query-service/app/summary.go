package app

import (
	"bytes"
	"io"
	"net/http"

	"github.com/SigNoz/signoz/pkg/errors"
	"github.com/SigNoz/signoz/pkg/http/render"
	"github.com/SigNoz/signoz/pkg/types/authtypes"
	"github.com/SigNoz/signoz/pkg/valuer"
	"log/slog"

	explorer "github.com/SigNoz/signoz/pkg/query-service/app/metricsexplorer"
)

func (aH *APIHandler) FilterKeysSuggestion(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	bodyBytes, _ := io.ReadAll(r.Body)
	r.Body = io.NopCloser(bytes.NewBuffer(bodyBytes))
	params, apiError := explorer.ParseFilterKeySuggestions(r)
	if apiError != nil {
		slog.ErrorContext(ctx, "error parsing summary filter keys request", errors.Attr(apiError.Err))
		RespondError(w, apiError, nil)
		return
	}
	keys, err := aH.SummaryService.FilterKeys(ctx, params)
	if err != nil {
		slog.ErrorContext(ctx, "error getting filter keys", errors.Attr(err))
		render.Error(w, err)
		return
	}
	aH.Respond(w, keys)
}

func (aH *APIHandler) FilterValuesSuggestion(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	claims, err := authtypes.ClaimsFromContext(r.Context())
	if err != nil {
		render.Error(w, err)
		return
	}
	orgID, err := valuer.NewUUID(claims.OrgID)
	if err != nil {
		render.Error(w, err)
		return
	}

	bodyBytes, _ := io.ReadAll(r.Body)
	r.Body = io.NopCloser(bytes.NewBuffer(bodyBytes))
	params, apiError := explorer.ParseFilterValueSuggestions(r)
	if apiError != nil {
		slog.ErrorContext(ctx, "error parsing summary filter values request", errors.Attr(apiError.Err))
		RespondError(w, apiError, nil)
		return
	}

	values, err := aH.SummaryService.FilterValues(ctx, orgID, params)
	if err != nil {
		slog.ErrorContext(ctx, "error getting filter values", errors.Attr(err))
		render.Error(w, err)
		return
	}
	aH.Respond(w, values)
}

func (aH *APIHandler) GetRelatedMetrics(w http.ResponseWriter, r *http.Request) {
	bodyBytes, _ := io.ReadAll(r.Body)
	r.Body = io.NopCloser(bytes.NewBuffer(bodyBytes))
	ctx := r.Context()
	params, apiError := explorer.ParseRelatedMetricsParams(r)
	if apiError != nil {
		slog.ErrorContext(ctx, "error parsing related metric params", errors.Attr(apiError.Err))
		RespondError(w, apiError, nil)
		return
	}
	result, err := aH.SummaryService.GetRelatedMetrics(ctx, params)
	if err != nil {
		slog.ErrorContext(ctx, "error getting related metrics", errors.Attr(err))
		render.Error(w, err)
		return
	}
	aH.Respond(w, result)

}

func (aH *APIHandler) GetInspectMetricsData(w http.ResponseWriter, r *http.Request) {
	bodyBytes, _ := io.ReadAll(r.Body)
	r.Body = io.NopCloser(bytes.NewBuffer(bodyBytes))
	ctx := r.Context()
	params, apiError := explorer.ParseInspectMetricsParams(r)
	if apiError != nil {
		slog.ErrorContext(ctx, "error parsing inspect metric params", errors.Attr(apiError.Err))
		RespondError(w, apiError, nil)
		return
	}
	result, err := aH.SummaryService.GetInspectMetrics(ctx, params)
	if err != nil {
		slog.ErrorContext(ctx, "error getting inspect metrics data", errors.Attr(err))
		render.Error(w, err)
		return
	}
	aH.Respond(w, result)

}
