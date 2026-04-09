package handler

import (
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"
	"rcprotocol/services/go-bff/internal/auth"
	"rcprotocol/services/go-bff/internal/upstream"
)

// BrandDetailViewModel is the B-端品牌详情聚合 ViewModel.
// 品牌真相源来自 rc-api，而不是 go-iam 组织信息。
type BrandDetailViewModel struct {
	BrandID      string       `json:"brand_id"`
	BrandName    string       `json:"brand_name"`
	ContactEmail string       `json:"contact_email"`
	Industry     string       `json:"industry"`
	Status       string       `json:"status"`
	CreatedAt    string       `json:"created_at"`
	UpdatedAt    string       `json:"updated_at"`
	ApiKeys      []ApiKeyItem `json:"api_keys"`
}

// ApiKeyItem mirrors rc-api /brands/{brandId}/api-keys response.
type ApiKeyItem struct {
	KeyID      string  `json:"key_id"`
	KeyPrefix  string  `json:"key_prefix"`
	Status     string  `json:"status"`
	CreatedAt  string  `json:"created_at"`
	LastUsedAt *string `json:"last_used_at"`
	RevokedAt  *string `json:"revoked_at"`
}

// BrandHandler handles B-端 brand detail requests.
type BrandHandler struct {
	upstreamClient *upstream.UpstreamClient
}

func NewBrandHandler(upstreamClient *upstream.UpstreamClient) *BrandHandler {
	return &BrandHandler{upstreamClient: upstreamClient}
}

// GetBrandDetail handles GET /console/brands/{brandId}.
func (h *BrandHandler) GetBrandDetail(w http.ResponseWriter, r *http.Request) {
	brandID := chi.URLParam(r, "brandId")
	claims := auth.ClaimsFromContext(r.Context())
	headers := upstream.GatewayAuthHeadersFromRequest(r)

	if claims.Role == "Brand" && claims.BrandID != brandID {
		WriteError(w, http.StatusForbidden, "FORBIDDEN", "access denied to this brand")
		return
	}

	brandData, brandErr := h.upstreamClient.RcApiGetWithGatewayAuth(r.Context(), "/brands/"+brandID, headers)
	if brandErr != nil {
		if ue, ok := brandErr.(*upstream.UpstreamError); ok {
			WriteError(w, ue.StatusCode, ue.Code, ue.Message)
			return
		}
		WriteError(w, http.StatusBadGateway, "UPSTREAM_FAILURE", "backend service unavailable")
		return
	}

	var brand struct {
		BrandID      string `json:"brand_id"`
		BrandName    string `json:"brand_name"`
		ContactEmail string `json:"contact_email"`
		Industry     string `json:"industry"`
		Status       string `json:"status"`
		CreatedAt    string `json:"created_at"`
		UpdatedAt    string `json:"updated_at"`
	}
	if json.Unmarshal(brandData, &brand) != nil {
		WriteError(w, http.StatusBadGateway, "UPSTREAM_FAILURE", "invalid upstream response")
		return
	}

	apiKeys := []ApiKeyItem{}
	keyData, keyErr := h.upstreamClient.RcApiGetWithGatewayAuth(r.Context(), "/brands/"+brandID+"/api-keys", headers)
	if keyErr == nil {
		var result struct {
			Keys []ApiKeyItem `json:"keys"`
		}
		if json.Unmarshal(keyData, &result) == nil && result.Keys != nil {
			apiKeys = result.Keys
		}
	}

	WriteJSON(w, http.StatusOK, BrandDetailViewModel{
		BrandID:      brand.BrandID,
		BrandName:    brand.BrandName,
		ContactEmail: brand.ContactEmail,
		Industry:     brand.Industry,
		Status:       brand.Status,
		CreatedAt:    brand.CreatedAt,
		UpdatedAt:    brand.UpdatedAt,
		ApiKeys:      apiKeys,
	})
}
