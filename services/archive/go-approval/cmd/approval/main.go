package main

import (
	"context"
	"log/slog"
	"net/http"
	"os"

	"rcprotocol/services/go-approval/internal/config"
	"rcprotocol/services/go-approval/internal/db"
	"rcprotocol/services/go-approval/internal/downstream"
	"rcprotocol/services/go-approval/internal/handler"
	"rcprotocol/services/go-approval/internal/repo"
	"rcprotocol/services/go-approval/internal/router"
)

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	slog.SetDefault(logger)

	cfg, err := config.Load()
	if err != nil {
		logger.Error("failed to load config", slog.String("error", err.Error()))
		os.Exit(1)
	}

	ctx := context.Background()
	pool, err := db.Connect(ctx, cfg.DatabaseURL)
	if err != nil {
		logger.Error("failed to connect database", slog.String("error", err.Error()))
		os.Exit(1)
	}
	defer pool.Close()

	if err := db.Migrate(ctx, pool, "migrations"); err != nil {
		logger.Error("failed to run migrations", slog.String("error", err.Error()))
		os.Exit(1)
	}

	approvalRepo := repo.NewApprovalRepo(pool)
	ds := downstream.New(cfg.RcApiBaseURL)
	h := handler.NewApprovalHandler(approvalRepo, ds)
	srv := router.New(logger, h)

	logger.Info("go-approval starting",
		slog.String("port", cfg.Port),
		slog.String("rc_api_base_url", cfg.RcApiBaseURL),
	)

	if err := http.ListenAndServe(cfg.Port, srv); err != nil {
		logger.Error("server exited", slog.String("error", err.Error()))
		os.Exit(1)
	}
}
