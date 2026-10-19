package lock

import "time"

// Option configures distributed lock acquisition.
type Option func(*options)

type options struct {
	tries      int
	retryDelay time.Duration
}

// WithTries sets the maximum number of attempts to acquire the lock.
func WithTries(tries int) Option {
	return func(o *options) {
		o.tries = tries
	}
}

// WithRetryDelay sets the duration between lock acquisition retries.
func WithRetryDelay(delay time.Duration) Option {
	return func(o *options) {
		o.retryDelay = delay
	}
}
