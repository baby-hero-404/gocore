package monitor

import (
	"context"
	"time"
)

// MetricsRecorder defines the contract for emitting application metrics.
type MetricsRecorder interface {
	RecordDuration(method, path string, statusCode int, duration time.Duration)
	IncrementInFlight()
	DecrementInFlight()
}

// ErrorTracker defines the contract for reporting unhandled errors to remote sinks (Sentry, etc.).
type ErrorTracker interface {
	CaptureException(err error, tags map[string]string)
	Flush(ctx context.Context, timeout time.Duration) bool
}
