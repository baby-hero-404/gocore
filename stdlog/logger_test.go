package stdlog

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"strings"
	"testing"

	"github.com/baby-hero-404/gocore/stderr"
)

func TestLevelFor(t *testing.T) {
	tests := []struct {
		err      error
		expected slog.Level
	}{
		{nil, slog.LevelInfo},
		{stderr.New(stderr.CodeOK, "ok"), slog.LevelInfo},
		{stderr.NewBadRequest("invalid body"), slog.LevelWarn},
		{stderr.NewValidation("field required"), slog.LevelWarn},
		{stderr.NewNotFound("user not found"), slog.LevelWarn},
		{stderr.NewUnauthorized("no auth"), slog.LevelWarn},
		{stderr.NewForbidden("access denied"), slog.LevelWarn},
		{stderr.NewConflict("conflict"), slog.LevelWarn},
		{stderr.NewInternal("db error"), slog.LevelError},
		{stderr.NewUnavailable("redis down"), slog.LevelError},
		{errors.New("plain standard error"), slog.LevelError},
	}

	for _, tt := range tests {
		got := LevelFor(tt.err)
		if got != tt.expected {
			t.Errorf("LevelFor(%v) = %v, want %v", tt.err, got, tt.expected)
		}
	}
}

func TestLogger_Redaction(t *testing.T) {
	var buf bytes.Buffer
	logger := New(
		WithOutput(&buf),
		WithJSON(true),
		WithLevel(slog.LevelDebug),
	)

	logger.Info("user login attempt",
		"username", "john_doe",
		"password", "super_secret_123",
		"auth_token", "jwt.token.here",
		"database_dsn", "postgres://user:pass@localhost:5432/db",
	)

	output := buf.String()
	var logMap map[string]any
	if err := json.Unmarshal([]byte(output), &logMap); err != nil {
		t.Fatalf("failed to parse log JSON: %v, raw: %s", err, output)
	}

	if logMap["username"] != "john_doe" {
		t.Errorf("expected username john_doe, got %v", logMap["username"])
	}
	if logMap["password"] != RedactedPlaceholder {
		t.Errorf("expected password to be redacted, got %v", logMap["password"])
	}
	if logMap["auth_token"] != RedactedPlaceholder {
		t.Errorf("expected auth_token to be redacted, got %v", logMap["auth_token"])
	}
	if logMap["database_dsn"] != RedactedPlaceholder {
		t.Errorf("expected database_dsn to be redacted, got %v", logMap["database_dsn"])
	}

	// Also ensure raw secret never appears anywhere in the buffer
	if strings.Contains(output, "super_secret_123") {
		t.Error("raw password leaked in log output")
	}
}

func TestLogger_WithContext(t *testing.T) {
	var buf bytes.Buffer
	logger := New(
		WithOutput(&buf),
		WithJSON(true),
	)

	ctx := context.WithValue(context.Background(), "request_id", "req-12345")
	ctx = context.WithValue(ctx, "trace_id", "trace-67890")

	logger.WithContext(ctx).Info("processing transaction", "amount", 100)

	output := buf.String()
	var logMap map[string]any
	if err := json.Unmarshal([]byte(output), &logMap); err != nil {
		t.Fatalf("failed to parse log JSON: %v, raw: %s", err, output)
	}

	if logMap["request_id"] != "req-12345" {
		t.Errorf("expected request_id req-12345, got %v", logMap["request_id"])
	}
	if logMap["trace_id"] != "trace-67890" {
		t.Errorf("expected trace_id trace-67890, got %v", logMap["trace_id"])
	}
	if logMap["amount"] != float64(100) {
		t.Errorf("expected amount 100, got %v", logMap["amount"])
	}
}

func TestLogger_ChainingAndPackageLevel(t *testing.T) {
	var buf bytes.Buffer
	customLogger := New(
		WithOutput(&buf),
		WithJSON(true),
	)
	SetDefault(customLogger)

	With("component", "order_service").Warn("stock low", "item_id", 42)

	output := buf.String()
	var logMap map[string]any
	if err := json.Unmarshal([]byte(output), &logMap); err != nil {
		t.Fatalf("failed to parse log JSON: %v, raw: %s", err, output)
	}

	if logMap["component"] != "order_service" {
		t.Errorf("expected component order_service, got %v", logMap["component"])
	}
	if logMap["level"] != "WARN" {
		t.Errorf("expected level WARN, got %v", logMap["level"])
	}

	// Verify package level Info, Debug, Error
	Info("info msg")
	Debug("debug msg")
	Error("error msg")
	if Get() == nil {
		t.Error("expected non-nil default logger from Get()")
	}
}
