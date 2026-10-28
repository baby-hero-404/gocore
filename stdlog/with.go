package stdlog

import (
	"context"
	"log/slog"
)

// typedContextKey is a typed context key to avoid collision with other packages.
type typedContextKey struct{ name string }

// Typed context keys — preferred over string keys.
var (
	RequestIDCtxKey     = typedContextKey{name: "request_id"}
	TraceIDCtxKey       = typedContextKey{name: "trace_id"}
	CorrelationIDCtxKey = typedContextKey{name: "correlation_id"}
)

// typedKeys maps typed context keys to their log attribute name.
var typedKeys = []typedContextKey{
	RequestIDCtxKey,
	TraceIDCtxKey,
	CorrelationIDCtxKey,
}

// stringKeys provides backward compatibility with services using string context keys.
var stringKeys = []string{
	"request_id",
	"requestID",
	"requestId",
	"trace_id",
	"traceID",
	"traceId",
	"correlation_id",
	"correlationID",
}

// WithContext returns a new Logger with request and trace IDs extracted from ctx.
// It first checks typed keys, then falls back to string keys for backward compatibility.
func (l *loggerImpl) WithContext(ctx context.Context) Logger {
	if ctx == nil {
		return l
	}

	seen := make(map[string]bool)
	var attrs []any

	// 1. Typed keys (preferred — no collision risk)
	for _, key := range typedKeys {
		if val, ok := ctx.Value(key).(string); ok && val != "" && !seen[key.name] {
			attrs = append(attrs, slog.String(key.name, val))
			seen[key.name] = true
		}
	}

	// 2. String keys (backward compatibility)
	for _, key := range stringKeys {
		normalized := normalizeKeyName(key)
		if seen[normalized] {
			continue
		}
		if val, ok := ctx.Value(key).(string); ok && val != "" {
			attrs = append(attrs, slog.String(normalized, val))
			seen[normalized] = true
		}
	}

	if len(attrs) == 0 {
		return l
	}

	return &loggerImpl{sl: l.sl.With(attrs...)}
}

// With adds key-value attributes to the logger.
func (l *loggerImpl) With(args ...any) Logger {
	return &loggerImpl{sl: l.sl.With(args...)}
}

// Slog returns the underlying *slog.Logger.
func (l *loggerImpl) Slog() *slog.Logger {
	return l.sl
}

// WithContext returns the default logger enriched with the IDs found in ctx.
func WithContext(ctx context.Context) Logger {
	return Get().WithContext(ctx)
}

// With returns the default logger with args attached to every record.
func With(args ...any) Logger {
	return Get().With(args...)
}

// normalizeKeyName maps camelCase variants to their canonical snake_case form.
func normalizeKeyName(key string) string {
	switch key {
	case "requestID", "requestId":
		return "request_id"
	case "traceID", "traceId":
		return "trace_id"
	case "correlationID":
		return "correlation_id"
	default:
		return key
	}
}
