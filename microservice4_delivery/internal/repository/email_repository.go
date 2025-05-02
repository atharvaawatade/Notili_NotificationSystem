package repository

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/appointy/notli/microservice4_delivery/internal/model"
	_ "github.com/lib/pq"
)

// EmailRepositoryInterface defines the interface that EmailRepository implements
// This allows for easier testing with mocks
type EmailRepositoryInterface interface {
	CreateDelivery(ctx context.Context, delivery *model.DeliveryRecord) error
	UpdateDeliveryStatus(ctx context.Context, messageID, status, details string) error
	UpdateRetryInfo(ctx context.Context, messageID string, retryCount int, nextRetryAt time.Time) error
	AddStatusEvent(ctx context.Context, event *model.StatusEvent) error
	GetDeliveryByID(ctx context.Context, messageID string) (*model.DeliveryRecord, error)
	GetDeliveryByIdempotencyKey(ctx context.Context, idempotencyKey string) (*model.DeliveryRecord, error)
	GetDeliveriesToRetry(ctx context.Context, limit int) ([]*model.DeliveryRecord, error)
	GetStatusEvents(ctx context.Context, messageID string) ([]*model.StatusEvent, error)
	Init(ctx context.Context) error
}

// EmailRepository provides access to email delivery records in the database
type EmailRepository struct {
	db *sql.DB
}

// NewEmailRepository creates a new email repository
func NewEmailRepository(db *sql.DB) *EmailRepository {
	return &EmailRepository{db: db}
}

// Init initializes the repository by creating necessary tables if they don't exist
func (r *EmailRepository) Init(ctx context.Context) error {
	// Create email_deliveries table
	_, err := r.db.ExecContext(ctx, `
		CREATE TABLE IF NOT EXISTS email_deliveries (
			message_id VARCHAR(100) PRIMARY KEY,
			idempotency_key VARCHAR(36) NOT NULL,
			recipient TEXT NOT NULL,
			sender TEXT NOT NULL,
			subject TEXT NOT NULL,
			template_id VARCHAR(100),
			provider VARCHAR(50) NOT NULL,
			status VARCHAR(20) NOT NULL,
			status_details TEXT,
			created_at TIMESTAMP NOT NULL DEFAULT NOW(),
			updated_at TIMESTAMP NOT NULL DEFAULT NOW(),
			retry_count INT NOT NULL DEFAULT 0,
			next_retry_at TIMESTAMP,
			CONSTRAINT idx_email_deliveries_idempotency UNIQUE (idempotency_key)
		)
	`)
	if err != nil {
		return fmt.Errorf("failed to create email_deliveries table: %w", err)
	}

	// Create email_status_events table
	_, err = r.db.ExecContext(ctx, `
		CREATE TABLE IF NOT EXISTS email_status_events (
			id SERIAL PRIMARY KEY,
			message_id VARCHAR(100) NOT NULL REFERENCES email_deliveries(message_id),
			status VARCHAR(20) NOT NULL,
			provider VARCHAR(50) NOT NULL,
			details TEXT,
			occurred_at TIMESTAMP NOT NULL,
			recorded_at TIMESTAMP NOT NULL DEFAULT NOW(),
			CONSTRAINT idx_email_status_message_occurred UNIQUE (message_id, status, occurred_at)
		)
	`)
	if err != nil {
		return fmt.Errorf("failed to create email_status_events table: %w", err)
	}

	// Create email_templates table
	_, err = r.db.ExecContext(ctx, `
		CREATE TABLE IF NOT EXISTS email_templates (
			template_id VARCHAR(100) PRIMARY KEY,
			name VARCHAR(200) NOT NULL,
			description TEXT,
			subject_template TEXT NOT NULL,
			plain_template TEXT NOT NULL,
			html_template TEXT NOT NULL,
			created_at TIMESTAMP NOT NULL DEFAULT NOW(),
			updated_at TIMESTAMP NOT NULL DEFAULT NOW(),
			active BOOLEAN NOT NULL DEFAULT TRUE
		)
	`)
	if err != nil {
		return fmt.Errorf("failed to create email_templates table: %w", err)
	}

	return nil
}

// CreateDelivery inserts a new email delivery record
func (r *EmailRepository) CreateDelivery(ctx context.Context, delivery *model.DeliveryRecord) error {
	query := `
		INSERT INTO email_deliveries (
			message_id, idempotency_key, recipient, sender, subject, template_id, 
			provider, status, status_details, created_at, updated_at, retry_count, next_retry_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13)
		ON CONFLICT (idempotency_key) DO NOTHING
	`

	_, err := r.db.ExecContext(ctx, query,
		delivery.MessageID, delivery.IdempotencyKey, delivery.Recipient, delivery.Sender,
		delivery.Subject, delivery.TemplateID, delivery.Provider, delivery.Status,
		delivery.StatusDetails, delivery.CreatedAt, delivery.UpdatedAt,
		delivery.RetryCount, delivery.NextRetryAt,
	)
	if err != nil {
		return fmt.Errorf("failed to create delivery record: %w", err)
	}

	return nil
}

// UpdateDeliveryStatus updates the status of an email delivery
func (r *EmailRepository) UpdateDeliveryStatus(ctx context.Context, messageID, status, details string) error {
	query := `
		UPDATE email_deliveries 
		SET status = $2, status_details = $3, updated_at = NOW() 
		WHERE message_id = $1
	`

	_, err := r.db.ExecContext(ctx, query, messageID, status, details)
	if err != nil {
		return fmt.Errorf("failed to update delivery status: %w", err)
	}

	return nil
}

// UpdateRetryInfo updates the retry information for a delivery
func (r *EmailRepository) UpdateRetryInfo(ctx context.Context, messageID string, retryCount int, nextRetryAt time.Time) error {
	query := `
		UPDATE email_deliveries 
		SET retry_count = $2, next_retry_at = $3, updated_at = NOW() 
		WHERE message_id = $1
	`

	_, err := r.db.ExecContext(ctx, query, messageID, retryCount, nextRetryAt)
	if err != nil {
		return fmt.Errorf("failed to update retry info: %w", err)
	}

	return nil
}

// AddStatusEvent records a new status event for an email
func (r *EmailRepository) AddStatusEvent(ctx context.Context, event *model.StatusEvent) error {
	query := `
		INSERT INTO email_status_events (
			message_id, status, provider, details, occurred_at
		) VALUES ($1, $2, $3, $4, $5)
		ON CONFLICT (message_id, status, occurred_at) DO NOTHING
	`

	_, err := r.db.ExecContext(ctx, query,
		event.MessageID, event.Status, event.Provider, event.Details, event.OccurredAt,
	)
	if err != nil {
		return fmt.Errorf("failed to add status event: %w", err)
	}

	return nil
}

// GetDeliveryByID retrieves a delivery record by message ID
func (r *EmailRepository) GetDeliveryByID(ctx context.Context, messageID string) (*model.DeliveryRecord, error) {
	query := `
		SELECT 
			message_id, idempotency_key, recipient, sender, subject, template_id, 
			provider, status, status_details, created_at, updated_at, retry_count, next_retry_at
		FROM email_deliveries 
		WHERE message_id = $1
	`

	row := r.db.QueryRowContext(ctx, query, messageID)

	var delivery model.DeliveryRecord
	var nextRetryAt sql.NullTime

	err := row.Scan(
		&delivery.MessageID, &delivery.IdempotencyKey, &delivery.Recipient, &delivery.Sender,
		&delivery.Subject, &delivery.TemplateID, &delivery.Provider, &delivery.Status,
		&delivery.StatusDetails, &delivery.CreatedAt, &delivery.UpdatedAt,
		&delivery.RetryCount, &nextRetryAt,
	)

	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get delivery: %w", err)
	}

	if nextRetryAt.Valid {
		delivery.NextRetryAt = nextRetryAt.Time
	}

	return &delivery, nil
}

// GetDeliveryByIdempotencyKey retrieves a delivery record by idempotency key
func (r *EmailRepository) GetDeliveryByIdempotencyKey(ctx context.Context, idempotencyKey string) (*model.DeliveryRecord, error) {
	query := `
		SELECT 
			message_id, idempotency_key, recipient, sender, subject, template_id, 
			provider, status, status_details, created_at, updated_at, retry_count, next_retry_at
		FROM email_deliveries 
		WHERE idempotency_key = $1
	`

	row := r.db.QueryRowContext(ctx, query, idempotencyKey)

	var delivery model.DeliveryRecord
	var nextRetryAt sql.NullTime

	err := row.Scan(
		&delivery.MessageID, &delivery.IdempotencyKey, &delivery.Recipient, &delivery.Sender,
		&delivery.Subject, &delivery.TemplateID, &delivery.Provider, &delivery.Status,
		&delivery.StatusDetails, &delivery.CreatedAt, &delivery.UpdatedAt,
		&delivery.RetryCount, &nextRetryAt,
	)

	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get delivery: %w", err)
	}

	if nextRetryAt.Valid {
		delivery.NextRetryAt = nextRetryAt.Time
	}

	return &delivery, nil
}

// GetDeliveriesToRetry retrieves deliveries that need to be retried
func (r *EmailRepository) GetDeliveriesToRetry(ctx context.Context, limit int) ([]*model.DeliveryRecord, error) {
	query := `
		SELECT 
			message_id, idempotency_key, recipient, sender, subject, template_id, 
			provider, status, status_details, created_at, updated_at, retry_count, next_retry_at
		FROM email_deliveries 
		WHERE status = 'failed' AND next_retry_at <= NOW() AND retry_count < 5
		ORDER BY next_retry_at
		LIMIT $1
	`

	rows, err := r.db.QueryContext(ctx, query, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to query deliveries to retry: %w", err)
	}
	defer rows.Close()

	var deliveries []*model.DeliveryRecord

	for rows.Next() {
		var delivery model.DeliveryRecord
		var nextRetryAt sql.NullTime

		err := rows.Scan(
			&delivery.MessageID, &delivery.IdempotencyKey, &delivery.Recipient, &delivery.Sender,
			&delivery.Subject, &delivery.TemplateID, &delivery.Provider, &delivery.Status,
			&delivery.StatusDetails, &delivery.CreatedAt, &delivery.UpdatedAt,
			&delivery.RetryCount, &nextRetryAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan delivery: %w", err)
		}

		if nextRetryAt.Valid {
			delivery.NextRetryAt = nextRetryAt.Time
		}

		deliveries = append(deliveries, &delivery)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating over rows: %w", err)
	}

	return deliveries, nil
}

// GetStatusEvents retrieves status events for a message
func (r *EmailRepository) GetStatusEvents(ctx context.Context, messageID string) ([]*model.StatusEvent, error) {
	query := `
		SELECT 
			id, message_id, status, provider, details, occurred_at, recorded_at
		FROM email_status_events 
		WHERE message_id = $1
		ORDER BY occurred_at
	`

	rows, err := r.db.QueryContext(ctx, query, messageID)
	if err != nil {
		return nil, fmt.Errorf("failed to query status events: %w", err)
	}
	defer rows.Close()

	var events []*model.StatusEvent

	for rows.Next() {
		var event model.StatusEvent

		err := rows.Scan(
			&event.ID, &event.MessageID, &event.Status, &event.Provider,
			&event.Details, &event.OccurredAt, &event.RecordedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan status event: %w", err)
		}

		events = append(events, &event)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating over rows: %w", err)
	}

	return events, nil
}
