package router

import (
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"rcprotocol/services/go-approval/internal/downstream"
	"rcprotocol/services/go-approval/internal/handler"
)

// stub repo that satisfies handler.ApprovalRepository
type stubRepo struct{}

func (s *stubRepo) Create(_ interface{}, _ interface{}) error { return nil }
func (s *stubRepo) GetByID(_ interface{}, _ string) (interface{}, error) {
	return nil, nil
}
func (s *stubRepo) ExistsPending(_ interface{}, _, _, _ string) (bool, error) {
	return false, nil
}
func (s *stubRepo) List(_ interface{}, _, _, _ string, _, _ int) (interface{}, int, error) {
	return nil, 0, nil
}
func (s *stubRepo) ListByResource(_ interface{}, _, _ string) (interface{}, error) {
	return nil, nil
}
func (s *stubRepo) UpdateStatus(_ interface{}, _, _, _ string, _, _, _ interface{}, _ interface{}) (bool, error) {
	return false, nil
}

func newTestRouter() http.Handler {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	ds := downstream.New("http://localhost:9999")
	// We need a real handler.ApprovalRepository. Let's use a minimal mock.
	h := handler.NewApprovalHandler(nil, ds)
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
