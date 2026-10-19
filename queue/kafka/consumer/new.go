package consumer

import (
	"strings"

	"github.com/baby-hero-404/gocore/stderr"
	"github.com/segmentio/kafka-go"
)

type impl struct {
	reader *kafka.Reader
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
