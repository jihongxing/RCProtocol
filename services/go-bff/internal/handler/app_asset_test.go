package handler

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/chi/v5"
	"rcprotocol/services/go-bff/internal/auth"
	"rcprotocol/services/go-bff/internal/upstream"
)

func setupMockRcAPI(t *testing.T) *httptest.Server {
	t.Helper()
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/assets":
			resp := map[string]interface{}{
				"items": []map[string]interface{}{
					{
						"asset_id":      "a-001",
						"brand_id":      "b-001",
						"product_id":    "p-001",
						"current_state": "Activated",
						"thumbnail_url": "https://img.example.com/a-001.jpg",
					},
					{
						"asset_id":              "a-002",
						"brand_id":              "b-001",
						"product_id":            "p-002",
						"current_state":         "Disputed",
						"external_product_name": "Diamond Ring",
					},
				},
				"total": 2,
			}
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(resp)
		case "/brands/batch":
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode([]map[string]string{{"brand_id": "b-001", "brand_name": "Luxury Brand"}})
		case "/brands/b-001":
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(map[string]string{"brand_name": "Luxury Brand"})
		case "/assets/a-001":
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(map[string]interface{}{
				"asset_id":      "a-001",
				"brand_id":      "b-001",
				"product_id":    "p-001",
				"current_state": "Activated",
				"uid":           "uid-abc-123",
				"created_at":    "2024-01-15T10:30:00Z",
			})
		case "/assets/a-ext-1":
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(map[string]interface{}{
				"asset_id":              "a-ext-1",
				"brand_id":              "b-001",
				"product_id":            "p-ext-1",
				"current_state":         "Activated",
				"uid":                   "uid-ext-1",
				"created_at":            "2024-01-15T10:30:00Z",
				"external_product_name": "External Product Name",
			})
		case "/assets/a-no-ext":
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(map[string]interface{}{
				"asset_id":      "a-no-ext",
				"brand_id":      "b-001",
				"product_id":    "p-no-ext",
				"current_state": "PreMinted",
				"uid":           "uid-no-ext",
				"created_at":    "2024-01-15T10:30:00Z",
			})
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
}

func newTestRequest(method, path string, claims *auth.Claims) *http.Request {
	req := httptest.NewRequest(method, path, nil)
	ctx := auth.NewContext(req.Context(), claims)
	return req.WithContext(ctx)
}

func TestListAssets_NormalAggregation(t *testing.T) {
	srv := setupMockRcAPI(t)
	defer srv.Close()

	client := upstream.New(srv.URL, srv.URL)
	h := NewAppAssetHandler(client)

	req := newTestRequest(http.MethodGet, "/app/assets", &auth.Claims{Sub: "user-1", Role: "Consumer"})
	rr := httptest.NewRecorder()
	h.ListAssets(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rr.Code)
	}

	var body struct {
		Items []AssetVM `json:"items"`
		Total int       `json:"total"`
	}
	if err := json.Unmarshal(rr.Body.Bytes(), &body); err != nil {
		t.Fatalf("invalid JSON response: %v", err)
	}
	if body.Total != 2 || len(body.Items) != 2 {
		t.Fatalf("unexpected list body: %+v", body)
	}

	a1 := body.Items[0]
	if a1.AssetID != "a-001" || a1.BrandName != "Luxury Brand" || a1.ProductName != "p-001" {
		t.Fatalf("unexpected item1: %+v", a1)
	}
	if a1.State != "Activated" || a1.StateLabel != "已激活" || len(a1.DisplayBadges) != 1 || a1.DisplayBadges[0] != "verified" {
		t.Fatalf("unexpected state mapping item1: %+v", a1)
	}

	a2 := body.Items[1]
	if a2.ProductName != "Diamond Ring" {
		t.Fatalf("expected external product name preferred, got %+v", a2)
	}
	if a2.State != "Disputed" || a2.StateLabel != "争议中" || len(a2.DisplayBadges) != 1 || a2.DisplayBadges[0] != "frozen" {
		t.Fatalf("unexpected state mapping item2: %+v", a2)
	}
}

func TestListAssets_UpstreamUnavailable(t *testing.T) {
	client := upstream.New("http://127.0.0.1:1", "http://127.0.0.1:1")
	h := NewAppAssetHandler(client)

	req := newTestRequest(http.MethodGet, "/app/assets", &auth.Claims{Sub: "user-1", Role: "Consumer"})
	rr := httptest.NewRecorder()
	h.ListAssets(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rr.Code)
	}
}

func TestListAssets_BrandNameFallback(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/assets":
			_ = json.NewEncoder(w).Encode(map[string]interface{}{
				"items": []map[string]interface{}{{
					"asset_id":              "a-010",
					"brand_id":              "b-missing",
					"product_id":            "p-010",
					"current_state":         "PreMinted",
					"external_product_name": "Some Product",
				}},
				"total": 1,
			})
		case "/brands/batch":
			_ = json.NewEncoder(w).Encode([]map[string]string{})
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer srv.Close()

	client := upstream.New(srv.URL, srv.URL)
	h := NewAppAssetHandler(client)
	req := newTestRequest(http.MethodGet, "/app/assets", &auth.Claims{Sub: "user-1", Role: "Consumer"})
	rr := httptest.NewRecorder()
	h.ListAssets(rr, req)

	var body struct{ Items []AssetVM `json:"items"` }
	_ = json.Unmarshal(rr.Body.Bytes(), &body)
	if len(body.Items) != 1 || body.Items[0].BrandName != "b-missing" || body.Items[0].ProductName != "Some Product" {
		t.Fatalf("unexpected fallback body: %+v", body)
	}
}

func TestListAssets_PaginationForwarded(t *testing.T) {
	var gotPage, gotPageSize, gotOwnerID string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/assets" {
			gotPage = r.URL.Query().Get("page")
			gotPageSize = r.URL.Query().Get("page_size")
			gotOwnerID = r.URL.Query().Get("owner_id")
			_ = json.NewEncoder(w).Encode(map[string]interface{}{"items": []interface{}{}, "total": 0})
			return
		}
		w.WriteHeader(http.StatusNotFound)
	}))
	defer srv.Close()

	client := upstream.New(srv.URL, srv.URL)
	h := NewAppAssetHandler(client)
	req := newTestRequest(http.MethodGet, "/app/assets?page=3&page_size=50", &auth.Claims{Sub: "user-42", Role: "Consumer"})
	rr := httptest.NewRecorder()
	h.ListAssets(rr, req)

	if gotPage != "3" || gotPageSize != "50" || gotOwnerID != "user-42" {
		t.Fatalf("unexpected forwarded pagination page=%q size=%q owner=%q", gotPage, gotPageSize, gotOwnerID)
	}
}

func TestListAssets_ContentType(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/assets" {
			w.Header().Set("Content-Type", "application/json")
			fmt.Fprint(w, `{"items":[],"total":0}`)
			return
		}
		w.WriteHeader(http.StatusNotFound)
	}))
	defer srv.Close()

	client := upstream.New(srv.URL, srv.URL)
	h := NewAppAssetHandler(client)
	req := newTestRequest(http.MethodGet, "/app/assets", &auth.Claims{Sub: "user-1", Role: "Consumer"})
	rr := httptest.NewRecorder()
	h.ListAssets(rr, req)

	if rr.Header().Get("Content-Type") != "application/json; charset=utf-8" {
		t.Fatalf("unexpected content-type %q", rr.Header().Get("Content-Type"))
	}
}

func TestGetAsset_NormalDetail(t *testing.T) {
	srv := setupMockRcAPI(t)
	defer srv.Close()

	client := upstream.New(srv.URL, srv.URL)
	h := NewAppAssetHandler(client)

	req := newTestRequest(http.MethodGet, "/app/assets/a-001", &auth.Claims{Sub: "user-1", Role: "Consumer"})
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("assetId", "a-001")
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))
	_rr := httptest.NewRecorder()
	h.GetAsset(_rr, req)

	if _rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d; body: %s", _rr.Code, _rr.Body.String())
	}

	var detail AssetDetailVM
	if err := json.Unmarshal(_rr.Body.Bytes(), &detail); err != nil {
		t.Fatalf("invalid JSON response: %v", err)
	}
	if detail.AssetID != "a-001" || detail.BrandName != "Luxury Brand" || detail.ProductName != "p-001" {
		t.Fatalf("unexpected detail body: %+v", detail)
	}
	if detail.State != "Activated" || detail.StateLabel != "已激活" || len(detail.DisplayBadges) != 1 || detail.DisplayBadges[0] != "verified" {
		t.Fatalf("unexpected state detail: %+v", detail)
	}
}

func TestGetAsset_NotFound(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	defer srv.Close()

	client := upstream.New(srv.URL, srv.URL)
	h := NewAppAssetHandler(client)
	req := newTestRequest(http.MethodGet, "/app/assets/nonexistent", &auth.Claims{Sub: "user-1", Role: "Consumer"})
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("assetId", "nonexistent")
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))
	rr := httptest.NewRecorder()
	h.GetAsset(rr, req)

	if rr.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", rr.Code)
	}
}

func TestGetAsset_ExternalProductNamePreferred(t *testing.T) {
	srv := setupMockRcAPI(t)
	defer srv.Close()
	client := upstream.New(srv.URL, srv.URL)
	h := NewAppAssetHandler(client)

	req := newTestRequest(http.MethodGet, "/app/assets/a-ext-1", &auth.Claims{Sub: "user-1", Role: "Consumer"})
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("assetId", "a-ext-1")
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))
	rr := httptest.NewRecorder()
	h.GetAsset(rr, req)

	var detail AssetDetailVM
	_ = json.Unmarshal(rr.Body.Bytes(), &detail)
	if detail.ProductName != "External Product Name" {
		t.Fatalf("expected external product preferred, got %+v", detail)
	}
}

func TestGetAsset_ProductIDFallbackWithoutExternalName(t *testing.T) {
	srv := setupMockRcAPI(t)
	defer srv.Close()
	client := upstream.New(srv.URL, srv.URL)
	h := NewAppAssetHandler(client)

	req := newTestRequest(http.MethodGet, "/app/assets/a-no-ext", &auth.Claims{Sub: "user-1", Role: "Consumer"})
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("assetId", "a-no-ext")
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))
	rr := httptest.NewRecorder()
	h.GetAsset(rr, req)

	var detail AssetDetailVM
	_ = json.Unmarshal(rr.Body.Bytes(), &detail)
	if detail.ProductName != "p-no-ext" || detail.StateLabel != "预铸造" {
		t.Fatalf("unexpected fallback detail: %+v", detail)
	}
}
