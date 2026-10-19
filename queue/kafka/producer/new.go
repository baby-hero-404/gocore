package producer

import (
	"strings"

	"github.com/baby-hero-404/gocore/stderr"
	"github.com/segmentio/kafka-go"
)

type impl struct {
	writer *kafka.Writer
	topic  string
}

// New initializes a Kafka producer implementing the Producer interface.
func New(cfg Config, opts ...Option) (Producer, error) {
	if len(cfg.Brokers) == 0 {
		return nil, stderr.NewValidation("kafka producer configuration invalid").
			WithDetails(stderr.FieldError{Field: "brokers", Message: "brokers list cannot be empty", Tag: "required"})
	}
	if strings.TrimSpace(cfg.Topic) == "" {
		return nil, stderr.NewValidation("kafka producer configuration invalid").
			WithDetails(stderr.FieldError{Field: "topic", Message: "topic cannot be empty", Tag: "required"})
	}

	cfg.applyDefaults()

	po := &options{
		balancer: &kafka.LeastBytes{},
		async:    cfg.Async,
	}
	for _, opt := range opts {
		opt(po)
	}

	writer := &kafka.Writer{
		Addr:         kafka.TCP(cfg.Brokers...),
		Topic:        cfg.Topic,
		Balancer:     po.balancer,
		BatchSize:    cfg.BatchSize,
		BatchTimeout: cfg.BatchTimeout,
		RequiredAcks: kafka.RequiredAcks(cfg.RequiredAcks),
		Async:        po.async,
	}

	return &impl{
		writer: writer,
		topic:  cfg.Topic,
	}, nil
}
