package lock

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/baby-hero-404/gocore/stderr"
	"github.com/baby-hero-404/gocore/stdlog"
	"github.com/go-redsync/redsync/v4"
	"github.com/redis/go-redis/v9"
)

// Acquire on redisLocker acquires a distributed lock.
func (l *redisLocker) Acquire(ctx context.Context, key string, ttl time.Duration, opts ...Option) (Lock, error) {
	if strings.TrimSpace(key) == "" {
		return nil, stderr.NewValidation("lock key cannot be empty").
			WithDetails(stderr.FieldError{Field: "key", Message: "lock key is required"})
	}
	if ttl <= 0 {
		return nil, stderr.NewValidation("lock ttl must be greater than zero").
			WithDetails(stderr.FieldError{Field: "ttl", Message: "ttl must be positive"})
	}

	o := &options{
		tries:      1, // default: non-blocking single attempt
		retryDelay: 50 * time.Millisecond,
	}
	for _, opt := range opts {
		opt(o)
	}

	redsyncOpts := []redsync.Option{
		redsync.WithExpiry(ttl),
		redsync.WithTries(o.tries),
		redsync.WithRetryDelay(o.retryDelay),
	}

	mutex := l.rs.NewMutex(key, redsyncOpts...)

	if err := mutex.LockContext(ctx); err != nil {
		// redsync returns ErrFailed when lock cannot be acquired
		stdlog.WithContext(ctx).Warn("lock contention: unable to acquire lock",
			"key", key,
			"ttl", ttl.String(),
			"error", err.Error(),
		)
		code, msg := classifyAcquireError(ctx, err)
		return nil, stderr.Wrap(code, err, fmt.Sprintf(msg, key))
	}

	stdlog.WithContext(ctx).Debug("distributed lock acquired", "key", key, "ttl", ttl.String())

	return &redisLock{
		mutex: mutex,
		key:   key,
	}, nil
}

// classifyAcquireError separates real contention (HTTP 409) from infrastructure failures, so an
// unreachable Redis is not reported to callers as "lock already held".
func classifyAcquireError(ctx context.Context, err error) (stderr.Code, string) {
	var taken *redsync.ErrTaken
	var nodeTaken *redsync.ErrNodeTaken
	var redisErr *redsync.RedisError
	switch {
	case ctx.Err() != nil:
		return stderr.CodeTimeout, "lock acquisition cancelled or timed out for key: %s"
	case errors.As(err, &taken), errors.Is(err, redsync.ErrFailed):
		return stderr.CodeLockHeld, "lock already held for key: %s"
	case errors.As(err, &redisErr):
		return stderr.CodeUnavailable, "lock backend unavailable for key: %s"
	case errors.As(err, &nodeTaken):
		return stderr.CodeLockHeld, "lock already held for key: %s"
	default:
		return stderr.CodeUnavailable, "failed to acquire lock for key: %s"
	}
}

// Acquire is a package-level helper that creates a temporary Locker and acquires the lock.
func Acquire(ctx context.Context, client *redis.Client, key string, ttl time.Duration, opts ...Option) (Lock, error) {
	if client == nil {
		return nil, stderr.NewBadRequest("redis client cannot be nil")
	}
	locker := New(client)
	return locker.Acquire(ctx, key, ttl, opts...)
}
