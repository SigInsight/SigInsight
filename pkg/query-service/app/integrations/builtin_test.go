package integrations

import (
	"context"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestBuiltinIntegrations(t *testing.T) {
	require := require.New(t)

	repo := BuiltInIntegrations{}

	builtins, apiErr := repo.list(context.Background())
	require.Nil(apiErr)
	require.ElementsMatch([]string{
		"builtin-clickhouse",
		"builtin-mongo",
		"builtin-nginx",
		"builtin-postgres",
		"builtin-redis",
	}, integrationIDs(builtins))

	nginxIntegrationId := "builtin-nginx"
	res, apiErr := repo.get(context.Background(), []string{
		nginxIntegrationId,
	})
	require.Nil(apiErr)

	nginxIntegration, exists := res[nginxIntegrationId]
	require.True(exists)
	require.False(strings.HasPrefix(nginxIntegration.Overview, "file://"))
}

func integrationIDs(integrations []IntegrationDetails) []string {
	ids := make([]string, 0, len(integrations))
	for _, integration := range integrations {
		ids = append(ids, integration.Id)
	}
	return ids
}
