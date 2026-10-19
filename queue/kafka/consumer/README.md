# Kafka Consumer Package

Production-ready Kafka consumer group reader for Go microservices powered by `segmentio/kafka-go`.

## Features
- **Decoupled Message Struct**: Exposes `consumer.Message` with JSON unmarshaling helper.
- **Handler-Driven Worker**: `Subscribe(ctx, handler)` auto-commits offsets on success (`return nil`) and skips commit on error for retry.
- **Manual Offset Control**: `FetchMessage` and `CommitMessages` available for advanced manual commit patterns.
- **Pre-built `consumermock.Consumer`**: Fast in-memory mock for testing business event handlers without Kafka containers.

## Usage

```go
package main

import (
	"context"
	"log"

	"github.com/baby-hero-404/gocore/queue/kafka/consumer"
)

func main() {
	c, err := consumer.New(consumer.Config{
		Brokers: []string{"localhost:9092"},
		Topic:   "user-events",
		GroupID: "notification-service",
	})
	if err != nil {
		log.Fatalf("failed to create consumer: %v", err)
	}
	defer c.Close()

	log.Println("Listening for events...")
	err = c.Subscribe(context.Background(), func(ctx context.Context, msg consumer.Message) error {
		log.Printf("Received message [key=%s, offset=%d]: %s\n", string(msg.Key), msg.Offset, string(msg.Value))
		return nil
	})
	if err != nil {
		log.Fatalf("consumer loop error: %v", err)
	}
}
```
