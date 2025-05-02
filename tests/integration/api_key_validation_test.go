package integration

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"testing"
	"time"
)

// TestAPIKeyValidation tests the API key validation flow between MS1 and MS2
func TestAPIKeyValidation(t *testing.T) {
	config := LoadConfig()

	// Step 1: Register a test user
	token, userID := registerTestUser(t, config)
	t.Logf("Test user registered with ID: %s", userID)
	
	// Step 2: Create an API key for the test user
	apiKey := createAPIKey(t, config, token)
	t.Logf("Created API key: %s...", apiKey[:10])
	
	// Step 3: Validate the API key using MS1's validation endpoint
	validateAPIKeyWithMS1(t, config, apiKey)
	
	// If we've made it this far without errors, the test has passed
	t.Log("API key validation test passed successfully")
}

// validateAPIKeyWithMS1 directly tests the API key validation endpoint in MS1
func validateAPIKeyWithMS1(t *testing.T, config TestConfig, apiKey string) {
	t.Log("Validating API key directly with MS1...")
	
	// Prepare validation request
	reqBody := map[string]string{
		"api_key": apiKey,
	}
	
	reqBodyBytes, err := json.Marshal(reqBody)
	if err != nil {
		t.Fatalf("Failed to marshal validation request: %v", err)
	}
	
	// Make validation request to MS1
	resp, err := http.Post(
		fmt.Sprintf("%s/api/auth/validate", config.AuthServiceURL),
		"application/json",
		bytes.NewBuffer(reqBodyBytes),
	)
	
	if err != nil {
		t.Fatalf("Failed to make validation request: %v", err)
	}
	defer resp.Body.Close()
	
	// Check response
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("Failed to validate API key, status code: %d", resp.StatusCode)
	}
	
	// Parse response
	var result struct {
		Valid  bool   `json:"valid"`
		UserID string `json:"user_id,omitempty"`
		Email  string `json:"email,omitempty"`
		Error  string `json:"error,omitempty"`
	}
	
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		t.Fatalf("Failed to parse validation response: %v", err)
	}
	
	if !result.Valid {
		t.Fatalf("API key validation failed: %s", result.Error)
	}
	
	t.Logf("API key is valid for user: %s", result.Email)
}

// measureEmailDeliveryTime would measure email delivery time in a real environment
// This is a simplified simulation since we can't run full end-to-end testing without Kafka
func measureEmailDeliveryTime(t *testing.T) {
	t.Log("Simulating email delivery time measurement...")

	// In a real test with Kafka running, we would:
	// 1. Record start time before sending notification
	// 2. Use webhooks or polling to detect when email is delivered
	// 3. Calculate the difference
	
	startTime := time.Now()
	
	// Simulate processing delay
	time.Sleep(500 * time.Millisecond)
	
	// Normally, this would be the webhook notification or polling result
	// indicating the email was delivered
	endTime := time.Now()
	
	deliveryTime := endTime.Sub(startTime)
	t.Logf("Simulated email delivery time: %v", deliveryTime)
	
	// In a production environment with real services running and Kafka available,
	// typical email delivery times might range from 500ms to several seconds
	// depending on system load, prioritization, and external email provider
	t.Log("Note: In a real environment, delivery times typically range from 500ms to 5 seconds")
}
