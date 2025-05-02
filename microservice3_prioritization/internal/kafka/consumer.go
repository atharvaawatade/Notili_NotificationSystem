package kafka

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/appointy/notli/microservice3_prioritization/internal/model"
	"github.com/segmentio/kafka-go"
)

// MessageConsumer is responsible for consuming messages from Kafka
type MessageConsumer struct {
	reader        *kafka.Reader
	processCh     chan *model.NotificationMessage
	stopCh        chan struct{}
	consumerGroup string
	topic         string
}

// NewConsumer creates a new Kafka consumer
func NewConsumer(bootstrapServers, consumerGroup, topic string) *MessageConsumer {
	reader := kafka.NewReader(kafka.ReaderConfig{
		Brokers:       []string{bootstrapServers},
		GroupID:       consumerGroup,
		Topic:         topic,
		MinBytes:      10e3,    // 10KB minimum batch size
		MaxBytes:      10e6,    // 10MB maximum batch size
		QueueCapacity: 1000,    // Internal queue capacity
		MaxWait:       100 * time.Millisecond,
		ReadBackoffMin: time.Millisecond * 50,
		ReadBackoffMax: time.Second * 5,
	})

	return &MessageConsumer{
		reader:        reader,
		processCh:     make(chan *model.NotificationMessage, 1000),
		stopCh:        make(chan struct{}),
		consumerGroup: consumerGroup,
		topic:         topic,
	}
}

// Start begins consuming messages in a goroutine
func (c *MessageConsumer) Start(ctx context.Context) error {
	go func() {
		for {
			select {
			case <-c.stopCh:
				return
			case <-ctx.Done():
				return
			default:
				message, err := c.reader.FetchMessage(ctx)
				if err != nil {
					fmt.Printf("Error fetching message: %v\n", err)
					time.Sleep(1 * time.Second) // Back off on error
					continue
				}

				var notification model.NotificationMessage
				if err := json.Unmarshal(message.Value, &notification); err != nil {
					fmt.Printf("Error unmarshalling message: %v\n", err)
					// Still commit this message as we can't process it
					if err := c.reader.CommitMessages(ctx, message); err != nil {
						fmt.Printf("Error committing message: %v\n", err)
					}
					continue
				}

				// Send to processing channel
				c.processCh <- &notification

				// Commit the message
				if err := c.reader.CommitMessages(ctx, message); err != nil {
					fmt.Printf("Error committing message: %v\n", err)
					continue
				}
			}
		}
	}()

	return nil
}

// GetProcessChannel returns the channel for processing messages
func (c *MessageConsumer) GetProcessChannel() <-chan *model.NotificationMessage {
	return c.processCh
}

// Stop stops the consumer
func (c *MessageConsumer) Stop() error {
	close(c.stopCh)
	return c.reader.Close()
}
