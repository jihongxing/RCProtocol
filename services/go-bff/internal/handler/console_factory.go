package handler

import (
	"encoding/json"
	"net/http"

	"rcprotocol/services/go-bff/internal/upstream"
)

// FactoryTaskHandler handles factory task list and future write actions.
type FactoryTaskHandler struct {
	client *upstream.UpstreamClient
}

// FactoryQuickLogCreateRequest is the minimal write DTO for future factory quick-log creation.
type FactoryQuickLogCreateRequest struct {
	BatchID   string `json:"batch_id"`
	EventType string `json:"event_type"`
}

// FactoryQuickLogCreateResponse is the minimal success DTO returned to the frontend.
type FactoryQuickLogCreateResponse struct {
	OK        bool   `json:"ok"`
	LogID     string `json:"log_id"`
	EventType string `json:"event_type"`
}

// NewFactoryTaskHandler creates a new FactoryTaskHandler.
func NewFactoryTaskHandler(client *upstream.UpstreamClient) *FactoryTaskHandler {
	return &FactoryTaskHandler{client: client}
}

// ListTasks GET /console/factory/tasks
// Phase 1 预留占位，返回空列表。
func (h *FactoryTaskHandler) ListTasks(w http.ResponseWriter, r *http.Request) {
	page, pageSize := ParsePagination(r)
	WriteList(w, []interface{}{}, 0, page, pageSize)
}

// CreateQuickLog POST /console/factory/quick-log
// 作为未来写链路示范位：统一走 gateway-auth-aware POST helper。
func (h *FactoryTaskHandler) CreateQuickLog(w http.ResponseWriter, r *http.Request) {
	if h.client == nil {
		WriteError(w, http.StatusServiceUnavailable, "UPSTREAM_FAILURE", "backend service unavailable")
		return
	}

	var reqDTO FactoryQuickLogCreateRequest
	if err := json.NewDecoder(r.Body).Decode(&reqDTO); err != nil {
		WriteError(w, http.StatusBadRequest, "INVALID_INPUT", "invalid request body")
		return
	}
	if reqDTO.BatchID == "" {
		WriteError(w, http.StatusBadRequest, "INVALID_INPUT", "batch_id is required")
		return
	}
	if reqDTO.EventType == "" {
		WriteError(w, http.StatusBadRequest, "INVALID_INPUT", "event_type is required")
		return
	}

	payload, err := json.Marshal(reqDTO)
	if err != nil {
		WriteError(w, http.StatusBadRequest, "INVALID_INPUT", "invalid request body")
		return
	}

	headers := upstream.GatewayAuthHeadersFromRequest(r)
	data, err := h.client.RcApiDoWithGatewayAuth(r.Context(), http.MethodPost, "/factory/quick-log", payload, "application/json", headers)
	if err != nil {
		if ue, ok := err.(*upstream.UpstreamError); ok {
			WriteError(w, ue.StatusCode, ue.Code, ue.Message)
			return
		}
		WriteError(w, http.StatusBadGateway, "UPSTREAM_FAILURE", "backend service unavailable")
		return
	}

	var respDTO FactoryQuickLogCreateResponse
	if err := json.Unmarshal(data, &respDTO); err != nil {
		WriteError(w, http.StatusBadGateway, "UPSTREAM_FAILURE", "invalid upstream response")
		return
	}

	WriteJSON(w, http.StatusCreated, respDTO)
}
