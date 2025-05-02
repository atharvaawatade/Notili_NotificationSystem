package handler

import (
	"context"
	"log"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/appointy/notli/microservice4_delivery/internal/model"
	"github.com/appointy/notli/microservice4_delivery/internal/service"
)

// EmailHandler handles email-related HTTP requests
type EmailHandler struct {
	emailService *service.EmailService
}

// NewEmailHandler creates a new email handler
func NewEmailHandler(emailService *service.EmailService) *EmailHandler {
	return &EmailHandler{
		emailService: emailService,
	}
}

// RegisterRoutes registers the handler routes
func (h *EmailHandler) RegisterRoutes(router *gin.Engine) {
	v1 := router.Group("/v1/email")
	{
		v1.GET("/status/:messageId", h.GetEmailStatus)
		v1.POST("/webhook", h.HandleWebhook)
	}
}

// GetEmailStatus handles the email status endpoint
func (h *EmailHandler) GetEmailStatus(c *gin.Context) {
	messageID := c.Param("messageId")
	if messageID == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "message_id is required",
		})
		return
	}

	// Create a context with timeout
	ctx, cancel := context.WithTimeout(c.Request.Context(), 5*time.Second)
	defer cancel()

	// Get email status
	status, err := h.emailService.GetEmailStatus(ctx, messageID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": err.Error(),
		})
		return
	}

	if status == nil {
		c.JSON(http.StatusNotFound, gin.H{
			"error": "email not found",
		})
		return
	}

	c.JSON(http.StatusOK, status)
}

// WebhookRequest represents a webhook request from the email provider
type WebhookRequest struct {
	MessageID  string            `json:"message_id"`
	Event      string            `json:"event"`
	Timestamp  time.Time         `json:"timestamp"`
	Recipient  string            `json:"recipient"`
	Metadata   map[string]string `json:"metadata"`
	Reason     string            `json:"reason,omitempty"`
}

// HandleWebhook processes webhook callbacks from email providers
func (h *EmailHandler) HandleWebhook(c *gin.Context) {
	var request WebhookRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "invalid request body",
		})
		return
	}

	// Map webhook event to internal status
	status := mapEventToStatus(request.Event)
	
	// Create status object
	emailStatus := &model.EmailStatus{
		MessageID: request.MessageID,
		Status:    status,
		Provider:  "maileroo", // We can enhance this to detect provider from headers or request
		Details:   request.Reason,
		Timestamp: request.Timestamp,
	}

	// Create a context with timeout
	ctx, cancel := context.WithTimeout(c.Request.Context(), 5*time.Second)
	defer cancel()

	// Log the status update
	log.Printf("Webhook received for message %s: %s", request.MessageID, status)
	
	// In a real implementation, we would update the status in our database
	// For now, we'll just log the status update
	log.Printf("Status update for message %s: %+v", emailStatus.MessageID, emailStatus)
	
	// Simulate checking if message exists in our system
	exists, err := messageExistsInSystem(ctx, request.MessageID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to process webhook",
		})
		return
	}
	
	if !exists {
		log.Printf("Message ID %s not found in our system", request.MessageID)
		// Still return OK to the webhook sender
	}

	// TODO: Implement proper webhook handling
	// This would involve calling repository methods to update the status

	c.JSON(http.StatusOK, gin.H{
		"status": "accepted",
	})
}

// mapEventToStatus maps webhook events to internal status values
func mapEventToStatus(event string) string {
	switch event {
	case "sent":
		return "sent"
	case "delivered":
		return "delivered"
	case "opened":
		return "opened"
	case "clicked":
		return "clicked"
	case "bounced":
		return "bounced"
	case "complained":
		return "complained"
	case "unsubscribed":
		return "unsubscribed"
	case "failed":
		return "failed"
	default:
		return "unknown"
	}
}

// messageExistsInSystem checks if a message ID exists in our system
// In a production implementation, this would check the database
func messageExistsInSystem(ctx context.Context, messageID string) (bool, error) {
	// This is a mock implementation for demonstration purposes
	// In a real implementation, we would check the database
	
	// Simulate a delay for database lookup
	time.Sleep(50 * time.Millisecond)
	
	// For demonstration, we'll assume messages with odd-length IDs exist
	return len(messageID) % 2 == 1, nil
}
