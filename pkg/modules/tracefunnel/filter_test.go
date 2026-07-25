package tracefunnel

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/SigNoz/signoz/pkg/query-service/model/querytypes"
)

func TestBuildTracesFilterQuery(t *testing.T) {
	filterSet := &querytypes.FilterSet{Operator: "AND", Items: []querytypes.FilterItem{
		{
			Key:      querytypes.AttributeKey{Key: "service.name", DataType: querytypes.AttributeKeyDataTypeString, Type: querytypes.AttributeKeyTypeResource},
			Value:    []any{"frontend", "api"},
			Operator: querytypes.FilterOperatorIn,
		},
		{
			Key:      querytypes.AttributeKey{Key: "http.method", DataType: querytypes.AttributeKeyDataTypeString, Type: querytypes.AttributeKeyTypeTag},
			Value:    "GET",
			Operator: querytypes.FilterOperatorEqual,
		},
		{
			Key:      querytypes.AttributeKey{Key: "duration", DataType: querytypes.AttributeKeyDataTypeInt64, Type: querytypes.AttributeKeyTypeTag},
			Value:    100,
			Operator: querytypes.FilterOperatorGreaterThan,
		},
	}}

	query, err := buildTracesFilterQuery(filterSet)
	require.NoError(t, err)
	require.Equal(
		t,
		"resources_string['service.name'] IN ['frontend','api'] AND attributes_string['http.method'] = 'GET' AND attributes_number['duration'] > 100",
		query,
	)
}

func TestBuildTracesFilterQueryRejectsUnsupportedOperator(t *testing.T) {
	_, err := buildTracesFilterQuery(&querytypes.FilterSet{Items: []querytypes.FilterItem{{
		Key:      querytypes.AttributeKey{Key: "host", DataType: querytypes.AttributeKeyDataTypeString},
		Operator: querytypes.FilterOperator("unsupported"),
	}}})
	require.ErrorContains(t, err, "unsupported operator")
}
