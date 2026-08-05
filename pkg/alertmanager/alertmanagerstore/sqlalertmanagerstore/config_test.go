package sqlalertmanagerstore

import (
	"context"
	"database/sql"
	"testing"

	"github.com/SigNoz/signoz/pkg/sqlstore"
	"github.com/SigNoz/signoz/pkg/types"
	"github.com/SigNoz/signoz/pkg/types/ruletypes"
	"github.com/SigNoz/signoz/pkg/valuer"
	"github.com/stretchr/testify/require"
	"github.com/uptrace/bun"
	"github.com/uptrace/bun/dialect/sqlitedialect"
	_ "modernc.org/sqlite"
)

type testSQLStore struct {
	sqlstore.SQLStore
	db *bun.DB
}

func (store *testSQLStore) BunDB() *bun.DB { return store.db }

func TestGetMatchersUsesCurrentRuleChannelsAndScopesRules(t *testing.T) {
	database, err := sql.Open("sqlite", ":memory:")
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, database.Close()) })

	db := bun.NewDB(database, sqlitedialect.New())
	t.Cleanup(func() { require.NoError(t, db.Close()) })
	ctx := context.Background()
	require.NoError(t, db.ResetModel(ctx, (*ruletypes.Rule)(nil)))

	orgID := "org-a"
	insertRule := func(orgID string, deleted int, data string) string {
		rule := &ruletypes.Rule{
			Identifiable: types.Identifiable{ID: valuer.GenerateUUID()},
			Deleted:      deleted,
			Data:         data,
			OrgID:        orgID,
		}
		_, err := db.NewInsert().Model(rule).Exec(ctx)
		require.NoError(t, err)
		return rule.ID.StringValue()
	}

	currentID := insertRule(orgID, 0, `{
		"alert":"cpu usage","alertType":"METRIC_BASED_ALERT","ruleType":"threshold_rule","version":"v5","schemaVersion":"v3alpha1",
		"condition":{"kind":"numeric","compositeQuery":{"queryType":"builder","queries":[{"type":"builder_query","spec":{"name":"A","signal":"metrics","aggregations":[{"metricName":"cpu_usage","spaceAggregation":"sum"}],"stepInterval":"1m"}}]},"selectedQueryName":"A","numeric":{"reduction":"at_least_once","operator":"gt","thresholds":[
			{"severity":"critical","target":90,"channels":["critical","pager"]},
			{"severity":"warning","target":80,"channels":["pager","email","critical"]}
		]}},
		"evaluation":{"kind":"rolling","spec":{"evalWindow":"5m","frequency":"1m"}},"notificationSettings":{"groupBy":[]}
	}`)
	insertRule(orgID, 0, `{"schemaVersion":"v2alpha1","preferredChannels":["legacy"]}`)
	insertRule("other-org", 0, `{"schemaVersion":"v2alpha1","preferredChannels":["other-org"]}`)
	insertRule(orgID, 1, `{"schemaVersion":"v2alpha1","preferredChannels":["deleted"]}`)
	insertRule(orgID, 0, `{not-json}`)

	matchers, err := NewConfigStore(&testSQLStore{db: db}).GetMatchers(ctx, orgID)
	require.NoError(t, err)
	require.Equal(t, map[string][]string{
		currentID: {"critical", "pager", "email"},
	}, matchers)
}
