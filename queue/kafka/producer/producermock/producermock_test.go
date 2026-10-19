package producermock

import (
	"context"
	"testing"
)

func TestMockProducer(t *testing.T) {
	m := NewProducer()
	ctx := context.Background()

	payload := map[string]string{"user_id": "usr_123", "action": "login"}
	if err := m.PublishJSON(ctx, "usr_123", payload); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(m.Published) != 1 {
		t.Fatalf("expected 1 message published, got %d", len(m.Published))
	}
	if string(m.Published[0].Key) != "usr_123" {
		t.Errorf("expected key 'usr_123', got %s", string(m.Published[0].Key))
	}

	if err := m.Close(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !m.Closed {
		t.Error("expected mock to be closed")
	}
}
