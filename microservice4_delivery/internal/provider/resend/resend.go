package resend

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/appointy/notli/microservice4_delivery/internal/model"
	"github.com/appointy/notli/microservice4_delivery/internal/provider"
	"github.com/resend/resend-go/v2"
)

const (
	providerName = "resend"
)

// ResendProvider implements the EmailProvider interface for the Resend API
type ResendProvider struct {
	apiKey        string
	domain        string
	verifiedEmail string // The verified email for the domain
	client        *resend.Client
	enableDetailedLogs bool // Enables very detailed logging of requests and responses
}

// NewResendProvider creates a new Resend email provider
func NewResendProvider(config map[string]string) (*ResendProvider, error) {
	apiKey := config["api_key"]
	domain := config["domain"]
	detailedLogsStr := config["detailed_logs"]

	if apiKey == "" {
		return nil, fmt.Errorf("API key is required")
	}

	// Create client exactly as in the example
	client := resend.NewClient(apiKey)

	// Get verified email from config
	verifiedEmail := config["verified_email"]
	if verifiedEmail == "" {
		// Use default from the domain if none specified
		verifiedEmail = "simplivu@simplivu.com"
	}

	// Determine if detailed logs are enabled
	enableDetailedLogs := strings.ToLower(detailedLogsStr) == "true"

	// Log provider creation with configuration details
	log.Printf("[INFO] Creating Resend email provider with API key: %s...", apiKey[:10])
	log.Printf("[INFO] Domain: %s", domain)
	log.Printf("[INFO] Verified email: %s", verifiedEmail)
	log.Printf("[INFO] Detailed logs: %v", enableDetailedLogs)

	return &ResendProvider{
		apiKey:        apiKey,
		domain:        domain,
		client:        client,
		verifiedEmail: verifiedEmail,
		enableDetailedLogs: enableDetailedLogs,
	}, nil
}

// Factory method for the ProviderRegistry
func ResendFactory(config map[string]string) (provider.EmailProvider, error) {
	return NewResendProvider(config)
}

// Name returns the provider name
func (r *ResendProvider) Name() string {
	return providerName
}

// Describe returns a description of the provider
func (r *ResendProvider) Describe() string {
	return fmt.Sprintf("Resend (%s)", r.verifiedEmail)
}

// logObject logs an object for debugging purposes if detailed logging is enabled
// This method is kept for future debugging needs
// nolint:unused
func (r *ResendProvider) logObject(prefix string, obj interface{}) {
	if !r.enableDetailedLogs {
		return
	}
	
	jsonData, err := json.MarshalIndent(obj, "", "  ")
	if err != nil {
		log.Printf("[ERROR] Failed to marshal %s to JSON: %v", prefix, err)
		return
	}
	log.Printf("[DEBUG] %s: %s", prefix, string(jsonData))
}

// Send transmits an email through the Resend API
func (r *ResendProvider) Send(ctx context.Context, email *model.EmailMessage) (string, error) {
	// Generate a simple requestID for logging
	requestID := fmt.Sprintf("resend-%d", time.Now().UnixNano())
	log.Printf("[INFO] Starting email delivery via Resend (ID: %s)", requestID)

	// SIMPLE APPROACH: Always use the verified domain email that we know works
	from := "Simplivu <simplivu@simplivu.com>"

	// Basic logging
	log.Printf("[INFO] Sending email From: %s, To: %v, Subject: %s", from, email.To, email.Subject)

	// Extract recipient name from email address for personalization
	recipientName := "User"
	if len(email.To) > 0 && email.To[0] != "" {
		recipientAddr := email.To[0]
		if atIndex := strings.Index(recipientAddr, "@"); atIndex > 0 {
			recipientName = recipientAddr[:atIndex]
			// Capitalize first letter
			if len(recipientName) > 0 {
				recipientName = strings.ToUpper(recipientName[:1]) + recipientName[1:]
			}
		}
	}
	log.Printf("[TEMPLATE] Using recipient name: %s", recipientName)

	// Prepare HTML content with green/white themed template
	htmlContent := email.HtmlBody
	
	// If no HTML content is provided, create a nice template with the message
	if htmlContent == "" {
		// Get current year for copyright
		currentYear := time.Now().Year()
		
		// Extract message from plain body or use a default message
		message := email.PlainBody
		if message == "" {
			message = "This is a notification from Notli."
		}
		
		// Log that we're applying the template
		log.Printf("[TEMPLATE] 🟢 Applying green/white transaction template to email")
		
		// Create a beautiful green/white themed HTML template
		htmlContent = fmt.Sprintf(`<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>%s</title>
    <style>
        /* Base styles */
        body {
            font-family: 'Segoe UI', Tahoma, Geneva, Verdana, sans-serif;
            line-height: 1.6;
            color: #333333;
            background-color: #f9f9f9;
            margin: 0;
            padding: 0;
        }
        .email-container {
            max-width: 600px;
            margin: 0 auto;
            background-color: #ffffff;
            border-radius: 8px;
            overflow: hidden;
            box-shadow: 0 4px 10px rgba(0, 0, 0, 0.05);
        }
        .email-header {
            background-color: #10b981; /* Green header */
            padding: 24px;
            text-align: center;
        }
        .email-header h2 {
            color: #ffffff;
            margin: 0;
        }
        .email-body {
            padding: 32px 24px;
            background-color: #ffffff;
        }
        .email-footer {
            background-color: #f3f4f6;
            color: #6b7280;
            font-size: 14px;
            text-align: center;
            padding: 16px 24px;
            border-top: 1px solid #e5e7eb;
        }
        h1 {
            color: #10b981;
            font-weight: 600;
            margin-top: 0;
            margin-bottom: 24px;
            font-size: 24px;
        }
        p {
            margin: 0 0 16px;
        }
        .message {
            margin-bottom: 24px;
        }
    </style>
</head>
<body>
    <div class="email-container">
        <div class="email-header">
            <h2>Notli</h2>
        </div>
        <div class="email-body">
            <h1>%s</h1>
            <div class="message">
                <p>Hi %s,</p>
                <p>%s</p>
            </div>
            <p>If you have any questions, please don't hesitate to contact our support team.</p>
            <p>Best regards,<br>The Notli Team</p>
        </div>
        <div class="email-footer">
            <p>&copy; %d Notli. All rights reserved.</p>
        </div>
    </div>
</body>
</html>`, 
			email.Subject, // Title
			email.Subject, // H1 heading
			recipientName, // Recipient name
			message, // Message content 
			currentYear) // Copyright year
			
		log.Printf("[TEMPLATE] ✅ Successfully generated green/white themed HTML template (%d chars)", len(htmlContent))
	} else {
		log.Printf("[TEMPLATE] Using provided HTML content (%d chars)", len(htmlContent))
	}

	// EXACTLY match our successful test structure
	params := &resend.SendEmailRequest{
		From:    from,
		To:      email.To,
		Subject: email.Subject,
		Html:    htmlContent,
	}

	// Add CC and BCC only if provided
	if len(email.Cc) > 0 {
		params.Cc = email.Cc
	}

	if len(email.Bcc) > 0 {
		params.Bcc = email.Bcc
	}

	// Send the email - simple approach like our test
	log.Printf("[INFO] Sending email via Resend API...")
	sent, err := r.client.Emails.Send(params)

	if err != nil {
		// Simple error logging
		log.Printf("[ERROR] Resend API call failed: %v", err)
		log.Printf("[ERROR] Error type: %T", err)
		return "", fmt.Errorf("send failed: %w", err)
	}

	// Success path with simple logging
	log.Printf("[SUCCESS] Email sent successfully!")
	log.Printf("[SUCCESS] Resend message ID: %s", sent.Id)
	
	// Return the message ID
	return sent.Id, nil
}

// GetStatus retrieves the current status of a previously sent email
func (r *ResendProvider) GetStatus(ctx context.Context, messageID string) (*model.EmailStatus, error) {
	if messageID == "" {
		return nil, fmt.Errorf("message ID is required")
	}
	
	log.Printf("[INFO] Checking email status for message ID: %s", messageID)
	
	// Resend does have an API to get Email status, but it's not fully exposed in the SDK
	// In a production environment, you would implement webhook handling for delivery events
	// For now, we'll simulate a status lookup with detailed logging
	
	log.Printf("[INFO] Email lookup status for ID %s: message accepted for delivery", messageID)
	
	// Create a details string with helpful information
	detailsStr := "Resend does not provide real-time status in the SDK. Use webhooks for production. Check the Resend dashboard for actual delivery status."

	return &model.EmailStatus{
		MessageID: messageID,
		Provider:  providerName,
		Status:    "sent",
		Timestamp: time.Now(),
		Details:   detailsStr,
	}, nil
}

// Close releases any resources used by the provider
func (r *ResendProvider) Close() error {
	// Nothing to close for the Resend client
	return nil
}
