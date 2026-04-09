package queue

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	_ "github.com/lib/pq"
)

type DB struct {
	*sql.DB
}

func NewDB(databaseURL string) (*DB, error) {
	db, err := sql.Open("postgres", databaseURL)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	if err := db.Ping(); err != nil {
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	db.SetMaxOpenConns(25)
	db.SetMaxIdleConns(5)
	db.SetConnMaxLifetime(5 * time.Minute)

	return &DB{db}, nil
}

type WebhookDelivery struct {
	DeliveryID       string
	WebhookID        string
	EventType        string
	Payload          json.RawMessage
	Status           string
	Attempts         int
	LastAttemptAt    *time.Time
	NextRetryAt      *time.Time
	ResponseStatus   *int
	ResponseBody     *string
	CreatedAt        time.Time
}

type WebhookConfig struct {
	WebhookID    string
	BrandID      string
	URL          string
	Secret       *string
	Events       []string
	Status       string
	RetryConfig  json.RawMessage
}

// FetchPendingDeliveries retrieves webhook deliveries ready for sending
func (db *DB) FetchPendingDeliveries(ctx context.Context, limit int) ([]WebhookDelivery, error) {
	query := `
		SELECT delivery_id, webhook_id, event_type, payload, status, attempts,
		       last_attempt_at, next_retry_at, response_status_code, response_body, created_at
		FROM webhook_deliveries
		WHERE status = 'Pending'
		  AND (next_retry_at IS NULL OR next_retry_at <= NOW())
		ORDER BY created_at ASC
		LIMIT $1
	`

	rows, err := db.QueryContext(ctx, query, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to query pending deliveries: %w", err)
	}
	defer rows.Close()

	var deliveries []WebhookDelivery
	for rows.Next() {
		var d WebhookDelivery
		err := rows.Scan(
			&d.DeliveryID, &d.WebhookID, &d.EventType, &d.Payload, &d.Status,
			&d.Attempts, &d.LastAttemptAt, &d.NextRetryAt,
			&d.ResponseStatus, &d.ResponseBody, &d.CreatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan delivery: %w", err)
		}
		deliveries = append(deliveries, d)
	}

	return deliveries, rows.Err()
}

// FetchWebhookConfig retrieves webhook configuration
func (db *DB) FetchWebhookConfig(ctx context.Context, webhookID string) (*WebhookConfig, error) {
	query := `
		SELECT webhook_id, brand_id, url, secret, events, status, retry_config
		FROM webhook_configs
		WHERE webhook_id = $1
	`

	var config WebhookConfig
	var events string
	err := db.QueryRowContext(ctx, query, webhookID).Scan(
		&config.WebhookID, &config.BrandID, &config.URL, &config.Secret,
		&events, &config.Status, &config.RetryConfig,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch webhook config: %w", err)
	}

	// Parse events array (PostgreSQL text array format)
	if err := json.Unmarshal([]byte(events), &config.Events); err != nil {
		return nil, fmt.Errorf("failed to parse events: %w", err)
	}

	return &config, nil
}

// UpdateDeliverySuccess marks a delivery as successfully sent
func (db *DB) UpdateDeliverySuccess(ctx context.Context, deliveryID string, statusCode int, responseBody string) error {
	query := `
		UPDATE webhook_deliveries
		SET status = 'Sent',
		    attempts = attempts + 1,
		    last_attempt_at = NOW(),
		    response_status_code = $2,
		    response_body = $3
		WHERE delivery_id = $1
	`

	_, err := db.ExecContext(ctx, query, deliveryID, statusCode, responseBody)
	return err
}

// UpdateDeliveryFailure records a failed delivery attempt
func (db *DB) UpdateDeliveryFailure(ctx context.Context, deliveryID string, statusCode *int, responseBody string, nextRetryAt *time.Time, maxRetriesReached bool) error {
	status := "Pending"
	if maxRetriesReached {
		status = "Abandoned"
	}

	query := `
		UPDATE webhook_deliveries
		SET status = $2,
		    attempts = attempts + 1,
		    last_attempt_at = NOW(),
		    next_retry_at = $3,
		    response_status_code = $4,
		    response_body = $5
		WHERE delivery_id = $1
	`

	_, err := db.ExecContext(ctx, query, deliveryID, status, nextRetryAt, statusCode, responseBody)
	return err
}

// CreateDelivery creates a new webhook delivery
func (db *DB) CreateDelivery(ctx context.Context, webhookID, eventType string, payload json.RawMessage) error {
	query := `
		INSERT INTO webhook_deliveries (webhook_id, event_type, payload, status, attempts)
		VALUES ($1, $2, $3, 'Pending', 0)
	`

	_, err := db.ExecContext(ctx, query, webhookID, eventType, payload)
	return err
}
