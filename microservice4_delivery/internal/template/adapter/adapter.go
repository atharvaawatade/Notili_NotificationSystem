package adapter

import (
	"context"
	
	"github.com/appointy/notli/microservice4_delivery/internal/template/model"
)

// TemplateAdapter defines the interface for interacting with template providers
type TemplateAdapter interface {
	// GetProviderName returns the identifier for this template provider
	GetProviderName() string
	
	// ListTemplates retrieves a list of templates with optional filtering
	ListTemplates(ctx context.Context, query *model.TemplateQuery) ([]model.Template, error)
	
	// GetTemplate retrieves a single template by ID, including its content
	GetTemplate(ctx context.Context, templateID string) (*model.Template, error)
	
	// GetTemplateVariables returns the variables used in a particular template
	GetTemplateVariables(ctx context.Context, templateID string) ([]model.TemplateVariable, error)
	
	// Close releases any resources used by the adapter
	Close() error
}

// Factory defines a function type that creates a template adapter
type Factory func(config map[string]interface{}) (TemplateAdapter, error)

// Registry maintains a map of available template adapter factories
type Registry struct {
	factories map[string]Factory
}

// NewRegistry creates a new registry of template adapter factories
func NewRegistry() *Registry {
	return &Registry{
		factories: make(map[string]Factory),
	}
}

// Register adds a new template adapter factory to the registry
func (r *Registry) Register(providerName string, factory Factory) {
	r.factories[providerName] = factory
}

// Create instantiates a template adapter for the specified provider
func (r *Registry) Create(providerName string, config map[string]interface{}) (TemplateAdapter, error) {
	factory, exists := r.factories[providerName]
	if !exists {
		return nil, ErrProviderNotFound
	}
	
	return factory(config)
}

// AdapterManager handles multiple template adapters
type AdapterManager struct {
	adapters map[string]TemplateAdapter
	registry *Registry
}

// NewAdapterManager creates a new adapter manager
func NewAdapterManager(registry *Registry) *AdapterManager {
	return &AdapterManager{
		adapters: make(map[string]TemplateAdapter),
		registry: registry,
	}
}

// AddAdapter adds a template adapter to the manager
func (m *AdapterManager) AddAdapter(adapter TemplateAdapter) {
	m.adapters[adapter.GetProviderName()] = adapter
}

// GetAdapter retrieves a template adapter by provider name
func (m *AdapterManager) GetAdapter(providerName string) (TemplateAdapter, bool) {
	adapter, exists := m.adapters[providerName]
	return adapter, exists
}

// GetDefaultAdapter returns the first registered adapter or nil if none exists
func (m *AdapterManager) GetDefaultAdapter() TemplateAdapter {
	for _, adapter := range m.adapters {
		return adapter
	}
	return nil
}

// Close closes all adapters and releases resources
func (m *AdapterManager) Close() error {
	var lastErr error
	for _, adapter := range m.adapters {
		if err := adapter.Close(); err != nil {
			lastErr = err
		}
	}
	return lastErr
}
