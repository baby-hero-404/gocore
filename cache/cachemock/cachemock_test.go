package cachemock

import (
	"context"
	"github.com/redis/go-redis/v9"
	"testing"
	"time"
)

func TestMockCache(t *testing.T) {
	mockCache := NewCache(t)
	ctx := context.Background()

	mockCache.On("Ping", ctx).Return(nil)
	mockCache.On("Close").Return(nil)
	mockCache.On("Client").Return((*redis.Client)(nil))
	mockCache.On("Get", ctx, "user:1").Return("value", nil)
	mockCache.On("Set", ctx, "user:1", "value", time.Minute).Return(nil)
	mockCache.On("Del", ctx, []string{"user:1"}).Return(int64(1), nil)
	mockCache.On("Exists", ctx, []string{"user:1"}).Return(int64(1), nil)
	mockCache.On("Expire", ctx, "user:1", time.Minute).Return(true, nil)

	if err := mockCache.Ping(ctx); err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if err := mockCache.Close(); err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if c := mockCache.Client(); c != nil {
		t.Errorf("expected nil client")
	}
	val, err := mockCache.Get(ctx, "user:1")
	if err != nil || val != "value" {
		t.Errorf("Get failed: %v, %s", err, val)
	}
	if err := mockCache.Set(ctx, "user:1", "value", time.Minute); err != nil {
		t.Errorf("Set failed: %v", err)
	}
	if n, err := mockCache.Del(ctx, "user:1"); err != nil || n != 1 {
		t.Errorf("Del failed: %v, %d", err, n)
	}
	if n, err := mockCache.Exists(ctx, "user:1"); err != nil || n != 1 {
		t.Errorf("Exists failed: %v, %d", err, n)
	}
	if ok, err := mockCache.Expire(ctx, "user:1", time.Minute); err != nil || !ok {
		t.Errorf("Expire failed: %v, %v", err, ok)
	}
}
