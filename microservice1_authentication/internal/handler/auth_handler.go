package handler

import (
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/appointy/notli/microservice1_authentication/internal/config"
	"github.com/appointy/notli/microservice1_authentication/internal/service"
	"github.com/appointy/notli/microservice1_authentication/internal/utils"
	"github.com/gin-gonic/gin"
)

// AuthHandler handles authentication requests
type AuthHandler struct {
	AuthService *service.AuthService
	Config      *config.Config
}

// NewAuthHandler creates a new authentication handler
func NewAuthHandler(authService *service.AuthService, cfg *config.Config) *AuthHandler {
	return &AuthHandler{AuthService: authService, Config: cfg}
}

// --- Structs for Request/Response Binding ---

type RegisterRequest struct {
	Email    string `json:"email" binding:"required,email"`
	Password string `json:"password" binding:"required,min=8"` // Min 8 chars requirement
}

type LoginRequest struct {
	Email    string `json:"email" binding:"required,email"`
	Password string `json:"password" binding:"required"`
	Remember bool   `json:"remember"` // Whether to create a long-lived session
}

type RefreshRequest struct {
	SessionID    string `json:"session_id" binding:"required"`
	RefreshToken string `json:"refresh_token" binding:"required"`
}

type AuthResponse struct {
	Token      string      `json:"token"` // Session ID to use as token
	User       interface{} `json:"user"` // User info without password
	Expiration time.Time   `json:"expiration"` // When the session expires
}

// --- Handler Functions ---

// Register handles user registration
func (h *AuthHandler) Register(c *gin.Context) {
	var req RegisterRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid input: " + err.Error()})
		return
	}

	_, err := h.AuthService.Register(req.Email, req.Password)
	if err != nil {
		// Handle specific errors (email already exists, etc.)
		if strings.Contains(err.Error(), "already registered") {
			c.JSON(http.StatusConflict, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Log the user in and create a session
	loginReq := service.LoginRequest{
		Email:     req.Email,
		Password:  req.Password,
		UserAgent: c.Request.UserAgent(),
		IPAddress: c.ClientIP(),
		Remember:  true, // By default, create long-lived session on registration
	}

	response, err := h.AuthService.Login(c.Request.Context(), loginReq)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create session after registration"})
		return
	}

	// Set session cookies
	utils.SetSessionCookie(c, h.Config, response.SessionID, response.ExpiresAt)
	if response.RefreshToken != "" {
		// Set refresh token cookie with longer expiry
		refreshExpiry := time.Now().Add(time.Duration(h.Config.RefreshTokenExpiryHours) * time.Hour)
		utils.SetRefreshTokenCookie(c, h.Config, response.RefreshToken, refreshExpiry)
	}

	// Create response data with token
	responseData := AuthResponse{
		Token:      response.SessionID,
		User:       response.User,
		Expiration: response.ExpiresAt,
	}
	
	// Debug log
	fmt.Printf("Sending register response: token='%s', expires='%v'\n", responseData.Token, responseData.Expiration)
	
	c.JSON(http.StatusCreated, responseData)
}

// Login handles user login
func (h *AuthHandler) Login(c *gin.Context) {
	var req LoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid input: " + err.Error()})
		return
	}

	loginReq := service.LoginRequest{
		Email:     req.Email,
		Password:  req.Password,
		UserAgent: c.Request.UserAgent(),
		IPAddress: c.ClientIP(),
		Remember:  req.Remember,
	}

	response, err := h.AuthService.Login(c.Request.Context(), loginReq)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": err.Error()}) // Use 401 for login failures
		return
	}

	// Set session cookies
	utils.SetSessionCookie(c, h.Config, response.SessionID, response.ExpiresAt)
	if response.RefreshToken != "" {
		// Set refresh token cookie with longer expiry
		refreshExpiry := time.Now().Add(time.Duration(h.Config.RefreshTokenExpiryHours) * time.Hour)
		utils.SetRefreshTokenCookie(c, h.Config, response.RefreshToken, refreshExpiry)
	}

	c.JSON(http.StatusOK, AuthResponse{
		Token:      response.SessionID,
		User:       response.User,
		Expiration: response.ExpiresAt,
	})
}

// Refresh handles session refresh using a refresh token
func (h *AuthHandler) Refresh(c *gin.Context) {
	var req RefreshRequest
	
	// Try to get session ID and refresh token from cookies first
	sessionID, sessionErr := utils.GetSessionCookie(c)
	refreshToken, refreshErr := utils.GetRefreshTokenCookie(c)
	
	// If either cookie is missing, try to get from request body
	if sessionErr != nil || refreshErr != nil {
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid input: " + err.Error()})
			return
		}
		sessionID = req.SessionID
		refreshToken = req.RefreshToken
	}
	
	// Validate both values are present
	if sessionID == "" || refreshToken == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Session ID and refresh token are required"})
		return
	}

	// Refresh the session
	response, err := h.AuthService.RefreshSession(c.Request.Context(), sessionID, refreshToken)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid or expired session"})
		return
	}

	// Set new session cookies
	utils.SetSessionCookie(c, h.Config, response.SessionID, response.ExpiresAt)
	if response.RefreshToken != "" {
		refreshExpiry := time.Now().Add(time.Duration(h.Config.RefreshTokenExpiryHours) * time.Hour)
		utils.SetRefreshTokenCookie(c, h.Config, response.RefreshToken, refreshExpiry)
	}

	c.JSON(http.StatusOK, AuthResponse{
		Token:      response.SessionID,
		User:       response.User,
		Expiration: response.ExpiresAt,
	})
}

// Logout handles user logout
func (h *AuthHandler) Logout(c *gin.Context) {
	// Get session ID from cookie or bearer token
	sessionID, err := h.AuthService.GetSessionFromRequest(c)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "No active session"})
		return
	}

	// Delete the session
	if err := h.AuthService.Logout(c.Request.Context(), sessionID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to logout"})
		return
	}

	// Clear cookies
	utils.ClearSessionCookies(c, h.Config)

	c.JSON(http.StatusOK, gin.H{"message": "Successfully logged out"})
}
