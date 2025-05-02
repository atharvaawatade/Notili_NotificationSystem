package service

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/appointy/notli/microservice1_authentication/internal/config"
	"github.com/appointy/notli/microservice1_authentication/internal/models"
	"github.com/appointy/notli/microservice1_authentication/internal/repository"
	"github.com/appointy/notli/microservice1_authentication/internal/session"
	"github.com/appointy/notli/microservice1_authentication/internal/utils"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// AuthService handles authentication logic
type AuthService struct {
	UserRepo *repository.UserRepository
	Cfg      *config.Config
	Sessions *SessionManager
}

// NewAuthService creates a new authentication service
func NewAuthService(userRepo *repository.UserRepository, cfg *config.Config, sessions *SessionManager) *AuthService {
	return &AuthService{UserRepo: userRepo, Cfg: cfg, Sessions: sessions}
}

// Register creates a new user account
func (s *AuthService) Register(email, password string) (*models.User, error) {
	// Basic validation (more can be added)
	if email == "" || password == "" {
		return nil, fmt.Errorf("email and password cannot be empty")
	}
	
	// Password complexity rules
	if len(password) < 8 {
		return nil, fmt.Errorf("password must be at least 8 characters long")
	}
	
	// TODO: Add more password complexity rules if needed

	hashedPassword, err := utils.HashPassword(password)
	if err != nil {
		return nil, fmt.Errorf("failed to hash password: %w", err)
	}

	user := &models.User{
		Email:    email,
		Password: hashedPassword,
	}

	if err := s.UserRepo.CreateUser(user); err != nil {
		return nil, err // Error from repository (e.g., email exists)
	}

	// Don't return password hash
	user.Password = ""
	return user, nil
}

// LoginRequest contains the login credentials and additional information for creating a session
type LoginRequest struct {
	Email     string
	Password  string
	UserAgent string
	IPAddress string
	Remember  bool // Whether to create a long-lived session with refresh capability
}

// LoginResponse contains the authentication response
type LoginResponse struct {
	User         *models.User
	SessionID    string
	RefreshToken string
	ExpiresAt    time.Time
}

// Login authenticates a user and creates a session
func (s *AuthService) Login(ctx context.Context, req LoginRequest) (*LoginResponse, error) {
	// Validate credentials
	user, err := s.UserRepo.GetUserByEmail(req.Email)
	if err != nil {
		return nil, fmt.Errorf("invalid email or password") // Generic error for security
	}

	if !utils.CheckPasswordHash(req.Password, user.Password) {
		return nil, fmt.Errorf("invalid email or password") // Generic error
	}

	// Create session options
	sessionOpts := &session.SessionOptions{
		IP:            req.IPAddress,
		UserAgent:     req.UserAgent,
		IsRefreshable: req.Remember,
	}

	// Create a new session
	ses, err := s.Sessions.CreateSession(user.ID, user.Email, sessionOpts)
	if err != nil {
		return nil, fmt.Errorf("failed to create session: %w", err)
	}

	// Don't return password hash
	user.Password = ""

	return &LoginResponse{
		User:         user,
		SessionID:    ses.ID,
		RefreshToken: ses.RefreshToken,
		ExpiresAt:    ses.ExpiresAt,
	}, nil
}

// RefreshSession refreshes a user session and returns a new session
func (s *AuthService) RefreshSession(ctx context.Context, sessionID, refreshToken string) (*LoginResponse, error) {
	// Validate and refresh the session
	ses, err := s.Sessions.RefreshSession(ctx, sessionID, refreshToken)
	if err != nil {
		return nil, fmt.Errorf("failed to refresh session: %w", err)
	}

	// Get updated user info
	user, err := s.UserRepo.GetUserByID(ses.UserID)
	if err != nil {
		return nil, fmt.Errorf("user not found: %w", err)
	}

	// Don't return password hash
	user.Password = ""

	return &LoginResponse{
		User:         user,
		SessionID:    ses.ID,
		RefreshToken: ses.RefreshToken,
		ExpiresAt:    ses.ExpiresAt,
	}, nil
}

// Logout ends a user session
func (s *AuthService) Logout(ctx context.Context, sessionID string) error {
	return s.Sessions.DeleteSession(ctx, sessionID)
}

// LogoutAllSessions logs out a user from all devices
func (s *AuthService) LogoutAllSessions(ctx context.Context, userID uuid.UUID) error {
	return s.Sessions.DeleteUserSessions(ctx, userID)
}

// ValidateSession validates a session and updates its last activity time
func (s *AuthService) ValidateSession(ctx context.Context, sessionID string) (*session.Session, error) {
	// Get session
	ses, err := s.Sessions.GetSession(ctx, sessionID)
	if err != nil {
		return nil, err
	}

	// Update activity timestamp
	err = s.Sessions.UpdateActivity(ctx, sessionID)
	if err != nil {
		// Just log the error but don't fail the validation
		fmt.Printf("Failed to update session activity: %v\n", err)
	}

	return ses, nil
}

// GetUserFromSession gets user info from a session
func (s *AuthService) GetUserFromSession(ctx context.Context, sessionID string) (*models.User, error) {
	// Validate session
	ses, err := s.ValidateSession(ctx, sessionID)
	if err != nil {
		return nil, err
	}

	// Get user
	user, err := s.UserRepo.GetUserByID(ses.UserID)
	if err != nil {
		return nil, fmt.Errorf("user not found: %w", err)
	}

	// Don't return password hash
	user.Password = ""
	return user, nil
}

// GetSessionFromRequest attempts to get a session ID from a request (either cookie or Authorization header)
func (s *AuthService) GetSessionFromRequest(c *gin.Context) (string, error) {
	// Try to get session from cookie first
	sessionID, err := utils.GetSessionCookie(c)
	if err == nil && sessionID != "" {
		return sessionID, nil
	}

	// If cookie not found, try Authorization header (Bearer token)
	authHeader := c.GetHeader("Authorization")
	if authHeader != "" {
		const prefix = "Bearer "
		if len(authHeader) > len(prefix) && authHeader[:len(prefix)] == prefix {
			return authHeader[len(prefix):], nil
		}
	}

	return "", http.ErrNoCookie
}
