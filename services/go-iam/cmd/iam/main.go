package main

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"os"

	"github.com/go-chi/chi/v5"

	"rcprotocol/services/go-iam/internal/auth"
	"rcprotocol/services/go-iam/internal/config"
	"rcprotocol/services/go-iam/internal/db"
	"rcprotocol/services/go-iam/internal/handler"
	"rcprotocol/services/go-iam/internal/middleware"
	"rcprotocol/services/go-iam/internal/repo"
)

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo}))
	slog.SetDefault(logger)

	cfg, err := config.Load()
	if err != nil {
		fmt.Fprintf(os.Stderr, "config error: %v\n", err)
		os.Exit(1)
	}

	ctx := context.Background()
	pool, err := db.Connect(ctx, cfg.DatabaseURL)
	if err != nil {
		logger.Error("database connection failed", slog.String("error", err.Error()))
		os.Exit(1)
	}
	defer pool.Close()

	if err := db.Migrate(ctx, pool, "migrations"); err != nil {
		logger.Error("migration failed", slog.String("error", err.Error()))
		os.Exit(1)
	}

	// Repositories
	userRepo := repo.NewUserRepo(pool)
	orgRepo := repo.NewOrgRepo(pool)
	positionRepo := repo.NewPositionRepo(pool)
	memberRepo := repo.NewMemberRepo(pool)
	apiKeyRepo := repo.NewApiKeyRepo(pool)

	// JWT Issuer
	jwtIssuer := auth.NewIssuer(cfg.JWTSecret, cfg.JWTExpiryHours)

	// Handlers
	userH := handler.NewUserHandler(userRepo)
	orgH := handler.NewOrgHandler(orgRepo)
	posH := handler.NewPositionHandler(positionRepo, orgRepo)
	memH := handler.NewMemberHandler(memberRepo, userRepo, positionRepo)
	authH := handler.NewAuthHandler(userRepo, memberRepo, apiKeyRepo, orgRepo, jwtIssuer)
	apiKeyH := handler.NewApiKeyHandler(apiKeyRepo, orgRepo)

	// Router — routes do NOT have /api/iam prefix (Gateway strips it)
	r := chi.NewRouter()
	r.Use(middleware.Logging(logger))

	r.Get("/healthz", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok"))
	})

	r.Post("/auth/login", authH.Login)
	r.Post("/auth/validate-api-key", authH.ValidateApiKey)

	// 管理接口——需要身份认证 + Platform 角色
	r.Group(func(r chi.Router) {
		r.Use(middleware.AuthGuard)
		r.Use(middleware.RequireRole("Platform"))

		r.Post("/users", userH.Create)
		r.Get("/users", userH.List)
		r.Get("/users/{id}", userH.GetByID)
		r.Put("/users/{id}", userH.Update)
		r.Delete("/users/{id}", userH.Delete)

		r.Post("/orgs", orgH.Create)
		r.Get("/orgs", orgH.List)
		r.Get("/orgs/{id}", orgH.GetByID)
		r.Put("/orgs/{id}", orgH.Update)

		r.Post("/orgs/{org_id}/positions", posH.Create)
		r.Get("/orgs/{org_id}/positions", posH.List)

		r.Post("/orgs/{org_id}/members", memH.Bind)
		r.Get("/orgs/{org_id}/members", memH.List)
		r.Delete("/orgs/{org_id}/members/{user_id}", memH.Unbind)

		// Legacy/backoffice brand API key management.
		// These org-scoped brand_* keys are retained for administrative compatibility;
		// current Gateway runtime brand API key truth flows through Gateway hash contract
		// and downstream brand lookup rather than this go-iam route family.
		r.Post("/orgs/{org_id}/api-keys", apiKeyH.Create)
		r.Get("/orgs/{org_id}/api-keys", apiKeyH.List)
		r.Delete("/orgs/{org_id}/api-keys/{key_id}", apiKeyH.Revoke)
	})

	// Start — log port only, never log JWTSecret or DatabaseURL
	logger.Info("go-iam starting", slog.String("port", cfg.Port))
	if err := http.ListenAndServe(cfg.Port, r); err != nil {
		logger.Error("server failed", slog.String("error", err.Error()))
		os.Exit(1)
	}
}
