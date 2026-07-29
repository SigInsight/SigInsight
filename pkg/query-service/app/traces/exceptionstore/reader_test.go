package exceptionstore

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"testing"
	"time"

	"github.com/ClickHouse/clickhouse-go/v2"
	"github.com/stretchr/testify/require"

	errorsV2 "github.com/SigNoz/signoz/pkg/errors"
	"github.com/SigNoz/signoz/pkg/query-service/model"
)

type selectResponse struct {
	queryContains string
	rows          []model.NextPrevErrorIDsDBResponse
}

type fakeConn struct {
	clickhouse.Conn
	t               *testing.T
	selectResponses []selectResponse
	selectCalls     int
}

type listErrorsConn struct {
	clickhouse.Conn
	err error
}

func (c listErrorsConn) Select(context.Context, any, string, ...any) error {
	return c.err
}

func TestListErrorsPreservesClickHouseError(t *testing.T) {
	expected := errors.New("ClickHouse unavailable")
	reader := New(slog.New(slog.NewTextHandler(io.Discard, nil)), listErrorsConn{err: expected}, Config{TraceDB: "traces", ErrorTable: "errors"})
	start := time.Unix(100, 0)
	end := time.Unix(200, 0)

	response, err := reader.ListErrors(context.Background(), &model.ListErrorsParams{Start: &start, End: &end})

	require.Nil(t, response)
	require.ErrorIs(t, err, expected)
	require.ErrorContains(t, err, "query exceptions")
}

func TestGetErrorFromErrorIDRejectsMissingID(t *testing.T) {
	reader := New(slog.New(slog.NewTextHandler(io.Discard, nil)), nil, Config{})

	response, err := reader.GetErrorFromErrorID(context.Background(), &model.GetErrorParams{})

	require.Nil(t, response)
	require.Error(t, err)
	require.True(t, errorsV2.Ast(err, errorsV2.TypeInvalidInput))
}

func (c *fakeConn) Select(_ context.Context, dest any, query string, _ ...any) error {
	c.t.Helper()
	require.Less(c.t, c.selectCalls, len(c.selectResponses))
	response := c.selectResponses[c.selectCalls]
	require.Contains(c.t, query, response.queryContains)
	c.selectCalls++

	rows, ok := dest.(*[]model.NextPrevErrorIDsDBResponse)
	require.True(c.t, ok)
	*rows = append(*rows, response.rows...)
	return nil
}

func TestGetNextErrorIDFallsBackToLaterTimestamp(t *testing.T) {
	now := time.Unix(100, 0)
	next := now.Add(time.Second)
	conn := &fakeConn{
		t: t,
		selectResponses: []selectResponse{
			{
				queryContains: "timestamp >= @timestamp",
				rows: []model.NextPrevErrorIDsDBResponse{
					{NextErrorID: "candidate-a", NextTimestamp: now, Timestamp: now},
					{NextErrorID: "candidate-b", NextTimestamp: now, Timestamp: now},
				},
			},
			{queryContains: "errorID > @errorID"},
			{
				queryContains: "timestamp > @timestamp",
				rows: []model.NextPrevErrorIDsDBResponse{
					{NextErrorID: "next", NextTimestamp: next},
				},
			},
		},
	}
	reader := New(slog.New(slog.NewTextHandler(io.Discard, nil)), conn, Config{TraceDB: "traces", ErrorTable: "errors"})

	errorID, timestamp, apiErr := reader.getNextErrorID(context.Background(), &model.GetErrorParams{
		GroupID:   "group",
		ErrorID:   "current",
		Timestamp: &now,
	})

	require.Nil(t, apiErr)
	require.Equal(t, "next", errorID)
	require.Equal(t, next, timestamp)
	require.Equal(t, 3, conn.selectCalls)
}

func TestGetPrevErrorIDUsesSameTimestampOrdering(t *testing.T) {
	now := time.Unix(100, 0)
	conn := &fakeConn{
		t: t,
		selectResponses: []selectResponse{
			{
				queryContains: "timestamp <= @timestamp",
				rows: []model.NextPrevErrorIDsDBResponse{
					{PrevErrorID: "candidate-a", PrevTimestamp: now, Timestamp: now},
					{PrevErrorID: "candidate-b", PrevTimestamp: now, Timestamp: now},
				},
			},
			{
				queryContains: "errorID < @errorID",
				rows: []model.NextPrevErrorIDsDBResponse{
					{PrevErrorID: "previous", PrevTimestamp: now},
				},
			},
		},
	}
	reader := New(slog.New(slog.NewTextHandler(io.Discard, nil)), conn, Config{TraceDB: "traces", ErrorTable: "errors"})

	errorID, timestamp, apiErr := reader.getPrevErrorID(context.Background(), &model.GetErrorParams{
		GroupID:   "group",
		ErrorID:   "current",
		Timestamp: &now,
	})

	require.Nil(t, apiErr)
	require.Equal(t, "previous", errorID)
	require.Equal(t, now, timestamp)
	require.Equal(t, 2, conn.selectCalls)
}
