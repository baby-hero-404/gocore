package database

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/baby-hero-404/gocore/stderr"
	"github.com/baby-hero-404/gocore/stdlog"
	"gorm.io/gorm"
)

// maxRetryBackoff bounds the exponential retry delay so large RetryAttempts cannot sleep for minutes.
const maxRetryBackoff = 30 * time.Second

// Connect opens dialector with connection pooling, a startup ping and bounded retries, and returns
// the *gorm.DB. Driver packages (database/postgres, database/mysql, database/sqlite) build the
// dialector; the core never imports a driver, so services only carry the driver they use.
//
// Connect owns the pool it opens: after a failed ping it closes that pool before retrying. A
// dialector that wraps an existing *sql.DB (its Conn field) therefore has that handle closed on
// failure; pass a DSN-based dialector, as the driver packages do, when retries must work.
func Connect(ctx context.Context, dialector gorm.Dialector, cfg Config, opts ...Option) (*gorm.DB, error) {
	if dialector == nil {
		return nil, stderr.NewValidation("database configuration invalid").
			WithDetails(stderr.FieldError{Field: "dialector", Message: "dialector cannot be nil", Tag: "required"})
	}
	driverName := dialector.Name()
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

		db, lastErr = gorm.Open(dialector, gormCfg)
		if lastErr == nil {
			sqlDB, err := db.DB()
			if err == nil {
				if pingErr := sqlDB.PingContext(ctx); pingErr == nil {
					// Configure connection pool
					sqlDB.SetMaxOpenConns(cfg.MaxOpenConns)
					sqlDB.SetMaxIdleConns(cfg.MaxIdleConns)
					sqlDB.SetConnMaxLifetime(cfg.ConnMaxLifetime)
					sqlDB.SetConnMaxIdleTime(cfg.ConnMaxIdleTime)

					stdlog.WithContext(ctx).Info(driverName+" connection established",
						"max_open_conns", cfg.MaxOpenConns,
						"max_idle_conns", cfg.MaxIdleConns,
					)
					return db, nil
				} else {
					lastErr = pingErr
					_ = sqlDB.Close() // release the pool of the failed attempt before retrying
				}
			} else {
				lastErr = err
			}
		}

		stdlog.WithContext(ctx).Warn(driverName+" connection failed, retrying",
			"attempt", attempt,
			"max_attempts", cfg.RetryAttempts,
			"backoff", backoff.String(),
			"error", lastErr.Error(),
		)

		if attempt < cfg.RetryAttempts {
			timer := time.NewTimer(backoff)
			select {
			case <-ctx.Done():
				timer.Stop()
				return nil, stderr.Wrap(stderr.CodeTimeout, ctx.Err(), "database connection cancelled while retrying")
			case <-timer.C:
			}
			backoff = min(backoff*2, maxRetryBackoff)
		}
	}

	return nil, stderr.Wrap(stderr.CodeUnavailable, lastErr,
		fmt.Sprintf("failed to connect to %s after %d attempts", driverName, cfg.RetryAttempts))
}
