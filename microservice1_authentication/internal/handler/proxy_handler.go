package handler

import (
	"bytes"
	"io"
	"log"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
)

// ProxyHandler handles proxying requests to other services
type ProxyHandler struct {
	client *http.Client
}

// NewProxyHandler creates a new proxy handler
func NewProxyHandler() *ProxyHandler {
	return &ProxyHandler{
		client: &http.Client{
			Timeout: 10 * time.Second,
		},
	}
}

// ProxyEmailRequest forwards email requests to the message queue service
func (h *ProxyHandler) ProxyEmailRequest(c *gin.Context) {
	// Log the received request for debugging
	log.Printf("Received email proxy request")

	// Get the API key from the header
	apiKey := c.GetHeader("X-API-Key")
	if apiKey == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "API key is required"})
		return
	}

	// Read the request body
	body, err := io.ReadAll(c.Request.Body)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to read request body"})
		return
	}

	// Log the body for debugging
	log.Printf("Request body: %s", string(body))

	// Create a new request to the message queue service
	req, err := http.NewRequest("POST", "http://localhost:3001/v1/email/notify", bytes.NewReader(body))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create request"})
		return
	}

	// Forward headers
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-API-Key", apiKey)

	// Make the request
	resp, err := h.client.Do(req)
	if err != nil {
		log.Printf("Error making request to Message Queue service: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to send email: " + err.Error()})
		return
	}
	defer resp.Body.Close()

	// Read the response body
	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to read response body"})
		return
	}

	// Log the response for debugging
	log.Printf("Response from Message Queue service (status %d): %s", resp.StatusCode, string(respBody))

	// Just forward the response as JSON
	c.Header("Content-Type", "application/json")
	c.String(resp.StatusCode, string(respBody))
}
