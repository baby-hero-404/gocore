// Package cachemock provides test doubles for github.com/baby-hero-404/gocore/cache.
package cachemock

import (
	"context"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/mock"
)

// Cache provides an in-memory mock of the Cache interface for unit testing downstream microservices.
type Cache struct {
	mock.Mock
}

// NewCache creates a new instance of Cache.
func NewCache(t mock.TestingT) *Cache {
	m := &Cache{}
	m.Test(t)
	return m
}

func (m *Cache) Client() *redis.Client {
	args := m.Called()
	if c, ok := args.Get(0).(*redis.Client); ok {
		return c
	}
	return nil
}

func (m *Cache) Get(ctx context.Context, key string) (string, error) {
	args := m.Called(ctx, key)
	return args.String(0), args.Error(1)
}

func (m *Cache) Set(ctx context.Context, key string, value any, exp time.Duration) error {
	args := m.Called(ctx, key, value, exp)
	return args.Error(0)
}

func (m *Cache) Del(ctx context.Context, keys ...string) (int64, error) {
	args := m.Called(ctx, keys)
	return args.Get(0).(int64), args.Error(1)
}

func (m *Cache) Exists(ctx context.Context, keys ...string) (int64, error) {
	args := m.Called(ctx, keys)
	return args.Get(0).(int64), args.Error(1)
}

func (m *Cache) Expire(ctx context.Context, key string, expiration time.Duration) (bool, error) {
	args := m.Called(ctx, key, expiration)
	return args.Bool(0), args.Error(1)
}

func (m *Cache) Ping(ctx context.Context) error {
	args := m.Called(ctx)
	return args.Error(0)
}

func (m *Cache) Close() error {
	args := m.Called()
	return args.Error(0)
}
