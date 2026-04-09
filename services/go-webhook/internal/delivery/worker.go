package delivery

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"time"

	"rcprotocol/services/go-webhook/internal/config"
	"rcprotocol/services/go-webhook/internal/queue"
)

type Worker struct {
	db          *queue.DB
	logger      *slog.Logger
	workerCount int
	retryConfig config.RetryConfig
	httpClient  *http.Client
}

func NewWorker(db *queue.DB, logger *slog.Logger, workerCount int, retryConfig config.RetryConfig) *Worker {
	return &Worker{
		db:          db,
		logger:      logger,
		workerCount: workerCount,
		retryConfig: retryConfig,
		httpClient: &http.Client{
			Timeout: time.Duration(retryConfig.TimeoutSeconds) * time.Second,
		},
	}
}

// Start begins processing webhook deliveries
func (w *Worker) Start(ctx context.Context) {
	w.logger.Info("webhook delivery worker started", slog.Int("workers", w.workerCount))

	for i := 0; i < w.workerCount; i++ {
		go w.processLoop(ctx, i)
	}

	<-ctx.Done()
	w.logger.Info("webhook delivery worker stopped")
}

func (w *Worker) processLoop(ctx context.Context, workerID int) {
	ticker := time.NewTicker(2 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			w.processBatch(ctx, workerID)
		}
	}
}

func (w *Worker) processBatch(ctx context.Context, workerID int) {
	deliveries, err := w.db.FetchPendingDeliveries(ctx, 10)
	if err != nil {
		w.logger.Error("failed to fetch pending deliveries",
			slog.Int("worker_id", workerID),
			slog.String("error", err.Error()))
		return
	}

	if len(deliveries) == 0 {
		return
	}

	w.logger.Debug("processing deliveries",
		slog.Int("worker_id", workerID),
		slog.Int("count", len(deliveries)))

	for _, delivery := range deliveries {
		if err := w.processDelivery(ctx, delivery); err != nil {
			w.logger.Error("failed to process delivery",
				slog.String("delivery_id", delivery.DeliveryID),
				slog.String("error", err.Error()))
		}
	}
}

func (w *Worker) processDelivery(ctx context.Context, delivery queue.WebhookDelivery) error {
	// Fetch webhook configuration
	config, err := w.db.FetchWebhookConfig(ctx, delivery.WebhookID)
	if err != nil {
		return fmt.Errorf("failed to fetch webhook config: %w", err)
	}

	// Check if webhook is active
	if config.Status != "Active" {
		w.logger.Warn("webhook is not active, skipping",
			slog.String("webhook_id", delivery.WebhookID),
			slog.String("status", config.Status))
		return nil
	}

	// Send webhook
	statusCode, responseBody, err := w.sendWebhook(ctx, config, delivery)

	// Update delivery status
	if err != nil || statusCode < 200 || statusCode >= 300 {
		return w.handleFailure(ctx, delivery, statusCode, responseBody, err)
	}

	return w.handleSuccess(ctx, delivery, statusCode, responseBody)
}

func (w *Worker) sendWebhook(ctx context.Context, config *queue.WebhookConfig, delivery queue.WebhookDelivery) (int, string, error) {
	// Create request
	req, err := http.NewRequestWithContext(ctx, "POST", config.URL, bytes.NewReader(delivery.Payload))
	if err != nil {
		return 0, "", fmt.Errorf("failed to create request: %w", err)
	}

	// Set headers
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", "RCProtocol-Webhook/1.0")
	req.Header.Set("X-Webhook-Event", delivery.EventType)
	req.Header.Set("X-Webhook-Delivery", delivery.DeliveryID)

	// Add signature if secret is configured
	if config.Secret != nil && *config.Secret != "" {
		signature := w.generateSignature(delivery.Payload, *config.Secret)
		req.Header.Set("X-Webhook-Signature", signature)
	}

	// Send request
	resp, err := w.httpClient.Do(req)
	if err != nil {
		return 0, "", fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	// Read response body (limit to 1KB)
	bodyBytes, _ := io.ReadAll(io.LimitReader(resp.Body, 1024))
	responseBody := string(bodyBytes)

	w.logger.Info("webhook sent",
		slog.String("delivery_id", delivery.DeliveryID),
		slog.String("url", config.URL),
		slog.Int("status_code", resp.StatusCode))

	return resp.StatusCode, responseBody, nil
}

func (w *Worker) generateSignature(payload []byte, secret string) string {
	h := hmac.New(sha256.New, []byte(secret))
	h.Write(payload)
	return "sha256=" + hex.EncodeToString(h.Sum(nil))
}

func (w *Worker) handleSuccess(ctx context.Context, delivery queue.WebhookDelivery, statusCode int, responseBody string) error {
	return w.db.UpdateDeliverySuccess(ctx, delivery.DeliveryID, statusCode, responseBody)
}

func (w *Worker) handleFailure(ctx context.Context, delivery queue.WebhookDelivery, statusCode int, responseBody string, sendErr error) error {
	attempts := delivery.Attempts + 1
	maxRetriesReached := attempts >= w.retryConfig.MaxRetries

	var nextRetryAt *time.Time
	if !maxRetriesReached && attempts-1 < len(w.retryConfig.BackoffSeconds) {
		backoffSeconds := w.retryConfig.BackoffSeconds[attempts-1]
		retry := time.Now().Add(time.Duration(backoffSeconds) * time.Second)
		nextRetryAt = &retry
	}

	if sendErr != nil {
		responseBody = sendErr.Error()
	}

	var statusCodePtr *int
	if statusCode > 0 {
		statusCodePtr = &statusCode
	}

	w.logger.Warn("webhook delivery failed",
		slog.String("delivery_id", delivery.DeliveryID),
		slog.Int("attempts", attempts),
		slog.Bool("max_retries_reached", maxRetriesReached),
		slog.String("error", responseBody))

	return w.db.UpdateDeliveryFailure(ctx, delivery.DeliveryID, statusCodePtr, responseBody, nextRetryAt, maxRetriesReached)
}

// EnqueueWebhook creates a new webhook delivery for an event
func EnqueueWebhook(ctx context.Context, db *queue.DB, webhookID, eventType string, eventData interface{}) error {
	payload, err := json.Marshal(map[string]interface{}{
		"event_type": eventType,
		"timestamp":  time.Now().UTC().Format(time.RFC3339),
		"data":       eventData,
	})
	if err != nil {
		return fmt.Errorf("failed to marshal payload: %w", err)
	}

	return db.CreateDelivery(ctx, webhookID, eventType, payload)
}
