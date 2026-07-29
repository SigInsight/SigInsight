package tracefunnelstore

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/require"

	chErrors "github.com/SigNoz/signoz/pkg/query-service/errors"
)

func TestQueryErrorMapsClickHouseLimits(t *testing.T) {
	testCases := []struct {
		name     string
		err      error
		expected error
	}{
		{name: "nil", err: nil, expected: nil},
		{name: "bytes", err: errors.New("clickhouse code: 307"), expected: chErrors.ErrResourceBytesLimitExceeded},
		{name: "time", err: errors.New("clickhouse code: 159"), expected: chErrors.ErrResourceTimeLimitExceeded},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			require.ErrorIs(t, queryError(testCase.err), testCase.expected)
		})
	}
}

func TestQueryErrorPreservesUnknownError(t *testing.T) {
	expected := errors.New("query failed")
	require.Same(t, expected, queryError(expected))
}
