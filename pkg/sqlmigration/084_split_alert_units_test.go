package sqlmigration

import (
	"context"
	"database/sql"
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/uptrace/bun"
	"github.com/uptrace/bun/dialect/sqlitedialect"
	_ "modernc.org/sqlite"
)

func TestSplitAlertUnitsMigratesLegacyLogCountRule(t *testing.T) {
	database, err := sql.Open("sqlite", ":memory:")
	require.NoError(t, err)
	db := bun.NewDB(database, sqlitedialect.New())
	t.Cleanup(func() { require.NoError(t, db.Close()) })

	ctx := context.Background()
	_, err = db.ExecContext(ctx, `CREATE TABLE rule (id TEXT PRIMARY KEY, data TEXT NOT NULL)`)
	require.NoError(t, err)
	legacyRule := `{
		"alert":"log count",
		"alertType":"LOGS_BASED_ALERT",
		"ruleType":"threshold_rule",
		"condition":{
			"thresholds":{"kind":"basic","spec":[{"name":"critical","target":1,"targetUnit":"s","matchType":"1","op":"1","channels":[]}]},
			"compositeQuery":{"queryType":"builder","panelType":"graph","unit":"s","queries":[{"type":"builder_query","spec":{"name":"A","signal":"logs","aggregations":[{"expression":"count()"}]}}]},
			"selectedQueryName":"A"
		},
		"labels":{},
		"annotations":{},
		"version":"v5",
		"schemaVersion":"v2alpha1"
	}`
	_, err = db.ExecContext(ctx, `INSERT INTO rule (id, data) VALUES (?, ?)`, "rule-1", legacyRule)
	require.NoError(t, err)

	require.NoError(t, (&splitAlertUnits{}).Up(ctx, db))

	var stored string
	require.NoError(t, db.NewSelect().Table("rule").Column("data").Where("id = ?", "rule-1").Scan(ctx, &stored))
	var decoded map[string]any
	require.NoError(t, json.Unmarshal([]byte(stored), &decoded))
	condition := decoded["condition"].(map[string]any)
	query := condition["compositeQuery"].(map[string]any)
	require.NotContains(t, query, "unit")
	require.Equal(t, "{count}", query["resultUnit"])
	require.Equal(t, "{count}", query["displayUnit"])
	threshold := condition["thresholds"].(map[string]any)["spec"].([]any)[0].(map[string]any)
	require.Equal(t, "{count}", threshold["targetUnit"])
}
