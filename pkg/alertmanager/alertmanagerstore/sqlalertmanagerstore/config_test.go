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
		"condition":{"thresholds":{"kind":"basic","spec":[
			{"channels":["critical", "pager"]},
			{"channels":["pager", "email", "critical"]}
		]}}
	}`)
	legacyID := insertRule(orgID, 0, `{"preferredChannels":["legacy"]}`)
	insertRule("other-org", 0, `{"preferredChannels":["other-org"]}`)
	insertRule(orgID, 1, `{"preferredChannels":["deleted"]}`)
	insertRule(orgID, 0, `{not-json}`)

	matchers, err := NewConfigStore(&testSQLStore{db: db}).GetMatchers(ctx, orgID)
	require.NoError(t, err)
	require.Equal(t, map[string][]string{
		currentID: {"critical", "pager", "email"},
		legacyID:  {"legacy"},
	}, matchers)
}
