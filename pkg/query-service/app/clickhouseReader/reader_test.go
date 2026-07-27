package clickhouseReader

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

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
