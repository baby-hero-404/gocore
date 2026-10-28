package httpserver

import "time"

// Config defines HTTP server parameters.
type Config struct {
	Port             int           `mapstructure:"port" json:"port"`
	ReadTimeout      time.Duration `mapstructure:"read_timeout" json:"read_timeout"`
	WriteTimeout     time.Duration `mapstructure:"write_timeout" json:"write_timeout"`
	IdleTimeout      time.Duration `mapstructure:"idle_timeout" json:"idle_timeout"`
	ShutdownTimeout  time.Duration `mapstructure:"shutdown_timeout" json:"shutdown_timeout"` // grace period for in-flight requests once Start's ctx is cancelled
	AllowOrigins     []string      `mapstructure:"allow_origins" json:"allow_origins"`
	AllowMethods     []string      `mapstructure:"allow_methods" json:"allow_methods"`
	AllowHeaders     []string      `mapstructure:"allow_headers" json:"allow_headers"`
	AllowCredentials bool          `mapstructure:"allow_credentials" json:"allow_credentials"`
}

// DefaultConfig returns safe production defaults for HTTP server.
func DefaultConfig() Config {
	return Config{
		Port:             8080,
		ReadTimeout:      10 * time.Second,
		WriteTimeout:     15 * time.Second,
		IdleTimeout:      60 * time.Second,
		ShutdownTimeout:  10 * time.Second,
		AllowOrigins:     []string{"*"},
		AllowMethods:     []string{"GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS"},
		AllowHeaders:     []string{"Origin", "Content-Type", "Accept", "Authorization", "X-Request-ID", "X-Guest-ID"},
		AllowCredentials: false,
	}
}

func (c *Config) applyDefaults() {
	def := DefaultConfig()
	if c.Port <= 0 {
		c.Port = def.Port
	}
	if c.ReadTimeout <= 0 {
		c.ReadTimeout = def.ReadTimeout
	}
	if c.WriteTimeout <= 0 {
		c.WriteTimeout = def.WriteTimeout
	}
	if c.IdleTimeout <= 0 {
		c.IdleTimeout = def.IdleTimeout
	}
	if c.ShutdownTimeout <= 0 {
		c.ShutdownTimeout = def.ShutdownTimeout
	}
	if len(c.AllowOrigins) == 0 {
		c.AllowOrigins = def.AllowOrigins
	}
	if len(c.AllowMethods) == 0 {
		c.AllowMethods = def.AllowMethods
	}
	if len(c.AllowHeaders) == 0 {
		c.AllowHeaders = def.AllowHeaders
	}
}
