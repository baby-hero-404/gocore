package database

import (
	"gorm.io/gorm"
	gormlogger "gorm.io/gorm/logger"
)

// Option configures Database initialization.
type Option func(*options)

type options struct {
	gormConfig *gorm.Config
	logger     gormlogger.Interface
}

// WithGormConfig allows passing a custom gorm.Config.
func WithGormConfig(cfg *gorm.Config) Option {
	return func(o *options) {
		o.gormConfig = cfg
	}
}

// WithLogger allows specifying a custom GORM logger.
func WithLogger(l gormlogger.Interface) Option {
	return func(o *options) {
		o.logger = l
	}
}
