package handler

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/crypto/bcrypt"

	"rcprotocol/services/go-iam/internal/auth"
	"rcprotocol/services/go-iam/internal/model"
	"rcprotocol/services/go-iam/internal/repo"
)

func newTestAuthHandler() (*AuthHandler, *mockUserRepo, *mockMemberRepo) {
	userRepo := newMockUserRepo()
	memberRepo := newMockMemberRepo()
	apiKeyRepo := &repo.ApiKeyRepo{}
	orgRepo := &repo.OrgRepo{}
	issuer := auth.NewIssuer("test-secret-key", 24)
	h := NewAuthHandler(userRepo, memberRepo, apiKeyRepo, orgRepo, issuer)
	return h, userRepo, memberRepo
}

func seedActiveUser(userRepo *mockUserRepo, userID, email, password string) {
	hash, _ := bcrypt.GenerateFromPassword([]byte(password), 10)
	userRepo.users[userID] = &model.User{
		UserID:       userID,
		Email:        email,
		PasswordHash: string(hash),
		DisplayName:  "Test User",
		Status:       "active",
	}
}

func TestAuthLogin200_SingleOrg(t *testing.T) {
	h, userRepo, memberRepo := newTestAuthHandler()
	seedActiveUser(userRepo, "user-1", "test@example.com", "password123")
	memberRepo.userOrgs["user-1"] = []model.UserOrgView{
		{OrgID: "org-1", OrgName: "Acme", OrgType: "platform", Role: "Platform"},
	}

	body := `{"email":"test@example.com","password":"password123"}`
	req := httptest.NewRequest(http.MethodPost, "/auth/login", bytes.NewBufferString(body))
	w := httptest.NewRecorder()
	h.Login(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var resp loginResponse
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if resp.Token == "" {
		t.Error("expected token in response")
	}
	if resp.ExpiresAt == 0 {
		t.Error("expected expires_at in response")
	}
	if resp.User == nil {
		t.Fatal("expected user in response")
	}
	if resp.User.UserID != "user-1" {
		t.Errorf("expected user-1, got %s", resp.User.UserID)
	}
	if resp.User.Email != "test@example.com" {
		t.Errorf("expected test@example.com, got %s", resp.User.Email)
	}

	// Verify password_hash is not in JSON
	var raw map[string]json.RawMessage
	_ = json.Unmarshal(w.Body.Bytes(), &raw)
	var userMap map[string]interface{}
	_ = json.Unmarshal(raw["user"], &userMap)
	if _, exists := userMap["password_hash"]; exists {
		t.Error("password_hash must not appear in JSON response")
	}
}

func TestAuthLogin401_EmailNotFound(t *testing.T) {
	h, _, _ := newTestAuthHandler()

	body := `{"email":"nobody@example.com","password":"password123"}`
	req := httptest.NewRequest(http.MethodPost, "/auth/login", bytes.NewBufferString(body))
	w := httptest.NewRecorder()
	h.Login(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d: %s", w.Code, w.Body.String())
	}
	var resp ErrorBody
	_ = json.Unmarshal(w.Body.Bytes(), &resp)
	if resp.Error.Message != "invalid email or password" {
		t.Errorf("unexpected message: %q", resp.Error.Message)
	}
}

func TestAuthLogin401_WrongPassword(t *testing.T) {
	h, userRepo, memberRepo := newTestAuthHandler()
	seedActiveUser(userRepo, "user-1", "test@example.com", "password123")
	memberRepo.userOrgs["user-1"] = []model.UserOrgView{
		{OrgID: "org-1", OrgName: "Acme", OrgType: "platform", Role: "Platform"},
	}

	body := `{"email":"test@example.com","password":"wrongpassword"}`
	req := httptest.NewRequest(http.MethodPost, "/auth/login", bytes.NewBufferString(body))
	w := httptest.NewRecorder()
	h.Login(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d: %s", w.Code, w.Body.String())
	}
	var resp ErrorBody
	_ = json.Unmarshal(w.Body.Bytes(), &resp)
	if resp.Error.Message != "invalid email or password" {
		t.Errorf("unexpected message: %q", resp.Error.Message)
	}
}

func TestAuthLogin403_Disabled(t *testing.T) {
	h, userRepo, _ := newTestAuthHandler()
	seedActiveUser(userRepo, "user-1", "test@example.com", "password123")
	userRepo.users["user-1"].Status = "disabled"

	body := `{"email":"test@example.com","password":"password123"}`
	req := httptest.NewRequest(http.MethodPost, "/auth/login", bytes.NewBufferString(body))
	w := httptest.NewRecorder()
	h.Login(w, req)

	if w.Code != http.StatusForbidden {
		t.Fatalf("expected 403, got %d: %s", w.Code, w.Body.String())
	}
	var resp ErrorBody
	_ = json.Unmarshal(w.Body.Bytes(), &resp)
	if resp.Error.Message != "user account is disabled" {
		t.Errorf("unexpected message: %q", resp.Error.Message)
	}
}

func TestAuthLogin403_NoOrgBinding(t *testing.T) {
	h, userRepo, _ := newTestAuthHandler()
	seedActiveUser(userRepo, "user-1", "test@example.com", "password123")
	// No org bindings set

	body := `{"email":"test@example.com","password":"password123"}`
	req := httptest.NewRequest(http.MethodPost, "/auth/login", bytes.NewBufferString(body))
	w := httptest.NewRecorder()
	h.Login(w, req)

	if w.Code != http.StatusForbidden {
		t.Fatalf("expected 403, got %d: %s", w.Code, w.Body.String())
	}
	var resp ErrorBody
	_ = json.Unmarshal(w.Body.Bytes(), &resp)
	if resp.Error.Message != "user has no organization binding" {
		t.Errorf("unexpected message: %q", resp.Error.Message)
	}
}

func TestAuthLogin400_MultipleOrgsNoOrgID(t *testing.T) {
	h, userRepo, memberRepo := newTestAuthHandler()
	seedActiveUser(userRepo, "user-1", "test@example.com", "password123")
	brandID := "brand-1"
	memberRepo.userOrgs["user-1"] = []model.UserOrgView{
		{OrgID: "org-1", OrgName: "Platform Co", OrgType: "platform", Role: "Platform"},
		{OrgID: "org-2", OrgName: "Brand Co", OrgType: "brand", Role: "Brand", BrandID: &brandID},
	}

	body := `{"email":"test@example.com","password":"password123"}`
	req := httptest.NewRequest(http.MethodPost, "/auth/login", bytes.NewBufferString(body))
	w := httptest.NewRecorder()
	h.Login(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d: %s", w.Code, w.Body.String())
	}

	var resp orgSelectionResponse
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if resp.Error.Code != "ORG_SELECTION_REQUIRED" {
		t.Errorf("expected ORG_SELECTION_REQUIRED, got %s", resp.Error.Code)
	}
	if len(resp.Orgs) != 2 {
		t.Errorf("expected 2 orgs, got %d", len(resp.Orgs))
	}
}

func TestAuthLogin200_MultipleOrgsWithOrgID(t *testing.T) {
	h, userRepo, memberRepo := newTestAuthHandler()
	seedActiveUser(userRepo, "user-1", "test@example.com", "password123")
	brandID := "brand-1"
	memberRepo.userOrgs["user-1"] = []model.UserOrgView{
		{OrgID: "org-1", OrgName: "Platform Co", OrgType: "platform", Role: "Platform"},
		{OrgID: "org-2", OrgName: "Brand Co", OrgType: "brand", Role: "Brand", BrandID: &brandID},
	}

	body := `{"email":"test@example.com","password":"password123","org_id":"org-2"}`
	req := httptest.NewRequest(http.MethodPost, "/auth/login", bytes.NewBufferString(body))
	w := httptest.NewRecorder()
	h.Login(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var resp loginResponse
	_ = json.Unmarshal(w.Body.Bytes(), &resp)
	if resp.Token == "" {
		t.Error("expected token")
	}
}

func TestAuthLogin_JWTClaimsCorrect(t *testing.T) {
	h, userRepo, memberRepo := newTestAuthHandler()
	seedActiveUser(userRepo, "user-1", "test@example.com", "password123")
	brandID := "brand-abc"
	memberRepo.userOrgs["user-1"] = []model.UserOrgView{
		{OrgID: "org-1", OrgName: "Brand Co", OrgType: "brand", Role: "Brand", BrandID: &brandID},
	}

	body := `{"email":"test@example.com","password":"password123"}`
	req := httptest.NewRequest(http.MethodPost, "/auth/login", bytes.NewBufferString(body))
	w := httptest.NewRecorder()
	h.Login(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var resp loginResponse
	_ = json.Unmarshal(w.Body.Bytes(), &resp)

	// Decode and verify JWT claims
	token, err := jwt.ParseWithClaims(resp.Token, &auth.Claims{}, func(token *jwt.Token) (interface{}, error) {
		return []byte("test-secret-key"), nil
	})
	if err != nil {
		t.Fatalf("parse JWT: %v", err)
	}

	claims, ok := token.Claims.(*auth.Claims)
	if !ok || !token.Valid {
		t.Fatal("invalid token claims")
	}

	if claims.Sub != "user-1" {
		t.Errorf("expected sub user-1, got %s", claims.Sub)
	}
	if claims.Role != "Brand" {
		t.Errorf("expected role Brand, got %s", claims.Role)
	}
	if claims.OrgID != "org-1" {
		t.Errorf("expected org_id org-1, got %s", claims.OrgID)
	}
	if claims.BrandID != "brand-abc" {
		t.Errorf("expected brand_id brand-abc, got %s", claims.BrandID)
	}
	if claims.Scopes == nil || len(claims.Scopes) != 0 {
		t.Errorf("expected empty scopes, got %v", claims.Scopes)
	}
	if claims.ExpiresAt == nil {
		t.Error("expected expires_at in claims")
	}
	if claims.IssuedAt == nil {
		t.Error("expected issued_at in claims")
	}
}
