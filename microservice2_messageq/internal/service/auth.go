package service

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"os"
	"time"
)

// AuthClient defines the interface for validating API keys.
type AuthClient interface {
	ValidateAPIKey(apiKey string) error
}

// NewAuthClient creates a new instance of the default AuthClient.
func NewAuthClient() AuthClient {
	return &defaultAuthClient{
		httpClient: &http.Client{
			Timeout: 5 * time.Second,
		},
	}
}

// defaultAuthClient implements the AuthClient interface.
type defaultAuthClient struct {
	httpClient *http.Client
}

// ValidateAPIKey checks the provided API key against the authentication service (MS1).
func (c *defaultAuthClient) ValidateAPIKey(apiKey string) error {
	authServiceURL := os.Getenv("AUTH_SERVICE_URL")
	if authServiceURL == "" {
		// Default to localhost:3000 if AUTH_SERVICE_URL is not set
		authServiceURL = "http://localhost:3000"
		fmt.Println("WARN: AUTH_SERVICE_URL not set, using default: http://localhost:3000")
	}

	// Use the new validation endpoint we created in MS1
	validationURL := fmt.Sprintf("%s/api/auth/validate", authServiceURL)

	// Create request body with the API key
	reqBody := map[string]string{"api_key": apiKey}
	reqBodyBytes, err := json.Marshal(reqBody)
	if err != nil {
		fmt.Printf("Error marshaling request body: %v\n", err)
		return errors.New("internal error creating auth request")
	}
	
	req, err := http.NewRequestWithContext(
		context.Background(), 
		"POST", 
		validationURL, 
		bytes.NewBuffer(reqBodyBytes),
	)
	if err != nil {
		fmt.Printf("Error creating request to auth service: %v\n", err)
		return errors.New("internal error creating auth request")
	}
	
	// Set content type header
	req.Header.Set("Content-Type", "application/json")

	// Add any necessary headers if MS1 requires them (e.g., internal service token)
	// req.Header.Add("X-Internal-Auth", "some_secret")

	resp, err := c.httpClient.Do(req) // Use client from struct
	if err != nil {
		fmt.Printf("Error connecting to auth service: %v\n", err)
		return errors.New("auth service unavailable")
	}
	defer resp.Body.Close()

	// Parse the response
	var result struct {
		Valid  bool   `json:"valid"`
		UserID string `json:"user_id,omitempty"`
		Email  string `json:"email,omitempty"`
		Error  string `json:"error,omitempty"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		fmt.Printf("Error parsing auth response: %v\n", err)
		return errors.New("auth service returned invalid response")
	}

	if !result.Valid {
		error := "invalid API key"
		if result.Error != "" {
			error = result.Error
		}
		return errors.New(error)
	}

	return nil
}

// min function removed as it's no longer used
