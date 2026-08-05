package querybuildertypesv5

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestQueryEnvelopeUnmarshalSupportedLiteTypes(t *testing.T) {
	tests := []struct {
		name string
		raw  string
		want any
	}{
		{
			name: "log builder",
			raw:  `{"type":"builder_query","spec":{"name":"A","signal":"logs","aggregations":[{"expression":"count()"}]}}`,
			want: QueryBuilderQuery[LogAggregation]{},
		},
		{
			name: "formula",
			raw:  `{"type":"builder_formula","spec":{"name":"F1","expression":"A"}}`,
			want: QueryBuilderFormula{},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			var envelope QueryEnvelope
			require.NoError(t, json.Unmarshal([]byte(test.raw), &envelope))
			require.IsType(t, test.want, envelope.Spec)
		})
	}
}

func TestQueryEnvelopeRejectsRetiredQueryType(t *testing.T) {
	var envelope QueryEnvelope
	err := json.Unmarshal(
		[]byte(`{"type":"builder_trace_operator","spec":{"name":"T","expression":"A -> B"}}`),
		&envelope,
	)
	require.ErrorContains(t, err, "unknown query type")
}

func TestUseDefaultOrderByForLiteListQuery(t *testing.T) {
	envelope := QueryEnvelope{
		Type: QueryTypeBuilder,
		Spec: QueryBuilderQuery[LogAggregation]{
			Name: "A",
		},
	}

	envelope.UseDefaultOrderByForListQuery()
	require.Len(t, envelope.GetOrder(), 2)
	require.Equal(t, "timestamp", envelope.GetOrder()[0].Key.Name)
	require.Equal(t, "id", envelope.GetOrder()[1].Key.Name)
}
