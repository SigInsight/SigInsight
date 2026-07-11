package assistant

import (
	"context"
	"errors"
	"net/http"
)

// ErrConfigNotFound is returned by Store.GetConfig when no config exists for the given org.
var ErrConfigNotFound = errors.New("assistant config not found")

const (
	DefaultBaseURL = "https://api.openai.com/v1"
	DefaultModel   = "gpt-4o-mini"
)

type Config struct {
	BaseURL string
	Model   string
	APIKey  string
}

type ConfigResponse struct {
	BaseURL          string `json:"baseUrl"`
	Model            string `json:"model"`
	APIKeyConfigured bool   `json:"apiKeyConfigured"`
}

type UpdatableConfig struct {
	BaseURL string  `json:"baseUrl"`
	Model   string  `json:"model"`
	APIKey  *string `json:"apiKey,omitempty"`
}

type ChatMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type TimeRange struct {
	StartTime *int64 `json:"startTime,omitempty"`
	EndTime   *int64 `json:"endTime,omitempty"`
	Label     string `json:"label,omitempty"`
}

type SelectedEntity struct {
	Type string `json:"type"`
	ID   string `json:"id"`
	Name string `json:"name,omitempty"`
}

type QuerySummary struct {
	Name               string `json:"name"`
	DataSource         string `json:"dataSource"`
	AggregateOperator  string `json:"aggregateOperator,omitempty"`
	AggregateAttribute string `json:"aggregateAttribute,omitempty"`
	FilterExpression   string `json:"filterExpression,omitempty"`
	GroupByCount       int    `json:"groupByCount"`
	Disabled           bool   `json:"disabled"`
}

type VisibleDataSummary struct {
	Description string `json:"description,omitempty"`
	RowCount    *int   `json:"rowCount,omitempty"`
	SeriesCount *int   `json:"seriesCount,omitempty"`
}

type ContextSnapshot struct {
	Route              string              `json:"route"`
	Search             string              `json:"search"`
	QueryType          string              `json:"queryType,omitempty"`
	PanelType          string              `json:"panelType,omitempty"`
	InitialDataSource  string              `json:"initialDataSource,omitempty"`
	TimeRange          *TimeRange          `json:"timeRange,omitempty"`
	SelectedEntity     *SelectedEntity     `json:"selectedEntity,omitempty"`
	VisibleDataSummary *VisibleDataSummary `json:"visibleDataSummary,omitempty"`
	QueryCount         int                 `json:"queryCount"`
	FormulaCount       int                 `json:"formulaCount"`
	TraceOperatorCount int                 `json:"traceOperatorCount"`
	Queries            []QuerySummary      `json:"queries"`
	CapturedAt         int64               `json:"capturedAt"`
}

type ChatRequest struct {
	Messages []ChatMessage   `json:"messages"`
	Context  ContextSnapshot `json:"context"`
}

type StreamEvent struct {
	Type    string `json:"-"`
	Delta   string `json:"delta,omitempty"`
	Message string `json:"message,omitempty"`
}

type Store interface {
	GetConfig(ctx context.Context, orgID string) (*Config, error)
	UpsertConfig(ctx context.Context, orgID string, config Config) error
}

type Module interface {
	GetConfig(ctx context.Context, orgID string) (*ConfigResponse, error)
	UpdateConfig(ctx context.Context, orgID string, input UpdatableConfig) error
	Chat(ctx context.Context, orgID string, request ChatRequest, emit func(StreamEvent) error) error
}

type Handler interface {
	GetConfig(rw http.ResponseWriter, r *http.Request)
	UpdateConfig(rw http.ResponseWriter, r *http.Request)
	Chat(rw http.ResponseWriter, r *http.Request)
}
