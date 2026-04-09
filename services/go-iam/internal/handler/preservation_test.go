package handler

// Preservation（保持性）测试 — IAM 登录不受影响
//
// 在未修复代码上运行确认基线行为正确。修复后重新运行确认无回归。
//
// **Validates: Requirements 3.9**

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"rcprotocol/services/go-iam/internal/model"
)

// ── 3.9: /auth/login 正常返回 JWT ──

func TestPreservation_3_9_Login_ReturnsJWTNormally(t *testing.T) {
	h, userRepo, memberRepo := newTestAuthHandler()
	seedActiveUser(userRepo, "user-1", "login@example.com", "password123")
	memberRepo.userOrgs["user-1"] = []model.UserOrgView{
		{OrgID: "org-1", OrgName: "Platform Co", OrgType: "platform", Role: "Platform"},
	}

	body := `{"email":"login@example.com","password":"password123"}`
	req := httptest.NewRequest(http.MethodPost, "/auth/login", bytes.NewBufferString(body))
	w := httptest.NewRecorder()
	h.Login(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("3.9: 合法用户登录应返回 200，实际 %d: %s", w.Code, w.Body.String())
	}

	var resp loginResponse
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("3.9: 无法解析登录响应: %v", err)
	}
	if resp.Token == "" {
		t.Fatal("3.9: 登录响应应包含 JWT token")
	}
	if resp.ExpiresAt == 0 {
		t.Fatal("3.9: 登录响应应包含 expires_at")
	}
	if resp.User == nil {
		t.Fatal("3.9: 登录响应应包含用户信息")
	}
}

func TestPreservation_3_9_Login_InvalidPassword_Returns401(t *testing.T) {
	h, userRepo, memberRepo := newTestAuthHandler()
	seedActiveUser(userRepo, "user-1", "login@example.com", "password123")
	memberRepo.userOrgs["user-1"] = []model.UserOrgView{
		{OrgID: "org-1", OrgName: "Platform Co", OrgType: "platform", Role: "Platform"},
	}

	body := `{"email":"login@example.com","password":"wrong_password"}`
	req := httptest.NewRequest(http.MethodPost, "/auth/login", bytes.NewBufferString(body))
	w := httptest.NewRecorder()
	h.Login(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Fatalf("3.9: 错误密码应返回 401，实际 %d", w.Code)
	}
}

func TestPreservation_3_9_Login_NonExistentUser_Returns401(t *testing.T) {
	h, _, _ := newTestAuthHandler()

	body := `{"email":"nobody@example.com","password":"password123"}`
	req := httptest.NewRequest(http.MethodPost, "/auth/login", bytes.NewBufferString(body))
	w := httptest.NewRecorder()
	h.Login(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Fatalf("3.9: 不存在的用户登录应返回 401，实际 %d", w.Code)
	}
}

func TestPreservation_3_9_Login_DisabledUser_Returns403(t *testing.T) {
	h, userRepo, _ := newTestAuthHandler()
	seedActiveUser(userRepo, "user-1", "disabled@example.com", "password123")
	userRepo.users["user-1"].Status = "disabled"

	body := `{"email":"disabled@example.com","password":"password123"}`
	req := httptest.NewRequest(http.MethodPost, "/auth/login", bytes.NewBufferString(body))
	w := httptest.NewRecorder()
	h.Login(w, req)

	if w.Code != http.StatusForbidden {
		t.Fatalf("3.9: 禁用用户登录应返回 403，实际 %d", w.Code)
	}
}

func TestPreservation_3_9_Login_ResponseExcludesPasswordHash(t *testing.T) {
	h, userRepo, memberRepo := newTestAuthHandler()
	seedActiveUser(userRepo, "user-1", "login@example.com", "password123")
	memberRepo.userOrgs["user-1"] = []model.UserOrgView{
		{OrgID: "org-1", OrgName: "Acme", OrgType: "platform", Role: "Platform"},
	}

	body := `{"email":"login@example.com","password":"password123"}`
	req := httptest.NewRequest(http.MethodPost, "/auth/login", bytes.NewBufferString(body))
	w := httptest.NewRecorder()
	h.Login(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("3.9: 登录应返回 200，实际 %d", w.Code)
	}

	var raw map[string]json.RawMessage
	_ = json.Unmarshal(w.Body.Bytes(), &raw)
	if raw["user"] != nil {
		var userMap map[string]interface{}
		_ = json.Unmarshal(raw["user"], &userMap)
		if _, exists := userMap["password_hash"]; exists {
			t.Fatal("3.9: 登录响应 user 中不应包含 password_hash")
		}
	}
}
