package priority

import (
	"time"

	"github.com/appointy/notli/microservice3_prioritization/internal/model"
)

// RuleEngine handles prioritization logic for notification messages
type RuleEngine struct {}

// NewRuleEngine creates a new rule engine
func NewRuleEngine() *RuleEngine {
	return &RuleEngine{}
}

// CalculatePriorityScore calculates the priority score for a notification
func (r *RuleEngine) CalculatePriorityScore(notification *model.NotificationMessage) int {
	// Apply different scoring strategies based on message type
	if notification.MessageType == "transactional" {
		return r.calculateTransactionalScore(notification)
	} else {
		return r.calculatePromotionalScore(notification)
	}
}

// calculateTransactionalScore calculates score for transactional messages
func (r *RuleEngine) calculateTransactionalScore(notification *model.NotificationMessage) int {
	// Base score from priority field
	var baseScore int
	switch notification.Priority {
	case "high":
		baseScore = 100
	case "normal":
		baseScore = 70
	case "low":
		baseScore = 50
	default:
		baseScore = 50
	}
	
	// User tier bonus
	var tierBonus int
	switch notification.UserTier {
	case "premium":
		tierBonus = 15
	case "standard":
		tierBonus = 0
	case "free":
		tierBonus = -10
	default:
		tierBonus = 0
	}
	
	// Time sensitivity factors - consider messages newer than 5 minutes as time-sensitive
	timeSensitivityBonus := 0
	if notification.CreatedAt.After(time.Now().Add(-5 * time.Minute)) {
		timeSensitivityBonus = 10
	}
	
	// Channel factors - different channels may have different base priorities
	channelBonus := 0
	switch notification.Channel {
	case "email":
		channelBonus = 0 // baseline
	case "sms":
		channelBonus = 5 // slightly higher priority
	case "push":
		channelBonus = 10 // higher priority
	case "whatsapp":
		channelBonus = 15 // highest priority
	}
	
	// Special template handling (order confirmations, password resets, etc.)
	templateBonus := 0
	if r.isHighPriorityTemplate(notification.TemplateID) {
		templateBonus = 20
	}
	
	// Combine all factors
	return baseScore + tierBonus + timeSensitivityBonus + channelBonus + templateBonus
}

// calculatePromotionalScore calculates score for promotional messages
func (r *RuleEngine) calculatePromotionalScore(notification *model.NotificationMessage) int {
	// Promotional messages start with a lower base score
	var baseScore int
	switch notification.Priority {
	case "high":
		baseScore = 40
	case "normal":
		baseScore = 30
	case "low":
		baseScore = 20
	default:
		baseScore = 20
	}
	
	// User tier bonus
	var tierBonus int
	switch notification.UserTier {
	case "premium":
		tierBonus = 10
	case "standard":
		tierBonus = 0
	case "free":
		tierBonus = -5
	default:
		tierBonus = 0
	}
	
	// Time of day adjustment - promotional messages are lower priority during peak hours
	hourOfDay := time.Now().Hour()
	timeOfDayAdjustment := 0
	if hourOfDay >= 9 && hourOfDay <= 18 {
		timeOfDayAdjustment = -10 // Lower priority during business hours
	}
	if hourOfDay >= 22 || hourOfDay <= 6 {
		timeOfDayAdjustment = -15 // Even lower priority during night hours
	}
	
	// Channel adjustment
	channelAdjustment := 0
	switch notification.Channel {
	case "email":
		channelAdjustment = 0 // baseline
	case "sms":
		channelAdjustment = -5 // lower priority for promotional SMS
	case "push":
		channelAdjustment = -2 // slightly lower for push
	}
	
	// Combine all factors
	return baseScore + tierBonus + timeOfDayAdjustment + channelAdjustment
}

// isHighPriorityTemplate determines if a template is high priority
func (r *RuleEngine) isHighPriorityTemplate(templateID string) bool {
	// List of high-priority templates
	highPriorityTemplates := map[string]bool{
		"password_reset": true,
		"account_lockout": true,
		"security_alert": true,
		"order_confirmation": true,
		"payment_confirmation": true,
		"appointment_reminder": true,
		"booking_confirmation": true,
	}
	
	return highPriorityTemplates[templateID]
}
