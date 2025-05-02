package handler

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
)

// HealthHandler provides health check endpoints
type HealthHandler struct {
	startTime time.Time
}

// NewHealthHandler creates a new health handler
func NewHealthHandler() *HealthHandler {
	return &HealthHandler{
		startTime: time.Now(),
	}
}

// RegisterRoutes registers the handler routes
func (h *HealthHandler) RegisterRoutes(router *gin.Engine) {
	router.GET("/health", h.Health)
	router.GET("/readiness", h.Readiness)
}

// Health handles the health check endpoint
func (h *HealthHandler) Health(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"status":      "UP",
		"service":     "email-delivery-service",
		"version":     "1.0.0",
		"uptime":      time.Since(h.startTime).String(),
		"timestamp":   time.Now(),
	})
}

// Readiness handles the readiness check endpoint
func (h *HealthHandler) Readiness(c *gin.Context) {
	// Can add additional checks here for required dependencies
	// like database, Kafka, or external APIs
	
	c.JSON(http.StatusOK, gin.H{
		"status":    "READY",
		"service":   "email-delivery-service",
		"timestamp": time.Now(),
	})
}
