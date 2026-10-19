package database

import (
	"context"

	"github.com/baby-hero-404/gocore/stderr"
	"gorm.io/gorm"
)

// Ping verifies connectivity on the database instance.
func (i *impl) Ping(ctx context.Context) error {
	return Ping(ctx, i.db)
}

// Ping verifies the database connection is alive on a *gorm.DB handle.
func Ping(ctx context.Context, db *gorm.DB) error {
	if db == nil {
		return stderr.NewBadRequest("database handle is nil")
	}
	sqlDB, err := db.DB()
	if err != nil {
		return stderr.Wrap(stderr.CodeDatabaseError, err, "failed to get underlying database handle")
	}
	if err := sqlDB.PingContext(ctx); err != nil {
		return stderr.Wrap(stderr.CodeUnavailable, err, "database ping failed")
	}
	return nil
}
