package database

import (
	"bytes"
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"gorm.io/gorm"
)

func TestResolveLogger_IgnoresRecordNotFoundAndHasNoColour(t *testing.T) {
	var buf bytes.Buffer
	l := resolveLoggerTo(&buf, "warn")
	sql := func() (string, int64) { return "SELECT 1", 0 }

	l.Trace(context.Background(), time.Now(), sql, gorm.ErrRecordNotFound)
	if buf.Len() != 0 {
		t.Fatalf("record not found must not be logged: %q", buf.String())
	}
	l.Trace(context.Background(), time.Now(), sql, errors.New("boom"))
	out := buf.String()
	if !strings.Contains(out, "boom") {
		t.Errorf("real errors must still be logged: %q", out)
	}
	if strings.Contains(out, "\x1b[") {
		t.Errorf("log output must not contain colour escape codes: %q", out)
	}
	// silent level logs nothing at all
	buf.Reset()
	resolveLoggerTo(&buf, "silent").Trace(context.Background(), time.Now(), sql, errors.New("boom"))
	if buf.Len() != 0 {
		t.Errorf("silent logger wrote %q", buf.String())
	}
}
