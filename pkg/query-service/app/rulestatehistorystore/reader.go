package rulestatehistorystore

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"log/slog"
	"math"
	"reflect"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/ClickHouse/clickhouse-go/v2"
	"github.com/ClickHouse/clickhouse-go/v2/lib/driver"
	errorsV2 "github.com/SigNoz/signoz/pkg/errors"
	"github.com/SigNoz/signoz/pkg/query-service/common"
	"github.com/SigNoz/signoz/pkg/query-service/constants"
	chErrors "github.com/SigNoz/signoz/pkg/query-service/errors"
	"github.com/SigNoz/signoz/pkg/query-service/interfaces"
	"github.com/SigNoz/signoz/pkg/query-service/model"
	"github.com/SigNoz/signoz/pkg/query-service/model/querytypes"
	"github.com/SigNoz/signoz/pkg/types/ctxtypes"
	"github.com/SigNoz/signoz/pkg/types/instrumentationtypes"
	"github.com/SigNoz/signoz/pkg/types/timeseriestypes"
)

const (
	defaultDatabase = "signoz_analytics"
	defaultTable    = "rule_state_history_v0"
)

type Config struct {
	Database string
	Table    string
}

func DefaultConfig() Config {
	return Config{Database: defaultDatabase, Table: defaultTable}
}

type Reader struct {
	db       clickhouse.Conn
	logger   *slog.Logger
	database string
	table    string
}

var (
	_ interfaces.RuleStateHistoryReader      = (*Reader)(nil)
	_ interfaces.RuleStateHistoryQueryReader = (*Reader)(nil)
)

func New(logger *slog.Logger, db clickhouse.Conn, config Config) *Reader {
	return &Reader{
		db:       db,
		logger:   logger,
		database: config.Database,
		table:    config.Table,
	}
}

// readRow maps a dynamically typed ClickHouse row into labels and a point.
func readRow(vars []interface{}, columnNames []string, countOfNumberCols int) ([]string, map[string]string, []map[string]string, *timeseriestypes.Point) {
	// Each row will have a value and a timestamp, and an optional list of label values
	// example: {Timestamp: ..., Value: ...}
	// The timestamp may also not present in some cases where the time series is reduced to single value
	var point timeseriestypes.Point

	// groupBy is a container to hold label values for the current point
	// example: ["frontend", "/fetch"]
	var groupBy []string

	var groupAttributesArray []map[string]string
	// groupAttributes is a container to hold the key-value pairs for the current
	// metric point.
	// example: {"serviceName": "frontend", "operation": "/fetch"}
	groupAttributes := make(map[string]string)

	isValidPoint := false

	for idx, v := range vars {
		colName := columnNames[idx]
		switch v := v.(type) {
		case *string:
			// special case for returning all labels in metrics datasource
			if colName == "fullLabels" {
				var metric map[string]string
				err := json.Unmarshal([]byte(*v), &metric)
				if err != nil {
					slog.Error("unexpected error encountered", errorsV2.Attr(err))
				}
				for key, val := range metric {
					groupBy = append(groupBy, val)
					if _, ok := groupAttributes[key]; !ok {
						groupAttributesArray = append(groupAttributesArray, map[string]string{key: val})
					}
					groupAttributes[key] = val
				}
			} else {
				groupBy = append(groupBy, *v)
				if _, ok := groupAttributes[colName]; !ok {
					groupAttributesArray = append(groupAttributesArray, map[string]string{colName: *v})
				}
				groupAttributes[colName] = *v
			}
		case *time.Time:
			point.Timestamp = v.UnixMilli()
		case *float64, *float32:
			if _, ok := constants.ReservedColumnTargetAliases[colName]; ok || countOfNumberCols == 1 {
				isValidPoint = true
				point.Value = float64(reflect.ValueOf(v).Elem().Float())
			} else {
				val := strconv.FormatFloat(reflect.ValueOf(v).Elem().Float(), 'f', -1, 64)
				groupBy = append(groupBy, val)
				if _, ok := groupAttributes[colName]; !ok {
					groupAttributesArray = append(groupAttributesArray, map[string]string{colName: val})
				}
				groupAttributes[colName] = val
			}
		case **float64, **float32:
			val := reflect.ValueOf(v)
			if val.IsValid() && !val.IsNil() && !val.Elem().IsNil() {
				value := reflect.ValueOf(v).Elem().Elem().Float()
				if _, ok := constants.ReservedColumnTargetAliases[colName]; ok || countOfNumberCols == 1 {
					isValidPoint = true
					point.Value = value
				} else {
					val := strconv.FormatFloat(value, 'f', -1, 64)
					groupBy = append(groupBy, val)
					if _, ok := groupAttributes[colName]; !ok {
						groupAttributesArray = append(groupAttributesArray, map[string]string{colName: val})
					}
					groupAttributes[colName] = val
				}
			}
		case *uint, *uint8, *uint64, *uint16, *uint32:
			if _, ok := constants.ReservedColumnTargetAliases[colName]; ok || countOfNumberCols == 1 {
				isValidPoint = true
				point.Value = float64(reflect.ValueOf(v).Elem().Uint())
			} else {
				groupBy = append(groupBy, fmt.Sprintf("%v", reflect.ValueOf(v).Elem().Uint()))
				if _, ok := groupAttributes[colName]; !ok {
					groupAttributesArray = append(groupAttributesArray, map[string]string{colName: fmt.Sprintf("%v", reflect.ValueOf(v).Elem().Uint())})
				}
				groupAttributes[colName] = fmt.Sprintf("%v", reflect.ValueOf(v).Elem().Uint())
			}
		case **uint, **uint8, **uint64, **uint16, **uint32:
			val := reflect.ValueOf(v)
			if val.IsValid() && !val.IsNil() && !val.Elem().IsNil() {
				value := reflect.ValueOf(v).Elem().Elem().Uint()
				if _, ok := constants.ReservedColumnTargetAliases[colName]; ok || countOfNumberCols == 1 {
					isValidPoint = true
					point.Value = float64(value)
				} else {
					groupBy = append(groupBy, fmt.Sprintf("%v", value))
					if _, ok := groupAttributes[colName]; !ok {
						groupAttributesArray = append(groupAttributesArray, map[string]string{colName: fmt.Sprintf("%v", value)})
					}
					groupAttributes[colName] = fmt.Sprintf("%v", value)
				}
			}
		case *int, *int8, *int16, *int32, *int64:
			if _, ok := constants.ReservedColumnTargetAliases[colName]; ok || countOfNumberCols == 1 {
				isValidPoint = true
				point.Value = float64(reflect.ValueOf(v).Elem().Int())
			} else {
				groupBy = append(groupBy, fmt.Sprintf("%v", reflect.ValueOf(v).Elem().Int()))
				if _, ok := groupAttributes[colName]; !ok {
					groupAttributesArray = append(groupAttributesArray, map[string]string{colName: fmt.Sprintf("%v", reflect.ValueOf(v).Elem().Int())})
				}
				groupAttributes[colName] = fmt.Sprintf("%v", reflect.ValueOf(v).Elem().Int())
			}
		case **int, **int8, **int16, **int32, **int64:
			val := reflect.ValueOf(v)
			if val.IsValid() && !val.IsNil() && !val.Elem().IsNil() {
				value := reflect.ValueOf(v).Elem().Elem().Int()
				if _, ok := constants.ReservedColumnTargetAliases[colName]; ok || countOfNumberCols == 1 {
					isValidPoint = true
					point.Value = float64(value)
				} else {
					groupBy = append(groupBy, fmt.Sprintf("%v", value))
					if _, ok := groupAttributes[colName]; !ok {
						groupAttributesArray = append(groupAttributesArray, map[string]string{colName: fmt.Sprintf("%v", value)})
					}
					groupAttributes[colName] = fmt.Sprintf("%v", value)
				}
			}
		case *bool:
			groupBy = append(groupBy, fmt.Sprintf("%v", *v))
			if _, ok := groupAttributes[colName]; !ok {
				groupAttributesArray = append(groupAttributesArray, map[string]string{colName: fmt.Sprintf("%v", *v)})
			}
			groupAttributes[colName] = fmt.Sprintf("%v", *v)

		default:
			slog.Error("unsupported var type found in query builder query result", "v", v, "colName", colName)
		}
	}
	if isValidPoint {
		return groupBy, groupAttributes, groupAttributesArray, &point
	}
	return groupBy, groupAttributes, groupAttributesArray, nil
}

func readRowsForTimeSeriesResult(rows driver.Rows, vars []interface{}, columnNames []string, countOfNumberCols int) ([]*timeseriestypes.Series, error) {
	// when groupBy is applied, each combination of cartesian product
	// of attribute values is a separate series. Each item in seriesToPoints
	// represent a unique series where the key is sorted attribute values joined
	// by "," and the value is the list of points for that series

	// For instance, group by (serviceName, operation)
	// with two services and three operations in each will result in (maximum of) 6 series
	// ("frontend", "order") x ("/fetch", "/fetch/{Id}", "/order")
	//
	// ("frontend", "/fetch")
	// ("frontend", "/fetch/{Id}")
	// ("frontend", "/order")
	// ("order", "/fetch")
	// ("order", "/fetch/{Id}")
	// ("order", "/order")
	seriesToPoints := make(map[string][]timeseriestypes.Point)
	var keys []string
	// seriesToAttrs is a mapping of key to a map of attribute key to attribute value
	// for each series. This is used to populate the series' attributes
	// For instance, for the above example, the seriesToAttrs will be
	// {
	//   "frontend,/fetch": {"serviceName": "frontend", "operation": "/fetch"},
	//   "frontend,/fetch/{Id}": {"serviceName": "frontend", "operation": "/fetch/{Id}"},
	//   "frontend,/order": {"serviceName": "frontend", "operation": "/order"},
	//   "order,/fetch": {"serviceName": "order", "operation": "/fetch"},
	//   "order,/fetch/{Id}": {"serviceName": "order", "operation": "/fetch/{Id}"},
	//   "order,/order": {"serviceName": "order", "operation": "/order"},
	// }
	seriesToAttrs := make(map[string]map[string]string)
	labelsArray := make(map[string][]map[string]string)
	for rows.Next() {
		if err := rows.Scan(vars...); err != nil {
			return nil, err
		}
		groupBy, groupAttributes, groupAttributesArray, metricPoint := readRow(vars, columnNames, countOfNumberCols)
		// skip the point if the value is NaN or Inf
		// are they ever useful enough to be returned?
		if metricPoint != nil && (math.IsNaN(metricPoint.Value) || math.IsInf(metricPoint.Value, 0)) {
			continue
		}
		sort.Strings(groupBy)
		key := strings.Join(groupBy, "")
		if _, exists := seriesToAttrs[key]; !exists {
			keys = append(keys, key)
		}
		seriesToAttrs[key] = groupAttributes
		labelsArray[key] = groupAttributesArray
		if metricPoint != nil {
			seriesToPoints[key] = append(seriesToPoints[key], *metricPoint)
		}
	}

	var seriesList []*timeseriestypes.Series
	for _, key := range keys {
		points := seriesToPoints[key]
		series := timeseriestypes.Series{Labels: seriesToAttrs[key], Points: points, LabelsArray: labelsArray[key]}
		seriesList = append(seriesList, &series)
	}
	return seriesList, getPersonalisedError(rows.Err())
}

func (r *Reader) getTimeSeriesResult(ctx context.Context, query string, args ...any) ([]*timeseriestypes.Series, error) {
	ctx = ctxtypes.NewContextWithCommentVals(ctx, map[string]string{
		instrumentationtypes.CodeNamespace:    "clickhouse-reader",
		instrumentationtypes.CodeFunctionName: "GetTimeSeriesResult",
	})
	rows, err := r.db.Query(ctx, query, args...)

	if err != nil {
		r.logger.Error("error while reading time series result", errorsV2.Attr(err))
		return nil, fmt.Errorf("query rule state history time series: %w", err)
	}
	defer rows.Close()

	var (
		columnTypes = rows.ColumnTypes()
		columnNames = rows.Columns()
		vars        = make([]interface{}, len(columnTypes))
	)
	var countOfNumberCols int

	for i := range columnTypes {
		vars[i] = reflect.New(columnTypes[i].ScanType()).Interface()
		switch columnTypes[i].ScanType().Kind() {
		case reflect.Float32,
			reflect.Float64,
			reflect.Uint,
			reflect.Uint8,
			reflect.Uint16,
			reflect.Uint32,
			reflect.Uint64,
			reflect.Int,
			reflect.Int8,
			reflect.Int16,
			reflect.Int32,
			reflect.Int64:
			countOfNumberCols++
		}
	}

	return readRowsForTimeSeriesResult(rows, vars, columnNames, countOfNumberCols)
}

func getPersonalisedError(err error) error {
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

func (r *Reader) AddRuleStateHistory(ctx context.Context, ruleStateHistory []model.RuleStateHistory) error {
	ctx = ctxtypes.NewContextWithCommentVals(ctx, map[string]string{
		instrumentationtypes.CodeNamespace:    "clickhouse-reader",
		instrumentationtypes.CodeFunctionName: "AddRuleStateHistory",
	})
	var statement driver.Batch
	var err error

	defer func() {
		if statement != nil {
			statement.Abort()
		}
	}()

	statement, err = r.db.PrepareBatch(ctx, fmt.Sprintf("INSERT INTO %s.%s (rule_id, rule_name, overall_state, overall_state_changed, state, state_changed, unix_milli, labels, fingerprint, value) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)",
		r.database, r.table))

	if err != nil {
		return err
	}

	for _, history := range ruleStateHistory {
		err = statement.Append(history.RuleID, history.RuleName, history.OverallState, history.OverallStateChanged, history.State, history.StateChanged, history.UnixMilli, history.Labels, history.Fingerprint, history.Value)
		if err != nil {
			return err
		}
	}

	err = statement.Send()
	if err != nil {
		return err
	}
	return nil
}

func (r *Reader) GetLastSavedRuleStateHistory(ctx context.Context, ruleID string) ([]model.RuleStateHistory, error) {
	ctx = ctxtypes.NewContextWithCommentVals(ctx, map[string]string{
		instrumentationtypes.CodeNamespace:    "clickhouse-reader",
		instrumentationtypes.CodeFunctionName: "GetLastSavedRuleStateHistory",
	})
	query := fmt.Sprintf("SELECT * FROM %s.%s WHERE rule_id = ? AND state_changed = true ORDER BY unix_milli DESC LIMIT 1 BY fingerprint",
		r.database, r.table)

	history := []model.RuleStateHistory{}
	err := r.db.Select(ctx, &history, query, ruleID)
	if err != nil {
		return nil, err
	}
	return history, nil
}

func ruleStateHistoryOrder(order string) (string, error) {
	switch strings.ToLower(strings.TrimSpace(order)) {
	case "asc":
		return "ASC", nil
	case "desc":
		return "DESC", nil
	default:
		return "", fmt.Errorf("order must be asc or desc")
	}
}

func buildRuleStateHistoryConditions(ruleID string, params *model.QueryRuleStateHistory) (string, []any, error) {
	conditions := []string{
		"rule_id = ?",
		"unix_milli >= ? AND unix_milli < ?",
	}
	args := []any{ruleID, params.Start, params.End}

	if params.State != "" {
		conditions = append(conditions, "state = ?")
		args = append(args, params.State)
	}

	if params.Filters == nil || len(params.Filters.Items) == 0 {
		return strings.Join(conditions, " AND "), args, nil
	}

	for _, item := range params.Filters.Items {
		value := item.Value
		op := querytypes.FilterOperator(strings.ToLower(strings.TrimSpace(string(item.Operator))))
		if op == querytypes.FilterOperatorContains || op == querytypes.FilterOperatorNotContains {
			value = fmt.Sprintf("%%%s%%", value)
		}
		label := "JSONExtractString(labels, ?)"
		args = append(args, item.Key.Key)

		switch op {
		case querytypes.FilterOperatorEqual:
			conditions = append(conditions, label+" = ?")
		case querytypes.FilterOperatorNotEqual:
			conditions = append(conditions, label+" != ?")
		case querytypes.FilterOperatorIn:
			conditions = append(conditions, label+" IN ?")
		case querytypes.FilterOperatorNotIn:
			conditions = append(conditions, label+" NOT IN ?")
		case querytypes.FilterOperatorLike, querytypes.FilterOperatorContains:
			conditions = append(conditions, "like("+label+", ?)")
		case querytypes.FilterOperatorNotLike, querytypes.FilterOperatorNotContains:
			conditions = append(conditions, "notLike("+label+", ?)")
		case querytypes.FilterOperatorRegex:
			conditions = append(conditions, "match("+label+", ?)")
		case querytypes.FilterOperatorNotRegex:
			conditions = append(conditions, "not match("+label+", ?)")
		case querytypes.FilterOperatorGreaterThan:
			conditions = append(conditions, label+" > ?")
		case querytypes.FilterOperatorGreaterThanOrEq:
			conditions = append(conditions, label+" >= ?")
		case querytypes.FilterOperatorLessThan:
			conditions = append(conditions, label+" < ?")
		case querytypes.FilterOperatorLessThanOrEq:
			conditions = append(conditions, label+" <= ?")
		case querytypes.FilterOperatorExists:
			conditions = append(conditions, "has(JSONExtractKeys(labels), ?)")
			continue
		case querytypes.FilterOperatorNotExists:
			conditions = append(conditions, "not has(JSONExtractKeys(labels), ?)")
			continue
		default:
			return "", nil, fmt.Errorf("unsupported filter operator")
		}
		args = append(args, value)
	}

	return strings.Join(conditions, " AND "), args, nil
}

func (r *Reader) ReadRuleStateHistoryByRuleID(
	ctx context.Context, ruleID string, params *model.QueryRuleStateHistory) (*model.RuleStateTimeline, error) {

	ctx = ctxtypes.NewContextWithCommentVals(ctx, map[string]string{
		instrumentationtypes.CodeNamespace:    "clickhouse-reader",
		instrumentationtypes.CodeFunctionName: "ReadRuleStateHistoryByRuleID",
	})
	whereClause, args, err := buildRuleStateHistoryConditions(ruleID, params)
	if err != nil {
		return nil, err
	}
	order, err := ruleStateHistoryOrder(params.Order)
	if err != nil {
		return nil, err
	}

	query := fmt.Sprintf("SELECT * FROM %s.%s WHERE %s ORDER BY unix_milli %s LIMIT %d OFFSET %d",
		r.database, r.table, whereClause, order, params.Limit, params.Offset)

	history := []model.RuleStateHistory{}
	r.logger.Debug("rule state history query", "query", query)
	err = r.db.Select(ctx, &history, query, args...)
	if err != nil {
		r.logger.Error("Error while reading rule state history", errorsV2.Attr(err))
		return nil, err
	}

	var total uint64
	r.logger.Debug("rule state history total query", "query", fmt.Sprintf("SELECT count(*) FROM %s.%s WHERE %s",
		r.database, r.table, whereClause))
	err = r.db.QueryRow(ctx, fmt.Sprintf("SELECT count(*) FROM %s.%s WHERE %s",
		r.database, r.table, whereClause), args...).Scan(&total)
	if err != nil {
		return nil, err
	}

	labelsQuery := fmt.Sprintf("SELECT DISTINCT labels FROM %s.%s WHERE rule_id = $1",
		r.database, r.table)
	rows, err := r.db.Query(ctx, labelsQuery, ruleID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	labelsMap := make(map[string][]string)
	for rows.Next() {
		var rawLabel string
		err = rows.Scan(&rawLabel)
		if err != nil {
			return nil, err
		}
		label := map[string]string{}
		err = json.Unmarshal([]byte(rawLabel), &label)
		if err != nil {
			return nil, err
		}
		for k, v := range label {
			labelsMap[k] = append(labelsMap[k], v)
		}
	}

	timeline := &model.RuleStateTimeline{
		Items:  history,
		Total:  total,
		Labels: labelsMap,
	}

	return timeline, nil
}

func (r *Reader) ReadRuleStateHistoryTopContributorsByRuleID(
	ctx context.Context, ruleID string, params *model.QueryRuleStateHistory) ([]model.RuleStateHistoryContributor, error) {
	ctx = ctxtypes.NewContextWithCommentVals(ctx, map[string]string{
		instrumentationtypes.CodeNamespace:    "clickhouse-reader",
		instrumentationtypes.CodeFunctionName: "ReadRuleStateHistoryTopContributorsByRuleID",
	})
	query := fmt.Sprintf(`SELECT
		fingerprint,
		any(labels) as labels,
		count(*) as count
	FROM %s.%s
	WHERE rule_id = ? AND (state_changed = true) AND (state = ?) AND unix_milli >= ? AND unix_milli <= ?
	GROUP BY fingerprint
	HAVING labels != '{}'
	ORDER BY count DESC`,
		r.database, r.table)

	r.logger.Debug("rule state history top contributors query", "query", query)
	contributors := []model.RuleStateHistoryContributor{}
	err := r.db.Select(ctx, &contributors, query, ruleID, model.StateFiring.String(), params.Start, params.End)
	if err != nil {
		r.logger.Error("Error while reading rule state history", errorsV2.Attr(err))
		return nil, err
	}

	return contributors, nil
}

func (r *Reader) GetOverallStateTransitions(ctx context.Context, ruleID string, params *model.QueryRuleStateHistory) ([]model.ReleStateItem, error) {

	ctx = ctxtypes.NewContextWithCommentVals(ctx, map[string]string{
		instrumentationtypes.CodeNamespace:    "clickhouse-reader",
		instrumentationtypes.CodeFunctionName: "GetOverallStateTransitions",
	})
	tmpl := `WITH firing_events AS (
    SELECT
        rule_id,
        state,
        unix_milli AS firing_time
    FROM %s.%s
    WHERE overall_state = ?
      AND overall_state_changed = true
	  AND rule_id = ?
	  AND unix_milli >= ? AND unix_milli <= ?
),
resolution_events AS (
    SELECT
        rule_id,
        state,
        unix_milli AS resolution_time
    FROM %s.%s
    WHERE overall_state = ?
      AND overall_state_changed = true
	  AND rule_id = ?
	  AND unix_milli >= ? AND unix_milli <= ?
),
matched_events AS (
    SELECT
        f.rule_id,
        f.state,
        f.firing_time,
        MIN(r.resolution_time) AS resolution_time
    FROM firing_events f
    LEFT JOIN resolution_events r
        ON f.rule_id = r.rule_id
    WHERE r.resolution_time > f.firing_time
    GROUP BY f.rule_id, f.state, f.firing_time
)
SELECT *
FROM matched_events
ORDER BY firing_time ASC;`

	query := fmt.Sprintf(tmpl, r.database, r.table, r.database, r.table)
	args := []any{
		model.StateFiring.String(), ruleID, params.Start, params.End,
		model.StateInactive.String(), ruleID, params.Start, params.End,
	}

	r.logger.Debug("overall state transitions query", "query", query)

	transitions := []model.RuleStateTransition{}
	err := r.db.Select(ctx, &transitions, query, args...)
	if err != nil {
		return nil, err
	}

	stateItems := []model.ReleStateItem{}

	for idx, item := range transitions {
		start := item.FiringTime
		end := item.ResolutionTime
		stateItems = append(stateItems, model.ReleStateItem{
			State: item.State,
			Start: start,
			End:   end,
		})
		if idx < len(transitions)-1 {
			nextStart := transitions[idx+1].FiringTime
			if nextStart > end {
				stateItems = append(stateItems, model.ReleStateItem{
					State: model.StateInactive,
					Start: end,
					End:   nextStart,
				})
			}
		}
	}

	// fetch the most recent overall_state from the table
	var state model.AlertState
	stateQuery := fmt.Sprintf("SELECT state FROM %s.%s WHERE rule_id = ? AND unix_milli <= ? ORDER BY unix_milli DESC LIMIT 1",
		r.database, r.table)
	if err := r.db.QueryRow(ctx, stateQuery, ruleID, params.End).Scan(&state); err != nil {
		if err != sql.ErrNoRows {
			return nil, err
		}
		state = model.StateInactive
	}

	if len(transitions) == 0 {
		// no transitions found, it is either firing or inactive for whole time range
		stateItems = append(stateItems, model.ReleStateItem{
			State: state,
			Start: params.Start,
			End:   params.End,
		})
	} else {
		// there were some transitions, we need to add the last state at the end
		if state == model.StateInactive {
			stateItems = append(stateItems, model.ReleStateItem{
				State: model.StateInactive,
				Start: transitions[len(transitions)-1].ResolutionTime,
				End:   params.End,
			})
		} else {
			// fetch the most recent firing event from the table in the given time range
			var firingTime int64
			firingQuery := fmt.Sprintf(`
			SELECT
				unix_milli
			FROM %s.%s
			WHERE rule_id = ? AND overall_state_changed = true AND overall_state = ? AND unix_milli <= ?
			ORDER BY unix_milli DESC LIMIT 1`, r.database, r.table)
			if err := r.db.QueryRow(ctx, firingQuery, ruleID, model.StateFiring.String(), params.End).Scan(&firingTime); err != nil {
				return nil, err
			}
			stateItems = append(stateItems, model.ReleStateItem{
				State: model.StateInactive,
				Start: transitions[len(transitions)-1].ResolutionTime,
				End:   firingTime,
			})
			stateItems = append(stateItems, model.ReleStateItem{
				State: model.StateFiring,
				Start: firingTime,
				End:   params.End,
			})
		}
	}
	return stateItems, nil
}

func (r *Reader) GetAvgResolutionTime(ctx context.Context, ruleID string, params *model.QueryRuleStateHistory) (float64, error) {

	ctx = ctxtypes.NewContextWithCommentVals(ctx, map[string]string{
		instrumentationtypes.CodeNamespace:    "clickhouse-reader",
		instrumentationtypes.CodeFunctionName: "GetAvgResolutionTime",
	})
	tmpl := `
WITH firing_events AS (
    SELECT
        rule_id,
        state,
        unix_milli AS firing_time
    FROM %s.%s
    WHERE overall_state = ?
      AND overall_state_changed = true
	  AND rule_id = ?
	  AND unix_milli >= ? AND unix_milli <= ?
),
resolution_events AS (
    SELECT
        rule_id,
        state,
        unix_milli AS resolution_time
    FROM %s.%s
    WHERE overall_state = ?
      AND overall_state_changed = true
	  AND rule_id = ?
	  AND unix_milli >= ? AND unix_milli <= ?
),
matched_events AS (
    SELECT
        f.rule_id,
        f.state,
        f.firing_time,
        MIN(r.resolution_time) AS resolution_time
    FROM firing_events f
    LEFT JOIN resolution_events r
        ON f.rule_id = r.rule_id
    WHERE r.resolution_time > f.firing_time
    GROUP BY f.rule_id, f.state, f.firing_time
)
SELECT AVG(resolution_time - firing_time) / 1000 AS avg_resolution_time
FROM matched_events;
`

	query := fmt.Sprintf(tmpl, r.database, r.table, r.database, r.table)
	args := []any{
		model.StateFiring.String(), ruleID, params.Start, params.End,
		model.StateInactive.String(), ruleID, params.Start, params.End,
	}

	r.logger.Debug("avg resolution time query", "query", query)
	var avgResolutionTime float64
	err := r.db.QueryRow(ctx, query, args...).Scan(&avgResolutionTime)
	if err != nil {
		return 0, err
	}

	return avgResolutionTime, nil
}

func (r *Reader) GetAvgResolutionTimeByInterval(ctx context.Context, ruleID string, params *model.QueryRuleStateHistory) (*timeseriestypes.Series, error) {

	step := common.MinAllowedStepInterval(params.Start, params.End)

	tmpl := `
WITH firing_events AS (
    SELECT
        rule_id,
        state,
        unix_milli AS firing_time
    FROM %s.%s
    WHERE overall_state = ?
      AND overall_state_changed = true
	  AND rule_id = ?
	  AND unix_milli >= ? AND unix_milli <= ?
),
resolution_events AS (
    SELECT
        rule_id,
        state,
        unix_milli AS resolution_time
    FROM %s.%s
    WHERE overall_state = ?
      AND overall_state_changed = true
	  AND rule_id = ?
	  AND unix_milli >= ? AND unix_milli <= ?
),
matched_events AS (
    SELECT
        f.rule_id,
        f.state,
        f.firing_time,
        MIN(r.resolution_time) AS resolution_time
    FROM firing_events f
    LEFT JOIN resolution_events r
        ON f.rule_id = r.rule_id
    WHERE r.resolution_time > f.firing_time
    GROUP BY f.rule_id, f.state, f.firing_time
)
SELECT toStartOfInterval(toDateTime(firing_time / 1000), INTERVAL %d SECOND) AS ts, AVG(resolution_time - firing_time) / 1000 AS avg_resolution_time
FROM matched_events
GROUP BY ts
ORDER BY ts ASC;`

	query := fmt.Sprintf(tmpl, r.database, r.table, r.database, r.table, step)
	args := []any{
		model.StateFiring.String(), ruleID, params.Start, params.End,
		model.StateInactive.String(), ruleID, params.Start, params.End,
	}

	r.logger.Debug("avg resolution time by interval query", "query", query)
	result, err := r.getTimeSeriesResult(ctx, query, args...)
	if err != nil || len(result) == 0 {
		return nil, err
	}

	return result[0], nil
}
