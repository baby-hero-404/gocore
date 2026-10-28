package lock

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/baby-hero-404/gocore/stderr"
	"github.com/redis/go-redis/v9"
)

// unreachableClient points at a closed port so every Redis call fails fast.
func unreachableClient(t *testing.T) *redis.Client {
	t.Helper()
	c := redis.NewClient(&redis.Options{
		Addr:        "127.0.0.1:1",
		MaxRetries:  -1,
		DialTimeout: 100 * time.Millisecond,
	})
	t.Cleanup(func() { _ = c.Close() })
	return c
}

func TestAcquire_RedisUnavailable(t *testing.T) {
	ctx := context.Background()
	l, err := Acquire(ctx, unreachableClient(t), "k", time.Second, WithTries(2), WithRetryDelay(time.Millisecond))
	var se *stderr.Error
	if l != nil || !errors.As(err, &se) || se.Code != stderr.CodeUnavailable {
		t.Fatalf("got lock=%v err=%v, want CodeUnavailable (an unreachable Redis is not lock contention)", l, err)
	}
}

func TestAcquire_InvalidTTL(t *testing.T) {
	_, err := Acquire(context.Background(), unreachableClient(t), "k", 0)
	var se *stderr.Error
	if !errors.As(err, &se) || se.Code != stderr.CodeValidationFailed {
		t.Fatalf("got %v, want validation error", err)
	}
}

func TestExtendAndRelease_RedisUnavailable(t *testing.T) {
	locker := New(unreachableClient(t)).(*redisLocker)
	l := &redisLock{mutex: locker.rs.NewMutex("k"), key: "k"}
	ctx := context.Background()

	tests := []struct {
		name string
		call func() error
	}{
		{"extend", func() error { return l.Extend(ctx) }},
		{"release", func() error { return l.Release(ctx) }},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := tt.call(); err == nil {
				t.Error("expected error when redis is unreachable or lock not held")
			}
		})
	}
	if l.Key() != "k" {
		t.Errorf("Key() = %q", l.Key())
	}
}
