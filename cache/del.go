package cache

import "context"

// Del removes one or more keys from the cache.
func (i *impl) Del(ctx context.Context, keys ...string) (int64, error) {
	n, err := i.client.Del(ctx, keys...).Result()
	if err != nil {
		return 0, WrapError(err)
	}
	return n, nil
}
