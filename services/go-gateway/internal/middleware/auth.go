package middleware

import (
	"crypto/sha256"
	"encoding/hex"
	"net/http"
	"strings"

	"github.com/golang-jwt/jwt/v5"

	"rcprotocol/services/go-gateway/internal/response"
)

const (
	// ApiKeyHeader is the canonical header for API Key authentication.
	ApiKeyHeader = "X-Api-Key"
	// ApiKeyPrefix is the required prefix for current RCProtocol brand API Keys.
	ApiKeyPrefix = "rcpk_live_"
)

// Claims represents the JWT payload structure used by RCProtocol.
type Claims struct {
	Sub     string   `json:"sub"`
	Role    string   `json:"role"`
	OrgID   string   `json:"org_id,omitempty"`
	BrandID string   `json:"brand_id,omitempty"`
	Scopes  []string `json:"scopes,omitempty"`
	jwt.RegisteredClaims
}

// validRoles 是协议定义的 5 角色白名单
var validRoles = map[string]bool{
	"Platform":  true,
	"Brand":     true,
	"Factory":   true,
	"Consumer":  true,
	"Moderator": true,
}

// publicPrefixes lists URL prefixes that bypass JWT authentication.
var publicPrefixes = []string{
	"/health",
	"/healthz",
	"/api/v1/verify",
	"/api/iam/auth/login",
}

// apiKeyRoutes lists URL prefixes that support API Key authentication.
var apiKeyRoutes = []string{
	"/api/v1/batches",
	"/api/v1/assets/blind-scan",
	"/api/v1/assets/activate",
	"/api/v1/assets/sell",
	"/api/v1/assets/",
	"/api/bff/console/brands/",
	"/api/bff/console/dashboard",
	"/api/bff/app/assets",
	"/api/bff/app/assets/",
}

// claimsHeaders 是 Gateway 注入的身份头，必须在验签前剥离客户端伪造值
var claimsHeaders = []string{
	"X-Claims-Sub",
	"X-Claims-Role",
	"X-Claims-Org-Id",
	"X-Claims-Brand-Id",
	"X-Api-Key-Verified",
}

// isPublicPath returns true for routes that do not require authentication.
func isPublicPath(path string) bool {
	for _, prefix := range publicPrefixes {
		if strings.HasPrefix(path, prefix) {
			return true
		}
	}
	return false
}

// supportsApiKey returns true if the path supports API Key authentication.
func supportsApiKey(path string) bool {
	for _, prefix := range apiKeyRoutes {
		if strings.HasPrefix(path, prefix) {
			return true
		}
	}
	return false
}

// hashApiKey creates a SHA-256 hash of the API key for comparison.
func hashApiKey(apiKey string) string {
	hash := sha256.Sum256([]byte(apiKey))
	return hex.EncodeToString(hash[:])
}

// ApiKeyValidator is an interface for validating API keys against the database.
type ApiKeyValidator interface {
	ValidateApiKey(keyHash string) (brandID string, valid bool, err error)
}

// Auth returns a middleware that validates HS256 JWT tokens or API Keys.
// 验签前剥离客户端伪造的 X-Claims-* 头，验签成功后注入可信身份头。
// Public paths are passed through without authentication (but仍剥离伪造头).
func Auth(jwtSecret string) func(http.Handler) http.Handler {
	secretBytes := []byte(jwtSecret)

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// 始终剥离客户端伪造的 Claims 头——即使 public path 也不允许注入
			for _, h := range claimsHeaders {
				r.Header.Del(h)
			}

			if isPublicPath(r.URL.Path) {
				next.ServeHTTP(w, r)
				return
			}

			traceID := r.Header.Get(TraceIDHeader)

			// API Key 认证路径（优先于 JWT）
			apiKey := r.Header.Get(ApiKeyHeader)
			if apiKey != "" {
				if !supportsApiKey(r.URL.Path) {
					response.WriteError(w, http.StatusUnauthorized,
						response.CodeAuthRequired, "API Key not supported for this route", traceID)
					return
				}

				// 验证 API Key 格式
				if !strings.HasPrefix(apiKey, ApiKeyPrefix) {
					response.WriteError(w, http.StatusUnauthorized,
						response.CodeAuthRequired, "invalid API Key format (must start with rcpk_live_)", traceID)
					return
				}

				// 计算 API Key 哈希
				keyHash := hashApiKey(apiKey)

				// 注入哈希到请求头，让上游服务验证
				// Gateway 只负责格式校验与哈希化，不负责数据库真验证。
				r.Header.Set("X-Api-Key-Hash", keyHash)
				r.Header.Set("X-Api-Key-Verified", "hash-only")
				r.Header.Set("X-Claims-Role", "Brand")

				next.ServeHTTP(w, r)
				return
			}

			// JWT 认证
			authHeader := r.Header.Get("Authorization")
			if authHeader == "" {
				response.WriteError(w, http.StatusUnauthorized,
					response.CodeAuthRequired, "missing Authorization header or X-Api-Key", traceID)
				return
			}

			tokenStr := strings.TrimPrefix(authHeader, "Bearer ")
			if tokenStr == authHeader {
				response.WriteError(w, http.StatusUnauthorized,
					response.CodeAuthRequired, "invalid Authorization format, expected Bearer token", traceID)
				return
			}

			token, err := jwt.ParseWithClaims(tokenStr, &Claims{}, func(t *jwt.Token) (interface{}, error) {
				if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
					return nil, jwt.ErrSignatureInvalid
				}
				return secretBytes, nil
			})

			if err != nil || !token.Valid {
				response.WriteError(w, http.StatusUnauthorized,
					response.CodeAuthRequired, "invalid or expired token", traceID)
				return
			}

			claims, ok := token.Claims.(*Claims)
			if !ok {
				response.WriteError(w, http.StatusUnauthorized,
					response.CodeAuthRequired, "invalid token claims", traceID)
				return
			}

			// 角色白名单校验
			if !validRoles[claims.Role] {
				response.WriteError(w, http.StatusForbidden,
					"INVALID_ROLE", "role not allowed: "+claims.Role, traceID)
				return
			}

			// 注入可信身份头到下游请求
			r.Header.Set("X-Claims-Sub", claims.Sub)
			r.Header.Set("X-Claims-Role", claims.Role)
			if claims.OrgID != "" {
				r.Header.Set("X-Claims-Org-Id", claims.OrgID)
			}
			if claims.BrandID != "" {
				r.Header.Set("X-Claims-Brand-Id", claims.BrandID)
			}

			next.ServeHTTP(w, r)
		})
	}
}
