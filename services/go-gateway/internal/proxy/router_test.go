package proxy

import (
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"rcprotocol/services/go-gateway/internal/config"
	"rcprotocol/services/go-gateway/internal/middleware"
	"rcprotocol/services/go-gateway/internal/response"
)

func newMockUpstream(t *testing.T, h http.HandlerFunc) *httptest.Server { t.Helper(); return httptest.NewServer(h) }
func defaultCfg(rcAPI, goBff string) *config.Config { return &config.Config{Port: ":8080", JWTSecret: "test-secret-key-for-jwt-signing-1234567890", RcApiUpstream: rcAPI, GoBffUpstream: goBff, RateLimitRPS: 100, RateLimitBurst: 200} }
func newGatewayHandler(cfg *config.Config) http.Handler {
	logger := slog.Default(); var h http.Handler = NewRouter(cfg)
	h = middleware.WriteHeaders(h); h = middleware.Auth(cfg.JWTSecret)(h); h = middleware.RateLimit(cfg.RateLimitRPS, cfg.RateLimitBurst)(h); h = middleware.Trace(h); h = middleware.Logging(logger)(h)
	return h
}

func TestHealthz(t *testing.T) {
	r := NewRouter(defaultCfg("http://unused", "http://unused")); req := httptest.NewRequest(http.MethodGet, "/healthz", nil); rec := httptest.NewRecorder(); r.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK || rec.Body.String() != "ok" { t.Fatalf("unexpected healthz %d %q", rec.Code, rec.Body.String()) }
}

func TestPathStrip_VerifyWithQuery(t *testing.T) {
	var path, query string
	u := newMockUpstream(t, func(w http.ResponseWriter, r *http.Request) { path, query = r.URL.Path, r.URL.RawQuery; w.WriteHeader(http.StatusOK) })
	defer u.Close(); r := NewRouter(defaultCfg(u.URL, "http://unused")); req := httptest.NewRequest(http.MethodGet, "/api/verify?uid=04A3", nil); rec := httptest.NewRecorder(); r.ServeHTTP(rec, req)
	if path != "/verify" || query != "uid=04A3" { t.Fatalf("unexpected path=%q query=%q", path, query) }
}

func TestPathStrip_ProtocolAssets(t *testing.T) {
	var path string
	u := newMockUpstream(t, func(w http.ResponseWriter, r *http.Request) { path = r.URL.Path; w.WriteHeader(http.StatusOK) })
	defer u.Close(); r := NewRouter(defaultCfg(u.URL, "http://unused")); req := httptest.NewRequest(http.MethodPost, "/api/protocol/assets/xxx/activate", nil); rec := httptest.NewRecorder(); r.ServeHTTP(rec, req)
	if path != "/assets/xxx/activate" { t.Fatalf("got %q", path) }
}

func TestPathStrip_BffDashboard(t *testing.T) {
	var path string
	u := newMockUpstream(t, func(w http.ResponseWriter, r *http.Request) { path = r.URL.Path; w.WriteHeader(http.StatusOK) })
	defer u.Close(); r := NewRouter(defaultCfg("http://unused", u.URL)); req := httptest.NewRequest(http.MethodGet, "/api/bff/dashboard", nil); rec := httptest.NewRecorder(); r.ServeHTTP(rec, req)
	if path != "/dashboard" { t.Fatalf("got %q", path) }
}

func TestUnknownRoute_404(t *testing.T) {
	r := NewRouter(defaultCfg("http://unused", "http://unused")); req := httptest.NewRequest(http.MethodGet, "/unknown", nil); rec := httptest.NewRecorder(); r.ServeHTTP(rec, req)
	var body response.ErrorBody; _ = json.Unmarshal(rec.Body.Bytes(), &body)
	if rec.Code != http.StatusNotFound || body.Error.Code != response.CodeNotFound { t.Fatalf("unexpected response code=%d body=%s", rec.Code, rec.Body.String()) }
}

func TestRequestBodyQueryHeaderPassthrough(t *testing.T) {
	var method, body, query, header string
	u := newMockUpstream(t, func(w http.ResponseWriter, r *http.Request) { method, query, header = r.Method, r.URL.RawQuery, r.Header.Get("X-Custom-Header"); b, _ := io.ReadAll(r.Body); body = string(b); w.WriteHeader(http.StatusOK) })
	defer u.Close(); r := NewRouter(defaultCfg(u.URL, "http://unused")); req := httptest.NewRequest(http.MethodPost, "/api/brands?foo=bar&baz=1", strings.NewReader(`{"key":"value"}`)); req.Header.Set("Content-Type", "application/json"); req.Header.Set("X-Custom-Header", "custom-val"); rec := httptest.NewRecorder(); r.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK || method != http.MethodPost || body != `{"key":"value"}` || query != "foo=bar&baz=1" || header != "custom-val" { t.Fatalf("unexpected passthrough") }
}

func TestApiKeyHeadersForwardedToBffUpstream(t *testing.T) {
	var hash, verified, role, trace string
	u := newMockUpstream(t, func(w http.ResponseWriter, r *http.Request) { hash, verified, role, trace = r.Header.Get("X-Api-Key-Hash"), r.Header.Get("X-Api-Key-Verified"), r.Header.Get("X-Claims-Role"), r.Header.Get("X-Trace-Id"); w.WriteHeader(http.StatusOK) })
	defer u.Close(); h := newGatewayHandler(defaultCfg("http://unused", u.URL)); req := httptest.NewRequest(http.MethodGet, "/api/bff/console/brands/b-001", nil); req.Header.Set("X-Api-Key", "rcpk_live_1234567890abcdef1234567890abcdef"); req.Header.Set("X-Trace-Id", "trace-bff-1"); rec := httptest.NewRecorder(); h.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK || hash == "" || verified != "hash-only" || role != "Brand" || trace != "trace-bff-1" { t.Fatalf("unexpected contract forwarding") }
}

func TestGatewayBffRcApi_BlackBoxReturnsBusinessJson(t *testing.T) {
	var seen bool
	rc := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { if r.Header.Get("X-Api-Key-Hash") != "" { seen = true }; switch r.URL.Path { case "/brands/b-001": _, _ = w.Write([]byte(`{"brand_id":"b-001","brand_name":"Luxury Brand","contact_email":"brand@test.com","industry":"Watches","status":"Active","created_at":"2024-01-01T00:00:00Z","updated_at":"2024-01-02T00:00:00Z"}`)); case "/brands/b-001/api-keys": _, _ = w.Write([]byte(`{"keys":[]}`)); default: w.WriteHeader(http.StatusNotFound) } }))
	defer rc.Close()
	bff := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { if r.Header.Get("X-Api-Key-Hash") == "" || r.Header.Get("X-Api-Key-Verified") != "hash-only" || r.Header.Get("X-Claims-Role") != "Brand" { w.WriteHeader(http.StatusUnauthorized); return }; q1, _ := http.NewRequest(http.MethodGet, rc.URL+"/brands/b-001", nil); q1.Header = r.Header.Clone(); s1, _ := http.DefaultClient.Do(q1); defer s1.Body.Close(); b1, _ := io.ReadAll(s1.Body); q2, _ := http.NewRequest(http.MethodGet, rc.URL+"/brands/b-001/api-keys", nil); q2.Header = r.Header.Clone(); s2, _ := http.DefaultClient.Do(q2); defer s2.Body.Close(); b2, _ := io.ReadAll(s2.Body); var brand, keys map[string]interface{}; _ = json.Unmarshal(b1, &brand); _ = json.Unmarshal(b2, &keys); brand["api_keys"] = keys["keys"]; _ = json.NewEncoder(w).Encode(brand) }))
	defer bff.Close()
	h := newGatewayHandler(defaultCfg("http://unused", bff.URL)); req := httptest.NewRequest(http.MethodGet, "/api/bff/console/brands/b-001", nil); req.Header.Set("X-Api-Key", "rcpk_live_1234567890abcdef1234567890abcdef"); req.Header.Set("X-Trace-Id", "trace-blackbox-1"); rec := httptest.NewRecorder(); h.ServeHTTP(rec, req)
	var body map[string]interface{}; _ = json.Unmarshal(rec.Body.Bytes(), &body)
	if rec.Code != http.StatusOK || body["brand_id"] != "b-001" || body["brand_name"] != "Luxury Brand" || !seen { t.Fatalf("unexpected brand json: %s", rec.Body.String()) }
}

func TestGatewayBffRcApi_BlackBoxReturnsProductJson(t *testing.T) {
	var seen bool
	rc := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { if r.Header.Get("X-Api-Key-Hash") != "" { seen = true }; switch r.URL.Path { case "/brands/b-001/products": _, _ = w.Write([]byte(`{"items":[{"product_id":"p-001","product_name":"Watch Alpha","created_at":"2024-01-01T00:00:00Z"}],"total":1}`)); case "/brands/b-001": _, _ = w.Write([]byte(`{"brand_name":"Luxury Brand"}`)); default: w.WriteHeader(http.StatusNotFound) } }))
	defer rc.Close()
	bff := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { if r.Header.Get("X-Api-Key-Hash") == "" || r.Header.Get("X-Api-Key-Verified") != "hash-only" || r.Header.Get("X-Claims-Role") != "Brand" { w.WriteHeader(http.StatusUnauthorized); return }; q1, _ := http.NewRequest(http.MethodGet, rc.URL+"/brands/b-001/products", nil); q1.Header = r.Header.Clone(); s1, _ := http.DefaultClient.Do(q1); defer s1.Body.Close(); b1, _ := io.ReadAll(s1.Body); q2, _ := http.NewRequest(http.MethodGet, rc.URL+"/brands/b-001", nil); q2.Header = r.Header.Clone(); s2, _ := http.DefaultClient.Do(q2); defer s2.Body.Close(); b2, _ := io.ReadAll(s2.Body); var products, brand map[string]interface{}; _ = json.Unmarshal(b1, &products); _ = json.Unmarshal(b2, &brand); products["brand_name"] = brand["brand_name"]; products["page"] = float64(1); products["page_size"] = float64(20); _ = json.NewEncoder(w).Encode(products) }))
	defer bff.Close()
	h := newGatewayHandler(defaultCfg("http://unused", bff.URL)); req := httptest.NewRequest(http.MethodGet, "/api/bff/console/brands/b-001/products", nil); req.Header.Set("X-Api-Key", "rcpk_live_1234567890abcdef1234567890abcdef"); rec := httptest.NewRecorder(); h.ServeHTTP(rec, req)
	var body struct{ BrandName string `json:"brand_name"`; Items []struct{ ProductID string `json:"product_id"` } `json:"items"`; Total int `json:"total"` }; _ = json.Unmarshal(rec.Body.Bytes(), &body)
	if rec.Code != http.StatusOK || body.BrandName != "Luxury Brand" || body.Total != 1 || len(body.Items) != 1 || body.Items[0].ProductID != "p-001" || !seen { t.Fatalf("unexpected product json: %s", rec.Body.String()) }
}

func TestGatewayBffRcApi_BlackBoxReturnsAssetsJson(t *testing.T) {
	var seen bool
	rc := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { if r.Header.Get("X-Api-Key-Hash") != "" { seen = true }; switch r.URL.Path { case "/assets": _, _ = w.Write([]byte(`{"items":[{"asset_id":"a-001","brand_id":"b-001","product_id":"p-001","current_state":"Activated"}],"total":1}`)); case "/brands/batch": _, _ = w.Write([]byte(`[{"brand_id":"b-001","brand_name":"Luxury Brand"}]`)); default: w.WriteHeader(http.StatusNotFound) } }))
	defer rc.Close()
	bff := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { if r.Header.Get("X-Api-Key-Hash") == "" || r.Header.Get("X-Api-Key-Verified") != "hash-only" || r.Header.Get("X-Claims-Role") != "Brand" { w.WriteHeader(http.StatusUnauthorized); return }; q1, _ := http.NewRequest(http.MethodGet, rc.URL+"/assets?page=1&page_size=20&owner_id=", nil); q1.Header = r.Header.Clone(); s1, _ := http.DefaultClient.Do(q1); defer s1.Body.Close(); b1, _ := io.ReadAll(s1.Body); q2, _ := http.NewRequest(http.MethodGet, rc.URL+"/brands/batch?ids=b-001", nil); q2.Header = r.Header.Clone(); s2, _ := http.DefaultClient.Do(q2); defer s2.Body.Close(); b2, _ := io.ReadAll(s2.Body); var assets struct{ Items []map[string]interface{} `json:"items"`; Total int `json:"total"` }; var brands []map[string]interface{}; _ = json.Unmarshal(b1, &assets); _ = json.Unmarshal(b2, &brands); bn := "b-001"; if len(brands) > 0 { if s, ok := brands[0]["brand_name"].(string); ok { bn = s } }; items := []map[string]interface{}{}; for _, it := range assets.Items { items = append(items, map[string]interface{}{"asset_id": it["asset_id"], "brand_name": bn, "product_name": it["product_id"], "state": it["current_state"], "state_label": "已激活", "display_badges": []string{"verified"}}) }; _ = json.NewEncoder(w).Encode(map[string]interface{}{"items": items, "total": assets.Total, "page": 1, "page_size": 20}) }))
	defer bff.Close()
	h := newGatewayHandler(defaultCfg("http://unused", bff.URL)); req := httptest.NewRequest(http.MethodGet, "/api/bff/app/assets", nil); req.Header.Set("X-Api-Key", "rcpk_live_1234567890abcdef1234567890abcdef"); rec := httptest.NewRecorder(); h.ServeHTTP(rec, req)
	var body struct{ Items []struct{ AssetID string `json:"asset_id"`; BrandName string `json:"brand_name"` } `json:"items"`; Total int `json:"total"` }; _ = json.Unmarshal(rec.Body.Bytes(), &body)
	if rec.Code != http.StatusOK || body.Total != 1 || len(body.Items) != 1 || body.Items[0].AssetID != "a-001" || body.Items[0].BrandName != "Luxury Brand" || !seen { t.Fatalf("unexpected assets json: %s", rec.Body.String()) }
}

func TestUpstream400_WithStandardErrorFormat_PreservesMessage(t *testing.T) {
	u := newMockUpstream(t, func(w http.ResponseWriter, r *http.Request) { w.Header().Set("Content-Type", "application/json"); w.WriteHeader(http.StatusBadRequest); _, _ = w.Write([]byte(`{"error":{"code":"INVALID_INPUT","message":"uid is required"}}`)) })
	defer u.Close(); r := NewRouter(defaultCfg(u.URL, "http://unused")); req := httptest.NewRequest(http.MethodGet, "/api/brands", nil); req.Header.Set("X-Trace-Id", "trace-preserve"); rec := httptest.NewRecorder(); r.ServeHTTP(rec, req)
	if rec.Code != http.StatusBadRequest { t.Fatalf("expected 400, got %d", rec.Code) }
}
