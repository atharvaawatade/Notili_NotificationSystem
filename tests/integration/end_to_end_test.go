package integration

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"testing"
	"time"
	
	"github.com/google/uuid"
)

// TestConfig holds all the necessary configuration for the integration tests
type TestConfig struct {
	AuthServiceURL       string
	MessageQueueURL      string
	PrioritizationURL    string
	DeliveryServiceURL   string
	TestEmailAddress     string
	RegistrationEmail    string
	RegistrationPassword string
}

// LoadConfig loads test configuration from environment variables with defaults
func LoadConfig() TestConfig {
	return TestConfig{
		AuthServiceURL:       getEnvWithDefault("AUTH_SERVICE_URL", "http://localhost:3000"),
		MessageQueueURL:      getEnvWithDefault("MESSAGE_QUEUE_URL", "http://localhost:3001"),
		PrioritizationURL:    getEnvWithDefault("PRIORITIZATION_URL", "http://localhost:3002"),
		DeliveryServiceURL:   getEnvWithDefault("DELIVERY_SERVICE_URL", "http://localhost:3003"),
		TestEmailAddress:     getEnvWithDefault("TEST_EMAIL_ADDRESS", "atharvaawatade@gmail.com"),
		RegistrationEmail:    getEnvWithDefault("REGISTRATION_EMAIL", "test_user@example.com"),
		RegistrationPassword: getEnvWithDefault("REGISTRATION_PASSWORD", "Password123!"),
	}
}

func getEnvWithDefault(key, defaultValue string) string {
	if value, exists := os.LookupEnv(key); exists {
		return value
	}
	return defaultValue
}

// TestEndToEnd performs a full end-to-end test of the notification system
func TestEndToEnd(t *testing.T) {
	config := LoadConfig()

	// Step 1: Register a test user
	token, userID := registerTestUser(t, config)
	t.Logf("Using user with ID: %s for testing", userID)
	
	// Step 2: Create an API key for the test user
	apiKey := createAPIKey(t, config, token)
	
	// Step 3: Use the API key to send a notification
	sendNotification(t, config, apiKey)
	
	// Step 4: Wait for processing (in a real test, we might poll or use webhooks)
	t.Log("Waiting for notification to be processed and delivered...")
	time.Sleep(10 * time.Second)
	
	t.Log("End-to-end test completed successfully.")
	t.Log("Please check the inbox of", config.TestEmailAddress, "for the test notification.")
}

// registerTestUser creates a test user account
func registerTestUser(t *testing.T, config TestConfig) (string, string) {
	t.Log("Step 1: Registering test user...")
	
	// Prepare registration request
	reqBody := map[string]string{
		"email":    config.RegistrationEmail,
		"password": "StrongPassword123!",  // Ensure this meets minimum password requirements (at least 8 chars)
	}
	
	reqBodyBytes, err := json.Marshal(reqBody)
	if err != nil {
		t.Fatalf("Failed to marshal registration request: %v", err)
	}
	
	// Make registration request
	resp, err := http.Post(
		fmt.Sprintf("%s/api/auth/signup", config.AuthServiceURL),
		"application/json",
		bytes.NewBuffer(reqBodyBytes),
	)
	
	// Log detailed request information for debugging
	t.Logf("Registration request URL: %s/api/auth/signup", config.AuthServiceURL)
	t.Logf("Registration request body: %s", string(reqBodyBytes))
	
	if err != nil {
		t.Fatalf("Failed to make registration request: %v", err)
	}
	defer resp.Body.Close()
	
	// Check response
	if resp.StatusCode != http.StatusCreated && resp.StatusCode != http.StatusOK {
		t.Fatalf("Failed to register user, status code: %d", resp.StatusCode)
	}
	
	// Parse response
	var result struct {
		Token string      `json:"token"`
		User  interface{} `json:"user"`
	}
	
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		t.Fatalf("Failed to parse registration response: %v", err)
	}
	
	// Extract user ID if possible
	userID := "unknown"
	if user, ok := result.User.(map[string]interface{}); ok {
		if id, ok := user["id"].(string); ok {
			userID = id
		}
	}
	
	t.Logf("Test user registered with ID: %s", userID)
	return result.Token, userID
}

// createAPIKey creates an API key for the test user
func createAPIKey(t *testing.T, config TestConfig, token string) string {
	t.Log("Step 2: Creating API key...")
	
	// Prepare API key request
	reqBody := map[string]string{
		"name": "Integration Test Key",
	}
	
	reqBodyBytes, err := json.Marshal(reqBody)
	if err != nil {
		t.Fatalf("Failed to marshal API key request: %v", err)
	}
	
	// Create request
	req, err := http.NewRequest(
		"POST",
		fmt.Sprintf("%s/api/keys/", config.AuthServiceURL),
		bytes.NewBuffer(reqBodyBytes),
	)
	if err != nil {
		t.Fatalf("Failed to create API key request: %v", err)
	}
	
	// Set headers
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", token))
	
	// Make request
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		t.Fatalf("Failed to make API key request: %v", err)
	}
	defer resp.Body.Close()
	
	// Check response
	if resp.StatusCode != http.StatusCreated && resp.StatusCode != http.StatusOK {
		t.Fatalf("Failed to create API key, status code: %d", resp.StatusCode)
	}
	
	// Parse response
	var result struct {
		KeyID     string `json:"id"`
		Name      string `json:"name"`
		Prefix    string `json:"prefix"`
		CreatedAt string `json:"created_at"`
		FullKey   string `json:"apiKey"`
	}
	
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		t.Fatalf("Failed to parse API key response: %v", err)
	}
	
	t.Logf("API key created with prefix: %s", result.Prefix)
	
	return result.FullKey
}

// sendNotification sends a test notification using the API key
func sendNotification(t *testing.T, config TestConfig, apiKey string) {
	t.Log("Step 3: Sending notification...")
	
	// Prepare notification request according to MS2's expected format (EmailRequest struct)
	// Generate a valid UUID for idempotency key
	idempotencyKey := uuid.New().String()
	
	// Create request matching the EmailRequest struct in MS2
	reqBody := map[string]interface{}{
		"recipient":      config.TestEmailAddress,
		"subject":        "NOTLI Integration Test",
		"body":           "hii test",
		"from_name":      "NOTLI Test",
		"idempotency_key": idempotencyKey,
	}
	
	reqBodyBytes, err := json.Marshal(reqBody)
	if err != nil {
		t.Fatalf("Failed to marshal notification request: %v", err)
	}
	
	// Create request
	req, err := http.NewRequest(
		"POST",
		fmt.Sprintf("%s/v1/email/notify", config.MessageQueueURL),
		bytes.NewBuffer(reqBodyBytes),
	)
	if err != nil {
		t.Fatalf("Failed to create notification request: %v", err)
	}
	
	// Set headers
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Api-Key", apiKey)
	
	// Make request
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		t.Fatalf("Failed to make notification request: %v", err)
	}
	defer resp.Body.Close()
	
	// Check response
	if resp.StatusCode != http.StatusAccepted && resp.StatusCode != http.StatusOK {
		t.Fatalf("Failed to send notification, status code: %d", resp.StatusCode)
	}
	
	t.Log("Notification request sent successfully.")
}
