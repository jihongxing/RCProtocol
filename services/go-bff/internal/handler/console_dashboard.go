package handler

import (
	"encoding/json"
	"net/http"
	"sync"

	"rcprotocol/services/go-bff/internal/auth"
	"rcprotocol/services/go-bff/internal/upstream"
)

// DashboardVM holds the 5 aggregate statistics for the B-端 console dashboard.
type DashboardVM struct {
	BrandCount          int `json:"brand_count"`
	ProductCount        int `json:"product_count"`
	AssetCount          int `json:"asset_count"`
	RecentVerifications int `json:"recent_verifications"`
	PendingTasks        int `json:"pending_tasks"`
}

type DashboardHandler struct {
	client *upstream.UpstreamClient
}

func NewDashboardHandler(client *upstream.UpstreamClient) *DashboardHandler {
	return &DashboardHandler{client: client}
}

func (h *DashboardHandler) GetDashboard(w http.ResponseWriter, r *http.Request) {
	claims := auth.ClaimsFromContext(r.Context())
	headers := upstream.GatewayAuthHeadersFromRequest(r)

	vm := DashboardVM{}
	var mu sync.Mutex
	var wg sync.WaitGroup

	brandScope := ""
	if claims.Role == "Brand" {
		brandScope = "?brand_id=" + claims.BrandID
	}

	type statFetcher struct {
		path   string
		target *int
	}
	fetchers := []statFetcher{
		{"/brands/count" + brandScope, &vm.BrandCount},
		{"/assets/count" + brandScope, &vm.AssetCount},
		{"/stats/recent-verifications" + brandScope, &vm.RecentVerifications},
	}

	wg.Add(len(fetchers))
	for _, f := range fetchers {
		go func(path string, target *int) {
			defer wg.Done()
			data, err := h.client.RcApiGetWithGatewayAuth(r.Context(), path, headers)
			if err != nil {
				return
			}
			var result struct { Count int `json:"count"` }
			if json.Unmarshal(data, &result) == nil {
				mu.Lock()
				*target = result.Count
				mu.Unlock()
			}
		}(f.path, f.target)
	}
	wg.Wait()

	vm.ProductCount = 0
	vm.PendingTasks = 0
	WriteJSON(w, http.StatusOK, vm)
}
