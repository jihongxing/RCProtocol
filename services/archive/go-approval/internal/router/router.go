package router

import (
	"log/slog"
	"net/http"

	"github.com/go-chi/chi/v5"
	"rcprotocol/services/go-approval/internal/handler"
	"rcprotocol/services/go-approval/internal/middleware"
)

// New 组装路由
func New(logger *slog.Logger, h *handler.ApprovalHandler) http.Handler {
	r := chi.NewRouter()

	r.Use(middleware.Logging(logger))

	r.Get("/healthz", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
		w.Write([]byte("ok"))
	})

	r.Route("/approvals", func(r chi.Router) {
		r.Post("/", h.Create)
		r.Get("/", h.List)
		r.Get("/by-resource", h.ListByResource)
		r.Get("/{approvalId}", h.GetByID)
		r.Post("/{approvalId}/approve", h.Approve)
		r.Post("/{approvalId}/reject", h.Reject)
	})

	return r
}
