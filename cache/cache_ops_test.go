package cache

import (
	"context"
	"encoding/json"
	"errors"
	"net"
	"strings"
	"testing"
	"time"

	"github.com/baby-hero-404/gocore/stderr"
	"github.com/redis/go-redis/v9"
)

func newCache(t *testing.T, f *fakeRedis, opts ...Option) Cache {
	t.Helper()
	c, err := New(context.Background(), Config{Addr: f.addr()}, opts...)
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	t.Cleanup(func() { _ = c.Close() })
	return c
}

func codeOf(t *testing.T, err error) stderr.Code {
	t.Helper()
	var se *stderr.Error
	if !errors.As(err, &se) {
		t.Fatalf("expected *stderr.Error, got %T (%v)", err, err)
	}
	return se.Code
}

func TestCache_RoundTrip(t *testing.T) {
	f := newFakeRedis(t)
	c := newCache(t, f)
	ctx := context.Background()

	if c.Client() == nil {
		t.Fatal("Client() must expose the redis client")
	}
	if err := c.Ping(ctx); err != nil {
		t.Fatalf("ping: %v", err)
	}
	if err := c.Set(ctx, "k", "v", time.Minute); err != nil {
		t.Fatalf("set: %v", err)
	}
	if got := f.ttlOf("k"); got != time.Minute {
		t.Errorf("ttl = %v, want 1m", got)
	}
	if got, err := c.Get(ctx, "k"); err != nil || got != "v" {
		t.Fatalf("get = %q, %v", got, err)
	}
	if n, err := c.Exists(ctx, "k", "missing"); err != nil || n != 1 {
		t.Fatalf("exists = %d, %v", n, err)
	}
	if ok, err := c.Expire(ctx, "k", 5*time.Second); err != nil || !ok || f.ttlOf("k") != 5*time.Second {
		t.Fatalf("expire ok=%v err=%v ttl=%v", ok, err, f.ttlOf("k"))
	}
	if ok, err := c.Expire(ctx, "missing", time.Second); err != nil || ok {
		t.Fatalf("expire on missing key: ok=%v err=%v", ok, err)
	}
	if n, err := c.Del(ctx, "k", "missing"); err != nil || n != 1 {
		t.Fatalf("del = %d, %v", n, err)
	}
	if _, err := c.Get(ctx, "k"); codeOf(t, err) != stderr.CodeNotFound {
		t.Fatalf("get after delete must be CodeNotFound, got %v", err)
	}
}

func TestCache_ServerErrorsAreWrapped(t *testing.T) {
	f := newFakeRedis(t)
	c := newCache(t, f)
	f.fail.Store(true)
	ctx := context.Background()

	ops := map[string]func() error{
		"get":    func() error { _, err := c.Get(ctx, "k"); return err },
		"set":    func() error { return c.Set(ctx, "k", "v", 0) },
		"del":    func() error { _, err := c.Del(ctx, "k"); return err },
		"exists": func() error { _, err := c.Exists(ctx, "k"); return err },
		"expire": func() error { _, err := c.Expire(ctx, "k", time.Second); return err },
		"ping":   func() error { return c.Ping(ctx) },
	}
	for name, op := range ops {
		t.Run(name, func(t *testing.T) {
			if got := codeOf(t, op()); got != stderr.CodeCacheError {
				t.Errorf("code = %v, want CodeCacheError", got)
			}
		})
	}
}

func TestCache_TimeoutMapsToTimeoutCode(t *testing.T) {
	f := newFakeRedis(t)
	c := newCache(t, f, WithReadTimeout(50*time.Millisecond))
	f.hang.Store(true)

	_, err := c.Get(context.Background(), "k")
	if got := codeOf(t, err); got != stderr.CodeTimeout {
		t.Errorf("code = %v, want CodeTimeout (err=%v)", got, err)
	}
}

func TestOptionsOverrideConfig(t *testing.T) {
	f := newFakeRedis(t)
	c := newCache(t, f,
		WithDialTimeout(time.Second),
		WithReadTimeout(2*time.Second),
		WithWriteTimeout(3*time.Second),
		WithPoolSize(7),
	)
	o := c.Client().Options()
	if o.DialTimeout != time.Second || o.ReadTimeout != 2*time.Second || o.WriteTimeout != 3*time.Second || o.PoolSize != 7 {
		t.Errorf("options not applied: %+v", o)
	}
}

func TestPackageLevelHelpersHandleNilClient(t *testing.T) {
	if err := Close(nil); err != nil {
		t.Errorf("Close(nil) = %v, want nil", err)
	}
	if got := codeOf(t, Ping(context.Background(), nil)); got != stderr.CodeBadRequest {
		t.Errorf("Ping(nil) code = %v, want CodeBadRequest", got)
	}
}

func TestClose_AlreadyClosedClientIsWrapped(t *testing.T) {
	f := newFakeRedis(t)
	client, err := NewRedis(context.Background(), Config{Addr: f.addr()})
	if err != nil {
		t.Fatal(err)
	}
	if err := Close(client); err != nil {
		t.Fatalf("first close: %v", err)
	}
	if got := codeOf(t, Close(client)); got != stderr.CodeCacheError {
		t.Errorf("second close code = %v, want CodeCacheError", got)
	}
}

type timeoutErr struct{}

func (timeoutErr) Error() string   { return "i/o timeout" }
func (timeoutErr) Timeout() bool   { return true }
func (timeoutErr) Temporary() bool { return true }

var _ net.Error = timeoutErr{}

func TestWrapError(t *testing.T) {
	tests := []struct {
		name string
		err  error
		want stderr.Code // 0 means nil result
	}{
		{"nil", nil, 0},
		{"redis nil is not found", redis.Nil, stderr.CodeNotFound},
		{"wrapped redis nil is not found", errors.Join(errors.New("ctx"), redis.Nil), stderr.CodeNotFound},
		{"transaction failure is conflict", redis.TxFailedErr, stderr.CodeConflict},
		{"net timeout", timeoutErr{}, stderr.CodeTimeout},
		{"anything else", errors.New("boom"), stderr.CodeCacheError},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := WrapError(tt.err)
			if tt.want == 0 {
				if got != nil {
					t.Fatalf("got %v, want nil", got)
				}
				return
			}
			if codeOf(t, got) != tt.want {
				t.Errorf("code = %v, want %v", codeOf(t, got), tt.want)
			}
			if !errors.Is(got, tt.err) {
				t.Error("original error must stay reachable via errors.Is")
			}
		})
	}
}

func TestConfigApplyDefaults(t *testing.T) {
	def := DefaultConfig()
	tests := []struct {
		name string
		in   Config
		want Config
	}{
		{"zero values get defaults", Config{}, Config{DialTimeout: def.DialTimeout, ReadTimeout: def.ReadTimeout, WriteTimeout: def.WriteTimeout, PoolSize: def.PoolSize, MaxRetries: def.MaxRetries}},
		{"explicit values kept", Config{DialTimeout: time.Second, ReadTimeout: 2 * time.Second, WriteTimeout: 3 * time.Second, PoolSize: 9, MinIdleConns: 2, MaxRetries: 5},
			Config{DialTimeout: time.Second, ReadTimeout: 2 * time.Second, WriteTimeout: 3 * time.Second, PoolSize: 9, MinIdleConns: 2, MaxRetries: 5}},
		{"negative min idle gets default", Config{MinIdleConns: -1}, Config{DialTimeout: def.DialTimeout, ReadTimeout: def.ReadTimeout, WriteTimeout: def.WriteTimeout, PoolSize: def.PoolSize, MinIdleConns: def.MinIdleConns, MaxRetries: def.MaxRetries}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.in
			got.applyDefaults()
			if got != tt.want {
				t.Errorf("got %+v, want %+v", got, tt.want)
			}
		})
	}
}

func TestConfig_SecretOmittedFromJSON(t *testing.T) {
	cfg := Config{
		Addr:     "127.0.0.1:6379",
		Password: "super_secret_redis_password",
	}
	b, err := json.Marshal(cfg)
	if err != nil {
		t.Fatalf("json.Marshal: %v", err)
	}
	out := string(b)
	if strings.Contains(out, "super_secret_redis_password") || strings.Contains(out, "password") {
		t.Errorf("expected secret password to be omitted from JSON, got %s", out)
	}
}
