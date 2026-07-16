package quickfiltertypes

import (
	"encoding/json"
	"testing"

	v3 "github.com/SigNoz/signoz/pkg/query-service/model/v3"
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

	var filters []v3.AttributeKey
	require.NoError(t, json.Unmarshal([]byte(tracesFilter.Filter), &filters))

	var keys []string
	for _, filter := range filters {
		keys = append(keys, filter.Key)
	}
	assert.Contains(t, keys, "http_method")
	assert.NotContains(t, keys, "http.method")
}
