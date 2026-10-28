package database

import (
	"context"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

// NewSQLite connects to SQLite with connection pooling, health checks, and bounded retries.
func NewSQLite(ctx context.Context, cfg Config, opts ...Option) (*gorm.DB, error) {
	return connect(ctx, "sqlite", sqlite.Open(cfg.DSN), cfg, opts...)
}
