package main

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"syscall"
	"time"

	"rcprotocol/services/go-webhook/internal/config"
	"rcprotocol/services/go-webhook/internal/delivery"
	"rcprotocol/services/go-webhook/internal/queue"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		fmt.Fprintf(os.Stderr, "config error: %v\n", err)
		os.Exit(1)
	}

	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	}))
	slog.SetDefault(logger)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Initialize database connection
	db, err := queue.NewDB(cfg.DatabaseURL)
	if err != nil {
		logger.Error("failed to connect to database", slog.String("error", err.Error()))
		os.Exit(1)
	}
	defer db.Close()

	// Initialize webhook queue
	webhookQueue := queue.NewQueue(db, logger)

	// Initialize delivery worker
	deliveryWorker := delivery.NewWorker(db, logger, cfg.WorkerCount, cfg.RetryConfig)

	logger.Info("go-webhook starting",
		slog.Int("worker_count", cfg.WorkerCount),
		slog.String("database_url", maskDatabaseURL(cfg.DatabaseURL)),
	)

	// Start workers
	go webhookQueue.Start(ctx)
	go deliveryWorker.Start(ctx)

	// Wait for interrupt signal
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)
	<-sigChan

	logger.Info("shutting down gracefully...")
	cancel()

	// Give workers time to finish
	time.Sleep(5 * time.Second)
	logger.Info("shutdown complete")
}

func maskDatabaseURL(url string) string {
	if len(url) < 20 {
		return "***"
	}
	return url[:10] + "***" + url[len(url)-5:]
}
