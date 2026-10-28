package database

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/baby-hero-404/gocore/stderr"
	"gorm.io/gorm"
)

func TestNewPostgres_EmptyDSNFailsFast(t *testing.T) {
	ctx := context.Background()
	cfg := Config{
		DSN: "",
	}

	db, err := NewPostgres(ctx, cfg)
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

	if len(stdErr.Details) == 0 || stdErr.Details[0].Field != "dsn" {
		t.Errorf("expected detail for field dsn, got %v", stdErr.Details)
	}
}

func TestNewPostgres_UnreachableServerRetriesAndFails(t *testing.T) {
	ctx := context.Background()
	// An unreachable port with fast retries for testing
	cfg := Config{
		DSN:           "host=127.0.0.1 port=59999 user=invalid password=invalid dbname=test sslmode=disable connect_timeout=1",
		RetryAttempts: 2,
		RetryBackoff:  10 * time.Millisecond,
		LogLevel:      "silent",
	}

	db, err := NewPostgres(ctx, cfg)
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
		t.Errorf("expected code CodeUnavailable, got %v", stdErr.Code)
	}
}

func TestDefaultConfig(t *testing.T) {
	cfg := DefaultConfig()
	if cfg.MaxOpenConns != 25 {
		t.Errorf("expected MaxOpenConns 25, got %d", cfg.MaxOpenConns)
	}
	if cfg.MaxIdleConns != 10 {
		t.Errorf("expected MaxIdleConns 10, got %d", cfg.MaxIdleConns)
	}
	if cfg.RetryAttempts != 3 {
		t.Errorf("expected RetryAttempts 3, got %d", cfg.RetryAttempts)
	}
}

func TestNew_EmptyDSNFailsFast(t *testing.T) {
	ctx := context.Background()
	cfg := Config{DSN: ""}
	db, err := New(ctx, cfg, WithGormConfig(&gorm.Config{}))
	if err == nil {
		t.Fatal("expected validation error for empty DSN")
	}
	if db != nil {
		t.Fatal("expected nil db interface on failure")
	}
}
