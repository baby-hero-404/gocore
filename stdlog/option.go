package stdlog

import (
	"io"
	"log/slog"
)

// Option configures a Logger.
type Option func(*options)

type options struct {
	level     slog.Level
	isJSON    bool
	addSource bool
	output    io.Writer
	handler   slog.Handler
}

// WithLevel sets the minimum logging level.
func WithLevel(level slog.Level) Option {
	return func(o *options) {
		o.level = level
	}
}

// WithJSON sets whether output should be JSON formatted (defaults to true).
func WithJSON(json bool) Option {
	return func(o *options) {
		o.isJSON = json
	}
}

// WithAddSource controls whether file caller info is included in log records.
func WithAddSource(addSource bool) Option {
	return func(o *options) {
		o.addSource = addSource
	}
}

// WithOutput sets the output destination (default: os.Stdout).
func WithOutput(w io.Writer) Option {
	return func(o *options) {
		o.output = w
	}
}

// WithHandler overrides the underlying slog.Handler completely.
func WithHandler(h slog.Handler) Option {
	return func(o *options) {
		o.handler = h
	}
}
