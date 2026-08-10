package queryparser

import (
	"context"

	"github.com/SigNoz/signoz/pkg/queryparser/queryfilterextractor"
	qbtypes "github.com/SigNoz/signoz/pkg/types/querybuildertypes/querybuildertypesv5"
)

// QueryParser defines the interface for parsing and analyzing queries.
type QueryParser interface {
	// AnalyzeQueryEnvelopes extracts filter conditions from a list of query envelopes.
	// Returns a map of query name to FilterResult.
	AnalyzeQueryEnvelopes(ctx context.Context, queries []qbtypes.QueryEnvelope) (map[string]*queryfilterextractor.FilterResult, error)
}
