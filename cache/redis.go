package cache

import (
	"context"
	"time"

	"data-hub/config"

	"github.com/redis/go-redis/v9"
)

// Cache wraps Redis with independent read/write control.
// Writes always happen when Redis is reachable (keeps the cache warm).
// Reads are only served when CACHE_ENABLED=true, so disabling it forces
// every request through to upstream while still populating Redis.
type Cache struct {
	client      *redis.Client
	readEnabled bool
}

// NewCache creates a Cache.
// If REDIS_ADDR is empty, both reads and writes are no-ops.
// If REDIS_ADDR is set, writes always happen; reads only happen when CACHE_ENABLED=true.
func NewCache(cfg *config.Config) *Cache {
	if cfg.RedisAddr == "" {
		return &Cache{}
	}
	return &Cache{
		client:      redis.NewClient(&redis.Options{Addr: cfg.RedisAddr}),
		readEnabled: cfg.CacheEnabled,
	}
}

// Get retrieves a cached value. Returns "", nil on miss or when reads are disabled.
func (c *Cache) Get(ctx context.Context, key string) (string, error) {
	if !c.readEnabled || c.client == nil {
		return "", nil
	}
	val, err := c.client.Get(ctx, key).Result()
	if err == redis.Nil {
		return "", nil // cache miss is not an error
	}
	return val, err
}

// Set stores a value with a TTL. Always writes when Redis is reachable, regardless of CACHE_ENABLED.
func (c *Cache) Set(ctx context.Context, key string, value string, ttl time.Duration) error {
	if c.client == nil {
		return nil
	}
	return c.client.Set(ctx, key, value, ttl).Err()
}