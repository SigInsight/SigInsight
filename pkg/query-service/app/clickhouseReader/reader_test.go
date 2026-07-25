package clickhouseReader

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestTraceAutocompleteColumn(t *testing.T) {
	if got := traceAutocompleteColumn("http.route"); got != "attribute_string_http$$route" {
		t.Fatalf("traceAutocompleteColumn(http.route) = %q", got)
	}
	if got := traceAutocompleteColumn("service.name"); got != "resource_string_service$$name" {
		t.Fatalf("traceAutocompleteColumn(service.name) = %q", got)
	}
	if got := traceAutocompleteColumn("rpc.method"); got != "attribute_string_rpc$$method" {
		t.Fatalf("traceAutocompleteColumn(rpc.method) = %q", got)
	}
}

type GetStatusFiltersTest struct {
	query        string
	statusParams []string
	excludeMap   map[string]struct{}
	expected     string
}

func TestGetLocalTableName(t *testing.T) {
	assert := assert.New(t)

	assert.Equal("signoz_traces.signoz_index_v3", getLocalTableName("signoz_traces.distributed_signoz_index_v3"))
	assert.Equal("signoz_traces.signoz_index_v3", getLocalTableName("signoz_traces.signoz_index_v3"))
	assert.Equal("signoz_index_v3", getLocalTableName("signoz_index_v3"))
	assert.Equal(
		[]string{"signoz_logs.logs_v2", "signoz_traces.signoz_index_v3"},
		getLocalTableNameArray([]string{"signoz_logs.logs_v2", "signoz_traces.distributed_signoz_index_v3"}),
	)
}

func TestGetStatusFilters(t *testing.T) {
	assert := assert.New(t)
	var tests = []GetStatusFiltersTest{
		{"", make([]string, 0), map[string]struct{}{}, ""},
		{"test", []string{"error"}, map[string]struct{}{}, "test AND hasError = true"},
		{"test", []string{"ok"}, map[string]struct{}{}, "test AND hasError = false"},
		{"test", []string{"error"}, map[string]struct{}{"status": {}}, "test AND hasError = false"},
		{"test", []string{"ok"}, map[string]struct{}{"status": {}}, "test AND hasError = true"},
		{"test", []string{"error", "ok"}, map[string]struct{}{}, "test"},
	}
	for _, test := range tests {
		assert.Equal(getStatusFilters(test.query, test.statusParams, test.excludeMap), test.expected)
	}
}
