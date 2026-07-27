package quickfilter

import (
	"context"
	"net/http"

	"github.com/SigNoz/signoz/pkg/query-service/model/querytypes"
	"github.com/SigNoz/signoz/pkg/types/quickfiltertypes"
	"github.com/SigNoz/signoz/pkg/valuer"
)

type Module interface {
	GetQuickFilters(ctx context.Context, orgID valuer.UUID) ([]*quickfiltertypes.SignalFilters, error)
	UpdateQuickFilters(ctx context.Context, orgID valuer.UUID, signal quickfiltertypes.Signal, filters []querytypes.AttributeKey) error
	GetSignalFilters(ctx context.Context, orgID valuer.UUID, signal quickfiltertypes.Signal) (*quickfiltertypes.SignalFilters, error)
	SetDefaultConfig(ctx context.Context, orgID valuer.UUID) error
}

type Handler interface {
	GetQuickFilters(http.ResponseWriter, *http.Request)
	UpdateQuickFilters(http.ResponseWriter, *http.Request)
	GetSignalFilters(http.ResponseWriter, *http.Request)
}
