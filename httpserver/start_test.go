package httpserver

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"testing"
	"time"

	"github.com/baby-hero-404/gocore/stderr"
	"github.com/gin-gonic/gin"
)

type ctxKey struct{}

func freePort(t *testing.T) int {
	t.Helper()
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = ln.Close() }()
	return ln.Addr().(*net.TCPAddr).Port
}

// run starts srv and returns a channel receiving Start's result.
func run(srv Server, ctx context.Context) <-chan error {
	done := make(chan error, 1)
	go func() { done <- srv.Start(ctx) }()
	return done
}

func waitUp(t *testing.T, port int) {
	t.Helper()
	deadline := time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) {
		if c, err := net.DialTimeout("tcp", fmt.Sprintf("127.0.0.1:%d", port), 50*time.Millisecond); err == nil {
			_ = c.Close()
			return
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatalf("server on port %d did not come up", port)
}

func wait(t *testing.T, done <-chan error) error {
	t.Helper()
	select {
	case err := <-done:
		return err
	case <-time.After(5 * time.Second):
		t.Fatal("Start did not return")
		return nil
	}
}

func get(t *testing.T, port int, path string) (int, string) {
	t.Helper()
	resp, err := http.Get(fmt.Sprintf("http://127.0.0.1:%d%s", port, path)) //nolint:noctx // test client
	if err != nil {
		t.Fatalf("GET %s: %v", path, err)
	}
	defer func() { _ = resp.Body.Close() }()
	b, _ := io.ReadAll(resp.Body)
	return resp.StatusCode, string(b)
}

func TestStart_ServesUntilContextCancelled(t *testing.T) {
	gin.SetMode(gin.TestMode)
	port := freePort(t)
	srv := New(Config{Port: port})
	srv.Engine().GET("/ping", func(c *gin.Context) {
		c.String(http.StatusOK, "pong:%v", c.Request.Context().Value(ctxKey{}))
	})

	ctx, cancel := context.WithCancel(context.WithValue(context.Background(), ctxKey{}, "v1"))
	defer cancel()
	done := run(srv, ctx)
	waitUp(t, port)

	if status, body := get(t, port, "/ping"); status != http.StatusOK || body != "pong:v1" {
		t.Fatalf("got %d %q, want 200 \"pong:v1\" (request ctx must inherit Start's values)", status, body)
	}

	cancel()
	if err := wait(t, done); err != nil {
		t.Fatalf("graceful stop must return nil, got %v", err)
	}
	if _, err := http.Get(fmt.Sprintf("http://127.0.0.1:%d/ping", port)); err == nil { //nolint:noctx,bodyclose // expecting failure
		t.Error("server still accepting connections after Start returned")
	}
}

func TestStart_GracefulShutdownLetsInFlightRequestsFinish(t *testing.T) {
	gin.SetMode(gin.TestMode)
	port := freePort(t)
	srv := New(Config{Port: port, ShutdownTimeout: 3 * time.Second})
	started := make(chan struct{})
	srv.Engine().GET("/slow", func(c *gin.Context) {
		close(started)
		time.Sleep(300 * time.Millisecond)
		c.String(http.StatusOK, "done")
	})

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	done := run(srv, ctx)
	waitUp(t, port)

	type result struct {
		status int
		body   string
	}
	res := make(chan result, 1)
	go func() {
		s, b := get(t, port, "/slow")
		res <- result{s, b}
	}()
	<-started
	cancel() // shutdown begins while the request is running

	if r := <-res; r.status != http.StatusOK || r.body != "done" {
		t.Errorf("in-flight request was aborted: %+v", r)
	}
	if err := wait(t, done); err != nil {
		t.Fatalf("Start = %v, want nil", err)
	}
}

func TestStart_ShutdownTimeoutIsReported(t *testing.T) {
	gin.SetMode(gin.TestMode)
	port := freePort(t)
	srv := New(Config{Port: port, ShutdownTimeout: 50 * time.Millisecond})
	started, release := make(chan struct{}), make(chan struct{})
	srv.Engine().GET("/stuck", func(c *gin.Context) {
		close(started)
		<-release
	})
	t.Cleanup(func() { close(release) })

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	done := run(srv, ctx)
	waitUp(t, port)
	reqErr := make(chan error, 1)
	go func() {
		resp, err := http.Get(fmt.Sprintf("http://127.0.0.1:%d/stuck", port)) //nolint:noctx // test client
		if err == nil {
			_ = resp.Body.Close()
		}
		reqErr <- err
	}()
	<-started
	cancel()

	err := wait(t, done)
	var se *stderr.Error
	if !errors.As(err, &se) || se.Code != stderr.CodeInternal || !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("want CodeInternal wrapping DeadlineExceeded, got %v", err)
	}

	// The stuck request must be cut off (connection closed), not left hanging until process exit.
	select {
	case err := <-reqErr:
		if err == nil {
			t.Error("the stuck request unexpectedly succeeded")
		}
	case <-time.After(3 * time.Second):
		t.Error("the stuck connection was not closed after the shutdown timeout")
	}
}

func TestStart_PortInUseReturnsError(t *testing.T) {
	ln, err := net.Listen("tcp", ":0")
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = ln.Close() }()
	port := ln.Addr().(*net.TCPAddr).Port

	err = wait(t, run(New(Config{Port: port}), context.Background()))
	var se *stderr.Error
	if !errors.As(err, &se) || se.Code != stderr.CodeInternal {
		t.Fatalf("want CodeInternal, got %v", err)
	}
}

func TestStart_AlreadyCancelledContextStopsCleanly(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if err := wait(t, run(New(Config{Port: freePort(t)}), ctx)); err != nil {
		t.Fatalf("Start = %v, want nil", err)
	}
}

func TestConfigShutdownTimeoutDefault(t *testing.T) {
	tests := []struct {
		name string
		in   time.Duration
		want time.Duration
	}{
		{"unset gets default", 0, DefaultConfig().ShutdownTimeout},
		{"negative gets default", -time.Second, DefaultConfig().ShutdownTimeout},
		{"explicit kept", 2 * time.Second, 2 * time.Second},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := Config{ShutdownTimeout: tt.in}
			c.applyDefaults()
			if c.ShutdownTimeout != tt.want {
				t.Errorf("got %v, want %v", c.ShutdownTimeout, tt.want)
			}
		})
	}
}
