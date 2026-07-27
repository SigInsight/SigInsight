package app

import (
	"encoding/json"
	"errors"
	"fmt"

	"net/http"
	"strconv"
	"time"

	"github.com/SigNoz/signoz/pkg/types/thirdpartyapitypes"

	errorsV2 "github.com/SigNoz/signoz/pkg/errors"
	baseconstants "github.com/SigNoz/signoz/pkg/query-service/constants"
	"github.com/SigNoz/signoz/pkg/query-service/model"
)

var allowedFunctions = []string{"count", "ratePerSec", "sum", "avg", "min", "max", "p50", "p90", "p95", "p99"}

func parseRegisterEventRequest(r *http.Request) (*model.RegisterEventParams, error) {
	var postData *model.RegisterEventParams
	err := json.NewDecoder(r.Body).Decode(&postData)
	if err != nil {
		return nil, err
	}
	// Validate the event type
	if !postData.EventType.IsValid() {
		return nil, errors.New("eventType param missing/incorrect in query")
	}

	if postData.EventType == model.TrackEvent && postData.EventName == "" {
		return nil, errors.New("eventName param missing in query")
	}

	return postData, nil
}

func parseGetServicesRequest(r *http.Request) (*model.GetServicesParams, error) {

	var postData *model.GetServicesParams
	err := json.NewDecoder(r.Body).Decode(&postData)

	if err != nil {
		return nil, err
	}

	postData.Start, err = parseTimeStr(postData.StartTime, "start")
	if err != nil {
		return nil, err
	}
	postData.End, err = parseTimeMinusBufferStr(postData.EndTime, "end")
	if err != nil {
		return nil, err
	}

	postData.Period = int(postData.End.Unix() - postData.Start.Unix())
	return postData, nil
}

func DoesExistInSlice(item string, list []string) bool {
	for _, element := range list {
		if item == element {
			return true
		}
	}
	return false
}
func parseListErrorsRequest(r *http.Request) (*model.ListErrorsParams, error) {

	var allowedOrderParams = []string{"exceptionType", "exceptionCount", "firstSeen", "lastSeen", "serviceName"}
	var allowedOrderDirections = []string{"ascending", "descending"}

	var postData *model.ListErrorsParams
	err := json.NewDecoder(r.Body).Decode(&postData)

	if err != nil {
		return nil, err
	}

	postData.Start, err = parseTimeStr(postData.StartStr, "start")
	if err != nil {
		return nil, err
	}
	postData.End, err = parseTimeMinusBufferStr(postData.EndStr, "end")
	if err != nil {
		return nil, err
	}
	if postData.Limit == 0 {
		return nil, fmt.Errorf("limit param cannot be empty from the query")
	}

	if len(postData.Order) > 0 && !DoesExistInSlice(postData.Order, allowedOrderDirections) {
		return nil, fmt.Errorf("given order: %s is not allowed in query", postData.Order)
	}

	if len(postData.Order) > 0 && !DoesExistInSlice(postData.OrderParam, allowedOrderParams) {
		return nil, fmt.Errorf("given orderParam: %s is not allowed in query", postData.OrderParam)
	}

	return postData, nil
}

func parseCountErrorsRequest(r *http.Request) (*model.CountErrorsParams, error) {

	var postData *model.CountErrorsParams
	err := json.NewDecoder(r.Body).Decode(&postData)

	if err != nil {
		return nil, err
	}

	postData.Start, err = parseTimeStr(postData.StartStr, "start")
	if err != nil {
		return nil, err
	}
	postData.End, err = parseTimeMinusBufferStr(postData.EndStr, "end")
	if err != nil {
		return nil, err
	}
	return postData, nil
}

func parseGetErrorRequest(r *http.Request) (*model.GetErrorParams, error) {

	timestamp, err := parseTime("timestamp", r)
	if err != nil {
		return nil, err
	}

	groupID := r.URL.Query().Get("groupID")

	if len(groupID) == 0 {
		return nil, fmt.Errorf("groupID param cannot be empty from the query")
	}
	errorID := r.URL.Query().Get("errorID")

	params := &model.GetErrorParams{
		Timestamp: timestamp,
		GroupID:   groupID,
		ErrorID:   errorID,
	}

	return params, nil
}

func parseTimeStr(timeStr string, param string) (*time.Time, error) {

	if len(timeStr) == 0 {
		return nil, fmt.Errorf("%s param missing in query", param)
	}

	timeUnix, err := strconv.ParseInt(timeStr, 10, 64)
	if err != nil || len(timeStr) == 0 {
		return nil, fmt.Errorf("%s param is not in correct timestamp format", param)
	}

	timeFmt := time.Unix(0, timeUnix)

	return &timeFmt, nil

}

func parseTimeMinusBufferStr(timeStr string, param string) (*time.Time, error) {

	if len(timeStr) == 0 {
		return nil, fmt.Errorf("%s param missing in query", param)
	}

	timeUnix, err := strconv.ParseInt(timeStr, 10, 64)
	if err != nil || len(timeStr) == 0 {
		return nil, fmt.Errorf("%s param is not in correct timestamp format", param)
	}

	timeUnixNow := time.Now().UnixNano()
	if timeUnix > timeUnixNow-30000000000 {
		timeUnix = timeUnix - 30000000000
	}

	timeFmt := time.Unix(0, timeUnix)

	return &timeFmt, nil

}

func parseTime(param string, r *http.Request) (*time.Time, error) {

	timeStr := r.URL.Query().Get(param)
	if len(timeStr) == 0 {
		return nil, fmt.Errorf("%s param missing in query", param)
	}

	timeUnix, err := strconv.ParseInt(timeStr, 10, 64)
	if err != nil || len(timeStr) == 0 {
		return nil, fmt.Errorf("%s param is not in correct timestamp format", param)
	}

	timeFmt := time.Unix(0, timeUnix)

	return &timeFmt, nil

}

func parseTTLParams(r *http.Request) (*model.TTLParams, error) {

	// make sure either of the query params are present
	typeTTL := r.URL.Query().Get("type")
	delDuration := r.URL.Query().Get("duration")
	coldStorage := r.URL.Query().Get("coldStorage")
	toColdDuration := r.URL.Query().Get("toColdDuration")

	if len(typeTTL) == 0 || len(delDuration) == 0 {
		return nil, fmt.Errorf("type and duration param cannot be empty from the query")
	}

	// Validate the type parameter
	if typeTTL != baseconstants.TraceTTL && typeTTL != baseconstants.MetricsTTL {
		return nil, fmt.Errorf("type param should be metrics|traces, got %v", typeTTL)
	}

	// Validate the TTL duration.
	durationParsed, err := time.ParseDuration(delDuration)
	if err != nil || durationParsed.Seconds() <= 0 {
		return nil, fmt.Errorf("not a valid TTL duration %v", delDuration)
	}

	var toColdParsed time.Duration

	// If some cold storage is provided, validate the cold storage move TTL.
	if len(coldStorage) > 0 {
		toColdParsed, err = time.ParseDuration(toColdDuration)
		if err != nil || toColdParsed.Seconds() <= 0 {
			return nil, fmt.Errorf("not a valid toCold TTL duration %v", toColdDuration)
		}
		if toColdParsed.Seconds() != 0 && toColdParsed.Seconds() >= durationParsed.Seconds() {
			return nil, fmt.Errorf("delete TTL should be greater than cold storage move TTL")
		}
	}

	return &model.TTLParams{
		Type:                  typeTTL,
		DelDuration:           int64(durationParsed.Seconds()),
		ColdStorageVolume:     coldStorage,
		ToColdStorageDuration: int64(toColdParsed.Seconds()),
	}, nil
}

func parseGetTTL(r *http.Request) (*model.GetTTLParams, error) {

	typeTTL := r.URL.Query().Get("type")

	if len(typeTTL) == 0 {
		return nil, fmt.Errorf("type param cannot be empty from the query")
	} else {
		// Validate the type parameter
		if typeTTL != baseconstants.TraceTTL && typeTTL != baseconstants.MetricsTTL {
			return nil, fmt.Errorf("type param should be metrics|traces, got %v", typeTTL)
		}
	}

	return &model.GetTTLParams{Type: typeTTL}, nil
}

// ParseRequestBody for third party APIs
func ParseRequestBody(r *http.Request) (*thirdpartyapitypes.ThirdPartyApiRequest, error) {
	req := new(thirdpartyapitypes.ThirdPartyApiRequest)
	if err := json.NewDecoder(r.Body).Decode(req); err != nil {
		return nil, errorsV2.Newf(errorsV2.TypeInvalidInput, errorsV2.CodeInvalidInput, "cannot parse the request body: %v", err)
	}

	if err := req.Validate(); err != nil {
		return nil, err
	}

	return req, nil
}
