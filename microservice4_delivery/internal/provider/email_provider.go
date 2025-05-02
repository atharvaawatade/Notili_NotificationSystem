package provider

import (
	"context"
	
	"github.com/appointy/notli/microservice4_delivery/internal/model"
)

// EmailProvider defines the interface that all email providers must implement
type EmailProvider interface {
	// Name returns the name of the provider
	Name() string
	
	// Send transmits an email through the provider
	Send(ctx context.Context, email *model.EmailMessage) (string, error)
	
	// GetStatus retrieves the current status of a previously sent email
	GetStatus(ctx context.Context, messageID string) (*model.EmailStatus, error)
	
	// Close releases any resources used by the provider
	Close() error
}

// Factory is a function type for creating email provider instances
type Factory func(config map[string]string) (EmailProvider, error)

// ProviderRegistry maintains a map of available email providers
type ProviderRegistry struct {
	providers map[string]Factory
}

// NewProviderRegistry creates a new registry of email providers
func NewProviderRegistry() *ProviderRegistry {
	return &ProviderRegistry{
		providers: make(map[string]Factory),
	}
}

// Register adds a new provider factory to the registry
func (r *ProviderRegistry) Register(name string, factory Factory) {
	r.providers[name] = factory
}

// Create instantiates a provider with the given name and configuration
func (r *ProviderRegistry) Create(name string, config map[string]string) (EmailProvider, error) {
	factory, exists := r.providers[name]
	if !exists {
		return nil, ErrProviderNotFound
	}
	
	return factory(config)
}

// Errors returned by the provider package
var (
	ErrProviderNotFound = EmailProviderError("provider not found")
	ErrInvalidConfig    = EmailProviderError("invalid provider configuration")
	ErrSendFailed       = EmailProviderError("failed to send email")
)

// EmailProviderError is a simple error implementation for provider errors
type EmailProviderError string

func (e EmailProviderError) Error() string {
	return string(e)
}
