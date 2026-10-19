package database

import (
	"context"

	"gorm.io/gorm"
)

// Database defines the standard contract for relational database access in gocore.
type Database interface {
	// DB returns the underlying *gorm.DB instance.
	DB() *gorm.DB

	// NewDb creates a new session scoped with ctx.
	NewDb(ctx context.Context) *gorm.DB

	// WithContext returns a scoped *gorm.DB session with the given context.
	WithContext(ctx context.Context) *gorm.DB

	// Ping verifies connectivity to the database.
	Ping(ctx context.Context) error

	// Close gracefully terminates the connection pool.
	Close() error
}
