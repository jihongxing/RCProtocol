package handler

// Bug Condition 探索性测试 — go-iam 鉴权缺陷
//
// 这些测试编码了**期望行为**：在未修复代码上应当 FAIL，证明缺陷存在。
// 修复后测试通过即确认修复成功。
//
// **Validates: Requirements 1.19**

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/chi/v5"
	"rcprotocol/services/go-iam/internal/middleware"
)

// ── BUG 1.19: go-iam 管理接口无鉴权 ──
// Bug: POST /users 等管理接口未添加鉴权中间件，任何匿名请求均可访问。
// 期望行为: 管理接口应要求 Platform 角色的有效 Claims。
func TestBug_1_19_IAM_UsersCreate_WithoutJWT_ShouldReturn401(t *testing.T) {
	userRepo := newMockUserRepo()
	userH := NewUserHandler(userRepo)

	r := chi.NewRouter()
	// 模拟 go-iam 的路由注册方式（与 main.go 修复后一致，含 AuthGuard + RequireRole）
	r.Group(func(r chi.Router) {
		r.Use(middleware.AuthGuard)
		r.Use(middleware.RequireRole("Platform"))
		r.Post("/users", userH.Create)
	})

	body := map[string]string{
		"email":    "evil@example.com",
		"name":     "Evil Admin",
		"password": "password123",
	}
	bodyBytes, _ := json.Marshal(body)

	req := httptest.NewRequest(http.MethodPost, "/users", bytes.NewReader(bodyBytes))
	req.Header.Set("Content-Type", "application/json")
	// 故意不设置 Authorization 或 X-Claims-* 头

	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, req)

	// 期望: 无鉴权请求应返回 401
	if rr.Code == http.StatusOK || rr.Code == http.StatusCreated {
		t.Fatalf(
			"BUG 1.19: POST /users 无 JWT 返回 %d，应返回 401。管理接口缺少鉴权中间件",
			rr.Code,
		)
	}
}
