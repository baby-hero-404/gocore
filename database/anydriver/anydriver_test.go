package anydriver

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/baby-hero-404/gocore/database"
	"github.com/baby-hero-404/gocore/stderr"
)

func TestNew_SelectsDriver(t *testing.T) {
	ctx := context.Background()
	t.Run("sqlite connects", func(t *testing.T) {
		db, err := New(ctx, database.Config{Driver: "SQLite", DSN: "file:anydrv?mode=memory&cache=shared", LogLevel: "silent"})
		if err != nil {
			t.Fatalf("New: %v", err)
		}
		defer func() { _ = db.Close() }()
		if err := db.Ping(ctx); err != nil {
			t.Fatalf("ping: %v", err)
		}
	})

	unreachable := map[string]string{
		"postgres":       "host=127.0.0.1 port=59999 user=u password=p dbname=d sslmode=disable connect_timeout=1",
		"default driver": "host=127.0.0.1 port=59999 user=u password=p dbname=d sslmode=disable connect_timeout=1",
		"mysql":          "user:pass@tcp(127.0.0.1:59999)/test?timeout=1s",
	}
	drivers := map[string]string{"postgres": "postgres", "default driver": "", "mysql": "mysql"}
	for name, dsn := range unreachable {
		t.Run(name+" is selected", func(t *testing.T) {
			cfg := database.Config{Driver: drivers[name], DSN: dsn, RetryAttempts: 1, RetryBackoff: time.Millisecond, LogLevel: "silent"}
			db, err := New(ctx, cfg)
			var se *stderr.Error
			if db != nil || !errors.As(err, &se) || se.Code != stderr.CodeUnavailable {
				t.Fatalf("expected CodeUnavailable from the selected driver, got db=%v err=%v", db, err)
			}
			if !strings.Contains(strings.ToLower(se.Error()), "connect") {
				t.Errorf("unexpected error text: %v", se)
			}
		})
	}
}

func TestNew_UnknownDriver(t *testing.T) {
	db, err := New(context.Background(), database.Config{Driver: "oracle", DSN: "x"})
	var se *stderr.Error
	if db != nil || !errors.As(err, &se) || se.Code != stderr.CodeValidationFailed {
		t.Fatalf("want CodeValidationFailed, got db=%v err=%v", db, err)
	}
	if len(se.Details) == 0 || se.Details[0].Field != "driver" {
		t.Errorf("expected a FieldError on driver, got %v", se.Details)
	}
}
