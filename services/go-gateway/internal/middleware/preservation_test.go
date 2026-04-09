package middleware

// Preservation（保持性）测试 — Gateway 合法路由不受影响
//
// 在未修复代码上运行确认基线行为正确。修复后重新运行确认无回归。
//
// **Validates: Requirements 3.8, 3.9**

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

// ── 3.8: 合法 JWT 请求正常转发 ──

func TestPreservation_3_8_ValidJWT_ForwardedToDownstream(t *testing.T) {
	var called bool
	inner := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
		w.WriteHeader(http.StatusOK)
	})

	handler := Auth(testSecret)(inner)

	claims := &Claims{
		Sub:     "user-1",
		Role:    "Brand",
		OrgID:   "org-1",
		BrandID: "brand-1",
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Hour)),
		},
	}
	token := signTestToken(claims, testSecret)

	req := httptest.NewRequest(http.MethodGet, "/api/brands", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("3.8: 合法 JWT 请求应返回 200，实际 %d", rr.Code)
	}
	if !called {
		t.Fatal("3.8: 合法 JWT 请求应被转发到下游处理器")
	}
}

func TestPreservation_3_8_ValidJWT_AuthorizationPreserved(t *testing.T) {
	var receivedAuth string
	inner := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		receivedAuth = r.Header.Get("Authorization")
		w.WriteHeader(http.StatusOK)
	})

	handler := Auth(testSecret)(inner)

	claims := &Claims{
		Sub:  "user-2",
		Role: "Platform",
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Hour)),
		},
	}
	token := signTestToken(claims, testSecret)
	bearer := "Bearer " + token

	req := httptest.NewRequest(http.MethodGet, "/api/assets", nil)
	req.Header.Set("Authorization", bearer)
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	if receivedAuth != bearer {
		t.Fatalf("3.8: Authorization 头应被原样传递到下游。期望 %q，实际 %q", bearer, receivedAuth)
	}
}

func TestPreservation_3_8_InvalidJWT_Rejected(t *testing.T) {
	var called bool
	handler := Auth(testSecret)(handlerCalled(&called))

	req := httptest.NewRequest(http.MethodGet, "/api/brands", nil)
	req.Header.Set("Authorization", "Bearer invalid-token")
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusUnauthorized {
		t.Fatalf("3.8: 无效 JWT 应返回 401，实际 %d", rr.Code)
	}
	if called {
		t.Fatal("3.8: 无效 JWT 不应转发到下游")
	}
}

// ── 3.9: /auth/login 公开路径不受鉴权影响 ──

func TestPreservation_3_9_IAMLogin_PublicPathBypass(t *testing.T) {
	var called bool
	handler := Auth(testSecret)(handlerCalled(&called))

	// /api/iam/auth/login 是公开路径，不需要 JWT
	req := httptest.NewRequest(http.MethodPost, "/api/iam/auth/login", nil)
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("3.9: /api/iam/auth/login 应无需 JWT 返回 200，实际 %d", rr.Code)
	}
	if !called {
		t.Fatal("3.9: /api/iam/auth/login 应被转发到下游")
	}
}

func TestPreservation_3_9_VerifyPath_PublicBypass(t *testing.T) {
	var called bool
	handler := Auth(testSecret)(handlerCalled(&called))

	req := httptest.NewRequest(http.MethodGet, "/api/v1/verify?uid=04A3B2C1", nil)
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("3.9: /api/verify 应无需 JWT 返回 200，实际 %d", rr.Code)
	}
	if !called {
		t.Fatal("3.9: /api/verify 应被转发到下游")
	}
}

func TestPreservation_3_8_Healthz_AlwaysAllowed(t *testing.T) {
	var called bool
	handler := Auth(testSecret)(handlerCalled(&called))

	req := httptest.NewRequest(http.MethodGet, "/healthz", nil)
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("3.8: /healthz 应返回 200，实际 %d", rr.Code)
	}
	if !called {
		t.Fatal("3.8: /healthz 应被转发到下游")
	}
}
