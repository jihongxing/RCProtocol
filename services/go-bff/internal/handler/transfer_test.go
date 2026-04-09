package handler

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/go-chi/chi/v5"
	"rcprotocol/services/go-bff/internal/auth"
	"rcprotocol/services/go-bff/internal/upstream"
)

func TestTransferAsset_ForwardsBodyAndReturnsResponse(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || r.URL.Path != "/assets/a-transfer/transfer" {
			w.WriteHeader(http.StatusNotFound)
			return
		}
		body := mustReadBody(r)
		if string(body) != `{"new_owner_id":"user-002"}` {
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"transfer_id":"tr-001","asset_id":"a-transfer"}`))
	}))
	defer srv.Close()

	client := upstream.New(srv.URL, srv.URL)
	h := NewAppAssetHandler(client)
	req := httptest.NewRequest(http.MethodPost, "/app/assets/a-transfer/transfer", strings.NewReader(`{"new_owner_id":"user-002"}`))
	req = req.WithContext(auth.NewContext(req.Context(), &auth.Claims{Sub: "user-001", Role: "Consumer"}))
	req.Header.Set("Content-Type", "application/json")
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("assetId", "a-transfer")
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))
	rr := httptest.NewRecorder()

	h.TransferAsset(rr, req)

	if rr.Code != http.StatusOK || !strings.Contains(rr.Body.String(), `"transfer_id":"tr-001"`) {
		t.Fatalf("unexpected transfer response code=%d body=%s", rr.Code, rr.Body.String())
	}
}

func TestTransferHandler_GetTransferAndReject(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/transfers/tr-001":
			_ = json.NewEncoder(w).Encode(map[string]any{
				"transfer_id":  "tr-001",
				"asset_id":     "a-001",
				"from_user_id": "user-001",
				"to_user_id":   "user-002",
				"status":       "pending",
				"created_at":   "2024-01-15T10:30:00Z",
			})
		case "/assets/a-001":
			_ = json.NewEncoder(w).Encode(map[string]any{
				"asset_id":              "a-001",
				"brand_id":              "b-001",
				"product_id":            "p-001",
				"current_state":         "Activated",
				"external_product_name": "Watch Alpha",
			})
		case "/brands/b-001":
			_ = json.NewEncoder(w).Encode(map[string]any{"brand_name": "Luxury Brand"})
		case "/transfers/reject":
			_ = json.NewEncoder(w).Encode(map[string]any{"transfer_id": "tr-001", "status": "rejected"})
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer srv.Close()

	client := upstream.New(srv.URL, srv.URL)
	h := NewTransferHandler(client)

	getReq := httptest.NewRequest(http.MethodGet, "/app/transfers/tr-001", nil)
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("transferId", "tr-001")
	getReq = getReq.WithContext(context.WithValue(getReq.Context(), chi.RouteCtxKey, rctx))
	getRR := httptest.NewRecorder()
	h.GetTransfer(getRR, getReq)
	if getRR.Code != http.StatusOK || !strings.Contains(getRR.Body.String(), `"status":"pending"`) {
		t.Fatalf("unexpected get transfer response code=%d body=%s", getRR.Code, getRR.Body.String())
	}

	rejectReq := httptest.NewRequest(http.MethodPost, "/app/transfers/reject", strings.NewReader(`{"transfer_id":"tr-001"}`))
	rejectReq.Header.Set("Content-Type", "application/json")
	rejectRR := httptest.NewRecorder()
	h.RejectTransfer(rejectRR, rejectReq)
	if rejectRR.Code != http.StatusOK || !strings.Contains(rejectRR.Body.String(), `"status":"rejected"`) {
		t.Fatalf("unexpected reject transfer response code=%d body=%s", rejectRR.Code, rejectRR.Body.String())
	}
}
