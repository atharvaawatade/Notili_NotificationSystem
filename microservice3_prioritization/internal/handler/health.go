package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

// HealthHandler handles health check requests
type HealthHandler struct{}

// NewHealthHandler creates a new health check handler
func NewHealthHandler() *HealthHandler {
	return &HealthHandler{}
}

// RegisterHealthRoutes registers the health check routes
func RegisterHealthRoutes(router *gin.Engine, h *HealthHandler) {
	router.GET("/health", h.HealthCheck)
	router.GET("/readiness", h.ReadinessCheck)
}

// HealthCheck handles the health check endpoint
func (h *HealthHandler) HealthCheck(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"status": "UP",
		"service": "notification-prioritization",
	})
}

// ReadinessCheck handles the readiness check endpoint
func (h *HealthHandler) ReadinessCheck(c *gin.Context) {
	// In a real implementation, this would check connections to dependent services
	c.JSON(http.StatusOK, gin.H{
		"status": "READY",
		"service": "notification-prioritization",
	})
}
