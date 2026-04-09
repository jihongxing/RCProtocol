package router

import (
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"rcprotocol/services/go-workorder/internal/downstream"
	"rcprotocol/services/go-workorder/internal/handler"
)

func newTestRouter() http.Handler {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	rcApi := downstream.NewRcApiClient("http://localhost:0")
	h := handler.NewWorkorderHandler(nil, rcApi)
	return New(logger, h)
}

func TestHealthz(t *testing.T) {
	r := newTestRouter()

	req := httptest.NewRequest("GET", "/healthz", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != 200 {
		t.Fatalf("expected 200, got %d", w.Code)
	}
	if w.Body.String() != "ok" {
		t.Errorf("body = %q, want ok", w.Body.String())
	}
}

func TestNotFoundRoute(t *testing.T) {
	r := newTestRouter()

	req := httptest.NewRequest("GET", "/nonexistent", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != 404 {
		t.Fatalf("expected 404, got %d", w.Code)
	}
}
