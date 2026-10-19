// Package stdlogmock provides test doubles for github.com/baby-hero-404/gocore/stdlog.
package stdlogmock

import (
	"context"
	"github.com/baby-hero-404/gocore/stdlog"
	"log/slog"

	"github.com/stretchr/testify/mock"
)

// Logger provides a mock implementation of Logger for unit testing.
type Logger struct {
	mock.Mock
}

// NewLogger creates a new instance of Logger.
func NewLogger(t mock.TestingT) *Logger {
	m := &Logger{}
	m.Test(t)
	return m
}

func (m *Logger) Debug(msg string, args ...any) {
	m.Called(msg, args)
}

func (m *Logger) Info(msg string, args ...any) {
	m.Called(msg, args)
}

func (m *Logger) Warn(msg string, args ...any) {
	m.Called(msg, args)
}

func (m *Logger) Error(msg string, args ...any) {
	m.Called(msg, args)
}

func (m *Logger) Log(ctx context.Context, level slog.Level, msg string, args ...any) {
	m.Called(ctx, level, msg, args)
}

func (m *Logger) With(args ...any) stdlog.Logger {
	res := m.Called(args)
	if l, ok := res.Get(0).(stdlog.Logger); ok {
		return l
	}
	return m
}

func (m *Logger) WithContext(ctx context.Context) stdlog.Logger {
	res := m.Called(ctx)
	if l, ok := res.Get(0).(stdlog.Logger); ok {
		return l
	}
	return m
}

func (m *Logger) Slog() *slog.Logger {
	args := m.Called()
	if sl, ok := args.Get(0).(*slog.Logger); ok {
		return sl
	}
	return nil
}
