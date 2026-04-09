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

func setupMockRcAPIForBrandDetail(t *testing.T, brandFail, keyFail bool) *httptest.Server {
	t.Helper()
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/brands/b-001":
			if brandFail {
				w.WriteHeader(http.StatusInternalServerError)
				return
			}
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(map[string]string{
				"brand_id":      "b-001",
				"brand_name":    "Luxury Brand Co.",
				"contact_email": "alice@luxury.com",
				"industry":      "Watches",
				"status":        "Active",
				"created_at":    "2024-01-10T08:00:00Z",
				"updated_at":    "2024-01-11T08:00:00Z",
			})
		case "/brands/b-001/api-keys":
			if keyFail {
				w.WriteHeader(http.StatusInternalServerError)
				return
			}
			w.Header().Set("Content-Type", "application/json")
			result := map[string]interface{}{
				"keys": []map[string]interface{}{
					{
						"key_id":       "ak-001",
						"key_prefix":   "rcpk_live_1234****",
						"created_at":   "2024-02-01T09:00:00Z",
						"last_used_at": "2024-06-15T12:00:00Z",
						"status":       "active",
						"revoked_at":   nil,
					},
					{
						"key_id":       "ak-002",
						"key_prefix":   "rcpk_live_5678****",
						"created_at":   "2024-03-01T10:00:00Z",
						"last_used_at": nil,
						"status":       "revoked",
						"revoked_at":   "2024-03-10T10:00:00Z",
					},
				},
			}
			_ = json.NewEncoder(w).Encode(result)
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
}

func newBrandTestRequest(method, path, brandID string, claims *auth.Claims) *http.Request {
	req := httptest.NewRequest(method, path, nil)
	ctx := auth.NewContext(req.Context(), claims)
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("brandId", brandID)
	ctx = context.WithValue(ctx, chi.RouteCtxKey, rctx)
	return req.WithContext(ctx)
}

func TestBrandDetail_Success(t *testing.T) {
	srv := setupMockRcAPIForBrandDetail(t, false, false)
	defer srv.Close()

	client := upstream.New(srv.URL, srv.URL)
	h := NewBrandHandler(client)

	req := newBrandTestRequest(http.MethodGet, "/console/brands/b-001", "b-001", &auth.Claims{
		Sub:     "user-1",
		Role:    "Brand",
		OrgID:   "org-001",
		BrandID: "b-001",
	})
	rr := httptest.NewRecorder()
	h.GetBrandDetail(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d; body: %s", rr.Code, rr.Body.String())
	}

	var vm BrandDetailViewModel
	if err := json.Unmarshal(rr.Body.Bytes(), &vm); err != nil {
		t.Fatalf("invalid JSON: %v", err)
	}

	if vm.BrandID != "b-001" || vm.BrandName != "Luxury Brand Co." || vm.ContactEmail != "alice@luxury.com" {
		t.Fatalf("unexpected brand detail: %+v", vm)
	}
	if vm.Industry != "Watches" || vm.Status != "Active" {
		t.Fatalf("unexpected industry/status: %+v", vm)
	}
	if len(vm.ApiKeys) != 2 {
		t.Fatalf("expected 2 api_keys, got %d", len(vm.ApiKeys))
	}
	if vm.ApiKeys[0].KeyPrefix == "" || vm.ApiKeys[0].KeyID != "ak-001" {
		t.Fatalf("unexpected first api key: %+v", vm.ApiKeys[0])
	}
}

func TestBrandDetail_BrandRoleForbidden(t *testing.T) {
	srv := setupMockRcAPIForBrandDetail(t, false, false)
	defer srv.Close()

	client := upstream.New(srv.URL, srv.URL)
	h := NewBrandHandler(client)

	req := newBrandTestRequest(http.MethodGet, "/console/brands/b-999", "b-999", &auth.Claims{
		Sub:     "user-1",
		Role:    "Brand",
		OrgID:   "org-001",
		BrandID: "b-001",
	})
	rr := httptest.NewRecorder()
	h.GetBrandDetail(rr, req)

	if rr.Code != http.StatusForbidden {
		t.Fatalf("expected 403, got %d; body: %s", rr.Code, rr.Body.String())
	}
}

func TestBrandDetail_PlatformRoleAccessAny(t *testing.T) {
	srv := setupMockRcAPIForBrandDetail(t, false, false)
	defer srv.Close()

	client := upstream.New(srv.URL, srv.URL)
	h := NewBrandHandler(client)

	req := newBrandTestRequest(http.MethodGet, "/console/brands/b-001", "b-001", &auth.Claims{
		Sub:   "platform-admin",
		Role:  "Platform",
		OrgID: "org-001",
	})
	rr := httptest.NewRecorder()
	h.GetBrandDetail(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d; body: %s", rr.Code, rr.Body.String())
	}
}

func TestBrandDetail_OrgInfoFailure(t *testing.T) {
	srv := setupMockRcAPIForBrandDetail(t, true, false)
	defer srv.Close()

	client := upstream.New(srv.URL, srv.URL)
	h := NewBrandHandler(client)

	req := newBrandTestRequest(http.MethodGet, "/console/brands/b-001", "b-001", &auth.Claims{
		Sub:     "user-1",
		Role:    "Brand",
		OrgID:   "org-001",
		BrandID: "b-001",
	})
	rr := httptest.NewRecorder()
	h.GetBrandDetail(rr, req)

	if rr.Code != http.StatusBadGateway {
		t.Fatalf("expected 502, got %d; body: %s", rr.Code, rr.Body.String())
	}
}

func TestBrandDetail_ApiKeyFailureDegradation(t *testing.T) {
	srv := setupMockRcAPIForBrandDetail(t, false, true)
	defer srv.Close()

	client := upstream.New(srv.URL, srv.URL)
	h := NewBrandHandler(client)

	req := newBrandTestRequest(http.MethodGet, "/console/brands/b-001", "b-001", &auth.Claims{
		Sub:     "user-1",
		Role:    "Brand",
		OrgID:   "org-001",
		BrandID: "b-001",
	})
	rr := httptest.NewRecorder()
	h.GetBrandDetail(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d; body: %s", rr.Code, rr.Body.String())
	}

	var vm BrandDetailViewModel
	if err := json.Unmarshal(rr.Body.Bytes(), &vm); err != nil {
		t.Fatalf("invalid JSON: %v", err)
	}
	if len(vm.ApiKeys) != 0 {
		t.Errorf("expected empty api_keys on failure, got %d items", len(vm.ApiKeys))
	}
}
