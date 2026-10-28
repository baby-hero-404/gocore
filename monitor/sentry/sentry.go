// Package sentry initializes Sentry error reporting and implements monitor.ErrorTracker helpers.
package sentry

import (
	"context"
	"strings"
	"time"

	"github.com/baby-hero-404/gocore/stderr"
	"github.com/baby-hero-404/gocore/stdlog"
	sentrygo "github.com/getsentry/sentry-go"
)

// Config configures Sentry error tracking.
type Config struct {
	DSN              string  `mapstructure:"dsn" json:"-" required:"true"`
	Environment      string  `mapstructure:"environment" json:"environment"`
	Release          string  `mapstructure:"release" json:"release"`
	TracesSampleRate float64 `mapstructure:"traces_sample_rate" json:"traces_sample_rate"`
}

// DefaultConfig returns default settings for Sentry.
func DefaultConfig() Config {
	return Config{
		Environment:      "development",
		TracesSampleRate: 1.0,
	}
}

// Init initializes the Sentry SDK.
func Init(cfg Config) error {
	if strings.TrimSpace(cfg.DSN) == "" {
		return stderr.NewValidation("sentry configuration invalid").
			WithDetails(stderr.FieldError{Field: "dsn", Message: "sentry DSN cannot be empty"})
	}

	err := sentrygo.Init(sentrygo.ClientOptions{
		Dsn:              cfg.DSN,
		Environment:      cfg.Environment,
		Release:          cfg.Release,
		TracesSampleRate: cfg.TracesSampleRate,
	})
	if err != nil {
		return stderr.Wrap(stderr.CodeInternal, err, "failed to initialize sentry")
	}

	stdlog.Info("sentry error reporting initialized", "environment", cfg.Environment)
	return nil
}

// CaptureError sends an error event to Sentry with contextual metadata.
func CaptureError(ctx context.Context, err error) {
	if err == nil {
		return
	}

	hub := sentrygo.CurrentHub().Clone()
	if ctx != nil {
		if reqID, ok := ctx.Value(stdlog.RequestIDCtxKey).(string); ok && reqID != "" {
			hub.Scope().SetTag("request_id", reqID)
		}
	}

	hub.CaptureException(err)
}

// Flush flushes queued events to Sentry before server shutdown.
func Flush(timeout time.Duration) bool {
	return sentrygo.Flush(timeout)
}
