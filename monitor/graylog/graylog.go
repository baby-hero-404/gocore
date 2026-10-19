// Package graylog provides a GELF UDP writer for Graylog log sinks.
package graylog

import (
	"io"
	"strings"

	"github.com/Graylog2/go-gelf/gelf"
	"github.com/baby-hero-404/gocore/stderr"
	"github.com/baby-hero-404/gocore/stdlog"
)

// Config defines GELF UDP endpoint configuration.
type Config struct {
	Addr string `mapstructure:"addr" json:"addr" required:"true"` // e.g. "localhost:12201"
}

// DefaultConfig returns standard default configuration for Graylog.
func DefaultConfig() Config {
	return Config{
		Addr: "127.0.0.1:12201",
	}
}

// NewWriter initializes a UDP GELF writer for Graylog log sinks.
func NewWriter(cfg Config) (io.Writer, error) {
	if strings.TrimSpace(cfg.Addr) == "" {
		return nil, stderr.NewValidation("graylog configuration invalid").
			WithDetails(stderr.FieldError{Field: "addr", Message: "graylog address cannot be empty"})
	}

	writer, err := gelf.NewWriter(cfg.Addr)
	if err != nil {
		return nil, stderr.Wrap(stderr.CodeUnavailable, err, "failed to initialize graylog gelf writer")
	}

	stdlog.Info("graylog gelf writer initialized", "addr", cfg.Addr)
	return writer, nil
}
