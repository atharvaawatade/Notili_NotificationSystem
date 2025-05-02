package service

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/appointy/notli/microservice1_authentication/internal/config"
	"github.com/appointy/notli/microservice1_authentication/internal/session"
	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
)

// SessionManager manages user sessions using Redis
type SessionManager struct {
	client                 *redis.Client
	sessionDuration        time.Duration
	maxInactivityDuration  time.Duration
	refreshTokenExpiration time.Duration
	keyPrefix              string
	Cfg                    *config.Config // Make config available to methods
}

// NewSessionManager creates a new session manager
func NewSessionManager(cfg *config.Config) (*SessionManager, error) {
	// Initialize Redis client
	client := redis.NewClient(&redis.Options{
		Addr:     fmt.Sprintf("%s:%s", cfg.RedisHost, cfg.RedisPort),
		Password: cfg.RedisPassword,
		DB:       0,
	})

	// Test connection
	ctx := context.Background()
	if err := client.Ping(ctx).Err(); err != nil {
		return nil, fmt.Errorf("failed to connect to Redis: %w", err)
	}

	return &SessionManager{
		client:                 client,
		sessionDuration:        time.Duration(cfg.SessionExpiryHours) * time.Hour,
		maxInactivityDuration:  time.Duration(cfg.InactivityTimeoutMinutes) * time.Minute,
		refreshTokenExpiration: time.Duration(cfg.RefreshTokenExpiryHours) * time.Hour,
		keyPrefix:              "session:",
		Cfg:                    cfg,
	}, nil
}

// CreateSession creates a new session for a user
func (sm *SessionManager) CreateSession(userID uuid.UUID, email string, opts *session.SessionOptions) (*session.Session, error) {
	sessionID := uuid.New().String()
	now := time.Now()

	// Generate refresh token if session is refreshable
	var refreshToken string
	if opts != nil && opts.IsRefreshable {
		refreshToken = uuid.New().String()
	}

	sess := &session.Session{
		ID:            sessionID,
		UserID:        userID,
		Email:         email,
		CreatedAt:     now,
		LastActivity:  now,
		ExpiresAt:     now.Add(sm.sessionDuration),
		RefreshToken:  refreshToken,
		IsRefreshable: opts != nil && opts.IsRefreshable,
	}

	// Set optional fields if provided
	if opts != nil {
		sess.UserAgent = opts.UserAgent
		sess.IP = opts.IP
	}

	// Save to Redis
	return sess, sm.saveSession(sess)
}

// GetSession retrieves a session by its ID
func (sm *SessionManager) GetSession(ctx context.Context, sessionID string) (*session.Session, error) {
	key := sm.keyPrefix + sessionID
	data, err := sm.client.Get(ctx, key).Bytes()
	if err != nil {
		if err == redis.Nil {
			return nil, session.ErrSessionNotFound
		}
		return nil, err
	}

	var sess session.Session
	if err := json.Unmarshal(data, &sess); err != nil {
		return nil, session.ErrInvalidSession
	}

	// Check if session has expired
	now := time.Now()
	if sess.ExpiresAt.Before(now) {
		// Clean up expired session
		sm.DeleteSession(ctx, sessionID)
		return nil, session.ErrSessionExpired
	}

	// Check inactivity timeout
	if sm.maxInactivityDuration > 0 && sess.LastActivity.Add(sm.maxInactivityDuration).Before(now) {
		sm.DeleteSession(ctx, sessionID)
		return nil, session.ErrSessionExpired
	}

	return &sess, nil
}

// UpdateActivity updates the last activity time for a session
func (sm *SessionManager) UpdateActivity(ctx context.Context, sessionID string) error {
	sess, err := sm.GetSession(ctx, sessionID)
	if err != nil {
		return err
	}

	sess.LastActivity = time.Now()
	return sm.saveSession(sess)
}

// RefreshSession extends a session's expiration and returns a new session ID
func (sm *SessionManager) RefreshSession(ctx context.Context, sessionID, refreshToken string) (*session.Session, error) {
	sess, err := sm.GetSession(ctx, sessionID)
	if err != nil {
		return nil, err
	}

	// Verify refresh token and check if session is refreshable
	if !sess.IsRefreshable || sess.RefreshToken != refreshToken {
		return nil, session.ErrInvalidRefreshToken
	}

	// Delete old session
	if err := sm.DeleteSession(ctx, sessionID); err != nil {
		return nil, err
	}

	// Create new session with same user data but new IDs and updated times
	now := time.Now()
	newSession := &session.Session{
		ID:            uuid.New().String(),
		UserID:        sess.UserID,
		Email:         sess.Email,
		UserAgent:     sess.UserAgent,
		IP:            sess.IP,
		LastActivity:  now,
		CreatedAt:     now,
		ExpiresAt:     now.Add(sm.sessionDuration),
		RefreshToken:  uuid.New().String(),
		IsRefreshable: sess.IsRefreshable,
	}

	// Save new session
	if err := sm.saveSession(newSession); err != nil {
		return nil, err
	}

	return newSession, nil
}

// DeleteSession removes a session
func (sm *SessionManager) DeleteSession(ctx context.Context, sessionID string) error {
	key := sm.keyPrefix + sessionID
	return sm.client.Del(ctx, key).Err()
}

// DeleteUserSessions removes all sessions for a user
func (sm *SessionManager) DeleteUserSessions(ctx context.Context, userID uuid.UUID) error {
	// Get all session keys
	pattern := sm.keyPrefix + "*"
	keys, err := sm.client.Keys(ctx, pattern).Result()
	if err != nil {
		return err
	}

	// Check each session and delete those belonging to the user
	for _, key := range keys {
		data, err := sm.client.Get(ctx, key).Bytes()
		if err != nil {
			continue
		}

		var sess session.Session
		if err := json.Unmarshal(data, &sess); err != nil {
			continue
		}

		if sess.UserID == userID {
			sm.client.Del(ctx, key)
		}
	}

	return nil
}

// saveSession saves a session to Redis
func (sm *SessionManager) saveSession(sess *session.Session) error {
	data, err := json.Marshal(sess)
	if err != nil {
		return err
	}

	key := sm.keyPrefix + sess.ID
	ctx := context.Background()
	
	// Calculate time until expiration
	expiration := sess.ExpiresAt.Sub(time.Now())
	if expiration <= 0 {
		return session.ErrSessionAlreadyExpired
	}

	// Save to Redis with expiration
	return sm.client.Set(ctx, key, data, expiration).Err()
}
