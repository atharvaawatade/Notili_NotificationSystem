package service

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/appointy/notli/microservice3_prioritization/internal/kafka"
	"github.com/appointy/notli/microservice3_prioritization/internal/model"
	"github.com/appointy/notli/microservice3_prioritization/internal/priority"
	"github.com/appointy/notli/microservice3_prioritization/internal/ratelimit"
)

// PriorityService orchestrates the prioritization process
type PriorityService struct {
	consumer     *kafka.MessageConsumer
	producer     *kafka.MessageProducer
	ruleEngine   *priority.RuleEngine
	rateLimiter  *ratelimit.RateLimiter
	priorityQueue *priority.PriorityQueue
	transactionalRate int
	promotionalRate int
}

// NewPriorityService creates a new priority service
func NewPriorityService(
	consumer *kafka.MessageConsumer,
	producer *kafka.MessageProducer,
	ruleEngine *priority.RuleEngine,
	rateLimiter *ratelimit.RateLimiter,
	transactionalRate int,
	promotionalRate int,
) *PriorityService {
	return &PriorityService{
		consumer:     consumer,
		producer:     producer,
		ruleEngine:   ruleEngine,
		rateLimiter:  rateLimiter,
		priorityQueue: priority.NewPriorityQueue(),
		transactionalRate: transactionalRate,
		promotionalRate: promotionalRate,
	}
}

// Start begins the priority service operations
func (s *PriorityService) Start(ctx context.Context) error {
	// Start the Kafka consumer
	if err := s.consumer.Start(ctx); err != nil {
		return fmt.Errorf("failed to start consumer: %w", err)
	}

	// Start the queue processor
	go s.processQueue(ctx)

	// Start the message handler
	go s.handleMessages(ctx)

	log.Println("Priority service started successfully")
	return nil
}

// handleMessages processes incoming messages from Kafka
func (s *PriorityService) handleMessages(ctx context.Context) {
	processCh := s.consumer.GetProcessChannel()

	for {
		select {
		case <-ctx.Done():
			return
		case msg := <-processCh:
			// Set default creation time if not set
			if msg.CreatedAt.IsZero() {
				msg.CreatedAt = time.Now()
			}

			// Apply prioritization rules
			priorityScore := s.ruleEngine.CalculatePriorityScore(msg)
			msg.PriorityScore = priorityScore

			// Apply rate limiting based on message type and recipient
			// Create rate limit info from message
			// Since NotificationMessage doesn't have UserID, we'll use IdempotencyKey
			// which should be unique per user message
			rateInfo := &model.RateLimitInfo{
				UserID:      msg.IdempotencyKey, // Using IdempotencyKey as UserID
				MessageType: msg.MessageType,
				Channel:     msg.Channel,
				Timestamp:   time.Now(),
			}
			
			// Get rate limit based on message type
			baseRate := s.getRateLimitForMessageType(msg.MessageType)
			
			// Check if message is allowed
			allowed, retryAfter, err := s.rateLimiter.AllowMessage(context.Background(), rateInfo, baseRate)
			if err != nil {
				log.Printf("Rate limiting check failed, allowing message: %v", err)
				allowed = true // Allow on failure
			}
			
			log.Printf("Processing message for %s (type: %s) - allowed: %v", 
				msg.Recipient, msg.MessageType, allowed)

			if allowed {
				// Add to priority queue
				s.priorityQueue.Enqueue(msg, priorityScore)
			} else {
				log.Printf("Rate limited message for %s, retry after %v", msg.Recipient, retryAfter)
				// Re-queue with delay based on rate limiter response
				delay := retryAfter
				if delay < time.Second {
					delay = time.Second // Minimum 1 second delay
				}
				time.AfterFunc(delay, func() {
					s.priorityQueue.Enqueue(msg, priorityScore)
				})
			}
		}
	}
}

// processQueue processes messages from the priority queue
func (s *PriorityService) processQueue(ctx context.Context) {
	ticker := time.NewTicker(10 * time.Millisecond) // Process queue every 10ms
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			// Process up to 100 messages per batch
			for i := 0; i < 100; i++ {
				// Get the highest priority message
				prioritizedMsg, ok := s.priorityQueue.Dequeue()
				if !ok {
					// No more messages
					break
				}

				// Send to output topic
				err := s.producer.Produce(ctx, prioritizedMsg.Message)
				if err != nil {
					log.Printf("Error producing message: %v", err)
					// Re-queue on error with a slight delay
					time.AfterFunc(100*time.Millisecond, func() {
						s.priorityQueue.Enqueue(prioritizedMsg.Message, prioritizedMsg.PriorityScore)
					})
				}
			}
		}
	}
}

// getRateLimitForMessageType returns the appropriate rate limit based on message type
func (s *PriorityService) getRateLimitForMessageType(messageType string) int {
	switch messageType {
	case "transactional":
		return s.transactionalRate
	case "promotional":
		return s.promotionalRate
	default:
		// Default to promotional (stricter) rate limiting for unknown types
		return s.promotionalRate
	}
}

// Stop gracefully stops the service
func (s *PriorityService) Stop() error {
	// Stop the consumer
	if err := s.consumer.Stop(); err != nil {
		return fmt.Errorf("failed to stop consumer: %w", err)
	}

	// Close the producer
	if err := s.producer.Close(); err != nil {
		return fmt.Errorf("failed to close producer: %w", err)
	}

	log.Println("Priority service stopped successfully")
	return nil
}
