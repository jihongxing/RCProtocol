package auth

import (
	"context"
	"encoding/json"
	"net/http"
)

// Claims represents the identity fields injected by Gateway via X-Claims-* headers.
type Claims struct {
	Sub     string `json:"sub"`
	Role    string `json:"role"`
	OrgID   string `json:"org_id"`
	BrandID string `json:"brand_id"`
}

type claimsKey struct{}

// ClaimsFromContext extracts Claims from request context (injected by Middleware).
func ClaimsFromContext(ctx context.Context) *Claims {
	c, _ := ctx.Value(claimsKey{}).(*Claims)
	return c
}

// NewContext returns a context with Claims injected, for use in tests and internal callers.
func NewContext(ctx context.Context, c *Claims) context.Context {
	return context.WithValue(ctx, claimsKey{}, c)
}

// ParseClaims extracts Claims from Gateway-injected X-Claims-* headers.
// 不再自行 base64 解码 JWT，信任 Gateway 已验签并注入身份头。
func ParseClaims(r *http.Request) (*Claims, string) {
	sub := r.Header.Get("X-Claims-Sub")
	role := r.Header.Get("X-Claims-Role")
	if sub == "" || role == "" {
		return nil, "missing identity headers (X-Claims-Sub, X-Claims-Role)"
	}

	claims := &Claims{
		Sub:     sub,
		Role:    role,
		OrgID:   r.Header.Get("X-Claims-Org-Id"),
		BrandID: r.Header.Get("X-Claims-Brand-Id"),
	}

	// 角色白名单校验
	switch claims.Role {
	case "Platform", "Brand", "Factory", "Consumer", "Moderator":
		// valid
	default:
		return nil, "invalid role: " + claims.Role
	}

	return claims, ""
}

// Middleware is the Claims parsing middleware.
// Skips /healthz; on failure returns 401; on success injects Claims into context.
func Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/healthz" {
			next.ServeHTTP(w, r)
			return
		}
		claims, errMsg := ParseClaims(r)
		if errMsg != "" {
			w.Header().Set("Content-Type", "application/json; charset=utf-8")
			w.WriteHeader(http.StatusUnauthorized)
			_ = json.NewEncoder(w).Encode(map[string]interface{}{
				"error": map[string]string{
					"code":    "AUTH_REQUIRED",
					"message": errMsg,
				},
			})
			return
		}
		ctx := context.WithValue(r.Context(), claimsKey{}, claims)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}
