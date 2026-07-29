package exceptionstore

import (
	"fmt"
	"math/rand"
	"strings"
	"sync"
	"time"

	"github.com/ClickHouse/clickhouse-go/v2"

	errorsV2 "github.com/SigNoz/signoz/pkg/errors"
	"github.com/SigNoz/signoz/pkg/query-service/model"
)

const parameterCharset = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"

var (
	parameterRand   = rand.New(rand.NewSource(time.Now().UnixNano()))
	parameterRandMu sync.Mutex
)

func createTagQueries(queryParams []model.TagQueryParam) []model.TagQuery {
	tags := []model.TagQuery{}
	for _, tag := range queryParams {
		if len(tag.StringValues) > 0 {
			tags = append(tags, model.NewTagQueryString(tag))
		}
		if len(tag.NumberValues) > 0 {
			tags = append(tags, model.NewTagQueryNumber(tag))
		}
		if len(tag.BoolValues) > 0 {
			tags = append(tags, model.NewTagQueryBool(tag))
		}
	}
	return tags
}

func randomParameterSuffix(length int) string {
	parameterRandMu.Lock()
	defer parameterRandMu.Unlock()

	b := make([]byte, length)
	for i := range b {
		b[i] = parameterCharset[parameterRand.Intn(len(parameterCharset))]
	}
	return string(b)
}

func buildTagQuery(tags []model.TagQuery) (string, []interface{}, error) {
	query := ""
	var args []interface{}
	for _, item := range tags {
		var subQuery string
		var subQueryArgs []interface{}
		tagMapType := item.GetTagMapColumn()
		switch item.GetOperator() {
		case model.EqualOperator:
			subQuery, subQueryArgs = addArithmeticOperator(item, tagMapType, "=")
		case model.NotEqualOperator:
			subQuery, subQueryArgs = addArithmeticOperator(item, tagMapType, "!=")
		case model.LessThanOperator:
			subQuery, subQueryArgs = addArithmeticOperator(item, tagMapType, "<")
		case model.GreaterThanOperator:
			subQuery, subQueryArgs = addArithmeticOperator(item, tagMapType, ">")
		case model.InOperator:
			subQuery, subQueryArgs = addInOperator(item, tagMapType, false)
		case model.NotInOperator:
			subQuery, subQueryArgs = addInOperator(item, tagMapType, true)
		case model.LessThanEqualOperator:
			subQuery, subQueryArgs = addArithmeticOperator(item, tagMapType, "<=")
		case model.GreaterThanEqualOperator:
			subQuery, subQueryArgs = addArithmeticOperator(item, tagMapType, ">=")
		case model.ContainsOperator:
			subQuery, subQueryArgs = addContainsOperator(item, tagMapType, false)
		case model.NotContainsOperator:
			subQuery, subQueryArgs = addContainsOperator(item, tagMapType, true)
		case model.StartsWithOperator:
			subQuery, subQueryArgs = addStartsWithOperator(item, tagMapType, false)
		case model.NotStartsWithOperator:
			subQuery, subQueryArgs = addStartsWithOperator(item, tagMapType, true)
		case model.ExistsOperator:
			subQuery, subQueryArgs = addExistsOperator(item, tagMapType, false)
		case model.NotExistsOperator:
			subQuery, subQueryArgs = addExistsOperator(item, tagMapType, true)
		default:
			return "", nil, errorsV2.NewInvalidInputf(errorsV2.CodeInvalidInput, "filter operator %s not supported", item.GetOperator())
		}
		query += subQuery
		args = append(args, subQueryArgs...)
	}
	return query, args, nil
}

func addInOperator(item model.TagQuery, tagMapType string, not bool) (string, []interface{}) {
	values := item.GetValues()
	args := []interface{}{}
	notStr := ""
	if not {
		notStr = "NOT"
	}
	tagValuePair := []string{}
	for _, value := range values {
		tagKey := "inTagKey" + randomParameterSuffix(5)
		tagValue := "inTagValue" + randomParameterSuffix(5)
		tagValuePair = append(tagValuePair, fmt.Sprintf("%s[@%s] = @%s", tagMapType, tagKey, tagValue))
		args = append(args, clickhouse.Named(tagKey, item.GetKey()))
		args = append(args, clickhouse.Named(tagValue, value))
	}
	return fmt.Sprintf(" AND %s (%s)", notStr, strings.Join(tagValuePair, " OR ")), args
}

func addContainsOperator(item model.TagQuery, tagMapType string, not bool) (string, []interface{}) {
	values := item.GetValues()
	args := []interface{}{}
	notStr := ""
	if not {
		notStr = "NOT"
	}
	tagValuePair := []string{}
	for _, value := range values {
		tagKey := "containsTagKey" + randomParameterSuffix(5)
		tagValue := "containsTagValue" + randomParameterSuffix(5)
		tagValuePair = append(tagValuePair, fmt.Sprintf("%s[@%s] ILIKE @%s", tagMapType, tagKey, tagValue))
		args = append(args, clickhouse.Named(tagKey, item.GetKey()))
		args = append(args, clickhouse.Named(tagValue, "%"+fmt.Sprintf("%v", value)+"%"))
	}
	return fmt.Sprintf(" AND %s (%s)", notStr, strings.Join(tagValuePair, " OR ")), args
}

func addStartsWithOperator(item model.TagQuery, tagMapType string, not bool) (string, []interface{}) {
	values := item.GetValues()
	args := []interface{}{}
	notStr := ""
	if not {
		notStr = "NOT"
	}
	tagValuePair := []string{}
	for _, value := range values {
		tagKey := "startsWithTagKey" + randomParameterSuffix(5)
		tagValue := "startsWithTagValue" + randomParameterSuffix(5)
		tagValuePair = append(tagValuePair, fmt.Sprintf("%s[@%s] ILIKE @%s", tagMapType, tagKey, tagValue))
		args = append(args, clickhouse.Named(tagKey, item.GetKey()))
		args = append(args, clickhouse.Named(tagValue, "%"+fmt.Sprintf("%v", value)+"%"))
	}
	return fmt.Sprintf(" AND %s (%s)", notStr, strings.Join(tagValuePair, " OR ")), args
}

func addArithmeticOperator(item model.TagQuery, tagMapType string, operator string) (string, []interface{}) {
	values := item.GetValues()
	args := []interface{}{}
	tagValuePair := []string{}
	for _, value := range values {
		tagKey := "arithmeticTagKey" + randomParameterSuffix(5)
		tagValue := "arithmeticTagValue" + randomParameterSuffix(5)
		tagValuePair = append(tagValuePair, fmt.Sprintf("%s[@%s] %s @%s", tagMapType, tagKey, operator, tagValue))
		args = append(args, clickhouse.Named(tagKey, item.GetKey()))
		args = append(args, clickhouse.Named(tagValue, value))
	}
	return fmt.Sprintf(" AND (%s)", strings.Join(tagValuePair, " OR ")), args
}

func addExistsOperator(item model.TagQuery, tagMapType string, not bool) (string, []interface{}) {
	values := item.GetValues()
	notStr := ""
	if not {
		notStr = "NOT"
	}
	args := []interface{}{}
	tagOperatorPair := []string{}
	for range values {
		tagKey := "existsTagKey" + randomParameterSuffix(5)
		tagOperatorPair = append(tagOperatorPair, fmt.Sprintf("mapContains(%s, @%s)", tagMapType, tagKey))
		args = append(args, clickhouse.Named(tagKey, item.GetKey()))
	}
	return fmt.Sprintf(" AND %s (%s)", notStr, strings.Join(tagOperatorPair, " OR ")), args
}
