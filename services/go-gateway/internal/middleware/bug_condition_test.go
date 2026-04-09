package middleware

// Bug Condition 探索性测试 — Gateway 身份信任链缺陷
//
// 这些测试编码了**期望行为**：在未修复代码上应当 FAIL，证明缺陷存在。
// 修复后测试通过即确认修复成功。
//
// **Validates: Requirements 1.17, 1.18**

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

// ── BUG 1.17: Gateway JWT 验签成功后未注入 X-Claims-* 头 ──
// Bug: Auth 中间件验签后直接调用 next.ServeHTTP(w, r)，不注入 Claims 到请求头。
// 期望行为: 验签成功后应在请求头中注入 X-Claims-Sub/Role/Org-Id/Brand-Id。
func TestBug_1_17_GatewayJWTValid_ClaimsShouldBeInjected(t *testing.T) {
	var receivedSub string
	inner := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		receivedSub = r.Header.Get("X-Claims-Sub")
		w.WriteHeader(http.StatusOK)
	})

	handler := Auth(testSecret)(inner)

	claims := &Claims{
		Sub:     "user-42",
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
		t.Fatalf("expected 200, got %d", rr.Code)
	}

	// 期望: 下游应收到 X-Claims-Sub = "user-42"
	if receivedSub != "user-42" {
		t.Fatalf("BUG 1.17: Gateway JWT 验签成功后未注入 X-Claims-Sub 头。期望 'user-42'，实际 '%s'", receivedSub)
	}
}

// ── BUG 1.18: 直连后端伪造 X-Claims-* 头被信任 ──
// Bug: Gateway Auth 中间件不剥离客户端伪造的 X-Claims-* 头。
// 期望行为: 应在验签前删除所有 X-Claims-* 头，防止伪造。
func TestBug_1_18_ForgedClaimsHeaders_ShouldBeStripped(t *testing.T) {
	var receivedSub string
	inner := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		receivedSub = r.Header.Get("X-Claims-Sub")
		w.WriteHeader(http.StatusOK)
	})

	handler := Auth(testSecret)(inner)

	// 构造合法 JWT，但同时伪造 X-Claims-Sub
	claims := &Claims{
		Sub:  "legit-user",
		Role: "Consumer",
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Hour)),
		},
	}
	token := signTestToken(claims, testSecret)

	req := httptest.NewRequest(http.MethodGet, "/api/brands", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	// 攻击者伪造的 X-Claims-Sub
	req.Header.Set("X-Claims-Sub", "admin-evil")

	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rr.Code)
	}

	// 期望: X-Claims-Sub 应被来自 JWT 的真实值覆盖，不应是伪造值
	if receivedSub == "admin-evil" {
		t.Fatal("BUG 1.18: 客户端伪造的 X-Claims-Sub='admin-evil' 被下游接受，Gateway 未剥离伪造头")
	}
	// 如果 Gateway 修复后会注入 JWT 中的 claims，此处应为 legit-user
	if receivedSub != "legit-user" {
		t.Fatalf("BUG 1.17+1.18: X-Claims-Sub 既不是伪造值也不是 JWT 中的值。实际: '%s'", receivedSub)
	}
}
