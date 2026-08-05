package utils

import (
	"fmt"
	"strings"
	"time"

	"github.com/SigNoz/signoz/pkg/query-service/constants"
	"github.com/SigNoz/signoz/pkg/query-service/model/metrics_explorer"
	"github.com/SigNoz/signoz/pkg/query-service/model/querytypes"
)

// skipKey is an optional parameter to skip processing of a specific key
func BuildFilterConditions(fs *querytypes.FilterSet, skipKey string) ([]string, error) {
	if fs == nil || len(fs.Items) == 0 {
		return nil, nil
	}

	var conditions []string

	for _, item := range fs.Items {
		if skipKey != "" && item.Key.Key == skipKey {
			continue
		}

		toFormat := item.Value
		op := querytypes.FilterOperator(strings.ToLower(strings.TrimSpace(string(item.Operator))))
		if op == querytypes.FilterOperatorContains || op == querytypes.FilterOperatorNotContains {
			toFormat = fmt.Sprintf("%%%s%%", toFormat)
		}
		fmtVal := ClickHouseFormattedValue(toFormat)

		// Determine if the key is a JSON key or a normal column
		isJSONKey := false
		if _, exists := metrics_explorer.AvailableColumnFilterMap[item.Key.Key]; exists {
			isJSONKey = false
		} else {
			isJSONKey = true
		}

		condition, err := buildSingleFilterCondition(item.Key.Key, op, fmtVal, isJSONKey)
		if err != nil {
			return nil, err
		}
		conditions = append(conditions, condition)
	}

	return conditions, nil
}

func buildSingleFilterCondition(key string, op querytypes.FilterOperator, fmtVal string, isJSONKey bool) (string, error) {
	var keyCondition string
	if isJSONKey {
		keyCondition = fmt.Sprintf("JSONExtractString(labels, '%s')", key)
	} else { // Assuming normal column access
		if key == "metric_unit" {
			key = "unit"
		}
		if key == "metric_type" {
			key = "type"
		}
		keyCondition = key
	}

	switch op {
	case querytypes.FilterOperatorEqual:
		return fmt.Sprintf("%s = %s", keyCondition, fmtVal), nil
	case querytypes.FilterOperatorNotEqual:
		return fmt.Sprintf("%s != %s", keyCondition, fmtVal), nil
	case querytypes.FilterOperatorIn:
		return fmt.Sprintf("%s IN %s", keyCondition, fmtVal), nil
	case querytypes.FilterOperatorNotIn:
		return fmt.Sprintf("%s NOT IN %s", keyCondition, fmtVal), nil
	case querytypes.FilterOperatorLike:
		return fmt.Sprintf("like(%s, %s)", keyCondition, fmtVal), nil
	case querytypes.FilterOperatorNotLike:
		return fmt.Sprintf("notLike(%s, %s)", keyCondition, fmtVal), nil
	case querytypes.FilterOperatorRegex:
		return fmt.Sprintf("match(%s, %s)", keyCondition, fmtVal), nil
	case querytypes.FilterOperatorNotRegex:
		return fmt.Sprintf("not match(%s, %s)", keyCondition, fmtVal), nil
	case querytypes.FilterOperatorGreaterThan:
		return fmt.Sprintf("%s > %s", keyCondition, fmtVal), nil
	case querytypes.FilterOperatorGreaterThanOrEq:
		return fmt.Sprintf("%s >= %s", keyCondition, fmtVal), nil
	case querytypes.FilterOperatorLessThan:
		return fmt.Sprintf("%s < %s", keyCondition, fmtVal), nil
	case querytypes.FilterOperatorLessThanOrEq:
		return fmt.Sprintf("%s <= %s", keyCondition, fmtVal), nil
	case querytypes.FilterOperatorContains:
		return fmt.Sprintf("ilike(%s, %s)", keyCondition, fmtVal), nil
	case querytypes.FilterOperatorNotContains:
		return fmt.Sprintf("notILike(%s, %s)", keyCondition, fmtVal), nil
	case querytypes.FilterOperatorExists:
		return fmt.Sprintf("has(JSONExtractKeys(labels), '%s')", key), nil
	case querytypes.FilterOperatorNotExists:
		return fmt.Sprintf("not has(JSONExtractKeys(labels), '%s')", key), nil
	default:
		return "", fmt.Errorf("unsupported filter operator: %s", op)
	}
}

var (
	sixHoursInMilliseconds = time.Hour.Milliseconds() * 6
	oneDayInMilliseconds   = time.Hour.Milliseconds() * 24
	oneWeekInMilliseconds  = oneDayInMilliseconds * 7
)

func WhichTSTableToUse(start, end int64) (int64, int64, string, string) {

	var tableName string
	var localTableName string
	if end-start < sixHoursInMilliseconds {
		// adjust the start time to nearest 1 hour
		start = start - (start % (time.Hour.Milliseconds() * 1))
		tableName = constants.SIGINSIGHT_TIMESERIES_v4_TABLENAME
		localTableName = constants.SIGINSIGHT_TIMESERIES_v4_LOCAL_TABLENAME
	} else if end-start < oneDayInMilliseconds {
		// adjust the start time to nearest 6 hours
		start = start - (start % (time.Hour.Milliseconds() * 6))
		tableName = constants.SIGINSIGHT_TIMESERIES_v4_6HRS_TABLENAME
		localTableName = constants.SIGINSIGHT_TIMESERIES_v4_6HRS_LOCAL_TABLENAME
	} else if end-start < oneWeekInMilliseconds {
		// adjust the start time to nearest 1 day
		start = start - (start % (time.Hour.Milliseconds() * 24))
		tableName = constants.SIGINSIGHT_TIMESERIES_v4_1DAY_TABLENAME
		localTableName = constants.SIGINSIGHT_TIMESERIES_v4_1DAY_LOCAL_TABLENAME
	} else {
		// adjust the start time to nearest 1 week
		start = start - (start % (time.Hour.Milliseconds() * 24 * 7))
		tableName = constants.SIGINSIGHT_TIMESERIES_v4_1WEEK_TABLENAME
		localTableName = constants.SIGINSIGHT_TIMESERIES_v4_1WEEK_LOCAL_TABLENAME
	}

	return start, end, tableName, localTableName
}

func WhichSampleTableToUse(start, end int64) (string, string) {
	if end-start < oneDayInMilliseconds {
		return constants.SIGINSIGHT_SAMPLES_V4_TABLENAME, "count(*)"
	} else if end-start < oneWeekInMilliseconds {
		return constants.SIGINSIGHT_SAMPLES_V4_AGG_5M_TABLENAME, "sum(count)"
	} else {
		return constants.SIGINSIGHT_SAMPLES_V4_AGG_30M_TABLENAME, "sum(count)"
	}
}
