package consumer

import (
	"context"
	"strings"

	"github.com/baby-hero-404/gocore/stderr"
	"github.com/segmentio/kafka-go"
)

// reader is the subset of *kafka.Reader the consumer uses; it is the test seam for a fake broker.
type reader interface {
	FetchMessage(ctx context.Context) (kafka.Message, error)
	CommitMessages(ctx context.Context, msgs ...kafka.Message) error
	Close() error
}

type impl struct {
	reader reader
	topic  string
}

// New initializes a Kafka consumer group reader implementing the Consumer interface.
func New(cfg Config, opts ...Option) (Consumer, error) {
	if len(cfg.Brokers) == 0 {
		return nil, stderr.NewValidation("kafka consumer configuration invalid").
			WithDetails(stderr.FieldError{Field: "brokers", Message: "brokers list cannot be empty", Tag: "required"})
	}
	if strings.TrimSpace(cfg.Topic) == "" {
		return nil, stderr.NewValidation("kafka consumer configuration invalid").
			WithDetails(stderr.FieldError{Field: "topic", Message: "topic cannot be empty", Tag: "required"})
	}
	if strings.TrimSpace(cfg.GroupID) == "" {
		return nil, stderr.NewValidation("kafka consumer configuration invalid").
			WithDetails(stderr.FieldError{Field: "group_id", Message: "group_id cannot be empty", Tag: "required"})
	}

	cfg.applyDefaults()

	co := &options{
		startOffset: cfg.StartOffset,
	}
	for _, opt := range opts {
		opt(co)
	}

	reader := kafka.NewReader(kafka.ReaderConfig{
		Brokers:     cfg.Brokers,
		Topic:       cfg.Topic,
		GroupID:     cfg.GroupID,
		MinBytes:    cfg.MinBytes,
		MaxBytes:    cfg.MaxBytes,
		MaxWait:     cfg.MaxWait,
		StartOffset: co.startOffset,
	})

	return &impl{
		reader: reader,
		topic:  cfg.Topic,
	}, nil
}
