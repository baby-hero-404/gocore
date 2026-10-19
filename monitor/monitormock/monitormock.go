// Package monitormock provides test doubles for github.com/baby-hero-404/gocore/monitor.
package monitormock

import (
	"context"
	"time"

	"github.com/stretchr/testify/mock"
)

// MetricsRecorder provides an in-memory mock of MetricsRecorder for unit testing.
type MetricsRecorder struct {
	mock.Mock
}

// NewMetricsRecorder creates a new MetricsRecorder instance.
func NewMetricsRecorder(t mock.TestingT) *MetricsRecorder {
	m := &MetricsRecorder{}
	m.Test(t)
	return m
}

func (m *MetricsRecorder) RecordDuration(method, path string, statusCode int, duration time.Duration) {
	m.Called(method, path, statusCode, duration)
}

func (m *MetricsRecorder) IncrementInFlight() {
	m.Called()
}

func (m *MetricsRecorder) DecrementInFlight() {
	m.Called()
}

// ErrorTracker provides an in-memory mock of ErrorTracker for unit testing.
type ErrorTracker struct {
	mock.Mock
}

// NewErrorTracker creates a new ErrorTracker instance.
func NewErrorTracker(t mock.TestingT) *ErrorTracker {
	m := &ErrorTracker{}
	m.Test(t)
	return m
}

func (m *ErrorTracker) CaptureException(err error, tags map[string]string) {
	m.Called(err, tags)
}

func (m *ErrorTracker) Flush(ctx context.Context, timeout time.Duration) bool {
	args := m.Called(ctx, timeout)
	return args.Bool(0)
}
