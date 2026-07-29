package clickhousehealth

import (
	"context"

	"github.com/ClickHouse/clickhouse-go/v2"

	"github.com/SigNoz/signoz/pkg/query-service/interfaces"
	"github.com/SigNoz/signoz/pkg/types/ctxtypes"
	"github.com/SigNoz/signoz/pkg/types/instrumentationtypes"
)

type Reader struct {
	db clickhouse.Conn
}

var _ interfaces.ClickHouseHealthReader = (*Reader)(nil)

func New(db clickhouse.Conn) *Reader {
	return &Reader{db: db}
}

func (r *Reader) CheckClickHouse(ctx context.Context) error {
	ctx = ctxtypes.NewContextWithCommentVals(ctx, map[string]string{
		instrumentationtypes.CodeNamespace:    "clickhouse-reader",
		instrumentationtypes.CodeFunctionName: "CheckClickHouse",
	})
	rows, err := r.db.Query(ctx, "SELECT 1")
	if err != nil {
		return err
	}
	defer rows.Close()

	return nil
}
