package consumer

import "context"

// Handler is a callback function invoked for each consumed Kafka message.
// If Handler returns nil, the consumer automatically commits the message offset.
// If Handler returns an error, the message is not committed (allowing retry or DLQ handling).
type Handler func(ctx context.Context, msg Message) error
