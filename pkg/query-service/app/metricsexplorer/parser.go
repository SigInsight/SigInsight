package metricsexplorer

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"

	"github.com/SigNoz/signoz/pkg/query-service/constants"
	"github.com/SigNoz/signoz/pkg/query-service/model"
	"github.com/SigNoz/signoz/pkg/query-service/model/metrics_explorer"
)

func ParseFilterKeySuggestions(r *http.Request) (*metrics_explorer.FilterKeyRequest, *model.ApiError) {

	searchText := r.URL.Query().Get("searchText")
	limit, err := strconv.Atoi(r.URL.Query().Get("limit"))
	if err != nil {
		limit = 50
	}

	return &metrics_explorer.FilterKeyRequest{Limit: limit, SearchText: searchText}, nil
}

func ParseFilterValueSuggestions(r *http.Request) (*metrics_explorer.FilterValueRequest, *model.ApiError) {
	var filterValueRequest metrics_explorer.FilterValueRequest

	// parse the request body
	if err := json.NewDecoder(r.Body).Decode(&filterValueRequest); err != nil {
		return nil, &model.ApiError{Typ: model.ErrorBadData, Err: fmt.Errorf("cannot parse the request body: %v", err)}
	}

	return &filterValueRequest, nil
}

func ParseRelatedMetricsParams(r *http.Request) (*metrics_explorer.RelatedMetricsRequest, *model.ApiError) {
	var relatedMetricParams metrics_explorer.RelatedMetricsRequest
	if err := json.NewDecoder(r.Body).Decode(&relatedMetricParams); err != nil {
		return nil, &model.ApiError{Typ: model.ErrorBadData, Err: fmt.Errorf("cannot parse the request body: %v", err)}
	}
	return &relatedMetricParams, nil
}

func ParseInspectMetricsParams(r *http.Request) (*metrics_explorer.InspectMetricsRequest, *model.ApiError) {
	var inspectMetricParams metrics_explorer.InspectMetricsRequest
	if err := json.NewDecoder(r.Body).Decode(&inspectMetricParams); err != nil {
		return nil, &model.ApiError{Typ: model.ErrorBadData, Err: fmt.Errorf("cannot parse the request body: %v", err)}
	}
	if inspectMetricParams.End-inspectMetricParams.Start > constants.InspectMetricsMaxTimeDiff { // half hour only
		return nil, &model.ApiError{Typ: model.ErrorBadData, Err: fmt.Errorf("time duration shouldn't be more than 30 mins")}
	}
	return &inspectMetricParams, nil
}
