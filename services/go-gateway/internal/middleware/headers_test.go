package middleware

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"rcprotocol/services/go-gateway/internal/response"
)

// okHandler is declared in ratelimit_test.go (same package).

func TestWriteHeaders_POST_WithKey_Pass(t *testing.T) {
	handler := WriteHeaders(okHandler)

	req := httptest.NewRequest(http.MethodPost, "/api/brands", nil)
	req.Header.Set("X-Idempotency-Key", "idem-123")
	rr := httptest.NewRecorder()

	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", rr.Code)
	}
}

func TestWriteHeaders_POST_WithoutKey_400(t *testing.T) {
	handler := WriteHeaders(okHandler)

	req := httptest.NewRequest(http.MethodPost, "/api/brands", nil)
	rr := httptest.NewRecorder()

	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", rr.Code)
	}

	var body response.ErrorBody
	if err := json.NewDecoder(rr.Body).Decode(&body); err != nil {
		t.Fatalf("failed to decode error body: %v", err)
	}
	if body.Error.Code != response.CodeInvalidInput {
		t.Errorf("expected code %s, got %s", response.CodeInvalidInput, body.Error.Code)
	}
}

func TestWriteHeaders_PUT_WithoutKey_400(t *testing.T) {
	handler := WriteHeaders(okHandler)

	req := httptest.NewRequest(http.MethodPut, "/api/brands/123", nil)
	rr := httptest.NewRecorder()

	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", rr.Code)
	}
}

func TestWriteHeaders_PATCH_WithoutKey_400(t *testing.T) {
	handler := WriteHeaders(okHandler)

	req := httptest.NewRequest(http.MethodPatch, "/api/brands/123", nil)
	rr := httptest.NewRecorder()

	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", rr.Code)
	}
}

func TestWriteHeaders_DELETE_WithoutKey_400(t *testing.T) {
	handler := WriteHeaders(okHandler)

	req := httptest.NewRequest(http.MethodDelete, "/api/brands/123", nil)
	rr := httptest.NewRecorder()

	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", rr.Code)
	}
}

func TestWriteHeaders_GET_WithoutKey_Pass(t *testing.T) {
	handler := WriteHeaders(okHandler)

	req := httptest.NewRequest(http.MethodGet, "/api/brands", nil)
	rr := httptest.NewRecorder()

	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", rr.Code)
	}
}

func TestWriteHeaders_POST_PublicRoute_WithoutKey_Pass(t *testing.T) {
	handler := WriteHeaders(okHandler)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/verify", nil)
	rr := httptest.NewRecorder()

	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("expected 200 for public route, got %d", rr.Code)
	}
}
