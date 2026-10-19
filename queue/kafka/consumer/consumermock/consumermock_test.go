package consumermock

import (
	"context"
	"github.com/baby-hero-404/gocore/queue/kafka/consumer"
	"testing"
)

func TestMockConsumer_Subscribe(t *testing.T) {
	msg1 := consumer.Message{
		Topic:  "orders",
		Offset: 1,
		Key:    []byte("order_1"),
		Value:  []byte(`{"status":"created"}`),
	}
	msg2 := consumer.Message{
		Topic:  "orders",
		Offset: 2,
		Key:    []byte("order_2"),
		Value:  []byte(`{"status":"failed"}`),
	}

	m := NewConsumer(msg1, msg2)

	processed := make([]string, 0)
	err := m.Subscribe(context.Background(), func(ctx context.Context, msg consumer.Message) error {
		processed = append(processed, string(msg.Key))
		return nil
	})
	if err != nil {
		t.Fatalf("unexpected subscribe error: %v", err)
	}

	if len(processed) != 2 {
		t.Fatalf("expected 2 processed messages, got %d", len(processed))
	}
	if len(m.Committed) != 2 {
		t.Fatalf("expected 2 committed messages, got %d", len(m.Committed))
	}
}
