package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

// okHandler is a simple handler that writes 200 OK.
var okHandler = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
})

func TestRateLimitSingleRequest(t *testing.T) {
	// A generous limiter: single request should always pass.
	handler := RateLimit(100, 100)(okHandler)

	req := httptest.NewRequest(http.MethodGet, "/api/test", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}

	if v := rec.Header().Get("X-RateLimit-Limit"); v != "100" {
		t.Errorf("expected X-RateLimit-Limit=100, got %q", v)
	}

	remaining := rec.Header().Get("X-RateLimit-Remaining")
	if remaining == "" {
		t.Error("expected X-RateLimit-Remaining header to be set")
	}
}

func TestRateLimitExceeded(t *testing.T) {
	// rps=1, burst=1: the first request consumes the single token,
	// subsequent requests should be rate-limited.
	handler := RateLimit(1, 1)(okHandler)

	got429 := false
	for i := 0; i < 3; i++ {
		req := httptest.NewRequest(http.MethodGet, "/api/test", nil)
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, req)

		if rec.Code == http.StatusTooManyRequests {
			got429 = true

			if v := rec.Header().Get("Retry-After"); v != "1" {
				t.Errorf("expected Retry-After=1, got %q", v)
			}
			if v := rec.Header().Get("X-RateLimit-Remaining"); v != "0" {
				t.Errorf("expected X-RateLimit-Remaining=0 on 429, got %q", v)
			}
		}
	}

	if !got429 {
		t.Fatal("expected at least one 429 response in 3 requests with rps=1/burst=1")
	}
}

func TestRateLimitHealthzExempt(t *testing.T) {
	// Even with an exhausted limiter, /healthz must pass.
	handler := RateLimit(1, 1)(okHandler)

	// Exhaust the limiter with non-healthz requests.
	for i := 0; i < 5; i++ {
		req := httptest.NewRequest(http.MethodGet, "/api/drain", nil)
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, req)
	}

	// /healthz should still succeed.
	req := httptest.NewRequest(http.MethodGet, "/healthz", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected /healthz to return 200 even when rate-limited, got %d", rec.Code)
	}

	// /healthz responses should NOT carry rate-limit headers.
	if v := rec.Header().Get("X-RateLimit-Limit"); v != "" {
		t.Errorf("expected no X-RateLimit-Limit on /healthz, got %q", v)
	}
}
