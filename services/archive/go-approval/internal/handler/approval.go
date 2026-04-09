package handler

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"rcprotocol/services/go-approval/internal/claims"
	"rcprotocol/services/go-approval/internal/downstream"
	"rcprotocol/services/go-approval/internal/model"
)

// ApprovalRepository 审批单仓储接口，handler 层依赖此接口便于测试 mock
type ApprovalRepository interface {
	Create(ctx context.Context, a *model.Approval) error
	GetByID(ctx context.Context, id string) (*model.Approval, error)
	ExistsPending(ctx context.Context, resourceType, resourceID, approvalType string) (bool, error)
	List(ctx context.Context, filterStatus, filterType, orgID string, page, pageSize int) ([]model.Approval, int, error)
	ListByResource(ctx context.Context, resourceType, resourceID string) ([]model.Approval, error)
	UpdateStatus(ctx context.Context, id, expectedStatus, newStatus string, reviewerID, reviewerRole, reviewComment *string, downstreamResult *[]byte) (bool, error)
}

// ApprovalHandler 审批 Handler
type ApprovalHandler struct {
	repo       ApprovalRepository
	downstream *downstream.Client
}

// NewApprovalHandler 创建 ApprovalHandler
func NewApprovalHandler(repo ApprovalRepository, ds *downstream.Client) *ApprovalHandler {
	return &ApprovalHandler{repo: repo, downstream: ds}
}

// Create POST /approvals
func (h *ApprovalHandler) Create(w http.ResponseWriter, r *http.Request) {
	c := claims.FromRequest(r)
	if !c.Valid() {
		WriteError(w, http.StatusUnauthorized, "AUTH_REQUIRED", "missing claims headers")
		return
	}

	var req struct {
		Type         string          `json:"type"`
		Payload      json.RawMessage `json:"payload"`
		ResourceType string          `json:"resource_type"`
		ResourceID   string          `json:"resource_id"`
		Reason       *string         `json:"reason"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		WriteError(w, http.StatusBadRequest, "INVALID_INPUT", "invalid request body")
		return
	}

	if !model.ValidTypes[req.Type] {
		WriteError(w, http.StatusBadRequest, "INVALID_INPUT", "invalid approval type")
		return
	}
	if req.ResourceType == "" || req.ResourceID == "" {
		WriteError(w, http.StatusBadRequest, "INVALID_INPUT", "resource_type and resource_id are required")
		return
	}
	if len(req.Payload) == 0 {
		WriteError(w, http.StatusBadRequest, "INVALID_INPUT", "payload is required")
		return
	}

	// M10 修复: 校验 payload.asset_id（如存在）与 resource_id 一致
	// 防止审批名义上批准 resource A 但实际执行 resource B
	var payloadMap map[string]interface{}
	if err := json.Unmarshal(req.Payload, &payloadMap); err == nil {
		if payloadAssetID, ok := payloadMap["asset_id"].(string); ok && payloadAssetID != "" {
			if payloadAssetID != req.ResourceID {
				WriteError(w, http.StatusBadRequest, "INVALID_INPUT",
					"payload.asset_id must match resource_id")
				return
			}
		}
	}

	exists, err := h.repo.ExistsPending(r.Context(), req.ResourceType, req.ResourceID, req.Type)
	if err != nil {
		WriteError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "check pending failed")
		return
	}
	if exists {
		WriteError(w, http.StatusConflict, "CONFLICT", "a pending approval already exists for this resource")
		return
	}

	approval := &model.Approval{
		ID:             uuid.New().String(),
		Type:           req.Type,
		Status:         model.StatusPending,
		ApplicantID:    c.Sub,
		ApplicantRole:  c.Role,
		ApplicantOrgID: c.OrgID,
		Payload:        req.Payload,
		Reason:         req.Reason,
		ResourceType:   req.ResourceType,
		ResourceID:     req.ResourceID,
		ExpiresAt:      time.Now().Add(72 * time.Hour),
	}

	if err := h.repo.Create(r.Context(), approval); err != nil {
		WriteError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "create approval failed")
		return
	}

	created, _ := h.repo.GetByID(r.Context(), approval.ID)
	if created != nil {
		WriteJSON(w, http.StatusCreated, created)
	} else {
		WriteJSON(w, http.StatusCreated, approval)
	}
}

// List GET /approvals
func (h *ApprovalHandler) List(w http.ResponseWriter, r *http.Request) {
	c := claims.FromRequest(r)
	if !c.Valid() {
		WriteError(w, http.StatusUnauthorized, "AUTH_REQUIRED", "missing claims headers")
		return
	}

	filterStatus := r.URL.Query().Get("status")
	filterType := r.URL.Query().Get("type")
	page, pageSize := ParsePagination(r)

	orgID := ""
	if c.Role == "Brand" {
		orgID = c.OrgID
	}

	items, total, err := h.repo.List(r.Context(), filterStatus, filterType, orgID, page, pageSize)
	if err != nil {
		WriteError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "list approvals failed")
		return
	}
	if items == nil {
		items = []model.Approval{}
	}

	WriteList(w, items, total, page, pageSize)
}

// GetByID GET /approvals/{approvalId}
func (h *ApprovalHandler) GetByID(w http.ResponseWriter, r *http.Request) {
	c := claims.FromRequest(r)
	if !c.Valid() {
		WriteError(w, http.StatusUnauthorized, "AUTH_REQUIRED", "missing claims headers")
		return
	}

	approvalID := chi.URLParam(r, "approvalId")
	approval, err := h.repo.GetByID(r.Context(), approvalID)
	if err != nil {
		WriteError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "query approval failed")
		return
	}
	if approval == nil {
		WriteError(w, http.StatusNotFound, "NOT_FOUND", "approval not found")
		return
	}

	if c.Role == "Brand" && approval.ApplicantOrgID != c.OrgID {
		WriteError(w, http.StatusForbidden, "FORBIDDEN", "access denied")
		return
	}

	WriteJSON(w, http.StatusOK, approval)
}

// ListByResource GET /approvals/by-resource
func (h *ApprovalHandler) ListByResource(w http.ResponseWriter, r *http.Request) {
	c := claims.FromRequest(r)
	if !c.Valid() {
		WriteError(w, http.StatusUnauthorized, "AUTH_REQUIRED", "missing claims headers")
		return
	}

	resourceType := r.URL.Query().Get("resource_type")
	resourceID := r.URL.Query().Get("resource_id")

	if resourceType == "" || resourceID == "" {
		WriteError(w, http.StatusBadRequest, "INVALID_INPUT", "resource_type and resource_id are required")
		return
	}

	items, err := h.repo.ListByResource(r.Context(), resourceType, resourceID)
	if err != nil {
		WriteError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "query approvals failed")
		return
	}
	if items == nil {
		items = []model.Approval{}
	}

	// Brand 角色只能查看自己品牌相关的审批
	if c.Role == "Brand" {
		filtered := make([]model.Approval, 0, len(items))
		for _, a := range items {
			if a.ApplicantOrgID == c.OrgID {
				filtered = append(filtered, a)
			}
		}
		items = filtered
	}

	WriteJSON(w, http.StatusOK, items)
}

// Approve POST /approvals/{approvalId}/approve
func (h *ApprovalHandler) Approve(w http.ResponseWriter, r *http.Request) {
	c := claims.FromRequest(r)
	if !c.Valid() {
		WriteError(w, http.StatusUnauthorized, "AUTH_REQUIRED", "missing claims headers")
		return
	}

	if !model.ReviewerRoles[c.Role] {
		WriteError(w, http.StatusForbidden, "FORBIDDEN", "only Platform or Moderator can approve")
		return
	}

	approvalID := chi.URLParam(r, "approvalId")
	approval, err := h.repo.GetByID(r.Context(), approvalID)
	if err != nil || approval == nil {
		WriteError(w, http.StatusNotFound, "NOT_FOUND", "approval not found")
		return
	}

	if approval.Status != model.StatusPending {
		WriteError(w, http.StatusConflict, "CONFLICT", "approval is not in pending status")
		return
	}

	if approval.IsExpired() {
		h.repo.UpdateStatus(r.Context(), approvalID, model.StatusPending, model.StatusExpired, nil, nil, nil, nil)
		WriteError(w, http.StatusConflict, "CONFLICT", "approval has expired")
		return
	}

	if approval.ApplicantID == c.Sub {
		WriteError(w, http.StatusForbidden, "FORBIDDEN", "applicant cannot approve their own request")
		return
	}

	var reqBody struct {
		ReviewComment *string `json:"review_comment"`
	}
	_ = json.NewDecoder(r.Body).Decode(&reqBody)

	ok, err := h.repo.UpdateStatus(r.Context(), approvalID, model.StatusPending, model.StatusApproved,
		&c.Sub, &c.Role, reqBody.ReviewComment, nil)
	if err != nil || !ok {
		WriteError(w, http.StatusConflict, "CONFLICT", "approval status changed concurrently")
		return
	}

	slog.Info("approval approved",
		slog.String("approval_id", approvalID),
		slog.String("from_status", model.StatusPending),
		slog.String("to_status", model.StatusApproved),
		slog.String("actor_id", c.Sub),
	)

	result := h.downstream.Execute(r.Context(), approval,
		r.Header.Get("Authorization"), r.Header.Get("X-Trace-Id"))

	var resultBytes []byte
	if result.Body != nil {
		resultBytes = []byte(result.Body)
	}

	if result.Success {
		h.repo.UpdateStatus(r.Context(), approvalID, model.StatusApproved, model.StatusExecuted,
			nil, nil, nil, &resultBytes)
		slog.Info("approval executed",
			slog.String("approval_id", approvalID),
			slog.String("from_status", model.StatusApproved),
			slog.String("to_status", model.StatusExecuted),
			slog.String("actor_id", c.Sub),
		)
		updated, _ := h.repo.GetByID(r.Context(), approvalID)
		if updated != nil {
			WriteJSON(w, http.StatusOK, updated)
		}
	} else {
		// M10 修复: 下游超时/异常时保持 approved 状态，不标记 failed
		// 记录下游结果但不改变审批状态，支持后续重试
		slog.Warn("approval downstream execution pending",
			slog.String("approval_id", approvalID),
			slog.String("status", model.StatusApproved),
			slog.String("actor_id", c.Sub),
			slog.String("downstream_error", string(result.Body)),
		)
		updated, _ := h.repo.GetByID(r.Context(), approvalID)
		if updated != nil {
			WriteJSON(w, http.StatusAccepted, map[string]interface{}{
				"approval":  updated,
				"execution": "pending_retry",
			})
		}
	}
}

// Reject POST /approvals/{approvalId}/reject
func (h *ApprovalHandler) Reject(w http.ResponseWriter, r *http.Request) {
	c := claims.FromRequest(r)
	if !c.Valid() {
		WriteError(w, http.StatusUnauthorized, "AUTH_REQUIRED", "missing claims headers")
		return
	}

	if !model.ReviewerRoles[c.Role] {
		WriteError(w, http.StatusForbidden, "FORBIDDEN", "only Platform or Moderator can reject")
		return
	}

	approvalID := chi.URLParam(r, "approvalId")
	approval, err := h.repo.GetByID(r.Context(), approvalID)
	if err != nil || approval == nil {
		WriteError(w, http.StatusNotFound, "NOT_FOUND", "approval not found")
		return
	}

	if approval.Status != model.StatusPending {
		WriteError(w, http.StatusConflict, "CONFLICT", "approval is not in pending status")
		return
	}

	var reqBody struct {
		ReviewComment string `json:"review_comment"`
	}
	if err := json.NewDecoder(r.Body).Decode(&reqBody); err != nil || reqBody.ReviewComment == "" {
		WriteError(w, http.StatusBadRequest, "INVALID_INPUT", "review_comment is required for rejection")
		return
	}

	ok, updateErr := h.repo.UpdateStatus(r.Context(), approvalID, model.StatusPending, model.StatusRejected,
		&c.Sub, &c.Role, &reqBody.ReviewComment, nil)
	if updateErr != nil || !ok {
		WriteError(w, http.StatusConflict, "CONFLICT", "approval status changed concurrently")
		return
	}

	slog.Info("approval rejected",
		slog.String("approval_id", approvalID),
		slog.String("from_status", model.StatusPending),
		slog.String("to_status", model.StatusRejected),
		slog.String("actor_id", c.Sub),
	)

	updated, _ := h.repo.GetByID(r.Context(), approvalID)
	if updated != nil {
		WriteJSON(w, http.StatusOK, updated)
	}
}
