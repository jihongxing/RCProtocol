package handler

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"rcprotocol/services/go-iam/internal/model"
	"rcprotocol/services/go-iam/internal/repo"
)

// orgTypeRoleCompat defines which protocol roles are allowed for each org type.
var orgTypeRoleCompat = map[string]map[string]bool{
	"platform": {"Platform": true, "Moderator": true},
	"brand":    {"Brand": true},
	"factory":  {"Factory": true},
}

// isRoleCompatible checks whether a protocol role is valid for the given org type.
func isRoleCompatible(orgType, protocolRole string) bool {
	roles, ok := orgTypeRoleCompat[orgType]
	return ok && roles[protocolRole]
}

// PositionHandler handles HTTP requests for position management.
type PositionHandler struct {
	posRepo repo.PositionRepository
	orgRepo repo.OrgRepository
}

// NewPositionHandler creates a PositionHandler with the given repositories.
func NewPositionHandler(posRepo repo.PositionRepository, orgRepo repo.OrgRepository) *PositionHandler {
	return &PositionHandler{posRepo: posRepo, orgRepo: orgRepo}
}

type createPositionRequest struct {
	PositionName string `json:"position_name"`
	ProtocolRole string `json:"protocol_role"`
}

// Create handles POST /orgs/{org_id}/positions.
func (h *PositionHandler) Create(w http.ResponseWriter, r *http.Request) {
	orgID := chi.URLParam(r, "org_id")

	var req createPositionRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		WriteError(w, http.StatusBadRequest, "INVALID_INPUT", "invalid request body")
		return
	}

	org, err := h.orgRepo.GetByID(r.Context(), orgID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			WriteError(w, http.StatusNotFound, "NOT_FOUND", "organization not found")
			return
		}
		WriteError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "failed to get organization")
		return
	}

	if !isRoleCompatible(org.OrgType, req.ProtocolRole) {
		WriteError(w, http.StatusBadRequest, "INVALID_INPUT", "protocol_role is not compatible with organization type")
		return
	}

	pos := &model.Position{
		PositionID:   uuid.New().String(),
		OrgID:        orgID,
		PositionName: req.PositionName,
		ProtocolRole: req.ProtocolRole,
	}

	if err := h.posRepo.Create(r.Context(), pos); err != nil {
		WriteError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "failed to create position")
		return
	}

	WriteSuccess(w, http.StatusCreated, pos)
}

// List handles GET /orgs/{org_id}/positions.
func (h *PositionHandler) List(w http.ResponseWriter, r *http.Request) {
	orgID := chi.URLParam(r, "org_id")

	positions, err := h.posRepo.ListByOrg(r.Context(), orgID)
	if err != nil {
		WriteError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "failed to list positions")
		return
	}

	WriteSuccess(w, http.StatusOK, positions)
}
