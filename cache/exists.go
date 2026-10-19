package cache

import "context"

// Exists checks how many of the given keys exist.
func (i *impl) Exists(ctx context.Context, keys ...string) (int64, error) {
	n, err := i.client.Exists(ctx, keys...).Result()
	if err != nil {
		return 0, WrapError(err)
	}
	return n, nil
}
