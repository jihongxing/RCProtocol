package handler

import (
	"encoding/json"
	"io"
	"net/http"

	"github.com/go-chi/chi/v5"
	"rcprotocol/services/go-bff/internal/upstream"
	"rcprotocol/services/go-bff/internal/viewmodel"
)

type AssetSummaryVM struct {
	AssetID             string  `json:"asset_id"`
	State               string  `json:"state"`
	StateLabel          string  `json:"state_label"`
	BrandName           string  `json:"brand_name"`
	ProductName         *string `json:"product_name,omitempty"`
	ExternalProductName *string `json:"external_product_name,omitempty"`
}

type TransferInfoVM struct {
	TransferID   string         `json:"transfer_id"`
	AssetID      string         `json:"asset_id"`
	FromUserID   string         `json:"from_user_id"`
	ToUserID     string         `json:"to_user_id"`
	Status       string         `json:"status"`
	CreatedAt    string         `json:"created_at"`
	AssetSummary AssetSummaryVM `json:"asset_summary"`
}

type TransferHandler struct {
	client *upstream.UpstreamClient
}

func NewTransferHandler(client *upstream.UpstreamClient) *TransferHandler {
	return &TransferHandler{client: client}
}

func (h *TransferHandler) GetTransfer(w http.ResponseWriter, r *http.Request) {
	transferID := chi.URLParam(r, "transferId")
	headers := upstream.GatewayAuthHeadersFromRequest(r)

	transferData, err := h.client.RcApiGetWithGatewayAuth(r.Context(), "/transfers/"+transferID, headers)
	if err != nil {
		if ue, ok := err.(*upstream.UpstreamError); ok {
			WriteError(w, ue.StatusCode, ue.Code, ue.Message)
			return
		}
		WriteError(w, http.StatusBadGateway, "UPSTREAM_FAILURE", "backend service unavailable")
		return
	}

	var transfer struct {
		TransferID string `json:"transfer_id"`
		AssetID    string `json:"asset_id"`
		FromUserID string `json:"from_user_id"`
		ToUserID   string `json:"to_user_id"`
		Status     string `json:"status"`
		CreatedAt  string `json:"created_at"`
	}
	if json.Unmarshal(transferData, &transfer) != nil {
		WriteError(w, http.StatusBadGateway, "UPSTREAM_FAILURE", "invalid upstream response")
		return
	}

	assetData, err := h.client.RcApiGetWithGatewayAuth(r.Context(), "/assets/"+transfer.AssetID, headers)
	if err != nil {
		if ue, ok := err.(*upstream.UpstreamError); ok {
			WriteError(w, ue.StatusCode, ue.Code, ue.Message)
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
		ExternalProductName *string `json:"external_product_name"`
	}
	if json.Unmarshal(assetData, &asset) != nil {
		WriteError(w, http.StatusBadGateway, "UPSTREAM_FAILURE", "invalid upstream response")
		return
	}

	brandName := h.client.GetBrandNameWithGatewayAuth(r.Context(), asset.BrandID, headers)
	productName := asset.ProductID
	if asset.ExternalProductName != nil && *asset.ExternalProductName != "" {
		productName = *asset.ExternalProductName
	}

	WriteJSON(w, http.StatusOK, TransferInfoVM{
		TransferID: transfer.TransferID,
		AssetID:    transfer.AssetID,
		FromUserID: transfer.FromUserID,
		ToUserID:   transfer.ToUserID,
		Status:     transfer.Status,
		CreatedAt:  transfer.CreatedAt,
		AssetSummary: AssetSummaryVM{
			AssetID:             asset.AssetID,
			State:               asset.State,
			StateLabel:          viewmodel.MapState(asset.State),
			BrandName:           brandName,
			ProductName:         &productName,
			ExternalProductName: asset.ExternalProductName,
		},
	})
}

func (h *TransferHandler) ConfirmTransfer(w http.ResponseWriter, r *http.Request) {
	h.handleTransferAction(w, r, "/transfers/confirm")
}

func (h *TransferHandler) RejectTransfer(w http.ResponseWriter, r *http.Request) {
	h.handleTransferAction(w, r, "/transfers/reject")
}

func (h *TransferHandler) handleTransferAction(w http.ResponseWriter, r *http.Request, path string) {
	headers := upstream.GatewayAuthHeadersFromRequest(r)
	body, err := h.client.RcApiDoWithGatewayAuth(r.Context(), http.MethodPost, path, mustReadJSONBody(r), "application/json", headers)
	if err != nil {
		if ue, ok := err.(*upstream.UpstreamError); ok {
			WriteError(w, ue.StatusCode, ue.Code, ue.Message)
			return
		}
		WriteError(w, http.StatusBadGateway, "UPSTREAM_FAILURE", "backend service unavailable")
		return
	}

	var transfer struct {
		TransferID string `json:"transfer_id"`
		Status     string `json:"status"`
	}
	if json.Unmarshal(body, &transfer) != nil {
		WriteError(w, http.StatusBadGateway, "UPSTREAM_FAILURE", "invalid upstream response")
		return
	}

	WriteJSON(w, http.StatusOK, map[string]any{
		"success":     true,
		"transfer_id": transfer.TransferID,
		"status":      transfer.Status,
	})
}

func mustReadJSONBody(r *http.Request) []byte {
	defer r.Body.Close()
	body, _ := io.ReadAll(r.Body)
	return body
}
