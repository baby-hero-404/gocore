package lock

import (
	"context"
	"time"
)

// Locker is the interface for acquiring distributed locks.
type Locker interface {
	Acquire(ctx context.Context, key string, ttl time.Duration, opts ...Option) (Lock, error)
}

// Lock represents an acquired distributed lock handle.
type Lock interface {
	Release(ctx context.Context) error
	Extend(ctx context.Context) error
	Key() string
}
