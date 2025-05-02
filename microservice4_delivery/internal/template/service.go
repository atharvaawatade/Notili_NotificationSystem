package template

import (
	"context"
	"fmt"

	"github.com/appointy/notli/microservice4_delivery/internal/template/adapter"
	"github.com/appointy/notli/microservice4_delivery/internal/template/model"
	"github.com/appointy/notli/microservice4_delivery/internal/template/provider/mailjet"
)

// Service provides operations for managing and using templates
type Service struct {
	manager *adapter.AdapterManager
}

// Config holds configuration for the template service
type Config struct {
	// Default provider to use when none is specified
	DefaultProvider string `json:"default_provider"`

	// Provider-specific configurations
	Providers map[string]map[string]interface{} `json:"providers"`
}

// DefaultConfig returns a default configuration with the provided Mailjet credentials
func DefaultConfig(mailjetAPIKey, mailjetSecretKey string) *Config {
	return &Config{
		DefaultProvider: mailjet.ProviderName,
		Providers: map[string]map[string]interface{}{
			mailjet.ProviderName: {
				"api_key":             mailjetAPIKey,
				"secret_key":          mailjetSecretKey,
				"template_owner_type": "user",
				"default_limit":       100,
			},
		},
	}
}

// NewService creates a new template service with the provided configuration
func NewService(cfg *Config) (*Service, error) {
	// Create registry and register adapter factories
	registry := adapter.NewRegistry()
	mailjet.Register(registry)

	// Create adapter manager
	manager := adapter.NewAdapterManager(registry)

	// Initialize providers from configuration
	for providerName, providerConfig := range cfg.Providers {
		templateAdapter, err := registry.Create(providerName, providerConfig)
		if err != nil {
			return nil, fmt.Errorf("failed to initialize template provider %s: %w", providerName, err)
		}
		manager.AddAdapter(templateAdapter)
	}

	return &Service{
		manager: manager,
	}, nil
}

// ListTemplates retrieves a list of templates from the specified provider
func (s *Service) ListTemplates(ctx context.Context, providerName string, query *model.TemplateQuery) ([]model.Template, error) {
	// Get the adapter for the specified provider
	templateAdapter, found := s.manager.GetAdapter(providerName)
	if !found {
		// Try to use the default provider
		templateAdapter = s.manager.GetDefaultAdapter()
		if templateAdapter == nil {
			return nil, adapter.ErrProviderNotFound
		}
	}

	return templateAdapter.ListTemplates(ctx, query)
}

// GetTemplate retrieves a template by ID from the specified provider
func (s *Service) GetTemplate(ctx context.Context, providerName, templateID string) (*model.Template, error) {
	// Get the adapter for the specified provider
	templateAdapter, found := s.manager.GetAdapter(providerName)
	if !found {
		// Try to use the default provider
		templateAdapter = s.manager.GetDefaultAdapter()
		if templateAdapter == nil {
			return nil, adapter.ErrProviderNotFound
		}
	}

	return templateAdapter.GetTemplate(ctx, templateID)
}

// GetTemplateVariables retrieves the variables for a template from the specified provider
func (s *Service) GetTemplateVariables(ctx context.Context, providerName, templateID string) ([]model.TemplateVariable, error) {
	// Get the adapter for the specified provider
	templateAdapter, found := s.manager.GetAdapter(providerName)
	if !found {
		// Try to use the default provider
		templateAdapter = s.manager.GetDefaultAdapter()
		if templateAdapter == nil {
			return nil, adapter.ErrProviderNotFound
		}
	}

	return templateAdapter.GetTemplateVariables(ctx, templateID)
}

// Close releases resources used by the service and its adapters
func (s *Service) Close() error {
	return s.manager.Close()
}
