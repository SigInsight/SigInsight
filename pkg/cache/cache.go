package cache

import (
	"context"
	"time"

	"github.com/SigNoz/signoz/pkg/types/cachetypes"
	"github.com/SigNoz/signoz/pkg/valuer"
)

type Cache interface {
	// Set sets the cacheable entity in cache.
	Set(ctx context.Context, orgID valuer.UUID, cacheKey string, data cachetypes.Cacheable, ttl time.Duration) error

	// Get gets the cacheble entity in the dest entity passed.
	Get(ctx context.Context, orgID valuer.UUID, cacheKey string, dest cachetypes.Cacheable) error

	// Delete deletes the cacheable entity from cache
	Delete(ctx context.Context, orgID valuer.UUID, cacheKey string)

	// DeleteMany deletes multiple cacheble entities from cache
	DeleteMany(ctx context.Context, orgID valuer.UUID, cacheKeys []string)
}
