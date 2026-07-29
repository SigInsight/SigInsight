package quickfiltertypes

import (
	"encoding/json"
	"testing"

	"github.com/SigNoz/signoz/pkg/query-service/model/querytypes"
	"github.com/SigNoz/signoz/pkg/valuer"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewDefaultQuickFilterUsesCanonicalHTTPMethod(t *testing.T) {
	quickFilters, err := NewDefaultQuickFilter(valuer.GenerateUUID())
	require.NoError(t, err)

	var tracesFilter *StorableQuickFilter
	for _, filter := range quickFilters {
		if filter.Signal == SignalTraces {
			tracesFilter = filter
			break
		}
	}
	require.NotNil(t, tracesFilter)

	var filters []querytypes.AttributeKey
	require.NoError(t, json.Unmarshal([]byte(tracesFilter.Filter), &filters))

	var keys []string
	for _, filter := range filters {
		keys = append(keys, filter.Key)
	}
	assert.Contains(t, keys, "http_method")
	assert.NotContains(t, keys, "http.method")
	assert.Contains(t, keys, "has_error")
	assert.NotContains(t, keys, "hasError")
}

func TestNewDefaultQuickFilterExcludesUnusedResourceFilters(t *testing.T) {
	quickFilters, err := NewDefaultQuickFilter(valuer.GenerateUUID())
	require.NoError(t, err)

	keysBySignal := map[string][]string{}
	for _, quickFilter := range quickFilters {
		var filters []querytypes.AttributeKey
		require.NoError(t, json.Unmarshal([]byte(quickFilter.Filter), &filters))
		for _, filter := range filters {
			keysBySignal[quickFilter.Signal.StringValue()] = append(keysBySignal[quickFilter.Signal.StringValue()], filter.Key)
		}
	}

	assert.NotContains(t, keysBySignal[SignalExceptions.StringValue()], "k8s.cluster.name")
	assert.NotContains(t, keysBySignal[SignalExceptions.StringValue()], "k8s.pod.name")
	assert.NotContains(t, keysBySignal[SignalLogs.StringValue()], "k8s.cluster.name")
	assert.NotContains(t, keysBySignal[SignalLogs.StringValue()], "k8s.pod.name")
	for signal, keys := range keysBySignal {
		assert.NotContains(t, keys, "deployment.environment", signal)
	}
}

func TestNewDefaultQuickFilterKeepsSeverityTextInLogContext(t *testing.T) {
	quickFilters, err := NewDefaultQuickFilter(valuer.GenerateUUID())
	require.NoError(t, err)

	var logsFilter *StorableQuickFilter
	for _, filter := range quickFilters {
		if filter.Signal == SignalLogs {
			logsFilter = filter
			break
		}
	}
	require.NotNil(t, logsFilter)

	var filters []querytypes.AttributeKey
	require.NoError(t, json.Unmarshal([]byte(logsFilter.Filter), &filters))
	for _, filter := range filters {
		if filter.Key == "severity_text" {
			assert.Empty(t, filter.Type)
			return
		}
	}
	t.Fatal("severity_text must be present in the default logs quick filters")
}
