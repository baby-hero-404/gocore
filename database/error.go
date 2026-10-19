package database

import (
	"errors"

	"github.com/baby-hero-404/gocore/stderr"
	"gorm.io/gorm"
)

// WrapError converts known GORM and database driver errors into standardized stderr.Error types.
func WrapError(err error) error {
	if err == nil {
		return nil
	}

	if errors.Is(err, gorm.ErrRecordNotFound) {
		return stderr.Wrap(stderr.CodeNotFound, err, "record not found")
	}

	if errors.Is(err, gorm.ErrDuplicatedKey) {
		return stderr.Wrap(stderr.CodeConflict, err, "record already exists")
	}

	if errors.Is(err, gorm.ErrInvalidTransaction) {
		return stderr.Wrap(stderr.CodeInternal, err, "invalid database transaction")
	}

	return stderr.Wrap(stderr.CodeDatabaseError, err, "database operation failed")
}
