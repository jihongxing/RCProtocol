package handler

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"rcprotocol/services/go-iam/internal/model"
	"rcprotocol/services/go-iam/internal/repo"
)

var validOrgTypes = map[string]bool{
	"platform": true,
	"brand":    true,
	"factory":  true,
}

// generateBrandID generates a brand_id in format: brand-{timestamp}-{random6}
func generateBrandID() string {
	timestamp := time.Now().Unix()
	randomBytes := make([]byte, 3) // 3 bytes = 6 hex chars
	_, _ = rand.Read(randomBytes)
	randomHex := hex.EncodeToString(randomBytes)
	return fmt.Sprintf("brand-%d-%s", timestamp, randomHex)
}

// OrgHandler handles HTTP requests for organization CRUD operations.
type OrgHandler struct {
	repo repo.OrgRepository
}

// NewOrgHandler creates an OrgHandler with the given repository.
func NewOrgHandler(r repo.OrgRepository) *OrgHandler {
	return &OrgHandler{repo: r}
}

type createOrgRequest struct {
	OrgName      string  `json:"org_name"`
	OrgType      string  `json:"org_type"`
	ParentOrgID  *string `json:"parent_org_id,omitempty"`
	BrandID      *string `json:"brand_id,omitempty"`
	ContactEmail *string `json:"contact_email,omitempty"`
	ContactPhone *string `json:"contact_phone,omitempty"`
}

// Create handles POST /orgs.
func (h *OrgHandler) Create(w http.ResponseWriter, r *http.Request) {
	var req createOrgRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		WriteError(w, http.StatusBadRequest, "INVALID_INPUT", "invalid request body")
		return
	}

	if !validOrgTypes[req.OrgType] {
		WriteError(w, http.StatusBadRequest, "INVALID_INPUT", "org_type must be one of: platform, brand, factory")
		return
	}

	// Phase 2: Brand simplified registration
	if req.OrgType == "brand" {
		// Auto-generate brand_id if not provided
		if req.BrandID == nil || *req.BrandID == "" {
			generatedBrandID := generateBrandID()
			req.BrandID = &generatedBrandID
		}

		// Validate contact fields are required for brands
		if req.ContactEmail == nil || *req.ContactEmail == "" {
			WriteError(w, http.StatusBadRequest, "INVALID_INPUT", "contact_email is required for brand organizations")
			return
		}
		if req.ContactPhone == nil || *req.ContactPhone == "" {
			WriteError(w, http.StatusBadRequest, "INVALID_INPUT", "contact_phone is required for brand organizations")
			return
		}
	}

	org := &model.Organization{
		OrgID:        uuid.New().String(),
		OrgName:      req.OrgName,
		OrgType:      req.OrgType,
		ParentOrgID:  req.ParentOrgID,
		BrandID:      req.BrandID,
		ContactEmail: req.ContactEmail,
		ContactPhone: req.ContactPhone,
		Status:       "active",
	}

	if err := h.repo.Create(r.Context(), org); err != nil {
		if isUniqueViolation(err) {
			WriteError(w, http.StatusConflict, "CONFLICT", "organization already exists")
			return
		}
		WriteError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "failed to create organization")
		return
	}

	WriteSuccess(w, http.StatusCreated, org)
}

// List handles GET /orgs with optional org_type filter.
func (h *OrgHandler) List(w http.ResponseWriter, r *http.Request) {
	page, pageSize := parsePagination(r)
	orgType := r.URL.Query().Get("org_type")

	orgs, total, err := h.repo.List(r.Context(), orgType, page, pageSize)
	if err != nil {
		WriteError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "failed to list organizations")
		return
	}

	WriteList(w, orgs, page, pageSize, total)
}

// GetByID handles GET /orgs/{id}.
func (h *OrgHandler) GetByID(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")

	org, err := h.repo.GetByID(r.Context(), id)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			WriteError(w, http.StatusNotFound, "NOT_FOUND", "organization not found")
			return
		}
		WriteError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "failed to get organization")
		return
	}

	WriteSuccess(w, http.StatusOK, org)
}

type updateOrgRequest struct {
	OrgName string `json:"org_name"`
	Status  string `json:"status"`
}

// Update handles PUT /orgs/{id}. org_type cannot be changed.
func (h *OrgHandler) Update(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")

	var req updateOrgRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		WriteError(w, http.StatusBadRequest, "INVALID_INPUT", "invalid request body")
		return
	}

	org, err := h.repo.Update(r.Context(), id, req.OrgName, req.Status)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			WriteError(w, http.StatusNotFound, "NOT_FOUND", "organization not found")
			return
		}
		WriteError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "failed to update organization")
		return
	}

	WriteSuccess(w, http.StatusOK, org)
}
