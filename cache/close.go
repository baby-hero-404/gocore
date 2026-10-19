package cache

import "github.com/redis/go-redis/v9"

// Close gracefully closes the Redis client connection pool.
func (i *impl) Close() error {
	return Close(i.client)
}

// Close gracefully closes the Redis client connection pool on a *redis.Client.
func Close(client *redis.Client) error {
	if client == nil {
		return nil
	}
	if err := client.Close(); err != nil {
		return WrapError(err)
	}
	return nil
}
