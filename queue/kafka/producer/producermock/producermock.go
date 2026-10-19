// Package producermock provides test doubles for github.com/baby-hero-404/gocore/queue/kafka/producer.
package producermock

import (
	"context"
	"encoding/json"
	"sync"
)

// Producer provides an in-memory mock of the Producer interface for unit testing.
type Producer struct {
	mu        sync.Mutex
	Published []Message
	Closed    bool
	Err       error
}

// Message represents a captured message in Producer.
type Message struct {
	Key   []byte
	Value []byte
}

// NewProducer returns an initialized Producer.
func NewProducer() *Producer {
	return &Producer{
		Published: make([]Message, 0),
	}
}

func (m *Producer) Publish(ctx context.Context, key []byte, value []byte) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.Err != nil {
		return m.Err
	}
	m.Published = append(m.Published, Message{Key: key, Value: value})
	return nil
}

func (m *Producer) PublishJSON(ctx context.Context, key string, data any) error {
	bytes, err := json.Marshal(data)
	if err != nil {
		return err
	}
	return m.Publish(ctx, []byte(key), bytes)
}

func (m *Producer) Close() error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.Closed = true
	return m.Err
}
