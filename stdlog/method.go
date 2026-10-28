package stdlog

import (
	"context"
	"log/slog"
)

// Debug logs at LevelDebug.
func (l *loggerImpl) Debug(msg string, args ...any) {
	l.sl.Debug(msg, args...)
}

// Info logs at LevelInfo.
func (l *loggerImpl) Info(msg string, args ...any) {
	l.sl.Info(msg, args...)
}

// Warn logs at LevelWarn.
func (l *loggerImpl) Warn(msg string, args ...any) {
	l.sl.Warn(msg, args...)
}

// Error logs at LevelError.
func (l *loggerImpl) Error(msg string, args ...any) {
	l.sl.Error(msg, args...)
}

// Log logs with explicit level and context.
func (l *loggerImpl) Log(ctx context.Context, level slog.Level, msg string, args ...any) {
	l.sl.Log(ctx, level, msg, args...)
}

// Package-level helpers forwarding to default logger

// Debug logs at debug level using the default logger.
func Debug(msg string, args ...any) {
	Get().Debug(msg, args...)
}

// Info logs at info level using the default logger.
func Info(msg string, args ...any) {
	Get().Info(msg, args...)
}

// Warn logs at warn level using the default logger.
func Warn(msg string, args ...any) {
	Get().Warn(msg, args...)
}

// Error logs at error level using the default logger.
func Error(msg string, args ...any) {
	Get().Error(msg, args...)
}
