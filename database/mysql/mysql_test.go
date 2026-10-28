package mysql

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/baby-hero-404/gocore/database"
	"github.com/baby-hero-404/gocore/stderr"
)

func code(t *testing.T, err error) stderr.Code {
	t.Helper()
	var se *stderr.Error
	if !errors.As(err, &se) {
		t.Fatalf("expected *stderr.Error, got %T (%v)", err, err)
	}
	return se.Code
}

func TestEmptyDSNFailsFast(t *testing.T) {
	if db, err := Open(context.Background(), database.Config{}); err == nil || db != nil {
		t.Fatalf("Open: expected failure, got db=%v err=%v", db, err)
	} else if code(t, err) != stderr.CodeValidationFailed {
		t.Errorf("Open code = %v", code(t, err))
	}
	if db, err := New(context.Background(), database.Config{}); err == nil || db != nil {
		t.Fatalf("New: expected failure, got db=%v err=%v", db, err)
	}
}

func TestOpen_UnreachableServerRetriesAndFails(t *testing.T) {
	cfg := database.Config{
		DSN:           "user:pass@tcp(127.0.0.1:59999)/test?timeout=1s",
		RetryAttempts: 2,
		RetryBackoff:  10 * time.Millisecond,
		LogLevel:      "silent",
	}
	db, err := Open(context.Background(), cfg)
	if err == nil || db != nil {
		t.Fatalf("expected connection failure, got db=%v err=%v", db, err)
	}
	if code(t, err) != stderr.CodeUnavailable {
		t.Errorf("code = %v, want CodeUnavailable", code(t, err))
	}
}
