package cache

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/baby-hero-404/gocore/stderr"
	"github.com/redis/go-redis/v9"
)

func TestNewRedis_EmptyAddrFailsFast(t *testing.T) {
	ctx := context.Background()
	cfg := Config{
		Addr: "",
	}

	client, err := NewRedis(ctx, cfg)
	if err == nil {
		t.Fatal("expected validation error for empty Addr, got nil")
	}
	if client != nil {
		t.Fatal("expected nil client on failure")
	}

	var stdErr *stderr.Error
	if !errors.As(err, &stdErr) {
		t.Fatalf("expected stderr.Error, got %T", err)
	}

	if stdErr.Code != stderr.CodeValidationFailed {
		t.Errorf("expected code CodeValidationFailed, got %v", stdErr.Code)
	}
}

func TestNewRedis_UnreachableFailsGracefully(t *testing.T) {
	ctx := context.Background()
	cfg := Config{
		Addr:        "127.0.0.1:59998",
		DialTimeout: 50 * time.Millisecond,
		MaxRetries:  1,
	}

	client, err := NewRedis(ctx, cfg)
	if err == nil {
		t.Fatal("expected connection error, got nil")
	}
	if client != nil {
		t.Fatal("expected nil client on connection failure")
	}

	var stdErr *stderr.Error
	if !errors.As(err, &stdErr) {
		t.Fatalf("expected stderr.Error, got %T", err)
	}

	if stdErr.Code != stderr.CodeUnavailable {
		t.Errorf("expected code CodeUnavailable, got %v", stdErr.Code)
	}
}

func TestWrapError(t *testing.T) {
	tests := []struct {
		err        error
		wantCode   stderr.Code
		wantStatus int
	}{
		{redis.Nil, stderr.CodeNotFound, 404},
		{redis.TxFailedErr, stderr.CodeConflict, 409},
		{errors.New("generic redis failure"), stderr.CodeCacheError, 500},
	}

	for _, tt := range tests {
		wrapped := WrapError(tt.err)
		var stdErr *stderr.Error
		if !errors.As(wrapped, &stdErr) {
			t.Fatalf("expected stderr.Error, got %T", wrapped)
		}
		if stdErr.Code != tt.wantCode {
			t.Errorf("WrapError(%v) code = %v, want %v", tt.err, stdErr.Code, tt.wantCode)
		}
		if stdErr.HTTPStatus() != tt.wantStatus {
			t.Errorf("WrapError(%v) status = %d, want %d", tt.err, stdErr.HTTPStatus(), tt.wantStatus)
		}
	}

	if WrapError(nil) != nil {
		t.Error("expected nil for WrapError(nil)")
	}
}

func TestPingAndClose_NilClient(t *testing.T) {
	ctx := context.Background()
	if err := Ping(ctx, nil); err == nil {
		t.Error("expected error pinging nil client")
	}
	if err := Close(nil); err != nil {
		t.Errorf("expected nil error closing nil client, got %v", err)
	}
}

func TestDefaultConfig(t *testing.T) {
	cfg := DefaultConfig()
	if cfg.PoolSize != 20 {
		t.Errorf("expected PoolSize 20, got %d", cfg.PoolSize)
	}
	if cfg.DialTimeout != 5*time.Second {
		t.Errorf("expected DialTimeout 5s, got %v", cfg.DialTimeout)
	}
}

func TestNew_EmptyAddrFailsFast(t *testing.T) {
	ctx := context.Background()
	cfg := Config{Addr: ""}
	c, err := New(ctx, cfg, WithDialTimeout(time.Second), WithPoolSize(10))
	if err == nil {
		t.Fatal("expected error for empty addr")
	}
	if c != nil {
		t.Fatal("expected nil cache interface")
	}
}
