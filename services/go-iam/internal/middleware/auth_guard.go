package middleware

import (
	"context"
	"net/http"
)

// Claims 从 Gateway 注入的 X-Claims-* 头解析身份信息
type Claims struct {
	Sub     string
	Role    string
	OrgID   string
	BrandID string
}

type claimsKey struct{}

// ClaimsFromContext 从 context 中获取 Claims
func ClaimsFromContext(ctx context.Context) (*Claims, bool) {
	c, ok := ctx.Value(claimsKey{}).(*Claims)
	return c, ok
}

// ClaimsFromRequest 从请求的 X-Claims-* 头解析 Claims
func ClaimsFromRequest(r *http.Request) *Claims {
	return &Claims{
		Sub:     r.Header.Get("X-Claims-Sub"),
		Role:    r.Header.Get("X-Claims-Role"),
		OrgID:   r.Header.Get("X-Claims-Org-Id"),
		BrandID: r.Header.Get("X-Claims-Brand-Id"),
	}
}

// Valid 检查 Claims 是否有效（至少有 Sub 和合法 Role）
func (c *Claims) Valid() bool {
	if c.Sub == "" || c.Role == "" {
		return false
	}
	switch c.Role {
	case "Platform", "Brand", "Factory", "Consumer", "Moderator":
		return true
	default:
		return false
	}
}

// AuthGuard 中间件：从 X-Claims-* 头提取身份，注入 context
func AuthGuard(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		claims := ClaimsFromRequest(r)
		if !claims.Valid() {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusUnauthorized)
			_, _ = w.Write([]byte(`{"error":"AUTH_REQUIRED","message":"valid identity claims required"}`))
			return
		}
		ctx := context.WithValue(r.Context(), claimsKey{}, claims)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// RequireRole 返回中间件：只允许指定角色通过
func RequireRole(roles ...string) func(http.Handler) http.Handler {
	allowed := make(map[string]bool, len(roles))
	for _, r := range roles {
		allowed[r] = true
	}
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			claims, ok := ClaimsFromContext(r.Context())
			if !ok || !allowed[claims.Role] {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusForbidden)
				_, _ = w.Write([]byte(`{"error":"FORBIDDEN","message":"insufficient role"}`))
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}
