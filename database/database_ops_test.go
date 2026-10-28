package database

import (
	"bytes"
	"context"
	"database/sql"
	"database/sql/driver"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/baby-hero-404/gocore/stderr"
	"gorm.io/driver/mysql"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	gormlogger "gorm.io/gorm/logger"
)

var memSeq atomic.Int64

func newMemDB(t *testing.T, opts ...Option) Database {
	t.Helper()
	dsn := fmt.Sprintf("file:dbops%d?mode=memory&cache=shared", memSeq.Add(1))
	db, err := Open(context.Background(), sqlite.Open(dsn), Config{DSN: dsn, LogLevel: "silent"}, opts...)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	return db
}

func codeOf(t *testing.T, err error) stderr.Code {
	t.Helper()
	var se *stderr.Error
	if !errors.As(err, &se) {
		t.Fatalf("expected *stderr.Error, got %T (%v)", err, err)
	}
	return se.Code
}

func TestDatabase_SessionsAndLifecycle(t *testing.T) {
	db := newMemDB(t)
	ctx := context.Background()

	if db.DB() == nil {
		t.Fatal("DB() must expose the gorm handle")
	}
	if got := db.WithContext(ctx); got.Statement.Context != ctx {
		t.Error("WithContext must bind ctx")
	}
	ndb := db.NewDb(ctx)
	if !ndb.SkipDefaultTransaction || ndb.CreateBatchSize != 1000 || !ndb.QueryFields {
		t.Errorf("NewDb session flags not applied: skipTx=%v batch=%d queryFields=%v", ndb.SkipDefaultTransaction, ndb.CreateBatchSize, ndb.QueryFields)
	}
	if err := db.Ping(ctx); err != nil {
		t.Fatalf("ping: %v", err)
	}
	if err := db.Close(); err != nil {
		t.Fatalf("close: %v", err)
	}
	if got := codeOf(t, db.Ping(ctx)); got != stderr.CodeUnavailable {
		t.Errorf("ping after close code = %v, want CodeUnavailable", got)
	}
}

func TestPackageHelpers_NilHandle(t *testing.T) {
	if err := Close(nil); err != nil {
		t.Errorf("Close(nil) = %v", err)
	}
	if got := codeOf(t, Ping(context.Background(), nil)); got != stderr.CodeBadRequest {
		t.Errorf("Ping(nil) code = %v, want CodeBadRequest", got)
	}
}

func TestOptions_CustomLoggerAndGormConfig(t *testing.T) {
	custom := gormlogger.Default.LogMode(gormlogger.Silent)
	withLogger := newMemDB(t, WithLogger(custom))
	if withLogger.DB().Logger != custom {
		t.Error("WithLogger must be used as the gorm logger")
	}

	cfg := &gorm.Config{SkipDefaultTransaction: true, Logger: custom}
	withCfg := newMemDB(t, WithGormConfig(cfg))
	if !withCfg.DB().SkipDefaultTransaction {
		t.Error("WithGormConfig must be used as the gorm config")
	}
}

func TestResolveLoggerTo_Levels(t *testing.T) {
	ctx := context.Background()
	tests := []struct {
		level                       string
		wantError, wantWarn, wantIn bool
	}{
		{"silent", false, false, false},
		{"error", true, false, false},
		{"", true, false, false}, // unknown -> error level
		{"warn", true, true, false},
		{"INFO", true, true, true},
	}
	for _, tt := range tests {
		t.Run("level "+tt.level, func(t *testing.T) {
			var buf bytes.Buffer
			l := resolveLoggerTo(&buf, tt.level)

			for _, c := range []struct {
				name string
				log  func()
				want bool
			}{
				{"error", func() { l.Error(ctx, "e-marker") }, tt.wantError},
				{"warn", func() { l.Warn(ctx, "w-marker") }, tt.wantWarn},
				{"info", func() { l.Info(ctx, "i-marker") }, tt.wantIn},
			} {
				buf.Reset()
				c.log()
				if got := buf.Len() > 0; got != c.want {
					t.Errorf("%s output present = %v, want %v", c.name, got, c.want)
				}
			}
		})
	}
}

// trackingConnector fails every connection and records whether its *sql.DB was closed.
type trackingConnector struct {
	opens  atomic.Int32
	closed atomic.Bool
}

func (c *trackingConnector) Connect(context.Context) (driver.Conn, error) {
	c.opens.Add(1)
	return nil, errors.New("connection refused")
}
func (c *trackingConnector) Driver() driver.Driver { return failingDriver{} }
func (c *trackingConnector) Close() error          { c.closed.Store(true); return nil }

type failingDriver struct{}

func (failingDriver) Open(string) (driver.Conn, error) { return nil, errors.New("connection refused") }

func TestConnect_FailedAttemptsReleaseTheirPool(t *testing.T) {
	conn := &trackingConnector{}
	sqlDB := sql.OpenDB(conn)

	// gorm.Open itself closes the pool when its automatic ping fails, so disable that ping to
	// exercise connect's own PingContext failure path.
	dialector := mysql.New(mysql.Config{Conn: sqlDB, SkipInitializeWithVersion: true})
	cfg := Config{DSN: "unused", RetryAttempts: 1, RetryBackoff: time.Millisecond, LogLevel: "silent"}
	db, err := Connect(context.Background(), dialector, cfg, WithGormConfig(&gorm.Config{DisableAutomaticPing: true}))
	if err == nil || db != nil {
		t.Fatalf("expected connection failure, got db=%v err=%v", db, err)
	}
	if got := codeOf(t, err); got != stderr.CodeUnavailable {
		t.Errorf("code = %v, want CodeUnavailable", got)
	}
	if !conn.closed.Load() {
		t.Error("the pool of a failed attempt must be closed, not leaked")
	}
}

type timeoutNetErr struct{}

func (timeoutNetErr) Error() string   { return "i/o timeout" }
func (timeoutNetErr) Timeout() bool   { return true }
func (timeoutNetErr) Temporary() bool { return true }

func TestWrapError(t *testing.T) {
	tests := []struct {
		name string
		err  error
		want stderr.Code // 0 = nil result
	}{
		{"nil", nil, 0},
		{"record not found", gorm.ErrRecordNotFound, stderr.CodeNotFound},
		{"duplicated key", gorm.ErrDuplicatedKey, stderr.CodeConflict},
		{"invalid transaction", gorm.ErrInvalidTransaction, stderr.CodeInternal},
		{"deadline exceeded", fmt.Errorf("query: %w", context.DeadlineExceeded), stderr.CodeTimeout},
		{"network timeout", timeoutNetErr{}, stderr.CodeTimeout},
		{"context canceled is a plain database error", context.Canceled, stderr.CodeDatabaseError},
		{"anything else", errors.New("boom"), stderr.CodeDatabaseError},
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

func TestConfig_SecretOmittedFromJSON(t *testing.T) {
	cfg := Config{
		Driver: "postgres",
		DSN:    "postgres://user:super_secret_pw@localhost:5432/db",
	}
	b, err := json.Marshal(cfg)
	if err != nil {
		t.Fatalf("json.Marshal: %v", err)
	}
	out := string(b)
	if strings.Contains(out, "super_secret_pw") || strings.Contains(out, "dsn") {
		t.Errorf("expected secret DSN to be omitted from JSON, got %s", out)
	}
}

func asStdErr(t *testing.T, err error) *stderr.Error {
	t.Helper()
	var se *stderr.Error
	if !errors.As(err, &se) {
		t.Fatalf("expected *stderr.Error, got %T (%v)", err, err)
	}
	return se
}
