package ruletypes

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/SigNoz/signoz/pkg/query-service/model/querytypes"
	qbtypes "github.com/SigNoz/signoz/pkg/types/querybuildertypes/querybuildertypesv5"
)

func TestRuleConditionGetSelectedQueryName(t *testing.T) {
	testCases := []struct {
		name      string
		condition *RuleCondition
		expected  string
	}{
		{name: "nil condition"},
		{name: "missing selected query", condition: &RuleCondition{CompositeQuery: v5CompositeQuery("F1")}},
		{name: "explicit selected query", condition: &RuleCondition{CompositeQuery: v5CompositeQuery("A"), SelectedQuery: "A"}, expected: "A"},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			assert.Equal(t, testCase.expected, testCase.condition.GetSelectedQueryName())
		})
	}
}

func TestRuleConditionIsValidRequiresV5Queries(t *testing.T) {
	require.False(t, (*RuleCondition)(nil).IsValid())

	condition := &RuleCondition{
		CompositeQuery: &CompositeQuery{
			QueryType: querytypes.QueryTypePromQL,
		},
	}

	require.False(t, condition.IsValid())

	condition.CompositeQuery = v5CompositeQuery("A")
	require.True(t, condition.IsValid())
}

func TestCompositeQueryRejectsLegacyFields(t *testing.T) {
	for _, field := range []string{"builderQueries", "promQueries", "chQueries"} {
		t.Run(field, func(t *testing.T) {
			var compositeQuery CompositeQuery
			err := json.Unmarshal([]byte(`{"queries":[],"`+field+`":{}}`), &compositeQuery)
			require.ErrorContains(t, err, "unsupported alert composite query field")
		})
	}
}

func v5CompositeQuery(name string) *CompositeQuery {
	return &CompositeQuery{
		QueryType: querytypes.QueryTypePromQL,
		Queries: []qbtypes.QueryEnvelope{{
			Type: qbtypes.QueryTypePromQL,
			Spec: qbtypes.PromQuery{Name: name, Query: "up"},
		}},
	}
}
