package kafka

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/appointy/notli/microservice3_prioritization/internal/model"
	"github.com/segmentio/kafka-go"
)

// MessageProducer is responsible for producing messages to Kafka
type MessageProducer struct {
	writer *kafka.Writer
	topic  string
}

// NewProducer creates a new Kafka producer
func NewProducer(bootstrapServers, topic string) *MessageProducer {
	writer := kafka.NewWriter(kafka.WriterConfig{
		Brokers:      []string{bootstrapServers},
		Topic:        topic,
		Balancer:     &kafka.LeastBytes{},
		BatchSize:    100,                // Maximum batch size
		BatchTimeout: 10 * time.Millisecond, // Maximum time to wait for a batch
		Async:        true,               // Use async mode for higher throughput
		RequiredAcks: 1,                 // Require ack from the leader only
	})

	return &MessageProducer{
		writer: writer,
		topic:  topic,
	}
}

// Produce sends a notification message to Kafka with priority information
func (p *MessageProducer) Produce(ctx context.Context, message *model.NotificationMessage) error {
	value, err := json.Marshal(message)
	if err != nil {
		return fmt.Errorf("error marshalling message: %w", err)
	}

	// Create message headers for priority information
	headers := []kafka.Header{
		{
			Key:   "priority",
			Value: []byte(message.Priority),
		},
		{
			Key:   "message_type",
			Value: []byte(message.MessageType),
		},
		{
			Key:   "channel",
			Value: []byte(message.Channel),
		},
		{
			Key:   "priority_score",
			Value: []byte(fmt.Sprintf("%d", message.PriorityScore)),
		},
	}

	err = p.writer.WriteMessages(ctx, kafka.Message{
		Key:     []byte(message.IdempotencyKey),
		Value:   value,
		Headers: headers,
	})

	if err != nil {
		return fmt.Errorf("error writing message to Kafka: %w", err)
	}

	return nil
}

// Close closes the producer
func (p *MessageProducer) Close() error {
	return p.writer.Close()
}
