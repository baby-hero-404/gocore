package stdlog

import (
	"context"
	"log/slog"
)

// Logger defines the structured logging contract.
type Logger interface {
	Debug(msg string, args ...any)
	Info(msg string, args ...any)
	Warn(msg string, args ...any)
	Error(msg string, args ...any)
	Log(ctx context.Context, level slog.Level, msg string, args ...any)
	With(args ...any) Logger
	WithContext(ctx context.Context) Logger
	Slog() *slog.Logger
}
