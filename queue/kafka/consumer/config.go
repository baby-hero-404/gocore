package consumer

import "time"

// Config defines the configuration for a Kafka consumer group reader.
type Config struct {
	Brokers     []string      `mapstructure:"brokers" json:"brokers"`
	Topic       string        `mapstructure:"topic" json:"topic"`
	GroupID     string        `mapstructure:"group_id" json:"group_id"`
	MinBytes    int           `mapstructure:"min_bytes" json:"min_bytes"`
	MaxBytes    int           `mapstructure:"max_bytes" json:"max_bytes"`
	MaxWait     time.Duration `mapstructure:"max_wait" json:"max_wait"`
	StartOffset int64         `mapstructure:"start_offset" json:"start_offset"` // e.g. -2 for FirstOffset, -1 for LastOffset
}

// DefaultConfig returns safe production defaults for Kafka consumer groups.
func DefaultConfig() Config {
	return Config{
		MinBytes:    10e3,            // 10KB
		MaxBytes:    10e6,            // 10MB
		MaxWait:     1 * time.Second, // 1s
		StartOffset: -2,              // FirstOffset
	}
}

// applyDefaults fills in zero values with defaults.
func (c *Config) applyDefaults() {
	def := DefaultConfig()
	if c.MinBytes <= 0 {
		c.MinBytes = def.MinBytes
	}
	if c.MaxBytes <= 0 {
		c.MaxBytes = def.MaxBytes
	}
	if c.MaxWait <= 0 {
		c.MaxWait = def.MaxWait
	}
	if c.StartOffset == 0 {
		c.StartOffset = def.StartOffset
	}
}
