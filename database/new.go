package database

import (
	"context"
	"fmt"
	"io"
	"log"
	"os"
	"strings"
	"time"

	"github.com/baby-hero-404/gocore/stderr"
	"github.com/baby-hero-404/gocore/stdlog"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	gormlogger "gorm.io/gorm/logger"
)

type impl struct {
	db *gorm.DB
}

// New creates a production-ready Database instance implementing the Database interface.
func New(ctx context.Context, cfg Config, opts ...Option) (Database, error) {
	db, err := NewPostgres(ctx, cfg, opts...)
	if err != nil {
		return nil, err
	}
	return &impl{db: db}, nil
}

// NewPostgres connects to PostgreSQL with connection pooling, health checks, and bounded retries.
func NewPostgres(ctx context.Context, cfg Config, opts ...Option) (*gorm.DB, error) {
	if strings.TrimSpace(cfg.DSN) == "" {
		return nil, stderr.NewValidation("database configuration invalid").
			WithDetails(stderr.FieldError{
				Field:   "dsn",
				Message: "dsn cannot be empty",
				Tag:     "required",
			})
	}

	cfg.applyDefaults()

	o := &options{}
	for _, opt := range opts {
		opt(o)
	}

	gormCfg := o.gormConfig
	if gormCfg == nil {
		l := o.logger
		if l == nil {
			l = resolveLogger(cfg.LogLevel)
		}
		gormCfg = &gorm.Config{
			Logger: l,
		}
	}

	var db *gorm.DB
	var lastErr error

	// Retry connection with exponential backoff
	backoff := cfg.RetryBackoff
	for attempt := 1; attempt <= cfg.RetryAttempts; attempt++ {
		select {
		case <-ctx.Done():
			return nil, stderr.Wrap(stderr.CodeTimeout, ctx.Err(), "database connection cancelled or timed out")
		default:
		}

		db, lastErr = gorm.Open(postgres.Open(cfg.DSN), gormCfg)
		if lastErr == nil {
			sqlDB, err := db.DB()
			if err == nil {
				if pingErr := sqlDB.PingContext(ctx); pingErr == nil {
					// Configure connection pool
					sqlDB.SetMaxOpenConns(cfg.MaxOpenConns)
					sqlDB.SetMaxIdleConns(cfg.MaxIdleConns)
					sqlDB.SetConnMaxLifetime(cfg.ConnMaxLifetime)
					sqlDB.SetConnMaxIdleTime(cfg.ConnMaxIdleTime)

					stdlog.WithContext(ctx).Info("postgres connection established",
						"max_open_conns", cfg.MaxOpenConns,
						"max_idle_conns", cfg.MaxIdleConns,
					)
					return db, nil
				} else {
					lastErr = pingErr
				}
			} else {
				lastErr = err
			}
		}

		stdlog.WithContext(ctx).Warn("postgres connection failed, retrying",
			"attempt", attempt,
			"max_attempts", cfg.RetryAttempts,
			"backoff", backoff.String(),
			"error", lastErr.Error(),
		)

		if attempt < cfg.RetryAttempts {
			time.Sleep(backoff)
			backoff *= 2
		}
	}

	return nil, stderr.Wrap(stderr.CodeUnavailable, lastErr,
		fmt.Sprintf("failed to connect to postgres after %d attempts", cfg.RetryAttempts))
}

// DB returns the underlying *gorm.DB instance.
func (i *impl) DB() *gorm.DB {
	return i.db
}

// WithContext returns a session with the provided context.
func (i *impl) WithContext(ctx context.Context) *gorm.DB {
	return i.db.WithContext(ctx)
}

// NewDb creates a new isolated session with ctx, skipping default transaction and setting batch size.
func (i *impl) NewDb(ctx context.Context) *gorm.DB {
	return i.db.WithContext(ctx).Session(&gorm.Session{
		NewDB:                  true,
		QueryFields:            true,
		SkipDefaultTransaction: true,
		CreateBatchSize:        1000,
	})
}

func resolveLogger(level string) gormlogger.Interface { return resolveLoggerTo(os.Stdout, level) }

func resolveLoggerTo(w io.Writer, level string) gormlogger.Interface {
	var gormLevel gormlogger.LogLevel
	switch strings.ToLower(level) {
	case "silent":
		gormLevel = gormlogger.Silent
	case "info":
		gormLevel = gormlogger.Info
	case "warn":
		gormLevel = gormlogger.Warn
	default:
		gormLevel = gormlogger.Error
	}

	// ErrRecordNotFound is normal control flow (First / lookups), not a warning; colour codes
	// would pollute log files.
	return gormlogger.New(log.New(w, "", log.LstdFlags), gormlogger.Config{
		SlowThreshold:             200 * time.Millisecond,
		LogLevel:                  gormLevel,
		IgnoreRecordNotFoundError: true,
		Colorful:                  false,
	})
}
