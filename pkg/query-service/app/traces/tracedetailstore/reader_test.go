package tracedetailstore

import (
	"context"
	"database/sql"
	"errors"
	"io"
	"log/slog"
	"testing"
	"time"

	"github.com/ClickHouse/clickhouse-go/v2"
	"github.com/ClickHouse/clickhouse-go/v2/lib/driver"
	"github.com/stretchr/testify/require"

	"github.com/SigNoz/signoz/pkg/query-service/model"
	"github.com/SigNoz/signoz/pkg/types/cachetypes"
	"github.com/SigNoz/signoz/pkg/valuer"
)

type traceDetailCache struct{ data map[string][]byte }

func (c *traceDetailCache) Set(_ context.Context, _ valuer.UUID, key string, value cachetypes.Cacheable, _ time.Duration) error {
	data, err := value.MarshalBinary()
	if err != nil {
		return err
	}
	c.data[key] = data
	return nil
}

func (c *traceDetailCache) Get(_ context.Context, _ valuer.UUID, key string, dest cachetypes.Cacheable) error {
	data, ok := c.data[key]
	if !ok {
		return errors.New("cache miss")
	}
	return dest.UnmarshalBinary(data)
}

func (c *traceDetailCache) Delete(_ context.Context, _ valuer.UUID, key string) {
	delete(c.data, key)
}

func (c *traceDetailCache) DeleteMany(_ context.Context, _ valuer.UUID, keys []string) {
	for _, key := range keys {
		delete(c.data, key)
	}
}

type traceDetailRow struct{ err error }

func (r traceDetailRow) Err() error           { return r.err }
func (r traceDetailRow) Scan(...any) error    { return r.err }
func (r traceDetailRow) ScanStruct(any) error { return r.err }

type traceSummaryConn struct {
	clickhouse.Conn
	queryRow func(context.Context, string, ...any) driver.Row
}

func (c traceSummaryConn) QueryRow(ctx context.Context, query string, args ...any) driver.Row {
	return c.queryRow(ctx, query, args...)
}

type traceDetailSelectConn struct {
	traceSummaryConn
	selectFn func(context.Context, any, string, ...any) error
}

func (c traceDetailSelectConn) Select(ctx context.Context, dest any, query string, args ...any) error {
	return c.selectFn(ctx, dest, query, args...)
}

func TestGetSpansForTraceReturnsEmptyWhenSummaryIsMissing(t *testing.T) {
	queries := 0
	reader := New(slog.New(slog.NewTextHandler(io.Discard, nil)), traceSummaryConn{
		queryRow: func(_ context.Context, query string, args ...any) driver.Row {
			queries++
			require.Contains(t, query, "FROM signoz_traces.trace_summaries")
			require.Equal(t, []any{"missing-trace"}, args)
			return traceDetailRow{err: sql.ErrNoRows}
		},
	}, nil, Config{TraceDB: "signoz_traces", TraceSummaryTable: "trace_summaries"})

	spans, apiErr := reader.getSpansForTrace(context.Background(), "missing-trace", "SELECT should_not_run")
	require.Nil(t, apiErr)
	require.Empty(t, spans)
	require.Equal(t, 1, queries)
}

func TestGetSpansForTraceUsesSummaryTimeWindow(t *testing.T) {
	start := time.Unix(2_000, 0)
	end := time.Unix(3_000, 0)
	reader := New(slog.New(slog.NewTextHandler(io.Discard, nil)), traceDetailSelectConn{
		traceSummaryConn: traceSummaryConn{
			queryRow: func(_ context.Context, query string, args ...any) driver.Row {
				require.Contains(t, query, "FROM signoz_traces.trace_summaries")
				require.Equal(t, []any{"trace-1"}, args)
				return traceDetailSummaryRow{traceID: "trace-1", start: start, end: end, numSpans: 1}
			},
		},
		selectFn: func(_ context.Context, dest any, query string, args ...any) error {
			require.Contains(t, query, "FROM signoz_traces.spans")
			require.Equal(t, []any{"trace-1", "200", "3000"}, args)
			*dest.(*[]model.SpanItemV2) = []model.SpanItemV2{{SpanID: "span-1", TraceID: "trace-1"}}
			return nil
		},
	}, nil, Config{TraceDB: "signoz_traces", TraceTableName: "spans", TraceSummaryTable: "trace_summaries"})

	spans, apiErr := reader.getSpansForTrace(context.Background(), "trace-1", "SELECT * FROM signoz_traces.spans")
	require.Nil(t, apiErr)
	require.Equal(t, []model.SpanItemV2{{SpanID: "span-1", TraceID: "trace-1"}}, spans)
}

func TestGetWaterfallSpansUsesStableCacheEntry(t *testing.T) {
	ctx := context.Background()
	orgID := valuer.GenerateUUID()
	cache := &traceDetailCache{data: map[string][]byte{}}
	end := time.Now().Add(-2 * time.Hour).UnixNano()
	cachedRoot := &model.Span{SpanID: "root", TraceID: "trace-1", ServiceName: "api", Name: "GET /orders", TimeUnixNano: uint64(end - int64(time.Second)), Children: []*model.Span{}}
	require.NoError(t, cache.Set(ctx, orgID, "getWaterfallSpansForTraceWithMetadata-trace-1", &model.GetWaterfallSpansForTraceWithMetadataCache{
		StartTime:                     uint64(end - int64(time.Second)),
		EndTime:                       uint64(end),
		DurationNano:                  uint64(time.Second),
		TotalSpans:                    1,
		SpanIdToSpanNodeMap:           map[string]*model.Span{"root": cachedRoot},
		ServiceNameToTotalDurationMap: map[string]uint64{"api": uint64(time.Second)},
		TraceRoots:                    []*model.Span{cachedRoot},
	}, time.Minute))

	reader := New(slog.New(slog.NewTextHandler(io.Discard, nil)), nil, cache, Config{FluxInterval: time.Minute})
	response, err := reader.GetWaterfallSpansForTraceWithMetadata(ctx, orgID, "trace-1", &model.GetWaterfallSpansForTraceWithMetadataParams{})

	require.NoError(t, err)
	require.Equal(t, uint64(1), response.TotalSpansCount)
	require.Equal(t, "api", response.RootServiceName)
	require.Equal(t, uint64(1000), response.ServiceNameToTotalDurationMap["api"])
	require.Len(t, response.Spans, 1)
}

func TestGetWaterfallSpansBuildsNestedAndMissingParentTrees(t *testing.T) {
	start := time.Unix(2_000, 0)
	end := start.Add(3 * time.Second)
	cache := &traceDetailCache{data: map[string][]byte{}}
	reader := New(slog.New(slog.NewTextHandler(io.Discard, nil)), traceDetailSelectConn{
		traceSummaryConn: traceSummaryConn{queryRow: func(_ context.Context, _ string, _ ...any) driver.Row {
			return traceDetailSummaryRow{traceID: "trace-1", start: start, end: end, numSpans: 3}
		}},
		selectFn: func(_ context.Context, dest any, _ string, _ ...any) error {
			*dest.(*[]model.SpanItemV2) = []model.SpanItemV2{
				{SpanID: "root", TraceID: "trace-1", ServiceName: "frontend", Name: "request", TimeUnixNano: start, DurationNano: uint64(3 * time.Second), References: "[]", Attributes_string: map[string]string{}, Attributes_number: map[string]float64{}, Attributes_bool: map[string]bool{}, Resources_string: map[string]string{}},
				{SpanID: "child", TraceID: "trace-1", ServiceName: "api", Name: "query", TimeUnixNano: start.Add(time.Second), DurationNano: uint64(time.Second), HasError: true, References: `[{"spanId":"root","refType":"CHILD_OF"}]`, Attributes_string: map[string]string{}, Attributes_number: map[string]float64{}, Attributes_bool: map[string]bool{}, Resources_string: map[string]string{}},
				{SpanID: "orphan", TraceID: "trace-1", ServiceName: "worker", Name: "job", TimeUnixNano: start.Add(2 * time.Second), DurationNano: uint64(time.Second), References: `[{"spanId":"absent","refType":"CHILD_OF"}]`, Attributes_string: map[string]string{}, Attributes_number: map[string]float64{}, Attributes_bool: map[string]bool{}, Resources_string: map[string]string{}},
			}
			return nil
		},
	}, cache, Config{TraceDB: "signoz_traces", TraceTableName: "spans", TraceSummaryTable: "trace_summaries", FluxInterval: time.Minute})

	response, err := reader.GetWaterfallSpansForTraceWithMetadata(context.Background(), valuer.GenerateUUID(), "trace-1", &model.GetWaterfallSpansForTraceWithMetadataParams{UncollapsedSpans: []string{"root", "absent"}})

	require.NoError(t, err)
	require.Equal(t, uint64(3), response.TotalSpansCount)
	require.Equal(t, uint64(1), response.TotalErrorSpansCount)
	require.True(t, response.HasMissingSpans)
	require.Equal(t, "frontend", response.RootServiceName)
	require.Equal(t, uint64(3000), response.ServiceNameToTotalDurationMap["frontend"])
	require.Equal(t, uint64(1000), response.ServiceNameToTotalDurationMap["api"])
	require.Contains(t, response.UncollapsedSpans, "root")
	require.Contains(t, response.UncollapsedSpans, "absent")
}

type traceDetailSummaryRow struct {
	traceID  string
	start    time.Time
	end      time.Time
	numSpans uint64
}

func (r traceDetailSummaryRow) Err() error { return nil }

func (r traceDetailSummaryRow) Scan(dest ...any) error {
	*dest[0].(*string) = r.traceID
	*dest[1].(*time.Time) = r.start
	*dest[2].(*time.Time) = r.end
	*dest[3].(*uint64) = r.numSpans
	return nil
}

func (r traceDetailSummaryRow) ScanStruct(any) error { return nil }
