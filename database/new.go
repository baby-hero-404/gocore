package database

import (
	"context"
	"io"
	"log"
	"os"
	"strings"
	"time"

	"gorm.io/gorm"
	gormlogger "gorm.io/gorm/logger"
)

type impl struct {
	db *gorm.DB
}

// Open is Connect wrapped in the Database interface.
func Open(ctx context.Context, dialector gorm.Dialector, cfg Config, opts ...Option) (Database, error) {
	db, err := Connect(ctx, dialector, cfg, opts...)
	if err != nil {
		return nil, err
	}
	return &impl{db: db}, nil
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
