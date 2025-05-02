package middleware

import (
	"fmt"
	"net/http"

	"github.com/appointy/notli/microservice1_authentication/internal/service"
	"github.com/appointy/notli/microservice1_authentication/internal/utils"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

const (
	AuthorizationHeaderKey  = "Authorization"
	AuthorizationTypeBearer = "Bearer"
	ContextUserIDKey        = "userID"
	ContextUserEmailKey     = "userEmail"
	ContextSessionIDKey     = "sessionID"
)

// AuthMiddleware creates a Gin middleware for session-based authentication
func AuthMiddleware(authService *service.AuthService) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Get session ID from cookie or authorization header
		sessionID, err := authService.GetSessionFromRequest(c)
		if err != nil {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "Authentication required"})
			return
		}

		// Validate session and get user information
		session, err := authService.ValidateSession(c.Request.Context(), sessionID)
		if err != nil {
			// Handle specific session errors
			errMsg := "Invalid or expired session"
			switch err.Error() {
			case "session not found":
				errMsg = "Session not found"
			case "session expired":
				errMsg = "Session has expired, please login again"
			}
			
			// Clear cookies on session error
			utils.ClearSessionCookies(c, authService.Cfg)
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": errMsg})
			return
		}

		// Set user information in the context for route handlers to use
		c.Set(ContextUserIDKey, session.UserID)
		c.Set(ContextUserEmailKey, session.Email)
		c.Set(ContextSessionIDKey, sessionID)

		// Proceed to the next handler
		c.Next()
	}
}

// GetCurrentUserID extracts the user ID from the context (convenience function)
func GetCurrentUserID(c *gin.Context) (uuid.UUID, error) {
	userID, exists := c.Get(ContextUserIDKey)
	if !exists {
		return uuid.Nil, fmt.Errorf("user ID not found in context")
	}

	id, ok := userID.(uuid.UUID)
	if !ok {
		return uuid.Nil, fmt.Errorf("user ID has invalid type")
	}

	return id, nil
}

// GetCurrentSessionID extracts the session ID from the context (convenience function)
func GetCurrentSessionID(c *gin.Context) (string, error) {
	sessionID, exists := c.Get(ContextSessionIDKey)
	if !exists {
		return "", fmt.Errorf("session ID not found in context")
	}

	id, ok := sessionID.(string)
	if !ok {
		return "", fmt.Errorf("session ID has invalid type")
	}

	return id, nil
}

// GetCurrentEmail extracts the user email from the context (convenience function)
func GetCurrentEmail(c *gin.Context) (string, error) {
	email, exists := c.Get(ContextUserEmailKey)
	if !exists {
		return "", fmt.Errorf("user email not found in context")
	}

	emailStr, ok := email.(string)
	if !ok {
		return "", fmt.Errorf("user email has invalid type")
	}

	return emailStr, nil
}
