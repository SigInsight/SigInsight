package utils

import (
	"reflect"
	"testing"

	"github.com/SigNoz/signoz/pkg/query-service/constants"
	"github.com/SigNoz/signoz/pkg/query-service/model/querytypes"
)

type args struct {
	v        interface{}
	dataType querytypes.AttributeKeyDataType
}

var testValidateAndCastValueData = []struct {
	name    string
	args    args
	want    interface{}
	wantErr bool
}{
	// Test cases for querytypes.AttributeKeyDataTypeString
	{
		name: "querytypes.AttributeKeyDataTypeString: Valid string",
		args: args{
			v:        "test",
			dataType: querytypes.AttributeKeyDataTypeString,
		},
		want:    "test",
		wantErr: false,
	},
	{
		name: "querytypes.AttributeKeyDataTypeString: Valid int",
		args: args{
			v:        1,
			dataType: querytypes.AttributeKeyDataTypeString,
		},
		want:    "1",
		wantErr: false,
	},
	{
		name: "querytypes.AttributeKeyDataTypeString: Valid float32",
		args: args{
			v:        float32(1.1),
			dataType: querytypes.AttributeKeyDataTypeString,
		},
		want:    "1.1",
		wantErr: false,
	},
	{
		name: "querytypes.AttributeKeyDataTypeString: Valid float64",
		args: args{
			v:        float64(1.1),
			dataType: querytypes.AttributeKeyDataTypeString,
		},
		want:    "1.1",
		wantErr: false,
	},
	{
		name: "querytypes.AttributeKeyDataTypeString: Valid bool",
		args: args{
			v:        true,
			dataType: querytypes.AttributeKeyDataTypeString,
		},
		want:    "true",
		wantErr: false,
	},
	{
		name: "querytypes.AttributeKeyDataTypeString: Valid []interface{}",
		args: args{
			v:        []interface{}{"test", "test2"},
			dataType: querytypes.AttributeKeyDataTypeString,
		},
		want:    []interface{}{"test", "test2"},
		wantErr: false,
	},
	{
		name: "querytypes.AttributeKeyDataTypeString: Valid []interface{}",
		args: args{
			v:        []interface{}{"test", 1},
			dataType: querytypes.AttributeKeyDataTypeString,
		},
		want:    []interface{}{"test", "1"},
		wantErr: false,
	},
	{
		name: "querytypes.AttributeKeyDataTypeString: Invalid []interface{}",
		args: args{
			v:        []interface{}{"test", [1]string{"string Array"}},
			dataType: querytypes.AttributeKeyDataTypeString,
		},
		want:    nil,
		wantErr: true,
	},
	{
		name: "querytypes.AttributeKeyDataTypeString: Invalid type",
		args: args{
			v:        map[string]interface{}{"test": "test"},
			dataType: querytypes.AttributeKeyDataTypeString,
		},
		want:    nil,
		wantErr: true,
	},
	// Test cases for querytypes.AttributeKeyDataTypeBool
	{
		name: "querytypes.AttributeKeyDataTypeBool: Valid bool",
		args: args{
			v:        true,
			dataType: querytypes.AttributeKeyDataTypeBool,
		},
		want:    true,
		wantErr: false,
	},
	{
		name: "querytypes.AttributeKeyDataTypeBool: Valid string",
		args: args{
			v:        "true",
			dataType: querytypes.AttributeKeyDataTypeBool,
		},
		want:    true,
		wantErr: false,
	},
	{
		name: "querytypes.AttributeKeyDataTypeBool: Valid []interface{}",
		args: args{
			v:        []interface{}{"true", false},
			dataType: querytypes.AttributeKeyDataTypeBool,
		},
		want:    []interface{}{true, false},
		wantErr: false,
	},
	{
		name: "querytypes.AttributeKeyDataTypeBool: Invalid type",
		args: args{
			v:        1,
			dataType: querytypes.AttributeKeyDataTypeBool,
		},
		want:    nil,
		wantErr: true,
	},
	{
		name: "querytypes.AttributeKeyDataTypeBool: Invalid []interface{}",
		args: args{
			v:        []interface{}{1, false},
			dataType: querytypes.AttributeKeyDataTypeBool,
		},
		want:    nil,
		wantErr: true,
	},
	// Test cases for querytypes.AttributeKeyDataTypeInt64
	{
		name: "querytypes.AttributeKeyDataTypeInt64: Valid int",
		args: args{
			v:        1,
			dataType: querytypes.AttributeKeyDataTypeInt64,
		},
		want:    1,
		wantErr: false,
	},
	{
		name: "querytypes.AttributeKeyDataTypeInt64: Valid int64",
		args: args{
			v:        int64(1),
			dataType: querytypes.AttributeKeyDataTypeInt64,
		},
		want:    int64(1),
		wantErr: false,
	},
	{
		name: "querytypes.AttributeKeyDataTypeInt64: Valid string",
		args: args{
			v:        "1",
			dataType: querytypes.AttributeKeyDataTypeInt64,
		},
		want:    int64(1),
		wantErr: false,
	},
	{
		name: "querytypes.AttributeKeyDataTypeInt64: Valid []interface{}",
		args: args{
			v:        []interface{}{"1", 2},
			dataType: querytypes.AttributeKeyDataTypeInt64,
		},
		want:    []interface{}{int64(1), int64(2)},
		wantErr: false,
	},
	{
		name: "querytypes.AttributeKeyDataTypeInt64: Invalid []interface{}",
		args: args{
			v:        []interface{}{"1", false},
			dataType: querytypes.AttributeKeyDataTypeInt64,
		},
		want:    nil,
		wantErr: true,
	},
	{
		name: "querytypes.AttributeKeyDataTypeInt64: Invalid type",
		args: args{
			v:        true,
			dataType: querytypes.AttributeKeyDataTypeInt64,
		},
		want:    nil,
		wantErr: true,
	},
	// Test cases for querytypes.AttributeKeyDataTypeFloat64
	{
		name: "querytypes.AttributeKeyDataTypeFloat64: Valid float32",
		args: args{
			v:        float32(1.1),
			dataType: querytypes.AttributeKeyDataTypeFloat64,
		},
		want:    float32(1.1),
		wantErr: false,
	},
	{
		name: "querytypes.AttributeKeyDataTypeFloat64: Valid float64",
		args: args{
			v:        float64(1.1),
			dataType: querytypes.AttributeKeyDataTypeFloat64,
		},
		want:    float64(1.1),
		wantErr: false,
	},
	{
		name: "querytypes.AttributeKeyDataTypeFloat64: Valid int",
		args: args{
			v:        1,
			dataType: querytypes.AttributeKeyDataTypeFloat64,
		},
		want:    float64(1),
		wantErr: false,
	},
	{
		name: "querytypes.AttributeKeyDataTypeFloat64: Valid string",
		args: args{
			v:        "1.1",
			dataType: querytypes.AttributeKeyDataTypeFloat64,
		},
		want:    float64(1.1),
		wantErr: false,
	},
	{
		name: "querytypes.AttributeKeyDataTypeFloat: Valid []interface{}",
		args: args{
			v:        []interface{}{4, 3},
			dataType: querytypes.AttributeKeyDataTypeFloat64,
		},
		want:    []interface{}{float64(4), float64(3)},
		wantErr: false,
	},
	{
		name: "querytypes.AttributeKeyDataTypeFloat: Valid []interface{}",
		args: args{
			v:        []interface{}{4, "3"},
			dataType: querytypes.AttributeKeyDataTypeFloat64,
		},
		want:    []interface{}{float64(4), float64(3)},
		wantErr: false,
	},
	{
		name: "querytypes.AttributeKeyDataTypeFloat: Invalid []interface{}",
		args: args{
			v:        []interface{}{4, "true"},
			dataType: querytypes.AttributeKeyDataTypeFloat64,
		},
		want:    nil,
		wantErr: true,
	},
	{
		name: "querytypes.AttributeKeyDataTypeFloat64: Invalid type",
		args: args{
			v:        true,
			dataType: querytypes.AttributeKeyDataTypeFloat64,
		},
		want:    nil,
		wantErr: true,
	},
	{
		name: "querytypes.AttributeKeyDataTypeInt64: valid float32",
		args: args{
			v:        float32(1000),
			dataType: querytypes.AttributeKeyDataTypeInt64,
		},
		want:    int64(1000),
		wantErr: false,
	},
	{
		name: "querytypes.AttributeKeyDataTypeInt64: valid float64",
		args: args{
			v:        float64(1000),
			dataType: querytypes.AttributeKeyDataTypeInt64,
		},
		want:    int64(1000),
		wantErr: false,
	},
}

// Test cases for ValidateAndCastValue function in pkg/query-service/utils/format.go
func TestValidateAndCastValue(t *testing.T) {
	for _, tt := range testValidateAndCastValueData {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ValidateAndCastValue(tt.args.v, tt.args.dataType)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateAndCastValue() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) && !reflect.DeepEqual(err, tt.wantErr) {
				t.Errorf("ValidateAndCastValue() = %v, want %v", got, tt.want)
			}
		})
	}
}

var one = 1
var onePointOne = 1.1
var oneString = "1"
var trueBool = true

var testClickHouseFormattedValueData = []struct {
	name  string
	value interface{}
	want  interface{}
}{
	{
		name:  "int",
		value: 1,
		want:  "1",
	},
	{
		name:  "int64",
		value: int64(1),
		want:  "1",
	},
	{
		name:  "float32",
		value: float32(1.1),
		want:  "1.100000",
	},
	{
		name:  "string",
		value: "1",
		want:  "'1'",
	},
	{
		name:  "bool",
		value: true,
		want:  "true",
	},
	{
		name:  "[]interface{}",
		value: []interface{}{1, 2},
		want:  "[1,2]",
	},
	{
		name:  "[]interface{}",
		value: []interface{}{"1", "2"},
		want:  "['1','2']",
	},
	{
		name:  "pointer int",
		value: &one,
		want:  "1",
	},
	{
		name:  "pointer float32",
		value: onePointOne,
		want:  "1.100000",
	},
	{
		name:  "pointer string",
		value: &oneString,
		want:  "'1'",
	},
	{
		name:  "pointer bool",
		value: &trueBool,
		want:  "true",
	},
	{
		name:  "pointer []interface{}",
		value: []interface{}{&one, &one},
		want:  "[1,1]",
	},
	{
		name:  "string with single quote",
		value: "test'1",
		want:  "'test\\'1'",
	},
	{
		name: "[]interface{} with string with single quote",
		value: []interface{}{
			"test'1",
			"test'2",
		},
		want: "['test\\'1','test\\'2']",
	},
}

func TestClickHouseFormattedValue(t *testing.T) {
	for _, tt := range testClickHouseFormattedValueData {
		t.Run(tt.name, func(t *testing.T) {
			got := ClickHouseFormattedValue(tt.value)
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("ClickHouseFormattedValue() = %v, want %v", got, tt.want)
			}
		})
	}
}

var testGetClickhouseColumnName = []struct {
	name     string
	typeName string
	dataType string
	field    string
	want     string
}{
	{
		name:     "tag",
		typeName: string(querytypes.AttributeKeyTypeTag),
		dataType: string(querytypes.AttributeKeyDataTypeInt64),
		field:    "tag1",
		want:     "`attribute_int64_tag1`",
	},
	{
		name:     "resource",
		typeName: string(querytypes.AttributeKeyTypeResource),
		dataType: string(querytypes.AttributeKeyDataTypeInt64),
		field:    "tag1",
		want:     "`resource_int64_tag1`",
	},
	{
		name:     "attribute old parser",
		typeName: constants.Attributes,
		dataType: string(querytypes.AttributeKeyDataTypeInt64),
		field:    "tag1",
		want:     "`attribute_int64_tag1`",
	},
	{
		name:     "resource old parser",
		typeName: constants.Resources,
		dataType: string(querytypes.AttributeKeyDataTypeInt64),
		field:    "tag1",
		want:     "`resource_int64_tag1`",
	},
}

func TestGetClickhouseColumnName(t *testing.T) {
	for _, tt := range testGetClickhouseColumnName {
		t.Run(tt.name, func(t *testing.T) {
			got := GetClickhouseColumnName(tt.typeName, tt.dataType, tt.field)
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("ClickHouseFormattedValue() = %v, want %v", got, tt.want)
			}
		})
	}
}

var testGetEpochNanoSecsData = []struct {
	Name       string
	Epoch      int64
	Multiplier int64
	Result     int64
}{
	{
		Name:   "Test 1",
		Epoch:  1680712080000,
		Result: 1680712080000000000,
	},
	{
		Name:   "Test 1",
		Epoch:  1680712080000000000,
		Result: 1680712080000000000,
	},
}

func TestGetEpochNanoSecs(t *testing.T) {
	for _, tt := range testGetEpochNanoSecsData {
		t.Run(tt.Name, func(t *testing.T) {
			got := GetEpochNanoSecs(tt.Epoch)
			if !reflect.DeepEqual(got, tt.Result) {
				t.Errorf("ClickHouseFormattedValue() = %v, want %v", got, tt.Result)
			}
		})
	}
}
