package handler

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
	"rcprotocol/services/go-iam/internal/model"
	"rcprotocol/services/go-iam/internal/repo"
)

type ApiKeyHandler struct {
	apiKeyRepo *repo.ApiKeyRepo
	orgRepo    *repo.OrgRepo
}

func NewApiKeyHandler(apiKeyRepo *repo.ApiKeyRepo, orgRepo *repo.OrgRepo) *ApiKeyHandler {
	return &ApiKeyHandler{
		apiKeyRepo: apiKeyRepo,
		orgRepo:    orgRepo,
	}
}

// generateApiKey generates a legacy/backoffice org-scoped API key in format: brand_{org_id}_{random32}.
// This format is retained for go-iam administrative compatibility and is distinct from
// current Gateway-facing brand keys (rcpk_live_*).
func generateApiKey(orgID string) (string, error) {
	randomBytes := make([]byte, 16) // 16 bytes = 32 hex chars
	if _, err := rand.Read(randomBytes); err != nil {
		return "", err
	}
	randomHex := hex.EncodeToString(randomBytes)
	return fmt.Sprintf("brand_%s_%s", orgID, randomHex), nil
}

// extractOrgIDFromKey extracts org_id from the legacy/backoffice API key format: brand_{org_id}_{random32}.
func extractOrgIDFromKey(apiKey string) (string, error) {
	parts := strings.Split(apiKey, "_")
	if len(parts) != 3 || parts[0] != "brand" {
		return "", fmt.Errorf("invalid API key format")
	}
	return parts[1], nil
}

// Create creates a Platform-managed legacy/backoffice API key for a brand organization.
// It is retained for administrative compatibility and does not define the current Gateway
// runtime brand API key truth.
// POST /orgs/{org_id}/api-keys
func (h *ApiKeyHandler) Create(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	orgID := chi.URLParam(r, "org_id")

	// Verify organization exists and is a brand
	org, err := h.orgRepo.GetByID(ctx, orgID)
	if err != nil {
		WriteError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to query organization")
		return
	}
	if org == nil {
		WriteError(w, http.StatusNotFound, "ORG_NOT_FOUND", "Organization not found")
		return
	}
	if org.OrgType != "brand" {
		WriteError(w, http.StatusBadRequest, "INVALID_ORG_TYPE", "Only brand organizations can create API keys")
		return
	}

	// Parse request body
	var input struct {
		Description *string `json:"description"`
	}
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		WriteError(w, http.StatusBadRequest, "INVALID_JSON", "Invalid JSON body")
		return
	}

	// Generate API key plaintext
	apiKeyPlaintext, err := generateApiKey(orgID)
	if err != nil {
		WriteError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to generate API key")
		return
	}

	// Hash the API key
	keyHash, err := bcrypt.GenerateFromPassword([]byte(apiKeyPlaintext), bcrypt.DefaultCost)
	if err != nil {
		WriteError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to hash API key")
		return
	}

	// Create API key record
	keyID := uuid.New().String()
	apiKey := &model.BrandApiKey{
		KeyID:       keyID,
		OrgID:       orgID,
		KeyHash:     string(keyHash),
		Description: input.Description,
		Status:      "active",
		CreatedAt:   time.Now(),
	}

	if err := h.apiKeyRepo.Create(ctx, apiKey); err != nil {
		WriteError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to create API key")
		return
	}

	// Return plaintext API key (only shown once)
	response := model.ApiKeyCreateResponse{
		KeyID:     keyID,
		ApiKey:    apiKeyPlaintext,
		CreatedAt: apiKey.CreatedAt,
	}
	WriteSuccess(w, http.StatusCreated, response)
}

// List returns legacy/backoffice API keys for an organization (excluding plaintext).
// These keys are administrative records in go-iam, not the current Gateway runtime truth.
// GET /orgs/{org_id}/api-keys
func (h *ApiKeyHandler) List(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	orgID := chi.URLParam(r, "org_id")

	keys, err := h.apiKeyRepo.ListByOrg(ctx, orgID)
	if err != nil {
		WriteError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to list API keys")
		return
	}

	WriteSuccess(w, http.StatusOK, keys)
}

// Revoke soft-deletes a legacy/backoffice API key.
// DELETE /orgs/{org_id}/api-keys/{key_id}
func (h *ApiKeyHandler) Revoke(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	keyID := chi.URLParam(r, "key_id")

	if err := h.apiKeyRepo.Revoke(ctx, keyID); err != nil {
		WriteError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to revoke API key")
		return
	}

	WriteSuccess(w, http.StatusOK, map[string]string{"message": "API key revoked successfully"})
}
