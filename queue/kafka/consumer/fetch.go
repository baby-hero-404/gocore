package consumer

import (
	"context"
	"errors"
	"io"

	"github.com/baby-hero-404/gocore/stderr"
)

// FetchMessage reads the next message from Kafka without auto-committing the offset.
func (c *impl) FetchMessage(ctx context.Context) (Message, error) {
	raw, err := c.reader.FetchMessage(ctx)
	if err != nil {
		if errors.Is(err, io.EOF) || errors.Is(err, context.Canceled) {
			return Message{}, err
		}
		if ctx.Err() != nil {
			return Message{}, stderr.Wrap(stderr.CodeTimeout, ctx.Err(), "kafka fetch timed out or cancelled")
		}
		return Message{}, stderr.Wrap(stderr.CodeUnavailable, err, "failed to fetch message from kafka")
	}

	return fromKafkaMessage(raw), nil
}
