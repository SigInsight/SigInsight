package model

import (
	"encoding/json"
	"time"

	"github.com/SigNoz/signoz/pkg/query-service/model/querytypes"
)

// MetricMetadata is derived directly from collected metric time series.
type MetricMetadata struct {
	MetricName  string                 `json:"metricName" ch:"metric_name"`
	MetricType  querytypes.MetricType  `json:"metricType" ch:"type"`
	Description string                 `json:"description" ch:"description"`
	Unit        string                 `json:"unit" ch:"unit"`
	Temporality querytypes.Temporality `json:"temporality" ch:"temporality"`
	IsMonotonic bool                   `json:"is_monotonic" ch:"is_monotonic"`
	CreatedAt   time.Time              `json:"created_at" ch:"created_at"`
}

func (c *MetricMetadata) MarshalBinary() (data []byte, err error) {
	return json.Marshal(c)
}
func (c *MetricMetadata) UnmarshalBinary(data []byte) error {
	return json.Unmarshal(data, c)
}
