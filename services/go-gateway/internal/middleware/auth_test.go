package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

const testSecret = "test-secret-key-for-jwt-signing"

func signTestToken(claims *Claims, secret string) string {
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	signed, err := token.SignedString([]byte(secret))
	if err != nil {
		panic("signTestToken: " + err.Error())
	}
	return signed
}

func handlerCalled(called *bool) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		*called = true
		w.WriteHeader(http.StatusOK)
	})
}

func TestAuth_NoAuthorization(t *testing.T) {
	called := false
	handler := Auth(testSecret)(handlerCalled(&called))
	req := httptest.NewRequest(http.MethodGet, "/api/brands", nil)
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)
	if rr.Code != http.StatusUnauthorized || called {
		t.Fatalf("expected 401 and no downstream call, got code=%d called=%v", rr.Code, called)
	}
}

func TestAuth_ValidToken(t *testing.T) {
	var called bool
	var receivedAuth, receivedSub, receivedRole string
	inner := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
		receivedAuth = r.Header.Get("Authorization")
		receivedSub = r.Header.Get("X-Claims-Sub")
		receivedRole = r.Header.Get("X-Claims-Role")
		w.WriteHeader(http.StatusOK)
	})
	handler := Auth(testSecret)(inner)

	claims := &Claims{Sub: "user-1", Role: "Brand", RegisteredClaims: jwt.RegisteredClaims{ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Hour))}}
	token := signTestToken(claims, testSecret)
	bearerValue := "Bearer " + token

	req := httptest.NewRequest(http.MethodGet, "/api/brands", nil)
	req.Header.Set("Authorization", bearerValue)
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK || !called || receivedAuth != bearerValue || receivedSub != "user-1" || receivedRole != "Brand" {
		t.Fatalf("unexpected JWT forwarding code=%d called=%v auth=%q sub=%q role=%q", rr.Code, called, receivedAuth, receivedSub, receivedRole)
	}
}

func TestAuth_StripsForgedClaimsHeaders(t *testing.T) {
	var receivedSub string
	inner := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		receivedSub = r.Header.Get("X-Claims-Sub")
		w.WriteHeader(http.StatusOK)
	})
	handler := Auth(testSecret)(inner)

	claims := &Claims{Sub: "real-user", Role: "Platform", RegisteredClaims: jwt.RegisteredClaims{ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Hour))}}
	token := signTestToken(claims, testSecret)

	req := httptest.NewRequest(http.MethodGet, "/api/brands", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("X-Claims-Sub", "forged-admin")
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK || receivedSub != "real-user" {
		t.Fatalf("unexpected stripped claims result code=%d sub=%q", rr.Code, receivedSub)
	}
}

func TestAuth_PublicPathsBypass(t *testing.T) {
	cases := []string{"/healthz", "/api/v1/verify?uid=04A3", "/api/iam/auth/login"}
	for _, path := range cases {
		called := false
		handler := Auth(testSecret)(handlerCalled(&called))
		req := httptest.NewRequest(http.MethodGet, path, nil)
		rr := httptest.NewRecorder()
		handler.ServeHTTP(rr, req)
		if rr.Code != http.StatusOK || !called {
			t.Fatalf("expected public path %s to pass, got code=%d called=%v", path, rr.Code, called)
		}
	}
}

func TestApiKeyAuth_SuccessInjectsHashOnlyHeaders(t *testing.T) {
	var called bool
	var receivedRole, receivedHash, receivedVerified, receivedPlain string
	inner := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
		receivedRole = r.Header.Get("X-Claims-Role")
		receivedHash = r.Header.Get("X-Api-Key-Hash")
		receivedVerified = r.Header.Get("X-Api-Key-Verified")
		receivedPlain = r.Header.Get(ApiKeyHeader)
		w.WriteHeader(http.StatusOK)
	})
	handler := Auth(testSecret)(inner)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/assets/blind-scan", nil)
	req.Header.Set(ApiKeyHeader, "rcpk_live_1234567890abcdef1234567890abcdef")
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK || !called || receivedRole != "Brand" || receivedHash == "" || receivedVerified != "hash-only" || receivedPlain != "rcpk_live_1234567890abcdef1234567890abcdef" {
		t.Fatalf("unexpected api key forwarding code=%d called=%v role=%q hash=%q verified=%q plain=%q", rr.Code, called, receivedRole, receivedHash, receivedVerified, receivedPlain)
	}
}

func TestApiKeyAuth_AllowedBffRouteAccepted(t *testing.T) {
	cases := []string{
		"/api/bff/console/brands/b-001",
		"/api/bff/console/brands/b-001/products",
		"/api/bff/console/dashboard",
		"/api/bff/app/assets",
		"/api/bff/app/assets/a-001",
	}
	for _, path := range cases {
		called := false
		handler := Auth(testSecret)(handlerCalled(&called))
		req := httptest.NewRequest(http.MethodGet, path, nil)
		req.Header.Set(ApiKeyHeader, "rcpk_live_1234567890abcdef1234567890abcdef")
		rr := httptest.NewRecorder()
		handler.ServeHTTP(rr, req)
		if rr.Code != http.StatusOK || !called {
			t.Fatalf("expected 200 and call for allowed BFF route %s, got code=%d called=%v", path, rr.Code, called)
		}
	}
}

func TestApiKeyAuth_InvalidFormatRejected(t *testing.T) {
	called := false
	handler := Auth(testSecret)(handlerCalled(&called))
	req := httptest.NewRequest(http.MethodGet, "/api/v1/assets/blind-scan", nil)
	req.Header.Set(ApiKeyHeader, "brand_test123abc")
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)
	if rr.Code != http.StatusUnauthorized || called {
		t.Fatalf("expected 401 and no call, got code=%d called=%v", rr.Code, called)
	}
}

func TestApiKeyAuth_UnsupportedRouteRejected(t *testing.T) {
	called := false
	handler := Auth(testSecret)(handlerCalled(&called))
	req := httptest.NewRequest(http.MethodGet, "/api/bff/unknown", nil)
	req.Header.Set(ApiKeyHeader, "rcpk_live_1234567890abcdef1234567890abcdef")
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)
	if rr.Code != http.StatusUnauthorized || called {
		t.Fatalf("expected 401 and no call, got code=%d called=%v", rr.Code, called)
	}
}

func TestApiKeyAuth_PrecedenceOverJWT(t *testing.T) {
	var called bool
	var receivedVerified string
	inner := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
		receivedVerified = r.Header.Get("X-Api-Key-Verified")
		w.WriteHeader(http.StatusOK)
	})
	handler := Auth(testSecret)(inner)

	claims := &Claims{Sub: "user-1", Role: "Brand", RegisteredClaims: jwt.RegisteredClaims{ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Hour))}}
	token := signTestToken(claims, testSecret)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/assets/blind-scan", nil)
	req.Header.Set(ApiKeyHeader, "rcpk_live_1234567890abcdef1234567890abcdef")
	req.Header.Set("Authorization", "Bearer "+token)
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK || !called || receivedVerified != "hash-only" {
		t.Fatalf("expected API key path precedence, got code=%d called=%v verified=%q", rr.Code, called, receivedVerified)
	}
}
