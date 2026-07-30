package alertmanager

import (
	"context"
	"testing"
	"time"

	"github.com/SigNoz/signoz/pkg/alertmanager/alertmanagerserver"
	"github.com/stretchr/testify/require"
)

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
