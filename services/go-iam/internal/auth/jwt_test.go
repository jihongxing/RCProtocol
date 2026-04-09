package auth

import (
	"testing"

	"github.com/golang-jwt/jwt/v5"
)

const testSecret = "test-secret-key-for-jwt-signing"

// decodeTestToken parses and validates a JWT string using the given secret.
func decodeTestToken(tokenStr, secret string) (*Claims, error) {
	token, err := jwt.ParseWithClaims(tokenStr, &Claims{}, func(t *jwt.Token) (interface{}, error) {
		return []byte(secret), nil
	})
	if err != nil {
		return nil, err
	}
	claims, ok := token.Claims.(*Claims)
	if !ok {
		return nil, jwt.ErrTokenInvalidClaims
	}
	return claims, nil
}

func TestJWTIssueClaimsCorrect(t *testing.T) {
	issuer := NewIssuer(testSecret, 24)
	result, err := issuer.Issue(IssueInput{
		UserID:  "user-123",
		Role:    "Brand",
		OrgID:   "org-456",
		BrandID: "brand-789",
	})
	if err != nil {
		t.Fatalf("Issue() error: %v", err)
	}

	claims, err := decodeTestToken(result.Token, testSecret)
	if err != nil {
		t.Fatalf("decodeTestToken error: %v", err)
	}

	if claims.Sub != "user-123" {
		t.Errorf("expected sub=user-123, got %s", claims.Sub)
	}
	if claims.Role != "Brand" {
		t.Errorf("expected role=Brand, got %s", claims.Role)
	}
	if claims.OrgID != "org-456" {
		t.Errorf("expected org_id=org-456, got %s", claims.OrgID)
	}
	if claims.BrandID != "brand-789" {
		t.Errorf("expected brand_id=brand-789, got %s", claims.BrandID)
	}
	if claims.Scopes == nil || len(claims.Scopes) != 0 {
		t.Errorf("expected scopes=[], got %v", claims.Scopes)
	}
}

func TestJWTExpiryMatchesConfig(t *testing.T) {
	expiryHours := 12
	issuer := NewIssuer(testSecret, expiryHours)
	result, err := issuer.Issue(IssueInput{
		UserID: "user-1",
		Role:   "Platform",
		OrgID:  "org-1",
	})
	if err != nil {
		t.Fatalf("Issue() error: %v", err)
	}

	claims, err := decodeTestToken(result.Token, testSecret)
	if err != nil {
		t.Fatalf("decodeTestToken error: %v", err)
	}

	iat := claims.IssuedAt.Unix()
	exp := claims.ExpiresAt.Unix()
	diff := exp - iat
	expected := int64(expiryHours * 3600)

	if diff != expected {
		t.Errorf("expected exp - iat = %d, got %d", expected, diff)
	}

	if result.ExpiresAt != exp {
		t.Errorf("expected IssueResult.ExpiresAt=%d to match claims exp=%d", result.ExpiresAt, exp)
	}
}

func TestJWTHS256VerifySameSecret(t *testing.T) {
	issuer := NewIssuer(testSecret, 1)
	result, err := issuer.Issue(IssueInput{
		UserID: "user-1",
		Role:   "Factory",
		OrgID:  "org-1",
	})
	if err != nil {
		t.Fatalf("Issue() error: %v", err)
	}

	_, err = decodeTestToken(result.Token, testSecret)
	if err != nil {
		t.Errorf("expected token to verify with same secret, got error: %v", err)
	}
}

func TestJWTDifferentSecretFails(t *testing.T) {
	issuer := NewIssuer(testSecret, 1)
	result, err := issuer.Issue(IssueInput{
		UserID: "user-1",
		Role:   "Platform",
		OrgID:  "org-1",
	})
	if err != nil {
		t.Fatalf("Issue() error: %v", err)
	}

	_, err = decodeTestToken(result.Token, "wrong-secret")
	if err == nil {
		t.Error("expected error when decoding with different secret, got nil")
	}
}
