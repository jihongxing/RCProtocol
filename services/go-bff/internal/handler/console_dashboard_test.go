package handler

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"

	"rcprotocol/services/go-bff/internal/auth"
	"rcprotocol/services/go-bff/internal/upstream"
)

func setupDashboardMock(t *testing.T, counts map[string]int, failPaths map[string]bool) *httptest.Server {
	t.Helper()
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		path := r.URL.Path
		if failPaths[path] {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		if count, ok := counts[path]; ok {
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(map[string]int{"count": count})
			return
		}
		w.WriteHeader(http.StatusNotFound)
	}))
}

func TestDashboard_AllSucceed(t *testing.T) {
	counts := map[string]int{
		"/brands/count":               10,
		"/assets/count":               100,
		"/stats/recent-verifications": 42,
	}
	srv := setupDashboardMock(t, counts, nil)
	defer srv.Close()

	client := upstream.New(srv.URL, srv.URL)
	h := NewDashboardHandler(client)
	req := newTestRequest(http.MethodGet, "/console/dashboard", &auth.Claims{Sub: "admin-1", Role: "Platform"})
	rr := httptest.NewRecorder()
	h.GetDashboard(rr, req)

	var vm DashboardVM
	_ = json.Unmarshal(rr.Body.Bytes(), &vm)
	if vm.BrandCount != 10 || vm.AssetCount != 100 || vm.RecentVerifications != 42 {
		t.Fatalf("unexpected dashboard body: %+v", vm)
	}
	if vm.ProductCount != 0 || vm.PendingTasks != 0 {
		t.Fatalf("expected conservative zeros for derived stats, got %+v", vm)
	}
}

func TestDashboard_PartialFailures(t *testing.T) {
	counts := map[string]int{"/brands/count": 5}
	failPaths := map[string]bool{
		"/assets/count":               true,
		"/stats/recent-verifications": true,
	}
	srv := setupDashboardMock(t, counts, failPaths)
	defer srv.Close()

	client := upstream.New(srv.URL, srv.URL)
	h := NewDashboardHandler(client)
	req := newTestRequest(http.MethodGet, "/console/dashboard", &auth.Claims{Sub: "admin-1", Role: "Platform"})
	rr := httptest.NewRecorder()
	h.GetDashboard(rr, req)

	var vm DashboardVM
	_ = json.Unmarshal(rr.Body.Bytes(), &vm)
	if vm.BrandCount != 5 || vm.AssetCount != 0 || vm.RecentVerifications != 0 {
		t.Fatalf("unexpected partial-failure dashboard: %+v", vm)
	}
	if vm.ProductCount != 0 || vm.PendingTasks != 0 {
		t.Fatalf("expected zeros for derived stats, got %+v", vm)
	}
}

func TestDashboard_BrandRoleScopesBrandID(t *testing.T) {
	var mu sync.Mutex
	receivedPaths := make([]string, 0, 3)

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		receivedPaths = append(receivedPaths, r.URL.String())
		mu.Unlock()
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]int{"count": 1})
	}))
	defer srv.Close()

	client := upstream.New(srv.URL, srv.URL)
	h := NewDashboardHandler(client)
	req := newTestRequest(http.MethodGet, "/console/dashboard", &auth.Claims{Sub: "brand-admin-1", Role: "Brand", BrandID: "b-777"})
	rr := httptest.NewRecorder()
	h.GetDashboard(rr, req)

	mu.Lock()
	paths := append([]string(nil), receivedPaths...)
	mu.Unlock()
	if len(paths) != 3 {
		t.Fatalf("expected 3 upstream calls, got %d", len(paths))
	}
	for _, p := range paths {
		if !strings.Contains(p, "brand_id=b-777") {
			t.Fatalf("expected brand_id scope in %q", p)
		}
	}
}

func TestDashboard_PlatformRoleNoBrandID(t *testing.T) {
	var mu sync.Mutex
	receivedPaths := make([]string, 0, 3)

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		receivedPaths = append(receivedPaths, r.URL.String())
		mu.Unlock()
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]int{"count": 1})
	}))
	defer srv.Close()

	client := upstream.New(srv.URL, srv.URL)
	h := NewDashboardHandler(client)
	req := newTestRequest(http.MethodGet, "/console/dashboard", &auth.Claims{Sub: "platform-admin-1", Role: "Platform"})
	rr := httptest.NewRecorder()
	h.GetDashboard(rr, req)

	mu.Lock()
	paths := append([]string(nil), receivedPaths...)
	mu.Unlock()
	if len(paths) != 3 {
		t.Fatalf("expected 3 upstream calls, got %d", len(paths))
	}
	for _, p := range paths {
		if strings.Contains(p, "brand_id") {
			t.Fatalf("unexpected brand_id scope in %q", p)
		}
	}
}

func TestDashboard_AllFail(t *testing.T) {
	client := upstream.New("http://127.0.0.1:1", "http://127.0.0.1:1")
	h := NewDashboardHandler(client)
	req := newTestRequest(http.MethodGet, "/console/dashboard", &auth.Claims{Sub: "admin-1", Role: "Platform"})
	rr := httptest.NewRecorder()
	h.GetDashboard(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rr.Code)
	}

	var vm DashboardVM
	_ = json.Unmarshal(rr.Body.Bytes(), &vm)
	if vm != (DashboardVM{}) {
		t.Fatalf("expected zero dashboard on full failure, got %+v", vm)
	}
}
