package tracefunnelstore

import (
	"context"
	"encoding/json"
	"log/slog"
	"reflect"
	"strings"
	"time"

	"github.com/ClickHouse/clickhouse-go/v2"
	"github.com/pkg/errors"

	errorsV2 "github.com/SigNoz/signoz/pkg/errors"
	chErrors "github.com/SigNoz/signoz/pkg/query-service/errors"
	"github.com/SigNoz/signoz/pkg/query-service/interfaces"
	"github.com/SigNoz/signoz/pkg/types/ctxtypes"
	"github.com/SigNoz/signoz/pkg/types/instrumentationtypes"
	"github.com/SigNoz/signoz/pkg/types/timeseriestypes"
)

type Reader struct {
	db     clickhouse.Conn
	logger *slog.Logger
}

var _ interfaces.TraceFunnelQueryReader = (*Reader)(nil)

func New(logger *slog.Logger, db clickhouse.Conn) *Reader {
	return &Reader{db: db, logger: logger}
}

// ExecuteTraceFunnelQuery runs a trace-funnel query and returns dynamically
// shaped result rows.
func (r *Reader) ExecuteTraceFunnelQuery(ctx context.Context, query string) ([]*timeseriestypes.Row, error) {
	ctx = ctxtypes.NewContextWithCommentVals(ctx, map[string]string{
		instrumentationtypes.CodeNamespace:    "clickhouse-reader",
		instrumentationtypes.CodeFunctionName: "ExecuteTraceFunnelQuery",
	})
	rows, err := r.db.Query(ctx, query)
	if err != nil {
		r.logger.Error("error while reading time series result", errorsV2.Attr(err))
		return nil, errors.New(err.Error())
	}
	defer rows.Close()

	columnTypes := rows.ColumnTypes()
	columnNames := rows.Columns()
	var rowList []*timeseriestypes.Row

	for rows.Next() {
		vars := make([]interface{}, len(columnTypes))
		for i := range columnTypes {
			vars[i] = reflect.New(columnTypes[i].ScanType()).Interface()
		}
		if err := rows.Scan(vars...); err != nil {
			return nil, err
		}

		row := map[string]interface{}{}
		var timestamp time.Time
		for idx, value := range vars {
			switch columnNames[idx] {
			case "timestamp":
				switch value := value.(type) {
				case *uint64:
					timestamp = time.Unix(0, int64(*value))
				case *time.Time:
					timestamp = *value
				}
			case "timestamp_datetime":
				timestamp = *value.(*time.Time)
			case "events":
				eventsFromDB, ok := value.(*[]string)
				if !ok {
					continue
				}
				var events []map[string]interface{}
				for _, event := range *eventsFromDB {
					var eventMap map[string]interface{}
					_ = json.Unmarshal([]byte(event), &eventMap)
					events = append(events, eventMap)
				}
				row[columnNames[idx]] = events
			default:
				row[columnNames[idx]] = value
			}
		}

		rowList = append(rowList, &timeseriestypes.Row{Timestamp: timestamp, Data: row})
	}

	return rowList, queryError(rows.Err())
}

func queryError(err error) error {
	if err == nil {
		return nil
	}
	slog.Error("error while reading result", errorsV2.Attr(err))
	if strings.Contains(err.Error(), "code: 307") {
		return chErrors.ErrResourceBytesLimitExceeded
	}
	if strings.Contains(err.Error(), "code: 159") {
		return chErrors.ErrResourceTimeLimitExceeded
	}
	return err
}
