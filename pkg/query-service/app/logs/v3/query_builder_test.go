package v3

import (
	"testing"

	v3 "github.com/SigNoz/signoz/pkg/query-service/model/v3"
	. "github.com/smartystreets/goconvey/convey"
)

var testGetClickhouseColumnNameData = []struct {
	Name               string
	AttributeKey       v3.AttributeKey
	ExpectedColumnName string
}{
	{
		Name:               "attribute",
		AttributeKey:       v3.AttributeKey{Key: "user_name", DataType: v3.AttributeKeyDataTypeString, Type: v3.AttributeKeyTypeTag},
		ExpectedColumnName: "attributes_string_value[indexOf(attributes_string_key, 'user_name')]",
	},
	{
		Name:               "resource",
		AttributeKey:       v3.AttributeKey{Key: "servicename", DataType: v3.AttributeKeyDataTypeString, Type: v3.AttributeKeyTypeResource},
		ExpectedColumnName: "resources_string_value[indexOf(resources_string_key, 'servicename')]",
	},
	{
		Name:               "selected field",
		AttributeKey:       v3.AttributeKey{Key: "servicename", DataType: v3.AttributeKeyDataTypeString, Type: v3.AttributeKeyTypeTag, IsColumn: true},
		ExpectedColumnName: "`attribute_string_servicename`",
	},
	{
		Name:               "selected field resource",
		AttributeKey:       v3.AttributeKey{Key: "sdk_version", DataType: v3.AttributeKeyDataTypeInt64, Type: v3.AttributeKeyTypeResource, IsColumn: true},
		ExpectedColumnName: "`resource_int64_sdk_version`",
	},
	{
		Name:               "selected field float",
		AttributeKey:       v3.AttributeKey{Key: "sdk_version", DataType: v3.AttributeKeyDataTypeFloat64, Type: v3.AttributeKeyTypeTag, IsColumn: true},
		ExpectedColumnName: "`attribute_float64_sdk_version`",
	},
	{
		Name:               "same name as top level column",
		AttributeKey:       v3.AttributeKey{Key: "trace_id", DataType: v3.AttributeKeyDataTypeString, Type: v3.AttributeKeyTypeTag},
		ExpectedColumnName: "attributes_string_value[indexOf(attributes_string_key, 'trace_id')]",
	},
	{
		Name:               "top level column",
		AttributeKey:       v3.AttributeKey{Key: "trace_id", DataType: v3.AttributeKeyDataTypeString, Type: v3.AttributeKeyTypeUnspecified, IsColumn: true},
		ExpectedColumnName: "trace_id",
	},
	{
		Name:               "name with - ",
		AttributeKey:       v3.AttributeKey{Key: "test-attr", DataType: v3.AttributeKeyDataTypeString, Type: v3.AttributeKeyTypeTag, IsColumn: true},
		ExpectedColumnName: "`attribute_string_test-attr`",
	},
	{
		Name:               "instrumentation scope attribute",
		AttributeKey:       v3.AttributeKey{Key: "version", DataType: v3.AttributeKeyDataTypeString, Type: v3.AttributeKeyTypeInstrumentationScope, IsColumn: false},
		ExpectedColumnName: "scope_string_value[indexOf(scope_string_key, 'version')]",
	},
}

func TestGetClickhouseColumnName(t *testing.T) {
	for _, tt := range testGetClickhouseColumnNameData {
		Convey("testGetClickhouseColumnNameData", t, func() {
			columnName := getClickhouseColumnName(tt.AttributeKey)
			So(columnName, ShouldEqual, tt.ExpectedColumnName)
		})
	}
}

var testGetSelectLabelsData = []struct {
	Name              string
	AggregateOperator v3.AggregateOperator
	GroupByTags       []v3.AttributeKey
	SelectLabels      string
}{
	{
		Name:              "select fields for groupBy attribute",
		AggregateOperator: v3.AggregateOperatorCount,
		GroupByTags:       []v3.AttributeKey{{Key: "user_name", DataType: v3.AttributeKeyDataTypeString, Type: v3.AttributeKeyTypeTag}},
		SelectLabels:      " attributes_string_value[indexOf(attributes_string_key, 'user_name')] as `user_name`,",
	},
	{
		Name:              "select fields for groupBy resource",
		AggregateOperator: v3.AggregateOperatorCount,
		GroupByTags:       []v3.AttributeKey{{Key: "user_name", DataType: v3.AttributeKeyDataTypeString, Type: v3.AttributeKeyTypeResource}},
		SelectLabels:      " resources_string_value[indexOf(resources_string_key, 'user_name')] as `user_name`,",
	},
	{
		Name:              "select fields for groupBy attribute and resource",
		AggregateOperator: v3.AggregateOperatorCount,
		GroupByTags: []v3.AttributeKey{
			{Key: "user_name", DataType: v3.AttributeKeyDataTypeString, Type: v3.AttributeKeyTypeResource},
			{Key: "host", DataType: v3.AttributeKeyDataTypeString, Type: v3.AttributeKeyTypeTag},
		},
		SelectLabels: " resources_string_value[indexOf(resources_string_key, 'user_name')] as `user_name`, attributes_string_value[indexOf(attributes_string_key, 'host')] as `host`,",
	},
	{
		Name:              "select fields for groupBy materialized columns",
		AggregateOperator: v3.AggregateOperatorCount,
		GroupByTags:       []v3.AttributeKey{{Key: "host", IsColumn: true}},
		SelectLabels:      " host as `host`,",
	},
	{
		Name:              "trace_id field as an attribute",
		AggregateOperator: v3.AggregateOperatorCount,
		GroupByTags:       []v3.AttributeKey{{Key: "trace_id", DataType: v3.AttributeKeyDataTypeString, Type: v3.AttributeKeyTypeTag}},
		SelectLabels:      " attributes_string_value[indexOf(attributes_string_key, 'trace_id')] as `trace_id`,",
	},
}

func TestGetSelectLabels(t *testing.T) {
	for _, tt := range testGetSelectLabelsData {
		Convey("testGetSelectLabelsData", t, func() {
			selectLabels := getSelectLabels(tt.AggregateOperator, tt.GroupByTags)
			So(selectLabels, ShouldEqual, tt.SelectLabels)
		})
	}
}

var timeSeriesFilterQueryData = []struct {
	Name           string
	FilterSet      *v3.FilterSet
	GroupBy        []v3.AttributeKey
	ExpectedFilter string
	Fields         map[string]v3.AttributeKey
	Error          string
}{
	{
		Name: "Test attribute and resource attribute",
		FilterSet: &v3.FilterSet{Operator: "AND", Items: []v3.FilterItem{
			{Key: v3.AttributeKey{Key: "user_name", DataType: v3.AttributeKeyDataTypeString, Type: v3.AttributeKeyTypeTag}, Value: "john", Operator: "="},
			{Key: v3.AttributeKey{Key: "k8s_namespace", DataType: v3.AttributeKeyDataTypeString, Type: v3.AttributeKeyTypeResource}, Value: "my_service", Operator: "!="},
		}},
		ExpectedFilter: "attributes_string_value[indexOf(attributes_string_key, 'user_name')] = 'john' AND resources_string_value[indexOf(resources_string_key, 'k8s_namespace')] != 'my_service'",
	},
	{
		Name: "Test instrumentation scope attribute",
		FilterSet: &v3.FilterSet{Operator: "AND", Items: []v3.FilterItem{
			{Key: v3.AttributeKey{Key: "version", DataType: v3.AttributeKeyDataTypeString, Type: v3.AttributeKeyTypeInstrumentationScope}, Value: "v1", Operator: "="},
		}},
		ExpectedFilter: "scope_string_value[indexOf(scope_string_key, 'version')] = 'v1'",
	},
	{
		Name: "Test attribute and resource attribute with different case",
		FilterSet: &v3.FilterSet{Operator: "AND", Items: []v3.FilterItem{
			{Key: v3.AttributeKey{Key: "user_name", DataType: v3.AttributeKeyDataTypeString, Type: v3.AttributeKeyTypeTag}, Value: "%JoHn%", Operator: "like"},
			{Key: v3.AttributeKey{Key: "k8s_namespace", DataType: v3.AttributeKeyDataTypeString, Type: v3.AttributeKeyTypeResource}, Value: "%MyService%", Operator: "nlike"},
		}},
		ExpectedFilter: "attributes_string_value[indexOf(attributes_string_key, 'user_name')] ILIKE '%JoHn%' AND resources_string_value[indexOf(resources_string_key, 'k8s_namespace')] NOT ILIKE '%MyService%'",
	},
	{
		Name: "Test materialized column",
		FilterSet: &v3.FilterSet{Operator: "AND", Items: []v3.FilterItem{
			{Key: v3.AttributeKey{Key: "user_name", DataType: v3.AttributeKeyDataTypeString, Type: v3.AttributeKeyTypeTag, IsColumn: true}, Value: "john", Operator: "="},
			{Key: v3.AttributeKey{Key: "k8s_namespace", DataType: v3.AttributeKeyDataTypeString, Type: v3.AttributeKeyTypeResource}, Value: "my_service", Operator: "!="},
		}},
		ExpectedFilter: "`attribute_string_user_name` = 'john' AND resources_string_value[indexOf(resources_string_key, 'k8s_namespace')] != 'my_service'",
	},
	{
		Name: "Test like",
		FilterSet: &v3.FilterSet{Operator: "AND", Items: []v3.FilterItem{
			{Key: v3.AttributeKey{Key: "host", DataType: v3.AttributeKeyDataTypeString, Type: v3.AttributeKeyTypeTag}, Value: "102.%", Operator: "like"},
		}},
		ExpectedFilter: "attributes_string_value[indexOf(attributes_string_key, 'host')] ILIKE '102.%'",
	},
	{
		Name: "Test IN",
		FilterSet: &v3.FilterSet{Operator: "AND", Items: []v3.FilterItem{
			{Key: v3.AttributeKey{Key: "bytes", DataType: v3.AttributeKeyDataTypeFloat64, Type: v3.AttributeKeyTypeTag}, Value: []interface{}{1, 2, 3, 4}, Operator: "in"},
		}},
		ExpectedFilter: "attributes_float64_value[indexOf(attributes_float64_key, 'bytes')] IN [1,2,3,4]",
	},
	{
		Name: "Test DataType int64",
		FilterSet: &v3.FilterSet{Operator: "AND", Items: []v3.FilterItem{
			{Key: v3.AttributeKey{Key: "bytes", DataType: v3.AttributeKeyDataTypeInt64, Type: v3.AttributeKeyTypeTag}, Value: 10, Operator: ">"},
		}},
		ExpectedFilter: "attributes_int64_value[indexOf(attributes_int64_key, 'bytes')] > 10",
	},
	{
		Name: "Test NOT IN",
		FilterSet: &v3.FilterSet{Operator: "AND", Items: []v3.FilterItem{
			{Key: v3.AttributeKey{Key: "name", DataType: v3.AttributeKeyDataTypeString, Type: v3.AttributeKeyTypeTag}, Value: []interface{}{"john", "bunny"}, Operator: "nin"},
		}},
		ExpectedFilter: "attributes_string_value[indexOf(attributes_string_key, 'name')] NOT IN ['john','bunny']",
	},
	{
		Name: "Test exists",
		FilterSet: &v3.FilterSet{Operator: "AND", Items: []v3.FilterItem{
			{Key: v3.AttributeKey{Key: "bytes", DataType: v3.AttributeKeyDataTypeString, Type: v3.AttributeKeyTypeTag}, Value: "", Operator: "exists"},
		}},
		ExpectedFilter: "has(attributes_string_key, 'bytes')",
	},
	{
		Name: "Test not exists",
		FilterSet: &v3.FilterSet{Operator: "AND", Items: []v3.FilterItem{
			{Key: v3.AttributeKey{Key: "bytes", DataType: v3.AttributeKeyDataTypeString, Type: v3.AttributeKeyTypeTag}, Value: "", Operator: "nexists"},
		}},
		ExpectedFilter: "not has(attributes_string_key, 'bytes')",
	},
	{
		Name: "Test contains",
		FilterSet: &v3.FilterSet{Operator: "AND", Items: []v3.FilterItem{
			{Key: v3.AttributeKey{Key: "host", DataType: v3.AttributeKeyDataTypeString, Type: v3.AttributeKeyTypeTag}, Value: "102.", Operator: "contains"},
		}},
		ExpectedFilter: "attributes_string_value[indexOf(attributes_string_key, 'host')] ILIKE '%102.%'",
	},
	{
		Name: "Test contains with single quotes",
		FilterSet: &v3.FilterSet{Operator: "AND", Items: []v3.FilterItem{
			{Key: v3.AttributeKey{Key: "message", DataType: v3.AttributeKeyDataTypeString, Type: v3.AttributeKeyTypeTag}, Value: "hello 'world'", Operator: "contains"},
		}},
		ExpectedFilter: "attributes_string_value[indexOf(attributes_string_key, 'message')] ILIKE '%hello \\'world\\'%'",
	},
	{
		Name: "Test not contains",
		FilterSet: &v3.FilterSet{Operator: "AND", Items: []v3.FilterItem{
			{Key: v3.AttributeKey{Key: "host", DataType: v3.AttributeKeyDataTypeString, Type: v3.AttributeKeyTypeTag}, Value: "102.", Operator: "ncontains"},
		}},
		ExpectedFilter: "attributes_string_value[indexOf(attributes_string_key, 'host')] NOT ILIKE '%102.%'",
	},
	{
		Name: "Test regex",
		FilterSet: &v3.FilterSet{Operator: "AND", Items: []v3.FilterItem{
			{Key: v3.AttributeKey{Key: "host", DataType: v3.AttributeKeyDataTypeString, Type: v3.AttributeKeyTypeTag, IsColumn: true}, Value: "host: \"(?P<host>\\S+)\"", Operator: "regex"},
		}},
		ExpectedFilter: "match(`attribute_string_host`, 'host: \"(?P<host>\\\\S+)\"')",
	},
	{
		Name: "Test not regex",
		FilterSet: &v3.FilterSet{Operator: "AND", Items: []v3.FilterItem{
			{Key: v3.AttributeKey{Key: "host", DataType: v3.AttributeKeyDataTypeString, Type: v3.AttributeKeyTypeTag}, Value: "102.", Operator: "nregex"},
		}},
		ExpectedFilter: "NOT match(attributes_string_value[indexOf(attributes_string_key, 'host')], '102.')",
	},
	{
		Name: "Test groupBy",
		FilterSet: &v3.FilterSet{Operator: "AND", Items: []v3.FilterItem{
			{Key: v3.AttributeKey{Key: "host", DataType: v3.AttributeKeyDataTypeString, Type: v3.AttributeKeyTypeTag}, Value: "102.", Operator: "ncontains"},
		}},
		GroupBy:        []v3.AttributeKey{{Key: "host", DataType: v3.AttributeKeyDataTypeString, Type: v3.AttributeKeyTypeTag}},
		ExpectedFilter: "attributes_string_value[indexOf(attributes_string_key, 'host')] NOT ILIKE '%102.%' AND has(attributes_string_key, 'host')",
	},
	{
		Name: "Test groupBy isColumn",
		FilterSet: &v3.FilterSet{Operator: "AND", Items: []v3.FilterItem{
			{Key: v3.AttributeKey{Key: "host", DataType: v3.AttributeKeyDataTypeString, Type: v3.AttributeKeyTypeTag}, Value: "102.", Operator: "ncontains"},
		}},
		GroupBy:        []v3.AttributeKey{{Key: "host", DataType: v3.AttributeKeyDataTypeString, Type: v3.AttributeKeyTypeTag, IsColumn: true}},
		ExpectedFilter: "attributes_string_value[indexOf(attributes_string_key, 'host')] NOT ILIKE '%102.%' AND `attribute_string_host_exists`=true",
	},
	{
		Name: "Wrong data",
		FilterSet: &v3.FilterSet{Operator: "AND", Items: []v3.FilterItem{
			{Key: v3.AttributeKey{Key: "bytes", Type: v3.AttributeKeyTypeTag, DataType: v3.AttributeKeyDataTypeFloat64}, Value: true, Operator: "="},
		}},
		Error: "failed to validate and cast value for bytes: invalid data type, expected float, got bool",
	},
	{
		Name: "Test top level field with metadata",
		FilterSet: &v3.FilterSet{Operator: "AND", Items: []v3.FilterItem{
			{Key: v3.AttributeKey{Key: "body", DataType: v3.AttributeKeyDataTypeString, Type: v3.AttributeKeyTypeTag}, Value: "%test%", Operator: "like"},
		}},
		ExpectedFilter: "attributes_string_value[indexOf(attributes_string_key, 'body')] ILIKE '%test%'",
	},
	{
		Name: "Test exists on top level field",
		FilterSet: &v3.FilterSet{Operator: "AND", Items: []v3.FilterItem{
			{Key: v3.AttributeKey{Key: "trace_id", DataType: v3.AttributeKeyDataTypeString, Type: v3.AttributeKeyTypeUnspecified, IsColumn: true}, Operator: "exists"},
		}},
		ExpectedFilter: "trace_id != ''",
	},
	{
		Name: "Test not exists on top level field",
		FilterSet: &v3.FilterSet{Operator: "AND", Items: []v3.FilterItem{
			{Key: v3.AttributeKey{Key: "span_id", DataType: v3.AttributeKeyDataTypeString, Type: v3.AttributeKeyTypeUnspecified, IsColumn: true}, Operator: "nexists"},
		}},
		ExpectedFilter: "span_id = ''",
	},
	{
		Name: "Test exists on top level field number",
		FilterSet: &v3.FilterSet{Operator: "AND", Items: []v3.FilterItem{
			{Key: v3.AttributeKey{Key: "trace_flags", DataType: v3.AttributeKeyDataTypeString, Type: v3.AttributeKeyTypeUnspecified, IsColumn: true}, Operator: "exists"},
		}},
		ExpectedFilter: "trace_flags != 0",
	},
	{
		Name: "Test not exists on top level field number",
		FilterSet: &v3.FilterSet{Operator: "AND", Items: []v3.FilterItem{
			{Key: v3.AttributeKey{Key: "severity_number", DataType: v3.AttributeKeyDataTypeString, Type: v3.AttributeKeyTypeUnspecified, IsColumn: true}, Operator: "nexists"},
		}},
		ExpectedFilter: "severity_number = 0",
	},
	{
		Name: "Test exists on materiazlied column",
		FilterSet: &v3.FilterSet{Operator: "AND", Items: []v3.FilterItem{
			{Key: v3.AttributeKey{Key: "method", DataType: v3.AttributeKeyDataTypeString, Type: v3.AttributeKeyTypeTag, IsColumn: true}, Operator: "exists"},
		}},
		ExpectedFilter: "`attribute_string_method_exists`=true",
	},
	{
		Name: "Test nexists on materiazlied column",
		FilterSet: &v3.FilterSet{Operator: "AND", Items: []v3.FilterItem{
			{Key: v3.AttributeKey{Key: "status", DataType: v3.AttributeKeyDataTypeInt64, Type: v3.AttributeKeyTypeTag, IsColumn: true}, Operator: "nexists"},
		}},
		ExpectedFilter: "`attribute_int64_status_exists`=false",
	},
	{
		Name: "Test for body contains and ncontains",
		FilterSet: &v3.FilterSet{Operator: "AND", Items: []v3.FilterItem{
			{Key: v3.AttributeKey{Key: "body", DataType: v3.AttributeKeyDataTypeString, IsColumn: true}, Operator: "contains", Value: "test"},
			{Key: v3.AttributeKey{Key: "body", DataType: v3.AttributeKeyDataTypeString, IsColumn: true}, Operator: "ncontains", Value: "test1"},
		}},
		ExpectedFilter: "lower(body) LIKE lower('%test%') AND lower(body) NOT LIKE lower('%test1%')",
	},
	{
		Name: "Test for body like and nlike",
		FilterSet: &v3.FilterSet{Operator: "AND", Items: []v3.FilterItem{
			{Key: v3.AttributeKey{Key: "body", DataType: v3.AttributeKeyDataTypeString, IsColumn: true}, Operator: "like", Value: "test"},
			{Key: v3.AttributeKey{Key: "body", DataType: v3.AttributeKeyDataTypeString, IsColumn: true}, Operator: "nlike", Value: "test1"},
		}},
		ExpectedFilter: "lower(body) LIKE lower('test') AND lower(body) NOT LIKE lower('test1')",
	},
}
