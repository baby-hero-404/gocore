package producer

import (
	"context"
	"encoding/json"

	"github.com/baby-hero-404/gocore/stderr"
	"github.com/segmentio/kafka-go"
)

// Publish writes raw bytes to Kafka.
func (p *impl) Publish(ctx context.Context, key []byte, value []byte) error {
	msg := kafka.Message{
		Key:   key,
		Value: value,
	}

	if err := p.writer.WriteMessages(ctx, msg); err != nil {
		if ctx.Err() != nil {
			return stderr.Wrap(stderr.CodeTimeout, ctx.Err(), "kafka publish timed out or cancelled")
		}
		return stderr.Wrap(stderr.CodeUnavailable, err, "failed to publish message to kafka")
	}

	return nil
}

// PublishJSON serializes data to JSON and publishes with a string key.
func (p *impl) PublishJSON(ctx context.Context, key string, data any) error {
	bytes, err := json.Marshal(data)
	if err != nil {
		return stderr.Wrap(stderr.CodeBadRequest, err, "failed to marshal kafka payload into json")
	}

	return p.Publish(ctx, []byte(key), bytes)
}
