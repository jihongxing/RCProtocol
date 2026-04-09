package auth

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"pgregory.net/rapid"
)

// makeTestRequest constructs an http.Request with X-Claims-* headers
func makeTestRequest(c Claims) *http.Request {
	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	req.Header.Set("X-Claims-Sub", c.Sub)
	req.Header.Set("X-Claims-Role", c.Role)
	if c.OrgID != "" {
		req.Header.Set("X-Claims-Org-Id", c.OrgID)
	}
	if c.BrandID != "" {
		req.Header.Set("X-Claims-Brand-Id", c.BrandID)
	}
	return req
}

// ---------------------------------------------------------------------------
// ParseClaims unit tests
// ---------------------------------------------------------------------------

func TestParseClaims_ValidHeaders(t *testing.T) {
	want := Claims{Sub: "user-1", Role: "Brand", OrgID: "org-1", BrandID: "brand-1"}
	req := makeTestRequest(want)

	got, errMsg := ParseClaims(req)
	if errMsg != "" {
		t.Fatalf("unexpected error: %s", errMsg)
	}
	if *got != want {
		t.Fatalf("claims mismatch: got %+v, want %+v", *got, want)
	}
}

func TestParseClaims_MissingSub(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	req.Header.Set("X-Claims-Role", "Platform")

	_, errMsg := ParseClaims(req)
	if errMsg == "" {
		t.Fatal("expected error for missing Sub")
	}
}

func TestParseClaims_MissingRole(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	req.Header.Set("X-Claims-Sub", "user-1")

	_, errMsg := ParseClaims(req)
	if errMsg == "" {
		t.Fatal("expected error for missing Role")
	}
}

func TestParseClaims_NoHeaders(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/test", nil)

	_, errMsg := ParseClaims(req)
	if errMsg == "" {
		t.Fatal("expected error for empty headers")
	}
}

// ---------------------------------------------------------------------------
// Middleware unit tests
// ---------------------------------------------------------------------------

func TestMiddleware_ValidHeaders(t *testing.T) {
	want := Claims{Sub: "u1", Role: "Platform", OrgID: "o1", BrandID: "b1"}

	var gotClaims *Claims
	inner := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotClaims = ClaimsFromContext(r.Context())
		w.WriteHeader(http.StatusOK)
	})

	req := makeTestRequest(want)
	req.URL.Path = "/app/assets"
	rr := httptest.NewRecorder()

	Middleware(inner).ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rr.Code)
	}
	if gotClaims == nil {
		t.Fatal("claims not injected into context")
	}
	if *gotClaims != want {
		t.Fatalf("claims mismatch: got %+v, want %+v", *gotClaims, want)
	}
}

func TestMiddleware_MissingHeaders(t *testing.T) {
	called := false
	inner := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
	})

	req := httptest.NewRequest(http.MethodGet, "/app/assets", nil)
	rr := httptest.NewRecorder()

	Middleware(inner).ServeHTTP(rr, req)

	if rr.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", rr.Code)
	}
	if called {
		t.Fatal("inner handler should not be called on auth failure")
	}

	body, _ := io.ReadAll(rr.Body)
	var resp struct {
		Error struct {
			Code    string `json:"code"`
			Message string `json:"message"`
		} `json:"error"`
	}
	if err := json.Unmarshal(body, &resp); err != nil {
		t.Fatalf("invalid JSON response: %v", err)
	}
	if resp.Error.Code != "AUTH_REQUIRED" {
		t.Fatalf("expected code AUTH_REQUIRED, got %q", resp.Error.Code)
	}
}

func TestMiddleware_HealthzSkipsAuth(t *testing.T) {
	called := false
	inner := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
		w.WriteHeader(http.StatusOK)
	})

	req := httptest.NewRequest(http.MethodGet, "/healthz", nil)
	rr := httptest.NewRecorder()

	Middleware(inner).ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rr.Code)
	}
	if !called {
		t.Fatal("/healthz should call inner handler without auth")
	}
}

// ---------------------------------------------------------------------------
// ClaimsFromContext edge case
// ---------------------------------------------------------------------------

func TestClaimsFromContext_NoClaims(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	got := ClaimsFromContext(req.Context())
	if got != nil {
		t.Fatalf("expected nil, got %+v", got)
	}
}

// ---------------------------------------------------------------------------
// Property-based tests using pgregory.net/rapid
// ---------------------------------------------------------------------------

func nonEmptyString() *rapid.Generator[string] {
	return rapid.StringMatching(`[a-zA-Z0-9_-]{1,64}`)
}

func validRole() *rapid.Generator[string] {
	return rapid.SampledFrom([]string{"Platform", "Brand", "Factory", "Consumer", "Moderator"})
}

// TestHeaderClaimsParseRoundTrip verifies that for any valid Claims,
// constructing X-Claims-* headers and parsing back yields identical values.
func TestHeaderClaimsParseRoundTrip(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		original := Claims{
			Sub:     nonEmptyString().Draw(t, "sub"),
			Role:    validRole().Draw(t, "role"),
			OrgID:   nonEmptyString().Draw(t, "org_id"),
			BrandID: nonEmptyString().Draw(t, "brand_id"),
		}

		req := makeTestRequest(original)
		parsed, errMsg := ParseClaims(req)

		if errMsg != "" {
			t.Fatalf("ParseClaims returned error for valid headers: %s", errMsg)
		}
		if parsed == nil {
			t.Fatal("ParseClaims returned nil Claims for valid headers")
		}
		if *parsed != original {
			t.Fatalf("round-trip mismatch:\n  original: %+v\n  parsed:   %+v", original, *parsed)
		}
	})
}

// TestMissingHeadersRejection verifies that requests without both
// X-Claims-Sub and X-Claims-Role are rejected.
func TestMissingHeadersRejection(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		kind := rapid.IntRange(0, 2).Draw(t, "kind")
		req := httptest.NewRequest(http.MethodGet, "/test", nil)

		switch kind {
		case 0:
			// No headers at all
		case 1:
			// Only Sub, no Role
			req.Header.Set("X-Claims-Sub", nonEmptyString().Draw(t, "sub"))
		case 2:
			// Only Role, no Sub
			req.Header.Set("X-Claims-Role", nonEmptyString().Draw(t, "role"))
		}

		_, errMsg := ParseClaims(req)
		if errMsg == "" {
			t.Fatal("ParseClaims should reject request missing Sub or Role")
		}
	})
}

// TestInvalidRoleRejection verifies that roles outside the whitelist are rejected.
func TestInvalidRoleRejection(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		role := nonEmptyString().Draw(t, "role")
		// Skip if randomly generated role happens to be valid
		switch role {
		case "Platform", "Brand", "Factory", "Consumer", "Moderator":
			return
		}

		req := httptest.NewRequest(http.MethodGet, "/test", nil)
		req.Header.Set("X-Claims-Sub", "user-1")
		req.Header.Set("X-Claims-Role", role)

		_, errMsg := ParseClaims(req)
		if errMsg == "" {
			t.Fatalf("ParseClaims should reject invalid role %q", role)
		}
	})
}
