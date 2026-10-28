package cache

import "time"

// Config defines connection parameters for Redis.
type Config struct {
	Addr         string        `mapstructure:"addr" json:"addr" required:"true"`
	Password     string        `mapstructure:"password" json:"-"`
	DB           int           `mapstructure:"db" json:"db"`
	DialTimeout  time.Duration `mapstructure:"dial_timeout" json:"dial_timeout"`
	ReadTimeout  time.Duration `mapstructure:"read_timeout" json:"read_timeout"`
	WriteTimeout time.Duration `mapstructure:"write_timeout" json:"write_timeout"`
	PoolSize     int           `mapstructure:"pool_size" json:"pool_size"`
	MinIdleConns int           `mapstructure:"min_idle_conns" json:"min_idle_conns"`
	MaxRetries   int           `mapstructure:"max_retries" json:"max_retries"`
}

// DefaultConfig returns production-grade default settings for Redis.
func DefaultConfig() Config {
	return Config{
		Addr:         "127.0.0.1:6379",
		DB:           0,
		DialTimeout:  5 * time.Second,
		ReadTimeout:  3 * time.Second,
		WriteTimeout: 3 * time.Second,
		PoolSize:     20,
		MinIdleConns: 5,
		MaxRetries:   3,
	}
}

func (c *Config) applyDefaults() {
	def := DefaultConfig()
	if c.DialTimeout <= 0 {
		c.DialTimeout = def.DialTimeout
	}
	if c.ReadTimeout <= 0 {
		c.ReadTimeout = def.ReadTimeout
	}
	if c.WriteTimeout <= 0 {
		c.WriteTimeout = def.WriteTimeout
	}
	if c.PoolSize <= 0 {
		c.PoolSize = def.PoolSize
	}
	if c.MinIdleConns < 0 {
		c.MinIdleConns = def.MinIdleConns
	}
	if c.MaxRetries <= 0 {
		c.MaxRetries = def.MaxRetries
	}
}
