package consumer

import (
	"context"
	"errors"
	"io"

	"github.com/baby-hero-404/gocore/stderr"
	"github.com/baby-hero-404/gocore/stdlog"
)

// Subscribe starts an automated consume loop invoking handler for each message.
// Auto-commits offset when handler returns nil.
func (c *impl) Subscribe(ctx context.Context, handler Handler) error {
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		raw, err := c.reader.FetchMessage(ctx)
		if err != nil {
			if errors.Is(err, io.EOF) || errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) || ctx.Err() != nil {
				return nil
			}
			return stderr.Wrap(stderr.CodeUnavailable, err, "kafka subscriber fetch failed")
		}

		msg := fromKafkaMessage(raw)
		if err := handler(ctx, msg); err != nil {
			stdlog.WithContext(ctx).Error("kafka message handler failed",
				"topic", msg.Topic,
				"partition", msg.Partition,
				"offset", msg.Offset,
				"error", err.Error(),
			)
			// Do not commit on error, allowing downstream retry
			continue
		}

		if err := c.reader.CommitMessages(ctx, raw); err != nil {
			stdlog.WithContext(ctx).Error("kafka auto-commit failed",
				"topic", msg.Topic,
				"offset", msg.Offset,
				"error", err.Error(),
			)
		}
	}
}
