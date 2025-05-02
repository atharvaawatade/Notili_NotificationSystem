package priority

import (
	"testing"
	"time"

	"github.com/appointy/notli/microservice3_prioritization/internal/model"
	"github.com/stretchr/testify/assert"
)

func TestCalculatePriorityScore(t *testing.T) {
	ruleEngine := NewRuleEngine()
	now := time.Now()

	tests := []struct {
		name           string
		notification   *model.NotificationMessage
		expectedScore  int
		description    string
	}{
		{
			name: "High priority transactional for premium user",
			notification: &model.NotificationMessage{
				IdempotencyKey: "test-1",
				Priority:       "high",
				MessageType:    "transactional",
				Channel:        "email",
				UserTier:       "premium",
				TemplateID:     "order_confirmation",
				CreatedAt:      now,
			},
			expectedScore: 100 + 15 + 10 + 0 + 20, // 145
			description:   "Base(100) + Premium(15) + TimeSensitive(10) + Email(0) + HighPriorityTemplate(20)",
		},
		{
			name: "Normal priority transactional for standard user",
			notification: &model.NotificationMessage{
				IdempotencyKey: "test-2",
				Priority:       "normal",
				MessageType:    "transactional",
				Channel:        "email",
				UserTier:       "standard",
				TemplateID:     "general_notification",
				CreatedAt:      now,
			},
			expectedScore: 70 + 0 + 10 + 0 + 0, // 80
			description:   "Base(70) + Standard(0) + TimeSensitive(10) + Email(0) + RegularTemplate(0)",
		},
		{
			name: "Low priority transactional for free user",
			notification: &model.NotificationMessage{
				IdempotencyKey: "test-3",
				Priority:       "low",
				MessageType:    "transactional",
				Channel:        "email",
				UserTier:       "free",
				CreatedAt:      now.Add(-10 * time.Minute), // Older message
				TemplateID:     "general_notification",
			},
			expectedScore: 50 - 10 + 0 + 0 + 0, // 40
			description:   "Base(50) + Free(-10) + NotTimeSensitive(0) + Email(0) + RegularTemplate(0)",
		},
		{
			name: "SMS channel gets higher priority",
			notification: &model.NotificationMessage{
				IdempotencyKey: "test-4",
				Priority:       "normal",
				MessageType:    "transactional",
				Channel:        "sms",
				UserTier:       "standard",
				CreatedAt:      now,
				TemplateID:     "general_notification",
			},
			expectedScore: 70 + 0 + 10 + 5 + 0, // 85
			description:   "Base(70) + Standard(0) + TimeSensitive(10) + SMS(5) + RegularTemplate(0)",
		},
		{
			name: "WhatsApp gets highest channel priority",
			notification: &model.NotificationMessage{
				IdempotencyKey: "test-5",
				Priority:       "normal",
				MessageType:    "transactional",
				Channel:        "whatsapp",
				UserTier:       "premium",
				CreatedAt:      now,
				TemplateID:     "general_notification",
			},
			expectedScore: 70 + 15 + 10 + 15 + 0, // 110
			description:   "Base(70) + Premium(15) + TimeSensitive(10) + WhatsApp(15) + RegularTemplate(0)",
		},
		{
			name: "Password reset gets template priority boost",
			notification: &model.NotificationMessage{
				IdempotencyKey: "test-6",
				Priority:       "high",
				MessageType:    "transactional",
				Channel:        "email",
				UserTier:       "standard",
				CreatedAt:      now,
				TemplateID:     "password_reset",
			},
			expectedScore: 100 + 0 + 10 + 0 + 20, // 130
			description:   "Base(100) + Standard(0) + TimeSensitive(10) + Email(0) + PasswordReset(20)",
		},
		{
			name: "Promotional message has lower base priority",
			notification: &model.NotificationMessage{
				IdempotencyKey: "test-7",
				Priority:       "high",
				MessageType:    "promotional",
				Channel:        "email",
				UserTier:       "premium",
				CreatedAt:      now,
				TemplateID:     "marketing_campaign",
			},
			expectedScore: 40 + 10, // 50 (time of day adjustment will vary)
			description:   "Base(40) + Premium(10) + TimeAdjustment(variable)",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			score := ruleEngine.CalculatePriorityScore(tc.notification)
			
			// For promotional messages, exact score depends on time of day
			// so we'll just check that it's in a reasonable range
			if tc.notification.MessageType == "promotional" {
				assert.GreaterOrEqual(t, score, 30, "Score should be at least base priority")
				assert.LessOrEqual(t, score, 60, "Score should have a reasonable upper bound")
			} else {
				// For transactional messages, we can check exact values
				assert.Equal(t, tc.expectedScore, score, tc.description)
			}
		})
	}
}

func TestIsHighPriorityTemplate(t *testing.T) {
	ruleEngine := NewRuleEngine()
	
	highPriorityTemplates := []string{
		"password_reset",
		"account_lockout",
		"security_alert",
		"order_confirmation",
		"payment_confirmation",
		"appointment_reminder",
		"booking_confirmation",
	}
	
	for _, templateID := range highPriorityTemplates {
		assert.True(t, ruleEngine.isHighPriorityTemplate(templateID), 
			"Template %s should be high priority", templateID)
	}
	
	regularTemplates := []string{
		"marketing_email",
		"newsletter",
		"product_recommendation",
		"general_notification",
		"survey_invitation",
	}
	
	for _, templateID := range regularTemplates {
		assert.False(t, ruleEngine.isHighPriorityTemplate(templateID),
			"Template %s should not be high priority", templateID)
	}
}
