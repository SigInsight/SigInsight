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
	err           error
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
	return response.err
}

func TestGetNextErrorIDUsesLexicographicCursor(t *testing.T) {
	now := time.Unix(100, 0)
	next := now.Add(time.Second)
	conn := &fakeConn{
		t: t,
		selectResponses: []selectResponse{
			{
				queryContains: "timestamp > @timestamp OR (timestamp = @timestamp AND errorID > @errorID)",
				rows: []model.NextPrevErrorIDsDBResponse{
					{ErrorID: "next", Timestamp: next},
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
	require.Equal(t, 1, conn.selectCalls)
}

func TestGetPrevErrorIDUsesSameTimestampOrdering(t *testing.T) {
	now := time.Unix(100, 0)
	conn := &fakeConn{
		t: t,
		selectResponses: []selectResponse{
			{
				queryContains: "timestamp < @timestamp OR (timestamp = @timestamp AND errorID < @errorID)",
				rows: []model.NextPrevErrorIDsDBResponse{
					{ErrorID: "previous", Timestamp: now},
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
	require.Equal(t, 1, conn.selectCalls)
}

func TestExceptionCursorBoundariesAndErrors(t *testing.T) {
	now := time.Unix(100, 0)

	t.Run("last item has no next cursor", func(t *testing.T) {
		conn := &fakeConn{t: t, selectResponses: []selectResponse{{queryContains: "ORDER BY timestamp ASC, errorID ASC"}}}
		reader := New(slog.New(slog.NewTextHandler(io.Discard, nil)), conn, Config{TraceDB: "traces", ErrorTable: "errors"})
		id, timestamp, err := reader.getNextErrorID(context.Background(), &model.GetErrorParams{GroupID: "group", ErrorID: "last", Timestamp: &now})
		require.NoError(t, err)
		require.Empty(t, id)
		require.True(t, timestamp.IsZero())
	})

	t.Run("clickhouse error is preserved", func(t *testing.T) {
		expected := errors.New("ClickHouse unavailable")
		conn := &fakeConn{t: t, selectResponses: []selectResponse{{queryContains: "ORDER BY timestamp DESC, errorID DESC", err: expected}}}
		reader := New(slog.New(slog.NewTextHandler(io.Discard, nil)), conn, Config{TraceDB: "traces", ErrorTable: "errors"})
		_, _, err := reader.getPrevErrorID(context.Background(), &model.GetErrorParams{GroupID: "group", ErrorID: "first", Timestamp: &now})
		require.ErrorIs(t, err, expected)
		require.ErrorContains(t, err, "query exceptions")
	})
}
