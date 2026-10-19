package consumer

import (
	"context"

	"github.com/baby-hero-404/gocore/stderr"
	"github.com/segmentio/kafka-go"
)

// CommitMessages manually commits offsets for processed messages.
func (c *impl) CommitMessages(ctx context.Context, msgs ...Message) error {
	rawMsgs := make([]kafka.Message, len(msgs))
	for i, m := range msgs {
		rawMsgs[i] = m.Raw
	}

	if err := c.reader.CommitMessages(ctx, rawMsgs...); err != nil {
		if ctx.Err() != nil {
			return stderr.Wrap(stderr.CodeTimeout, ctx.Err(), "kafka commit timed out or cancelled")
		}
		return stderr.Wrap(stderr.CodeUnavailable, err, "failed to commit kafka messages")
	}

	return nil
}
