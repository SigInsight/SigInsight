package savedviewtypes

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/require"

	qbtypes "github.com/SigNoz/signoz/pkg/types/querybuildertypes/querybuildertypesv5"
)

func validView() *View {
	return &View{
		CompositeQuery: &CompositeQuery{
			Queries: []qbtypes.QueryEnvelope{{
				Type: qbtypes.QueryTypeClickHouseSQL,
				Spec: qbtypes.ClickHouseQuery{Name: "A", Query: "SELECT 1"},
			}},
			PanelType: PanelTypeTable,
			QueryType: QueryTypeClickHouseSQL,
		},
	}
}

func TestViewValidate(t *testing.T) {
	require.NoError(t, validView().Validate())

	view := validView()
	view.CompositeQuery.Queries = nil
	require.ErrorContains(t, view.Validate(), "at least one query")

	view = validView()
	view.CompositeQuery.PanelType = "invalid"
	require.ErrorContains(t, view.Validate(), "invalid panel type")

	view = validView()
	view.CompositeQuery.QueryType = "invalid"
	require.ErrorContains(t, view.Validate(), "invalid query type")

	require.ErrorContains(t, (&View{}).Validate(), "composite query is required")
}

func TestCompositeQueryRejectsLegacyFields(t *testing.T) {
	var query CompositeQuery
	err := json.Unmarshal([]byte(`{
		"queries": [],
		"panelType": "table",
		"queryType": "builder",
		"builderQueries": {}
	}`), &query)

	require.ErrorContains(t, err, `unknown field "builderQueries"`)
}
