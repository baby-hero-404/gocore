package stdlog

import (
	"context"
	"log/slog"
	"strings"
)

const RedactedPlaceholder = "[REDACTED]"

var defaultSensitiveKeywords = []string{
	"password",
	"token",
	"secret",
	"authorization",
	"dsn",
	"api_key",
	"apikey",
	"access_token",
	"refresh_token",
	"private_key",
}

// IsSensitiveKey returns true if the key contains any sensitive keyword.
func IsSensitiveKey(key string) bool {
	lower := strings.ToLower(key)
	for _, kw := range defaultSensitiveKeywords {
		if strings.Contains(lower, kw) {
			return true
		}
	}
	return false
}

// SanitizeAttr inspects an attribute and replaces its value with [REDACTED] if the key is sensitive.
func SanitizeAttr(groups []string, a slog.Attr) slog.Attr {
	if a.Value.Kind() == slog.KindGroup {
		return a
	}

	if IsSensitiveKey(a.Key) {
		return slog.String(a.Key, RedactedPlaceholder)
	}

	return a
}

// RedactingHandler wraps an existing slog.Handler to automatically sanitize attributes.
type RedactingHandler struct {
	inner slog.Handler
}

// NewRedactingHandler creates a new RedactingHandler wrapping inner.
func NewRedactingHandler(inner slog.Handler) *RedactingHandler {
	return &RedactingHandler{inner: inner}
}

func (h *RedactingHandler) Enabled(ctx context.Context, level slog.Level) bool {
	return h.inner.Enabled(ctx, level)
}

func (h *RedactingHandler) Handle(ctx context.Context, r slog.Record) error {
	newRecord := slog.NewRecord(r.Time, r.Level, r.Message, r.PC)

	r.Attrs(func(a slog.Attr) bool {
		newRecord.AddAttrs(sanitizeAttrRecursive(a))
		return true
	})

	return h.inner.Handle(ctx, newRecord)
}

func (h *RedactingHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	sanitized := make([]slog.Attr, len(attrs))
	for i, a := range attrs {
		sanitized[i] = sanitizeAttrRecursive(a)
	}
	return &RedactingHandler{inner: h.inner.WithAttrs(sanitized)}
}

func (h *RedactingHandler) WithGroup(name string) slog.Handler {
	return &RedactingHandler{inner: h.inner.WithGroup(name)}
}

func sanitizeAttrRecursive(a slog.Attr) slog.Attr {
	if a.Value.Kind() == slog.KindGroup {
		groupAttrs := a.Value.Group()
		sanitizedGroup := make([]any, len(groupAttrs))
		for i, ga := range groupAttrs {
			sanitizedGroup[i] = sanitizeAttrRecursive(ga)
		}
		return slog.Group(a.Key, sanitizedGroup...)
	}

	if IsSensitiveKey(a.Key) {
		return slog.String(a.Key, RedactedPlaceholder)
	}

	return a
}
