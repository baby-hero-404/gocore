package database

import "time"

// Config defines connection and pool parameters for PostgreSQL.
type Config struct {
	DSN             string        `mapstructure:"dsn" json:"dsn" required:"true"`
	MaxOpenConns    int           `mapstructure:"max_open_conns" json:"max_open_conns"`
	MaxIdleConns    int           `mapstructure:"max_idle_conns" json:"max_idle_conns"`
	ConnMaxLifetime time.Duration `mapstructure:"conn_max_lifetime" json:"conn_max_lifetime"`
	ConnMaxIdleTime time.Duration `mapstructure:"conn_max_idle_time" json:"conn_max_idle_time"`
	RetryAttempts   int           `mapstructure:"retry_attempts" json:"retry_attempts"`
	RetryBackoff    time.Duration `mapstructure:"retry_backoff" json:"retry_backoff"`
	LogLevel        string        `mapstructure:"log_level" json:"log_level"` // "silent", "error", "warn", "info"
}

// DefaultConfig returns conservative production-ready defaults.
func DefaultConfig() Config {
	return Config{
		MaxOpenConns:    25,
		MaxIdleConns:    10,
		ConnMaxLifetime: 5 * time.Minute,
		ConnMaxIdleTime: 2 * time.Minute,
		RetryAttempts:   3,
		RetryBackoff:    200 * time.Millisecond,
		LogLevel:        "warn",
	}
}

// applyDefaults fills in zero values with safe defaults.
func (c *Config) applyDefaults() {
	def := DefaultConfig()
	if c.MaxOpenConns <= 0 {
		c.MaxOpenConns = def.MaxOpenConns
	}
	if c.MaxIdleConns <= 0 {
		c.MaxIdleConns = def.MaxIdleConns
	}
	if c.ConnMaxLifetime <= 0 {
		c.ConnMaxLifetime = def.ConnMaxLifetime
	}
	if c.ConnMaxIdleTime <= 0 {
		c.ConnMaxIdleTime = def.ConnMaxIdleTime
	}
	if c.RetryAttempts <= 0 {
		c.RetryAttempts = def.RetryAttempts
	}
	if c.RetryBackoff <= 0 {
		c.RetryBackoff = def.RetryBackoff
	}
	if c.LogLevel == "" {
		c.LogLevel = def.LogLevel
	}
}
