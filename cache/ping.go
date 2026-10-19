package cache

import (
	"context"

	"github.com/baby-hero-404/gocore/stderr"
	"github.com/redis/go-redis/v9"
)

// Ping checks if the Redis server is responding.
func (i *impl) Ping(ctx context.Context) error {
	return Ping(ctx, i.client)
}

// Ping checks if the Redis server is responding on a *redis.Client.
func Ping(ctx context.Context, client *redis.Client) error {
	if client == nil {
		return stderr.NewBadRequest("redis client handle is nil")
	}
	if err := client.Ping(ctx).Err(); err != nil {
		return WrapError(err)
	}
	return nil
}
