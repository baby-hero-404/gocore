package database

import (
	"context"
	"testing"
	"time"

	"github.com/baby-hero-404/gocore/stderr"
	"gorm.io/driver/mysql"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func TestConnect_EmptyDSNFailsFast(t *testing.T) {
	db, err := Connect(context.Background(), sqlite.Open(""), Config{})
	if err == nil || db != nil {
		t.Fatalf("expected validation failure, got db=%v err=%v", db, err)
	}
	var found bool
	if se := asStdErr(t, err); se != nil {
		for _, d := range se.Details {
			found = found || d.Field == "dsn"
		}
	}
	if !found {
		t.Errorf("expected a FieldError on dsn, got %v", err)
	}
}

func TestConnect_NilDialectorIsAValidationError(t *testing.T) {
	db, err := Connect(context.Background(), nil, Config{DSN: "x"})
	if db != nil || asStdErr(t, err).Code != stderr.CodeValidationFailed {
		t.Fatalf("want validation error, got db=%v err=%v", db, err)
	}
	if _, err := Open(context.Background(), nil, Config{DSN: "x"}); err == nil {
		t.Fatal("Open must reject a nil dialector too")
	}
}

func TestOpen_InMemorySQLite(t *testing.T) {
	ctx := context.Background()
	dsn := "file:openmem?mode=memory&cache=shared"
	db, err := Open(ctx, sqlite.Open(dsn), Config{DSN: dsn, RetryAttempts: 1, LogLevel: "silent"})
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	defer func() { _ = db.Close() }()
	if db.DB() == nil {
		t.Fatal("expected non-nil gorm.DB")
	}
	if err := db.Ping(ctx); err != nil {
		t.Fatalf("ping: %v", err)
	}
}

func TestOpen_FailureReturnsNilDatabase(t *testing.T) {
	db, err := Open(context.Background(), sqlite.Open(""), Config{})
	if err == nil || db != nil {
		t.Fatalf("expected nil Database and an error, got db=%v err=%v", db, err)
	}
}

func TestConnect_UnreachableServerRetriesAndFails(t *testing.T) {
	cfg := Config{
		DSN:           "user:pass@tcp(127.0.0.1:59999)/test?timeout=1s",
		RetryAttempts: 2,
		RetryBackoff:  10 * time.Millisecond,
		LogLevel:      "silent",
	}
	db, err := Connect(context.Background(), mysql.Open(cfg.DSN), cfg)
	if err == nil || db != nil {
		t.Fatalf("expected connection failure, got db=%v err=%v", db, err)
	}
	if got := codeOf(t, err); got != stderr.CodeUnavailable {
		t.Errorf("code = %v, want CodeUnavailable", got)
	}
}

func TestConnect_RetryBackoffStopsOnContextCancel(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 150*time.Millisecond)
	defer cancel()
	cfg := Config{
		DSN:           "file:/nonexistent-dir/db.sqlite?mode=ro",
		RetryAttempts: 5,
		RetryBackoff:  time.Hour,
	}

	start := time.Now()
	db, err := Connect(ctx, sqlite.Open(cfg.DSN), cfg)
	if err == nil || db != nil {
		t.Fatalf("expected failure, got db=%v err=%v", db, err)
	}
	if elapsed := time.Since(start); elapsed > 5*time.Second {
		t.Fatalf("retry wait ignored context cancellation, took %v", elapsed)
	}
}

func TestDefaultConfig(t *testing.T) {
	cfg := DefaultConfig()
	if cfg.MaxOpenConns != 25 || cfg.MaxIdleConns != 10 || cfg.RetryAttempts != 3 {
		t.Errorf("unexpected defaults: %+v", cfg)
	}
}

func TestOptionsCompile(t *testing.T) {
	// WithGormConfig must be accepted by Open without a nil logger panicking.
	dsn := "file:optcompile?mode=memory&cache=shared"
	db, err := Open(context.Background(), sqlite.Open(dsn), Config{DSN: dsn, LogLevel: "silent"}, WithGormConfig(&gorm.Config{}))
	if err != nil {
		t.Fatal(err)
	}
	_ = db.Close()
}
