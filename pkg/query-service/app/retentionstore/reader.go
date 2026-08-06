package retentionstore

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"log/slog"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/ClickHouse/clickhouse-go/v2"
	"github.com/google/uuid"

	errorsV2 "github.com/SigNoz/signoz/pkg/errors"
	"github.com/SigNoz/signoz/pkg/query-service/constants"
	"github.com/SigNoz/signoz/pkg/query-service/interfaces"
	"github.com/SigNoz/signoz/pkg/query-service/model"
	"github.com/SigNoz/signoz/pkg/sqlstore"
	"github.com/SigNoz/signoz/pkg/types"
	"github.com/SigNoz/signoz/pkg/types/ctxtypes"
	"github.com/SigNoz/signoz/pkg/types/instrumentationtypes"
	"github.com/SigNoz/signoz/pkg/types/telemetrytypes"
	"github.com/SigNoz/signoz/pkg/valuer"
)

const (
	siginsightMetricDBName = "siginsight_metrics"

	siginsightSampleLocalTableName = "metric_points"
	siginsightSampleTableName      = "metric_points"

	siginsightSamplesAgg5mLocalTableName  = "metric_rollup_5m"
	siginsightSamplesAgg30mLocalTableName = "metric_rollup_30m"
	siginsightExpHistLocalTableName       = "exp_hist"
	siginsightTSLocalTableNameV4          = "metric_series"
	siginsightTSLocalTableNameV46Hrs      = "metric_series_6h"
	siginsightTSLocalTableNameV41Day      = "metric_series_1d"
	siginsightTSLocalTableNameV41Week     = "metric_series_1w"
)

type Config struct {
	TraceDB                string
	TraceTable             string
	TraceLocalTable        string
	TraceResourceTable     string
	ErrorTable             string
	DependencyGraphTable   string
	TraceSummaryTable      string
	SpanAttributeKeysTable string
	LogsDB                 string
	LogsTable              string
	LogsLocalTable         string
	LogsResourceTable      string
	LogsResourceLocalTable string
	LogsAttributeKeysTable string
	LogsResourceKeysTable  string
}

type Reader struct {
	db                       clickhouse.Conn
	sqlDB                    sqlstore.SQLStore
	logger                   *slog.Logger
	traceDB                  string
	traceTableName           string
	traceLocalTableName      string
	traceResourceTableV3     string
	errorTable               string
	dependencyGraphTable     string
	traceSummaryTable        string
	spanAttributesKeysTable  string
	logsDB                   string
	logsTableV2              string
	logsLocalTableV2         string
	logsResourceTableV2      string
	logsResourceLocalTableV2 string
	logsAttributeKeys        string
	logsResourceKeys         string
}

var _ interfaces.RetentionReader = (*Reader)(nil)

func New(logger *slog.Logger, sqlDB sqlstore.SQLStore, db clickhouse.Conn, config Config) *Reader {
	return &Reader{
		db:                       db,
		sqlDB:                    sqlDB,
		logger:                   logger,
		traceDB:                  config.TraceDB,
		traceTableName:           config.TraceTable,
		traceLocalTableName:      config.TraceLocalTable,
		traceResourceTableV3:     config.TraceResourceTable,
		errorTable:               config.ErrorTable,
		dependencyGraphTable:     config.DependencyGraphTable,
		traceSummaryTable:        config.TraceSummaryTable,
		spanAttributesKeysTable:  config.SpanAttributeKeysTable,
		logsDB:                   config.LogsDB,
		logsTableV2:              config.LogsTable,
		logsLocalTableV2:         config.LogsLocalTable,
		logsResourceTableV2:      config.LogsResourceTable,
		logsResourceLocalTableV2: config.LogsResourceLocalTable,
		logsAttributeKeys:        config.LogsAttributeKeysTable,
		logsResourceKeys:         config.LogsResourceKeysTable,
	}
}

// getLocalTableName keeps the table-name boundary explicit for TTL status rows.
// Canonical schema is single-node and does not accept historical aliases.
func getLocalTableName(tableName string) string {
	return tableName
}

func (r *Reader) setTTLTraces(ctx context.Context, orgID string, params *model.TTLParams) (*model.SetTTLResponseItem, error) {
	ctx = ctxtypes.NewContextWithCommentVals(ctx, map[string]string{
		instrumentationtypes.TelemetrySignal:  telemetrytypes.SignalTraces.StringValue(),
		instrumentationtypes.CodeNamespace:    "clickhouse-reader",
		instrumentationtypes.CodeFunctionName: "setTTLTraces",
	})
	// uuid is used as transaction id
	uuidWithHyphen := uuid.New()
	uuid := strings.Replace(uuidWithHyphen.String(), "-", "", -1)
	tableNames := []string{
		r.traceDB + "." + r.traceTableName,
		r.traceDB + "." + r.traceResourceTableV3,
		r.traceDB + "." + r.errorTable,
		r.traceDB + "." + r.dependencyGraphTable,
		r.traceDB + "." + r.traceSummaryTable,
		r.traceDB + "." + r.spanAttributesKeysTable,
	}

	coldStorageDuration := -1
	if len(params.ColdStorageVolume) > 0 {
		coldStorageDuration = int(params.ToColdStorageDuration)
	}

	// check if there is existing things to be done
	for _, tableName := range tableNames {
		statusItem, err := r.checkTTLStatusItem(ctx, orgID, tableName)
		if err != nil {
			return nil, err
		}
		if isRecentTTLPending(statusItem, time.Now()) {
			return nil, errorsV2.Newf(errorsV2.TypeAlreadyExists, errorsV2.CodeAlreadyExists, "TTL is already running")
		}
	}

	// TTL query
	ttlV2 := "ALTER TABLE %s MODIFY TTL toDateTime(%s) + INTERVAL %v SECOND DELETE"
	ttlV2ColdStorage := ", toDateTime(%s) + INTERVAL %v SECOND TO VOLUME '%s'"

	// TTL query for resource table
	ttlV2Resource := "ALTER TABLE %s MODIFY TTL toDateTime(seen_at_ts_bucket_start) + toIntervalSecond(1800) + INTERVAL %v SECOND DELETE"
	ttlTracesV2ResourceColdStorage := ", toDateTime(seen_at_ts_bucket_start) + toIntervalSecond(1800) + INTERVAL %v SECOND TO VOLUME '%s'"

	operationCtx := asyncTTLContext(ctx)
	for _, distributedTableName := range tableNames {
		go func(distributedTableName string) {
			tableName := getLocalTableName(distributedTableName)

			// for trace summary table, we need to use end instead of timestamp
			timestamp := "timestamp"
			if strings.HasSuffix(distributedTableName, r.traceSummaryTable) {
				timestamp = "end"
			}

			ttl := types.TTLSetting{
				Identifiable: types.Identifiable{
					ID: valuer.GenerateUUID(),
				},
				TimeAuditable: types.TimeAuditable{
					CreatedAt: time.Now(),
					UpdatedAt: time.Now(),
				},
				TransactionID:  uuid,
				TableName:      tableName,
				TTL:            int(params.DelDuration),
				Status:         constants.StatusPending,
				ColdStorageTTL: coldStorageDuration,
				OrgID:          orgID,
			}
			_, dbErr := r.
				sqlDB.
				BunDB().
				NewInsert().
				Model(&ttl).
				Exec(operationCtx)
			if dbErr != nil {
				r.logger.Error("error in inserting to ttl_status table", errorsV2.Attr(dbErr))
				return
			}

			req := fmt.Sprintf(ttlV2, tableName, timestamp, params.DelDuration)
			if strings.HasSuffix(distributedTableName, r.traceResourceTableV3) {
				req = fmt.Sprintf(ttlV2Resource, tableName, params.DelDuration)
			}

			if len(params.ColdStorageVolume) > 0 && !strings.HasSuffix(distributedTableName, r.spanAttributesKeysTable) {
				if strings.HasSuffix(distributedTableName, r.traceResourceTableV3) {
					req += fmt.Sprintf(ttlTracesV2ResourceColdStorage, params.ToColdStorageDuration, params.ColdStorageVolume)
				} else {
					req += fmt.Sprintf(ttlV2ColdStorage, timestamp, params.ToColdStorageDuration, params.ColdStorageVolume)
				}
			}
			err := r.setColdStorage(operationCtx, tableName, params.ColdStorageVolume)
			if err != nil {
				r.logger.Error("Error in setting cold storage", errorsV2.Attr(err))
				if dbErr := r.updateTTLStatus(operationCtx, ttl.ID.StringValue(), constants.StatusFailed); dbErr != nil {
					r.logger.Error("Error in processing ttl_status update sql query", errorsV2.Attr(dbErr))
				}
				return
			}
			req += " SETTINGS materialize_ttl_after_modify=0;"
			r.logger.Error(" ExecutingTTL request: ", "request", req)
			if err := r.db.Exec(operationCtx, req); err != nil {
				r.logger.Error("Error in executing set TTL query", errorsV2.Attr(err))
				if dbErr := r.updateTTLStatus(operationCtx, ttl.ID.StringValue(), constants.StatusFailed); dbErr != nil {
					r.logger.Error("Error in processing ttl_status update sql query", errorsV2.Attr(dbErr))
				}
				return
			}
			if dbErr := r.updateTTLStatus(operationCtx, ttl.ID.StringValue(), constants.StatusSuccess); dbErr != nil {
				r.logger.Error("Error in processing ttl_status update sql query", errorsV2.Attr(dbErr))
			}
		}(distributedTableName)
	}
	return &model.SetTTLResponseItem{Message: "move ttl has been successfully set up"}, nil
}

func (r *Reader) SetCustomRetentionTTL(ctx context.Context, orgID string, params *model.CustomRetentionTTLParams) (*model.CustomRetentionTTLResponse, error) {

	ctx = ctxtypes.NewContextWithCommentVals(ctx, map[string]string{
		instrumentationtypes.TelemetrySignal:  telemetrytypes.SignalLogs.StringValue(),
		instrumentationtypes.CodeNamespace:    "clickhouse-reader",
		instrumentationtypes.CodeFunctionName: "SetCustomRetentionTTL",
	})

	// Keep only latest 100 transactions/requests
	r.deleteTtlTransactions(ctx, orgID, 100)

	uuidWithHyphen := valuer.GenerateUUID()
	uuid := strings.Replace(uuidWithHyphen.String(), "-", "", -1)

	if params.Type != constants.LogsTTL {
		return nil, errorsV2.Newf(errorsV2.TypeInternal, errorsV2.CodeInternal, "custom retention TTL only supported for logs")
	}

	// Validate TTL conditions
	if err := r.validateTTLConditions(ctx, params.TTLConditions); err != nil {
		return nil, err
	}

	// Calculate cold storage duration
	coldStorageDuration := -1
	if len(params.ColdStorageVolume) > 0 && params.ToColdStorageDurationDays > 0 {
		coldStorageDuration = int(params.ToColdStorageDurationDays) // Already in days
	}

	tableNames := []string{
		r.logsDB + "." + r.logsLocalTableV2,
		r.logsDB + "." + r.logsResourceLocalTableV2,
		getLocalTableName(r.logsDB + "." + r.logsAttributeKeys),
		getLocalTableName(r.logsDB + "." + r.logsResourceKeys),
	}
	distributedTableNames := []string{
		r.logsDB + "." + r.logsTableV2,
		r.logsDB + "." + r.logsResourceTableV2,
	}

	for _, tableName := range tableNames {
		statusItem, apiErr := r.checkCustomRetentionTTLStatusItem(ctx, orgID, tableName)
		if apiErr != nil {
			return nil, errorsV2.Newf(errorsV2.TypeInternal, errorsV2.CodeInternal, "error in processing custom_retention_ttl_status check sql query")
		}
		if statusItem.Status == constants.StatusPending {
			return nil, errorsV2.Newf(errorsV2.TypeInternal, errorsV2.CodeInternal, "custom retention TTL is already running")
		}
	}

	multiIfExpr := r.buildMultiIfExpression(params.TTLConditions, params.DefaultTTLDays, false)
	resourceMultiIfExpr := r.buildMultiIfExpression(params.TTLConditions, params.DefaultTTLDays, true)

	ttlPayload := make(map[string][]string)

	queries := []string{
		fmt.Sprintf(`ALTER TABLE %s MODIFY COLUMN _retention_days UInt16 DEFAULT %s`,
			tableNames[0], multiIfExpr),
		// for distributed table
		fmt.Sprintf(`ALTER TABLE %s MODIFY COLUMN _retention_days UInt16 DEFAULT %s`,
			distributedTableNames[0], multiIfExpr),
	}

	if len(params.ColdStorageVolume) > 0 && coldStorageDuration > 0 {
		queries = append(queries, fmt.Sprintf(`ALTER TABLE %s MODIFY COLUMN _retention_days_cold UInt16 DEFAULT %d`,
			tableNames[0], coldStorageDuration))
		// for distributed table
		queries = append(queries, fmt.Sprintf(`ALTER TABLE %s MODIFY COLUMN _retention_days_cold UInt16 DEFAULT %d`,
			distributedTableNames[0], coldStorageDuration))

		queries = append(queries, fmt.Sprintf(`ALTER TABLE %s MODIFY TTL toDateTime(timestamp / 1000000000) + toIntervalDay(_retention_days) DELETE, toDateTime(timestamp / 1000000000) + toIntervalDay(_retention_days_cold) TO VOLUME '%s' SETTINGS materialize_ttl_after_modify=0`,
			tableNames[0], params.ColdStorageVolume))
	}

	ttlPayload[tableNames[0]] = queries

	resourceQueries := []string{
		fmt.Sprintf(`ALTER TABLE %s MODIFY COLUMN _retention_days UInt16 DEFAULT %s`,
			tableNames[1], resourceMultiIfExpr),
		// for distributed table
		fmt.Sprintf(`ALTER TABLE %s MODIFY COLUMN _retention_days UInt16 DEFAULT %s`,
			distributedTableNames[1], resourceMultiIfExpr),
	}

	if len(params.ColdStorageVolume) > 0 && coldStorageDuration > 0 {
		resourceQueries = append(resourceQueries, fmt.Sprintf(`ALTER TABLE %s MODIFY COLUMN _retention_days_cold UInt16 DEFAULT %d`,
			tableNames[1], coldStorageDuration))
		// for distributed table
		resourceQueries = append(resourceQueries, fmt.Sprintf(`ALTER TABLE %s MODIFY COLUMN _retention_days_cold UInt16 DEFAULT %d`,
			distributedTableNames[1], coldStorageDuration))
		resourceQueries = append(resourceQueries, fmt.Sprintf(`ALTER TABLE %s MODIFY TTL toDateTime(seen_at_ts_bucket_start) + toIntervalSecond(1800) + toIntervalDay(_retention_days) DELETE, toDateTime(seen_at_ts_bucket_start) + toIntervalSecond(1800) + toIntervalDay(_retention_days_cold) TO VOLUME '%s' SETTINGS materialize_ttl_after_modify=0`,
			tableNames[1], params.ColdStorageVolume))
	}

	ttlPayload[tableNames[1]] = resourceQueries

	// NOTE: Since logs support custom rule based retention, that makes it difficult to identify which attributes, resource keys
	// we need to keep, hence choosing MAX for safe side and not to create any complex solution for this.
	maxRetentionTTL := params.DefaultTTLDays
	for _, rule := range params.TTLConditions {
		maxRetentionTTL = max(maxRetentionTTL, rule.TTLDays)
	}

	ttlPayload[tableNames[2]] = []string{
		fmt.Sprintf("ALTER TABLE %s MODIFY TTL timestamp + toIntervalDay(%d) DELETE SETTINGS materialize_ttl_after_modify=0",
			tableNames[2], maxRetentionTTL),
	}

	ttlPayload[tableNames[3]] = []string{
		fmt.Sprintf("ALTER TABLE %s MODIFY TTL timestamp + toIntervalDay(%d) DELETE SETTINGS materialize_ttl_after_modify=0",
			tableNames[3], maxRetentionTTL),
	}

	ttlConditionsJSON, err := json.Marshal(params.TTLConditions)
	if err != nil {
		return nil, errorsV2.Wrapf(err, errorsV2.TypeInternal, errorsV2.CodeInternal, "error marshalling TTL condition")
	}

	for tableName, queries := range ttlPayload {
		customTTL := types.TTLSetting{
			Identifiable: types.Identifiable{
				ID: valuer.GenerateUUID(),
			},
			TimeAuditable: types.TimeAuditable{
				CreatedAt: time.Now(),
				UpdatedAt: time.Now(),
			},
			TransactionID:  uuid,
			TableName:      tableName,
			TTL:            params.DefaultTTLDays,
			Condition:      string(ttlConditionsJSON),
			Status:         constants.StatusPending,
			ColdStorageTTL: coldStorageDuration,
			OrgID:          orgID,
		}

		// Insert TTL setting record
		_, dbErr := r.sqlDB.BunDB().NewInsert().Model(&customTTL).Exec(ctx)
		if dbErr != nil {
			r.logger.Error("error in inserting to custom_retention_ttl_settings table", errorsV2.Attr(dbErr))
			return nil, errorsV2.Wrapf(dbErr, errorsV2.TypeInternal, errorsV2.CodeInternal, "error inserting TTL settings")
		}

		if len(params.ColdStorageVolume) > 0 && coldStorageDuration > 0 {
			err := r.setColdStorage(ctx, tableName, params.ColdStorageVolume)
			if err != nil {
				r.logger.Error("error in setting cold storage", errorsV2.Attr(err))
				r.updateCustomRetentionTTLStatus(ctx, orgID, tableName, constants.StatusFailed)
				return nil, errorsV2.Wrapf(err, errorsV2.TypeInternal, errorsV2.CodeInternal, "error setting cold storage for table %s", tableName)
			}
		}

		for i, query := range queries {
			r.logger.Debug("Executing custom retention TTL request: ", "request", query, "step", i+1)
			if err := r.db.Exec(ctx, query); err != nil {
				r.logger.Error("error while setting custom retention ttl", errorsV2.Attr(err))
				r.updateCustomRetentionTTLStatus(ctx, orgID, tableName, constants.StatusFailed)
				return nil, errorsV2.Wrapf(err, errorsV2.TypeInternal, errorsV2.CodeInternal, "error setting custom retention TTL for table %s, query: %s", tableName, query)
			}
		}

		r.updateCustomRetentionTTLStatus(ctx, orgID, tableName, constants.StatusSuccess)
	}

	return &model.CustomRetentionTTLResponse{
		Message: "custom retention TTL has been successfully set up",
	}, nil
}

// New method to build multiIf expressions with support for multiple AND conditions
func (r *Reader) buildMultiIfExpression(ttlConditions []model.CustomRetentionRule, defaultTTLDays int, isResourceTable bool) string {
	var conditions []string

	for i, rule := range ttlConditions {
		r.logger.Debug("Processing rule", "ruleIndex", i, "ttlDays", rule.TTLDays, "conditionsCount", len(rule.Filters))

		if len(rule.Filters) == 0 {
			r.logger.Warn("Rule has no filters, skipping", "ruleIndex", i)
			continue
		}

		// Build AND conditions for this rule
		var andConditions []string
		for j, condition := range rule.Filters {
			r.logger.Debug("Processing condition", "ruleIndex", i, "conditionIndex", j, "key", condition.Key, "values", condition.Values)

			// This should not happen as validation should catch it
			if len(condition.Values) == 0 {
				r.logger.Error("Condition has no values - this should have been caught in validation", "ruleIndex", i, "conditionIndex", j)
				continue
			}

			// Properly quote values for IN clause
			quotedValues := make([]string, len(condition.Values))
			for k, v := range condition.Values {
				quotedValues[k] = fmt.Sprintf("'%s'", v)
			}

			var conditionExpr string
			if isResourceTable {
				// For resource table, use JSONExtractString
				conditionExpr = fmt.Sprintf(
					"JSONExtractString(labels, '%s') IN (%s)",
					condition.Key,
					strings.Join(quotedValues, ", "),
				)
			} else {
				// For main logs table, use resources_string
				conditionExpr = fmt.Sprintf(
					"resources_string['%s'] IN (%s)",
					condition.Key,
					strings.Join(quotedValues, ", "),
				)
			}
			andConditions = append(andConditions, conditionExpr)
		}

		if len(andConditions) > 0 {
			// Join all conditions with AND
			fullCondition := strings.Join(andConditions, " AND ")
			conditionWithTTL := fmt.Sprintf("%s, %d", fullCondition, rule.TTLDays)
			r.logger.Debug("Adding condition to multiIf", "condition", conditionWithTTL)
			conditions = append(conditions, conditionWithTTL)
		}
	}

	// Handle case where no valid conditions were found
	if len(conditions) == 0 {
		r.logger.Info("No valid conditions found, returning default TTL", "defaultTTLDays", defaultTTLDays)
		return fmt.Sprintf("%d", defaultTTLDays)
	}

	result := fmt.Sprintf(
		"multiIf(%s, %d)",
		strings.Join(conditions, ", "),
		defaultTTLDays,
	)

	r.logger.Debug("Final multiIf expression", "expression", result)
	return result
}

func (r *Reader) GetCustomRetentionTTL(ctx context.Context, orgID string) (*model.GetCustomRetentionTTLResponse, error) {
	response := &model.GetCustomRetentionTTLResponse{}
	customTTL := new(types.TTLSetting)
	err := r.sqlDB.BunDB().NewSelect().
		Model(customTTL).
		Where("org_id = ?", orgID).
		Where("table_name = ?", r.logsDB+"."+r.logsLocalTableV2).
		OrderExpr("created_at DESC").
		Limit(1).
		Scan(ctx)
	if err != nil && err != sql.ErrNoRows {
		r.logger.Error("Error in processing sql query", errorsV2.Attr(err))
		return nil, errorsV2.Newf(errorsV2.TypeInternal, errorsV2.CodeInternal, "error in processing get custom ttl query")
	}
	if err == sql.ErrNoRows {
		response.DefaultTTLDays = 15
		response.TTLConditions = []model.CustomRetentionRule{}
		response.Status = constants.StatusSuccess
		response.ColdStorageTTLDays = -1
		return response, nil
	}

	var ttlConditions []model.CustomRetentionRule
	if customTTL.Condition != "" {
		if err := json.Unmarshal([]byte(customTTL.Condition), &ttlConditions); err != nil {
			r.logger.Error("Error parsing TTL conditions", errorsV2.Attr(err))
			ttlConditions = []model.CustomRetentionRule{}
		}
	}

	response.DefaultTTLDays = customTTL.TTL
	response.TTLConditions = ttlConditions
	response.Status = customTTL.Status
	response.ColdStorageTTLDays = customTTL.ColdStorageTTL

	return response, nil
}

func (r *Reader) checkCustomRetentionTTLStatusItem(ctx context.Context, orgID string, tableName string) (*types.TTLSetting, error) {
	ttl := new(types.TTLSetting)
	err := r.sqlDB.BunDB().NewSelect().
		Model(ttl).
		Where("table_name = ?", tableName).
		Where("org_id = ?", orgID).
		OrderExpr("created_at DESC").
		Limit(1).
		Scan(ctx)

	if err != nil && err != sql.ErrNoRows {
		r.logger.Error("Error in processing sql query", errorsV2.Attr(err))
		return ttl, errorsV2.Newf(errorsV2.TypeInternal, errorsV2.CodeInternal, "error in processing custom_retention_ttl_status check sql query")
	}

	return ttl, nil
}

func (r *Reader) updateCustomRetentionTTLStatus(ctx context.Context, orgID, tableName, status string) {
	statusItem, apiErr := r.checkCustomRetentionTTLStatusItem(ctx, orgID, tableName)
	if apiErr == nil && statusItem != nil {
		_, dbErr := r.sqlDB.BunDB().NewUpdate().
			Model(new(types.TTLSetting)).
			Set("updated_at = ?", time.Now()).
			Set("status = ?", status).
			Where("id = ?", statusItem.ID.StringValue()).
			Exec(ctx)
		if dbErr != nil {
			r.logger.Error("Error in processing custom_retention_ttl_status update sql query", errorsV2.Attr(dbErr))
		}
	}
}

// Enhanced validation function with duplicate detection and efficient key validation
func (r *Reader) validateTTLConditions(ctx context.Context, ttlConditions []model.CustomRetentionRule) error {
	ctx = ctxtypes.NewContextWithCommentVals(ctx, map[string]string{
		instrumentationtypes.CodeNamespace:    "clickhouse-reader",
		instrumentationtypes.CodeFunctionName: "validateTTLConditions",
	})
	if len(ttlConditions) == 0 {
		return nil
	}

	// Collect all unique keys and detect duplicates
	var allKeys []string
	keySet := make(map[string]struct{})
	conditionSignatures := make(map[string]bool)

	for i, rule := range ttlConditions {
		if len(rule.Filters) == 0 {
			return errorsV2.Newf(errorsV2.TypeInternal, errorsV2.CodeInternal, "rule at index %d has no filters", i)
		}

		// Create a signature for this rule's conditions to detect duplicates
		var conditionKeys []string
		var conditionValues []string

		for j, condition := range rule.Filters {
			if len(condition.Values) == 0 {
				return errorsV2.Newf(errorsV2.TypeInternal, errorsV2.CodeInternal, "condition at rule %d, condition %d has no values", i, j)
			}

			// Collect unique keys
			if _, exists := keySet[condition.Key]; !exists {
				allKeys = append(allKeys, condition.Key)
				keySet[condition.Key] = struct{}{}
			}

			// Build signature for duplicate detection
			conditionKeys = append(conditionKeys, condition.Key)
			conditionValues = append(conditionValues, strings.Join(condition.Values, ","))
		}

		// Create signature by sorting keys and values to handle order-independent comparison
		sort.Strings(conditionKeys)
		sort.Strings(conditionValues)
		signature := strings.Join(conditionKeys, "|") + ":" + strings.Join(conditionValues, "|")

		if conditionSignatures[signature] {
			return errorsV2.Newf(errorsV2.TypeInternal, errorsV2.CodeInternal, "duplicate rule detected at index %d: rules with identical conditions are not allowed", i)
		}
		conditionSignatures[signature] = true
	}

	if len(allKeys) == 0 {
		return nil
	}

	// Create placeholders for IN query
	placeholders := make([]string, len(allKeys))
	for i := range allKeys {
		placeholders[i] = "?"
	}

	// Efficient validation using IN query
	query := fmt.Sprintf("SELECT name FROM %s.%s WHERE name IN (%s)",
		r.logsDB, r.logsResourceKeys, strings.Join(placeholders, ", "))

	// Convert keys to interface{} for query parameters
	params := make([]interface{}, len(allKeys))
	for i, key := range allKeys {
		params[i] = key
	}

	rows, err := r.db.Query(ctx, query, params...)
	if err != nil {
		return errorsV2.Wrapf(err, errorsV2.TypeInternal, errorsV2.CodeInternal, "failed to validate resource keys")
	}
	defer rows.Close()

	// Collect valid keys
	validKeys := make(map[string]struct{})
	for rows.Next() {
		var name string
		if err := rows.Scan(&name); err != nil {
			return errorsV2.Wrapf(err, errorsV2.TypeInternal, errorsV2.CodeInternal, "failed to scan resource keys")
		}
		validKeys[name] = struct{}{}
	}

	// Find invalid keys
	var invalidKeys []string
	for _, key := range allKeys {
		if _, exists := validKeys[key]; !exists {
			invalidKeys = append(invalidKeys, key)
		}
	}

	if len(invalidKeys) > 0 {
		return errorsV2.Newf(errorsV2.TypeInternal, errorsV2.CodeInternal, "invalid resource keys found: %v. Please check logs_resource_keys table for valid keys", invalidKeys)
	}

	return nil
}

// SetTTL sets the TTL for traces or metrics tables.
// This is an async API which creates goroutines to set TTL.
// Status of TTL update is tracked with ttl_status table in sqlite db.
func (r *Reader) SetTTL(ctx context.Context, orgID string, params *model.TTLParams) (*model.SetTTLResponseItem, error) {
	// Keep only latest 100 transactions/requests
	r.deleteTtlTransactions(ctx, orgID, 100)

	switch params.Type {
	case constants.TraceTTL:
		return r.setTTLTraces(ctx, orgID, params)
	case constants.MetricsTTL:
		return r.setTTLMetrics(ctx, orgID, params)
	default:
		return nil, errorsV2.NewInvalidInputf(errorsV2.CodeInvalidInput, "ttl type should be metrics or traces, got %v", params.Type)
	}

}

func (r *Reader) setTTLMetrics(ctx context.Context, orgID string, params *model.TTLParams) (*model.SetTTLResponseItem, error) {
	ctx = ctxtypes.NewContextWithCommentVals(ctx, map[string]string{
		instrumentationtypes.TelemetrySignal:  telemetrytypes.SignalMetrics.StringValue(),
		instrumentationtypes.CodeNamespace:    "clickhouse-reader",
		instrumentationtypes.CodeFunctionName: "setTTLMetrics",
	})
	// uuid is used as transaction id
	uuidWithHyphen := uuid.New()
	uuid := strings.Replace(uuidWithHyphen.String(), "-", "", -1)

	coldStorageDuration := -1
	if len(params.ColdStorageVolume) > 0 {
		coldStorageDuration = int(params.ToColdStorageDuration)
	}
	tableNames := []string{
		siginsightMetricDBName + "." + siginsightSampleLocalTableName,
		siginsightMetricDBName + "." + siginsightSamplesAgg5mLocalTableName,
		siginsightMetricDBName + "." + siginsightSamplesAgg30mLocalTableName,
		siginsightMetricDBName + "." + siginsightExpHistLocalTableName,
		siginsightMetricDBName + "." + siginsightTSLocalTableNameV4,
		siginsightMetricDBName + "." + siginsightTSLocalTableNameV46Hrs,
		siginsightMetricDBName + "." + siginsightTSLocalTableNameV41Day,
		siginsightMetricDBName + "." + siginsightTSLocalTableNameV41Week,
	}
	for _, tableName := range tableNames {
		statusItem, err := r.checkTTLStatusItem(ctx, orgID, tableName)
		if err != nil {
			return nil, err
		}
		if isRecentTTLPending(statusItem, time.Now()) {
			return nil, errorsV2.Newf(errorsV2.TypeAlreadyExists, errorsV2.CodeAlreadyExists, "TTL is already running")
		}
	}
	operationCtx := asyncTTLContext(ctx)
	metricTTL := func(tableName string) {
		ttl := types.TTLSetting{
			Identifiable: types.Identifiable{
				ID: valuer.GenerateUUID(),
			},
			TimeAuditable: types.TimeAuditable{
				CreatedAt: time.Now(),
				UpdatedAt: time.Now(),
			},
			TransactionID:  uuid,
			TableName:      tableName,
			TTL:            int(params.DelDuration),
			Status:         constants.StatusPending,
			ColdStorageTTL: coldStorageDuration,
			OrgID:          orgID,
		}
		_, dbErr := r.
			sqlDB.
			BunDB().
			NewInsert().
			Model(&ttl).
			Exec(operationCtx)
		if dbErr != nil {
			r.logger.Error("error in inserting to ttl_status table", errorsV2.Attr(dbErr))
			return
		}
		// Every canonical metrics table stores event time as unix_milli. The
		// former name-based v4 check became false after M16 renamed the tables,
		// causing asynchronous TTL updates to reference nonexistent timestamp_ms.
		timeColumn := "unix_milli"

		req := fmt.Sprintf(
			"ALTER TABLE %v MODIFY TTL toDateTime(toUInt32(%s / 1000), 'UTC') + "+
				"INTERVAL %v SECOND DELETE", tableName, timeColumn, params.DelDuration)
		if len(params.ColdStorageVolume) > 0 {
			req += fmt.Sprintf(", toDateTime(toUInt32(%s / 1000), 'UTC')"+
				" + INTERVAL %v SECOND TO VOLUME '%s'",
				timeColumn, params.ToColdStorageDuration, params.ColdStorageVolume)
		}
		err := r.setColdStorage(operationCtx, tableName, params.ColdStorageVolume)
		if err != nil {
			r.logger.Error("Error in setting cold storage", errorsV2.Attr(err))
			if dbErr := r.updateTTLStatus(operationCtx, ttl.ID.StringValue(), constants.StatusFailed); dbErr != nil {
				r.logger.Error("Error in processing ttl_status update sql query", errorsV2.Attr(dbErr))
			}
			return
		}
		req += " SETTINGS materialize_ttl_after_modify=0"
		r.logger.Info("Executing TTL request: ", "request", req)
		if err := r.db.Exec(operationCtx, req); err != nil {
			r.logger.Error("error while setting ttl.", errorsV2.Attr(err))
			if dbErr := r.updateTTLStatus(operationCtx, ttl.ID.StringValue(), constants.StatusFailed); dbErr != nil {
				r.logger.Error("Error in processing ttl_status update sql query", errorsV2.Attr(dbErr))
			}
			return
		}
		if dbErr := r.updateTTLStatus(operationCtx, ttl.ID.StringValue(), constants.StatusSuccess); dbErr != nil {
			r.logger.Error("Error in processing ttl_status update sql query", errorsV2.Attr(dbErr))
		}
	}
	for _, tableName := range tableNames {
		go metricTTL(tableName)
	}
	return &model.SetTTLResponseItem{Message: "move ttl has been successfully set up"}, nil
}

func asyncTTLContext(ctx context.Context) context.Context {
	return context.WithoutCancel(ctx)
}

func (r *Reader) setColdStorage(ctx context.Context, tableName string, coldStorageVolume string) error {

	ctx = ctxtypes.NewContextWithCommentVals(ctx, map[string]string{
		instrumentationtypes.CodeNamespace:    "clickhouse-reader",
		instrumentationtypes.CodeFunctionName: "setColdStorage",
	})
	// Set the storage policy for the required table. If it is already set, then setting it again
	// will not a problem.
	if len(coldStorageVolume) > 0 {
		policyReq := fmt.Sprintf("ALTER TABLE %s MODIFY SETTING storage_policy='tiered'", tableName)

		r.logger.Info("Executing Storage policy request: ", "request", policyReq)
		if err := r.db.Exec(ctx, policyReq); err != nil {
			r.logger.Error("error while setting storage policy", errorsV2.Attr(err))
			return fmt.Errorf("set storage policy: %w", err)
		}
	}
	return nil
}

// GetDisks returns a list of disks {name, type} configured in clickhouse DB.
func (r *Reader) GetDisks(ctx context.Context) (*[]model.DiskItem, error) {
	ctx = ctxtypes.NewContextWithCommentVals(ctx, map[string]string{
		instrumentationtypes.CodeNamespace:    "clickhouse-reader",
		instrumentationtypes.CodeFunctionName: "GetDisks",
	})
	diskItems := []model.DiskItem{}

	query := "SELECT name,type FROM system.disks"
	if err := r.db.Select(ctx, &diskItems, query); err != nil {
		r.logger.Error("Error in processing sql query", errorsV2.Attr(err))
		return nil, fmt.Errorf("get ClickHouse disks: %w", err)
	}

	return &diskItems, nil
}

func getLocalTableNameArray(tableNames []string) []string {
	localTableNames := make([]string, 0, len(tableNames))
	for _, tableName := range tableNames {
		localTableNames = append(localTableNames, getLocalTableName(tableName))
	}

	return localTableNames
}

// GetTTL returns current ttl, expected ttl and past setTTL status for metrics/traces.
func (r *Reader) GetTTL(ctx context.Context, orgID string, ttlParams *model.GetTTLParams) (*model.GetTTLResponseItem, error) {

	ctx = ctxtypes.NewContextWithCommentVals(ctx, map[string]string{
		instrumentationtypes.CodeNamespace:    "clickhouse-reader",
		instrumentationtypes.CodeFunctionName: "GetTTL",
	})
	parseTTL := func(queryResp string) (int, int) {

		r.logger.Info("Parsing TTL from: ", "queryResp", queryResp)
		deleteTTLExp := regexp.MustCompile(`toIntervalSecond\(([0-9]*)\)`)
		moveTTLExp := regexp.MustCompile(`toIntervalSecond\(([0-9]*)\) TO VOLUME`)

		var delTTL, moveTTL int = -1, -1

		m := deleteTTLExp.FindStringSubmatch(queryResp)
		if len(m) > 1 {
			seconds_int, err := strconv.Atoi(m[1])
			if err != nil {
				return -1, -1
			}
			delTTL = seconds_int / 3600
		}

		m = moveTTLExp.FindStringSubmatch(queryResp)
		if len(m) > 1 {
			seconds_int, err := strconv.Atoi(m[1])
			if err != nil {
				return -1, -1
			}
			moveTTL = seconds_int / 3600
		}

		return delTTL, moveTTL
	}

	getMetricsTTL := func() (*model.DBResponseTTL, error) {
		var dbResp []model.DBResponseTTL

		query := fmt.Sprintf("SELECT engine_full FROM system.tables WHERE name='%v' AND database='%v'", siginsightSampleLocalTableName, siginsightMetricDBName)

		err := r.db.Select(ctx, &dbResp, query)

		if err != nil {
			r.logger.Error("error while getting ttl", errorsV2.Attr(err))
			return nil, fmt.Errorf("query metrics ttl: %w", err)
		}
		if len(dbResp) == 0 {
			return nil, nil
		} else {
			return &dbResp[0], nil
		}
	}

	getTracesTTL := func() (*model.DBResponseTTL, error) {
		var dbResp []model.DBResponseTTL

		query := fmt.Sprintf("SELECT engine_full FROM system.tables WHERE name='%v' AND database='%v'", r.traceLocalTableName, r.traceDB)

		err := r.db.Select(ctx, &dbResp, query)

		if err != nil {
			r.logger.Error("error while getting ttl", errorsV2.Attr(err))
			return nil, fmt.Errorf("query traces ttl: %w", err)
		}
		if len(dbResp) == 0 {
			return nil, nil
		} else {
			return &dbResp[0], nil
		}
	}

	switch ttlParams.Type {
	case constants.TraceTTL:
		tableNameArray := []string{
			r.traceDB + "." + r.traceTableName,
			r.traceDB + "." + r.traceResourceTableV3,
			r.traceDB + "." + r.errorTable,
			r.traceDB + "." + r.dependencyGraphTable,
			r.traceDB + "." + r.traceSummaryTable,
		}
		tableNameArray = getLocalTableNameArray(tableNameArray)
		status, err := r.getTTLQueryStatus(ctx, orgID, tableNameArray)
		if err != nil {
			return nil, err
		}
		dbResp, err := getTracesTTL()
		if err != nil {
			return nil, err
		}
		if dbResp == nil {
			return nil, fmt.Errorf("trace table %s is missing from ClickHouse", r.traceLocalTableName)
		}
		ttlQuery, err := r.checkTTLStatusItem(ctx, orgID, tableNameArray[0])
		if err != nil {
			return nil, err
		}
		expectedTTL, expectedColdStorageTTL := ttlHours(ttlQuery)

		delTTL, moveTTL := parseTTL(dbResp.EngineFull)
		return &model.GetTTLResponseItem{TracesTime: delTTL, TracesMoveTime: moveTTL, ExpectedTracesTime: expectedTTL, ExpectedTracesMoveTime: expectedColdStorageTTL, Status: status}, nil

	case constants.MetricsTTL:
		tableNameArray := []string{siginsightMetricDBName + "." + siginsightSampleTableName}
		tableNameArray = getLocalTableNameArray(tableNameArray)
		status, err := r.getTTLQueryStatus(ctx, orgID, tableNameArray)
		if err != nil {
			return nil, err
		}
		dbResp, err := getMetricsTTL()
		if err != nil {
			return nil, err
		}
		if dbResp == nil {
			return nil, fmt.Errorf("metrics table %s is missing from ClickHouse", siginsightSampleLocalTableName)
		}
		ttlQuery, err := r.checkTTLStatusItem(ctx, orgID, tableNameArray[0])
		if err != nil {
			return nil, err
		}
		expectedTTL, expectedColdStorageTTL := ttlHours(ttlQuery)

		delTTL, moveTTL := parseTTL(dbResp.EngineFull)
		return &model.GetTTLResponseItem{MetricsTime: delTTL, MetricsMoveTime: moveTTL, ExpectedMetricsTime: expectedTTL, ExpectedMetricsMoveTime: expectedColdStorageTTL, Status: status}, nil

	default:
		return nil, errorsV2.NewInvalidInputf(errorsV2.CodeInvalidInput, "ttl type should be metrics or traces, got %v", ttlParams.Type)
	}

}

func ttlHours(status *types.TTLSetting) (int, int) {
	if status == nil {
		return -1, -1
	}

	coldStorageTTL := status.ColdStorageTTL
	if coldStorageTTL != -1 {
		coldStorageTTL /= 3600
	}
	return status.TTL / 3600, coldStorageTTL
}
