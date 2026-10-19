package consumer

import (
	"encoding/json"
	"time"

	"github.com/segmentio/kafka-go"
)

// Message encapsulates a received Kafka record with metadata and payload.
type Message struct {
	Topic     string            `json:"topic"`
	Partition int               `json:"partition"`
	Offset    int64             `json:"offset"`
	Key       []byte            `json:"key"`
	Value     []byte            `json:"value"`
	Headers   map[string]string `json:"headers"`
	Time      time.Time         `json:"time"`
	Raw       kafka.Message     `json:"-"`
}

// BindJSON unmarshals the message's value bytes into target.
func (m *Message) BindJSON(target any) error {
	return json.Unmarshal(m.Value, target)
}

// fromKafkaMessage converts a segmentio/kafka-go message to our decoupled Message.
func fromKafkaMessage(msg kafka.Message) Message {
	headers := make(map[string]string, len(msg.Headers))
	for _, h := range msg.Headers {
		headers[h.Key] = string(h.Value)
	}

	return Message{
		Topic:     msg.Topic,
		Partition: msg.Partition,
		Offset:    msg.Offset,
		Key:       msg.Key,
		Value:     msg.Value,
		Headers:   headers,
		Time:      msg.Time,
		Raw:       msg,
	}
}
