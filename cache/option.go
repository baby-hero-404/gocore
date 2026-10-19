package cache

import "time"

// Option configures cache initialization.
type Option func(*options)

type options struct {
	dialTimeout  time.Duration
	readTimeout  time.Duration
	writeTimeout time.Duration
	poolSize     int
}

// WithDialTimeout overrides dial timeout.
func WithDialTimeout(d time.Duration) Option {
	return func(o *options) { o.dialTimeout = d }
}

// WithReadTimeout overrides read timeout.
func WithReadTimeout(d time.Duration) Option {
	return func(o *options) { o.readTimeout = d }
}

// WithWriteTimeout overrides write timeout.
func WithWriteTimeout(d time.Duration) Option {
	return func(o *options) { o.writeTimeout = d }
}

// WithPoolSize overrides connection pool size.
func WithPoolSize(size int) Option {
	return func(o *options) { o.poolSize = size }
}
