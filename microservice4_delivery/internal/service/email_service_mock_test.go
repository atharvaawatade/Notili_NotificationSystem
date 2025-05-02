package service

import (
	"context"
	"time"

	"github.com/appointy/notli/microservice4_delivery/internal/model"
	"github.com/appointy/notli/microservice4_delivery/internal/repository"
	"github.com/stretchr/testify/mock"
)

// MockEmailRepository is a mock implementation of EmailRepository for testing
type MockEmailRepository struct {
	mock.Mock
	*repository.EmailRepository
}

// NewMockEmailRepository creates a new mock repository for testing
func NewMockEmailRepository() *MockEmailRepository {
	return &MockEmailRepository{
		EmailRepository: &repository.EmailRepository{}, // Empty struct
	}
}

// CreateDelivery mocks the CreateDelivery method
func (m *MockEmailRepository) CreateDelivery(ctx context.Context, delivery *model.DeliveryRecord) error {
	args := m.Called(ctx, delivery)
	return args.Error(0)
}

// UpdateDeliveryStatus mocks the UpdateDeliveryStatus method
func (m *MockEmailRepository) UpdateDeliveryStatus(ctx context.Context, messageID, status, details string) error {
	args := m.Called(ctx, messageID, status, details)
	return args.Error(0)
}

// UpdateRetryInfo mocks the UpdateRetryInfo method
func (m *MockEmailRepository) UpdateRetryInfo(ctx context.Context, messageID string, retryCount int, nextRetryAt time.Time) error {
	args := m.Called(ctx, messageID, retryCount, nextRetryAt)
	return args.Error(0)
}

// AddStatusEvent mocks the AddStatusEvent method
func (m *MockEmailRepository) AddStatusEvent(ctx context.Context, event *model.StatusEvent) error {
	args := m.Called(ctx, event)
	return args.Error(0)
}

// GetDeliveryByID mocks the GetDeliveryByID method
func (m *MockEmailRepository) GetDeliveryByID(ctx context.Context, messageID string) (*model.DeliveryRecord, error) {
	args := m.Called(ctx, messageID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*model.DeliveryRecord), args.Error(1)
}

// GetDeliveryByIdempotencyKey mocks the GetDeliveryByIdempotencyKey method
func (m *MockEmailRepository) GetDeliveryByIdempotencyKey(ctx context.Context, idempotencyKey string) (*model.DeliveryRecord, error) {
	args := m.Called(ctx, idempotencyKey)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*model.DeliveryRecord), args.Error(1)
}

// GetDeliveriesToRetry mocks the GetDeliveriesToRetry method
func (m *MockEmailRepository) GetDeliveriesToRetry(ctx context.Context, limit int) ([]*model.DeliveryRecord, error) {
	args := m.Called(ctx, limit)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*model.DeliveryRecord), args.Error(1)
}

// GetStatusEvents mocks the GetStatusEvents method
func (m *MockEmailRepository) GetStatusEvents(ctx context.Context, messageID string) ([]*model.StatusEvent, error) {
	args := m.Called(ctx, messageID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*model.StatusEvent), args.Error(1)
}

// Init mocks the Init method
func (m *MockEmailRepository) Init(ctx context.Context) error {
	args := m.Called(ctx)
	return args.Error(0)
}

// MockTemplateEngineAdapter adapts our mock to the template.Engine type
type MockTemplateEngineAdapter struct {
	mock *TestTemplateEngine
}

// Render delegates to the mock
func (m *MockTemplateEngineAdapter) Render(templateID string, data map[string]interface{}) (string, string, string, error) {
	return m.mock.Render(templateID, data)
}

// Get delegates to the mock
func (m *MockTemplateEngineAdapter) Get(templateID string) (interface{}, error) {
	m.mock.Get(templateID)
	return nil, nil
}

// ReloadTemplates is a no-op for the mock
func (m *MockTemplateEngineAdapter) ReloadTemplates() error {
	return nil
}

// AddTemplate is a no-op for the mock
func (m *MockTemplateEngineAdapter) AddTemplate(tmpl interface{}) {
	// No-op
}
