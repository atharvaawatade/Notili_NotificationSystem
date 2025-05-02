package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/joho/godotenv"
	"github.com/segmentio/kafka-go"
)

// TestMessage represents a notification message to be sent to Kafka
type TestMessage struct {
	IdempotencyKey string                 `json:"idempotency_key"`
	Recipient      string                 `json:"recipient"`
	TemplateID     string                 `json:"template_id,omitempty"`
	Priority       int                    `json:"priority"`
	MessageType    string                 `json:"message_type"`
	Channel        string                 `json:"channel"`
	ChannelData    map[string]interface{} `json:"channel_data"`
	UserID         string                 `json:"user_id,omitempty"`
	UserTier       string                 `json:"user_tier,omitempty"`
}

func main() {
	// Load environment variables
	if err := godotenv.Load(); err != nil {
		log.Printf("Warning: .env file not found: %v", err)
	}

	// Prepare test messages
	testMessages := []TestMessage{
		{
			// Test 1: Basic transactional email with direct content
			IdempotencyKey: fmt.Sprintf("test-direct-%d", time.Now().UnixNano()),
			Recipient:      "test@example.com",
			Priority:       10,
			MessageType:    "transactional",
			Channel:        "email",
			ChannelData: map[string]interface{}{
				"subject":    "Test Transactional Email",
				"plain_body": "This is a test transactional email sent from Notli.",
				"html_body":  "<h1>Test Email</h1><p>This is a test transactional email sent from Notli.</p>",
			},
			UserID:   "user-123",
			UserTier: "premium",
		},
		{
			// Test 2: Email using a template
			IdempotencyKey: fmt.Sprintf("test-template-%d", time.Now().UnixNano()),
			Recipient:      "template@example.com",
			TemplateID:     "welcome",
			Priority:       8,
			MessageType:    "transactional",
			Channel:        "email",
			ChannelData: map[string]interface{}{
				"template_data": map[string]interface{}{
					"Name":    "Template User",
					"AppName": "Notli Testing",
				},
			},
			UserID:   "user-456",
			UserTier: "standard",
		},
		{
			// Test 3: Promotional email (lower priority)
			IdempotencyKey: fmt.Sprintf("test-promo-%d", time.Now().UnixNano()),
			Recipient:      "promo@example.com",
			Priority:       3,
			MessageType:    "promotional",
			Channel:        "email",
			ChannelData: map[string]interface{}{
				"subject":    "Special Offer Inside!",
				"plain_body": "Check out our latest offers exclusively for you!",
				"html_body":  "<h1>Special Offer!</h1><p>Check out our <strong>latest offers</strong> exclusively for you!</p>",
			},
			UserID:   "user-789",
			UserTier: "free",
		},
		{
			// Test 4: Sending to a different channel (should be ignored by our service)
			IdempotencyKey: fmt.Sprintf("test-sms-%d", time.Now().UnixNano()),
			Recipient:      "+1234567890",
			Priority:       9,
			MessageType:    "transactional",
			Channel:        "sms", // Will be ignored by our email service
			ChannelData: map[string]interface{}{
				"text": "This is a test SMS that should be ignored by the email service",
			},
			UserID:   "user-101",
			UserTier: "premium",
		},
		{
			// Test 5: Duplicate message (should be ignored due to idempotency)
			IdempotencyKey: "test-duplicate-key", // Fixed key for testing deduplication
			Recipient:      "duplicate@example.com",
			Priority:       7,
			MessageType:    "transactional",
			Channel:        "email",
			ChannelData: map[string]interface{}{
				"subject":    "This is a duplicate message",
				"plain_body": "This message should be deduplicated.",
				"html_body":  "<p>This message should be deduplicated.</p>",
			},
		},
	}

	// Get Kafka configuration from environment variables
	bootstrapServers := getEnv("KAFKA_BOOTSTRAP_SERVERS", "localhost:9092")
	inputTopic := getEnv("KAFKA_INPUT_TOPIC", "distributed-notifications")

	// Create a context that can be canceled
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer cancel()

	// Create Kafka writer
	writer := kafka.NewWriter(kafka.WriterConfig{
		Brokers:  []string{bootstrapServers},
		Topic:    inputTopic,
		Balancer: &kafka.LeastBytes{},
	})
	defer writer.Close()

	// Send messages to Kafka
	for i, msg := range testMessages {
		// Marshal the message to JSON
		msgBytes, err := json.Marshal(msg)
		if err != nil {
			log.Printf("Error marshaling message %d: %v", i+1, err)
			continue
		}

		// Create Kafka message
		kafkaMsg := kafka.Message{
			Key:   []byte(msg.IdempotencyKey),
			Value: msgBytes,
			Time:  time.Now(),
		}

		// Send the message
		log.Printf("Sending test message %d: %s to %s", i+1, msg.IdempotencyKey, msg.Recipient)
		if err := writer.WriteMessages(ctx, kafkaMsg); err != nil {
			log.Printf("Error sending message %d: %v", i+1, err)
			continue
		}

		log.Printf("Successfully sent message %d", i+1)

		// Wait a moment between messages
		time.Sleep(500 * time.Millisecond)
	}

	// Send the duplicate message again to test idempotency
	time.Sleep(2 * time.Second)
	log.Println("Sending duplicate message again to test idempotency...")
	
	// Marshal the duplicate message again
	dupMsg := testMessages[4] // The duplicate message
	msgBytes, _ := json.Marshal(dupMsg)
	kafkaMsg := kafka.Message{
		Key:   []byte(dupMsg.IdempotencyKey),
		Value: msgBytes,
		Time:  time.Now(),
	}
	
	if err := writer.WriteMessages(ctx, kafkaMsg); err != nil {
		log.Printf("Error sending duplicate message: %v", err)
	} else {
		log.Println("Successfully sent duplicate message")
	}

	log.Println("Test script completed. Check logs for results.")
}

// getEnv gets an environment variable with a fallback
func getEnv(key, fallback string) string {
	if value, exists := os.LookupEnv(key); exists {
		return value
	}
	return fallback
}
