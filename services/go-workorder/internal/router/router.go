package router

import (
	"log/slog"
	"net/http"

	"github.com/go-chi/chi/v5"
	"rcprotocol/services/go-workorder/internal/handler"
	"rcprotocol/services/go-workorder/internal/middleware"
)

// New 组装路由
func New(logger *slog.Logger, h *handler.WorkorderHandler) http.Handler {
	r := chi.NewRouter()

	r.Use(middleware.Logging(logger))

	r.Get("/healthz", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
		w.Write([]byte("ok"))
	})

	r.Route("/workorders", func(r chi.Router) {
		r.Post("/", h.Create)
		r.Get("/", h.List)
		r.Get("/by-asset", h.ListByAsset)
		r.Get("/{workorderId}", h.GetByID)
		r.Post("/{workorderId}/assign", h.Assign)
		r.Post("/{workorderId}/advance", h.Advance)
		r.Post("/{workorderId}/close", h.Close)
		r.Post("/{workorderId}/cancel", h.Cancel)
	})

	return r
}
