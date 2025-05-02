package service

import (
	"context"
	// "encoding/json" // Commented out as it's not used after removing prettyPrint
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/appointy/notli/microservice3_prioritization/internal/kafka"
	"github.com/appointy/notli/microservice3_prioritization/internal/model"
	"github.com/appointy/notli/microservice3_prioritization/internal/priority"
	"github.com/appointy/notli/microservice3_prioritization/internal/ratelimit"
	"github.com/go-redis/redis/v8"
	"github.com/stretchr/testify/assert"
)

// MockMessageConsumer implements a simplified version of the consumer for testing
type MockMessageConsumer struct {
	processCh chan *model.NotificationMessage
}

func NewMockMessageConsumer() *MockMessageConsumer {
	return &MockMessageConsumer{
		processCh: make(chan *model.NotificationMessage, 100),
	}
}

func (c *MockMessageConsumer) Start(ctx context.Context) error {
	return nil
}

func (c *MockMessageConsumer) GetProcessChannel() <-chan *model.NotificationMessage {
	return c.processCh
}

func (c *MockMessageConsumer) Stop() error {
	close(c.processCh)
	return nil
}

func (c *MockMessageConsumer) SendMessage(msg *model.NotificationMessage) {
	c.processCh <- msg
}

// MockMessageProducer implements a simplified version of the producer for testing
type MockMessageProducer struct {
	messages []model.NotificationMessage
	mutex    sync.Mutex
}

func NewMockMessageProducer() *MockMessageProducer {
	return &MockMessageProducer{
		messages: make([]model.NotificationMessage, 0),
	}
}

func (p *MockMessageProducer) Produce(ctx context.Context, message *model.NotificationMessage) error {
	p.mutex.Lock()
	defer p.mutex.Unlock()
	p.messages = append(p.messages, *message)
	return nil
}

func (p *MockMessageProducer) Close() error {
	return nil
}

func (p *MockMessageProducer) GetMessages() []model.NotificationMessage {
	p.mutex.Lock()
	defer p.mutex.Unlock()
	return p.messages
}

func TestPriorityService_Integration(t *testing.T) {
	// Setup in-memory Redis for testing
	mr, err := miniredis.Run()
	if err != nil {
		t.Fatalf("Failed to create miniredis: %v", err)
	}
	defer mr.Close()

	redisClient := redis.NewClient(&redis.Options{
		Addr: mr.Addr(),
	})
	defer redisClient.Close()

	// Create mock components
	mockConsumer := NewMockMessageConsumer()
	mockProducer := NewMockMessageProducer()
	ruleEngine := priority.NewRuleEngine()
	rateLimiter := ratelimit.NewRateLimiter(redisClient)

	// Create priority service with mocks
	// Create a test-specific priority service
	priorityService := &PriorityService{
		consumer:          interface{}(mockConsumer).(*kafka.MessageConsumer),
		producer:          interface{}(mockProducer).(*kafka.MessageProducer),
		ruleEngine:        ruleEngine,
		rateLimiter:       rateLimiter,
		priorityQueue:     priority.NewPriorityQueue(),
		transactionalRate: 1000,
		promotionalRate:   100,
	}

	// Start the service
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	err = priorityService.Start(ctx)
	assert.NoError(t, err)

	// Create test messages with different priorities
	now := time.Now()

	// High priority transactional message
	highPriorityMsg := &model.NotificationMessage{
		IdempotencyKey: "high-priority-test",
		Priority:       "high",
		MessageType:    "transactional",
		Channel:        "email",
		UserTier:       "premium",
		TemplateID:     "password_reset",
		Recipient:      "user1@example.com",
		CreatedAt:      now,
		ChannelData: map[string]interface{}{
			"subject": "Password Reset",
			"body":    "Your password has been reset",
		},
	}

	// Medium priority transactional message
	mediumPriorityMsg := &model.NotificationMessage{
		IdempotencyKey: "medium-priority-test",
		Priority:       "normal",
		MessageType:    "transactional",
		Channel:        "email",
		UserTier:       "standard",
		TemplateID:     "order_confirmation",
		Recipient:      "user2@example.com",
		CreatedAt:      now,
		ChannelData: map[string]interface{}{
			"subject": "Order Confirmation",
			"body":    "Thank you for your order",
		},
	}

	// Low priority promotional message
	lowPriorityMsg := &model.NotificationMessage{
		IdempotencyKey: "low-priority-test",
		Priority:       "low",
		MessageType:    "promotional",
		Channel:        "email",
		UserTier:       "free",
		TemplateID:     "marketing_campaign",
		Recipient:      "user3@example.com",
		CreatedAt:      now,
		ChannelData: map[string]interface{}{
			"subject": "Special Offer",
			"body":    "Check out our latest deals",
		},
	}

	// Send messages to the consumer in reverse priority order
	mockConsumer.SendMessage(lowPriorityMsg)
	mockConsumer.SendMessage(mediumPriorityMsg)
	mockConsumer.SendMessage(highPriorityMsg)

	// Allow time for processing
	time.Sleep(500 * time.Millisecond)

	// Get processed messages
	processedMessages := mockProducer.GetMessages()

	// We should have 3 messages
	assert.Equal(t, 3, len(processedMessages))

	// Messages should be processed in order of priority
	if len(processedMessages) >= 3 {
		// First message should be high priority
		assert.Equal(t, "high-priority-test", processedMessages[0].IdempotencyKey)

		// Second message should be medium priority
		assert.Equal(t, "medium-priority-test", processedMessages[1].IdempotencyKey)

		// Third message should be low priority
		assert.Equal(t, "low-priority-test", processedMessages[2].IdempotencyKey)
	}

	// Stop the service
	err = priorityService.Stop()
	assert.NoError(t, err)
}

func TestPriorityService_RateLimiting(t *testing.T) {
	// Setup in-memory Redis for testing
	mr, err := miniredis.Run()
	if err != nil {
		t.Fatalf("Failed to create miniredis: %v", err)
	}
	defer mr.Close()

	redisClient := redis.NewClient(&redis.Options{
		Addr: mr.Addr(),
	})
	defer redisClient.Close()

	// Create mock components
	mockConsumer := NewMockMessageConsumer()
	mockProducer := NewMockMessageProducer()
	ruleEngine := priority.NewRuleEngine()
	rateLimiter := ratelimit.NewRateLimiter(redisClient)

	// Create priority service with strict rate limits for testing
	priorityService := &PriorityService{
		consumer:          interface{}(mockConsumer).(*kafka.MessageConsumer),
		producer:          interface{}(mockProducer).(*kafka.MessageProducer),
		ruleEngine:        ruleEngine,
		rateLimiter:       rateLimiter,
		priorityQueue:     priority.NewPriorityQueue(),
		transactionalRate: 100, // transactional rate - higher
		promotionalRate:   2,   // promotional rate - very restrictive for testing
	}

	// Start the service
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	err = priorityService.Start(ctx)
	assert.NoError(t, err)

	// Create a bunch of promotional messages for the same user
	now := time.Now()

	// Send 10 promotional messages for the same user in rapid succession
	for i := 0; i < 10; i++ {
		promoMsg := &model.NotificationMessage{
			IdempotencyKey: fmt.Sprintf("promo-test-%d", i),
			Priority:       "normal",
			MessageType:    "promotional",
			Channel:        "email",
			UserTier:       "standard",
			TemplateID:     "marketing_campaign",
			Recipient:      "same-user@example.com", // Same user for all messages
			CreatedAt:      now,
			ChannelData: map[string]interface{}{
				"subject": fmt.Sprintf("Promo Offer %d", i),
				"body":    "Check out our latest deals",
			},
		}
		mockConsumer.SendMessage(promoMsg)
	}

	// Allow time for processing
	time.Sleep(500 * time.Millisecond)

	// Get processed messages
	processedMessages := mockProducer.GetMessages()

	// Due to rate limiting, we should have processed fewer than 10 messages
	// The exact number depends on rate limiter implementation, but should be around 2-3
	assert.Less(t, len(processedMessages), 10, "Rate limiting should prevent processing all messages")
	assert.GreaterOrEqual(t, len(processedMessages), 1, "At least some messages should be processed")

	// Send a transactional message - should be processed despite rate limits on promotional
	transMsg := &model.NotificationMessage{
		IdempotencyKey: "trans-test",
		Priority:       "high",
		MessageType:    "transactional",
		Channel:        "email",
		UserTier:       "standard",
		TemplateID:     "password_reset",
		Recipient:      "same-user@example.com", // Same user that hit rate limits
		CreatedAt:      now,
		ChannelData: map[string]interface{}{
			"subject": "Important Notification",
			"body":    "Critical information",
		},
	}
	mockConsumer.SendMessage(transMsg)

	// Allow time for processing
	time.Sleep(200 * time.Millisecond)

	// Get updated processed messages
	updatedMessages := mockProducer.GetMessages()

	// We should have at least one more message than before
	assert.Greater(t, len(updatedMessages), len(processedMessages),
		"Transactional message should be processed despite rate limiting on promotional")

	// Check if the last processed message is the transactional one
	if len(updatedMessages) > len(processedMessages) {
		lastMsg := updatedMessages[len(updatedMessages)-1]
		assert.Equal(t, "trans-test", lastMsg.IdempotencyKey,
			"Last processed message should be the transactional one")
		assert.Equal(t, "transactional", lastMsg.MessageType,
			"Message type should be transactional")
	}

	// Stop the service
	err = priorityService.Stop()
	assert.NoError(t, err)
}

// Utility function for test
// prettyPrint function commented out as it's currently unused
// func prettyPrint(v interface{}) string {
// 	b, _ := json.MarshalIndent(v, "", "  ")
// 	return string(b)
// }
