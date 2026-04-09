package main

import (
	"log/slog"
	"net/http"
	"os"

	"rcprotocol/services/go-bff/internal/config"
	"rcprotocol/services/go-bff/internal/router"
	"rcprotocol/services/go-bff/internal/upstream"
)

func main() {
	cfg := config.Load()

	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	}))
	slog.SetDefault(logger)

	client := upstream.New(cfg.RcApiBaseURL, cfg.GoIamBaseURL)
	handler := router.New(logger, client)

	logger.Info("go-bff starting",
		slog.String("port", cfg.Port),
		slog.String("rc_api_base_url", cfg.RcApiBaseURL),
		slog.String("go_iam_base_url", cfg.GoIamBaseURL),
	)

	if err := http.ListenAndServe(cfg.Port, handler); err != nil {
		logger.Error("server failed", slog.String("error", err.Error()))
		os.Exit(1)
	}
}
