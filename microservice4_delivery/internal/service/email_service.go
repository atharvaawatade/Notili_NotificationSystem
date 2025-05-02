package service

import (
	"context"
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/appointy/notli/microservice4_delivery/internal/model"
	"github.com/appointy/notli/microservice4_delivery/internal/provider"
	"github.com/appointy/notli/microservice4_delivery/internal/repository"
	"github.com/appointy/notli/microservice4_delivery/internal/retry"
	"github.com/appointy/notli/microservice4_delivery/internal/template"
)

// EmailService coordinates the sending and tracking of emails
type EmailService struct {
	emailProvider    provider.EmailProvider
	templateEngine   template.TemplateEngineInterface
	emailRepository  repository.EmailRepositoryInterface
	retryStrategy    *retry.Strategy
	workerCount      int
	processingQueue  chan *model.KafkaMessage
	workerCancelFunc context.CancelFunc
}

// NewEmailService creates a new email service
func NewEmailService(
	emailProvider provider.EmailProvider,
	templateEngine template.TemplateEngineInterface,
	emailRepository repository.EmailRepositoryInterface,
	retryStrategy *retry.Strategy,
	workerCount int,
) *EmailService {
	return &EmailService{
		emailProvider:   emailProvider,
		templateEngine:  templateEngine,
		emailRepository: emailRepository,
		retryStrategy:   retryStrategy,
		workerCount:     workerCount,
		processingQueue: make(chan *model.KafkaMessage, 1000), // Buffer size of 1000
	}
}

// Start initializes the service and starts background workers
func (s *EmailService) Start(ctx context.Context) error {
	// Start worker goroutines
	workerCtx, cancel := context.WithCancel(ctx)
	s.workerCancelFunc = cancel

	for i := 0; i < s.workerCount; i++ {
		go s.worker(workerCtx, i)
	}

	// Start retry worker
	go s.retryWorker(workerCtx)

	log.Printf("Email service started with %d workers", s.workerCount)
	return nil
}

// Stop gracefully shuts down the service
func (s *EmailService) Stop() {
	if s.workerCancelFunc != nil {
		s.workerCancelFunc()
	}
	close(s.processingQueue)
	log.Println("Email service stopped")
}

// HandleMessage processes a notification message from Kafka
func (s *EmailService) HandleMessage(ctx context.Context, msg *model.KafkaMessage) error {
	// Check for duplicates based on idempotency key
	existing, err := s.emailRepository.GetDeliveryByIdempotencyKey(ctx, msg.IdempotencyKey)
	if err != nil {
		return fmt.Errorf("error checking for duplicate: %w", err)
	}

	if existing != nil {
		log.Printf("Skipping duplicate message with idempotency key: %s", msg.IdempotencyKey)
		return nil
	}

	// Queue the message for processing by a worker
	select {
	case s.processingQueue <- msg:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	default:
		// Queue is full, process synchronously
		log.Println("Processing queue is full, processing message synchronously")
		return s.processMessage(ctx, msg)
	}
}

// HandleBatch processes a batch of messages
func (s *EmailService) HandleBatch(ctx context.Context, messages []*model.KafkaMessage) error {
	for _, msg := range messages {
		if err := s.HandleMessage(ctx, msg); err != nil {
			// Log error but continue processing other messages
			log.Printf("Error handling message %s: %v", msg.IdempotencyKey, err)
		}
	}
	return nil
}

// worker is a background goroutine that processes messages from the queue
func (s *EmailService) worker(ctx context.Context, id int) {
	log.Printf("Worker %d started", id)
	for {
		select {
		case <-ctx.Done():
			log.Printf("Worker %d stopping due to context cancellation", id)
			return
		case msg, ok := <-s.processingQueue:
			if !ok {
				log.Printf("Worker %d stopping due to closed channel", id)
				return
			}
			if err := s.processMessage(ctx, msg); err != nil {
				log.Printf("Worker %d error processing message: %v", id, err)
			}
		}
	}
}

// retryWorker periodically checks for failed deliveries that need to be retried
func (s *EmailService) retryWorker(ctx context.Context) {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	log.Println("Retry worker started")

	for {
		select {
		case <-ctx.Done():
			log.Println("Retry worker stopping due to context cancellation")
			return
		case <-ticker.C:
			if err := s.processRetries(ctx); err != nil {
				log.Printf("Error processing retries: %v", err)
			}
		}
	}
}

// processRetries checks for failed deliveries and retries them
func (s *EmailService) processRetries(ctx context.Context) error {
	// Get deliveries that need to be retried (limit to 100 at a time)
	deliveries, err := s.emailRepository.GetDeliveriesToRetry(ctx, 100)
	if err != nil {
		return fmt.Errorf("error fetching deliveries to retry: %w", err)
	}

	if len(deliveries) == 0 {
		return nil
	}

	log.Printf("Processing %d deliveries for retry", len(deliveries))

	for _, delivery := range deliveries {
		// Check if we should retry based on retry count
		if !s.retryStrategy.ShouldRetry(delivery.RetryCount) {
			// Max retries reached, mark as permanently failed
			err := s.emailRepository.UpdateDeliveryStatus(ctx, delivery.MessageID, "failed_permanent", "Max retry attempts reached")
			if err != nil {
				log.Printf("Error updating delivery status to permanent failure: %v", err)
			}
			continue
		}

		// Retry sending the email
		log.Printf("Retrying delivery %s (attempt %d)", delivery.MessageID, delivery.RetryCount+1)

		// Create a simple email message for retry
		email := &model.EmailMessage{
			From:           delivery.Sender,
			To:             []string{delivery.Recipient},
			Subject:        delivery.Subject,
			IdempotencyKey: delivery.IdempotencyKey,
		}

		// Attempt to resend
		messageID, err := s.emailProvider.Send(ctx, email)

		// Update retry information
		nextRetryAt := s.retryStrategy.CalculateRetryTime(delivery.RetryCount + 1)
		newStatus := "queued"
		details := ""

		if err != nil {
			newStatus = "failed"
			details = err.Error()
			log.Printf("Retry failed for %s: %v", delivery.MessageID, err)
		} else {
			log.Printf("Retry succeeded for %s, new message ID: %s", delivery.MessageID, messageID)
		}

		// Update the delivery record
		err = s.emailRepository.UpdateRetryInfo(ctx, delivery.MessageID, delivery.RetryCount+1, nextRetryAt)
		if err != nil {
			log.Printf("Error updating retry info: %v", err)
		}

		err = s.emailRepository.UpdateDeliveryStatus(ctx, delivery.MessageID, newStatus, details)
		if err != nil {
			log.Printf("Error updating delivery status: %v", err)
		}

		// Record the status event
		event := &model.StatusEvent{
			MessageID:  delivery.MessageID,
			Status:     newStatus,
			Provider:   s.emailProvider.Name(),
			Details:    details,
			OccurredAt: time.Now(),
		}

		err = s.emailRepository.AddStatusEvent(ctx, event)
		if err != nil {
			log.Printf("Error adding status event: %v", err)
		}
	}

	return nil
}

// processMessage handles the actual processing of a message
func (s *EmailService) processMessage(ctx context.Context, msg *model.KafkaMessage) error {
	// Only process email channel messages
	if msg.Channel != "email" {
		log.Printf("Skipping non-email channel message: %s", msg.Channel)
		return nil
	}

	// Parse channel-specific data
	var subject, plainBody, htmlBody string

	// Extract email data from the message
	channelData := msg.ChannelData
	
	// Extract standard fields
	if subj, ok := channelData["subject"].(string); ok {
		subject = subj
	}
	if plain, ok := channelData["plain_body"].(string); ok {
		plainBody = plain
	}
	if html, ok := channelData["html_body"].(string); ok {
		htmlBody = html
	}
	
	// Extract body from 'body' field if provided (for backwards compatibility)
	if plainBody == "" && htmlBody == "" {
		if body, ok := channelData["body"].(string); ok && body != "" {
			plainBody = body
		}
	}

	// Prepare template data regardless if template ID is provided
	templateData := make(map[string]interface{})
	if data, ok := channelData["template_data"]; ok {
		templateData, ok = data.(map[string]interface{})
		if !ok {
			log.Printf("Warning: template_data is not a map, creating empty map")
			templateData = make(map[string]interface{})
		}
	}

	// Add standard template variables
	templateData["AppName"] = "Notli"
	templateData["UserID"] = msg.UserID
	templateData["UserTier"] = msg.UserTier
	
	// For transactional emails, always use our transaction template
	templateID := msg.TemplateID
	log.Printf("DEBUG: Original template_id: '%s', message_type: '%s'", templateID, msg.MessageType)
	
	if msg.MessageType == "transactional" && templateID == "" {
		// Auto-apply our transaction template
		templateID = "transaction"
		log.Printf("🔍 TEMPLATE: Auto-applying 'transaction' template to transactional email (green/white theme)")
		
		// Extract name from email (or use a friendly default)
		recipientName := "User"
		if atIndex := strings.Index(msg.Recipient, "@"); atIndex > 0 {
			recipientName = msg.Recipient[:atIndex]
			// Capitalize first letter
			if len(recipientName) > 0 {
				recipientName = strings.ToUpper(recipientName[:1]) + recipientName[1:]
			}
		}
		log.Printf("🔍 TEMPLATE: Extracted recipient name: '%s'", recipientName)
		
		// Add required template variables
		templateData["Subject"] = subject
		templateData["Message"] = plainBody
		templateData["Name"] = recipientName
		log.Printf("🔍 TEMPLATE: Set template variables: Subject='%s', Message='%s'", subject, plainBody)
		
		// Make logo available in template
		templateData["LogoURL"] = "https://notli.io/logo.png" // Use default if none provided
		
		// Add current time for copyright year
		templateData["Now"] = time.Now()
		
		log.Printf("🔍 TEMPLATE: Template data prepared with %d variables", len(templateData))
	}

	// Use template if available
	if templateID != "" {
		log.Printf("🔍 TEMPLATE: Fetching template '%s' from template engine", templateID)
		
		// Try to get the template first to verify it exists
		tmpl, err := s.templateEngine.Get(templateID)
		if err != nil {
			log.Printf("⚠️ TEMPLATE ERROR: Could not find template '%s': %v", templateID, err)
			log.Printf("🔍 TEMPLATE: Available templates in system may not include '%s'", templateID)
		} else {
			log.Printf("🔍 TEMPLATE: Successfully found template '%s' - '%s'", tmpl.ID, tmpl.Name)
		}
		
		// Render template
		log.Printf("🔍 TEMPLATE: Starting template rendering process for '%s'", templateID)
		
		var renderErr error
		originalSubject := subject
		originalPlain := plainBody
		
		subject, plainBody, htmlBody, renderErr = s.templateEngine.Render(templateID, templateData)
		
		if renderErr != nil {
			log.Printf("⚠️ TEMPLATE ERROR: Failed to render template '%s': %v", templateID, renderErr)
			log.Printf("🔍 TEMPLATE: Falling back to original email content")
		} else {
			htmlLength := 0
			templateSuccess := "❌ NO TEMPLATE APPLIED"
			
			if htmlBody != "" {
				htmlLength = len(htmlBody)
				templateSuccess = "✅ GREEN TEMPLATE APPLIED"
			}
			
			log.Printf("🔍 TEMPLATE: %s for template '%s'!", templateSuccess, templateID)
			log.Printf("🔍 TEMPLATE: Final email - Subject: '%s'", subject)
			log.Printf("🔍 TEMPLATE: Content lengths - Plain: %d chars, HTML: %d chars", 
				len(plainBody), htmlLength)
			
			// Log brief preview of HTML content if available
			if htmlLength > 0 {
				previewLength := 100
				if htmlLength < previewLength {
					previewLength = htmlLength
				}
				log.Printf("🔍 TEMPLATE: HTML preview (first %d chars): %s...", 
					previewLength, htmlBody[:previewLength])
			}
			
			// Log changes to subject and body for debugging
			if originalSubject != subject {
				log.Printf("🔍 TEMPLATE: Subject changed from '%s' to '%s'", originalSubject, subject)
			}
			
			if originalPlain != plainBody {
				log.Printf("🔍 TEMPLATE: Plain text content changed from %d chars to %d chars", 
					len(originalPlain), len(plainBody))
			}
		}
	} else {
		log.Printf("🔍 TEMPLATE: No template ID provided, sending plain email without template")
	}

	// Check for sender information in channel data
	var fromEmail string
	var fromName string // Store this for logging only
	
	// Extract from_name for logging
	if name, ok := channelData["from_name"].(string); ok && name != "" {
		fromName = name
	}
	
	// Extract from_email to be used in the email
	if email, ok := channelData["from_email"].(string); ok && email != "" {
		fromEmail = email
		if fromName != "" {
			log.Printf("Using custom sender: %s <%s>", fromName, fromEmail)
		} else {
			log.Printf("Using custom sender email: %s", fromEmail)
		}
	}

	// Create email message
	// Calculate priority as an integer - using the PriorityScore if available, otherwise use 1 as default
	priority := 1 // Default priority
	if msg.PriorityScore > 0 {
		priority = msg.PriorityScore
	}
	
	// Log priority information for debugging
	log.Printf("Message priority: %s, priority score: %d", msg.Priority, msg.PriorityScore)

	email := &model.EmailMessage{
		To:             []string{msg.Recipient},
		Subject:        subject,
		PlainBody:      plainBody,
		HtmlBody:       htmlBody,
		From:           fromEmail, // Set the From field with the extracted email
		IdempotencyKey: msg.IdempotencyKey,
		Priority:       priority,  // Use the calculated integer priority 
		MessageType:    msg.MessageType,
		Channel:        msg.Channel,
		UserID:         msg.UserID,
		UserTier:       msg.UserTier,
		TemplateID:     msg.TemplateID,
	}

	// Send the email
	log.Printf("Sending email to %s with subject '%s'", msg.Recipient, subject)
	messageID, err := s.emailProvider.Send(ctx, email)

	// Create delivery record
	now := time.Now()
	delivery := &model.DeliveryRecord{
		MessageID:      messageID,
		IdempotencyKey: msg.IdempotencyKey,
		Recipient:      msg.Recipient,
		Sender:         email.From,
		Subject:        subject,
		TemplateID:     msg.TemplateID,
		Provider:       s.emailProvider.Name(),
		CreatedAt:      now,
		UpdatedAt:      now,
		RetryCount:     0,
	}

	// Set status based on send result
	if err != nil {
		delivery.Status = "failed"
		delivery.StatusDetails = err.Error()

		// Schedule retry if appropriate
		if retry.IsRetriable(err) {
			delivery.NextRetryAt = s.retryStrategy.CalculateRetryTime(1)
		}
	} else {
		delivery.Status = "sent"
	}

	// Save delivery record
	if err := s.emailRepository.CreateDelivery(ctx, delivery); err != nil {
		log.Printf("Error saving delivery record: %v", err)
	}

	// Record status event
	event := &model.StatusEvent{
		MessageID:  messageID,
		Status:     delivery.Status,
		Provider:   s.emailProvider.Name(),
		Details:    delivery.StatusDetails,
		OccurredAt: now,
	}

	if err := s.emailRepository.AddStatusEvent(ctx, event); err != nil {
		log.Printf("Error saving status event: %v", err)
	}

	if err != nil {
		return fmt.Errorf("error sending email: %w", err)
	}

	log.Printf("Email sent successfully, message ID: %s", messageID)
	return nil
}

// GetEmailStatus retrieves the current status of an email
func (s *EmailService) GetEmailStatus(ctx context.Context, messageID string) (*model.EmailStatus, error) {
	// Check local database first
	delivery, err := s.emailRepository.GetDeliveryByID(ctx, messageID)
	if err != nil {
		return nil, fmt.Errorf("error fetching delivery record: %w", err)
	}

	if delivery == nil {
		return nil, fmt.Errorf("delivery not found: %s", messageID)
	}

	// For non-final statuses, check with the provider for latest status
	if delivery.Status != "delivered" && delivery.Status != "failed_permanent" {
		status, err := s.emailProvider.GetStatus(ctx, messageID)
		if err == nil {
			// Update our records with latest status from provider
			err = s.emailRepository.UpdateDeliveryStatus(ctx, messageID, status.Status, status.Details)
			if err != nil {
				log.Printf("Error updating delivery status: %v", err)
			}

			// Record the status event
			event := &model.StatusEvent{
				MessageID:  messageID,
				Status:     status.Status,
				Provider:   status.Provider,
				Details:    status.Details,
				OccurredAt: status.Timestamp,
			}

			err = s.emailRepository.AddStatusEvent(ctx, event)
			if err != nil {
				log.Printf("Error adding status event: %v", err)
			}

			return status, nil
		}

		// If provider check fails, fall back to our records
		log.Printf("Error checking status with provider: %v", err)
	}

	// Return status from our records
	return &model.EmailStatus{
		MessageID: delivery.MessageID,
		Status:    delivery.Status,
		Provider:  delivery.Provider,
		Details:   delivery.StatusDetails,
		Timestamp: delivery.UpdatedAt,
	}, nil
}

// InitDB initializes the database
func (s *EmailService) InitDB(ctx context.Context) error {
	return s.emailRepository.Init(ctx)
}
