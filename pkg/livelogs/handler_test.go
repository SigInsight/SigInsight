package livelogs

import (
	"context"
	"testing"
	"time"

	"github.com/SigNoz/signoz/pkg/errors"
	"github.com/SigNoz/signoz/pkg/litequery"
	"github.com/SigNoz/signoz/pkg/types/telemetrytypes"
	"github.com/SigNoz/signoz/pkg/types/telemetrytypes/telemetrytypestest"
	"github.com/stretchr/testify/require"
)

func TestReadBuildsBoundFilterAndChronologicalCursorQuery(t *testing.T) {
	var statement string
	var args []any
	handler := newHandler(func(_ context.Context, sql string, values ...any) (litequery.Rows, error) {
		statement, args = sql, values
		return &testRows{data: [][]any{{uint64(50_000_000), "a", "INFO", "first", "trace-a", "span-a"}, {uint64(60_000_000), "b", "ERROR", "second", "trace-b", "span-b"}}}, nil
	})
	handler.now = func() time.Time { return time.UnixMilli(100) }
	filter, err := parseFilter("resource.service.name = 'api\" OR 1=1 --'")
	require.NoError(t, err)

	rows, cursor, err := handler.read(context.Background(), 1, filter, nil)
	require.NoError(t, err)
	require.Len(t, rows, 2)
	require.Equal(t, "second", rows[1].Data["body"])
	require.Equal(t, &litequery.RawLogCursor{TimestampNS: 60_000_000, ID: "b"}, cursor)
	require.Contains(t, statement, "ORDER BY timestamp ASC, id ASC")
	require.NotContains(t, statement, "OR 1=1")
	require.Contains(t, args, `api" OR 1=1 --`)
}

func TestReadUsesStrictTimestampAndIDCursor(t *testing.T) {
	var statement string
	var args []any
	handler := newHandler(func(_ context.Context, sql string, values ...any) (litequery.Rows, error) {
		statement, args = sql, values
		return &testRows{}, nil
	})
	handler.now = func() time.Time { return time.UnixMilli(100) }

	rows, cursor, err := handler.read(context.Background(), 1, nil, &litequery.RawLogCursor{TimestampNS: 12_345, ID: "last"})
	require.NoError(t, err)
	require.Empty(t, rows)
	require.Nil(t, cursor)
	require.Contains(t, statement, "(timestamp > toUInt64(?)) OR (timestamp = toUInt64(?) AND id > ?)")
	require.Contains(t, args, uint64(12_345))
	require.Contains(t, args, "last")
}

func TestParseStartAndRawRowValidation(t *testing.T) {
	now := time.UnixMilli(100)
	start, err := parseStart("", now)
	require.NoError(t, err)
	require.Equal(t, int64(100), start)
	_, err = parseStart("-1", now)
	require.Error(t, err)
	_, _, err = rawRowFromValues([]any{uint64(1)})
	require.Error(t, err)
	_, _, err = rawRowFromValues([]any{uint64(1), "", "INFO", "body", "trace", "span"})
	require.Error(t, err)

	filter, err := parseFilter("service.name like 'api%'")
	require.NoError(t, err)
	require.Equal(t, litequery.Predicate{
		Field: litequery.FieldRef{Name: "service.name", Context: litequery.FieldContextLog, Type: litequery.ValueTypeString},
		Op:    litequery.FilterLike,
		Value: litequery.Value{Kind: litequery.ValueString, String: "api%"},
	}, filter)
}

func TestParseFilterResolvesLogFieldsFromMetadata(t *testing.T) {
	metadata := telemetrytypestest.NewMockMetadataStore()
	metadata.SetKeys([]*telemetrytypes.TelemetryFieldKey{
		{Name: "host.name", Signal: telemetrytypes.SignalLogs, FieldContext: telemetrytypes.FieldContextResource, FieldDataType: telemetrytypes.FieldDataTypeString},
	})
	handler := newHandler(func(context.Context, string, ...any) (litequery.Rows, error) { return &testRows{}, nil })
	handler.metadata = metadata

	filter, err := handler.parseFilter(context.Background(), "host.name = 'worker-1' AND response.size >= 512", 1, 100)
	require.NoError(t, err)
	logical := filter.(litequery.LogicalFilter)
	require.Equal(t, litequery.FieldRef{Name: "host.name", Context: litequery.FieldContextResource, Type: litequery.ValueTypeString}, logical.Items[0].(litequery.Predicate).Field)
	require.Equal(t, litequery.FieldRef{Name: "response.size", Context: litequery.FieldContextAttribute, Type: litequery.ValueTypeNumber}, logical.Items[1].(litequery.Predicate).Field)
}

func TestParseFilterRequiresContextForUnknownUnqualifiedField(t *testing.T) {
	metadata := telemetrytypestest.NewMockMetadataStore()
	handler := newHandler(func(context.Context, string, ...any) (litequery.Rows, error) { return &testRows{}, nil })
	handler.metadata = metadata

	filter, err := handler.parseFilter(context.Background(), "unknown.field = 'value'", 1, 100)
	require.Nil(t, filter)
	require.ErrorContains(t, err, "qualify it as resource or attribute")
}

type testRows struct {
	data  [][]any
	index int
	err   error
}

func (rows *testRows) Columns() []string {
	return []string{"field_0", "field_1", "field_2", "field_3", "field_4", "field_5"}
}
func (rows *testRows) Next() bool { return rows.index < len(rows.data) }
func (rows *testRows) Scan(destinations ...any) error {
	if rows.err != nil {
		return rows.err
	}
	if len(destinations) != len(rows.data[rows.index]) {
		return errors.New(errors.TypeInternal, errors.CodeInternal, "invalid scan destination count")
	}
	for index, value := range rows.data[rows.index] {
		target, ok := destinations[index].(*any)
		if !ok {
			return errors.New(errors.TypeInternal, errors.CodeInternal, "invalid scan destination")
		}
		*target = value
	}
	rows.index++
	return nil
}
func (rows *testRows) Err() error   { return rows.err }
func (rows *testRows) Close() error { return nil }
