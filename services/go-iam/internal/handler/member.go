package handler

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5"

	"rcprotocol/services/go-iam/internal/model"
	"rcprotocol/services/go-iam/internal/repo"
)

// MemberHandler handles HTTP requests for member binding operations.
type MemberHandler struct {
	memberRepo repo.MemberRepository
	userRepo   repo.UserRepository
	posRepo    repo.PositionRepository
}

// NewMemberHandler creates a MemberHandler with the given repositories.
func NewMemberHandler(memberRepo repo.MemberRepository, userRepo repo.UserRepository, posRepo repo.PositionRepository) *MemberHandler {
	return &MemberHandler{memberRepo: memberRepo, userRepo: userRepo, posRepo: posRepo}
}

type bindRequest struct {
	UserID     string `json:"user_id"`
	PositionID string `json:"position_id"`
}

// Bind handles POST /orgs/{org_id}/members.
func (h *MemberHandler) Bind(w http.ResponseWriter, r *http.Request) {
	orgID := chi.URLParam(r, "org_id")

	var req bindRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		WriteError(w, http.StatusBadRequest, "INVALID_INPUT", "invalid request body")
		return
	}

	// Check user exists
	_, err := h.userRepo.GetByID(r.Context(), req.UserID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			WriteError(w, http.StatusNotFound, "NOT_FOUND", "user not found")
			return
		}
		WriteError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "failed to get user")
		return
	}

	// Check position exists and belongs to this org
	pos, err := h.posRepo.GetByID(r.Context(), req.PositionID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			WriteError(w, http.StatusBadRequest, "INVALID_INPUT", "position does not belong to this organization")
			return
		}
		WriteError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "failed to get position")
		return
	}
	if pos.OrgID != orgID {
		WriteError(w, http.StatusBadRequest, "INVALID_INPUT", "position does not belong to this organization")
		return
	}

	// Check duplicate binding
	exists, err := h.memberRepo.ExistsByUserAndOrg(r.Context(), req.UserID, orgID)
	if err != nil {
		WriteError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "failed to check membership")
		return
	}
	if exists {
		WriteError(w, http.StatusConflict, "CONFLICT", "user already has a position in this organization")
		return
	}

	uop := &model.UserOrgPosition{
		UserID:     req.UserID,
		OrgID:      orgID,
		PositionID: req.PositionID,
	}

	if err := h.memberRepo.Bind(r.Context(), uop); err != nil {
		WriteError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "failed to bind member")
		return
	}

	WriteSuccess(w, http.StatusCreated, uop)
}

// List handles GET /orgs/{org_id}/members.
func (h *MemberHandler) List(w http.ResponseWriter, r *http.Request) {
	orgID := chi.URLParam(r, "org_id")

	members, err := h.memberRepo.ListByOrg(r.Context(), orgID)
	if err != nil {
		WriteError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "failed to list members")
		return
	}

	WriteSuccess(w, http.StatusOK, members)
}

// Unbind handles DELETE /orgs/{org_id}/members/{user_id}.
func (h *MemberHandler) Unbind(w http.ResponseWriter, r *http.Request) {
	orgID := chi.URLParam(r, "org_id")
	userID := chi.URLParam(r, "user_id")

	if err := h.memberRepo.Unbind(r.Context(), userID, orgID); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			WriteError(w, http.StatusNotFound, "NOT_FOUND", "member binding not found")
			return
		}
		WriteError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "failed to unbind member")
		return
	}

	WriteSuccess(w, http.StatusOK, map[string]string{"status": "unbound"})
}
