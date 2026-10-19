# Queue Foundation Wrappers

This directory houses modular message queue and streaming platform connectors for Go microservices.

## Architecture

Following domain separation best practices, message brokers are isolated to prevent dependency bloat and avoid leaky abstractions:

```text
queue/
├── kafka/
│   ├── producer/   # Dedicated Kafka producer (JSON publishing, batching, acks)
│   └── consumer/   # Dedicated Kafka consumer group (handler-driven, auto/manual commit)
│
├── rabbitmq/       # (Planned) AMQP task queues
└── redis/          # (Planned) Redis Streams / Asynq task queues
```

## Quick Reference

### Kafka Producer
```go
import "github.com/baby-hero-404/gocore/queue/kafka/producer"

p, err := producer.New(producer.Config{
    Brokers: []string{"localhost:9092"},
    Topic:   "user-events",
})
p.PublishJSON(ctx, "user_1", eventData)
```

### Kafka Consumer
```go
import "github.com/baby-hero-404/gocore/queue/kafka/consumer"

c, err := consumer.New(consumer.Config{
    Brokers: []string{"localhost:9092"},
    Topic:   "user-events",
    GroupID: "analytics-worker",
})
c.Subscribe(ctx, func(ctx context.Context, msg consumer.Message) error {
    // Process message
    return nil // Nil automatically commits offset
})
```
