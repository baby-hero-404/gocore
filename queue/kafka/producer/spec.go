package producer

import "context"

// Producer defines the standard interface for publishing messages to Kafka.
type Producer interface {
	// Publish sends raw bytes to Kafka with an optional key.
	Publish(ctx context.Context, key []byte, value []byte) error

	// PublishJSON marshals data into JSON and sends to Kafka with string key.
	PublishJSON(ctx context.Context, key string, data any) error

	// Close safely shuts down the producer and flushes internal buffers.
	Close() error
}
