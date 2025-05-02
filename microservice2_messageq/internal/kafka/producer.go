package kafka

import (
	"context"
	"encoding/json"
	"log"
	"os"
	"strings"
	"time"

	kafka "github.com/segmentio/kafka-go"

	"github.com/appointy/notli/microservice2_messageq/internal/model"
)

// MessageProducer defines the interface for sending messages.
type MessageProducer interface {
	Enqueue(msg model.NotificationMessage) error
	Close() error
}

// kafkaProducer implements the MessageProducer interface.
type kafkaProducer struct {
	writer *kafka.Writer
}

var (
	writer *kafka.Writer // Keep global writer for now, or remove if NewKafkaProducer is always used
)

// NewKafkaProducer creates a new Kafka-backed MessageProducer optimized for high throughput.
func NewKafkaProducer() (MessageProducer, error) {
	brokers := strings.Split(os.Getenv("KAFKA_BROKERS"), ",")
	if len(brokers) == 0 || (len(brokers) == 1 && brokers[0] == "") {
		brokers = []string{"localhost:9092"}
	}
	topic := os.Getenv("INGRESS_TOPIC")
	if topic == "" {
		topic = "ingress-notifications"
	}
	
	// Get batch size from env or use default
	batchSize := 100
	batchBytes := 1048576 // 1MB default
	batchTimeout := 10 * time.Millisecond
	
	// Create a high-performance writer with batching and async mode
	kw := &kafka.Writer{
		Addr:         kafka.TCP(brokers...),
		Topic:        topic,
		Balancer:     &kafka.LeastBytes{},
		RequiredAcks: kafka.RequireOne, // Changed from RequireAll for better throughput
		Async:        true,            // Enable async mode for much higher throughput
		BatchSize:    batchSize,       // Configure batching
		BatchBytes:   int64(batchBytes),
		BatchTimeout: batchTimeout,
		// Handle errors with logger
		ErrorLogger:  log.New(os.Stderr, "kafka-error: ", log.LstdFlags),
	}

	log.Printf("High-Performance Kafka Producer initialized for topic '%s' with batch size %d", 
		topic, batchSize)

	return &kafkaProducer{
		writer: kw,
	}, nil
}

func init() {
	// Old init logic - can be removed if NewKafkaProducer is sufficient.
	/*
		brokers := strings.Split(os.Getenv("KAFKA_BROKERS"), ",")
		if len(brokers) == 0 || (len(brokers) == 1 && brokers[0] == "") {
			brokers = []string{"localhost:9092"}
		}
		topic := os.Getenv("INGRESS_TOPIC")
		if topic == "" {
			topic = "ingress-notifications"
		}

		writer = &kafka.Writer{
			Addr:         kafka.TCP(brokers...),
			Topic:        topic,
			Balancer:     &kafka.LeastBytes{},
			RequiredAcks: kafka.RequireAll, // Ensures message is written to leader and replicas
			Async:        false,          // Changed to false for simpler error handling initially
			// ErrorLog:    log.New(os.Stderr, "kafka-writer: ", log.LstdFlags), // Requires 'log'
		}
	*/
	// Adding back log import since it's used in NewKafkaProducer
	_ = log.LstdFlags // Dummy use to keep import
}

// Enqueue serializes the message and writes it to Kafka with optimized performance.
func (p *kafkaProducer) Enqueue(msg model.NotificationMessage) error {
	payload, err := json.Marshal(msg)
	if err != nil {
		return err
	}

	// Use a shorter timeout for better performance under high load
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Using message key for partitioning (based on idempotency key)
	// In async mode, errors will be logged but not returned
	return p.writer.WriteMessages(ctx, kafka.Message{
		Value: payload, 
		Key: []byte(msg.IdempotencyKey),
		// Add timestamp to help with message ordering
		Time: time.Now(),
	})
}

// Close closes the Kafka writer connection.
// func Close() error { // Convert to method
func (p *kafkaProducer) Close() error {
	if p.writer != nil {
		log.Println("Closing Kafka writer...")
		return p.writer.Close()
	}
	return nil
}

// Global Close function - can be removed if NewKafkaProducer is always used
func Close() error {
	if writer != nil {
		return writer.Close()
	}
	return nil
}
