package database

import (
	"github.com/baby-hero-404/gocore/stderr"
	"gorm.io/gorm"
)

// Close closes the connection pool of this database instance.
func (i *impl) Close() error {
	return Close(i.db)
}

// Close gracefully closes the underlying connection pool.
func Close(db *gorm.DB) error {
	if db == nil {
		return nil
	}
	sqlDB, err := db.DB()
	if err != nil {
		return stderr.Wrap(stderr.CodeDatabaseError, err, "failed to get underlying database handle")
	}
	if err := sqlDB.Close(); err != nil {
		return stderr.Wrap(stderr.CodeDatabaseError, err, "failed to close database pool")
	}
	return nil
}
