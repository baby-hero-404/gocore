package stdlog

import (
	"bytes"
	"context"
	"log/slog"
	"strings"
	"testing"
)

func TestDefaultLogger_PackageHelpers(t *testing.T) {
	var buf bytes.Buffer
	prev := Get()
	SetDefault(New(WithOutput(&buf), WithLevel(slog.LevelDebug)))
	t.Cleanup(func() { SetDefault(prev) })

	ctx := context.WithValue(context.Background(), RequestIDCtxKey, "rid-7")
	tests := []struct {
		name string
		log  func()
		want string
	}{
		{"debug", func() { Debug("d-msg") }, "d-msg"},
		{"info", func() { Info("i-msg") }, "i-msg"},
		{"warn", func() { Warn("w-msg") }, "w-msg"},
		{"error", func() { Error("e-msg") }, "e-msg"},
		{"with", func() { With("k", "v").Info("with-msg") }, "with-msg"},
		{"with context carries request id", func() { WithContext(ctx).Info("ctx-msg") }, "rid-7"},
		{"log at level", func() { Get().Log(ctx, slog.LevelInfo, "log-msg") }, "log-msg"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			buf.Reset()
			tt.log()
			if !strings.Contains(buf.String(), tt.want) {
				t.Errorf("output %q does not contain %q", buf.String(), tt.want)
			}
		})
	}
}

func TestSetDefault_NilIgnoredAndGetStable(t *testing.T) {
	before := Get()
	SetDefault(nil)
	if Get() != before {
		t.Error("SetDefault(nil) must not replace the default logger")
	}
}

func TestNew_HandlerOptions(t *testing.T) {
	var buf bytes.Buffer
	custom := slog.NewJSONHandler(&buf, nil)
	New(WithHandler(custom), WithAddSource(true)).Info("via custom handler")
	if !strings.Contains(buf.String(), "via custom handler") {
		t.Errorf("custom handler not used: %q", buf.String())
	}
}
