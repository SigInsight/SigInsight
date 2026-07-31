package litequery

import (
	"fmt"
	"reflect"

	"github.com/ClickHouse/clickhouse-go/v2/lib/driver"
)

// WrapClickHouseRows adapts the ClickHouse driver's typed Scan contract to
// the intentionally dynamic Rows boundary used by Executor.
func WrapClickHouseRows(rows driver.Rows) Rows {
	return &clickHouseRows{Rows: rows}
}

type clickHouseRows struct{ driver.Rows }

func (rows *clickHouseRows) Scan(destinations ...any) error {
	types := rows.ColumnTypes()
	if len(destinations) != len(types) {
		return fmt.Errorf("lightweight scan destination count %d does not match column count %d", len(destinations), len(types))
	}
	values := make([]any, len(types))
	for index, columnType := range types {
		values[index] = reflect.New(columnType.ScanType()).Interface()
	}
	if err := rows.Rows.Scan(values...); err != nil {
		return err
	}
	for index, value := range values {
		target, ok := destinations[index].(*any)
		if !ok {
			return fmt.Errorf("lightweight scan destination %d has type %T, want *any", index, destinations[index])
		}
		*target = dereferenceClickHouseValue(value)
	}
	return nil
}

func dereferenceClickHouseValue(value any) any {
	if value == nil {
		return nil
	}
	reflected := reflect.ValueOf(value)
	for reflected.IsValid() && reflected.Kind() == reflect.Pointer {
		if reflected.IsNil() {
			return nil
		}
		reflected = reflected.Elem()
	}
	if !reflected.IsValid() {
		return nil
	}
	return reflected.Interface()
}
