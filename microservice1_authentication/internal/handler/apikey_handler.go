package handler

import (
	"net/http"
	"strings"

	"github.com/appointy/notli/microservice1_authentication/internal/middleware"
	"github.com/appointy/notli/microservice1_authentication/internal/service"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// APIKeyHandler handles API key related requests
type APIKeyHandler struct {
	APIKeyService *service.APIKeyService
}

// NewAPIKeyHandler creates a new API key handler
func NewAPIKeyHandler(apiKeyService *service.APIKeyService) *APIKeyHandler {
	return &APIKeyHandler{APIKeyService: apiKeyService}
}

// --- Structs ---

type CreateAPIKeyRequest struct {
	Name string `json:"name" binding:"required"`
}

// Represents the response when a key is created (includes the one-time visible key)
type CreateAPIKeyResponse struct {
	KeyID     uuid.UUID `json:"id"`
	Name      string    `json:"name"`
	Prefix    string    `json:"prefix"`
	CreatedAt string    `json:"created_at"` // Use string for simplicity in JSON
	FullKey   string    `json:"apiKey"`     // The actual key, only shown once
}

// Represents an API key when listed (without the full key)
type ListAPIKeyResponse struct {
	ID        uuid.UUID `json:"id"`
	Name      string    `json:"name"`
	Prefix    string    `json:"prefix"`
	LastUsed  *string   `json:"last_used,omitempty"` // Use string pointer
	CreatedAt string    `json:"created_at"`
}

// --- Handler Functions ---

// CreateAPIKey handles the creation of a new API key
func (h *APIKeyHandler) CreateAPIKey(c *gin.Context) {
	userIDValue, exists := c.Get(middleware.ContextUserIDKey)
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User ID not found in context"})
		return
	}
	userID, ok := userIDValue.(uuid.UUID)
	if !ok {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Invalid User ID type in context"})
		return
	}

	var req CreateAPIKeyRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid input: " + err.Error()})
		return
	}

	fullKey, apiKeyModel, err := h.APIKeyService.CreateAPIKey(userID, req.Name)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create API key: " + err.Error()})
		return
	}

	// Return the full key only on creation
	resp := CreateAPIKeyResponse{
		KeyID:     apiKeyModel.ID,
		Name:      apiKeyModel.Name,
		Prefix:    apiKeyModel.Prefix,
		CreatedAt: apiKeyModel.CreatedAt.Format(http.TimeFormat),
		FullKey:   fullKey,
	}
	c.JSON(http.StatusCreated, resp)
}

// GetAPIKeys handles listing API keys for the authenticated user
func (h *APIKeyHandler) GetAPIKeys(c *gin.Context) {
	userIDValue, exists := c.Get(middleware.ContextUserIDKey)
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User ID not found in context"})
		return
	}
	userID, ok := userIDValue.(uuid.UUID)
	if !ok {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Invalid User ID type in context"})
		return
	}

	keys, err := h.APIKeyService.GetAPIKeys(userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to retrieve API keys: " + err.Error()})
		return
	}

	// Format response to exclude sensitive info like KeyHash
	respKeys := make([]ListAPIKeyResponse, len(keys))
	for i, key := range keys {
		var lastUsedStr *string
		if key.LastUsed != nil {
			lu := key.LastUsed.Format(http.TimeFormat)
			lastUsedStr = &lu
		}
		respKeys[i] = ListAPIKeyResponse{
			ID:        key.ID,
			Name:      key.Name,
			Prefix:    key.Prefix,
			LastUsed:  lastUsedStr,
			CreatedAt: key.CreatedAt.Format(http.TimeFormat),
		}
	}

	c.JSON(http.StatusOK, respKeys)
}

// DeleteAPIKey handles the deletion of a specific API key
func (h *APIKeyHandler) DeleteAPIKey(c *gin.Context) {
	userIDValue, exists := c.Get(middleware.ContextUserIDKey)
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User ID not found in context"})
		return
	}
	userID, ok := userIDValue.(uuid.UUID)
	if !ok {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Invalid User ID type in context"})
		return
	}

	keyIDStr := c.Param("keyId") // Get keyId from URL path parameter
	keyID, err := uuid.Parse(keyIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid API Key ID format"})
		return
	}

	err = h.APIKeyService.DeleteAPIKey(keyID, userID)
	if err != nil {
		// Distinguish between not found/permission error vs internal error
		if strings.Contains(err.Error(), "not found or does not belong to user") {
			c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		} else {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete API key: " + err.Error()})
		}
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "API Key deleted successfully"})
}
