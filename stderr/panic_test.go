package stderr

import (
	"errors"
	"testing"
)

func TestFromPanic(t *testing.T) {
	sentinel := errors.New("boom")
	tests := []struct {
		name      string
		recovered any
		wantMsg   string
		wantIs    error
	}{
		{"error value keeps identity", sentinel, "boom", sentinel},
		{"string value", "bad state", "bad state", nil},
		{"arbitrary value", 42, "42", nil},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := FromPanic(tt.recovered)
			if got.Code != CodeInternal {
				t.Errorf("code = %v, want CodeInternal", got.Code)
			}
			if got.Message != "internal server error" {
				t.Errorf("message must not leak the panic value, got %q", got.Message)
			}
			if got.Err == nil || got.Err.Error() != tt.wantMsg {
				t.Errorf("cause = %v, want %q", got.Err, tt.wantMsg)
			}
			if tt.wantIs != nil && !errors.Is(got, tt.wantIs) {
				t.Errorf("errors.Is(%v) = false", tt.wantIs)
			}
		})
	}
}
