package sentry

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/baby-hero-404/gocore/stderr"
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
