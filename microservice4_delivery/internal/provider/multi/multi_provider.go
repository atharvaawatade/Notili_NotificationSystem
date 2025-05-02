package multi

import (
	"context"
	"errors"
	"fmt"
	"log"

	"github.com/appointy/notli/microservice4_delivery/internal/model"
	"github.com/appointy/notli/microservice4_delivery/internal/provider"
)

// MultiProvider implements the EmailProvider interface with multiple underlying providers
// It attempts to send emails using providers in order, falling back if one fails
type MultiProvider struct {
	providers []provider.EmailProvider
	names     []string
}

// NewMultiProvider creates a new multi-provider with the given providers
// The order of providers determines priority - first is tried first
func NewMultiProvider(providers ...provider.EmailProvider) *MultiProvider {
	names := make([]string, len(providers))
	for i, p := range providers {
		names[i] = p.Name()
	}

	return &MultiProvider{
		providers: providers,
		names:     names,
	}
}

// Name returns the name of the provider
func (m *MultiProvider) Name() string {
	return "multi-provider"
}

// Send attempts to send an email using the providers in order
// If a provider fails, it logs the error and tries the next provider
func (m *MultiProvider) Send(ctx context.Context, email *model.EmailMessage) (string, error) {
	var lastErr error
	
	for i, provider := range m.providers {
		providerName := m.names[i]
		log.Printf("[INFO] Attempting to send email using provider %s (%d/%d)", 
			providerName, i+1, len(m.providers))
		
		messageID, err := provider.Send(ctx, email)
		if err == nil {
			log.Printf("[INFO] Successfully sent email using provider %s", providerName)
			// Append provider name to message ID to track which provider was used
			return fmt.Sprintf("%s:%s", providerName, messageID), nil
		}
		
		log.Printf("[WARN] Provider %s failed to send email: %v", providerName, err)
		lastErr = err
	}
	
	return "", fmt.Errorf("all providers failed to send email: %w", lastErr)
}

// GetStatus gets the status of an email from the appropriate provider
func (m *MultiProvider) GetStatus(ctx context.Context, messageID string) (*model.EmailStatus, error) {
	// Extract provider name from messageID if it has the format "provider:id"
	for i, providerName := range m.names {
		prefix := providerName + ":"
		if len(messageID) > len(prefix) && messageID[:len(prefix)] == prefix {
			// This message was sent by this provider
			actualID := messageID[len(prefix):]
			return m.providers[i].GetStatus(ctx, actualID)
		}
	}
	
	// If no provider prefix is found, try all providers
	var errs []error
	for _, provider := range m.providers {
		status, err := provider.GetStatus(ctx, messageID)
		if err == nil {
			return status, nil
		}
		errs = append(errs, err)
	}
	
	return nil, fmt.Errorf("no provider could get status for message ID %s", messageID)
}

// Close closes all providers
func (m *MultiProvider) Close() error {
	var errs []error
	
	for i, provider := range m.providers {
		if err := provider.Close(); err != nil {
			errs = append(errs, fmt.Errorf("error closing provider %s: %w", m.names[i], err))
		}
	}
	
	if len(errs) > 0 {
		return errors.Join(errs...)
	}
	return nil
}
