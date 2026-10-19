package lock

import (
	"context"
	"fmt"

	"github.com/baby-hero-404/gocore/stderr"
)

// Extend resets the TTL of the currently held lock.
func (l *redisLock) Extend(ctx context.Context) error {
	if l == nil || l.mutex == nil {
		return stderr.NewBadRequest("lock handle is nil")
	}

	ok, err := l.mutex.ExtendContext(ctx)
	if err != nil {
		return stderr.Wrap(stderr.CodeCacheError, err, fmt.Sprintf("failed to extend lock for key: %s", l.key))
	}
	if !ok {
		return stderr.New(stderr.CodeConflict, fmt.Sprintf("failed to extend: lock already expired for key: %s", l.key))
	}

	return nil
}

// Key returns the lock's resource key.
func (l *redisLock) Key() string {
	if l == nil {
		return ""
	}
	return l.key
}
