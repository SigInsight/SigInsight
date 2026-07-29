package rulestatehistorystore

import (
	"context"
	"fmt"

	"github.com/SigNoz/signoz/pkg/query-service/common"
	"github.com/SigNoz/signoz/pkg/query-service/model"
	"github.com/SigNoz/signoz/pkg/types/ctxtypes"
	"github.com/SigNoz/signoz/pkg/types/instrumentationtypes"
	"github.com/SigNoz/signoz/pkg/types/timeseriestypes"
)

// GetTotalTriggers and GetTriggersByInterval are the compact statistics query
// surface for Alert History; timeline reads live in reader.go.
func (r *Reader) GetTotalTriggers(ctx context.Context, ruleID string, params *model.QueryRuleStateHistory) (uint64, error) {
	ctx = ctxtypes.NewContextWithCommentVals(ctx, map[string]string{
		instrumentationtypes.CodeNamespace:    "clickhouse-reader",
		instrumentationtypes.CodeFunctionName: "GetTotalTriggers",
	})
	query := fmt.Sprintf("SELECT count(*) FROM %s.%s WHERE rule_id = ? AND (state_changed = true) AND (state = ?) AND unix_milli >= ? AND unix_milli <= ?", r.database, r.table)
	var totalTriggers uint64
	if err := r.db.QueryRow(ctx, query, ruleID, model.StateFiring.String(), params.Start, params.End).Scan(&totalTriggers); err != nil {
		return 0, err
	}
	return totalTriggers, nil
}

func (r *Reader) GetTriggersByInterval(ctx context.Context, ruleID string, params *model.QueryRuleStateHistory) (*timeseriestypes.Series, error) {
	step := common.MinAllowedStepInterval(params.Start, params.End)
	query := fmt.Sprintf("SELECT count(*), toStartOfInterval(toDateTime(intDiv(unix_milli, 1000)), INTERVAL %d SECOND) as ts FROM %s.%s WHERE rule_id = ? AND (state_changed = true) AND (state = ?) AND unix_milli >= ? AND unix_milli <= ? GROUP BY ts ORDER BY ts ASC", step, r.database, r.table)
	result, err := r.getTimeSeriesResult(ctx, query, ruleID, model.StateFiring.String(), params.Start, params.End)
	if err != nil || len(result) == 0 {
		return nil, err
	}
	return result[0], nil
}
