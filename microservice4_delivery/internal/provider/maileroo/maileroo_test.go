package maileroo

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/appointy/notli/microservice4_delivery/internal/model"
	"github.com/stretchr/testify/assert"
)

func TestMailerooProvider_Send(t *testing.T) {
	// Create a mock server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Check request method
		assert.Equal(t, "POST", r.Method)
		
		// Check authorization header
		assert.Equal(t, "Bearer test-api-key", r.Header.Get("Authorization"))
		
		// Check content type
		assert.Equal(t, "application/json", r.Header.Get("Content-Type"))
		
		// Return a mock response
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{
			"message_id": "test-message-id-123",
			"status": "queued",
			"message": "Email has been queued for delivery"
		}`))
	}))
	defer server.Close()
	
	// Create a provider with test credentials
	provider, err := NewMailerooProvider(
		"test-api-key",
		"test-domain.maileroo.org",
		"test@test-domain.maileroo.org",
		"Test Sender",
	)
	assert.NoError(t, err)
	
	// Set test server URL
	provider.WithBaseURL(server.URL)
	
	// Create a test email
	email := &model.EmailMessage{
		To:        []string{"recipient@example.com"},
		Subject:   "Test Subject",
		PlainBody: "This is a test email",
		HtmlBody:  "<p>This is a test email</p>",
		Headers:   map[string]string{"X-Test": "test-value"},
	}
	
	// Send the email
	messageID, err := provider.Send(context.Background(), email)
	assert.NoError(t, err)
	assert.Equal(t, "test-message-id-123", messageID)
}

func TestMailerooProvider_GetStatus(t *testing.T) {
	// Create a mock server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Check request method
		assert.Equal(t, "GET", r.Method)
		
		// Check authorization header
		assert.Equal(t, "Bearer test-api-key", r.Header.Get("Authorization"))
		
		// Return a mock response
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{
			"message_id": "test-message-id-123",
			"status": "delivered",
			"message": "Email has been delivered",
			"delivered_at": "2023-06-15T12:34:56Z"
		}`))
	}))
	defer server.Close()
	
	// Create a provider with test credentials
	provider, err := NewMailerooProvider(
		"test-api-key",
		"test-domain.maileroo.org",
		"test@test-domain.maileroo.org",
		"Test Sender",
	)
	assert.NoError(t, err)
	
	// Set test server URL
	provider.WithBaseURL(server.URL)
	
	// Get the status of a test email
	status, err := provider.GetStatus(context.Background(), "test-message-id-123")
	assert.NoError(t, err)
	assert.Equal(t, "delivered", status.Status)
	assert.Equal(t, "maileroo", status.Provider)
	assert.Equal(t, "maileroo", status.Provider)
}

func TestMailerooProvider_InvalidConfig(t *testing.T) {
	// Try to create a provider with an empty API key
	provider, err := NewMailerooProvider("", "domain", "from", "name")
	assert.Error(t, err)
	assert.Nil(t, provider)
}
