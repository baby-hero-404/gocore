# Kafka Producer Package

High-throughput, durable Kafka producer wrapper for Go microservices powered by `segmentio/kafka-go`.

## Features
- Strong durability: Configurable ACKs, default `-1` (WaitForAll replicas).
- Fluent JSON marshalling via `PublishJSON(ctx, key, data)`.
- Batching controls (`BatchSize`, `BatchTimeout`).
- Pre-built `producermock.Producer` for testing downstream business handlers.

## Usage

```go
package main

import (
	"context"
	"log"

	"github.com/baby-hero-404/gocore/queue/kafka/producer"
)

func main() {
	p, err := producer.New(producer.Config{
		Brokers: []string{"localhost:9092"},
		Topic:   "user-events",
	})
	if err != nil {
		log.Fatalf("failed to create producer: %v", err)
	}
	defer p.Close()

	err = p.PublishJSON(context.Background(), "usr_1001", map[string]any{
		"event": "user_created",
		"id":    "usr_1001",
	})
	if err != nil {
		log.Fatalf("failed to publish: %v", err)
	}
}
```
