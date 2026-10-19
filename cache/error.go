package cache

import (
	"errors"
	"net"

	"github.com/baby-hero-404/gocore/stderr"
	"github.com/redis/go-redis/v9"
)

// WrapError converts known Redis client errors into standardized stderr.Error types.
func WrapError(err error) error {
	if err == nil {
		return nil
	}

	if errors.Is(err, redis.Nil) {
		return stderr.Wrap(stderr.CodeNotFound, err, "cache key not found")
	}

	if errors.Is(err, redis.TxFailedErr) {
		return stderr.Wrap(stderr.CodeConflict, err, "transaction conflict")
	}

	var netErr net.Error
	if errors.As(err, &netErr) && netErr.Timeout() {
		return stderr.Wrap(stderr.CodeTimeout, err, "cache operation timed out")
	}

	return stderr.Wrap(stderr.CodeCacheError, err, "cache operation failed")
}
