package producer

import (
	"errors"
	"testing"
	"time"

	"github.com/baby-hero-404/gocore/stderr"
)

func TestNewProducer_Validation(t *testing.T) {
	tests := []struct {
		name        string
		cfg         Config
		expectError bool
	}{
		{
			name: "missing brokers",
			cfg: Config{
				Topic: "test-topic",
			},
			expectError: true,
		},
		{
			name: "missing topic",
			cfg: Config{
				Brokers: []string{"localhost:9092"},
			},
			expectError: true,
		},
		{
			name: "valid config",
			cfg: Config{
				Brokers:      []string{"localhost:9092"},
				Topic:        "user-events",
				BatchSize:    50,
				BatchTimeout: 5 * time.Millisecond,
			},
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p, err := New(tt.cfg)
			if tt.expectError {
				if err == nil {
					t.Fatalf("expected error, got nil")
				}
				var stdErr *stderr.Error
				if !errors.As(err, &stdErr) {
					t.Fatalf("expected stderr.Error, got %T", err)
				}
				if stdErr.Code != stderr.CodeValidationFailed {
					t.Errorf("expected CodeValidationFailed, got %v", stdErr.Code)
				}
			} else {
				if err != nil {
					t.Fatalf("unexpected error: %v", err)
				}
				if p == nil {
					t.Fatal("expected producer instance, got nil")
				}
			}
		})
	}
}
