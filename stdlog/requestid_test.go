package stdlog

import (
	"context"
	"testing"
)

func TestRequestIDContextRoundTrip(t *testing.T) {
	tests := []struct {
		name string
		ctx  context.Context
		want string
	}{
		{"set", ContextWithRequestID(context.Background(), "rid-1"), "rid-1"},
		{"unset", context.Background(), ""},
		{"overwritten", ContextWithRequestID(ContextWithRequestID(context.Background(), "a"), "b"), "b"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := RequestIDFromContext(tt.ctx); got != tt.want {
				t.Errorf("got %q, want %q", got, tt.want)
			}
		})
	}
}
