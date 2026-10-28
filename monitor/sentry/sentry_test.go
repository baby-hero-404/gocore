package sentry

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/baby-hero-404/gocore/stderr"
	"github.com/baby-hero-404/gocore/stdlog"
)

func TestSentry_EmptyDSNFailsFast(t *testing.T) {
	err := Init(Config{DSN: ""})
	if err == nil {
		t.Fatal("expected validation error for empty DSN")
	}
	var stdErr *stderr.Error
	if !errors.As(err, &stdErr) || stdErr.Code != stderr.CodeValidationFailed {
		t.Errorf("expected CodeValidationFailed, got %v", err)
	}
}

func TestSentry_CaptureAndFlush(t *testing.T) {
	// Calling CaptureError with nil should not panic
	CaptureError(context.Background(), nil)
	CaptureError(context.Background(), errors.New("sample error"))
	_ = Flush(10 * time.Millisecond)
}

func TestDefaultConfig(t *testing.T) {
	cfg := DefaultConfig()
	if cfg.Environment != "development" || cfg.TracesSampleRate != 1.0 || cfg.DSN != "" {
		t.Errorf("unexpected defaults: %+v", cfg)
	}
}

func TestInit(t *testing.T) {
	tests := []struct {
		name     string
		cfg      Config
		wantCode stderr.Code // 0 = success
	}{
		{"blank dsn", Config{DSN: "   "}, stderr.CodeValidationFailed},
		{"malformed dsn", Config{DSN: "not a dsn"}, stderr.CodeInternal},
		{"valid dsn", Config{DSN: "https://public@example.invalid/1", Environment: "test", Release: "v1"}, 0},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := Init(tt.cfg)
			if tt.wantCode == 0 {
				if err != nil {
					t.Fatalf("unexpected error: %v", err)
				}
				return
			}
			var se *stderr.Error
			if !errors.As(err, &se) || se.Code != tt.wantCode {
				t.Fatalf("want code %v, got %v", tt.wantCode, err)
			}
		})
	}
}

func TestCaptureError_WithRequestIDAndNilContext(t *testing.T) {
	ctx := stdlog.ContextWithRequestID(context.Background(), "rid-9")
	CaptureError(ctx, errors.New("with request id"))
	CaptureError(context.Background(), errors.New("without request id"))
	CaptureError(nil, errors.New("nil context")) //nolint:staticcheck // CaptureError documents tolerating a nil context
}

func TestConfig_SecretOmittedFromJSON(t *testing.T) {
	cfg := Config{
		DSN:         "https://my_secret_token@sentry.io/12345",
		Environment: "production",
	}
	b, err := json.Marshal(cfg)
	if err != nil {
		t.Fatalf("json.Marshal: %v", err)
	}
	out := string(b)
	if strings.Contains(out, "my_secret_token") || strings.Contains(out, "dsn") {
		t.Errorf("expected secret DSN to be omitted from JSON, got %s", out)
	}
}
