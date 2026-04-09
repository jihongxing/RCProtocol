package handler

import (
	"encoding/json"
	"io"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
	"rcprotocol/services/go-bff/internal/auth"
	"rcprotocol/services/go-bff/internal/upstream"
	"rcprotocol/services/go-bff/internal/viewmodel"
)

// AssetVM is the C-端资产列表 ViewModel, directly consumable by the frontend.
type AssetVM struct {
	AssetID             string   `json:"asset_id"`
	BrandName           string   `json:"brand_name"`
	ProductName         string   `json:"product_name"`
	State               string   `json:"state"`
	StateLabel          string   `json:"state_label"`
	DisplayBadges       []string `json:"display_badges"`
	ThumbnailURL        string   `json:"thumbnail_url,omitempty"`
	ExternalProductID   *string  `json:"external_product_id"`
	ExternalProductName *string  `json:"external_product_name"`
	ExternalProductURL  *string  `json:"external_product_url"`
}

type VirtualMotherCardVM struct {
	AuthorityUID   string  `json:"authority_uid"`
	AuthorityType  string  `json:"authority_type"`
	CredentialHash *string `json:"credential_hash"`
	Epoch          int     `json:"epoch"`
}

// AssetDetailVM is the C-端资产详情 ViewModel.
type AssetDetailVM struct {
	AssetID             string               `json:"asset_id"`
	BrandName           string               `json:"brand_name"`
	ProductName         string               `json:"product_name"`
	State               string               `json:"state"`
	StateLabel          string               `json:"state_label"`
	DisplayBadges       []string             `json:"display_badges"`
	UID                 string               `json:"uid"`
	CreatedAt           string               `json:"created_at"`
	ExternalProductID   *string              `json:"external_product_id"`
	ExternalProductName *string              `json:"external_product_name"`
	ExternalProductURL  *string              `json:"external_product_url"`
	VirtualMotherCard   *VirtualMotherCardVM `json:"virtual_mother_card,omitempty"`
}

type AppAssetHandler struct {
	client *upstream.UpstreamClient
}

func NewAppAssetHandler(client *upstream.UpstreamClient) *AppAssetHandler {
	return &AppAssetHandler{client: client}
}

func resolveProductDisplayName(productID string, externalProductName *string) string {
	if externalProductName != nil && *externalProductName != "" {
		return *externalProductName
	}
	if productID != "" {
		return productID
	}
	return ""
}

func (h *AppAssetHandler) ListAssets(w http.ResponseWriter, r *http.Request) {
	claims := auth.ClaimsFromContext(r.Context())
	headers := upstream.GatewayAuthHeadersFromRequest(r)
	page, pageSize := ParsePagination(r)

	path := "/assets?page=" + strconv.Itoa(page) + "&page_size=" + strconv.Itoa(pageSize) + "&owner_id=" + claims.Sub
	data, err := h.client.RcApiGetWithGatewayAuth(r.Context(), path, headers)
	if err != nil {
		WriteList(w, []AssetVM{}, 0, page, pageSize)
		return
	}

	var upstreamResp struct {
		Items []struct {
			AssetID             string  `json:"asset_id"`
			BrandID             string  `json:"brand_id"`
			ProductID           string  `json:"product_id"`
			State               string  `json:"current_state"`
			ThumbnailURL        string  `json:"thumbnail_url"`
			ExternalProductID   *string `json:"external_product_id"`
			ExternalProductName *string `json:"external_product_name"`
			ExternalProductURL  *string `json:"external_product_url"`
		} `json:"items"`
		Total int `json:"total"`
	}
	if json.Unmarshal(data, &upstreamResp) != nil {
		WriteList(w, []AssetVM{}, 0, page, pageSize)
		return
	}

	items := make([]AssetVM, 0, len(upstreamResp.Items))
	brandIDSet := make(map[string]struct{})
	for _, a := range upstreamResp.Items {
		brandIDSet[a.BrandID] = struct{}{}
	}
	brandIDs := make([]string, 0, len(brandIDSet))
	for id := range brandIDSet {
		brandIDs = append(brandIDs, id)
	}

	brandNames := h.client.GetBrandNamesBatchWithGatewayAuth(r.Context(), brandIDs, headers)

	for _, a := range upstreamResp.Items {
		brandName := brandNames[a.BrandID]
		if brandName == "" {
			brandName = a.BrandID
		}
		items = append(items, AssetVM{
			AssetID:             a.AssetID,
			BrandName:           brandName,
			ProductName:         resolveProductDisplayName(a.ProductID, a.ExternalProductName),
			State:               a.State,
			StateLabel:          viewmodel.MapState(a.State),
			DisplayBadges:       viewmodel.MapBadges(a.State),
			ThumbnailURL:        a.ThumbnailURL,
			ExternalProductID:   a.ExternalProductID,
			ExternalProductName: a.ExternalProductName,
			ExternalProductURL:  a.ExternalProductURL,
		})
	}

	WriteList(w, items, upstreamResp.Total, page, pageSize)
}

func (h *AppAssetHandler) GetAsset(w http.ResponseWriter, r *http.Request) {
	assetID := chi.URLParam(r, "assetId")
	headers := upstream.GatewayAuthHeadersFromRequest(r)

	data, err := h.client.RcApiGetWithGatewayAuth(r.Context(), "/assets/"+assetID, headers)
	if err != nil {
		if ue, ok := err.(*upstream.UpstreamError); ok && ue.StatusCode == 404 {
			WriteError(w, http.StatusNotFound, "NOT_FOUND", "asset not found")
			return
		}
		WriteError(w, http.StatusBadGateway, "UPSTREAM_FAILURE", "backend service unavailable")
		return
	}

	var asset struct {
		AssetID             string  `json:"asset_id"`
		BrandID             string  `json:"brand_id"`
		ProductID           string  `json:"product_id"`
		State               string  `json:"current_state"`
		UID                 string  `json:"uid"`
		CreatedAt           string  `json:"created_at"`
		ExternalProductID   *string `json:"external_product_id"`
		ExternalProductName *string `json:"external_product_name"`
		ExternalProductURL  *string `json:"external_product_url"`
		VirtualMotherCard   *struct {
			AuthorityUID   string  `json:"authority_uid"`
			AuthorityType  string  `json:"authority_type"`
			CredentialHash *string `json:"credential_hash"`
			Epoch          int     `json:"epoch"`
		} `json:"virtual_mother_card"`
	}
	if json.Unmarshal(data, &asset) != nil {
		WriteError(w, http.StatusBadGateway, "UPSTREAM_FAILURE", "invalid upstream response")
		return
	}

	brandName := h.client.GetBrandNameWithGatewayAuth(r.Context(), asset.BrandID, headers)
	productName := resolveProductDisplayName(asset.ProductID, asset.ExternalProductName)

	var virtualMotherCard *VirtualMotherCardVM
	if asset.VirtualMotherCard != nil {
		virtualMotherCard = &VirtualMotherCardVM{
			AuthorityUID:   asset.VirtualMotherCard.AuthorityUID,
			AuthorityType:  asset.VirtualMotherCard.AuthorityType,
			CredentialHash: asset.VirtualMotherCard.CredentialHash,
			Epoch:          asset.VirtualMotherCard.Epoch,
		}
	}

	WriteJSON(w, http.StatusOK, AssetDetailVM{
		AssetID:             asset.AssetID,
		BrandName:           brandName,
		ProductName:         productName,
		State:               asset.State,
		StateLabel:          viewmodel.MapState(asset.State),
		DisplayBadges:       viewmodel.MapBadges(asset.State),
		UID:                 asset.UID,
		CreatedAt:           asset.CreatedAt,
		ExternalProductID:   asset.ExternalProductID,
		ExternalProductName: asset.ExternalProductName,
		ExternalProductURL:  asset.ExternalProductURL,
		VirtualMotherCard:   virtualMotherCard,
	})
}

func (h *AppAssetHandler) TransferAsset(w http.ResponseWriter, r *http.Request) {
	assetID := chi.URLParam(r, "assetId")
	headers := upstream.GatewayAuthHeadersFromRequest(r)
	body, err := h.client.RcApiDoWithGatewayAuth(r.Context(), http.MethodPost, "/assets/"+assetID+"/transfer", mustReadBody(r), "application/json", headers)
	if err != nil {
		if ue, ok := err.(*upstream.UpstreamError); ok {
			WriteError(w, ue.StatusCode, ue.Code, ue.Message)
			return
		}
		WriteError(w, http.StatusBadGateway, "UPSTREAM_FAILURE", "backend service unavailable")
		return
	}

	var response map[string]any
	if json.Unmarshal(body, &response) != nil {
		WriteError(w, http.StatusBadGateway, "UPSTREAM_FAILURE", "invalid upstream response")
		return
	}

	WriteJSON(w, http.StatusOK, response)
}

func mustReadBody(r *http.Request) []byte {
	defer r.Body.Close()
	body, _ := io.ReadAll(r.Body)
	return body
}
