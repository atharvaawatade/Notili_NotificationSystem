package model

import (
	"time"
)

// EmailMessage represents a structured email to be sent through a provider
type EmailMessage struct {
	// Core email fields
	From        string   `json:"from"`
	To          []string `json:"to"`
	Cc          []string `json:"cc,omitempty"`
	Bcc         []string `json:"bcc,omitempty"`
	Subject     string   `json:"subject"`
	PlainBody   string   `json:"plain_body,omitempty"`
	HtmlBody    string   `json:"html_body,omitempty"`
	
	// Template fields
	TemplateID  string                 `json:"template_id,omitempty"`
	TemplateData map[string]interface{} `json:"template_data,omitempty"`
	
	// Additional fields
	Attachments []Attachment          `json:"attachments,omitempty"`
	Headers     map[string]string     `json:"headers,omitempty"`
	
	// Metadata
	IdempotencyKey string              `json:"idempotency_key"`
	Priority       int                 `json:"priority"`
	MessageType    string              `json:"message_type"` // "transactional" or "promotional"
	Channel        string              `json:"channel"`     // "email", "sms", "push"
	UserID         string              `json:"user_id,omitempty"`
	UserTier       string              `json:"user_tier,omitempty"` // "free", "standard", "premium"
}

// Attachment represents a file attached to an email
type Attachment struct {
	Filename    string `json:"filename"`
	ContentType string `json:"content_type"`
	Content     []byte `json:"content"`
}

// EmailStatus represents the current status of an email delivery
type EmailStatus struct {
	MessageID    string    `json:"message_id"`
	Status       string    `json:"status"` // "queued", "sent", "delivered", "failed", "bounced", "opened", "clicked"
	Provider     string    `json:"provider"`
	Details      string    `json:"details,omitempty"`
	ErrorMessage string    `json:"error_message,omitempty"`
	Timestamp    time.Time `json:"timestamp"`
}

// DeliveryRecord represents an email delivery in the database
type DeliveryRecord struct {
	MessageID      string    `json:"message_id"`
	IdempotencyKey string    `json:"idempotency_key"`
	Recipient      string    `json:"recipient"`
	Sender         string    `json:"sender"`
	Subject        string    `json:"subject"`
	TemplateID     string    `json:"template_id,omitempty"`
	Provider       string    `json:"provider"`
	Status         string    `json:"status"`
	StatusDetails  string    `json:"status_details,omitempty"`
	CreatedAt      time.Time `json:"created_at"`
	UpdatedAt      time.Time `json:"updated_at"`
	RetryCount     int       `json:"retry_count"`
	NextRetryAt    time.Time `json:"next_retry_at,omitempty"`
}

// StatusEvent represents a status change event for an email
type StatusEvent struct {
	ID          int64     `json:"id"`
	MessageID   string    `json:"message_id"`
	Status      string    `json:"status"`
	Provider    string    `json:"provider"`
	Details     string    `json:"details,omitempty"`
	OccurredAt  time.Time `json:"occurred_at"`
	RecordedAt  time.Time `json:"recorded_at"`
}

// KafkaMessage represents a message consumed from Kafka
type KafkaMessage struct {
	IdempotencyKey string                 `json:"idempotency_key"`
	Recipient      string                 `json:"recipient"`
	TemplateID     string                 `json:"template_id,omitempty"`
	Priority       string                 `json:"priority"`      // Changed from int to string to match M3's format
	PriorityScore  int                    `json:"priority_score"` // The actual numeric priority
	MessageType    string                 `json:"message_type"`
	Channel        string                 `json:"channel"`
	ChannelData    map[string]interface{} `json:"channel_data"`
	UserID         string                 `json:"user_id,omitempty"`
	UserTier       string                 `json:"user_tier,omitempty"`
	CreatedAt      time.Time              `json:"created_at,omitempty"`
}
