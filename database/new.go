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
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	gormlogger "gorm.io/gorm/logger"
)

type impl struct {
	db *gorm.DB
}

// New creates a production-ready Database instance implementing the Database interface.
func New(ctx context.Context, cfg Config, opts ...Option) (Database, error) {
	cfg.applyDefaults()
	var db *gorm.DB
	var err error

	switch strings.ToLower(cfg.Driver) {
	case "mysql":
		db, err = NewMySQL(ctx, cfg, opts...)
	case "sqlite":
		db, err = NewSQLite(ctx, cfg, opts...)
	case "postgres", "":
		db, err = NewPostgres(ctx, cfg, opts...)
	default:
		return nil, stderr.NewValidation("database configuration invalid").
			WithDetails(stderr.FieldError{
				Field:   "driver",
				Message: fmt.Sprintf("unsupported database driver: %s", cfg.Driver),
				Tag:     "oneof",
			})
	}
	if err != nil {
		return nil, err
	}
	return &impl{db: db}, nil
}

// NewPostgres connects to PostgreSQL with connection pooling, health checks, and bounded retries.
func NewPostgres(ctx context.Context, cfg Config, opts ...Option) (*gorm.DB, error) {
	return connect(ctx, "postgres", postgres.Open(cfg.DSN), cfg, opts...)
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
