package handler

import (
	"net/http"

	"github.com/appointy/notli/microservice1_authentication/internal/service"
	"github.com/gin-gonic/gin"
)

// ValidateHandler handles API key validation requests
type ValidateHandler struct {
	APIKeyService *service.APIKeyService
}

// NewValidateHandler creates a new validation handler
func NewValidateHandler(apiKeyService *service.APIKeyService) *ValidateHandler {
	return &ValidateHandler{APIKeyService: apiKeyService}
}

// APIKeyRequest represents the structure of an API key validation request
type APIKeyRequest struct {
	APIKey string `json:"api_key" binding:"required"`
}

// APIKeyValidationResponse represents the response for API key validation
type APIKeyValidationResponse struct {
	Valid  bool   `json:"valid"`
	UserID string `json:"user_id,omitempty"`
	Email  string `json:"email,omitempty"`
	Error  string `json:"error,omitempty"`
}

// ValidateAPIKey handles API key validation requests
func (h *ValidateHandler) ValidateAPIKey(c *gin.Context) {
	var req APIKeyRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, APIKeyValidationResponse{
			Valid: false,
			Error: "Invalid request format: " + err.Error(),
		})
		return
	}

	valid, user, err := h.APIKeyService.ValidateAPIKey(req.APIKey)
	if err != nil {
		c.JSON(http.StatusOK, APIKeyValidationResponse{
			Valid: false,
			Error: err.Error(),
		})
		return
	}

	if !valid || user == nil {
		c.JSON(http.StatusOK, APIKeyValidationResponse{
			Valid: false,
			Error: "Invalid API key",
		})
		return
	}

	c.JSON(http.StatusOK, APIKeyValidationResponse{
		Valid:  true,
		UserID: user.ID.String(),
		Email:  user.Email,
	})
}
