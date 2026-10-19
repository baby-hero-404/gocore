package cache

import (
	"context"
	"time"
)

// Set stores a key-value pair with expiration.
func (i *impl) Set(ctx context.Context, key string, value any, exp time.Duration) error {
	if err := i.client.Set(ctx, key, value, exp).Err(); err != nil {
		return WrapError(err)
	}
	return nil
}
