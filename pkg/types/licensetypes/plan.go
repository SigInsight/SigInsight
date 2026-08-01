package licensetypes

import (
	"github.com/SigNoz/signoz/pkg/valuer"
)

var (
	// Feature key for the dot-metrics (delta-to-cumulative) toggle.
	DotMetricsEnabled = valuer.NewString("dot_metrics_enabled")
	// Feature key advertised when the running server can execute the
	// lightweight V5 query subset.
)

// Feature represents a single feature flag entry returned by the
// /api/v5/features/ui endpoint. The community build returns a small static
// set assembled in pkg/query-service/app/http_handler.go:getFeatureFlags.
type Feature struct {
	Name       valuer.String `json:"name"`
	Active     bool          `json:"active"`
	Usage      int64         `json:"usage"`
	UsageLimit int64         `json:"usage_limit"`
	Route      string        `json:"route"`
}
