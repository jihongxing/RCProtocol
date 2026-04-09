package handler

import (
	"context"
	"encoding/json"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"rcprotocol/services/go-workorder/internal/claims"
	"rcprotocol/services/go-workorder/internal/downstream"
	"rcprotocol/services/go-workorder/internal/model"
)

// WorkorderRepository 工单仓储接口，handler 层依赖此接口便于测试 mock
type WorkorderRepository interface {
	Create(ctx context.Context, w *model.Workorder) error
	GetByID(ctx context.Context, id string) (*model.Workorder, error)
	List(ctx context.Context, filterStatus, filterType, assigneeID, orgID, brandID string, page, pageSize int) ([]model.Workorder, int, error)
	ListByAsset(ctx context.Context, assetID string) ([]model.Workorder, error)
	UpdateStatus(ctx context.Context, id, expectedStatus, newStatus string) (bool, error)
	Assign(ctx context.Context, id string, expectedStatuses []string, assigneeID, assigneeRole string) (bool, error)
	Advance(ctx context.Context, id string, newStatus, conclusion, conclusionType string, approvalID *string, downstreamResult *[]byte) (bool, error)
}

// WorkorderHandler 工单 Handler
type WorkorderHandler struct {
	repo        WorkorderRepository
	rcApiClient *downstream.RcApiClient
}

// NewWorkorderHandler 创建 WorkorderHandler
func NewWorkorderHandler(repo WorkorderRepository, rcApi *downstream.RcApiClient) *WorkorderHandler {
	return &WorkorderHandler{repo: repo, rcApiClient: rcApi}
}

// Create POST /workorders
func (h *WorkorderHandler) Create(w http.ResponseWriter, r *http.Request) {
	c := claims.FromRequest(r)
	if !c.Valid() {
		WriteError(w, http.StatusUnauthorized, "AUTH_REQUIRED", "missing claims headers")
		return
	}

	var req struct {
		Type        string           `json:"type"`
		Title       string           `json:"title"`
		Description *string          `json:"description"`
		AssetID     *string          `json:"asset_id"`
		BrandID     *string          `json:"brand_id"`
		Metadata    *json.RawMessage `json:"metadata"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		WriteError(w, http.StatusBadRequest, "INVALID_INPUT", "invalid request body")
		return
	}

	if !model.ValidTypes[req.Type] {
		WriteError(w, http.StatusBadRequest, "INVALID_INPUT", "invalid workorder type")
		return
	}
	if req.Title == "" {
		WriteError(w, http.StatusBadRequest, "INVALID_INPUT", "title is required")
		return
	}
	if req.Type == model.TypeRecovery && (req.AssetID == nil || *req.AssetID == "") {
		WriteError(w, http.StatusBadRequest, "INVALID_INPUT", "asset_id is required for recovery workorders")
		return
	}

	now := time.Now()
	wo := &model.Workorder{
		ID:           uuid.New().String(),
		Type:         req.Type,
		Status:       model.StatusOpen,
		Title:        req.Title,
		Description:  req.Description,
		CreatorID:    c.Sub,
		CreatorRole:  c.Role,
		CreatorOrgID: c.OrgID,
		AssetID:      req.AssetID,
		BrandID:      req.BrandID,
		Metadata:     req.Metadata,
		CreatedAt:    now,
		UpdatedAt:    now,
	}

	if err := h.repo.Create(r.Context(), wo); err != nil {
		WriteError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "create workorder failed")
		return
	}

	WriteJSON(w, http.StatusCreated, wo)
}

// List GET /workorders
func (h *WorkorderHandler) List(w http.ResponseWriter, r *http.Request) {
	c := claims.FromRequest(r)
	if !c.Valid() {
		WriteError(w, http.StatusUnauthorized, "AUTH_REQUIRED", "missing claims headers")
		return
	}

	filterStatus := r.URL.Query().Get("status")
	filterType := r.URL.Query().Get("type")
	assigneeID := r.URL.Query().Get("assignee_id")
	page, pageSize := ParsePagination(r)

	orgID := ""
	brandID := ""
	if c.Role == "Brand" {
		orgID = c.OrgID
		brandID = c.OrgID
	}

	items, total, err := h.repo.List(r.Context(), filterStatus, filterType, assigneeID, orgID, brandID, page, pageSize)
	if err != nil {
		WriteError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "list workorders failed")
		return
	}
	if items == nil {
		items = []model.Workorder{}
	}

	WriteList(w, items, total, page, pageSize)
}

// GetByID GET /workorders/{workorderId}
func (h *WorkorderHandler) GetByID(w http.ResponseWriter, r *http.Request) {
	c := claims.FromRequest(r)
	if !c.Valid() {
		WriteError(w, http.StatusUnauthorized, "AUTH_REQUIRED", "missing claims headers")
		return
	}

	workorderID := chi.URLParam(r, "workorderId")
	wo, err := h.repo.GetByID(r.Context(), workorderID)
	if err != nil {
		WriteError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "query workorder failed")
		return
	}
	if wo == nil {
		WriteError(w, http.StatusNotFound, "NOT_FOUND", "workorder not found")
		return
	}

	if c.Role == "Brand" && wo.CreatorOrgID != c.OrgID && (wo.BrandID == nil || *wo.BrandID != c.OrgID) {
		WriteError(w, http.StatusForbidden, "FORBIDDEN", "access denied")
		return
	}

	WriteJSON(w, http.StatusOK, wo)
}

// ListByAsset GET /workorders/by-asset
func (h *WorkorderHandler) ListByAsset(w http.ResponseWriter, r *http.Request) {
	c := claims.FromRequest(r)
	if !c.Valid() {
		WriteError(w, http.StatusUnauthorized, "AUTH_REQUIRED", "missing claims headers")
		return
	}

	assetID := r.URL.Query().Get("asset_id")
	if assetID == "" {
		WriteError(w, http.StatusBadRequest, "INVALID_INPUT", "asset_id is required")
		return
	}

	items, err := h.repo.ListByAsset(r.Context(), assetID)
	if err != nil {
		WriteError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "query workorders failed")
		return
	}
	if items == nil {
		items = []model.Workorder{}
	}

	// Brand 角色只能查看自己品牌相关的工单
	if c.Role == "Brand" {
		filtered := make([]model.Workorder, 0, len(items))
		for _, wo := range items {
			if wo.BrandID != nil && *wo.BrandID == c.OrgID {
				filtered = append(filtered, wo)
			} else if wo.CreatorOrgID == c.OrgID {
				filtered = append(filtered, wo)
			}
		}
		items = filtered
	}

	WriteJSON(w, http.StatusOK, items)
}

// Assign POST /workorders/{workorderId}/assign
func (h *WorkorderHandler) Assign(w http.ResponseWriter, r *http.Request) {
	c := claims.FromRequest(r)
	if !c.Valid() {
		WriteError(w, http.StatusUnauthorized, "AUTH_REQUIRED", "missing claims headers")
		return
	}
	if !model.ManagerRoles[c.Role] {
		WriteError(w, http.StatusForbidden, "FORBIDDEN", "only Platform/Moderator can assign")
		return
	}

	workorderID := chi.URLParam(r, "workorderId")

	var req struct {
		AssigneeID   string `json:"assignee_id"`
		AssigneeRole string `json:"assignee_role"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		WriteError(w, http.StatusBadRequest, "INVALID_INPUT", "invalid request body")
		return
	}
	if req.AssigneeID == "" || req.AssigneeRole == "" {
		WriteError(w, http.StatusBadRequest, "INVALID_INPUT", "assignee_id and assignee_role are required")
		return
	}

	expectedStatuses := []string{model.StatusOpen, model.StatusAssigned}
	ok, err := h.repo.Assign(r.Context(), workorderID, expectedStatuses, req.AssigneeID, req.AssigneeRole)
	if err != nil {
		WriteError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "assign workorder failed")
		return
	}
	if !ok {
		WriteError(w, http.StatusConflict, "STATUS_CONFLICT", "workorder not found or status does not allow assign")
		return
	}

	wo, err := h.repo.GetByID(r.Context(), workorderID)
	if err != nil || wo == nil {
		WriteError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "fetch workorder after assign failed")
		return
	}

	WriteJSON(w, http.StatusOK, wo)
}

// Advance POST /workorders/{workorderId}/advance
func (h *WorkorderHandler) Advance(w http.ResponseWriter, r *http.Request) {
	c := claims.FromRequest(r)
	if !c.Valid() {
		WriteError(w, http.StatusUnauthorized, "AUTH_REQUIRED", "missing claims headers")
		return
	}

	workorderID := chi.URLParam(r, "workorderId")

	// 先查工单确认存在及状态
	wo, err := h.repo.GetByID(r.Context(), workorderID)
	if err != nil {
		WriteError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "query workorder failed")
		return
	}
	if wo == nil {
		WriteError(w, http.StatusNotFound, "NOT_FOUND", "workorder not found")
		return
	}

	// 校验角色：assignee 或 Platform
	isAssignee := wo.AssigneeID != nil && *wo.AssigneeID == c.Sub
	isPlatform := c.Role == "Platform"
	if !isAssignee && !isPlatform {
		WriteError(w, http.StatusForbidden, "FORBIDDEN", "only assignee or Platform can advance")
		return
	}

	// 校验状态可推进
	if !model.AdvancableStatuses[wo.Status] {
		WriteError(w, http.StatusConflict, "STATUS_CONFLICT", "workorder status does not allow advance")
		return
	}

	var req struct {
		Conclusion     string `json:"conclusion"`
		ConclusionType string `json:"conclusion_type"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		WriteError(w, http.StatusBadRequest, "INVALID_INPUT", "invalid request body")
		return
	}
	if req.Conclusion == "" || req.ConclusionType == "" {
		WriteError(w, http.StatusBadRequest, "INVALID_INPUT", "conclusion and conclusion_type are required")
		return
	}
	if !model.ValidConclusionTypes[req.ConclusionType] {
		WriteError(w, http.StatusBadRequest, "INVALID_INPUT", "invalid conclusion_type")
		return
	}

	authHeader := r.Header.Get("Authorization")
	traceID := r.Header.Get("X-Trace-Id")
	newStatus := model.StatusResolved
	var approvalID *string
	var downstreamResult *[]byte

	switch req.ConclusionType {
	case model.ConclusionFreeze:
		if wo.AssetID == nil || *wo.AssetID == "" {
			WriteError(w, http.StatusBadRequest, "INVALID_INPUT", "asset_id is required for freeze")
			return
		}
		result := h.rcApiClient.Freeze(r.Context(), *wo.AssetID, authHeader, traceID)
		b, _ := result.Body.MarshalJSON()
		downstreamResult = &b
		if !result.Success {
			newStatus = model.StatusInProgress
		}

	case model.ConclusionRecover:
		if wo.AssetID == nil || *wo.AssetID == "" {
			WriteError(w, http.StatusBadRequest, "INVALID_INPUT", "asset_id is required for recover")
			return
		}
		// Phase 2: 直接调用 rc-api recover，不需要审批流
		result := h.rcApiClient.Recover(r.Context(), *wo.AssetID, authHeader, traceID, "")
		b, _ := result.Body.MarshalJSON()
		downstreamResult = &b
		if !result.Success {
			newStatus = model.StatusInProgress
		}

	case model.ConclusionMarkTampered:
		// M11 修复: 工单推进到 tampered 时同步调用 rc-api，确保协议状态一致
		if wo.AssetID == nil || *wo.AssetID == "" {
			WriteError(w, http.StatusBadRequest, "INVALID_INPUT", "asset_id is required for mark_tampered")
			return
		}
		result := h.rcApiClient.MarkTampered(r.Context(), *wo.AssetID, authHeader, traceID)
		b, _ := result.Body.MarshalJSON()
		downstreamResult = &b
		if !result.Success {
			// 协议同步失败：标记为 awaiting_protocol_execution 而非 resolved
			newStatus = model.StatusInProgress
		}

	case model.ConclusionMarkCompromised:
		// M11 修复: 工单推进到 compromised 时同步调用 rc-api
		if wo.AssetID == nil || *wo.AssetID == "" {
			WriteError(w, http.StatusBadRequest, "INVALID_INPUT", "asset_id is required for mark_compromised")
			return
		}
		result := h.rcApiClient.MarkCompromised(r.Context(), *wo.AssetID, authHeader, traceID)
		b, _ := result.Body.MarshalJSON()
		downstreamResult = &b
		if !result.Success {
			newStatus = model.StatusInProgress
		}

	case model.ConclusionDismiss:
		newStatus = model.StatusResolved
	}

	ok, err := h.repo.Advance(r.Context(), workorderID, newStatus, req.Conclusion, req.ConclusionType, approvalID, downstreamResult)
	if err != nil {
		WriteError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "advance workorder failed")
		return
	}
	if !ok {
		WriteError(w, http.StatusConflict, "STATUS_CONFLICT", "advance failed, status may have changed")
		return
	}

	updated, _ := h.repo.GetByID(r.Context(), workorderID)
	WriteJSON(w, http.StatusOK, updated)
}

// Close POST /workorders/{workorderId}/close
func (h *WorkorderHandler) Close(w http.ResponseWriter, r *http.Request) {
	c := claims.FromRequest(r)
	if !c.Valid() {
		WriteError(w, http.StatusUnauthorized, "AUTH_REQUIRED", "missing claims headers")
		return
	}
	if !model.ManagerRoles[c.Role] {
		WriteError(w, http.StatusForbidden, "FORBIDDEN", "only Platform/Moderator can close")
		return
	}

	workorderID := chi.URLParam(r, "workorderId")
	ok, err := h.repo.UpdateStatus(r.Context(), workorderID, model.StatusResolved, model.StatusClosed)
	if err != nil {
		WriteError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "close workorder failed")
		return
	}
	if !ok {
		WriteError(w, http.StatusConflict, "STATUS_CONFLICT", "workorder not found or not in resolved status")
		return
	}

	wo, _ := h.repo.GetByID(r.Context(), workorderID)
	WriteJSON(w, http.StatusOK, wo)
}

// Cancel POST /workorders/{workorderId}/cancel
func (h *WorkorderHandler) Cancel(w http.ResponseWriter, r *http.Request) {
	c := claims.FromRequest(r)
	if !c.Valid() {
		WriteError(w, http.StatusUnauthorized, "AUTH_REQUIRED", "missing claims headers")
		return
	}
	if !model.ManagerRoles[c.Role] {
		WriteError(w, http.StatusForbidden, "FORBIDDEN", "only Platform/Moderator can cancel")
		return
	}

	workorderID := chi.URLParam(r, "workorderId")

	// 尝试 open → cancelled
	ok, err := h.repo.UpdateStatus(r.Context(), workorderID, model.StatusOpen, model.StatusCancelled)
	if err != nil {
		WriteError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "cancel workorder failed")
		return
	}
	if !ok {
		// 尝试 assigned → cancelled
		ok, err = h.repo.UpdateStatus(r.Context(), workorderID, model.StatusAssigned, model.StatusCancelled)
		if err != nil {
			WriteError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "cancel workorder failed")
			return
		}
		if !ok {
			WriteError(w, http.StatusConflict, "STATUS_CONFLICT", "workorder not found or status does not allow cancel")
			return
		}
	}

	wo, _ := h.repo.GetByID(r.Context(), workorderID)
	WriteJSON(w, http.StatusOK, wo)
}
