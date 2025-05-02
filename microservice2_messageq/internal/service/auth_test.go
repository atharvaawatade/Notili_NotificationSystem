package service

import (
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestValidateAPIKey_Success(t *testing.T) {
	// Start a local HTTP server
	server := httptest.NewServer(http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
		// Test request parameters
		assert.Equal(t, "/v1/internal/validate-key", req.URL.Path)
		assert.Equal(t, "test-key-success", req.URL.Query().Get("key"))
		assert.Equal(t, "email", req.URL.Query().Get("channel"))
		// Send response to be tested
		rw.WriteHeader(http.StatusOK)
	}))
	// Close the server when test finishes
	defer server.Close()

	// Set the AUTH_SERVICE_URL environment variable to the mock server's URL
	originalURL := os.Getenv("AUTH_SERVICE_URL")
	os.Setenv("AUTH_SERVICE_URL", server.URL)
	t.Cleanup(func() {
		os.Setenv("AUTH_SERVICE_URL", originalURL)
	})

	authClient := NewAuthClient()
	err := authClient.ValidateAPIKey("test-key-success")
	assert.NoError(t, err)
}

func TestValidateAPIKey_Failure_Unauthorized(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
		assert.Equal(t, "test-key-fail", req.URL.Query().Get("key"))
		rw.WriteHeader(http.StatusForbidden) // Simulate Forbidden
	}))
	defer server.Close()

	originalURL := os.Getenv("AUTH_SERVICE_URL")
	os.Setenv("AUTH_SERVICE_URL", server.URL)
	t.Cleanup(func() {
		os.Setenv("AUTH_SERVICE_URL", originalURL)
	})

	authClient := NewAuthClient()
	err := authClient.ValidateAPIKey("test-key-fail")
	assert.Error(t, err)
	assert.EqualError(t, err, "invalid or unauthorized API key")
}

func TestValidateAPIKey_Failure_ServerError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
		assert.Equal(t, "test-key-server-error", req.URL.Query().Get("key"))
		rw.WriteHeader(http.StatusInternalServerError) // Internal Server Error
	}))
	defer server.Close()

	originalURL := os.Getenv("AUTH_SERVICE_URL")
	os.Setenv("AUTH_SERVICE_URL", server.URL)
	t.Cleanup(func() {
		os.Setenv("AUTH_SERVICE_URL", originalURL)
	})

	authClient := NewAuthClient()
	err := authClient.ValidateAPIKey("test-key-server-error")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "auth service error") // unexpected response from authentication service")
}

func TestValidateAPIKey_Failure_ServerDown(t *testing.T) {
	// Don't start a server, or close it immediately
	server := httptest.NewServer(http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {}))
	server.Close() // Ensure server is not running

	originalURL := os.Getenv("AUTH_SERVICE_URL")
	os.Setenv("AUTH_SERVICE_URL", server.URL) // Point to the closed server's URL
	t.Cleanup(func() {
		os.Setenv("AUTH_SERVICE_URL", originalURL)
	})

	authClient := NewAuthClient()
	err := authClient.ValidateAPIKey("test-key-server-down")
	assert.Error(t, err)
	assert.EqualError(t, err, "failed to contact authentication service")
}

func TestValidateAPIKey_NotSet(t *testing.T) {
	// Ensure AUTH_SERVICE_URL is not set
	originalURL := os.Getenv("AUTH_SERVICE_URL")
	os.Unsetenv("AUTH_SERVICE_URL")
	t.Cleanup(func() {
		os.Setenv("AUTH_SERVICE_URL", originalURL)
	})

	// Currently, the function logs a warning and returns nil if URL is not set.
	// Depending on requirements, this might need to return an error.
	authClient := NewAuthClient()
	err := authClient.ValidateAPIKey("test-key-anykey")
	assert.NoError(t, err) // No error because the validation is skipped
}
