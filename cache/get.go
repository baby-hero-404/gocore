package cache

import "context"

// Get retrieves a value by key, wrapping Redis errors with stderr codes.
func (i *impl) Get(ctx context.Context, key string) (string, error) {
	val, err := i.client.Get(ctx, key).Result()
	if err != nil {
		return "", WrapError(err)
	}
	return val, nil
}
