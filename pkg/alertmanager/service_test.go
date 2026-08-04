package alertmanager

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/SigNoz/signoz/pkg/alertmanager/alertmanagerserver"
	"github.com/SigNoz/signoz/pkg/factory"
	"github.com/SigNoz/signoz/pkg/factory/factorytest"
	"github.com/SigNoz/signoz/pkg/types"
	"github.com/SigNoz/signoz/pkg/valuer"
	"github.com/prometheus/alertmanager/matcher/compat"
	"github.com/prometheus/common/model"
	"github.com/stretchr/testify/require"
)

type emptyOrganizationGetter struct{}

func (emptyOrganizationGetter) Get(context.Context, valuer.UUID) (*types.Organization, error) {
	return &types.Organization{}, nil
}

func (emptyOrganizationGetter) GetByIDOrName(context.Context, valuer.UUID, string) (*types.Organization, bool, error) {
	return &types.Organization{}, false, nil
}

func (emptyOrganizationGetter) ListByOwnedKeyRange(context.Context) ([]*types.Organization, error) {
	return nil, nil
}

func (emptyOrganizationGetter) GetByName(context.Context, string) (*types.Organization, error) {
	return &types.Organization{}, nil
}

func TestSyncServersDoesNotMutateMatcherCompatibility(t *testing.T) {
	settings := factory.NewScopedProviderSettings(factorytest.NewSettings(), "test-alertmanager")
	service := New(
		context.Background(),
		settings,
		alertmanagerserver.Config{},
		nil,
		nil,
		emptyOrganizationGetter{},
		nil,
	)

	const iterations = 1000
	validNames := make([]bool, iterations)
	syncErrors := make([]error, iterations)
	var workers sync.WaitGroup
	workers.Add(2)
	go func() {
		defer workers.Done()
		for idx := range iterations {
			validNames[idx] = compat.IsValidLabelName(model.LabelName("alertname"))
		}
	}()
	go func() {
		defer workers.Done()
		for idx := range iterations {
			syncErrors[idx] = service.SyncServers(context.Background())
		}
	}()

	workers.Wait()
	for idx := range iterations {
		require.True(t, validNames[idx])
		require.NoError(t, syncErrors[idx])
	}
}

func TestServiceStopWaitsForServerSync(t *testing.T) {
	service := &Service{
		servers: make(map[string]*alertmanagerserver.Server),
	}

	service.serversMtx.Lock()
	stopResult := make(chan error, 1)
	go func() {
		stopResult <- service.Stop(context.Background())
	}()

	select {
	case err := <-stopResult:
		require.Failf(t, "Stop returned during server sync", "error: %v", err)
	case <-time.After(20 * time.Millisecond):
	}

	service.serversMtx.Unlock()
	require.NoError(t, <-stopResult)
}
