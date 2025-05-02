package model

import "time"

// NotificationMessage represents a notification message being processed
type NotificationMessage struct {
	IdempotencyKey string                 `json:"idempotency_key"`
	Recipient      string                 `json:"recipient"`
	TemplateID     string                 `json:"template_id"`
	Priority       string                 `json:"priority"`       // "high", "normal", "low"
	MessageType    string                 `json:"message_type"`   // "transactional", "promotional"
	Channel        string                 `json:"channel"`        // "email", "sms", "push", etc.
	ChannelData    map[string]interface{} `json:"channel_data"`
	CreatedAt      time.Time              `json:"created_at"`
	UserTier       string                 `json:"user_tier"`      // "premium", "standard", "free"
	PriorityScore  int                    `json:"priority_score"` // Calculated priority score
}

// PrioritizedMessage is a message with priority information for the queue
type PrioritizedMessage struct {
	Message       *NotificationMessage
	PriorityScore int       // Higher numbers = higher priority
	QueuedAt      time.Time // When the message was added to the priority queue
}

// RateLimitInfo contains information used for rate limiting decisions
type RateLimitInfo struct {
	UserID     string
	MessageType string
	Channel    string
	Timestamp  time.Time
}
