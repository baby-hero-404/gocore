package sqlite

import (
	"context"
	"testing"

	"github.com/baby-hero-404/gocore/database"
)

func TestNew_InMemory(t *testing.T) {
	ctx := context.Background()
	db, err := New(ctx, database.Config{DSN: "file:sqlitepkg?mode=memory&cache=shared", RetryAttempts: 1, LogLevel: "silent"})
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	defer func() { _ = db.Close() }()
	if db.DB() == nil {
		t.Fatal("expected non-nil gorm.DB")
	}
	if err := db.Ping(ctx); err != nil {
		t.Fatalf("ping: %v", err)
	}
}

func TestOpen_InMemory(t *testing.T) {
	db, err := Open(context.Background(), database.Config{DSN: "file:sqlitepkgopen?mode=memory&cache=shared", LogLevel: "silent"})
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	if err := database.Close(db); err != nil {
		t.Fatalf("close: %v", err)
	}
}

func TestEmptyDSNFailsFast(t *testing.T) {
	if db, err := New(context.Background(), database.Config{}); err == nil || db != nil {
		t.Fatalf("expected failure, got db=%v err=%v", db, err)
	}
	if db, err := Open(context.Background(), database.Config{}); err == nil || db != nil {
		t.Fatalf("expected failure, got db=%v err=%v", db, err)
	}
}
