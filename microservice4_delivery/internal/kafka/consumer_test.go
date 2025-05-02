package kafka

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/appointy/notli/microservice4_delivery/internal/model"
	"github.com/segmentio/kafka-go"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// MockKafkaReader implements the KafkaReader interface for testing
type MockKafkaReader struct {
	mock.Mock
	messages []kafka.Message
	index    int
}

// NewMockKafkaReader creates a new mock reader with test messages
func NewMockKafkaReader() *MockKafkaReader {
	return &MockKafkaReader{
		messages: []kafka.Message{},
		index:    0,
	}
}

// SetupWithMessages configures the mock reader to return the given messages
func (m *MockKafkaReader) SetupWithMessages(messages []kafka.Message, err error) {
	for _, msg := range messages {
		m.On("FetchMessage", mock.Anything).Return(msg, nil).Once()
	}
	
	// After all messages are consumed, block on context cancellation
	m.On("FetchMessage", mock.Anything).Run(func(args mock.Arguments) {
		ctx := args.Get(0).(context.Context)
		<-ctx.Done() // Block until context is canceled
	}).Return(kafka.Message{}, context.Canceled)
	
	m.On("CommitMessages", mock.Anything, mock.Anything).Return(nil)
	m.On("Close").Return(nil)
}

// No longer needed as we can use NewConsumerWithReader from consumer.go

// ReadMessage is an alias for FetchMessage to satisfy the kafka.Reader interface
func (m *MockKafkaReader) ReadMessage(ctx context.Context) (kafka.Message, error) {
	return m.FetchMessage(ctx)
}

// FetchMessage implements kafka.Reader.FetchMessage
func (m *MockKafkaReader) FetchMessage(ctx context.Context) (kafka.Message, error) {
	args := m.Called(ctx)
	return args.Get(0).(kafka.Message), args.Error(1)
}

// CommitMessages implements kafka.Reader.CommitMessages
func (m *MockKafkaReader) CommitMessages(ctx context.Context, msgs ...kafka.Message) error {
	args := m.Called(ctx, msgs)
	return args.Error(0)
}

// Close implements kafka.Reader.Close
func (m *MockKafkaReader) Close() error {
	args := m.Called()
	return args.Error(0)
}

// SetOffsetAt implements the required kafka.Reader method
func (m *MockKafkaReader) SetOffsetAt(ctx context.Context, t time.Time) error {
	return nil
}

// SetOffset implements the required kafka.Reader method
func (m *MockKafkaReader) SetOffset(offset int64) error {
	return nil
}

// Lag implements the required kafka.Reader method
func (m *MockKafkaReader) Lag() int64 {
	return 0
}

// Stats implements the required kafka.Reader method
func (m *MockKafkaReader) Stats() kafka.ReaderStats {
	return kafka.ReaderStats{}
}

// TestConsume tests the Consume method
func TestConsume(t *testing.T) {
	// Create test messages
	testMsg := model.KafkaMessage{
		IdempotencyKey: "test-123",
		Recipient:      "test@example.com",
		TemplateID:     "welcome",
		Priority:       "5", // Changed from int to string
		MessageType:    "transactional",
		Channel:        "email",
		ChannelData: map[string]interface{}{
			"subject": "Test Subject",
		},
	}
	
	msgBytes, err := json.Marshal(testMsg)
	assert.NoError(t, err)
	
	kafkaMessages := []kafka.Message{
		{
			Key:   []byte("test-key"),
			Value: msgBytes,
			Time:  time.Now(),
		},
	}
	
	// Create mock reader and set up expected behavior
	mockReader := NewMockKafkaReader()
	mockReader.SetupWithMessages(kafkaMessages, nil)
	
	// Create consumer with mock reader
	consumer := NewConsumerWithReader(mockReader, "test-topic")
	
	// Create processing channel to collect handled messages
	processed := make(chan *model.KafkaMessage, 10)
	handler := func(ctx context.Context, msg *model.KafkaMessage) error {
		processed <- msg
		return nil
	}
	
	// Start consuming in a goroutine with a cancelable context
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	
	go func() {
		// Cancel after a short delay to stop consumption
		time.Sleep(100 * time.Millisecond)
		cancel()
	}()
	
	// Consume messages
	err = consumer.Consume(ctx, handler)
	assert.Equal(t, context.Canceled, err)
	
	// Check that the message was processed
	select {
	case processedMsg := <-processed:
		assert.Equal(t, "test-123", processedMsg.IdempotencyKey)
		assert.Equal(t, "test@example.com", processedMsg.Recipient)
		assert.Equal(t, "welcome", processedMsg.TemplateID)
	case <-time.After(500 * time.Millisecond):
		t.Fatal("Timeout waiting for message to be processed")
	}
}

// TestBatchConsume tests the BatchConsume method
func TestBatchConsume(t *testing.T) {
	// Create test messages
	testMsgs := []model.KafkaMessage{
		{
			IdempotencyKey: "test-1",
			Recipient:      "test1@example.com",
			Channel:        "email",
		},
		{
			IdempotencyKey: "test-2",
			Recipient:      "test2@example.com",
			Channel:        "email",
		},
	}
	
	kafkaMessages := make([]kafka.Message, len(testMsgs))
	for i, msg := range testMsgs {
		msgBytes, err := json.Marshal(msg)
		assert.NoError(t, err)
		
		kafkaMessages[i] = kafka.Message{
			Key:   []byte(msg.IdempotencyKey),
			Value: msgBytes,
			Time:  time.Now(),
		}
	}
	
	// Create mock reader and set up expected behavior
	mockReader := NewMockKafkaReader()
	mockReader.SetupWithMessages(kafkaMessages, nil)
	
	// Create consumer with mock reader
	consumer := NewConsumerWithReader(mockReader, "test-topic")
	
	// Create processing channel to collect handled batches
	processed := make(chan []*model.KafkaMessage, 1)
	handler := func(ctx context.Context, msgs []*model.KafkaMessage) error {
		processed <- msgs
		return nil
	}
	
	// Start consuming in a goroutine with a cancelable context
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	
	go func() {
		// Cancel after a short delay to stop consumption
		time.Sleep(100 * time.Millisecond)
		cancel()
	}()
	
	// Consume messages in batches
	err := consumer.BatchConsume(ctx, 10, 50*time.Millisecond, handler)
	assert.Equal(t, context.Canceled, err)
	
	// Check that the batch was processed
	select {
	case processedBatch := <-processed:
		assert.Equal(t, 2, len(processedBatch))
		assert.Equal(t, "test-1", processedBatch[0].IdempotencyKey)
		assert.Equal(t, "test-2", processedBatch[1].IdempotencyKey)
	case <-time.After(500 * time.Millisecond):
		t.Fatal("Timeout waiting for batch to be processed")
	}
}
