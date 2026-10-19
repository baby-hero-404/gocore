package lock

import (
	"github.com/go-redsync/redsync/v4"
	"github.com/go-redsync/redsync/v4/redis/goredis/v9"
	"github.com/redis/go-redis/v9"
)

type redisLocker struct {
	client *redis.Client
	rs     *redsync.Redsync
}

type redisLock struct {
	mutex *redsync.Mutex
	key   string
}

// New initializes a Locker backed by Redis Redlock.
func New(client *redis.Client) Locker {
	if client == nil {
		return nil
	}
	pool := goredis.NewPool(client)
	return &redisLocker{
		client: client,
		rs:     redsync.New(pool),
	}
}
