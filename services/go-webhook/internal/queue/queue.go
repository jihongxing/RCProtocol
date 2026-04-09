package queue

import (
	"context"
	"log/slog"
	"time"
)

type Queue struct {
	db     *DB
	logger *slog.Logger
}

func NewQueue(db *DB, logger *slog.Logger) *Queue {
	return &Queue{
		db:     db,
		logger: logger,
	}
}

// Start begins polling for pending webhook deliveries
func (q *Queue) Start(ctx context.Context) {
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()

	q.logger.Info("webhook queue started")

	for {
		select {
		case <-ctx.Done():
			q.logger.Info("webhook queue stopped")
			return
		case <-ticker.C:
			// Polling is handled by the delivery worker
			// This queue just ensures the service stays alive
		}
	}
}
