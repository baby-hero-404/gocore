package lock

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/baby-hero-404/gocore/stderr"
	"github.com/redis/go-redis/v9"
)

func TestAcquire_Validation(t *testing.T) {
	ctx := context.Background()

	// Case 1: Nil client
	l, err := Acquire(ctx, nil, "my-key", 5*time.Second)
	if err == nil || l != nil {
		t.Error("expected error for nil client")
	}

	// Case 2: Empty key
	client := redis.NewClient(&redis.Options{Addr: "localhost:6379"})
	defer client.Close()

	l, err = Acquire(ctx, client, "", 5*time.Second)
	if err == nil || l != nil {
		t.Error("expected error for empty key")
	}
	var stdErr *stderr.Error
	if !errors.As(err, &stdErr) || stdErr.Code != stderr.CodeValidationFailed {
		t.Errorf("expected CodeValidationFailed, got %v", err)
	}

	// Case 3: Invalid TTL
	l, err = Acquire(ctx, client, "my-key", 0)
	if err == nil || l != nil {
		t.Error("expected error for zero TTL")
	}
	if !errors.As(err, &stdErr) || stdErr.Code != stderr.CodeValidationFailed {
		t.Errorf("expected CodeValidationFailed, got %v", err)
	}
}

func TestLock_NilSafe(t *testing.T) {
	ctx := context.Background()
	var l *redisLock
	if err := l.Release(ctx); err != nil {
		t.Errorf("expected nil error on nil Lock.Release, got %v", err)
	}
	if err := l.Extend(ctx); err == nil {
		t.Error("expected error extending nil lock")
	}
	if l.Key() != "" {
		t.Errorf("expected empty key for nil lock, got %s", l.Key())
	}
}

func TestOptions(t *testing.T) {
	o := &options{}
	WithTries(5)(o)
	WithRetryDelay(100 * time.Millisecond)(o)

	if o.tries != 5 {
		t.Errorf("expected tries 5, got %d", o.tries)
	}
	if o.retryDelay != 100*time.Millisecond {
		t.Errorf("expected retryDelay 100ms, got %v", o.retryDelay)
	}
}

func TestNew_NilClient(t *testing.T) {
	locker := New(nil)
	if locker != nil {
		t.Error("expected nil locker for nil redis client")
	}
}
