package producer

import "time"

// Config defines the configuration for a Kafka producer.
type Config struct {
	Brokers      []string      `mapstructure:"brokers" json:"brokers"`
	Topic        string        `mapstructure:"topic" json:"topic"`
	BatchSize    int           `mapstructure:"batch_size" json:"batch_size"`
	BatchTimeout time.Duration `mapstructure:"batch_timeout" json:"batch_timeout"`
	RequiredAcks int           `mapstructure:"required_acks" json:"required_acks"`
	Async        bool          `mapstructure:"async" json:"async"`
}

// DefaultConfig returns safe production defaults for Kafka producer.
func DefaultConfig() Config {
	return Config{
		BatchSize:    100,
		BatchTimeout: 10 * time.Millisecond,
		RequiredAcks: -1, // Wait for all in-sync replicas (strong durability)
		Async:        false,
	}
}

// applyDefaults fills in zero values with defaults.
func (c *Config) applyDefaults() {
	def := DefaultConfig()
	if c.BatchSize <= 0 {
		c.BatchSize = def.BatchSize
	}
	if c.BatchTimeout <= 0 {
		c.BatchTimeout = def.BatchTimeout
	}
	if c.RequiredAcks == 0 {
		c.RequiredAcks = def.RequiredAcks
	}
}
