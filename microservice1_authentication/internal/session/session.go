package session

import (
	"errors"
	"time"

	"github.com/google/uuid"
)

// Common errors
var (
	ErrSessionNotFound     = errors.New("session not found")
	ErrSessionExpired      = errors.New("session expired")
	ErrInvalidSession      = errors.New("invalid session")
	ErrInvalidRefreshToken = errors.New("invalid refresh token")
	ErrSessionAlreadyExpired = errors.New("session already expired")
)

// Session represents a user session
type Session struct {
	ID            string    `json:"id"`
	UserID        uuid.UUID `json:"user_id"`
	Email         string    `json:"email"`
	UserAgent     string    `json:"user_agent"`
	IP            string    `json:"ip"`
	LastActivity  time.Time `json:"last_activity"`
	CreatedAt     time.Time `json:"created_at"`
	ExpiresAt     time.Time `json:"expires_at"`
	RefreshToken  string    `json:"refresh_token,omitempty"`
	IsRefreshable bool      `json:"is_refreshable"`
}

// SessionOptions contains optional configuration for creating a session
type SessionOptions struct {
	IP            string
	UserAgent     string
	IsRefreshable bool
}
