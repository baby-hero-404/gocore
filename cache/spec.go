package cache

import (
	"context"
	"time"

	"github.com/redis/go-redis/v9"
)

// Cache defines the contract for Redis operations.
type Cache interface {
	// Client returns the underlying *redis.Client instance.
	Client() *redis.Client

	// Get retrieves a value by key.
	Get(ctx context.Context, key string) (string, error)

	// Set stores a key-value pair with expiration.
	Set(ctx context.Context, key string, value any, exp time.Duration) error

	// Del deletes one or more keys.
	Del(ctx context.Context, keys ...string) (int64, error)

	// Exists checks if keys exist.
	Exists(ctx context.Context, keys ...string) (int64, error)

	// Expire sets a key's expiration.
	Expire(ctx context.Context, key string, expiration time.Duration) (bool, error)

	// Ping checks Redis connectivity.
	Ping(ctx context.Context) error

	// Close gracefully terminates the connection pool.
	Close() error
}
