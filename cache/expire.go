package cache

import (
	"context"
	"time"
)

// Expire sets a TTL on the given key.
func (i *impl) Expire(ctx context.Context, key string, expiration time.Duration) (bool, error) {
	ok, err := i.client.Expire(ctx, key, expiration).Result()
	if err != nil {
		return false, WrapError(err)
	}
	return ok, nil
}
