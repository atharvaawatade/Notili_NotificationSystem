package kafka

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"time"

	"github.com/segmentio/kafka-go"
	"github.com/appointy/notli/microservice4_delivery/internal/config"
	"github.com/appointy/notli/microservice4_delivery/internal/model"
)

// KafkaReader interface enables mocking in tests
type KafkaReader interface {
	FetchMessage(ctx context.Context) (kafka.Message, error)
	CommitMessages(ctx context.Context, msgs ...kafka.Message) error
	Close() error
}

// Consumer reads messages from Kafka
type Consumer struct {
	reader KafkaReader
	topic  string
}

// NewConsumer creates a new Kafka consumer
func NewConsumer(bootstrapServers, groupID, topic string, cfg *config.Config) *Consumer {
	// Use configured parameters if available, otherwise fall back to defaults
	// Configure Kafka reader with longer timeouts to prevent connection issues
	
	// Use much longer timeouts and more resilient settings to prevent timeout issues
	reader := kafka.NewReader(kafka.ReaderConfig{
		Brokers:           []string{bootstrapServers},
		GroupID:           groupID,
		Topic:             topic,
		MinBytes:          1,     // Accept smaller messages
		MaxBytes:          10e6,  // 10MB maximum batch size
		MaxWait:           10 * time.Second, // Wait longer for messages
		ReadLagInterval:   -1,    // Disable lag reporting
		SessionTimeout:    60 * time.Second, // Much longer session timeout
		HeartbeatInterval: 10 * time.Second, // Less frequent heartbeats
		RebalanceTimeout:  120 * time.Second, // Allow more time for rebalancing
		RetentionTime:     24 * time.Hour, // Retain messages in the group for 24 hours
		ReadBackoffMin:    100 * time.Millisecond, // Slightly higher min backoff
		ReadBackoffMax:    30 * time.Second, // Much longer max backoff
		Logger:            kafka.LoggerFunc(log.Printf),
		ErrorLogger:       kafka.LoggerFunc(log.Printf),
		// Always start from the earliest offset to avoid missing messages
		StartOffset:       kafka.FirstOffset,
	})
	
	log.Printf("Created Kafka consumer for topic '%s' with group '%s'", topic, groupID)
	
	return NewConsumerWithReader(reader, topic)
}

// getStartOffset calculates the initial offset based on the mode
// Note: This function is reserved for future use when implementing different offset strategies
// nolint:unused
func getStartOffset(strategy string) int64 {
	switch strategy {
	case "earliest":
		return kafka.FirstOffset
	case "latest":
		return kafka.LastOffset
	default:
		return kafka.FirstOffset // Default to earliest as it's safer
	}
}

// NewConsumerWithReader creates a consumer with a custom reader (for testing)
func NewConsumerWithReader(reader KafkaReader, topic string) *Consumer {
	return &Consumer{
		reader: reader,
		topic:  topic,
	}
}

// Consume reads messages from Kafka and passes them to the handler function
func (c *Consumer) Consume(ctx context.Context, handler func(context.Context, *model.KafkaMessage) error) error {
	log.Printf("Starting Kafka consumer for topic %s", c.topic)

	// Add a recovery mechanism to prevent crashes
	defer func() {
		if r := recover(); r != nil {
			log.Printf("RECOVERED from panic in Kafka consumer: %v", r)
		}
	}()

	// Use a more resilient loop with better error handling
	for {
		select {
		case <-ctx.Done():
			log.Printf("Stopping Kafka consumer for topic %s: context canceled", c.topic)
			return nil
		default:
			// Create a timeout context for each fetch
			fetchCtx, cancel := context.WithTimeout(ctx, 60*time.Second)
			
			// Fetch message with timeout protection
			msg, err := c.reader.FetchMessage(fetchCtx)
			cancel() // Always cancel the context to prevent leaks
			
			if err != nil {
				if ctx.Err() != nil {
					// Parent context was canceled
					return nil
				}

				// More detailed error logging
				log.Printf("Error fetching message from Kafka topic %s: %v (type: %T)", c.topic, err, err)
				
				// More intelligent backoff
				log.Printf("Waiting 5 seconds before retrying fetch from Kafka")
				time.Sleep(5 * time.Second)
				continue
			}

			// More detailed logging of received message
			log.Printf("Received Kafka message from topic %s, partition: %d, offset: %d", 
				c.topic, msg.Partition, msg.Offset)
			
			// Log raw message for debugging
			rawMsg := string(msg.Value)
			if len(rawMsg) > 1000 {
				rawMsg = rawMsg[:1000] + "..." // Truncate if too long
			}
			log.Printf("Raw Kafka message: %s", rawMsg)
			
			// Parse message with better error handling
			var kafkaMsg model.KafkaMessage
			if err := json.Unmarshal(msg.Value, &kafkaMsg); err != nil {
				log.Printf("ERROR unmarshaling Kafka message: %v", err)
				log.Printf("Attempting to fix the message format...")
				
				// Try to extract the message by using a more permissive approach with map
				var rawMap map[string]interface{}
				if jsonErr := json.Unmarshal(msg.Value, &rawMap); jsonErr == nil {
					// Successfully parsed as map, let's construct a valid KafkaMessage manually
					log.Printf("Successfully parsed raw message as map, reconstructing...")
					
					// Extract key fields
					kafkaMsg.IdempotencyKey, _ = rawMap["idempotency_key"].(string)
					kafkaMsg.Recipient, _ = rawMap["recipient"].(string)
					kafkaMsg.TemplateID, _ = rawMap["template_id"].(string)
					kafkaMsg.MessageType, _ = rawMap["message_type"].(string)
					kafkaMsg.Channel, _ = rawMap["channel"].(string)
					kafkaMsg.Priority, _ = rawMap["priority"].(string)
					kafkaMsg.UserID, _ = rawMap["user_id"].(string)
					kafkaMsg.UserTier, _ = rawMap["user_tier"].(string)
					
					// Extract priority score
					if priorityScore, ok := rawMap["priority_score"].(float64); ok {
						kafkaMsg.PriorityScore = int(priorityScore)
					}
					
					// Handle channel data
					if channelData, ok := rawMap["channel_data"].(map[string]interface{}); ok {
						kafkaMsg.ChannelData = channelData
					}
					
					log.Printf("Message recovery successful - proceeding with reconstructed message")
				} else {
					log.Printf("Could not recover message: %v", jsonErr)
					// Commit the bad message so we don't get stuck
					if err := c.reader.CommitMessages(ctx, msg); err != nil {
						log.Printf("ERROR committing bad Kafka message: %v", err)
					} else {
						log.Printf("Successfully committed malformed message to skip it")
					}
					continue
				}
			}

			// Process message with more detailed logging
			log.Printf("Processing Kafka message: IdempotencyKey=%s, Channel=%s, Recipient=%s", 
				kafkaMsg.IdempotencyKey, kafkaMsg.Channel, kafkaMsg.Recipient)
			
			// Use a longer timeout for processing
			handlerCtx, cancel := context.WithTimeout(ctx, 60*time.Second)
			errorCh := make(chan error, 1)
			
			// Run handler in a goroutine with panic recovery
			go func() {
				defer func() {
					if r := recover(); r != nil {
						log.Printf("RECOVERED from panic in message handler: %v", r)
						errorCh <- fmt.Errorf("handler panic: %v", r)
					}
				}()
				errorCh <- handler(handlerCtx, &kafkaMsg)
			}()
			
			// Wait for handler to complete or timeout
			select {
			case err = <-errorCh:
			case <-handlerCtx.Done():
				err = handlerCtx.Err()
			}
			cancel()

			if err != nil {
				log.Printf("ERROR processing message: %v", err)
				// Don't commit the message so it can be reprocessed
				continue
			}

			// Commit the message after successful processing
			commitCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
			if err := c.reader.CommitMessages(commitCtx, msg); err != nil {
				log.Printf("ERROR committing message: %v", err)
			} else {
				log.Printf("Successfully committed Kafka message at offset %d", msg.Offset)
			}
			cancel()
		}
	}
}

// BatchConsume reads batches of messages from Kafka and passes them to the handler function
func (c *Consumer) BatchConsume(ctx context.Context, batchSize int, timeout time.Duration, handler func(context.Context, []*model.KafkaMessage) error) error {
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
			// Read a batch of messages
			batch := make([]*model.KafkaMessage, 0, batchSize)
			msgs := make([]kafka.Message, 0, batchSize)
			batchStart := time.Now()

			// Continue reading until we have a full batch or timeout
			for len(batch) < batchSize && time.Since(batchStart) < timeout {
				// Set a timeout for reading the message
				readCtx, cancel := context.WithTimeout(ctx, timeout-time.Since(batchStart))
				msg, err := c.reader.FetchMessage(readCtx)
				cancel()

				if err != nil {
					// If we have messages in the batch, process them
					if len(batch) > 0 {
						break
					}
					if err == context.DeadlineExceeded || err == context.Canceled {
						break // Just a timeout, not an error
					}
					log.Printf("Error fetching message: %v", err)
					time.Sleep(time.Second) // Backoff before retrying
					break
				}

				// Parse the message
				var kafkaMsg model.KafkaMessage
				if err := json.Unmarshal(msg.Value, &kafkaMsg); err != nil {
					log.Printf("Error unmarshaling message: %v", err)
					// Commit the message even if it's invalid to avoid getting stuck
					commitCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
					if commitErr := c.reader.CommitMessages(commitCtx, msg); commitErr != nil {
						log.Printf("Error committing invalid message: %v", commitErr)
					}
					cancel()
					continue
				}

				// Add the message to the batch
				batch = append(batch, &kafkaMsg)
				msgs = append(msgs, msg)
			}

			// If we have messages, process them
			if len(batch) > 0 {
				// Process the batch
				if err := handler(ctx, batch); err != nil {
					log.Printf("Error processing batch: %v", err)
					// Don't commit - the messages will be reprocessed
					continue
				}

				// Commit all messages after successful processing
				commitCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
				if err := c.reader.CommitMessages(commitCtx, msgs...); err != nil {
					log.Printf("Error committing batch: %v", err)
				}
				cancel()
			}
		}
	}
}

// Close releases resources used by the consumer
func (c *Consumer) Close() error {
	return c.reader.Close()
}
