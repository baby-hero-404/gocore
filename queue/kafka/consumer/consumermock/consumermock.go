// Package consumermock provides test doubles for github.com/baby-hero-404/gocore/queue/kafka/consumer.
package consumermock

import (
	"context"
	"github.com/baby-hero-404/gocore/queue/kafka/consumer"
	"sync"
)

// Consumer provides an in-memory mock of the Consumer interface for testing.
type Consumer struct {
	mu        sync.Mutex
	Queue     []consumer.Message
	Committed []consumer.Message
	Closed    bool
	Err       error
}

// NewConsumer returns an initialized Consumer.
func NewConsumer(initialMessages ...consumer.Message) *Consumer {
	return &Consumer{
		Queue:     initialMessages,
		Committed: make([]consumer.Message, 0),
	}
}

func (m *Consumer) FetchMessage(ctx context.Context) (consumer.Message, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.Err != nil {
		return consumer.Message{}, m.Err
	}
	if len(m.Queue) == 0 {
		return consumer.Message{}, context.Canceled
	}
	msg := m.Queue[0]
	m.Queue = m.Queue[1:]
	return msg, nil
}

func (m *Consumer) CommitMessages(ctx context.Context, msgs ...consumer.Message) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.Err != nil {
		return m.Err
	}
	m.Committed = append(m.Committed, msgs...)
	return nil
}

func (m *Consumer) Subscribe(ctx context.Context, handler consumer.Handler) error {
	for {
		msg, err := m.FetchMessage(ctx)
		if err != nil {
			return nil
		}
		if err := handler(ctx, msg); err == nil {
			_ = m.CommitMessages(ctx, msg)
		}
	}
}

func (m *Consumer) Close() error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.Closed = true
	return m.Err
}
