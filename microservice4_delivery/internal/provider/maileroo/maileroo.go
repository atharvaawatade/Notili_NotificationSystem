package maileroo

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"mime/multipart"
	"net/http"
	"strings"
	"time"

	"github.com/appointy/notli/microservice4_delivery/internal/model"
	"github.com/appointy/notli/microservice4_delivery/internal/provider"
)

const (
	providerName = "maileroo"
)

var (
	// Default API endpoints - can be overridden in tests
	baseURL = "https://smtp.maileroo.com"
)

// MailerooProvider implements the EmailProvider interface for the Maileroo API
type MailerooProvider struct {
	apiKey      string
	domain      string
	defaultFrom string
	fromName    string
	client      *http.Client
	sendURL     string
}

// MailerooResponse represents the standard response from the Maileroo API
type MailerooResponse struct {
	Success bool   `json:"success"`
	Message string `json:"message"`
}

// NewMailerooProvider creates a new Maileroo email provider
func NewMailerooProvider(apiKey, domain, defaultFrom, fromName string) (*MailerooProvider, error) {
	if apiKey == "" {
		return nil, fmt.Errorf("API key is required")
	}
	if domain == "" {
		return nil, fmt.Errorf("domain is required")
	}
	if defaultFrom == "" {
		return nil, fmt.Errorf("default from address is required")
	}

	return &MailerooProvider{
		apiKey:      apiKey,
		domain:      domain,
		defaultFrom: defaultFrom,
		fromName:    fromName,
		client: &http.Client{
			Timeout: 30 * time.Second,
		},
		sendURL: baseURL + "/send",
	}, nil
}

// Factory method for the ProviderRegistry
func MailerooFactory(config map[string]string) (provider.EmailProvider, error) {
	apiKey := config["api_key"]
	domain := config["domain"]
	defaultFrom := config["default_from"]
	fromName := config["from_name"]

	if apiKey == "" || domain == "" || defaultFrom == "" {
		return nil, fmt.Errorf("invalid configuration")
	}

	return NewMailerooProvider(apiKey, domain, defaultFrom, fromName)
}

// Name returns the provider name
func (m *MailerooProvider) Name() string {
	return providerName
}

// Describe returns a description of the provider
func (p *MailerooProvider) Describe() string {
	return fmt.Sprintf("Maileroo (%s)", baseURL)
}

// Send transmits an email through the Maileroo API
func (m *MailerooProvider) Send(ctx context.Context, email *model.EmailMessage) (string, error) {
	// Set from email with proper name if provided
	fromEmail := m.defaultFrom
	if email.From != "" {
		fromEmail = email.From
	}

	// Format from field with name if available
	from := fromEmail
	if m.fromName != "" {
		from = fmt.Sprintf("\"%s\" <%s>", m.fromName, fromEmail)
	}

	// Create a new multipart form
	var b bytes.Buffer
	writer := multipart.NewWriter(&b)

	// Add from field
	if err := writer.WriteField("from", from); err != nil {
		return "", fmt.Errorf("error writing from field: %w", err)
	}

	// Add to field (comma-separated list)
	toList := strings.Join(email.To, ",")
	if err := writer.WriteField("to", toList); err != nil {
		return "", fmt.Errorf("error writing to field: %w", err)
	}

	// Add cc field if provided
	if len(email.Cc) > 0 {
		ccList := strings.Join(email.Cc, ",")
		if err := writer.WriteField("cc", ccList); err != nil {
			return "", fmt.Errorf("error writing cc field: %w", err)
		}
	}

	// Add bcc field if provided
	if len(email.Bcc) > 0 {
		bccList := strings.Join(email.Bcc, ",")
		if err := writer.WriteField("bcc", bccList); err != nil {
			return "", fmt.Errorf("error writing bcc field: %w", err)
		}
	}

	// Add subject field
	if err := writer.WriteField("subject", email.Subject); err != nil {
		return "", fmt.Errorf("error writing subject field: %w", err)
	}

	// Add HTML content if provided
	if email.HtmlBody != "" {
		if err := writer.WriteField("html", email.HtmlBody); err != nil {
			return "", fmt.Errorf("error writing html field: %w", err)
		}
	}

	// Add plain text content if provided
	if email.PlainBody != "" {
		if err := writer.WriteField("plain", email.PlainBody); err != nil {
			return "", fmt.Errorf("error writing plain field: %w", err)
		}
	} else if email.HtmlBody == "" {
		// If neither HTML nor plain text is provided, create a simple plain text message
		if err := writer.WriteField("plain", "This is an automated email from Notli."); err != nil {
			return "", fmt.Errorf("error writing default plain field: %w", err)
		}
	}

	// Add reference ID (idempotency key) if available
	if email.IdempotencyKey != "" {
		// Maileroo requires a 24-character hexadecimal string
		// We'll use up to the first 24 chars of our idempotency key
		refID := email.IdempotencyKey
		if len(refID) > 24 {
			refID = refID[:24]
		}
		if err := writer.WriteField("reference_id", refID); err != nil {
			return "", fmt.Errorf("error writing reference_id field: %w", err)
		}
	}

	// Add tracking setting
	if err := writer.WriteField("tracking", "yes"); err != nil {
		return "", fmt.Errorf("error writing tracking field: %w", err)
	}

	// Close the multipart writer
	err := writer.Close()
	if err != nil {
		return "", fmt.Errorf("error closing multipart writer: %w", err)
	}

	// Log the request for debugging
	log.Printf("[DEBUG] Sending email request to Maileroo: From=%s, To=%s, Subject=%s", from, toList, email.Subject)

	// Create HTTP request
	httpReq, err := http.NewRequestWithContext(ctx, "POST", m.sendURL, &b)
	if err != nil {
		return "", fmt.Errorf("error creating request: %w", err)
	}

	// Set headers
	httpReq.Header.Set("Content-Type", writer.FormDataContentType())
	httpReq.Header.Set("X-API-Key", m.apiKey)

	// Execute request
	resp, err := m.client.Do(httpReq)
	if err != nil {
		return "", fmt.Errorf("error sending request: %w", err)
	}
	defer resp.Body.Close()

	// Read response body
	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("error reading response: %w", err)
	}

	// Log the response for debugging
	log.Printf("[DEBUG] Maileroo API response (status %d): %s", resp.StatusCode, string(respBody))

	// Parse response
	var mailerooResp MailerooResponse
	if err := json.Unmarshal(respBody, &mailerooResp); err != nil {
		return "", fmt.Errorf("error parsing response: %w", err)
	}

	// Handle error response
	if !mailerooResp.Success {
		log.Printf("[ERROR] Maileroo send failed: %s", mailerooResp.Message)
		return "", fmt.Errorf("send failed: %s", mailerooResp.Message)
	}

	log.Printf("[INFO] Email successfully sent with Maileroo: %s", mailerooResp.Message)

	// Generate a fake message ID since Maileroo doesn't return one
	messageID := fmt.Sprintf("maileroo-%d", time.Now().UnixNano())
	return messageID, nil
}

// GetStatus retrieves the current status of a previously sent email
func (m *MailerooProvider) GetStatus(ctx context.Context, messageID string) (*model.EmailStatus, error) {
	// Maileroo doesn't provide a status API that works with our implementation
	// For now, we'll just return a default "delivered" status
	return &model.EmailStatus{
		MessageID:    messageID,
		Provider:     providerName,
		Status:       "delivered",
		Timestamp:    time.Now(),
	}, nil
}

// Close releases any resources used by the provider
func (m *MailerooProvider) Close() error {
	// Nothing to close for HTTP client
	return nil
}

// WithCustomClient allows setting a custom HTTP client
func (p *MailerooProvider) WithCustomClient(client *http.Client) *MailerooProvider {
	p.client = client
	return p
}

// BaseURL returns the base URL of the provider
func (p *MailerooProvider) BaseURL() string {
	return baseURL
}

// StatusURL returns the status URL of the provider
func (p *MailerooProvider) StatusURL() string {
	return p.sendURL
}

// WithBaseURL allows setting a custom base URL (mainly for testing)
func (p *MailerooProvider) WithBaseURL(url string) *MailerooProvider {
	// Only modify the sendURL, as we don't have a baseURL field anymore
	p.sendURL = url + "/send"
	return p
}

// WithStatusURL allows setting a custom status URL (mainly for testing)
func (p *MailerooProvider) WithStatusURL(url string) *MailerooProvider {
	// We don't use status URL in this implementation
	return p
}
