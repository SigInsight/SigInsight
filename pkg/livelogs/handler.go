// Package livelogs owns the narrow SSE boundary for live log tailing.
// It intentionally does not construct a V5 builder request or depend on the
// generic legacy querier.
package livelogs

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/SigNoz/signoz/pkg/errors"
	"github.com/SigNoz/signoz/pkg/http/render"
	"github.com/SigNoz/signoz/pkg/litequery"
	"github.com/SigNoz/signoz/pkg/querier/liteadapter"
	"github.com/SigNoz/signoz/pkg/telemetrystore"
	"github.com/SigNoz/signoz/pkg/types/authtypes"
	"github.com/SigNoz/signoz/pkg/valuer"
)

const (
	batchSize    = 500
	pollInterval = 5 * time.Second
)

// Handler serves the existing /api/v5/logs/livetail SSE contract.
type Handler struct {
	query litequery.QueryFunc
	now   func() time.Time
	poll  time.Duration
}

func New(store telemetrystore.TelemetryStore) *Handler {
	return newHandler(func(ctx context.Context, statement string, args ...any) (litequery.Rows, error) {
		rows, err := store.ClickhouseDB().Query(ctx, statement, args...)
		if err != nil {
			return nil, err
		}
		return litequery.WrapClickHouseRows(rows), nil
	})
}

func newHandler(query litequery.QueryFunc) *Handler {
	return &Handler{query: query, now: time.Now, poll: pollInterval}
}

// Stream authenticates a live-tail request and writes JSON RawRow values as
// SSE data events. The frontend wire format remains unchanged.
func (h *Handler) Stream(rw http.ResponseWriter, req *http.Request) {
	ctx := req.Context()
	claims, err := authtypes.ClaimsFromContext(ctx)
	if err != nil {
		render.Error(rw, err)
		return
	}
	if _, err := valuer.NewUUID(claims.OrgID); err != nil {
		render.Error(rw, err)
		return
	}

	startMS, err := parseStart(req.URL.Query().Get("start"), h.now())
	if err != nil {
		render.Error(rw, errors.NewInvalidInputf(errors.CodeInvalidInput, "invalid live-tail start: %v", err))
		return
	}
	filter, err := parseFilter(req.URL.Query().Get("filter"))
	if err != nil {
		render.Error(rw, errors.NewInvalidInputf(errors.CodeInvalidInput, "invalid live-tail filter: %v", err))
		return
	}

	flusher, ok := rw.(http.Flusher)
	if !ok {
		render.Error(rw, errors.Newf(errors.TypeUnsupported, errors.CodeUnsupported, "streaming is not supported"))
		return
	}
	rw.Header().Set("Connection", "keep-alive")
	rw.Header().Set("Content-Type", "text/event-stream")
	rw.Header().Set("Cache-Control", "no-cache")
	rw.Header().Set("Access-Control-Allow-Origin", "*")
	rw.WriteHeader(http.StatusOK)
	flusher.Flush()

	state := streamState{startMS: startMS, filter: filter}
	for {
		if err := h.writeBatch(ctx, rw, flusher, &state); err != nil {
			if ctx.Err() != nil {
				return
			}
			fmt.Fprintf(rw, "event: error\ndata: %v\n\n", err.Error())
			flusher.Flush()
			return
		}
		select {
		case <-ctx.Done():
			return
		case <-time.After(h.poll):
		}
	}
}

type streamState struct {
	startMS int64
	filter  litequery.FilterNode
	after   *litequery.RawLogCursor
}

func parseStart(raw string, now time.Time) (int64, error) {
	if raw == "" {
		return now.UnixMilli(), nil
	}
	start, err := strconv.ParseInt(raw, 10, 64)
	if err != nil || start < 0 {
		return 0, fmt.Errorf("must be a non-negative millisecond timestamp")
	}
	return start, nil
}

func parseFilter(expression string) (litequery.FilterNode, error) {
	if strings.TrimSpace(expression) == "" {
		return nil, nil
	}
	return liteadapter.ParseFilter(expression, litequery.SignalLogs)
}

func (h *Handler) writeBatch(ctx context.Context, rw http.ResponseWriter, flusher http.Flusher, state *streamState) error {
	rows, next, err := h.read(ctx, state.startMS, state.filter, state.after)
	if err != nil {
		return err
	}
	for _, row := range rows {
		var encoded bytes.Buffer
		if err := json.NewEncoder(&encoded).Encode(row); err != nil {
			return err
		}
		if _, err := fmt.Fprintf(rw, "data: %v\n\n", encoded.String()); err != nil {
			return err
		}
		flusher.Flush()
	}
	if next != nil {
		state.after = next
	}
	return nil
}

type rawRow struct {
	Timestamp time.Time      `json:"timestamp"`
	Data      map[string]any `json:"data"`
}

func (h *Handler) read(ctx context.Context, startMS int64, filter litequery.FilterNode, after *litequery.RawLogCursor) ([]rawRow, *litequery.RawLogCursor, error) {
	endMS := h.now().UnixMilli() + 1
	if endMS <= startMS {
		endMS = startMS + 1
	}
	plan, err := (litequery.DefaultPlanner{}).Plan(litequery.Request{
		Range:      litequery.TimeRange{StartMS: startMS, EndMS: endMS},
		ResultType: litequery.ResultRaw,
		Queries: []litequery.Query{litequery.LogQuery{Common: litequery.CommonQuery{
			Name: "live_logs", Filter: filter, Limit: batchSize, After: after,
			Order: []litequery.Order{
				{Target: litequery.OrderByField, Field: litequery.FieldRef{Name: "timestamp", Context: litequery.FieldContextLog, Type: litequery.ValueTypeNumber}, Direction: litequery.SortAscending},
				{Target: litequery.OrderByField, Field: litequery.FieldRef{Name: "id", Context: litequery.FieldContextLog, Type: litequery.ValueTypeString}, Direction: litequery.SortAscending},
			},
		}, Aggregation: litequery.LogAggregateCount}},
	})
	if err != nil {
		return nil, nil, err
	}
	result, err := (litequery.Executor{Compiler: litequery.NewCompiler(nil), Query: h.query, Config: litequery.ExecutorConfig{MaxConcurrent: 1}}).Execute(ctx, plan)
	if err != nil {
		return nil, nil, err
	}
	if len(result.Queries) != 1 {
		return nil, nil, fmt.Errorf("live log query returned %d result sets", len(result.Queries))
	}
	rows := make([]rawRow, 0, len(result.Queries[0].Rows))
	var cursor *litequery.RawLogCursor
	for _, values := range result.Queries[0].Rows {
		row, next, err := rawRowFromValues(values)
		if err != nil {
			return nil, nil, err
		}
		rows = append(rows, row)
		cursor = next
	}
	return rows, cursor, nil
}

func rawRowFromValues(values []any) (rawRow, *litequery.RawLogCursor, error) {
	if len(values) != 6 {
		return rawRow{}, nil, fmt.Errorf("live log row has %d columns, want 6", len(values))
	}
	timestamp, err := uint64Value(values[0])
	if err != nil {
		return rawRow{}, nil, fmt.Errorf("invalid live log timestamp: %w", err)
	}
	id := stringValue(values[1])
	if id == "" || id == "<nil>" {
		return rawRow{}, nil, fmt.Errorf("live log row has no id")
	}
	return rawRow{
		Timestamp: time.Unix(0, int64(timestamp)),
		Data: map[string]any{
			"id":            values[1],
			"severity_text": values[2],
			"body":          values[3],
			"trace_id":      values[4],
			"span_id":       values[5],
		},
	}, &litequery.RawLogCursor{TimestampNS: timestamp, ID: id}, nil
}

func stringValue(value any) string {
	switch value := value.(type) {
	case string:
		return value
	case []byte:
		return string(value)
	default:
		return fmt.Sprint(value)
	}
}

func uint64Value(value any) (uint64, error) {
	switch value := value.(type) {
	case uint64:
		return value, nil
	case uint32:
		return uint64(value), nil
	case int64:
		if value >= 0 {
			return uint64(value), nil
		}
	case int:
		if value >= 0 {
			return uint64(value), nil
		}
	}
	return 0, fmt.Errorf("%T is not an unsigned timestamp", value)
}
