package consumer

import "context"

// Consumer defines the standard interface for receiving messages from Kafka.
type Consumer interface {
	// FetchMessage reads the next message from Kafka without committing the offset.
	FetchMessage(ctx context.Context) (Message, error)

	// CommitMessages manually commits offsets for the given messages.
	CommitMessages(ctx context.Context, msgs ...Message) error

	// Subscribe runs an automatic consumption loop invoking handler for each message.
	// Offsets are committed automatically if handler returns nil.
	Subscribe(ctx context.Context, handler Handler) error

	// Close safely closes the consumer reader connection.
	Close() error
}
