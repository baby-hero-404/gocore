package cache

import (
	"context"
	"strings"

	"github.com/baby-hero-404/gocore/stderr"
	"github.com/baby-hero-404/gocore/stdlog"
	"github.com/redis/go-redis/v9"
)

type impl struct {
	client *redis.Client
}

// New creates a new Cache instance adhering to the Cache interface.
func New(ctx context.Context, cfg Config, opts ...Option) (Cache, error) {
	client, err := NewRedis(ctx, cfg, opts...)
	if err != nil {
		return nil, err
	}
	return &impl{client: client}, nil
}

// NewRedis initializes a connected Redis client with timeout and pool settings.
func NewRedis(ctx context.Context, cfg Config, opts ...Option) (*redis.Client, error) {
	if strings.TrimSpace(cfg.Addr) == "" {
		return nil, stderr.NewValidation("cache configuration invalid").
			WithDetails(stderr.FieldError{
				Field:   "addr",
				Message: "redis address cannot be empty",
				Tag:     "required",
			})
	}

	cfg.applyDefaults()

	o := &options{
		dialTimeout:  cfg.DialTimeout,
		readTimeout:  cfg.ReadTimeout,
		writeTimeout: cfg.WriteTimeout,
		poolSize:     cfg.PoolSize,
	}
	for _, opt := range opts {
		opt(o)
	}

	redisOpts := &redis.Options{
		Addr:         cfg.Addr,
		Password:     cfg.Password,
		DB:           cfg.DB,
		DialTimeout:  o.dialTimeout,
		ReadTimeout:  o.readTimeout,
		WriteTimeout: o.writeTimeout,
		PoolSize:     o.poolSize,
		MinIdleConns: cfg.MinIdleConns,
		MaxRetries:   cfg.MaxRetries,
	}

	client := redis.NewClient(redisOpts)

	// Verify connectivity via Ping
	if err := client.Ping(ctx).Err(); err != nil {
		_ = client.Close()
		return nil, stderr.Wrap(stderr.CodeUnavailable, err, "failed to connect to redis")
	}

	stdlog.WithContext(ctx).Info("redis connection established",
		"addr", cfg.Addr,
		"db", cfg.DB,
		"pool_size", o.poolSize,
	)

	return client, nil
}

// Client returns the underlying *redis.Client.
func (i *impl) Client() *redis.Client {
	return i.client
}
