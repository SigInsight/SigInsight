package savedviewtypes

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/SigNoz/signoz/pkg/types"
	qbtypes "github.com/SigNoz/signoz/pkg/types/querybuildertypes/querybuildertypesv5"
	"github.com/SigNoz/signoz/pkg/valuer"
	"github.com/uptrace/bun"
)

type SavedView struct {
	bun.BaseModel `bun:"table:saved_views"`

	types.Identifiable
	types.TimeAuditable
	types.UserAuditable
	OrgID      string `json:"orgId" bun:"org_id,notnull"`
	Name       string `json:"name" bun:"name,type:text,notnull"`
	Category   string `json:"category" bun:"category,type:text,notnull"`
	SourcePage string `json:"sourcePage" bun:"source_page,type:text,notnull"`
	Tags       string `json:"tags" bun:"tags,type:text"`
	Data       string `json:"data" bun:"data,type:text,notnull"`
	ExtraData  string `json:"extraData" bun:"extra_data,type:text"`
}

func NewStatsFromSavedViews(savedViews []*SavedView) map[string]any {
	stats := make(map[string]any)
	for _, savedView := range savedViews {
		key := "savedview.source." + strings.ToLower(string(savedView.SourcePage)) + ".count"
		if _, ok := stats[key]; !ok {
			stats[key] = int64(1)
		} else {
			stats[key] = stats[key].(int64) + 1
		}
	}
	stats["savedview.count"] = int64(len(savedViews))
	return stats
}

type QueryType string

const (
	QueryTypeBuilder       QueryType = "builder"
	QueryTypeClickHouseSQL QueryType = "clickhouse_sql"
	QueryTypePromQL        QueryType = "promql"
)

func (queryType QueryType) Validate() error {
	switch queryType {
	case QueryTypeBuilder, QueryTypeClickHouseSQL, QueryTypePromQL:
		return nil
	default:
		return fmt.Errorf("invalid query type: %s", queryType)
	}
}

type PanelType string

const (
	PanelTypeValue PanelType = "value"
	PanelTypeGraph PanelType = "graph"
	PanelTypeTable PanelType = "table"
	PanelTypeList  PanelType = "list"
	PanelTypeTrace PanelType = "trace"
)

func (panelType PanelType) Validate() error {
	switch panelType {
	case PanelTypeValue, PanelTypeGraph, PanelTypeTable, PanelTypeList, PanelTypeTrace:
		return nil
	default:
		return fmt.Errorf("invalid panel type: %s", panelType)
	}
}

type CompositeQuery struct {
	Queries   []qbtypes.QueryEnvelope `json:"queries"`
	PanelType PanelType               `json:"panelType"`
	QueryType QueryType               `json:"queryType"`
	Unit      string                  `json:"unit,omitempty"`
	FillGaps  bool                    `json:"fillGaps,omitempty"`
}

func (query *CompositeQuery) UnmarshalJSON(data []byte) error {
	type alias CompositeQuery
	var value alias
	if err := json.Unmarshal(data, &value); err != nil {
		return err
	}

	var fields map[string]json.RawMessage
	if err := json.Unmarshal(data, &fields); err != nil {
		return err
	}
	validFields := map[string]struct{}{
		"queries":   {},
		"panelType": {},
		"queryType": {},
		"unit":      {},
		"fillGaps":  {},
	}
	for field := range fields {
		if _, ok := validFields[field]; !ok {
			return fmt.Errorf("unknown field %q in saved view composite query", field)
		}
	}

	*query = CompositeQuery(value)
	return nil
}

func (query *CompositeQuery) Validate() error {
	if query == nil {
		return fmt.Errorf("composite query is required")
	}
	if len(query.Queries) == 0 {
		return fmt.Errorf("composite query must contain at least one query")
	}
	if err := query.PanelType.Validate(); err != nil {
		return fmt.Errorf("panel type is invalid: %w", err)
	}
	if err := query.QueryType.Validate(); err != nil {
		return fmt.Errorf("query type is invalid: %w", err)
	}
	return nil
}

type View struct {
	ID             valuer.UUID     `json:"id,omitempty"`
	Name           string          `json:"name"`
	Category       string          `json:"category"`
	CreatedAt      time.Time       `json:"createdAt"`
	CreatedBy      string          `json:"createdBy"`
	UpdatedAt      time.Time       `json:"updatedAt"`
	UpdatedBy      string          `json:"updatedBy"`
	SourcePage     string          `json:"sourcePage"`
	Tags           []string        `json:"tags"`
	CompositeQuery *CompositeQuery `json:"compositeQuery"`
	ExtraData      string          `json:"extraData"`
}

func (view *View) Validate() error {
	if view.CompositeQuery == nil {
		return fmt.Errorf("composite query is required")
	}
	return view.CompositeQuery.Validate()
}
