package router

import (
	"fmt"
	"log/slog"
	"net/http"

	"github.com/go-chi/chi/v5"
	"rcprotocol/services/go-bff/internal/auth"
	"rcprotocol/services/go-bff/internal/handler"
	"rcprotocol/services/go-bff/internal/middleware"
	"rcprotocol/services/go-bff/internal/upstream"
)

// New assembles the chi router with logging middleware, healthz endpoint,
// and JWT-protected business routes. Routes are registered without /api/bff
// prefix — Gateway strips it before forwarding.
func New(logger *slog.Logger, client *upstream.UpstreamClient) http.Handler {
	r := chi.NewRouter()
	r.Use(middleware.Logging(logger))

	r.Get("/healthz", func(w http.ResponseWriter, r *http.Request) {
		_, _ = fmt.Fprint(w, "ok")
	})

	r.Group(func(r chi.Router) {
		r.Use(auth.Middleware)

		appAssetH := handler.NewAppAssetHandler(client)
		r.Get("/app/assets", appAssetH.ListAssets)
		r.Get("/app/assets/{assetId}", appAssetH.GetAsset)
		r.Post("/app/assets/{assetId}/transfer", appAssetH.TransferAsset)

		transferH := handler.NewTransferHandler(client)
		r.Get("/app/transfers/{transferId}", transferH.GetTransfer)
		r.Post("/app/transfers/confirm", transferH.ConfirmTransfer)
		r.Post("/app/transfers/reject", transferH.RejectTransfer)

		dashH := handler.NewDashboardHandler(client)
		r.Get("/console/dashboard", dashH.GetDashboard)

		brandH := handler.NewBrandHandler(client)
		r.Get("/console/brands/{brandId}", brandH.GetBrandDetail)

		prodH := handler.NewConsoleProductHandler(client)
		r.Get("/console/brands/{brandId}/products", prodH.ListProducts)

		factoryH := handler.NewFactoryTaskHandler(client)
		r.Get("/console/factory/tasks", factoryH.ListTasks)
		r.Post("/console/factory/quick-log", factoryH.CreateQuickLog)
	})

	return r
}
