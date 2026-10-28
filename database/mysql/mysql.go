package mysql

import (
	"context"

	"github.com/baby-hero-404/gocore/database"
	gormmysql "gorm.io/driver/mysql"
	"gorm.io/gorm"
)

// Open connects to MySQL/TiDB with connection pooling, a startup ping and bounded retries.
func Open(ctx context.Context, cfg database.Config, opts ...database.Option) (*gorm.DB, error) {
	return database.Connect(ctx, gormmysql.Open(cfg.DSN), cfg, opts...)
}

// New is Open wrapped in the database.Database interface.
func New(ctx context.Context, cfg database.Config, opts ...database.Option) (database.Database, error) {
	return database.Open(ctx, gormmysql.Open(cfg.DSN), cfg, opts...)
}
