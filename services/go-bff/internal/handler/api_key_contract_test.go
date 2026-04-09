package handler

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/chi/v5"
	"rcprotocol/services/go-bff/internal/auth"
	"rcprotocol/services/go-bff/internal/upstream"
)

func setupMockRcAPIForBrandContract(t *testing.T, checker func(*http.Request)) *httptest.Server {
	t.Helper()
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		checker(r)
		switch r.URL.Path {
		case "/brands/b-001":
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(map[string]string{
				"brand_id":      "b-001",
				"brand_name":    "TestBrand",
				"contact_email": "brand@test.com",
				"industry":      "Watches",
				"status":        "Active",
				"created_at":    "2024-01-01T00:00:00Z",
				"updated_at":    "2024-01-02T00:00:00Z",
			})
		case "/brands/b-001/api-keys":
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(map[string]interface{}{
				"keys": []map[string]interface{}{{
					"key_id":      "key-001",
					"key_prefix":  "rcpk_live_1234****",
					"status":      "Active",
					"created_at":  "2024-01-01T00:00:00Z",
					"last_used_at": nil,
					"revoked_at":  nil,
				}},
			})
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
}

func TestBrandDetail_ForwardsGatewayApiKeyHeadersToRcAPI(t *testing.T) {
	var seenHash bool
	var seenVerified bool
	var seenAuth bool
	var seenTrace bool

	srv := setupMockRcAPIForBrandContract(t, func(r *http.Request) {
		if r.Header.Get("X-Api-Key-Hash") == "hash-123" {
			seenHash = true
		}
		if r.Header.Get("X-Api-Key-Verified") == "hash-only" {
			seenVerified = true
		}
		if r.Header.Get("Authorization") == "Bearer test-token" {
			seenAuth = true
		}
		if r.Header.Get("X-Trace-Id") == "trace-123" {
			seenTrace = true
		}
	})
	defer srv.Close()

	client := upstream.New(srv.URL, srv.URL)
	h := NewBrandHandler(client)

	req := httptest.NewRequest(http.MethodGet, "/console/brands/b-001", nil)
	req.Header.Set("Authorization", "Bearer test-token")
	req.Header.Set("X-Trace-Id", "trace-123")
	req.Header.Set("X-Api-Key-Hash", "hash-123")
	req.Header.Set("X-Api-Key-Verified", "hash-only")
	ctx := auth.NewContext(req.Context(), &auth.Claims{Sub: "brand-user", Role: "Brand", BrandID: "b-001"})
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("brandId", "b-001")
	ctx = context.WithValue(ctx, chi.RouteCtxKey, rctx)
	req = req.WithContext(ctx)

	rr := httptest.NewRecorder()
	h.GetBrandDetail(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d; body: %s", rr.Code, rr.Body.String())
	}
	if !(seenHash && seenVerified && seenAuth && seenTrace) {
		t.Fatalf("expected forwarded headers auth=%v trace=%v hash=%v verified=%v", seenAuth, seenTrace, seenHash, seenVerified)
	}
}
