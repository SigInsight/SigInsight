package app

import (
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"

	"net/http"
	"strconv"
	"time"

	"github.com/SigNoz/signoz/pkg/types/thirdpartyapitypes"

	"github.com/gorilla/mux"

	errorsV2 "github.com/SigNoz/signoz/pkg/errors"
	baseconstants "github.com/SigNoz/signoz/pkg/query-service/constants"
	"github.com/SigNoz/signoz/pkg/query-service/model"
	"github.com/SigNoz/signoz/pkg/query-service/model/querytypes"
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

func parseGetUsageRequest(r *http.Request) (*model.GetUsageParams, error) {
	startTime, err := parseTime("start", r)
	if err != nil {
		return nil, err
	}
	endTime, err := parseTime("end", r)
	if err != nil {
		return nil, err
	}

	stepStr := r.URL.Query().Get("step")
	if len(stepStr) == 0 {
		return nil, errors.New("step param missing in query")
	}
	stepInt, err := strconv.Atoi(stepStr)
	if err != nil {
		return nil, errors.New("step param is not in correct format")
	}

	serviceName := r.URL.Query().Get("service")
	stepHour := stepInt / 3600

	getUsageParams := model.GetUsageParams{
		StartTime:   startTime.Format(time.RFC3339Nano),
		EndTime:     endTime.Format(time.RFC3339Nano),
		Start:       startTime,
		End:         endTime,
		ServiceName: serviceName,
		Period:      fmt.Sprintf("PT%dH", stepHour),
		StepHour:    stepHour,
	}

	return &getUsageParams, nil

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

func ParseSearchTracesParams(r *http.Request) (*model.SearchTracesParams, error) {
	vars := mux.Vars(r)
	params := &model.SearchTracesParams{}
	params.TraceID = vars["traceId"]
	params.SpanID = r.URL.Query().Get("spanId")

	levelUpStr := r.URL.Query().Get("levelUp")
	levelDownStr := r.URL.Query().Get("levelDown")
	SpanRenderLimitStr := r.URL.Query().Get("spanRenderLimit")
	if levelUpStr == "" || levelUpStr == "null" {
		levelUpStr = "0"
	}
	if levelDownStr == "" || levelDownStr == "null" {
		levelDownStr = "0"
	}
	if SpanRenderLimitStr == "" || SpanRenderLimitStr == "null" {
		SpanRenderLimitStr = baseconstants.SpanRenderLimitStr
	}

	levelUpInt, err := strconv.Atoi(levelUpStr)
	if err != nil {
		return nil, err
	}
	levelDownInt, err := strconv.Atoi(levelDownStr)
	if err != nil {
		return nil, err
	}
	SpanRenderLimitInt, err := strconv.Atoi(SpanRenderLimitStr)
	if err != nil {
		return nil, err
	}
	MaxSpansInTraceInt, err := strconv.Atoi(baseconstants.MaxSpansInTraceStr)
	if err != nil {
		return nil, err
	}
	params.LevelUp = levelUpInt
	params.LevelDown = levelDownInt
	params.SpansRenderLimit = SpanRenderLimitInt
	params.MaxSpansInTrace = MaxSpansInTraceInt
	return params, nil
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
	if typeTTL != baseconstants.TraceTTL && typeTTL != baseconstants.MetricsTTL && typeTTL != baseconstants.LogsTTL {
		return nil, fmt.Errorf("type param should be metrics|traces|logs, got %v", typeTTL)
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
		if typeTTL != baseconstants.TraceTTL && typeTTL != baseconstants.MetricsTTL && typeTTL != baseconstants.LogsTTL {
			return nil, fmt.Errorf("type param should be metrics|traces|logs, got %v", typeTTL)
		}
	}

	return &model.GetTTLParams{Type: typeTTL}, nil
}

func parseAggregateAttributeRequest(r *http.Request) (*querytypes.AggregateAttributeRequest, error) {
	var req querytypes.AggregateAttributeRequest

	aggregateOperator := querytypes.AggregateOperator(r.URL.Query().Get("aggregateOperator"))
	dataSource := querytypes.DataSource(r.URL.Query().Get("dataSource"))
	aggregateAttribute := r.URL.Query().Get("searchText")

	limit, err := strconv.Atoi(r.URL.Query().Get("limit"))
	if err != nil {
		limit = 50
	}

	if dataSource != querytypes.DataSourceMetrics && dataSource != querytypes.DataSourceMeter {
		if err := aggregateOperator.Validate(); err != nil {
			return nil, err
		}
	}

	if err := dataSource.Validate(); err != nil {
		return nil, err
	}

	req = querytypes.AggregateAttributeRequest{
		Operator:   aggregateOperator,
		SearchText: aggregateAttribute,
		Limit:      limit,
		DataSource: dataSource,
	}
	return &req, nil
}

func parseQBFilterSuggestionsRequest(r *http.Request) (
	*querytypes.QBFilterSuggestionsRequest, *model.ApiError,
) {
	dataSource := querytypes.DataSource(r.URL.Query().Get("dataSource"))
	if err := dataSource.Validate(); err != nil {
		return nil, model.BadRequest(err)
	}

	parsePositiveIntQP := func(
		queryParam string, defaultValue uint64, maxValue uint64,
	) (uint64, *model.ApiError) {
		value := defaultValue

		qpValue := r.URL.Query().Get(queryParam)
		if len(qpValue) > 0 {
			value, err := strconv.Atoi(qpValue)

			if err != nil || value < 1 || value > int(maxValue) {
				return 0, model.BadRequest(fmt.Errorf(
					"invalid %s: %s", queryParam, qpValue,
				))
			}
		}

		return value, nil
	}

	attributesLimit, err := parsePositiveIntQP(
		"attributesLimit",
		baseconstants.DefaultFilterSuggestionsAttributesLimit,
		baseconstants.MaxFilterSuggestionsAttributesLimit,
	)
	if err != nil {
		return nil, err
	}

	examplesLimit, err := parsePositiveIntQP(
		"examplesLimit",
		baseconstants.DefaultFilterSuggestionsExamplesLimit,
		baseconstants.MaxFilterSuggestionsExamplesLimit,
	)
	if err != nil {
		return nil, err
	}

	var existingFilter *querytypes.FilterSet
	existingFilterB64 := r.URL.Query().Get("existingFilter")
	if len(existingFilterB64) > 0 {
		decodedFilterJson, err := base64.RawURLEncoding.DecodeString(existingFilterB64)
		if err != nil {
			return nil, model.BadRequest(fmt.Errorf("couldn't base64 decode existingFilter: %w", err))
		}

		existingFilter = &querytypes.FilterSet{}
		err = json.Unmarshal(decodedFilterJson, existingFilter)
		if err != nil {
			return nil, model.BadRequest(fmt.Errorf("couldn't JSON decode existingFilter: %w", err))
		}
	}

	searchText := r.URL.Query().Get("searchText")

	return &querytypes.QBFilterSuggestionsRequest{
		DataSource:      dataSource,
		SearchText:      searchText,
		ExistingFilter:  existingFilter,
		AttributesLimit: attributesLimit,
		ExamplesLimit:   examplesLimit,
	}, nil
}

func parseFilterAttributeKeyRequest(r *http.Request) (*querytypes.FilterAttributeKeyRequest, error) {
	var req querytypes.FilterAttributeKeyRequest

	dataSource := querytypes.DataSource(r.URL.Query().Get("dataSource"))
	aggregateOperator := querytypes.AggregateOperator(r.URL.Query().Get("aggregateOperator"))
	aggregateAttribute := r.URL.Query().Get("aggregateAttribute")
	limit, err := strconv.Atoi(r.URL.Query().Get("limit"))
	tagType := querytypes.TagType(r.URL.Query().Get("tagType"))

	// empty string is a valid tagType
	// i.e retrieve all attributes
	if tagType != "" {
		// what is happening here?
		// if tagType is undefined(uh oh javascript) or any invalid value, set it to empty string
		// instead of failing the request. Ideally, we should fail the request.
		// but we are not doing that to maintain backward compatibility.
		if err := tagType.Validate(); err != nil {
			// if the tagType is invalid, set it to empty string
			tagType = ""
		}
	}

	if err != nil {
		limit = 50
	}

	if err := dataSource.Validate(); err != nil {
		return nil, err
	}

	if dataSource != querytypes.DataSourceMetrics && dataSource != querytypes.DataSourceMeter {
		if err := aggregateOperator.Validate(); err != nil {
			return nil, err
		}
	}

	req = querytypes.FilterAttributeKeyRequest{
		DataSource:         dataSource,
		AggregateOperator:  aggregateOperator,
		AggregateAttribute: aggregateAttribute,
		Limit:              limit,
		SearchText:         r.URL.Query().Get("searchText"),
		TagType:            tagType,
	}
	return &req, nil
}

func parseFilterAttributeValueRequestBody(r *http.Request) (*querytypes.FilterAttributeValueRequest, error) {

	var req querytypes.FilterAttributeValueRequest

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		return nil, err
	}

	if err := req.Validate(); err != nil {
		return nil, err
	}

	// offset by two windows periods for start for better results
	req.StartTimeMillis = req.StartTimeMillis - time.Hour.Milliseconds()*6*2
	req.EndTimeMillis = req.EndTimeMillis + time.Hour.Milliseconds()*6

	return &req, nil
}

func parseFilterAttributeValueRequest(r *http.Request) (*querytypes.FilterAttributeValueRequest, error) {

	var req querytypes.FilterAttributeValueRequest

	dataSource := querytypes.DataSource(r.URL.Query().Get("dataSource"))
	aggregateOperator := querytypes.AggregateOperator(r.URL.Query().Get("aggregateOperator"))
	filterAttributeKeyDataType := querytypes.AttributeKeyDataType(r.URL.Query().Get("filterAttributeKeyDataType")) // can be empty
	aggregateAttribute := r.URL.Query().Get("aggregateAttribute")
	tagType := querytypes.TagType(r.URL.Query().Get("tagType")) // can be empty

	limit, err := strconv.Atoi(r.URL.Query().Get("limit"))
	if err != nil {
		limit = 50
	}

	if err := dataSource.Validate(); err != nil {
		return nil, err
	}

	if dataSource != querytypes.DataSourceMetrics {
		if err := aggregateOperator.Validate(); err != nil {
			return nil, err
		}
	}

	req = querytypes.FilterAttributeValueRequest{
		DataSource:                 dataSource,
		AggregateOperator:          aggregateOperator,
		AggregateAttribute:         aggregateAttribute,
		TagType:                    tagType,
		Limit:                      limit,
		SearchText:                 r.URL.Query().Get("searchText"),
		FilterAttributeKey:         r.URL.Query().Get("attributeKey"),
		FilterAttributeKeyDataType: filterAttributeKeyDataType,
	}
	return &req, nil
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
