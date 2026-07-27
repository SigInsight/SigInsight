package resource

import (
	"reflect"
	"testing"

	"github.com/SigNoz/signoz/pkg/query-service/model/querytypes"
)

func Test_buildResourceFilter(t *testing.T) {
	type args struct {
		logsOp string
		key    string
		op     querytypes.FilterOperator
		value  interface{}
	}
	tests := []struct {
		name string
		args args
		want string
	}{
		{
			name: "test exists",
			args: args{
				key: "service.name",
				op:  querytypes.FilterOperatorExists,
			},
			want: `simpleJSONHas(labels, 'service.name')`,
		},
		{
			name: "test nexists",
			args: args{
				key: "service.name",
				op:  querytypes.FilterOperatorNotExists,
			},
			want: `not simpleJSONHas(labels, 'service.name')`,
		},
		{
			name: "test regex",
			args: args{
				logsOp: "match(%s, %s)",
				key:    "service.name",
				op:     querytypes.FilterOperatorRegex,
				value:  ".*",
			},
			want: `match(simpleJSONExtractString(labels, 'service.name'), '.*')`,
		},
		{
			name: "test contains",
			args: args{
				logsOp: "LIKE",
				key:    "service.name",
				op:     querytypes.FilterOperatorContains,
				value:  "Application%_",
			},
			want: `simpleJSONExtractString(lower(labels), 'service.name') LIKE '%application\%\_%'`,
		},
		{
			name: "test eq",
			args: args{
				logsOp: "=",
				key:    "service.name",
				op:     querytypes.FilterOperatorEqual,
				value:  "Application%",
			},
			want: `simpleJSONExtractString(labels, 'service.name') = 'Application%'`,
		},
		{
			name: "test value with quotes",
			args: args{
				logsOp: "=",
				key:    "service.name",
				op:     querytypes.FilterOperatorEqual,
				value:  "Application's",
			},
			want: `simpleJSONExtractString(labels, 'service.name') = 'Application\'s'`,
		},
		{
			name: "test like",
			args: args{
				logsOp: "LIKE",
				key:    "service.name",
				op:     querytypes.FilterOperatorLike,
				value:  "Application%_",
			},
			want: `simpleJSONExtractString(lower(labels), 'service.name') LIKE 'application%_'`,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := buildResourceFilter(tt.args.logsOp, tt.args.key, tt.args.op, tt.args.value); got != tt.want {
				t.Errorf("buildResourceFilter() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_buildIndexFilterForInOperator(t *testing.T) {
	type args struct {
		key   string
		op    querytypes.FilterOperator
		value interface{}
	}
	tests := []struct {
		name string
		args args
		want string
	}{
		{
			name: "test in array",
			args: args{
				key:   "service.name",
				op:    querytypes.FilterOperatorIn,
				value: []interface{}{"Application", "Test"},
			},
			want: `(labels like '%"service.name":"Application"%' OR labels like '%"service.name":"Test"%')`,
		},
		{
			name: "test nin array",
			args: args{
				key:   "service.name",
				op:    querytypes.FilterOperatorNotIn,
				value: []interface{}{"Application", "Test"},
			},
			want: `(labels not like '%"service.name":"Application"%' AND labels not like '%"service.name":"Test"%')`,
		},
		{
			name: "test in string",
			args: args{
				key:   "service.name",
				op:    querytypes.FilterOperatorIn,
				value: "application%",
			},
			want: `(labels like '%"service.name":"application\%"%')`,
		},
		{
			name: "test nin string",
			args: args{
				key:   "service.name",
				op:    querytypes.FilterOperatorNotIn,
				value: `application'"_s`,
			},
			want: `(labels not like '%"service.name":"application\'\\\\"\_s"%')`,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := buildIndexFilterForInOperator(tt.args.key, tt.args.op, tt.args.value); got != tt.want {
				t.Errorf("buildIndexFilterForInOperator() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_buildResourceIndexFilter(t *testing.T) {
	type args struct {
		key   string
		op    querytypes.FilterOperator
		value interface{}
	}
	tests := []struct {
		name string
		args args
		want string
	}{
		{
			name: "test eq",
			args: args{
				key:   "service.name",
				op:    querytypes.FilterOperatorEqual,
				value: `Application"`,
			},
			want: `labels like '%service.name":"Application\\\\"%'`,
		},
		{
			name: "test not eq",
			args: args{
				key:   "service.name",
				op:    querytypes.FilterOperatorNotEqual,
				value: `Application"`,
			},
			want: `labels not like '%service.name":"Application\\\\"%'`,
		},
		{
			name: "test like with % and _",
			args: args{
				key:   "service.name",
				op:    querytypes.FilterOperatorLike,
				value: "Application%_test",
			},
			want: `lower(labels) like '%service.name%application%_test%'`,
		},
		{
			name: "test like with % and _",
			args: args{
				key:   "service.name",
				op:    querytypes.FilterOperatorLike,
				value: "application%_test",
			},
			want: `lower(labels) like '%service.name%application%_test%'`,
		},
		{
			name: "test not like with % and _",
			args: args{
				key:   "service.name",
				op:    querytypes.FilterOperatorLike,
				value: "application%_test",
			},
			want: `lower(labels) like '%service.name%application%_test%'`,
		},
		{
			name: "test not like with % and _",
			args: args{
				key:   "service.name",
				op:    querytypes.FilterOperatorILike,
				value: "application%_test",
			},
			want: `lower(labels) like '%service.name%application%_test%'`,
		},
		{
			name: "test contains",
			args: args{
				key:   "service.name",
				op:    querytypes.FilterOperatorContains,
				value: "application",
			},
			want: `lower(labels) like '%service.name%application%'`,
		},
		{
			name: "test not contains",
			args: args{
				key:   "service.name",
				op:    querytypes.FilterOperatorNotContains,
				value: "application",
			},
			want: ``,
		},
		{
			name: "test contains with % and _",
			args: args{
				key:   "service.name",
				op:    querytypes.FilterOperatorNotContains,
				value: "application%_test",
			},
			want: ``,
		},

		{
			name: "test not regex",
			args: args{
				key:   "service.name",
				op:    querytypes.FilterOperatorNotRegex,
				value: ".*",
			},
			want: ``,
		},
		{
			name: "test in",
			args: args{
				key:   "service.name",
				op:    querytypes.FilterOperatorNotIn,
				value: []interface{}{"Application", "Test"},
			},
			want: `(labels not like '%"service.name":"Application"%' AND labels not like '%"service.name":"Test"%')`,
		},
		{
			name: "test exists",
			args: args{
				key: "service.name",
				op:  querytypes.FilterOperatorExists,
			},
			want: `lower(labels) like '%service.name%'`,
		},
		{
			name: "test not exists",
			args: args{
				key: "service.name",
				op:  querytypes.FilterOperatorNotExists,
			},
			want: `lower(labels) not like '%service.name%'`,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := buildResourceIndexFilter(tt.args.key, tt.args.op, tt.args.value); got != tt.want {
				t.Errorf("buildResourceIndexFilter() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_buildResourceFiltersFromFilterItems(t *testing.T) {
	type args struct {
		fs *querytypes.FilterSet
	}
	tests := []struct {
		name    string
		args    args
		want    []string
		wantErr bool
	}{
		{
			name: "ignore attribute",
			args: args{
				fs: &querytypes.FilterSet{
					Items: []querytypes.FilterItem{
						{
							Key: querytypes.AttributeKey{
								Key:      "service.name",
								DataType: querytypes.AttributeKeyDataTypeString,
								Type:     querytypes.AttributeKeyTypeTag,
							},
							Operator: querytypes.FilterOperatorEqual,
							Value:    "test",
						},
					},
				},
			},
			want:    nil,
			wantErr: false,
		},
		{
			name: "build filter",
			args: args{
				fs: &querytypes.FilterSet{
					Items: []querytypes.FilterItem{
						{
							Key: querytypes.AttributeKey{
								Key:      "service.name",
								DataType: querytypes.AttributeKeyDataTypeString,
								Type:     querytypes.AttributeKeyTypeResource,
							},
							Operator: querytypes.FilterOperatorEqual,
							Value:    "test",
						},
					},
				},
			},
			want: []string{
				"simpleJSONExtractString(labels, 'service.name') = 'test'",
				"labels like '%service.name\":\"test%'",
			},
			wantErr: false,
		},
		{
			name: "build filter with multiple items",
			args: args{
				fs: &querytypes.FilterSet{
					Items: []querytypes.FilterItem{
						{
							Key: querytypes.AttributeKey{
								Key:      "service.name",
								DataType: querytypes.AttributeKeyDataTypeString,
								Type:     querytypes.AttributeKeyTypeResource,
							},
							Operator: querytypes.FilterOperatorEqual,
							Value:    "test",
						},
						{
							Key: querytypes.AttributeKey{
								Key:      "namespace",
								DataType: querytypes.AttributeKeyDataTypeString,
								Type:     querytypes.AttributeKeyTypeResource,
							},
							Operator: querytypes.FilterOperatorContains,
							Value:    `test1"`,
						},
					},
				},
			},
			want: []string{
				"simpleJSONExtractString(labels, 'service.name') = 'test'",
				"labels like '%service.name\":\"test%'",
				`simpleJSONExtractString(lower(labels), 'namespace') LIKE '%test1"%'`,
				`lower(labels) like '%namespace%test1\\\\"%'`,
			},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := buildResourceFiltersFromFilterItems(tt.args.fs)
			if (err != nil) != tt.wantErr {
				t.Errorf("buildResourceFiltersFromFilterItems() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("buildResourceFiltersFromFilterItems() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_buildResourceFiltersFromGroupBy(t *testing.T) {
	type args struct {
		groupBy []querytypes.AttributeKey
	}
	tests := []struct {
		name string
		args args
		want []string
	}{
		{
			name: "build filter",
			args: args{
				groupBy: []querytypes.AttributeKey{
					{
						Key:      "service.name",
						DataType: querytypes.AttributeKeyDataTypeString,
						Type:     querytypes.AttributeKeyTypeResource,
					},
				},
			},
			want: []string{
				"(simpleJSONHas(labels, 'service.name') AND labels like '%service.name%')",
			},
		},
		{
			name: "build filter multiple group by",
			args: args{
				groupBy: []querytypes.AttributeKey{
					{
						Key:      "service.name",
						DataType: querytypes.AttributeKeyDataTypeString,
						Type:     querytypes.AttributeKeyTypeResource,
					},
					{
						Key:      "namespace",
						DataType: querytypes.AttributeKeyDataTypeString,
						Type:     querytypes.AttributeKeyTypeResource,
					},
				},
			},
			want: []string{
				"(simpleJSONHas(labels, 'service.name') AND labels like '%service.name%')",
				"(simpleJSONHas(labels, 'namespace') AND labels like '%namespace%')",
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := buildResourceFiltersFromGroupBy(tt.args.groupBy); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("buildResourceFiltersFromGroupBy() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_buildResourceFiltersFromAggregateAttribute(t *testing.T) {
	type args struct {
		aggregateAttribute querytypes.AttributeKey
	}
	tests := []struct {
		name string
		args args
		want string
	}{
		{
			name: "build filter",
			args: args{
				aggregateAttribute: querytypes.AttributeKey{
					Key:      "service.name",
					DataType: querytypes.AttributeKeyDataTypeString,
					Type:     querytypes.AttributeKeyTypeResource,
				},
			},
			want: "(simpleJSONHas(labels, 'service.name') AND labels like '%service.name%')",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := buildResourceFiltersFromAggregateAttribute(tt.args.aggregateAttribute); got != tt.want {
				t.Errorf("buildResourceFiltersFromAggregateAttribute() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_buildResourceSubQuery(t *testing.T) {
	type args struct {
		bucketStart        int64
		bucketEnd          int64
		fs                 *querytypes.FilterSet
		groupBy            []querytypes.AttributeKey
		aggregateAttribute querytypes.AttributeKey
	}
	tests := []struct {
		name    string
		args    args
		want    string
		wantErr bool
	}{
		{
			name: "build sub query",
			args: args{
				bucketStart: 1680064560,
				bucketEnd:   1680066458,
				fs: &querytypes.FilterSet{
					Items: []querytypes.FilterItem{
						{
							Key: querytypes.AttributeKey{
								Key:      "service.name",
								DataType: querytypes.AttributeKeyDataTypeString,
								Type:     querytypes.AttributeKeyTypeResource,
							},
							Operator: querytypes.FilterOperatorEqual,
							Value:    "test",
						},
						{
							Key: querytypes.AttributeKey{
								Key:      "namespace",
								DataType: querytypes.AttributeKeyDataTypeString,
								Type:     querytypes.AttributeKeyTypeResource,
							},
							Operator: querytypes.FilterOperatorContains,
							Value:    "test1",
						},
					},
				},
				groupBy: []querytypes.AttributeKey{
					{
						Key:      "host.name",
						DataType: querytypes.AttributeKeyDataTypeString,
						Type:     querytypes.AttributeKeyTypeResource,
					},
				},
				aggregateAttribute: querytypes.AttributeKey{
					Key:      "cluster.name",
					DataType: querytypes.AttributeKeyDataTypeString,
					Type:     querytypes.AttributeKeyTypeResource,
				},
			},
			want: "(SELECT fingerprint FROM signoz_logs.logs_v2_resource WHERE " +
				"(seen_at_ts_bucket_start >= 1680064560) AND (seen_at_ts_bucket_start <= 1680066458) AND " +
				"simpleJSONExtractString(labels, 'service.name') = 'test' AND labels like '%service.name\":\"test%' " +
				"AND simpleJSONExtractString(lower(labels), 'namespace') LIKE '%test1%' AND lower(labels) like '%namespace%test1%' " +
				"AND (simpleJSONHas(labels, 'cluster.name') AND labels like '%cluster.name%') AND " +
				"( (simpleJSONHas(labels, 'host.name') AND labels like '%host.name%') ))",
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := BuildResourceSubQuery("signoz_logs", "logs_v2_resource", tt.args.bucketStart, tt.args.bucketEnd, tt.args.fs, tt.args.groupBy, tt.args.aggregateAttribute, false)
			if (err != nil) != tt.wantErr {
				t.Errorf("buildResourceSubQuery() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("buildResourceSubQuery() = %v, want %v", got, tt.want)
			}
		})
	}
}
