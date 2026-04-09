package auth

import (
	"time"

	"github.com/golang-jwt/jwt/v5"
)

// Claims matches the JWT structure defined in Spec-03 for cross-service compatibility.
type Claims struct {
	Sub     string   `json:"sub"`
	Role    string   `json:"role"`
	OrgID   string   `json:"org_id"`
	BrandID string   `json:"brand_id,omitempty"`
	Scopes  []string `json:"scopes"`
	jwt.RegisteredClaims
}

// Issuer signs JWTs using HS256 + a shared secret.
type Issuer struct {
	secret      []byte
	expiryHours int
}

// NewIssuer creates an Issuer with the given secret and token lifetime.
func NewIssuer(secret string, expiryHours int) *Issuer {
	return &Issuer{
		secret:      []byte(secret),
		expiryHours: expiryHours,
	}
}

// IssueInput carries the data needed to build JWT claims.
type IssueInput struct {
	UserID  string
	Role    string
	OrgID   string
	BrandID string
}

// IssueResult is the signed token plus its unix expiry timestamp.
type IssueResult struct {
	Token     string `json:"token"`
	ExpiresAt int64  `json:"expires_at"`
}

// Issue builds Claims from input, signs with HS256, and returns the token string + expiry.
func (iss *Issuer) Issue(input IssueInput) (*IssueResult, error) {
	now := time.Now()
	expiresAt := now.Add(time.Duration(iss.expiryHours) * time.Hour)

	claims := Claims{
		Sub:     input.UserID,
		Role:    input.Role,
		OrgID:   input.OrgID,
		BrandID: input.BrandID,
		Scopes:  []string{},
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(expiresAt),
			IssuedAt:  jwt.NewNumericDate(now),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	signed, err := token.SignedString(iss.secret)
	if err != nil {
		return nil, err
	}

	return &IssueResult{
		Token:     signed,
		ExpiresAt: expiresAt.Unix(),
	}, nil
}
