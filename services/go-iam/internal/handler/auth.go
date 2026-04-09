package handler

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/jackc/pgx/v5"
	"golang.org/x/crypto/bcrypt"

	"rcprotocol/services/go-iam/internal/auth"
	"rcprotocol/services/go-iam/internal/model"
	"rcprotocol/services/go-iam/internal/repo"
)

// AuthHandler handles authentication requests.
type AuthHandler struct {
	userRepo   repo.UserRepository
	memberRepo repo.MemberRepository
	apiKeyRepo *repo.ApiKeyRepo
	orgRepo    *repo.OrgRepo
	jwtIssuer  *auth.Issuer
}

// NewAuthHandler creates an AuthHandler with the given dependencies.
func NewAuthHandler(userRepo repo.UserRepository, memberRepo repo.MemberRepository, apiKeyRepo *repo.ApiKeyRepo, orgRepo *repo.OrgRepo, jwtIssuer *auth.Issuer) *AuthHandler {
	return &AuthHandler{
		userRepo:   userRepo,
		memberRepo: memberRepo,
		apiKeyRepo: apiKeyRepo,
		orgRepo:    orgRepo,
		jwtIssuer:  jwtIssuer,
	}
}

type loginRequest struct {
	Email    string  `json:"email"`
	Password string  `json:"password"`
	OrgID    *string `json:"org_id,omitempty"`
}

type loginResponse struct {
	Token     string      `json:"token"`
	ExpiresAt int64       `json:"expires_at"`
	User      *model.User `json:"user"`
}

type orgSelectionResponse struct {
	Error ErrorDetail         `json:"error"`
	Orgs  []model.UserOrgView `json:"organizations"`
}

// Login handles POST /auth/login.
func (h *AuthHandler) Login(w http.ResponseWriter, r *http.Request) {
	var req loginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		WriteError(w, http.StatusBadRequest, "INVALID_INPUT", "invalid request body")
		return
	}

	user, err := h.userRepo.GetByEmail(r.Context(), req.Email)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			WriteError(w, http.StatusUnauthorized, "AUTH_REQUIRED", "invalid email or password")
			return
		}
		WriteError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "failed to get user")
		return
	}

	if user.Status == "disabled" {
		WriteError(w, http.StatusForbidden, "FORBIDDEN", "user account is disabled")
		return
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(req.Password)); err != nil {
		WriteError(w, http.StatusUnauthorized, "AUTH_REQUIRED", "invalid email or password")
		return
	}

	cost, _ := bcrypt.Cost([]byte(user.PasswordHash))
	if cost < 12 {
		if newHash, err := bcrypt.GenerateFromPassword([]byte(req.Password), 12); err == nil {
			_ = h.userRepo.UpdatePasswordHash(r.Context(), user.UserID, string(newHash))
		}
	}

	orgs, err := h.memberRepo.GetUserOrgs(r.Context(), user.UserID)
	if err != nil {
		WriteError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "failed to get user organizations")
		return
	}
	if len(orgs) == 0 {
		WriteError(w, http.StatusForbidden, "FORBIDDEN", "user has no organization binding")
		return
	}

	var selectedOrg model.UserOrgView
	if len(orgs) == 1 {
		selectedOrg = orgs[0]
	} else {
		if req.OrgID == nil || *req.OrgID == "" {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusBadRequest)
			_ = json.NewEncoder(w).Encode(orgSelectionResponse{
				Error: ErrorDetail{Code: "ORG_SELECTION_REQUIRED", Message: "multiple organizations found, please specify org_id"},
				Orgs:  orgs,
			})
			return
		}
		found := false
		for _, o := range orgs {
			if o.OrgID == *req.OrgID {
				selectedOrg = o
				found = true
				break
			}
		}
		if !found {
			WriteError(w, http.StatusBadRequest, "INVALID_INPUT", "specified org_id not found in user's organizations")
			return
		}
	}

	brandID := ""
	if selectedOrg.BrandID != nil {
		brandID = *selectedOrg.BrandID
	}

	result, err := h.jwtIssuer.Issue(auth.IssueInput{
		UserID:  user.UserID,
		Role:    selectedOrg.Role,
		OrgID:   selectedOrg.OrgID,
		BrandID: brandID,
	})
	if err != nil {
		WriteError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "failed to issue token")
		return
	}

	WriteJSON(w, http.StatusOK, loginResponse{
		Token:     result.Token,
		ExpiresAt: result.ExpiresAt,
		User:      user,
	})
}

// ValidateApiKey handles POST /auth/validate-api-key.
// Legacy compatibility only: this endpoint validates only go-iam legacy/backoffice
// brand_* org-scoped keys. Current Gateway runtime brand keys (rcpk_live_*) do not
// use this endpoint as their primary source of truth.
func (h *AuthHandler) ValidateApiKey(w http.ResponseWriter, r *http.Request) {
	var req struct {
		ApiKey string `json:"api_key"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		WriteError(w, http.StatusBadRequest, "INVALID_INPUT", "invalid request body")
		return
	}

	if req.ApiKey == "" {
		WriteError(w, http.StatusBadRequest, "INVALID_INPUT", "api_key is required")
		return
	}

	// Legacy format only. Current Gateway-facing keys are not validated here.
	orgID, err := extractOrgIDFromKey(req.ApiKey)
	if err != nil {
		WriteError(w, http.StatusUnauthorized, "INVALID_API_KEY", "legacy API key format not recognized by go-iam compatibility endpoint")
		return
	}

	keys, err := h.apiKeyRepo.GetActiveKeysByOrg(r.Context(), orgID)
	if err != nil {
		WriteError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "failed to query API keys")
		return
	}

	var matchedKey *model.BrandApiKey
	for i := range keys {
		if err := bcrypt.CompareHashAndPassword([]byte(keys[i].KeyHash), []byte(req.ApiKey)); err == nil {
			matchedKey = &keys[i]
			break
		}
	}

	if matchedKey == nil {
		WriteError(w, http.StatusUnauthorized, "INVALID_API_KEY", "API key not found or revoked")
		return
	}

	org, err := h.orgRepo.GetByID(r.Context(), orgID)
	if err != nil {
		WriteError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "failed to query organization")
		return
	}
	if org == nil {
		WriteError(w, http.StatusUnauthorized, "INVALID_API_KEY", "organization not found")
		return
	}

	go func() {
		_ = h.apiKeyRepo.UpdateLastUsed(r.Context(), matchedKey.KeyID)
	}()

	response := model.ApiKeyValidateResponse{
		OrgID:   org.OrgID,
		BrandID: org.BrandID,
		OrgName: org.OrgName,
	}
	WriteSuccess(w, http.StatusOK, response)
}
