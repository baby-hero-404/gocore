package graylog

import (
	"errors"
	"io"
	"net"
	"testing"
	"time"

	"github.com/baby-hero-404/gocore/stderr"
)

func TestGraylog_EmptyAddrFailsFast(t *testing.T) {
	_, err := NewWriter(Config{Addr: ""})
	if err == nil {
		t.Fatal("expected validation error for empty Addr")
	}
	var stdErr *stderr.Error
	if !errors.As(err, &stdErr) || stdErr.Code != stderr.CodeValidationFailed {
		t.Errorf("expected CodeValidationFailed, got %v", err)
	}
}

func TestDefaultConfig(t *testing.T) {
	if got := DefaultConfig().Addr; got != "127.0.0.1:12201" {
		t.Errorf("default addr = %q", got)
	}
}

func TestNewWriter_BlankAddrVariantsFailValidation(t *testing.T) {
	for _, addr := range []string{"", "   ", "\t"} {
		if _, err := NewWriter(Config{Addr: addr}); err == nil {
			t.Errorf("addr %q: expected validation error", addr)
		}
	}
}

func TestNewWriter_UnresolvableAddrIsUnavailable(t *testing.T) {
	_, err := NewWriter(Config{Addr: "not-a-valid-address"})
	var stdErr *stderr.Error
	if !errors.As(err, &stdErr) || stdErr.Code != stderr.CodeUnavailable {
		t.Fatalf("want CodeUnavailable, got %v", err)
	}
}

func TestNewWriter_SendsGELFToCollector(t *testing.T) {
	conn, err := net.ListenPacket("udp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = conn.Close() }()

	w, err := NewWriter(Config{Addr: conn.LocalAddr().String()})
	if err != nil {
		t.Fatalf("NewWriter: %v", err)
	}
	if c, ok := w.(io.Closer); ok {
		defer func() { _ = c.Close() }()
	}
	if _, err := w.Write([]byte("hello graylog")); err != nil {
		t.Fatalf("write: %v", err)
	}

	_ = conn.SetReadDeadline(time.Now().Add(2 * time.Second))
	buf := make([]byte, 8192)
	n, _, err := conn.ReadFrom(buf)
	if err != nil || n == 0 {
		t.Fatalf("collector received nothing: n=%d err=%v", n, err)
	}
}
