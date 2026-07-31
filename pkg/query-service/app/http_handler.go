package app

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"

	"github.com/SigNoz/signoz/pkg/errors"
	"github.com/SigNoz/signoz/pkg/flagger"
	"github.com/SigNoz/signoz/pkg/livelogs"

	"io"
	"log/slog"
	"math"
	"net/http"
	"strconv"
	"time"

	"github.com/SigNoz/signoz/pkg/alertmanager"
	errorsV2 "github.com/SigNoz/signoz/pkg/errors"
	"github.com/SigNoz/signoz/pkg/http/middleware"
	"github.com/SigNoz/signoz/pkg/http/render"
	"github.com/SigNoz/signoz/pkg/query-service/app/metricsexplorer"
	"github.com/SigNoz/signoz/pkg/signoz"
	"github.com/SigNoz/signoz/pkg/valuer"

	"github.com/gorilla/mux"
	jsoniter "github.com/json-iterator/go"
	_ "modernc.org/sqlite"

	"github.com/SigNoz/signoz/pkg/contextlinks"
	traceFunnelsModule "github.com/SigNoz/signoz/pkg/modules/tracefunnel"
	"github.com/SigNoz/signoz/pkg/query-service/constants"
	"github.com/SigNoz/signoz/pkg/types"
	"github.com/SigNoz/signoz/pkg/types/authtypes"
	"github.com/SigNoz/signoz/pkg/types/featuretypes"
	"github.com/SigNoz/signoz/pkg/types/licensetypes"
	qbtypes "github.com/SigNoz/signoz/pkg/types/querybuildertypes/querybuildertypesv5"
	"github.com/SigNoz/signoz/pkg/types/ruletypes"
	traceFunnels "github.com/SigNoz/signoz/pkg/types/tracefunneltypes"

	"github.com/SigNoz/signoz/pkg/query-service/interfaces"
	"github.com/SigNoz/signoz/pkg/query-service/model"
	"github.com/SigNoz/signoz/pkg/query-service/rules"
	"github.com/SigNoz/signoz/pkg/version"
)

type status string

const (
	statusSuccess status = "success"
	statusError   status = "error"
)

// NewRouter creates and configures a Gorilla Router.
func NewRouter() *mux.Router {
	return mux.NewRouter().UseEncodedPath()
}

// APIHandler implements the query service public API
type APIHandler struct {
	logger           *slog.Logger
	services         interfaces.ServiceReader
	retention        interfaces.RetentionReader
	exceptions       interfaces.ExceptionReader
	traceDetail      interfaces.TraceDetailReader
	ruleStateHistory interfaces.RuleStateHistoryQueryReader
	clickHouseHealth interfaces.ClickHouseHealthReader
	metricMetadata   interfaces.MetricMetadataReader
	traceFunnelQuery interfaces.TraceFunnelQueryReader
	ruleManager      *rules.Manager

	// SetupCompleted indicates if SigNoz is ready for general use.
	// at the moment, we mark the app ready when the first user
	// is registers.
	SetupCompleted bool

	SummaryService *metricsexplorer.SummaryService

	AlertmanagerAPI *alertmanager.API

	Signoz *signoz.SigNoz
}

type APIHandlerOpts struct {
	Services         interfaces.ServiceReader
	Retention        interfaces.RetentionReader
	Exceptions       interfaces.ExceptionReader
	TraceDetail      interfaces.TraceDetailReader
	RuleStateHistory interfaces.RuleStateHistoryQueryReader
	ClickHouseHealth interfaces.ClickHouseHealthReader
	MetricMetadata   interfaces.MetricMetadataReader
	MetricsExplorer  interfaces.MetricsExplorerReader
	TraceFunnelQuery interfaces.TraceFunnelQueryReader

	// rule manager handles rule crud operations
	RuleManager *rules.Manager

	AlertmanagerAPI *alertmanager.API

	Signoz *signoz.SigNoz
}

// NewAPIHandler returns an APIHandler
func NewAPIHandler(opts APIHandlerOpts, config signoz.Config) (*APIHandler, error) {
	summaryService := metricsexplorer.NewSummaryService(opts.MetricsExplorer, opts.RuleManager)
	//quickFilterModule := quickfilter.NewAPI(opts.QuickFilterModule)

	aH := &APIHandler{
		logger:           slog.Default(),
		services:         opts.Services,
		retention:        opts.Retention,
		exceptions:       opts.Exceptions,
		traceDetail:      opts.TraceDetail,
		ruleStateHistory: opts.RuleStateHistory,
		clickHouseHealth: opts.ClickHouseHealth,
		metricMetadata:   opts.MetricMetadata,
		traceFunnelQuery: opts.TraceFunnelQuery,
		ruleManager:      opts.RuleManager,
		SummaryService:   summaryService,
		AlertmanagerAPI:  opts.AlertmanagerAPI,
		Signoz:           opts.Signoz,
	}

	// TODO(nitya): remote this in later for multitenancy.
	orgs, err := opts.Signoz.Modules.OrgGetter.ListByOwnedKeyRange(context.Background())
	if err != nil {
		aH.logger.Warn("unexpected error while fetching orgs while initializing base api handler", errors.Attr(err))
	}
	// if the first org with the first user is created then the setup is complete.
	if len(orgs) == 1 {
		count, err := opts.Signoz.Modules.UserGetter.CountByOrgID(context.Background(), orgs[0].ID)
		if err != nil {
			aH.logger.Warn("unexpected error while fetching user count while initializing base api handler", errors.Attr(err))
		}

		if count > 0 {
			aH.SetupCompleted = true
		}
	}

	// If the root user is enabled, the setup is complete
	if config.User.Root.Enabled {
		aH.SetupCompleted = true
	}

	return aH, nil
}

// todo(remove): Implemented at render package (github.com/SigNoz/signoz/pkg/http/render) with the new error structure
type structuredResponse struct {
	Data   interface{}       `json:"data"`
	Total  int               `json:"total"`
	Limit  int               `json:"limit"`
	Offset int               `json:"offset"`
	Errors []structuredError `json:"errors"`
}

// todo(remove): Implemented at render package (github.com/SigNoz/signoz/pkg/http/render) with the new error structure
type structuredError struct {
	Code int    `json:"code,omitempty"`
	Msg  string `json:"msg"`
}

// todo(remove): Implemented at render package (github.com/SigNoz/signoz/pkg/http/render) with the new error structure
type ApiResponse struct {
	Status    status          `json:"status"`
	Data      interface{}     `json:"data,omitempty"`
	ErrorType model.ErrorType `json:"errorType,omitempty"`
	Error     string          `json:"error,omitempty"`
}

// todo(remove): Implemented at render package (github.com/SigNoz/signoz/pkg/http/render) with the new error structure
func RespondError(w http.ResponseWriter, apiErr model.BaseApiError, data interface{}) {
	json := jsoniter.ConfigCompatibleWithStandardLibrary
	b, err := json.Marshal(&ApiResponse{
		Status:    statusError,
		ErrorType: apiErr.Type(),
		Error:     apiErr.Error(),
		Data:      data,
	})
	if err != nil {
		slog.Error("error marshalling json response", errors.Attr(err))
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	var code int
	switch apiErr.Type() {
	case model.ErrorBadData:
		code = http.StatusBadRequest
	case model.ErrorExec:
		code = 422
	case model.ErrorCanceled, model.ErrorTimeout:
		code = http.StatusServiceUnavailable
	case model.ErrorInternal:
		code = http.StatusInternalServerError
	case model.ErrorNotFound:
		code = http.StatusNotFound
	case model.ErrorNotImplemented:
		code = http.StatusNotImplemented
	case model.ErrorUnauthorized:
		code = http.StatusUnauthorized
	case model.ErrorForbidden:
		code = http.StatusForbidden
	case model.ErrorConflict:
		code = http.StatusConflict
	default:
		code = http.StatusInternalServerError
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	if n, err := w.Write(b); err != nil {
		slog.Error("error writing response", "bytes_written", n, errors.Attr(err))
	}
}

// todo(remove): Implemented at render package (github.com/SigNoz/signoz/pkg/http/render) with the new error structure
func writeHttpResponse(w http.ResponseWriter, data interface{}) {
	json := jsoniter.ConfigCompatibleWithStandardLibrary
	b, err := json.Marshal(&ApiResponse{
		Status: statusSuccess,
		Data:   data,
	})
	if err != nil {
		slog.Error("error marshalling json response", errors.Attr(err))
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	if n, err := w.Write(b); err != nil {
		slog.Error("error writing response", "bytes_written", n, errors.Attr(err))
	}
}

func (aH *APIHandler) RegisterQueryRangeV5Routes(router *mux.Router, am *middleware.AuthZ) {
	subRouter := router.PathPrefix("/api/v5").Subrouter()
	subRouter.HandleFunc("/logs/livetail", am.ViewAccess(livelogs.New(aH.Signoz.TelemetryStore).Stream)).Methods(http.MethodGet)
	subRouter.HandleFunc("/metric/metric_metadata", am.ViewAccess(aH.getMetricMetadata)).Methods(http.MethodGet)
}

// todo(remove): Implemented at render package (github.com/SigNoz/signoz/pkg/http/render) with the new error structure
func (aH *APIHandler) Respond(w http.ResponseWriter, data interface{}) {
	writeHttpResponse(w, data)
}

// RegisterRoutes registers routes for this handler on the given router
func (aH *APIHandler) RegisterRoutes(router *mux.Router, am *middleware.AuthZ) {
	router.HandleFunc("/api/v5/channels", am.ViewAccess(aH.AlertmanagerAPI.ListChannels)).Methods(http.MethodGet)
	router.HandleFunc("/api/v5/channels/{id}", am.ViewAccess(aH.AlertmanagerAPI.GetChannelByID)).Methods(http.MethodGet)
	router.HandleFunc("/api/v5/channels/{id}", am.AdminAccess(aH.AlertmanagerAPI.UpdateChannelByID)).Methods(http.MethodPut)
	router.HandleFunc("/api/v5/channels/{id}", am.AdminAccess(aH.AlertmanagerAPI.DeleteChannelByID)).Methods(http.MethodDelete)
	router.HandleFunc("/api/v5/channels", am.EditAccess(aH.AlertmanagerAPI.CreateChannel)).Methods(http.MethodPost)
	router.HandleFunc("/api/v5/testChannel", am.EditAccess(aH.AlertmanagerAPI.TestReceiver)).Methods(http.MethodPost)

	router.HandleFunc("/api/v5/alerts", am.ViewAccess(aH.AlertmanagerAPI.GetAlerts)).Methods(http.MethodGet)

	router.HandleFunc("/api/v5/rules", am.ViewAccess(aH.listRules)).Methods(http.MethodGet)
	router.HandleFunc("/api/v5/rules/{id}", am.ViewAccess(aH.getRule)).Methods(http.MethodGet)
	router.HandleFunc("/api/v5/rules", am.EditAccess(aH.createRule)).Methods(http.MethodPost)
	router.HandleFunc("/api/v5/rules/{id}", am.EditAccess(aH.editRule)).Methods(http.MethodPut)
	router.HandleFunc("/api/v5/rules/{id}", am.EditAccess(aH.deleteRule)).Methods(http.MethodDelete)
	router.HandleFunc("/api/v5/rules/{id}", am.EditAccess(aH.patchRule)).Methods(http.MethodPatch)
	router.HandleFunc("/api/v5/testRule", am.EditAccess(aH.testRule)).Methods(http.MethodPost)
	router.HandleFunc("/api/v5/rules/{id}/history/stats", am.ViewAccess(aH.getRuleStats)).Methods(http.MethodPost)
	router.HandleFunc("/api/v5/rules/{id}/history/timeline", am.ViewAccess(aH.getRuleStateHistory)).Methods(http.MethodPost)
	router.HandleFunc("/api/v5/rules/{id}/history/top_contributors", am.ViewAccess(aH.getRuleStateHistoryTopContributors)).Methods(http.MethodPost)
	router.HandleFunc("/api/v5/rules/{id}/history/overall_status", am.ViewAccess(aH.getOverallStateTransitions)).Methods(http.MethodPost)

	router.HandleFunc("/api/v5/explorer/views", am.ViewAccess(aH.Signoz.Handlers.SavedView.List)).Methods(http.MethodGet)
	router.HandleFunc("/api/v5/explorer/views", am.EditAccess(aH.Signoz.Handlers.SavedView.Create)).Methods(http.MethodPost)
	router.HandleFunc("/api/v5/explorer/views/{viewId}", am.ViewAccess(aH.Signoz.Handlers.SavedView.Get)).Methods(http.MethodGet)
	router.HandleFunc("/api/v5/explorer/views/{viewId}", am.EditAccess(aH.Signoz.Handlers.SavedView.Update)).Methods(http.MethodPut)
	router.HandleFunc("/api/v5/explorer/views/{viewId}", am.EditAccess(aH.Signoz.Handlers.SavedView.Delete)).Methods(http.MethodDelete)
	router.HandleFunc("/api/v5/event", am.ViewAccess(aH.registerEvent)).Methods(http.MethodPost)

	router.HandleFunc("/api/v5/services", am.ViewAccess(aH.Signoz.Handlers.Services.Get)).Methods(http.MethodPost)

	router.HandleFunc("/api/v5/service/top_operations", am.ViewAccess(aH.Signoz.Handlers.Services.GetTopOperations)).Methods(http.MethodPost)
	router.HandleFunc("/api/v5/service/top_level_operations", am.ViewAccess(aH.getServicesTopLevelOps)).Methods(http.MethodPost)

	router.HandleFunc("/api/v5/service/entry_point_operations", am.ViewAccess(aH.Signoz.Handlers.Services.GetEntryPointOperations)).Methods(http.MethodPost)
	router.HandleFunc("/api/v5/services/dependency_graph", am.ViewAccess(aH.dependencyGraph)).Methods(http.MethodPost)
	router.HandleFunc("/api/v5/settings/ttl", am.AdminAccess(aH.setTTL)).Methods(http.MethodPost)
	router.HandleFunc("/api/v5/settings/ttl", am.ViewAccess(aH.getTTL)).Methods(http.MethodGet)
	router.HandleFunc("/api/v5/settings/logs/ttl", am.AdminAccess(aH.setCustomRetentionTTL)).Methods(http.MethodPost)
	router.HandleFunc("/api/v5/settings/logs/ttl", am.ViewAccess(aH.getCustomRetentionTTL)).Methods(http.MethodGet)

	router.HandleFunc("/api/v5/settings/apdex", am.AdminAccess(aH.Signoz.Handlers.Apdex.Set)).Methods(http.MethodPost)
	router.HandleFunc("/api/v5/settings/apdex", am.ViewAccess(aH.Signoz.Handlers.Apdex.Get)).Methods(http.MethodGet)

	router.HandleFunc("/api/v5/traces/flamegraph/{traceId}", am.ViewAccess(aH.GetFlamegraphSpansForTrace)).Methods(http.MethodPost)
	router.HandleFunc("/api/v5/traces/waterfall/{traceId}", am.ViewAccess(aH.GetWaterfallSpansForTraceWithMetadata)).Methods(http.MethodPost)

	router.HandleFunc("/api/v5/version", am.OpenAccess(aH.getVersion)).Methods(http.MethodGet)
	router.HandleFunc("/api/v5/features/ui", am.ViewAccess(aH.getFeatureFlags)).Methods(http.MethodGet)
	router.HandleFunc("/api/v5/health", am.OpenAccess(aH.getHealth)).Methods(http.MethodGet)

	router.HandleFunc("/api/v5/exceptions", am.ViewAccess(aH.listErrors)).Methods(http.MethodPost)
	router.HandleFunc("/api/v5/exceptions/count", am.ViewAccess(aH.countErrors)).Methods(http.MethodPost)
	router.HandleFunc("/api/v5/exceptions/by-error-id", am.ViewAccess(aH.getErrorFromErrorID)).Methods(http.MethodGet)
	router.HandleFunc("/api/v5/exceptions/by-group-id", am.ViewAccess(aH.getErrorFromGroupID)).Methods(http.MethodGet)
	router.HandleFunc("/api/v5/exceptions/next-prev", am.ViewAccess(aH.getNextPrevErrorIDs)).Methods(http.MethodGet)

	router.HandleFunc("/api/v5/settings/disks", am.ViewAccess(aH.getDisks)).Methods(http.MethodGet)

	// Quick Filters
	router.HandleFunc("/api/v5/orgs/me/filters", am.ViewAccess(aH.Signoz.Handlers.QuickFilter.GetQuickFilters)).Methods(http.MethodGet)
	router.HandleFunc("/api/v5/orgs/me/filters/{signal}", am.ViewAccess(aH.Signoz.Handlers.QuickFilter.GetSignalFilters)).Methods(http.MethodGet)
	router.HandleFunc("/api/v5/orgs/me/filters", am.AdminAccess(aH.Signoz.Handlers.QuickFilter.UpdateQuickFilters)).Methods(http.MethodPut)

	router.HandleFunc("/api/v5/register", am.OpenAccess(aH.registerUser)).Methods(http.MethodPost)

	router.HandleFunc("/api/v5/traces/span_percentile", am.ViewAccess(aH.Signoz.Handlers.SpanPercentile.GetSpanPercentileDetails)).Methods(http.MethodPost)

}

func (ah *APIHandler) MetricExplorerRoutes(router *mux.Router, am *middleware.AuthZ) {
	router.HandleFunc("/api/v5/metrics/filters/keys",
		am.ViewAccess(ah.FilterKeysSuggestion)).
		Methods(http.MethodGet)
	router.HandleFunc("/api/v5/metrics/filters/values",
		am.ViewAccess(ah.FilterValuesSuggestion)).
		Methods(http.MethodPost)
	router.HandleFunc("/api/v5/metrics/related",
		am.ViewAccess(ah.GetRelatedMetrics)).
		Methods(http.MethodPost)
	router.HandleFunc("/api/v5/metrics/inspect",
		am.ViewAccess(ah.GetInspectMetricsData)).
		Methods(http.MethodPost)
}

func Intersection(a, b []int) (c []int) {
	m := make(map[int]bool)

	for _, item := range a {
		m[item] = true
	}

	for _, item := range b {
		if _, ok := m[item]; ok {
			c = append(c, item)
		}
	}
	return
}

func (aH *APIHandler) getRule(w http.ResponseWriter, r *http.Request) {
	idStr := mux.Vars(r)["id"]
	id, err := valuer.NewUUID(idStr)
	if err != nil {
		RespondError(w, &model.ApiError{Typ: model.ErrorBadData, Err: err}, nil)
		return
	}

	ruleResponse, err := aH.ruleManager.GetRule(r.Context(), id)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			RespondError(w, &model.ApiError{Typ: model.ErrorNotFound, Err: fmt.Errorf("rule not found")}, nil)
			return
		}
		RespondError(w, &model.ApiError{Typ: model.ErrorInternal, Err: err}, nil)
		return
	}
	aH.Respond(w, ruleResponse)
}

func ruleIDFromRequest(r *http.Request) (valuer.UUID, error) {
	return valuer.NewUUID(mux.Vars(r)["id"])
}

func (aH *APIHandler) getRuleStats(w http.ResponseWriter, r *http.Request) {
	ruleID, err := ruleIDFromRequest(r)
	if err != nil {
		RespondError(w, &model.ApiError{Typ: model.ErrorBadData, Err: err}, nil)
		return
	}

	params := model.QueryRuleStateHistory{}
	err = json.NewDecoder(r.Body).Decode(&params)
	if err != nil {
		RespondError(w, &model.ApiError{Typ: model.ErrorBadData, Err: err}, nil)
		return
	}

	totalCurrentTriggers, err := aH.ruleStateHistory.GetTotalTriggers(r.Context(), ruleID.StringValue(), &params)
	if err != nil {
		RespondError(w, &model.ApiError{Typ: model.ErrorInternal, Err: err}, nil)
		return
	}
	currentTriggersSeries, err := aH.ruleStateHistory.GetTriggersByInterval(r.Context(), ruleID.StringValue(), &params)
	if err != nil {
		RespondError(w, &model.ApiError{Typ: model.ErrorInternal, Err: err}, nil)
		return
	}

	currentAvgResolutionTime, err := aH.ruleStateHistory.GetAvgResolutionTime(r.Context(), ruleID.StringValue(), &params)
	if err != nil {
		RespondError(w, &model.ApiError{Typ: model.ErrorInternal, Err: err}, nil)
		return
	}
	currentAvgResolutionTimeSeries, err := aH.ruleStateHistory.GetAvgResolutionTimeByInterval(r.Context(), ruleID.StringValue(), &params)
	if err != nil {
		RespondError(w, &model.ApiError{Typ: model.ErrorInternal, Err: err}, nil)
		return
	}

	if params.End-params.Start >= 86400000 {
		days := int64(math.Ceil(float64(params.End-params.Start) / 86400000))
		params.Start -= days * 86400000
		params.End -= days * 86400000
	} else {
		params.Start -= 86400000
		params.End -= 86400000
	}

	totalPastTriggers, err := aH.ruleStateHistory.GetTotalTriggers(r.Context(), ruleID.StringValue(), &params)
	if err != nil {
		RespondError(w, &model.ApiError{Typ: model.ErrorInternal, Err: err}, nil)
		return
	}
	pastTriggersSeries, err := aH.ruleStateHistory.GetTriggersByInterval(r.Context(), ruleID.StringValue(), &params)
	if err != nil {
		RespondError(w, &model.ApiError{Typ: model.ErrorInternal, Err: err}, nil)
		return
	}

	pastAvgResolutionTime, err := aH.ruleStateHistory.GetAvgResolutionTime(r.Context(), ruleID.StringValue(), &params)
	if err != nil {
		RespondError(w, &model.ApiError{Typ: model.ErrorInternal, Err: err}, nil)
		return
	}
	pastAvgResolutionTimeSeries, err := aH.ruleStateHistory.GetAvgResolutionTimeByInterval(r.Context(), ruleID.StringValue(), &params)
	if err != nil {
		RespondError(w, &model.ApiError{Typ: model.ErrorInternal, Err: err}, nil)
		return
	}
	if math.IsNaN(currentAvgResolutionTime) || math.IsInf(currentAvgResolutionTime, 0) {
		currentAvgResolutionTime = 0
	}
	if math.IsNaN(pastAvgResolutionTime) || math.IsInf(pastAvgResolutionTime, 0) {
		pastAvgResolutionTime = 0
	}

	stats := model.Stats{
		TotalCurrentTriggers:           totalCurrentTriggers,
		TotalPastTriggers:              totalPastTriggers,
		CurrentTriggersSeries:          currentTriggersSeries,
		PastTriggersSeries:             pastTriggersSeries,
		CurrentAvgResolutionTime:       strconv.FormatFloat(currentAvgResolutionTime, 'f', -1, 64),
		PastAvgResolutionTime:          strconv.FormatFloat(pastAvgResolutionTime, 'f', -1, 64),
		CurrentAvgResolutionTimeSeries: currentAvgResolutionTimeSeries,
		PastAvgResolutionTimeSeries:    pastAvgResolutionTimeSeries,
	}

	aH.Respond(w, stats)
}

func (aH *APIHandler) getOverallStateTransitions(w http.ResponseWriter, r *http.Request) {
	ruleID, err := ruleIDFromRequest(r)
	if err != nil {
		RespondError(w, &model.ApiError{Typ: model.ErrorBadData, Err: err}, nil)
		return
	}

	params := model.QueryRuleStateHistory{}
	err = json.NewDecoder(r.Body).Decode(&params)
	if err != nil {
		RespondError(w, &model.ApiError{Typ: model.ErrorBadData, Err: err}, nil)
		return
	}

	stateItems, err := aH.ruleStateHistory.GetOverallStateTransitions(r.Context(), ruleID.StringValue(), &params)
	if err != nil {
		RespondError(w, &model.ApiError{Typ: model.ErrorInternal, Err: err}, nil)
		return
	}

	aH.Respond(w, stateItems)
}

func relatedRuleLinks(rule *ruletypes.GettableRule, start, end time.Time, labels map[string]string) (string, string) {
	if rule == nil || rule.RuleCondition == nil || rule.RuleCondition.CompositeQuery == nil {
		return "", ""
	}

	selectedQuery := rule.RuleCondition.GetSelectedQueryName()
	for _, query := range rule.RuleCondition.CompositeQuery.Queries {
		switch spec := query.Spec.(type) {
		case qbtypes.QueryBuilderQuery[qbtypes.LogAggregation]:
			if rule.AlertType != ruletypes.AlertTypeLogs || spec.Name != selectedQuery {
				continue
			}
			filterExpression := ""
			if spec.Filter != nil {
				filterExpression = spec.Filter.Expression
			}
			whereClause := contextlinks.PrepareFilterExpression(labels, filterExpression, spec.GroupBy)
			return contextlinks.PrepareLinksToLogsV5(start, end, whereClause), ""
		case qbtypes.QueryBuilderQuery[qbtypes.TraceAggregation]:
			if rule.AlertType != ruletypes.AlertTypeTraces || spec.Name != selectedQuery {
				continue
			}
			filterExpression := ""
			if spec.Filter != nil {
				filterExpression = spec.Filter.Expression
			}
			whereClause := contextlinks.PrepareFilterExpression(labels, filterExpression, spec.GroupBy)
			return "", contextlinks.PrepareLinksToTracesV5(start, end, whereClause)
		}
	}
	return "", ""
}

func (aH *APIHandler) getRuleStateHistory(w http.ResponseWriter, r *http.Request) {
	id, err := ruleIDFromRequest(r)
	if err != nil {
		RespondError(w, &model.ApiError{Typ: model.ErrorBadData, Err: err}, nil)
		return
	}

	params := model.QueryRuleStateHistory{}
	err = json.NewDecoder(r.Body).Decode(&params)
	if err != nil {
		RespondError(w, &model.ApiError{Typ: model.ErrorBadData, Err: err}, nil)
		return
	}
	if err := params.Validate(); err != nil {
		RespondError(w, &model.ApiError{Typ: model.ErrorBadData, Err: err}, nil)
		return
	}

	res, err := aH.ruleStateHistory.ReadRuleStateHistoryByRuleID(r.Context(), id.StringValue(), &params)
	if err != nil {
		RespondError(w, &model.ApiError{Typ: model.ErrorInternal, Err: err}, nil)
		return
	}

	rule, err := aH.ruleManager.GetRule(r.Context(), id)
	if err == nil {
		for idx := range res.Items {
			lbls := make(map[string]string)
			err := json.Unmarshal([]byte(res.Items[idx].Labels), &lbls)
			if err != nil {
				continue
			}
			end := time.Unix(res.Items[idx].UnixMilli/1000, 0)
			// Alert evaluation includes a built-in delay, so widen the link range by three minutes.
			start := end.Add(-rule.EvalWindow.Duration() - 3*time.Minute)
			res.Items[idx].RelatedLogsLink, res.Items[idx].RelatedTracesLink = relatedRuleLinks(
				rule,
				start,
				end,
				lbls,
			)
		}
	}

	aH.Respond(w, res)
}

func (aH *APIHandler) getRuleStateHistoryTopContributors(w http.ResponseWriter, r *http.Request) {
	id, err := ruleIDFromRequest(r)
	if err != nil {
		RespondError(w, &model.ApiError{Typ: model.ErrorBadData, Err: err}, nil)
		return
	}

	params := model.QueryRuleStateHistory{}
	err = json.NewDecoder(r.Body).Decode(&params)
	if err != nil {
		RespondError(w, &model.ApiError{Typ: model.ErrorBadData, Err: err}, nil)
		return
	}

	res, err := aH.ruleStateHistory.ReadRuleStateHistoryTopContributorsByRuleID(r.Context(), id.StringValue(), &params)
	if err != nil {
		RespondError(w, &model.ApiError{Typ: model.ErrorInternal, Err: err}, nil)
		return
	}

	rule, err := aH.ruleManager.GetRule(r.Context(), id)
	if err == nil {
		for idx := range res {
			lbls := make(map[string]string)
			err := json.Unmarshal([]byte(res[idx].Labels), &lbls)
			if err != nil {
				continue
			}
			end := time.Unix(params.End/1000, 0)
			start := time.Unix(params.Start/1000, 0)
			res[idx].RelatedLogsLink, res[idx].RelatedTracesLink = relatedRuleLinks(rule, start, end, lbls)
		}
	}

	aH.Respond(w, res)
}

func (aH *APIHandler) listRules(w http.ResponseWriter, r *http.Request) {

	rules, err := aH.ruleManager.ListRuleStates(r.Context())
	if err != nil {
		RespondError(w, &model.ApiError{Typ: model.ErrorInternal, Err: err}, nil)
		return
	}

	// todo(amol): need to add sorter

	aH.Respond(w, rules)
}

func (aH *APIHandler) testRule(w http.ResponseWriter, r *http.Request) {
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
	defer r.Body.Close()
	body, err := io.ReadAll(r.Body)
	if err != nil {
		aH.logger.ErrorContext(r.Context(), "error reading request body for test rule", errors.Attr(err))
		RespondError(w, &model.ApiError{Typ: model.ErrorBadData, Err: err}, nil)
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Minute)
	defer cancel()

	alertCount, apiRrr := aH.ruleManager.TestNotification(ctx, orgID, string(body))
	if apiRrr != nil {
		RespondError(w, apiRrr, nil)
		return
	}

	response := map[string]interface{}{
		"alertCount": alertCount,
		"message":    "notification sent",
	}
	aH.Respond(w, response)
}

func (aH *APIHandler) deleteRule(w http.ResponseWriter, r *http.Request) {

	id := mux.Vars(r)["id"]

	err := aH.ruleManager.DeleteRule(r.Context(), id)

	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			RespondError(w, &model.ApiError{Typ: model.ErrorNotFound, Err: fmt.Errorf("rule not found")}, nil)
			return
		}
		RespondError(w, &model.ApiError{Typ: model.ErrorInternal, Err: err}, nil)
		return
	}

	aH.Respond(w, "rule successfully deleted")

}

// patchRule updates only requested changes in the rule
func (aH *APIHandler) patchRule(w http.ResponseWriter, r *http.Request) {
	idStr := mux.Vars(r)["id"]
	id, err := valuer.NewUUID(idStr)
	if err != nil {
		RespondError(w, &model.ApiError{Typ: model.ErrorBadData, Err: err}, nil)
		return
	}

	defer r.Body.Close()
	body, err := io.ReadAll(r.Body)
	if err != nil {
		aH.logger.ErrorContext(r.Context(), "error reading request body for patch rule", errors.Attr(err))
		RespondError(w, &model.ApiError{Typ: model.ErrorBadData, Err: err}, nil)
		return
	}

	gettableRule, err := aH.ruleManager.PatchRule(r.Context(), string(body), id)

	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			RespondError(w, &model.ApiError{Typ: model.ErrorNotFound, Err: fmt.Errorf("rule not found")}, nil)
			return
		}
		RespondError(w, &model.ApiError{Typ: model.ErrorInternal, Err: err}, nil)
		return
	}

	aH.Respond(w, gettableRule)
}

func (aH *APIHandler) editRule(w http.ResponseWriter, r *http.Request) {
	idStr := mux.Vars(r)["id"]
	id, err := valuer.NewUUID(idStr)
	if err != nil {
		RespondError(w, &model.ApiError{Typ: model.ErrorBadData, Err: err}, nil)
		return
	}

	defer r.Body.Close()
	body, err := io.ReadAll(r.Body)
	if err != nil {
		aH.logger.ErrorContext(r.Context(), "error reading request body for edit rule", errors.Attr(err))
		RespondError(w, &model.ApiError{Typ: model.ErrorBadData, Err: err}, nil)
		return
	}

	err = aH.ruleManager.EditRule(r.Context(), string(body), id)

	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			RespondError(w, &model.ApiError{Typ: model.ErrorNotFound, Err: fmt.Errorf("rule not found")}, nil)
			return
		}
		RespondError(w, &model.ApiError{Typ: model.ErrorInternal, Err: err}, nil)
		return
	}

	aH.Respond(w, "rule successfully edited")

}

func (aH *APIHandler) createRule(w http.ResponseWriter, r *http.Request) {

	defer r.Body.Close()
	body, err := io.ReadAll(r.Body)
	if err != nil {
		aH.logger.ErrorContext(r.Context(), "error reading request body for create rule", errors.Attr(err))
		RespondError(w, &model.ApiError{Typ: model.ErrorBadData, Err: err}, nil)
		return
	}

	rule, err := aH.ruleManager.CreateRule(r.Context(), string(body))
	if err != nil {
		RespondError(w, &model.ApiError{Typ: model.ErrorBadData, Err: err}, nil)
		return
	}

	aH.Respond(w, rule)

}

func (aH *APIHandler) registerEvent(w http.ResponseWriter, r *http.Request) {
	request, err := parseRegisterEventRequest(r)
	if aH.HandleError(w, err, http.StatusBadRequest) {
		return
	}
	claims, errv2 := authtypes.ClaimsFromContext(r.Context())
	if errv2 == nil {
		switch request.EventType {
		case model.TrackEvent:
			aH.Signoz.Analytics.TrackUser(r.Context(), claims.OrgID, claims.UserID, request.EventName, request.Attributes)
		}
		aH.WriteJSON(w, r, map[string]string{"data": "Event Processed Successfully"})
	} else {
		RespondError(w, &model.ApiError{Typ: model.ErrorInternal, Err: err}, nil)
	}
}

func (aH *APIHandler) getServicesTopLevelOps(w http.ResponseWriter, r *http.Request) {

	var start, end time.Time
	var services []string

	type topLevelOpsParams struct {
		Service string `json:"service"`
		Start   string `json:"start"`
		End     string `json:"end"`
	}

	var params topLevelOpsParams
	err := json.NewDecoder(r.Body).Decode(&params)
	if err != nil {
		aH.logger.ErrorContext(r.Context(), "error reading request body for get top operations", errors.Attr(err))
	}

	if params.Service != "" {
		services = []string{params.Service}
	}

	startEpoch := params.Start
	if startEpoch != "" {
		startEpochInt, err := strconv.ParseInt(startEpoch, 10, 64)
		if err != nil {
			RespondError(w, &model.ApiError{Typ: model.ErrorBadData, Err: err}, "Error reading start time")
			return
		}
		start = time.Unix(0, startEpochInt)
	}
	endEpoch := params.End
	if endEpoch != "" {
		endEpochInt, err := strconv.ParseInt(endEpoch, 10, 64)
		if err != nil {
			RespondError(w, &model.ApiError{Typ: model.ErrorBadData, Err: err}, "Error reading end time")
			return
		}
		end = time.Unix(0, endEpochInt)
	}

	result, err := aH.services.GetTopLevelOperations(r.Context(), start, end, services)
	if err != nil {
		RespondError(w, &model.ApiError{Typ: model.ErrorInternal, Err: err}, nil)
		return
	}

	aH.WriteJSON(w, r, result)
}

func (aH *APIHandler) dependencyGraph(w http.ResponseWriter, r *http.Request) {

	query, err := parseGetServicesRequest(r)
	if aH.HandleError(w, err, http.StatusBadRequest) {
		return
	}

	result, err := aH.services.GetDependencyGraph(r.Context(), query)
	if aH.HandleError(w, err, http.StatusInternalServerError) {
		return
	}

	aH.WriteJSON(w, r, result)
}

func (aH *APIHandler) GetWaterfallSpansForTraceWithMetadata(w http.ResponseWriter, r *http.Request) {
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
	traceID := mux.Vars(r)["traceId"]
	if traceID == "" {
		render.Error(w, errors.NewInvalidInputf(errors.CodeInvalidInput, "traceID is required"))
		return
	}

	req := new(model.GetWaterfallSpansForTraceWithMetadataParams)
	err = json.NewDecoder(r.Body).Decode(&req)
	if err != nil {
		RespondError(w, model.BadRequest(err), nil)
		return
	}

	result, apiErr := aH.traceDetail.GetWaterfallSpansForTraceWithMetadata(r.Context(), orgID, traceID, req)
	if apiErr != nil {
		render.Error(w, apiErr)
		return
	}

	aH.WriteJSON(w, r, result)
}

func (aH *APIHandler) GetFlamegraphSpansForTrace(w http.ResponseWriter, r *http.Request) {
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

	traceID := mux.Vars(r)["traceId"]
	if traceID == "" {
		render.Error(w, errors.NewInvalidInputf(errors.CodeInvalidInput, "traceID is required"))
		return
	}

	req := new(model.GetFlamegraphSpansForTraceParams)
	err = json.NewDecoder(r.Body).Decode(&req)
	if err != nil {
		RespondError(w, model.BadRequest(err), nil)
		return
	}

	result, apiErr := aH.traceDetail.GetFlamegraphSpansForTrace(r.Context(), orgID, traceID, req)
	if apiErr != nil {
		render.Error(w, apiErr)
		return
	}

	aH.WriteJSON(w, r, result)
}

func (aH *APIHandler) listErrors(w http.ResponseWriter, r *http.Request) {

	query, err := parseListErrorsRequest(r)
	if aH.HandleError(w, err, http.StatusBadRequest) {
		return
	}
	result, err := aH.exceptions.ListErrors(r.Context(), query)
	if err != nil {
		render.Error(w, err)
		return
	}

	aH.WriteJSON(w, r, result)
}

func (aH *APIHandler) countErrors(w http.ResponseWriter, r *http.Request) {

	query, err := parseCountErrorsRequest(r)
	if aH.HandleError(w, err, http.StatusBadRequest) {
		return
	}
	result, err := aH.exceptions.CountErrors(r.Context(), query)
	if err != nil {
		render.Error(w, err)
		return
	}

	aH.WriteJSON(w, r, result)
}

func (aH *APIHandler) getErrorFromErrorID(w http.ResponseWriter, r *http.Request) {

	query, err := parseGetErrorRequest(r)
	if aH.HandleError(w, err, http.StatusBadRequest) {
		return
	}
	result, err := aH.exceptions.GetErrorFromErrorID(r.Context(), query)
	if err != nil {
		render.Error(w, err)
		return
	}

	aH.WriteJSON(w, r, result)
}

func (aH *APIHandler) getNextPrevErrorIDs(w http.ResponseWriter, r *http.Request) {

	query, err := parseGetErrorRequest(r)
	if aH.HandleError(w, err, http.StatusBadRequest) {
		return
	}
	result, err := aH.exceptions.GetNextPrevErrorIDs(r.Context(), query)
	if err != nil {
		render.Error(w, err)
		return
	}

	aH.WriteJSON(w, r, result)
}

func (aH *APIHandler) getErrorFromGroupID(w http.ResponseWriter, r *http.Request) {

	query, err := parseGetErrorRequest(r)
	if aH.HandleError(w, err, http.StatusBadRequest) {
		return
	}
	result, err := aH.exceptions.GetErrorFromGroupID(r.Context(), query)
	if err != nil {
		render.Error(w, err)
		return
	}

	aH.WriteJSON(w, r, result)
}

func (aH *APIHandler) setTTL(w http.ResponseWriter, r *http.Request) {
	ttlParams, err := parseTTLParams(r)
	if aH.HandleError(w, err, http.StatusBadRequest) {
		return
	}

	ctx := r.Context()
	claims, err := authtypes.ClaimsFromContext(ctx)
	if err != nil {
		render.Error(w, errors.NewInternalf(errors.CodeInternal, "failed to get org id from context"))
		return
	}

	// Context is not used here as TTL is long duration DB operation
	result, err := aH.retention.SetTTL(context.Background(), claims.OrgID, ttlParams)
	if err != nil {
		render.Error(w, err)
		return
	}

	aH.WriteJSON(w, r, result)

}

func (aH *APIHandler) setCustomRetentionTTL(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	claims, errv2 := authtypes.ClaimsFromContext(ctx)
	if errv2 != nil {
		render.Error(w, errorsV2.Newf(errorsV2.TypeInternal, errorsV2.CodeInternal, "failed to get org id from context"))
		return
	}

	var params model.CustomRetentionTTLParams
	if err := json.NewDecoder(r.Body).Decode(&params); err != nil {
		render.Error(w, errorsV2.Newf(errorsV2.TypeInvalidInput, errorsV2.CodeInvalidInput, "Invalid data"))
		return
	}

	// Context is not used here as TTL is long duration DB operation
	result, apiErr := aH.retention.SetCustomRetentionTTL(context.Background(), claims.OrgID, &params)
	if apiErr != nil {
		render.Error(w, errorsV2.New(errorsV2.TypeInvalidInput, errorsV2.CodeInternal, apiErr.Error()))
		return
	}

	aH.WriteJSON(w, r, result)
}

func (aH *APIHandler) getCustomRetentionTTL(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	claims, errv2 := authtypes.ClaimsFromContext(ctx)
	if errv2 != nil {
		render.Error(w, errorsV2.New(errorsV2.TypeInternal, errorsV2.CodeInternal, "failed to get org id from context"))
		return
	}

	result, apiErr := aH.retention.GetCustomRetentionTTL(r.Context(), claims.OrgID)
	if apiErr != nil {
		render.Error(w, errorsV2.New(errorsV2.TypeInvalidInput, errorsV2.CodeInternal, apiErr.Error()))
		return
	}

	aH.WriteJSON(w, r, result)
}

func (aH *APIHandler) getTTL(w http.ResponseWriter, r *http.Request) {
	ttlParams, err := parseGetTTL(r)
	if aH.HandleError(w, err, http.StatusBadRequest) {
		return
	}

	ctx := r.Context()
	claims, err := authtypes.ClaimsFromContext(ctx)
	if err != nil {
		render.Error(w, err)
		return
	}
	result, err := aH.retention.GetTTL(r.Context(), claims.OrgID, ttlParams)
	if err != nil {
		render.Error(w, err)
		return
	}
	aH.WriteJSON(w, r, result)
}

func (aH *APIHandler) getDisks(w http.ResponseWriter, r *http.Request) {
	result, err := aH.retention.GetDisks(context.Background())
	if err != nil && aH.HandleError(w, err, http.StatusInternalServerError) {
		return
	}

	aH.WriteJSON(w, r, result)
}

func (aH *APIHandler) getVersion(w http.ResponseWriter, r *http.Request) {
	versionResponse := model.GetVersionResponse{
		Version:        version.Info.Version(),
		EE:             "Y",
		SetupCompleted: aH.SetupCompleted,
	}

	aH.WriteJSON(w, r, versionResponse)
}

func (aH *APIHandler) getFeatureFlags(w http.ResponseWriter, r *http.Request) {
	claims, err := authtypes.ClaimsFromContext(r.Context())
	if err != nil {
		aH.HandleError(w, err, http.StatusInternalServerError)
		return
	}

	orgID := valuer.MustNewUUID(claims.OrgID)

	evalCtx := featuretypes.NewFlaggerEvaluationContext(orgID)
	useSpanMetrics := aH.Signoz.Flagger.BooleanOrEmpty(r.Context(), flagger.FeatureUseSpanMetrics, evalCtx)

	aH.Respond(w, uiFeatureFlags(
		constants.IsDotMetricsEnabled,
		useSpanMetrics,
	))
}

func uiFeatureFlags(dotMetricsEnabled, useSpanMetrics bool) []*licensetypes.Feature {
	return []*licensetypes.Feature{
		{
			Name:       licensetypes.DotMetricsEnabled,
			Active:     dotMetricsEnabled,
			Usage:      0,
			UsageLimit: -1,
			Route:      "",
		},
		{
			Name:       valuer.NewString(flagger.FeatureUseSpanMetrics.String()),
			Active:     useSpanMetrics,
			Usage:      0,
			UsageLimit: -1,
			Route:      "",
		},
	}
}

// getHealth is used to check the health of the service.
// 'live' query param can be used to check liveliness of
// the service by checking the database connection.
func (aH *APIHandler) getHealth(w http.ResponseWriter, r *http.Request) {
	_, ok := r.URL.Query()["live"]
	if ok {
		err := aH.clickHouseHealth.CheckClickHouse(r.Context())
		if err != nil {
			RespondError(w, &model.ApiError{Err: err, Typ: model.ErrorStatusServiceUnavailable}, nil)
			return
		}
	}

	aH.WriteJSON(w, r, map[string]string{"status": "ok"})
}

func (aH *APIHandler) registerUser(w http.ResponseWriter, r *http.Request) {
	if aH.SetupCompleted {
		render.Error(w, errors.NewInvalidInputf(errors.CodeInvalidInput, "self-registration is disabled"))
		return
	}

	var req types.PostableRegisterOrgAndAdmin
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		render.Error(w, err)
		return
	}

	organization := types.NewOrganization(req.OrgDisplayName, req.OrgName)
	user, errv2 := aH.Signoz.Modules.UserSetter.CreateFirstUser(r.Context(), organization, req.Name, req.Email, req.Password)
	if errv2 != nil {
		render.Error(w, errv2)
		return
	}

	// since the first user is now created, we can disable self-registration as
	// from here onwards, we expect admin (owner) to invite other users.
	aH.SetupCompleted = true

	aH.Respond(w, user)
}

func (aH *APIHandler) HandleError(w http.ResponseWriter, err error, statusCode int) bool {
	if err == nil {
		return false
	}
	if statusCode == http.StatusInternalServerError {
		aH.logger.Error("internal server error in http handler", errors.Attr(err))
	}
	structuredResp := structuredResponse{
		Errors: []structuredError{
			{
				Code: statusCode,
				Msg:  err.Error(),
			},
		},
	}
	resp, _ := json.Marshal(&structuredResp)
	http.Error(w, string(resp), statusCode)
	return true
}

func (aH *APIHandler) WriteJSON(w http.ResponseWriter, r *http.Request, response interface{}) {
	marshall := json.Marshal
	if prettyPrint := r.FormValue("pretty"); prettyPrint != "" && prettyPrint != "false" {
		marshall = func(v interface{}) ([]byte, error) {
			return json.MarshalIndent(v, "", "    ")
		}
	}
	resp, _ := marshall(response)
	w.Header().Set("Content-Type", "application/json")
	w.Write(resp)
}

func (aH *APIHandler) getMetricMetadata(w http.ResponseWriter, r *http.Request) {
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

	metricName := r.URL.Query().Get("metricName")
	serviceName := r.URL.Query().Get("serviceName")
	metricMetadata, err := aH.metricMetadata.GetMetricMetadata(r.Context(), orgID, metricName, serviceName)
	if err != nil {
		RespondError(w, &model.ApiError{Err: err, Typ: model.ErrorInternal}, nil)
		return
	}

	aH.WriteJSON(w, r, metricMetadata)
}

func (aH *APIHandler) respondTraceFunnelQuery(w http.ResponseWriter, r *http.Request, query string) {
	results, err := aH.traceFunnelQuery.ExecuteTraceFunnelQuery(r.Context(), query)
	if err != nil {
		RespondError(w, &model.ApiError{Typ: model.ErrorInternal, Err: fmt.Errorf("error converting clickhouse results to list: %v", err)}, nil)
		return
	}
	aH.Respond(w, results)
}

// RegisterTraceFunnelsRoutes adds trace funnels routes
func (aH *APIHandler) RegisterTraceFunnelsRoutes(router *mux.Router, am *middleware.AuthZ) {
	// Main trace funnels router
	traceFunnelsRouter := router.PathPrefix("/api/v5/trace-funnels").Subrouter()

	// API endpoints
	traceFunnelsRouter.HandleFunc("/new",
		am.EditAccess(aH.Signoz.Handlers.TraceFunnel.New)).
		Methods(http.MethodPost)
	traceFunnelsRouter.HandleFunc("/list",
		am.ViewAccess(aH.Signoz.Handlers.TraceFunnel.List)).
		Methods(http.MethodGet)
	traceFunnelsRouter.HandleFunc("/steps/update",
		am.EditAccess(aH.Signoz.Handlers.TraceFunnel.UpdateSteps)).
		Methods(http.MethodPut)

	traceFunnelsRouter.HandleFunc("/{funnel_id}",
		am.ViewAccess(aH.Signoz.Handlers.TraceFunnel.Get)).
		Methods(http.MethodGet)
	traceFunnelsRouter.HandleFunc("/{funnel_id}",
		am.EditAccess(aH.Signoz.Handlers.TraceFunnel.Delete)).
		Methods(http.MethodDelete)
	traceFunnelsRouter.HandleFunc("/{funnel_id}",
		am.EditAccess(aH.Signoz.Handlers.TraceFunnel.UpdateFunnel)).
		Methods(http.MethodPut)

	// Analytics endpoints
	traceFunnelsRouter.HandleFunc("/{funnel_id}/analytics/validate", aH.handleValidateTraces).Methods("POST")
	traceFunnelsRouter.HandleFunc("/{funnel_id}/analytics/overview", aH.handleFunnelAnalytics).Methods("POST")
	traceFunnelsRouter.HandleFunc("/{funnel_id}/analytics/steps", aH.handleStepAnalytics).Methods("POST")
	traceFunnelsRouter.HandleFunc("/{funnel_id}/analytics/steps/overview", aH.handleFunnelStepAnalytics).Methods("POST")
	traceFunnelsRouter.HandleFunc("/{funnel_id}/analytics/slow-traces", aH.handleFunnelSlowTraces).Methods("POST")
	traceFunnelsRouter.HandleFunc("/{funnel_id}/analytics/error-traces", aH.handleFunnelErrorTraces).Methods("POST")

	// Analytics endpoints
	traceFunnelsRouter.HandleFunc("/analytics/validate", aH.handleValidateTracesWithPayload).Methods("POST")
	traceFunnelsRouter.HandleFunc("/analytics/overview", aH.handleFunnelAnalyticsWithPayload).Methods("POST")
	traceFunnelsRouter.HandleFunc("/analytics/steps", aH.handleStepAnalyticsWithPayload).Methods("POST")
	traceFunnelsRouter.HandleFunc("/analytics/steps/overview", aH.handleFunnelStepAnalyticsWithPayload).Methods("POST")
	traceFunnelsRouter.HandleFunc("/analytics/slow-traces", aH.handleFunnelSlowTracesWithPayload).Methods("POST")
	traceFunnelsRouter.HandleFunc("/analytics/error-traces", aH.handleFunnelErrorTracesWithPayload).Methods("POST")
}

func (aH *APIHandler) handleValidateTraces(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	funnelID := vars["funnel_id"]

	claims, err := authtypes.ClaimsFromContext(r.Context())

	if err != nil {
		render.Error(w, err)
		return
	}

	funnel, err := aH.Signoz.Modules.TraceFunnel.Get(r.Context(), valuer.MustNewUUID(funnelID), valuer.MustNewUUID(claims.OrgID))
	if err != nil {
		RespondError(w, &model.ApiError{Typ: model.ErrorNotFound, Err: fmt.Errorf("funnel not found: %v", err)}, nil)
		return
	}

	var timeRange traceFunnels.TimeRange
	if err := json.NewDecoder(r.Body).Decode(&timeRange); err != nil {
		RespondError(w, &model.ApiError{Typ: model.ErrorBadData, Err: fmt.Errorf("error decoding time range: %v", err)}, nil)
		return
	}

	if len(funnel.Steps) < 2 {
		RespondError(w, &model.ApiError{Typ: model.ErrorBadData, Err: fmt.Errorf("funnel must have at least 2 steps")}, nil)
		return
	}

	chq, err := traceFunnelsModule.ValidateTraces(funnel, timeRange)
	if err != nil {
		RespondError(w, &model.ApiError{Typ: model.ErrorInternal, Err: fmt.Errorf("error building clickhouse query: %v", err)}, nil)
		return
	}

	aH.respondTraceFunnelQuery(w, r, chq.Query)
}

func (aH *APIHandler) handleFunnelAnalytics(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	funnelID := vars["funnel_id"]

	claims, err := authtypes.ClaimsFromContext(r.Context())

	if err != nil {
		render.Error(w, err)
		return
	}

	funnel, err := aH.Signoz.Modules.TraceFunnel.Get(r.Context(), valuer.MustNewUUID(funnelID), valuer.MustNewUUID(claims.OrgID))
	if err != nil {
		RespondError(w, &model.ApiError{Typ: model.ErrorNotFound, Err: fmt.Errorf("funnel not found: %v", err)}, nil)
		return
	}

	var stepTransition traceFunnels.StepTransitionRequest
	if err := json.NewDecoder(r.Body).Decode(&stepTransition); err != nil {
		RespondError(w, &model.ApiError{Typ: model.ErrorBadData, Err: fmt.Errorf("error decoding time range: %v", err)}, nil)
		return
	}

	chq, err := traceFunnelsModule.GetFunnelAnalytics(funnel, stepTransition.TimeRange)
	if err != nil {
		RespondError(w, &model.ApiError{Typ: model.ErrorInternal, Err: fmt.Errorf("error building clickhouse query: %v", err)}, nil)
		return
	}

	aH.respondTraceFunnelQuery(w, r, chq.Query)
}

func (aH *APIHandler) handleFunnelStepAnalytics(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	funnelID := vars["funnel_id"]

	claims, err := authtypes.ClaimsFromContext(r.Context())

	if err != nil {
		render.Error(w, err)
		return
	}

	funnel, err := aH.Signoz.Modules.TraceFunnel.Get(r.Context(), valuer.MustNewUUID(funnelID), valuer.MustNewUUID(claims.OrgID))
	if err != nil {
		RespondError(w, &model.ApiError{Typ: model.ErrorNotFound, Err: fmt.Errorf("funnel not found: %v", err)}, nil)
		return
	}

	var stepTransition traceFunnels.StepTransitionRequest
	if err := json.NewDecoder(r.Body).Decode(&stepTransition); err != nil {
		RespondError(w, &model.ApiError{Typ: model.ErrorBadData, Err: fmt.Errorf("error decoding time range: %v", err)}, nil)
		return
	}

	chq, err := traceFunnelsModule.GetFunnelStepAnalytics(funnel, stepTransition.TimeRange, stepTransition.StepStart, stepTransition.StepEnd)
	if err != nil {
		RespondError(w, &model.ApiError{Typ: model.ErrorInternal, Err: fmt.Errorf("error building clickhouse query: %v", err)}, nil)
		return
	}

	aH.respondTraceFunnelQuery(w, r, chq.Query)
}

func (aH *APIHandler) handleStepAnalytics(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	funnelID := vars["funnel_id"]

	claims, err := authtypes.ClaimsFromContext(r.Context())

	if err != nil {
		render.Error(w, err)
		return
	}

	funnel, err := aH.Signoz.Modules.TraceFunnel.Get(r.Context(), valuer.MustNewUUID(funnelID), valuer.MustNewUUID(claims.OrgID))
	if err != nil {
		RespondError(w, &model.ApiError{Typ: model.ErrorNotFound, Err: fmt.Errorf("funnel not found: %v", err)}, nil)
		return
	}

	var timeRange traceFunnels.TimeRange
	if err := json.NewDecoder(r.Body).Decode(&timeRange); err != nil {
		RespondError(w, &model.ApiError{Typ: model.ErrorBadData, Err: fmt.Errorf("error decoding time range: %v", err)}, nil)
		return
	}

	chq, err := traceFunnelsModule.GetStepAnalytics(funnel, timeRange)
	if err != nil {
		RespondError(w, &model.ApiError{Typ: model.ErrorInternal, Err: fmt.Errorf("error building clickhouse query: %v", err)}, nil)
		return
	}

	aH.respondTraceFunnelQuery(w, r, chq.Query)
}

func (aH *APIHandler) handleFunnelSlowTraces(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	funnelID := vars["funnel_id"]

	claims, err := authtypes.ClaimsFromContext(r.Context())

	if err != nil {
		render.Error(w, err)
		return
	}

	funnel, err := aH.Signoz.Modules.TraceFunnel.Get(r.Context(), valuer.MustNewUUID(funnelID), valuer.MustNewUUID(claims.OrgID))
	if err != nil {
		RespondError(w, &model.ApiError{Typ: model.ErrorNotFound, Err: fmt.Errorf("funnel not found: %v", err)}, nil)
		return
	}

	var req traceFunnels.StepTransitionRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		RespondError(w, &model.ApiError{Typ: model.ErrorBadData, Err: fmt.Errorf("invalid request body: %v", err)}, nil)
		return
	}

	chq, err := traceFunnelsModule.GetSlowestTraces(funnel, req.TimeRange, req.StepStart, req.StepEnd)
	if err != nil {
		RespondError(w, &model.ApiError{Typ: model.ErrorInternal, Err: fmt.Errorf("error building clickhouse query: %v", err)}, nil)
		return
	}

	aH.respondTraceFunnelQuery(w, r, chq.Query)
}

func (aH *APIHandler) handleFunnelErrorTraces(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	funnelID := vars["funnel_id"]

	claims, err := authtypes.ClaimsFromContext(r.Context())

	if err != nil {
		render.Error(w, err)
		return
	}

	funnel, err := aH.Signoz.Modules.TraceFunnel.Get(r.Context(), valuer.MustNewUUID(funnelID), valuer.MustNewUUID(claims.OrgID))
	if err != nil {
		RespondError(w, &model.ApiError{Typ: model.ErrorNotFound, Err: fmt.Errorf("funnel not found: %v", err)}, nil)
		return
	}

	var req traceFunnels.StepTransitionRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		RespondError(w, &model.ApiError{Typ: model.ErrorBadData, Err: fmt.Errorf("invalid request body: %v", err)}, nil)
		return
	}

	chq, err := traceFunnelsModule.GetErroredTraces(funnel, req.TimeRange, req.StepStart, req.StepEnd)
	if err != nil {
		RespondError(w, &model.ApiError{Typ: model.ErrorInternal, Err: fmt.Errorf("error building clickhouse query: %v", err)}, nil)
		return
	}

	aH.respondTraceFunnelQuery(w, r, chq.Query)
}

func (aH *APIHandler) handleValidateTracesWithPayload(w http.ResponseWriter, r *http.Request) {
	var req traceFunnels.PostableFunnel
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		RespondError(w, &model.ApiError{Typ: model.ErrorBadData, Err: fmt.Errorf("error decoding request: %v", err)}, nil)
		return
	}

	if len(req.Steps) < 2 {
		RespondError(w, &model.ApiError{Typ: model.ErrorBadData, Err: fmt.Errorf("funnel must have at least 2 steps")}, nil)
		return
	}

	// Create a StorableFunnel from the request
	funnel := &traceFunnels.StorableFunnel{
		Steps: req.Steps,
	}

	chq, err := traceFunnelsModule.ValidateTraces(funnel, traceFunnels.TimeRange{
		StartTime: req.StartTime,
		EndTime:   req.EndTime,
	})
	if err != nil {
		RespondError(w, &model.ApiError{Typ: model.ErrorInternal, Err: fmt.Errorf("error building clickhouse query: %v", err)}, nil)
		return
	}

	aH.respondTraceFunnelQuery(w, r, chq.Query)
}

func (aH *APIHandler) handleFunnelAnalyticsWithPayload(w http.ResponseWriter, r *http.Request) {
	var req traceFunnels.PostableFunnel
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		RespondError(w, &model.ApiError{Typ: model.ErrorBadData, Err: fmt.Errorf("error decoding request: %v", err)}, nil)
		return
	}

	funnel := &traceFunnels.StorableFunnel{
		Steps: req.Steps,
	}

	chq, err := traceFunnelsModule.GetFunnelAnalytics(funnel, traceFunnels.TimeRange{
		StartTime: req.StartTime,
		EndTime:   req.EndTime,
	})
	if err != nil {
		RespondError(w, &model.ApiError{Typ: model.ErrorInternal, Err: fmt.Errorf("error building clickhouse query: %v", err)}, nil)
		return
	}

	aH.respondTraceFunnelQuery(w, r, chq.Query)
}

func (aH *APIHandler) handleStepAnalyticsWithPayload(w http.ResponseWriter, r *http.Request) {
	var req traceFunnels.PostableFunnel
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		RespondError(w, &model.ApiError{Typ: model.ErrorBadData, Err: fmt.Errorf("error decoding request: %v", err)}, nil)
		return
	}

	funnel := &traceFunnels.StorableFunnel{
		Steps: req.Steps,
	}

	chq, err := traceFunnelsModule.GetStepAnalytics(funnel, traceFunnels.TimeRange{
		StartTime: req.StartTime,
		EndTime:   req.EndTime,
	})
	if err != nil {
		RespondError(w, &model.ApiError{Typ: model.ErrorInternal, Err: fmt.Errorf("error building clickhouse query: %v", err)}, nil)
		return
	}

	aH.respondTraceFunnelQuery(w, r, chq.Query)
}

func (aH *APIHandler) handleFunnelStepAnalyticsWithPayload(w http.ResponseWriter, r *http.Request) {
	var req traceFunnels.PostableFunnel
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		RespondError(w, &model.ApiError{Typ: model.ErrorBadData, Err: fmt.Errorf("error decoding request: %v", err)}, nil)
		return
	}

	funnel := &traceFunnels.StorableFunnel{
		Steps: req.Steps,
	}

	chq, err := traceFunnelsModule.GetFunnelStepAnalytics(funnel, traceFunnels.TimeRange{
		StartTime: req.StartTime,
		EndTime:   req.EndTime,
	}, req.StepStart, req.StepEnd)
	if err != nil {
		RespondError(w, &model.ApiError{Typ: model.ErrorInternal, Err: fmt.Errorf("error building clickhouse query: %v", err)}, nil)
		return
	}

	aH.respondTraceFunnelQuery(w, r, chq.Query)
}

func (aH *APIHandler) handleFunnelSlowTracesWithPayload(w http.ResponseWriter, r *http.Request) {
	var req traceFunnels.PostableFunnel
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		RespondError(w, &model.ApiError{Typ: model.ErrorBadData, Err: fmt.Errorf("error decoding request: %v", err)}, nil)
		return
	}

	funnel := &traceFunnels.StorableFunnel{
		Steps: req.Steps,
	}

	chq, err := traceFunnelsModule.GetSlowestTraces(funnel, traceFunnels.TimeRange{
		StartTime: req.StartTime,
		EndTime:   req.EndTime,
	}, req.StepStart, req.StepEnd)
	if err != nil {
		RespondError(w, &model.ApiError{Typ: model.ErrorInternal, Err: fmt.Errorf("error building clickhouse query: %v", err)}, nil)
		return
	}

	aH.respondTraceFunnelQuery(w, r, chq.Query)
}

func (aH *APIHandler) handleFunnelErrorTracesWithPayload(w http.ResponseWriter, r *http.Request) {
	var req traceFunnels.PostableFunnel
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		RespondError(w, &model.ApiError{Typ: model.ErrorBadData, Err: fmt.Errorf("error decoding request: %v", err)}, nil)
		return
	}

	funnel := &traceFunnels.StorableFunnel{
		Steps: req.Steps,
	}

	chq, err := traceFunnelsModule.GetErroredTraces(funnel, traceFunnels.TimeRange{
		StartTime: req.StartTime,
		EndTime:   req.EndTime,
	}, req.StepStart, req.StepEnd)
	if err != nil {
		RespondError(w, &model.ApiError{Typ: model.ErrorInternal, Err: fmt.Errorf("error building clickhouse query: %v", err)}, nil)
		return
	}

	aH.respondTraceFunnelQuery(w, r, chq.Query)
}
