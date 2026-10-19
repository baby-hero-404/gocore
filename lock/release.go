package lock

import (
	"context"
	"fmt"

	"github.com/baby-hero-404/gocore/stderr"
	"github.com/baby-hero-404/gocore/stdlog"
)

// Release unlocks the distributed lock.
func (l *redisLock) Release(ctx context.Context) error {
	if l == nil || l.mutex == nil {
		return nil
	}

	ok, err := l.mutex.UnlockContext(ctx)
	if err != nil {
		stdlog.WithContext(ctx).Error("failed to release lock", "key", l.key, "error", err.Error())
		return stderr.Wrap(stderr.CodeCacheError, err, fmt.Sprintf("failed to release lock for key: %s", l.key))
	}
	if !ok {
		return stderr.New(stderr.CodeConflict, fmt.Sprintf("lock already released or expired for key: %s", l.key))
	}

	stdlog.WithContext(ctx).Debug("distributed lock released", "key", l.key)
	return nil
}
