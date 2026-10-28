package database

import (
	"context"

	"gorm.io/driver/mysql"
	"gorm.io/gorm"
)

// NewMySQL connects to MySQL/TiDB with connection pooling, health checks, and bounded retries.
func NewMySQL(ctx context.Context, cfg Config, opts ...Option) (*gorm.DB, error) {
	return connect(ctx, "mysql", mysql.Open(cfg.DSN), cfg, opts...)
}
