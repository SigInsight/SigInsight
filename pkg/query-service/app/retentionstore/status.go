package retentionstore

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/uptrace/bun"

	errorsV2 "github.com/SigNoz/signoz/pkg/errors"
	"github.com/SigNoz/signoz/pkg/query-service/constants"
	"github.com/SigNoz/signoz/pkg/types"
)

// deleteTtlTransactions bounds retained TTL operation history per organization.
func (r *Reader) deleteTtlTransactions(ctx context.Context, orgID string, numberOfTransactionsStore int) {
	limitTransactions := []string{}
	err := r.sqlDB.BunDB().NewSelect().Column("transaction_id").Model(new(types.TTLSetting)).Where("org_id = ?", orgID).Group("transaction_id").OrderExpr("MAX(created_at) DESC").Limit(numberOfTransactionsStore).Scan(ctx, &limitTransactions)
	if err != nil {
		r.logger.Error("Error in processing ttl_status delete sql query", errorsV2.Attr(err))
	}

	_, err = r.sqlDB.BunDB().NewDelete().Model(new(types.TTLSetting)).Where("transaction_id NOT IN (?)", bun.In(limitTransactions)).Exec(ctx)
	if err != nil {
		r.logger.Error("Error in processing ttl_status delete sql query", errorsV2.Attr(err))
	}
}

func (r *Reader) checkTTLStatusItem(ctx context.Context, orgID string, tableName string) (*types.TTLSetting, error) {
	r.logger.Info("checkTTLStatusItem query", "tableName", tableName)
	ttl := new(types.TTLSetting)
	err := r.sqlDB.BunDB().NewSelect().Model(ttl).Where("table_name = ?", tableName).Where("org_id = ?", orgID).OrderExpr("created_at DESC").Limit(1).Scan(ctx)
	if err != nil && err != sql.ErrNoRows {
		r.logger.Error("Error in processing sql query", errorsV2.Attr(err))
		return nil, fmt.Errorf("query ttl status: %w", err)
	}
	if err == sql.ErrNoRows {
		return nil, nil
	}
	return ttl, nil
}

func (r *Reader) updateTTLStatus(ctx context.Context, id string, status string) error {
	_, err := r.sqlDB.BunDB().NewUpdate().Model(new(types.TTLSetting)).Set("updated_at = ?", time.Now()).Set("status = ?", status).Where("id = ?", id).Exec(ctx)
	return err
}

func isRecentTTLPending(status *types.TTLSetting, now time.Time) bool {
	return status != nil && status.Status == constants.StatusPending && now.Before(status.UpdatedAt.Add(time.Hour))
}

func (r *Reader) getTTLQueryStatus(ctx context.Context, orgID string, tableNameArray []string) (string, error) {
	failFlag := false
	status := constants.StatusSuccess
	for _, tableName := range tableNameArray {
		statusItem, err := r.checkTTLStatusItem(ctx, orgID, tableName)
		if err != nil {
			return "", err
		}
		if statusItem == nil {
			return "", nil
		}
		if isRecentTTLPending(statusItem, time.Now()) {
			return constants.StatusPending, nil
		}
		if statusItem.Status == constants.StatusFailed {
			failFlag = true
		}
	}
	if failFlag {
		status = constants.StatusFailed
	}
	return status, nil
}
