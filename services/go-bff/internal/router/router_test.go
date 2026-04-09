package router_test

import (
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	"rcprotocol/services/go-bff/internal/router"
	"rcprotocol/services/go-bff/internal/upstream"
)

func newTestRouter() http.Handler {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelError}))
	client := upstream.New("http://localhost:19999", "http://localhost:19998")
	return router.New(logger, client)
}

func TestHealthz_Returns200OK_WithoutAuthorization(t *testing.T) {
	h := newTestRouter()
	req := httptest.NewRequest(http.MethodGet, "/healthz", nil)
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK || rr.Body.String() != "ok" {
		t.Fatalf("unexpected healthz response code=%d body=%q", rr.Code, rr.Body.String())
	}
}

func TestAppAssets_WithoutAuthorization_Returns401(t *testing.T) {
	h := newTestRouter()
	req := httptest.NewRequest(http.MethodGet, "/app/assets", nil)
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)
	if rr.Code != http.StatusUnauthorized {
		t.Fatalf("expected status 401, got %d", rr.Code)
	}
}

func TestConsoleDashboard_WithoutAuthorization_Returns401(t *testing.T) {
	h := newTestRouter()
	req := httptest.NewRequest(http.MethodGet, "/console/dashboard", nil)
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)
	if rr.Code != http.StatusUnauthorized {
		t.Fatalf("expected status 401, got %d", rr.Code)
	}
}

func TestConsoleBrands_WithoutAuthorization_Returns401(t *testing.T) {
	h := newTestRouter()
	req := httptest.NewRequest(http.MethodGet, "/console/brands/b-001", nil)
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)
	if rr.Code != http.StatusUnauthorized {
		t.Fatalf("expected status 401, got %d", rr.Code)
	}
}

func TestRouter_BrandDetail_ForwardsGatewayApiKeyHeadersToRcAPI(t *testing.T) {
	var seenHash, seenVerified, seenTrace bool
	rcAPI := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("X-Api-Key-Hash") == "hash-123" {
			seenHash = true
		}
		if r.Header.Get("X-Api-Key-Verified") == "hash-only" {
			seenVerified = true
		}
		if r.Header.Get("X-Trace-Id") == "trace-123" {
			seenTrace = true
		}
		switch r.URL.Path {
		case "/brands/b-001":
			_ = json.NewEncoder(w).Encode(map[string]string{"brand_id": "b-001", "brand_name": "Luxury Brand", "contact_email": "brand@test.com", "industry": "Watches", "status": "Active", "created_at": "2024-01-01T00:00:00Z", "updated_at": "2024-01-02T00:00:00Z"})
		case "/brands/b-001/api-keys":
			_ = json.NewEncoder(w).Encode(map[string]interface{}{"keys": []interface{}{}})
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer rcAPI.Close()
	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelError}))
	h := router.New(logger, upstream.New(rcAPI.URL, rcAPI.URL))
	req := httptest.NewRequest(http.MethodGet, "/console/brands/b-001", nil)
	req.Header.Set("X-Claims-Sub", "brand-user")
	req.Header.Set("X-Claims-Role", "Brand")
	req.Header.Set("X-Claims-Brand-Id", "b-001")
	req.Header.Set("X-Trace-Id", "trace-123")
	req.Header.Set("X-Api-Key-Hash", "hash-123")
	req.Header.Set("X-Api-Key-Verified", "hash-only")
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK || !(seenHash && seenVerified && seenTrace) {
		t.Fatalf("expected forwarded headers hash=%v verified=%v trace=%v body=%s", seenHash, seenVerified, seenTrace, rr.Body.String())
	}
}

func TestRouter_ProductList_RealAssemblyAggregatesRcAPI(t *testing.T) {
	var seenHash bool
	rcAPI := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("X-Api-Key-Hash") != "" {
			seenHash = true
		}
		switch r.URL.Path {
		case "/brands/b-001/products":
			_ = json.NewEncoder(w).Encode(map[string]interface{}{"items": []map[string]string{{"product_id": "p-001", "product_name": "Watch Alpha", "created_at": "2024-01-01T00:00:00Z"}}, "total": 1})
		case "/brands/b-001":
			_ = json.NewEncoder(w).Encode(map[string]string{"brand_name": "Luxury Brand"})
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer rcAPI.Close()
	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelError}))
	h := router.New(logger, upstream.New(rcAPI.URL, rcAPI.URL))
	req := httptest.NewRequest(http.MethodGet, "/console/brands/b-001/products", nil)
	req.Header.Set("X-Claims-Sub", "brand-user")
	req.Header.Set("X-Claims-Role", "Brand")
	req.Header.Set("X-Claims-Brand-Id", "b-001")
	req.Header.Set("X-Api-Key-Hash", "hash-123")
	req.Header.Set("X-Api-Key-Verified", "hash-only")
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)
	var body struct {
		BrandName string `json:"brand_name"`
		Items     []struct {
			ProductID string `json:"product_id"`
		} `json:"items"`
		Total int `json:"total"`
	}
	_ = json.Unmarshal(rr.Body.Bytes(), &body)
	if rr.Code != http.StatusOK || body.BrandName != "Luxury Brand" || body.Total != 1 || len(body.Items) != 1 || body.Items[0].ProductID != "p-001" || !seenHash {
		t.Fatalf("unexpected product response: code=%d body=%s seenHash=%v", rr.Code, rr.Body.String(), seenHash)
	}
}

func TestRouter_AssetsList_RealAssemblyAggregatesRcAPI(t *testing.T) {
	var seenHash bool
	rcAPI := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("X-Api-Key-Hash") != "" {
			seenHash = true
		}
		switch r.URL.Path {
		case "/assets":
			_ = json.NewEncoder(w).Encode(map[string]interface{}{"items": []map[string]string{{"asset_id": "a-001", "brand_id": "b-001", "product_id": "p-001", "current_state": "Activated"}}, "total": 1})
		case "/brands/batch":
			_ = json.NewEncoder(w).Encode([]map[string]string{{"brand_id": "b-001", "brand_name": "Luxury Brand"}})
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer rcAPI.Close()
	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelError}))
	h := router.New(logger, upstream.New(rcAPI.URL, rcAPI.URL))
	req := httptest.NewRequest(http.MethodGet, "/app/assets", nil)
	req.Header.Set("X-Claims-Sub", "user-1")
	req.Header.Set("X-Claims-Role", "Consumer")
	req.Header.Set("X-Api-Key-Hash", "hash-123")
	req.Header.Set("X-Api-Key-Verified", "hash-only")
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)
	var body struct {
		Items []struct {
			AssetID   string `json:"asset_id"`
			BrandName string `json:"brand_name"`
		} `json:"items"`
		Total int `json:"total"`
	}
	_ = json.Unmarshal(rr.Body.Bytes(), &body)
	if rr.Code != http.StatusOK || body.Total != 1 || len(body.Items) != 1 || body.Items[0].AssetID != "a-001" || body.Items[0].BrandName != "Luxury Brand" || !seenHash {
		t.Fatalf("unexpected assets response: code=%d body=%s seenHash=%v", rr.Code, rr.Body.String(), seenHash)
	}
}

func TestRouter_FactoryQuickLog_Post_RealAssemblyUsesDTOAndUpstream(t *testing.T) {
	var gotHash, gotVerified, gotContentType, gotBody string
	rcAPI := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotHash = r.Header.Get("X-Api-Key-Hash")
		gotVerified = r.Header.Get("X-Api-Key-Verified")
		gotContentType = r.Header.Get("Content-Type")
		b, _ := io.ReadAll(r.Body)
		gotBody = string(b)
		if r.Method != http.MethodPost || r.URL.Path != "/factory/quick-log" {
			w.WriteHeader(http.StatusNotFound)
			return
		}
		w.WriteHeader(http.StatusCreated)
		_, _ = w.Write([]byte(`{"ok":true,"log_id":"ql-001","event_type":"scan"}`))
	}))
	defer rcAPI.Close()

	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelError}))
	h := router.New(logger, upstream.New(rcAPI.URL, rcAPI.URL))
	req := httptest.NewRequest(http.MethodPost, "/console/factory/quick-log", strings.NewReader(`{"batch_id":"bat-001","event_type":"scan"}`))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Claims-Sub", "factory-user")
	req.Header.Set("X-Claims-Role", "Factory")
	req.Header.Set("X-Api-Key-Hash", "hash-123")
	req.Header.Set("X-Api-Key-Verified", "hash-only")
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)

	var body struct {
		OK        bool   `json:"ok"`
		LogID     string `json:"log_id"`
		EventType string `json:"event_type"`
	}
	_ = json.Unmarshal(rr.Body.Bytes(), &body)
	if rr.Code != http.StatusCreated || !body.OK || body.LogID != "ql-001" || body.EventType != "scan" || gotHash != "hash-123" || gotVerified != "hash-only" || gotContentType != "application/json" || gotBody != `{"batch_id":"bat-001","event_type":"scan"}` {
		t.Fatalf("unexpected factory post response code=%d body=%s upstreamBody=%q hash=%q verified=%q ct=%q", rr.Code, rr.Body.String(), gotBody, gotHash, gotVerified, gotContentType)
	}
}
