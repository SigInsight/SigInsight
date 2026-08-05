package sqlmigration

import (
	"context"
	"database/sql"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/uptrace/bun"
	"github.com/uptrace/bun/dialect/sqlitedialect"
	_ "modernc.org/sqlite"
)

const currentAlertRule = `{
	"alert":"cpu usage",
	"alertType":"METRIC_BASED_ALERT",
	"ruleType":"threshold_rule",
	"version":"v5",
	"schemaVersion":"v3alpha1",
	"condition":{
		"kind":"numeric",
		"compositeQuery":{"queryType":"builder","queries":[{"type":"builder_query","spec":{"name":"A","signal":"metrics","aggregations":[{"metricName":"cpu_usage","spaceAggregation":"sum"}],"stepInterval":"1m"}}]},
		"selectedQueryName":"A",
		"numeric":{"reduction":"at_least_once","operator":"gt","thresholds":[{"severity":"critical","target":90,"channels":["email"]}]}
	},
	"evaluation":{"kind":"rolling","spec":{"evalWindow":"5m","frequency":"1m"}},
	"notificationSettings":{"groupBy":[]}
}`

func TestDropRetiredAlertRulesKeepsOnlyCurrentContract(t *testing.T) {
	database, err := sql.Open("sqlite", ":memory:")
	require.NoError(t, err)
	db := bun.NewDB(database, sqlitedialect.New())
	t.Cleanup(func() { require.NoError(t, db.Close()) })

	ctx := context.Background()
	_, err = db.ExecContext(ctx, `CREATE TABLE rule (id TEXT PRIMARY KEY, data TEXT NOT NULL)`)
	require.NoError(t, err)
	for _, row := range []struct {
		id   string
		data string
	}{
		{id: "current", data: currentAlertRule},
		{id: "v2", data: `{"schemaVersion":"v2alpha1"}`},
		{id: "malformed", data: "not json"},
		{id: "incomplete-v3", data: `{"schemaVersion":"v3alpha1"}`},
	} {
		_, err = db.ExecContext(ctx, `INSERT INTO rule (id, data) VALUES (?, ?)`, row.id, row.data)
		require.NoError(t, err)
	}

	require.NoError(t, (&dropRetiredAlertRules{}).Up(ctx, db))

	var ids []string
	require.NoError(t, db.NewSelect().Table("rule").Column("id").Order("id").Scan(ctx, &ids))
	require.Equal(t, []string{"current"}, ids)
}
