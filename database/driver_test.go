package database

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/baby-hero-404/gocore/stderr"
)

func TestNew_UnsupportedDriverFailsValidation(t *testing.T) {
	ctx := context.Background()
	cfg := Config{
		Driver: "oracle",
		DSN:    "oracle://localhost/db",
	}

	db, err := New(ctx, cfg)
	if err == nil {
		t.Fatal("expected validation error for unsupported driver, got nil")
	}
	if db != nil {
		t.Fatal("expected nil db on failure")
	}

	var stdErr *stderr.Error
	if !errors.As(err, &stdErr) {
		t.Fatalf("expected stderr.Error, got %T", err)
	}

	if stdErr.Code != stderr.CodeValidationFailed {
		t.Errorf("expected code CodeValidationFailed, got %v", stdErr.Code)
	}
}

func TestNewMySQL_EmptyDSNFailsFast(t *testing.T) {
	ctx := context.Background()
	cfg := Config{
		Driver: "mysql",
		DSN:    "",
	}

	db, err := NewMySQL(ctx, cfg)
	if err == nil {
		t.Fatal("expected validation error for empty DSN, got nil")
	}
	if db != nil {
		t.Fatal("expected nil db on failure")
	}

	var stdErr *stderr.Error
	if !errors.As(err, &stdErr) {
		t.Fatalf("expected stderr.Error, got %T", err)
	}

	if stdErr.Code != stderr.CodeValidationFailed {
		t.Errorf("expected code CodeValidationFailed, got %v", stdErr.Code)
	}
}

func TestNewSQLite_InMemorySuccess(t *testing.T) {
	ctx := context.Background()
	cfg := Config{
		Driver:        "sqlite",
		DSN:           "file::memory:?cache=shared",
		RetryAttempts: 1,
		LogLevel:      "silent",
	}

	dbInstance, err := New(ctx, cfg)
	if err != nil {
		t.Fatalf("expected in-memory sqlite connection success, got error: %v", err)
	}
	defer dbInstance.Close()

	if dbInstance.DB() == nil {
		t.Fatal("expected non-nil gorm.DB")
	}

	if err := dbInstance.Ping(ctx); err != nil {
		t.Fatalf("expected ping to succeed, got %v", err)
	}
}

func TestNewMySQL_UnreachableServerRetriesAndFails(t *testing.T) {
	ctx := context.Background()
	cfg := Config{
		Driver:        "mysql",
		DSN:           "user:pass@tcp(127.0.0.1:59999)/test?timeout=1s",
		RetryAttempts: 2,
		RetryBackoff:  10 * time.Millisecond,
		LogLevel:      "silent",
	}

	db, err := NewMySQL(ctx, cfg)
	if err == nil {
		t.Fatal("expected connection failure, got nil")
	}
	if db != nil {
		t.Fatal("expected nil db on connection failure")
	}

	var stdErr *stderr.Error
	if !errors.As(err, &stdErr) {
		t.Fatalf("expected stderr.Error, got %T", err)
	}

	if stdErr.Code != stderr.CodeUnavailable {
		t.Errorf("expected CodeUnavailable, got %v", stdErr.Code)
	}
}

func TestConnect_RetryBackoffStopsOnContextCancel(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 150*time.Millisecond)
	defer cancel()
	cfg := Config{
		Driver:        "sqlite",
		DSN:           "file:/nonexistent-dir/db.sqlite?mode=ro",
		RetryAttempts: 5,
		RetryBackoff:  time.Hour,
	}

	start := time.Now()
	db, err := NewSQLite(ctx, cfg)
	if err == nil || db != nil {
		t.Fatalf("expected failure, got db=%v err=%v", db, err)
	}
	if elapsed := time.Since(start); elapsed > 5*time.Second {
		t.Fatalf("retry wait ignored context cancellation, took %v", elapsed)
	}
}
