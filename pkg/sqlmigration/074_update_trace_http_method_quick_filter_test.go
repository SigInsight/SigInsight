package sqlmigration

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMigrateTraceHTTPMethodQuickFilter(t *testing.T) {
	filterJSON := `[{"custom":"without-key"},{"key":"service.name","dataType":"string","type":"resource","custom":true},{"key":"http.method","dataType":"string","type":"tag"},{"key":"http.method.extra","dataType":"string","type":"tag"}]`

	updatedFilter, changed, err := migrateTraceHTTPMethodQuickFilter(filterJSON)
	require.NoError(t, err)
	assert.True(t, changed)

	var filters []map[string]interface{}
	require.NoError(t, json.Unmarshal([]byte(updatedFilter), &filters))
	require.Len(t, filters, 4)
	assert.Equal(t, "without-key", filters[0]["custom"])
	assert.Equal(t, "service.name", filters[1]["key"])
	assert.Equal(t, true, filters[1]["custom"])
	assert.Equal(t, "http_method", filters[2]["key"])
	assert.Equal(t, "string", filters[2]["dataType"])
	assert.Equal(t, "tag", filters[2]["type"])
	assert.Equal(t, "http.method.extra", filters[3]["key"])
}

func TestMigrateTraceHTTPMethodQuickFilterAlreadyMigrated(t *testing.T) {
	filterJSON := `[{"key":"http_method","dataType":"string","type":"tag"}]`

	updatedFilter, changed, err := migrateTraceHTTPMethodQuickFilter(filterJSON)
	require.NoError(t, err)
	assert.False(t, changed)
	assert.Equal(t, filterJSON, updatedFilter)
}
