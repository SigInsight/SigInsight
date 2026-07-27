package tracefunnel

import (
	"fmt"
	"strings"

	"github.com/SigNoz/signoz/pkg/errors"
	"github.com/SigNoz/signoz/pkg/query-service/constants"
	"github.com/SigNoz/signoz/pkg/query-service/model/querytypes"
	"github.com/SigNoz/signoz/pkg/query-service/utils"
)

var traceFilterOperators = map[querytypes.FilterOperator]string{
	querytypes.FilterOperatorIn:              "IN",
	querytypes.FilterOperatorNotIn:           "NOT IN",
	querytypes.FilterOperatorEqual:           "=",
	querytypes.FilterOperatorNotEqual:        "!=",
	querytypes.FilterOperatorLessThan:        "<",
	querytypes.FilterOperatorLessThanOrEq:    "<=",
	querytypes.FilterOperatorGreaterThan:     ">",
	querytypes.FilterOperatorGreaterThanOrEq: ">=",
	querytypes.FilterOperatorLike:            "ILIKE",
	querytypes.FilterOperatorNotLike:         "NOT ILIKE",
	querytypes.FilterOperatorRegex:           "match(%s, %s)",
	querytypes.FilterOperatorNotRegex:        "NOT match(%s, %s)",
	querytypes.FilterOperatorContains:        "ILIKE",
	querytypes.FilterOperatorNotContains:     "NOT ILIKE",
	querytypes.FilterOperatorExists:          "mapContains(%s, '%s')",
	querytypes.FilterOperatorNotExists:       "NOT mapContains(%s, '%s')",
	querytypes.FilterOperatorILike:           "ILIKE",
	querytypes.FilterOperatorNotILike:        "NOT ILIKE",
}

func traceFilterColumnType(columnType querytypes.AttributeKeyType) string {
	if columnType == querytypes.AttributeKeyTypeResource {
		return "resources"
	}
	return "attributes"
}

func traceFilterColumnDataType(columnDataType querytypes.AttributeKeyDataType) string {
	if columnDataType == querytypes.AttributeKeyDataTypeFloat64 || columnDataType == querytypes.AttributeKeyDataTypeInt64 {
		return "number"
	}
	if columnDataType == querytypes.AttributeKeyDataTypeBool {
		return "bool"
	}
	return "string"
}

func traceFilterColumnName(key querytypes.AttributeKey) string {
	if _, ok := constants.StaticFieldsTraces[key.Key]; ok {
		return key.Key
	}

	if !key.IsColumn {
		return fmt.Sprintf(
			"%s_%s['%s']",
			traceFilterColumnType(key.Type),
			traceFilterColumnDataType(key.DataType),
			key.Key,
		)
	}

	return "`" + utils.GetClickhouseColumnNameV2(string(key.Type), string(key.DataType), key.Key) + "`"
}

func traceFilterFixedColumnExists(key querytypes.AttributeKey, operator querytypes.FilterOperator) (string, error) {
	if key.DataType != querytypes.AttributeKeyDataTypeString {
		return "", errors.New(errors.TypeInvalidInput, errors.CodeInvalidInput, "exists and not exists only support custom attributes or string columns")
	}
	if operator == querytypes.FilterOperatorExists {
		return fmt.Sprintf("%s != ''", traceFilterColumnName(key)), nil
	}
	return fmt.Sprintf("%s = ''", traceFilterColumnName(key)), nil
}

func buildTracesFilterQuery(filterSet *querytypes.FilterSet) (string, error) {
	if filterSet == nil || len(filterSet.Items) == 0 {
		return "", nil
	}

	conditions := make([]string, 0, len(filterSet.Items))
	for _, item := range filterSet.Items {
		item.Operator = querytypes.FilterOperator(strings.ToLower(strings.TrimSpace(string(item.Operator))))
		operator, ok := traceFilterOperators[item.Operator]
		if !ok {
			return "", errors.Newf(errors.TypeInvalidInput, errors.CodeInvalidInput, "unsupported operator %s", item.Operator)
		}

		columnName := traceFilterColumnName(item.Key)
		value := item.Value
		var formattedValue string
		if item.Operator != querytypes.FilterOperatorExists && item.Operator != querytypes.FilterOperatorNotExists {
			var err error
			value, err = utils.ValidateAndCastValue(value, item.Key.DataType)
			if err != nil {
				return "", errors.Newf(errors.TypeInvalidInput, errors.CodeInvalidInput, "invalid value for key %s: %v", item.Key.Key, err)
			}
			if value != nil {
				formattedValue = utils.ClickHouseFormattedValue(value)
			}
		}

		switch item.Operator {
		case querytypes.FilterOperatorContains, querytypes.FilterOperatorNotContains:
			escaped := utils.QuoteEscapedStringForContains(fmt.Sprintf("%s", item.Value), false)
			conditions = append(conditions, fmt.Sprintf("%s %s '%%%s%%'", columnName, operator, escaped))
		case querytypes.FilterOperatorRegex, querytypes.FilterOperatorNotRegex:
			conditions = append(conditions, fmt.Sprintf(operator, columnName, formattedValue))
		case querytypes.FilterOperatorExists, querytypes.FilterOperatorNotExists:
			if item.Key.IsColumn {
				condition, err := traceFilterFixedColumnExists(item.Key, item.Operator)
				if err != nil {
					return "", err
				}
				conditions = append(conditions, condition)
				continue
			}
			columnType := traceFilterColumnType(item.Key.Type)
			columnDataType := traceFilterColumnDataType(item.Key.DataType)
			conditions = append(conditions, fmt.Sprintf(operator, columnType+"_"+columnDataType, item.Key.Key))
		default:
			conditions = append(conditions, fmt.Sprintf("%s %s %s", columnName, operator, formattedValue))
		}
	}

	return strings.Join(conditions, " AND "), nil
}
