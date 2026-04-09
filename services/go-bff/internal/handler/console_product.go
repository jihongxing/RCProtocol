package handler

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
	"rcprotocol/services/go-bff/internal/auth"
	"rcprotocol/services/go-bff/internal/upstream"
)

// ProductListVM is the ViewModel for the brand external product mapping page.
type ProductListVM struct {
	BrandName string      `json:"brand_name"`
	Items     []ProductVM `json:"items"`
	Total     int         `json:"total"`
	Page      int         `json:"page"`
	PageSize  int         `json:"page_size"`
}

// ProductVM represents a single brand-facing product item.
type ProductVM struct {
	ProductID           string  `json:"product_id"`
	ProductName         string  `json:"product_name"`
	CreatedAt           string  `json:"created_at"`
	ExternalProductID   *string `json:"external_product_id,omitempty"`
	ExternalProductName *string `json:"external_product_name,omitempty"`
	ExternalProductURL  *string `json:"external_product_url,omitempty"`
}

type ConsoleProductHandler struct {
	client *upstream.UpstreamClient
}

func NewConsoleProductHandler(client *upstream.UpstreamClient) *ConsoleProductHandler {
	return &ConsoleProductHandler{client: client}
}

func normalizeProductItem(item ProductVM) ProductVM {
	if item.ExternalProductName != nil && *item.ExternalProductName != "" {
		item.ProductName = *item.ExternalProductName
	}
	if item.ProductName == "" {
		item.ProductName = item.ProductID
	}
	return item
}

func (h *ConsoleProductHandler) ListProducts(w http.ResponseWriter, r *http.Request) {
	brandID := chi.URLParam(r, "brandId")
	claims := auth.ClaimsFromContext(r.Context())
	headers := upstream.GatewayAuthHeadersFromRequest(r)

	if claims.Role == "Brand" && claims.BrandID != brandID {
		WriteError(w, http.StatusForbidden, "FORBIDDEN", "access denied to this brand")
		return
	}

	page, pageSize := ParsePagination(r)
	path := "/brands/" + brandID + "/products?page=" + strconv.Itoa(page) + "&page_size=" + strconv.Itoa(pageSize)
	data, err := h.client.RcApiGetWithGatewayAuth(r.Context(), path, headers)
	if err != nil {
		if ue, ok := err.(*upstream.UpstreamError); ok && ue.StatusCode == 404 {
			WriteError(w, http.StatusNotFound, "NOT_FOUND", "brand not found")
			return
		}
		WriteError(w, http.StatusBadGateway, "UPSTREAM_FAILURE", "backend service unavailable")
		return
	}

	var upstreamResp struct {
		Items []ProductVM `json:"items"`
		Total int         `json:"total"`
	}
	if json.Unmarshal(data, &upstreamResp) != nil {
		WriteError(w, http.StatusBadGateway, "UPSTREAM_FAILURE", "invalid upstream response")
		return
	}

	for i := range upstreamResp.Items {
		upstreamResp.Items[i] = normalizeProductItem(upstreamResp.Items[i])
	}

	brandName := h.client.GetBrandNameWithGatewayAuth(r.Context(), brandID, headers)

	WriteJSON(w, http.StatusOK, ProductListVM{
		BrandName: brandName,
		Items:     upstreamResp.Items,
		Total:     upstreamResp.Total,
		Page:      page,
		PageSize:  pageSize,
	})
}
