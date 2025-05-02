package service

import (
	"context"
	"sync"

	"github.com/appointy/notli/microservice3_prioritization/internal/model"
)

// MockConsumer is a mock implementation of kafka.MessageConsumer for testing
type MockConsumer struct {
	processCh chan *model.NotificationMessage
}

func NewMockConsumer() *MockConsumer {
	return &MockConsumer{
		processCh: make(chan *model.NotificationMessage, 100),
	}
}

func (c *MockConsumer) Start(ctx context.Context) error {
	return nil
}

func (c *MockConsumer) GetProcessChannel() <-chan *model.NotificationMessage {
	return c.processCh
}

func (c *MockConsumer) Stop() error {
	close(c.processCh)
	return nil
}

func (c *MockConsumer) SendMessage(msg *model.NotificationMessage) {
	c.processCh <- msg
}

// MockProducer is a mock implementation of kafka.MessageProducer for testing
type MockProducer struct {
	messages []model.NotificationMessage
	mutex    sync.Mutex
}

func NewMockProducer() *MockProducer {
	return &MockProducer{
		messages: make([]model.NotificationMessage, 0),
	}
}

func (p *MockProducer) Produce(ctx context.Context, message *model.NotificationMessage) error {
	p.mutex.Lock()
	defer p.mutex.Unlock()
	p.messages = append(p.messages, *message)
	return nil
}

func (p *MockProducer) Close() error {
	return nil
}

func (p *MockProducer) GetMessages() []model.NotificationMessage {
	p.mutex.Lock()
	defer p.mutex.Unlock()
	return p.messages
}
