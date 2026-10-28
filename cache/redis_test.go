package cache

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/baby-hero-404/gocore/stderr"
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
