package handler

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/chi/v5"
	"pgregory.net/rapid"
	"rcprotocol/services/go-bff/internal/auth"
	"rcprotocol/services/go-bff/internal/upstream"
)

// setupProductMock creates a mock rc-api that serves brand detail and product list.
func setupProductMock(t *testing.T) *httptest.Server {
	t.Helper()
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		// Brand detail: GET /brands/b-001
		case r.URL.Path == "/brands/b-001" && r.Method == http.MethodGet:
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(map[string]string{
				"brand_id":   "b-001",
				"brand_name": "TestBrand",
			})
		// Product list: GET /brands/b-001/products
		case r.URL.Path == "/brands/b-001/products" && r.Method == http.MethodGet:
			w.Header().Set("Content-Type", "application/json")
			resp := map[string]interface{}{
				"items": []map[string]string{
					{"product_id": "p-001", "product_name": "Product A", "created_at": "2024-01-01T00:00:00Z"},
					{"product_id": "p-002", "product_name": "Product B", "created_at": "2024-01-02T00:00:00Z"},
				},
				"total": 2,
			}
			_ = json.NewEncoder(w).Encode(resp)
		// Brand not found: GET /brands/nonexistent
		case r.URL.Path == "/brands/nonexistent/products":
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusNotFound)
			_ = json.NewEncoder(w).Encode(map[string]interface{}{
				"error": map[string]string{"code": "NOT_FOUND", "message": "brand not found"},
			})
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
}

// newProductTestRequest creates a request with Claims and chi URL params injected.
func newProductTestRequest(method, path string, claims *auth.Claims, brandID string) *http.Request {
	req := httptest.NewRequest(method, path, nil)
	ctx := auth.NewContext(req.Context(), claims)
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("brandId", brandID)
	ctx = context.WithValue(ctx, chi.RouteCtxKey, rctx)
	return req.WithContext(ctx)
}

// TestProductList_Normal verifies that ListProducts returns a proper ProductListVM
// with brand_name aggregated and items from upstream.
func TestProductList_Normal(t *testing.T) {
	srv := setupProductMock(t)
	defer srv.Close()

	client := upstream.New(srv.URL, srv.URL)
	h := NewConsoleProductHandler(client)

	req := newProductTestRequest(http.MethodGet, "/console/brands/b-001/products", &auth.Claims{
		Sub:     "admin-1",
		Role:    "Platform",
		OrgID:   "org-001",
		BrandID: "",
	}, "b-001")
	rr := httptest.NewRecorder()
	h.ListProducts(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d; body: %s", rr.Code, rr.Body.String())
	}

	var vm ProductListVM
	if err := json.Unmarshal(rr.Body.Bytes(), &vm); err != nil {
		t.Fatalf("invalid JSON: %v", err)
	}

	if vm.BrandName != "TestBrand" {
		t.Errorf("expected brand_name='TestBrand', got %q", vm.BrandName)
	}
	if len(vm.Items) != 2 {
		t.Fatalf("expected 2 items, got %d", len(vm.Items))
	}
	if vm.Items[0].ProductID != "p-001" {
		t.Errorf("expected first item product_id='p-001', got %q", vm.Items[0].ProductID)
	}
	if vm.Items[1].ProductName != "Product B" {
		t.Errorf("expected second item product_name='Product B', got %q", vm.Items[1].ProductName)
	}
	if vm.Total != 2 {
		t.Errorf("expected total=2, got %d", vm.Total)
	}
	if vm.Page != 1 {
		t.Errorf("expected page=1, got %d", vm.Page)
	}
	if vm.PageSize != 20 {
		t.Errorf("expected page_size=20, got %d", vm.PageSize)
	}
}

// TestProductList_BrandAccessOtherBrand_403 verifies that a Brand-role user
// accessing a brand they don't own gets 403 FORBIDDEN.
func TestProductList_BrandAccessOtherBrand_403(t *testing.T) {
	srv := setupProductMock(t)
	defer srv.Close()

	client := upstream.New(srv.URL, srv.URL)
	h := NewConsoleProductHandler(client)

	// Brand user with BrandID "b-999" tries to access "b-001"
	req := newProductTestRequest(http.MethodGet, "/console/brands/b-001/products", &auth.Claims{
		Sub:     "brand-admin-1",
		Role:    "Brand",
		OrgID:   "org-001",
		BrandID: "b-999",
	}, "b-001")
	rr := httptest.NewRecorder()
	h.ListProducts(rr, req)

	if rr.Code != http.StatusForbidden {
		t.Fatalf("expected 403, got %d; body: %s", rr.Code, rr.Body.String())
	}

	var body ErrorBody
	if err := json.Unmarshal(rr.Body.Bytes(), &body); err != nil {
		t.Fatalf("invalid JSON: %v", err)
	}
	if body.Error.Code != "FORBIDDEN" {
		t.Errorf("expected error code 'FORBIDDEN', got %q", body.Error.Code)
	}
}

// TestProductList_BrandAccessOwnBrand_200 verifies that a Brand-role user
// accessing their own brand gets 200 OK.
func TestProductList_BrandAccessOwnBrand_200(t *testing.T) {
	srv := setupProductMock(t)
	defer srv.Close()

	client := upstream.New(srv.URL, srv.URL)
	h := NewConsoleProductHandler(client)

	// Brand user with BrandID "b-001" accesses "b-001" — allowed
	req := newProductTestRequest(http.MethodGet, "/console/brands/b-001/products", &auth.Claims{
		Sub:     "brand-admin-1",
		Role:    "Brand",
		OrgID:   "org-001",
		BrandID: "b-001",
	}, "b-001")
	rr := httptest.NewRecorder()
	h.ListProducts(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d; body: %s", rr.Code, rr.Body.String())
	}

	var vm ProductListVM
	if err := json.Unmarshal(rr.Body.Bytes(), &vm); err != nil {
		t.Fatalf("invalid JSON: %v", err)
	}
	if vm.BrandName != "TestBrand" {
		t.Errorf("expected brand_name='TestBrand', got %q", vm.BrandName)
	}
	if len(vm.Items) != 2 {
		t.Errorf("expected 2 items, got %d", len(vm.Items))
	}
}

// TestProductList_BrandNotFound_404 verifies that when the upstream brand
// does not exist (404), the handler returns 404 NOT_FOUND.
func TestProductList_BrandNotFound_404(t *testing.T) {
	srv := setupProductMock(t)
	defer srv.Close()

	client := upstream.New(srv.URL, srv.URL)
	h := NewConsoleProductHandler(client)

	req := newProductTestRequest(http.MethodGet, "/console/brands/nonexistent/products", &auth.Claims{
		Sub:  "admin-1",
		Role: "Platform",
	}, "nonexistent")
	rr := httptest.NewRecorder()
	h.ListProducts(rr, req)

	if rr.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d; body: %s", rr.Code, rr.Body.String())
	}

	var body ErrorBody
	if err := json.Unmarshal(rr.Body.Bytes(), &body); err != nil {
		t.Fatalf("invalid JSON: %v", err)
	}
	if body.Error.Code != "NOT_FOUND" {
		t.Errorf("expected error code 'NOT_FOUND', got %q", body.Error.Code)
	}
}

// ---------------------------------------------------------------------------
// Property-Based Test: Brand 角色品牌边界校验 (Property 5)
// ---------------------------------------------------------------------------

// TestBrandBoundaryEnforcement uses rapid to verify that for any Brand-role
// user with brand_id=X, accessing /console/brands/{Y}/products where Y≠X
// always returns 403 FORBIDDEN.
//
// **Validates: Requirements FR-08 (8.4)**
func TestBrandBoundaryEnforcement(t *testing.T) {
	// Mock server: all product list requests succeed (the handler should
	// short-circuit before reaching upstream when brand boundary is violated)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"items": []interface{}{},
			"total": 0,
		})
	}))
	defer srv.Close()

	client := upstream.New(srv.URL, srv.URL)
	h := NewConsoleProductHandler(client)

	rapid.Check(t, func(t *rapid.T) {
		// Generate two distinct brand IDs
		claimsBrandID := rapid.StringMatching(`[a-z0-9\-]{3,20}`).Draw(t, "claims_brand_id")
		urlBrandID := rapid.StringMatching(`[a-z0-9\-]{3,20}`).Draw(t, "url_brand_id")

		// Ensure the two brand IDs are different (boundary violation)
		if claimsBrandID == urlBrandID {
			t.Skip("same brand IDs — not a boundary violation scenario")
		}

		req := newProductTestRequest(
			http.MethodGet,
			"/console/brands/"+urlBrandID+"/products",
			&auth.Claims{
				Sub:     rapid.StringMatching(`[a-z0-9\-]{3,20}`).Draw(t, "sub"),
				Role:    "Brand",
				OrgID:   rapid.StringMatching(`[a-z0-9\-]{3,20}`).Draw(t, "org_id"),
				BrandID: claimsBrandID,
			},
			urlBrandID,
		)
		rr := httptest.NewRecorder()
		h.ListProducts(rr, req)

		if rr.Code != http.StatusForbidden {
			t.Fatalf("expected 403 for Brand(brand_id=%q) accessing brands/%q/products, got %d",
				claimsBrandID, urlBrandID, rr.Code)
		}

		var body ErrorBody
		if err := json.Unmarshal(rr.Body.Bytes(), &body); err != nil {
			t.Fatalf("invalid JSON response: %v", err)
		}
		if body.Error.Code != "FORBIDDEN" {
			t.Fatalf("expected error code 'FORBIDDEN', got %q", body.Error.Code)
		}
	})
}
