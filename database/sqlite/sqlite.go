package sqlite

import (
	"context"

	"github.com/baby-hero-404/gocore/database"
	gormsqlite "gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

// Open connects to SQLite with connection pooling, a startup ping and bounded retries.
func Open(ctx context.Context, cfg database.Config, opts ...database.Option) (*gorm.DB, error) {
	return database.Connect(ctx, gormsqlite.Open(cfg.DSN), cfg, opts...)
}

// New is Open wrapped in the database.Database interface.
func New(ctx context.Context, cfg database.Config, opts ...database.Option) (database.Database, error) {
	return database.Open(ctx, gormsqlite.Open(cfg.DSN), cfg, opts...)
}
