package v3

import (
	"fmt"
	"strings"

	"github.com/SigNoz/signoz/pkg/query-service/constants"
	v3 "github.com/SigNoz/signoz/pkg/query-service/model/v3"
	"github.com/SigNoz/signoz/pkg/query-service/utils"
)

var AggregateOperatorToPercentile = map[v3.AggregateOperator]float64{
	v3.AggregateOperatorP05: 0.05,
	v3.AggregateOperatorP10: 0.10,
	v3.AggregateOperatorP20: 0.20,
	v3.AggregateOperatorP25: 0.25,
	v3.AggregateOperatorP50: 0.50,
	v3.AggregateOperatorP75: 0.75,
	v3.AggregateOperatorP90: 0.90,
	v3.AggregateOperatorP95: 0.95,
	v3.AggregateOperatorP99: 0.99,
}

var AggregateOperatorToSQLFunc = map[v3.AggregateOperator]string{
	v3.AggregateOperatorAvg:     "avg",
	v3.AggregateOperatorMax:     "max",
	v3.AggregateOperatorMin:     "min",
	v3.AggregateOperatorSum:     "sum",
	v3.AggregateOperatorRate:    "count",
	v3.AggregateOperatorRateSum: "sum",
	v3.AggregateOperatorRateAvg: "avg",
	v3.AggregateOperatorRateMax: "max",
	v3.AggregateOperatorRateMin: "min",
}

var logOperators = map[v3.FilterOperator]string{
	v3.FilterOperatorEqual:           "=",
	v3.FilterOperatorNotEqual:        "!=",
	v3.FilterOperatorLessThan:        "<",
	v3.FilterOperatorLessThanOrEq:    "<=",
	v3.FilterOperatorGreaterThan:     ">",
	v3.FilterOperatorGreaterThanOrEq: ">=",
	v3.FilterOperatorLike:            "ILIKE",
	v3.FilterOperatorNotLike:         "NOT ILIKE",
	v3.FilterOperatorContains:        "ILIKE",
	v3.FilterOperatorNotContains:     "NOT ILIKE",
	v3.FilterOperatorRegex:           "match(%s, %s)",
	v3.FilterOperatorNotRegex:        "NOT match(%s, %s)",
	v3.FilterOperatorIn:              "IN",
	v3.FilterOperatorNotIn:           "NOT IN",
	v3.FilterOperatorExists:          "has(%s_%s_key, '%s')",
	v3.FilterOperatorNotExists:       "not has(%s_%s_key, '%s')",
}

const BODY = "body"

func GetClickhouseLogsColumnType(columnType v3.AttributeKeyType) string {
	if columnType == v3.AttributeKeyTypeTag {
		return "attributes"
	}
	if columnType == v3.AttributeKeyTypeInstrumentationScope {
		return "scope"
	}
	return "resources"
}

func getClickhouseLogsColumnDataType(columnDataType v3.AttributeKeyDataType) string {
	if columnDataType == v3.AttributeKeyDataTypeFloat64 {
		return "float64"
	}
	if columnDataType == v3.AttributeKeyDataTypeInt64 {
		return "int64"
	}
	if columnDataType == v3.AttributeKeyDataTypeBool {
		return "bool"
	}
	return "string"
}

// getClickhouseColumnName returns the corresponding clickhouse column name for the given attribute/resource key
func getClickhouseColumnName(key v3.AttributeKey) string {
	clickhouseColumn := key.Key
	if key.Key == constants.TIMESTAMP || key.Key == "id" {
		return key.Key
	}

	//if the key is present in the topLevelColumn then it will be only searched in those columns,
	//regardless if it is indexed/present again in resource or column attribute
	if !key.IsColumn {
		columnType := GetClickhouseLogsColumnType(key.Type)
		columnDataType := getClickhouseLogsColumnDataType(key.DataType)
		clickhouseColumn = fmt.Sprintf("%s_%s_value[indexOf(%s_%s_key, '%s')]", columnType, columnDataType, columnType, columnDataType, key.Key)
		return clickhouseColumn
	}

	// check if it is a static field
	if key.Type == v3.AttributeKeyTypeUnspecified {
		// name is the column name
		return clickhouseColumn
	}

	// materialized column created from query
	clickhouseColumn = utils.GetClickhouseColumnName(string(key.Type), string(key.DataType), key.Key)
	return clickhouseColumn
}

// getSelectLabels returns the select labels for the query based on groupBy and aggregateOperator
func getSelectLabels(aggregatorOperator v3.AggregateOperator, groupBy []v3.AttributeKey) string {
	var selectLabels string
	if aggregatorOperator == v3.AggregateOperatorNoOp {
		selectLabels = ""
	} else {
		for _, tag := range groupBy {
			columnName := getClickhouseColumnName(tag)
			selectLabels += fmt.Sprintf(" %s as `%s`,", columnName, tag.Key)
		}
	}
	return selectLabels
}

func GetSelectKeys(aggregatorOperator v3.AggregateOperator, groupBy []v3.AttributeKey) string {
	var selectLabels []string
	if aggregatorOperator == v3.AggregateOperatorNoOp {
		return ""
	} else {
		for _, tag := range groupBy {
			selectLabels = append(selectLabels, "`"+tag.Key+"`")
		}
	}
	return strings.Join(selectLabels, ",")
}

func GetExistsNexistsFilter(op v3.FilterOperator, item v3.FilterItem) string {
	if item.Key.Type == v3.AttributeKeyTypeUnspecified {
		top := "!="
		if op == v3.FilterOperatorNotExists {
			top = "="
		}
		if val, ok := constants.StaticFieldsLogsV3[item.Key.Key]; ok {
			// skip for timestamp and id
			if val.Key == "" {
				return ""
			}

			columnName := getClickhouseColumnName(item.Key)
			if val.DataType == v3.AttributeKeyDataTypeString {
				return fmt.Sprintf("%s %s ''", columnName, top)
			} else {
				// we just have two types, number and string
				return fmt.Sprintf("%s %s 0", columnName, top)
			}
		}

	} else if item.Key.IsColumn {
		val := true
		if op == v3.FilterOperatorNotExists {
			val = false
		}
		return fmt.Sprintf("%s_exists`=%v", strings.TrimSuffix(getClickhouseColumnName(item.Key), "`"), val)
	}
	columnType := GetClickhouseLogsColumnType(item.Key.Type)
	columnDataType := getClickhouseLogsColumnDataType(item.Key.DataType)
	return fmt.Sprintf(logOperators[op], columnType, columnDataType, item.Key.Key)
}

func groupBy(panelType v3.PanelType, graphLimitQtype string, tags ...string) string {
	if (graphLimitQtype != constants.FirstQueryGraphLimit) && (panelType == v3.PanelTypeGraph || panelType == v3.PanelTypeValue) {
		tags = append(tags, "ts")
	}
	return strings.Join(tags, ",")
}

func GroupByAttributeKeyTags(panelType v3.PanelType, graphLimitQtype string, tags ...v3.AttributeKey) string {
	groupTags := []string{}
	for _, tag := range tags {
		groupTags = append(groupTags, "`"+tag.Key+"`")
	}
	return groupBy(panelType, graphLimitQtype, groupTags...)
}

// orderBy returns a string of comma separated tags for order by clause
// if there are remaining items which are not present in tags they are also added
// if the order is not specified, it defaults to ASC
func Having(items []v3.Having) string {
	// aggregate something and filter on that aggregate
	var having []string
	for _, item := range items {
		having = append(having, fmt.Sprintf("value %s %s", item.Operator, utils.ClickHouseFormattedValue(item.Value)))
	}
	return strings.Join(having, " AND ")
}

func ReduceQuery(query string, reduceTo v3.ReduceToOperator, aggregateOperator v3.AggregateOperator) (string, error) {
	// the timestamp picked is not relevant here since the final value used is show the single
	// chart with just the query value.
	switch reduceTo {
	case v3.ReduceToOperatorLast:
		query = fmt.Sprintf("SELECT anyLast(value) as value, now() as ts FROM (%s)", query)
	case v3.ReduceToOperatorSum:
		query = fmt.Sprintf("SELECT sum(value) as value, now() as ts FROM (%s)", query)
	case v3.ReduceToOperatorAvg:
		query = fmt.Sprintf("SELECT avg(value) as value, now() as ts FROM (%s)", query)
	case v3.ReduceToOperatorMax:
		query = fmt.Sprintf("SELECT max(value) as value, now() as ts FROM (%s)", query)
	case v3.ReduceToOperatorMin:
		query = fmt.Sprintf("SELECT min(value) as value, now() as ts FROM (%s)", query)
	default:
		return "", fmt.Errorf("unsupported reduce operator")
	}
	return query, nil
}

func AddLimitToQuery(query string, limit uint64) string {
	if limit == 0 {
		return query
	}
	return fmt.Sprintf("%s LIMIT %d", query, limit)
}

func AddOffsetToQuery(query string, offset uint64) string {
	return fmt.Sprintf("%s OFFSET %d", query, offset)
}

func IsOrderByTs(orderBy []v3.OrderBy) bool {
	if len(orderBy) == 1 && (orderBy[0].Key == constants.TIMESTAMP || orderBy[0].ColumnName == constants.TIMESTAMP) {
		return true
	}
	return false
}
