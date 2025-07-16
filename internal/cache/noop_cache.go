package cache

import (
	"context"
	"time"
)

// NoOpCache is a cache implementation that does nothing. It's used when
// caching is disabled or unavailable, allowing the application to run
// without caching logic causing errors.
type NoOpCache struct{}

func (n *NoOpCache) Get(ctx context.Context, key string) ([]byte, error) {
	return nil, nil
}

func (n *NoOpCache) Set(ctx context.Context, key string, value []byte, ttl time.Duration) error {
	return nil
}

func (n *NoOpCache) Close() error {
	return nil
}
