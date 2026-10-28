package postgres

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/baby-hero-404/gocore/database"
	"github.com/baby-hero-404/gocore/stderr"
)

func codeAndDetails(t *testing.T, err error) (stderr.Code, []stderr.FieldError) {
	t.Helper()
	var se *stderr.Error
	if !errors.As(err, &se) {
		t.Fatalf("expected *stderr.Error, got %T (%v)", err, err)
	}
	return se.Code, se.Details
}

func TestOpenAndNew_EmptyDSNFailsFast(t *testing.T) {
	calls := map[string]func() (any, error){
		"Open": func() (any, error) { return Open(context.Background(), database.Config{}) },
		"New":  func() (any, error) { return New(context.Background(), database.Config{}) },
	}
	for name, call := range calls {
		t.Run(name, func(t *testing.T) {
			got, err := call()
			if err == nil {
				t.Fatal("expected validation error")
			}
			code, details := codeAndDetails(t, err)
			if code != stderr.CodeValidationFailed || len(details) == 0 || details[0].Field != "dsn" {
				t.Errorf("code=%v details=%v, want validation failure on dsn", code, details)
			}
			if v, ok := got.(interface{ Close() error }); ok && v != nil {
				t.Errorf("expected nil result on failure, got %T", got)
			}
		})
	}
}

func TestOpen_UnreachableServerRetriesAndFails(t *testing.T) {
	cfg := database.Config{
		DSN:           "host=127.0.0.1 port=59999 user=invalid password=invalid dbname=test sslmode=disable connect_timeout=1",
		RetryAttempts: 2,
		RetryBackoff:  10 * time.Millisecond,
		LogLevel:      "silent",
	}
	db, err := Open(context.Background(), cfg)
	if err == nil || db != nil {
		t.Fatalf("expected connection failure, got db=%v err=%v", db, err)
	}
	if code, _ := codeAndDetails(t, err); code != stderr.CodeUnavailable {
		t.Errorf("code = %v, want CodeUnavailable", code)
	}
}
