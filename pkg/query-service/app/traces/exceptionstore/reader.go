package exceptionstore

import (
	"context"
	"fmt"
	"log/slog"
	"strconv"
	"time"

	"github.com/ClickHouse/clickhouse-go/v2"

	errorsV2 "github.com/SigNoz/signoz/pkg/errors"
	"github.com/SigNoz/signoz/pkg/query-service/constants"
	"github.com/SigNoz/signoz/pkg/query-service/model"
	"github.com/SigNoz/signoz/pkg/types/ctxtypes"
	"github.com/SigNoz/signoz/pkg/types/instrumentationtypes"
	"github.com/SigNoz/signoz/pkg/types/telemetrytypes"
)

type Config struct {
	TraceDB    string
	ErrorTable string
}

type Reader struct {
	db         clickhouse.Conn
	logger     *slog.Logger
	traceDB    string
	errorTable string
}

func New(logger *slog.Logger, db clickhouse.Conn, config Config) *Reader {
	return &Reader{
		db:         db,
		logger:     logger,
		traceDB:    config.TraceDB,
		errorTable: config.ErrorTable,
	}
}

func (r *Reader) ListErrors(ctx context.Context, queryParams *model.ListErrorsParams) (*[]model.Error, error) {
	ctx = withTraceQueryMetadata(ctx, "ListErrors")
	var responses []model.Error

	query := "SELECT any(exceptionMessage) as exceptionMessage, count() AS exceptionCount, min(timestamp) as firstSeen, max(timestamp) as lastSeen, groupID"
	if len(queryParams.ServiceName) != 0 {
		query += ", serviceName"
	} else {
		query += ", any(serviceName) as serviceName"
	}
	if len(queryParams.ExceptionType) != 0 {
		query += ", exceptionType"
	} else {
		query += ", any(exceptionType) as exceptionType"
	}
	query += fmt.Sprintf(" FROM %s.%s WHERE timestamp >= @timestampL AND timestamp <= @timestampU", r.traceDB, r.errorTable)
	args := []interface{}{
		clickhouse.Named("timestampL", strconv.FormatInt(queryParams.Start.UnixNano(), 10)),
		clickhouse.Named("timestampU", strconv.FormatInt(queryParams.End.UnixNano(), 10)),
	}

	if len(queryParams.ServiceName) != 0 {
		query += " AND serviceName ilike @serviceName"
		args = append(args, clickhouse.Named("serviceName", "%"+queryParams.ServiceName+"%"))
	}
	if len(queryParams.ExceptionType) != 0 {
		query += " AND exceptionType ilike @exceptionType"
		args = append(args, clickhouse.Named("exceptionType", "%"+queryParams.ExceptionType+"%"))
	}

	tagQuery, tagArgs, err := buildTagQuery(createTagQueries(queryParams.Tags))
	if err != nil {
		r.logger.Error("Error in processing tags", errorsV2.Attr(err))
		return nil, err
	}
	query += tagQuery
	args = append(args, tagArgs...)

	query += " GROUP BY groupID"
	if len(queryParams.ServiceName) != 0 {
		query += ", serviceName"
	}
	if len(queryParams.ExceptionType) != 0 {
		query += ", exceptionType"
	}
	if len(queryParams.OrderParam) != 0 {
		if queryParams.Order == constants.Descending {
			query += " ORDER BY " + queryParams.OrderParam + " DESC"
		} else if queryParams.Order == constants.Ascending {
			query += " ORDER BY " + queryParams.OrderParam + " ASC"
		}
	}
	if queryParams.Limit > 0 {
		query += " LIMIT @limit"
		args = append(args, clickhouse.Named("limit", queryParams.Limit))
	}
	if queryParams.Offset > 0 {
		query += " OFFSET @offset"
		args = append(args, clickhouse.Named("offset", queryParams.Offset))
	}

	err = r.db.Select(ctx, &responses, query, args...)
	r.logger.Info(query)
	if err != nil {
		r.logger.Error("Error in processing sql query", errorsV2.Attr(err))
		return nil, executionError(err)
	}
	return &responses, nil
}

func (r *Reader) CountErrors(ctx context.Context, queryParams *model.CountErrorsParams) (uint64, error) {
	ctx = withTraceQueryMetadata(ctx, "CountErrors")
	var errorCount uint64

	query := fmt.Sprintf("SELECT count(distinct(groupID)) FROM %s.%s WHERE timestamp >= @timestampL AND timestamp <= @timestampU", r.traceDB, r.errorTable)
	args := []interface{}{
		clickhouse.Named("timestampL", strconv.FormatInt(queryParams.Start.UnixNano(), 10)),
		clickhouse.Named("timestampU", strconv.FormatInt(queryParams.End.UnixNano(), 10)),
	}
	if len(queryParams.ServiceName) != 0 {
		query += " AND serviceName ilike @serviceName"
		args = append(args, clickhouse.Named("serviceName", "%"+queryParams.ServiceName+"%"))
	}
	if len(queryParams.ExceptionType) != 0 {
		query += " AND exceptionType ilike @exceptionType"
		args = append(args, clickhouse.Named("exceptionType", "%"+queryParams.ExceptionType+"%"))
	}

	tagQuery, tagArgs, err := buildTagQuery(createTagQueries(queryParams.Tags))
	if err != nil {
		r.logger.Error("Error in processing tags", errorsV2.Attr(err))
		return 0, err
	}
	query += tagQuery
	args = append(args, tagArgs...)

	err = r.db.QueryRow(ctx, query, args...).Scan(&errorCount)
	r.logger.Info(query)
	if err != nil {
		r.logger.Error("Error in processing sql query", errorsV2.Attr(err))
		return 0, executionError(err)
	}
	return errorCount, nil
}

func (r *Reader) GetErrorFromErrorID(ctx context.Context, queryParams *model.GetErrorParams) (*model.ErrorWithSpan, error) {
	ctx = withTraceQueryMetadata(ctx, "GetErrorFromErrorID")
	if queryParams.ErrorID == "" {
		r.logger.Error("errorId missing from params")
		return nil, missingErrorIDError()
	}

	var responses []model.ErrorWithSpan
	query := fmt.Sprintf("SELECT errorID, exceptionType, exceptionStacktrace, exceptionEscaped, exceptionMessage, timestamp, spanID, traceID, serviceName, groupID FROM %s.%s WHERE timestamp = @timestamp AND groupID = @groupID AND errorID = @errorID LIMIT 1", r.traceDB, r.errorTable)
	args := []interface{}{
		clickhouse.Named("errorID", queryParams.ErrorID),
		clickhouse.Named("groupID", queryParams.GroupID),
		clickhouse.Named("timestamp", strconv.FormatInt(queryParams.Timestamp.UnixNano(), 10)),
	}

	err := r.db.Select(ctx, &responses, query, args...)
	r.logger.Info(query)
	if err != nil {
		r.logger.Error("Error in processing sql query", errorsV2.Attr(err))
		return nil, executionError(err)
	}
	if len(responses) > 0 {
		return &responses[0], nil
	}
	return nil, exceptionNotFoundError()
}

func (r *Reader) GetErrorFromGroupID(ctx context.Context, queryParams *model.GetErrorParams) (*model.ErrorWithSpan, error) {
	ctx = withTraceQueryMetadata(ctx, "GetErrorFromGroupID")
	var responses []model.ErrorWithSpan

	query := fmt.Sprintf("SELECT errorID, exceptionType, exceptionStacktrace, exceptionEscaped, exceptionMessage, timestamp, spanID, traceID, serviceName, groupID FROM %s.%s WHERE timestamp = @timestamp AND groupID = @groupID LIMIT 1", r.traceDB, r.errorTable)
	args := []interface{}{
		clickhouse.Named("groupID", queryParams.GroupID),
		clickhouse.Named("timestamp", strconv.FormatInt(queryParams.Timestamp.UnixNano(), 10)),
	}

	err := r.db.Select(ctx, &responses, query, args...)
	r.logger.Info(query)
	if err != nil {
		r.logger.Error("Error in processing sql query", errorsV2.Attr(err))
		return nil, executionError(err)
	}
	if len(responses) > 0 {
		return &responses[0], nil
	}
	return nil, exceptionNotFoundError()
}

func (r *Reader) GetNextPrevErrorIDs(ctx context.Context, queryParams *model.GetErrorParams) (*model.NextPrevErrorIDs, error) {
	if queryParams.ErrorID == "" {
		r.logger.Error("errorId missing from params")
		return nil, missingErrorIDError()
	}

	response := model.NextPrevErrorIDs{GroupID: queryParams.GroupID}
	var err error
	response.NextErrorID, response.NextTimestamp, err = r.getNextErrorID(ctx, queryParams)
	if err != nil {
		r.logger.Error("Unable to get next error ID due to err: ", errorsV2.Attr(err))
		return nil, err
	}
	response.PrevErrorID, response.PrevTimestamp, err = r.getPrevErrorID(ctx, queryParams)
	if err != nil {
		r.logger.Error("Unable to get prev error ID due to err: ", errorsV2.Attr(err))
		return nil, err
	}
	return &response, nil
}

func (r *Reader) getNextErrorID(ctx context.Context, queryParams *model.GetErrorParams) (string, time.Time, error) {
	ctx = withTraceQueryMetadata(ctx, "getNextErrorID")
	var responses []model.NextPrevErrorIDsDBResponse

	query := fmt.Sprintf("SELECT errorID as nextErrorID, timestamp as nextTimestamp FROM %s.%s WHERE groupID = @groupID AND timestamp >= @timestamp AND errorID != @errorID ORDER BY timestamp ASC LIMIT 2", r.traceDB, r.errorTable)
	args := errorPositionArgs(queryParams)
	err := r.db.Select(ctx, &responses, query, args...)
	r.logger.Info(query)
	if err != nil {
		r.logger.Error("Error in processing sql query", errorsV2.Attr(err))
		return "", time.Time{}, executionError(err)
	}
	if len(responses) == 0 {
		r.logger.Info("NextErrorID not found")
		return "", time.Time{}, nil
	}
	if len(responses) == 1 || responses[0].Timestamp.UnixNano() != responses[1].Timestamp.UnixNano() {
		r.logger.Info("NextErrorID found")
		return responses[0].NextErrorID, responses[0].NextTimestamp, nil
	}

	query = fmt.Sprintf("SELECT errorID as nextErrorID, timestamp as nextTimestamp FROM %s.%s WHERE groupID = @groupID AND timestamp = @timestamp AND errorID > @errorID ORDER BY errorID ASC LIMIT 1", r.traceDB, r.errorTable)
	responses = nil
	err = r.db.Select(ctx, &responses, query, args...)
	r.logger.Info(query)
	if err != nil {
		r.logger.Error("Error in processing sql query", errorsV2.Attr(err))
		return "", time.Time{}, executionError(err)
	}
	if len(responses) > 0 {
		r.logger.Info("NextErrorID found")
		return responses[0].NextErrorID, responses[0].NextTimestamp, nil
	}

	query = fmt.Sprintf("SELECT errorID as nextErrorID, timestamp as nextTimestamp FROM %s.%s WHERE groupID = @groupID AND timestamp > @timestamp ORDER BY timestamp ASC LIMIT 1", r.traceDB, r.errorTable)
	err = r.db.Select(ctx, &responses, query, args...)
	r.logger.Info(query)
	if err != nil {
		r.logger.Error("Error in processing sql query", errorsV2.Attr(err))
		return "", time.Time{}, executionError(err)
	}
	if len(responses) == 0 {
		r.logger.Info("NextErrorID not found")
		return "", time.Time{}, nil
	}
	r.logger.Info("NextErrorID found")
	return responses[0].NextErrorID, responses[0].NextTimestamp, nil
}

func (r *Reader) getPrevErrorID(ctx context.Context, queryParams *model.GetErrorParams) (string, time.Time, error) {
	ctx = withTraceQueryMetadata(ctx, "getPrevErrorID")
	var responses []model.NextPrevErrorIDsDBResponse

	query := fmt.Sprintf("SELECT errorID as prevErrorID, timestamp as prevTimestamp FROM %s.%s WHERE groupID = @groupID AND timestamp <= @timestamp AND errorID != @errorID ORDER BY timestamp DESC LIMIT 2", r.traceDB, r.errorTable)
	args := errorPositionArgs(queryParams)
	err := r.db.Select(ctx, &responses, query, args...)
	r.logger.Info(query)
	if err != nil {
		r.logger.Error("Error in processing sql query", errorsV2.Attr(err))
		return "", time.Time{}, executionError(err)
	}
	if len(responses) == 0 {
		r.logger.Info("PrevErrorID not found")
		return "", time.Time{}, nil
	}
	if len(responses) == 1 || responses[0].Timestamp.UnixNano() != responses[1].Timestamp.UnixNano() {
		r.logger.Info("PrevErrorID found")
		return responses[0].PrevErrorID, responses[0].PrevTimestamp, nil
	}

	query = fmt.Sprintf("SELECT errorID as prevErrorID, timestamp as prevTimestamp FROM %s.%s WHERE groupID = @groupID AND timestamp = @timestamp AND errorID < @errorID ORDER BY errorID DESC LIMIT 1", r.traceDB, r.errorTable)
	responses = nil
	err = r.db.Select(ctx, &responses, query, args...)
	r.logger.Info(query)
	if err != nil {
		r.logger.Error("Error in processing sql query", errorsV2.Attr(err))
		return "", time.Time{}, executionError(err)
	}
	if len(responses) > 0 {
		r.logger.Info("PrevErrorID found")
		return responses[0].PrevErrorID, responses[0].PrevTimestamp, nil
	}

	query = fmt.Sprintf("SELECT errorID as prevErrorID, timestamp as prevTimestamp FROM %s.%s WHERE groupID = @groupID AND timestamp < @timestamp ORDER BY timestamp DESC LIMIT 1", r.traceDB, r.errorTable)
	err = r.db.Select(ctx, &responses, query, args...)
	r.logger.Info(query)
	if err != nil {
		r.logger.Error("Error in processing sql query", errorsV2.Attr(err))
		return "", time.Time{}, executionError(err)
	}
	if len(responses) == 0 {
		r.logger.Info("PrevErrorID not found")
		return "", time.Time{}, nil
	}
	r.logger.Info("PrevErrorID found")
	return responses[0].PrevErrorID, responses[0].PrevTimestamp, nil
}

func withTraceQueryMetadata(ctx context.Context, functionName string) context.Context {
	return ctxtypes.NewContextWithCommentVals(ctx, map[string]string{
		instrumentationtypes.TelemetrySignal:  telemetrytypes.SignalTraces.StringValue(),
		instrumentationtypes.CodeNamespace:    "clickhouse-reader",
		instrumentationtypes.CodeFunctionName: functionName,
	})
}

func errorPositionArgs(queryParams *model.GetErrorParams) []interface{} {
	return []interface{}{
		clickhouse.Named("errorID", queryParams.ErrorID),
		clickhouse.Named("groupID", queryParams.GroupID),
		clickhouse.Named("timestamp", strconv.FormatInt(queryParams.Timestamp.UnixNano(), 10)),
	}
}

func executionError(cause error) error {
	return fmt.Errorf("query exceptions: %w", cause)
}

func missingErrorIDError() error {
	return errorsV2.NewInvalidInputf(errorsV2.CodeInvalidInput, "ErrorID missing from params")
}

func exceptionNotFoundError() error {
	return errorsV2.NewNotFoundf(errorsV2.CodeNotFound, "Error/Exception not found")
}
