package anydriver

import (
	"context"
	"fmt"
	"strings"

	"github.com/baby-hero-404/gocore/database"
	"github.com/baby-hero-404/gocore/database/mysql"
	"github.com/baby-hero-404/gocore/database/postgres"
	"github.com/baby-hero-404/gocore/database/sqlite"
	"github.com/baby-hero-404/gocore/stderr"
)

// New connects with the driver named by cfg.Driver: "postgres" (or empty), "mysql" or "sqlite".
func New(ctx context.Context, cfg database.Config, opts ...database.Option) (database.Database, error) {
	switch strings.ToLower(cfg.Driver) {
	case "postgres", "":
		return postgres.New(ctx, cfg, opts...)
	case "mysql":
		return mysql.New(ctx, cfg, opts...)
	case "sqlite":
		return sqlite.New(ctx, cfg, opts...)
	default:
		return nil, stderr.NewValidation("database configuration invalid").
			WithDetails(stderr.FieldError{
				Field:   "driver",
				Message: fmt.Sprintf("unsupported database driver: %s", cfg.Driver),
				Tag:     "oneof",
			})
	}
}
