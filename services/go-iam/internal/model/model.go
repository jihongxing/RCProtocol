package model

import "time"

type User struct {
	UserID       string    `json:"user_id"`
	Email        string    `json:"email"`
	PasswordHash string    `json:"-"`
	DisplayName  string    `json:"display_name"`
	Status       string    `json:"status"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

type Organization struct {
	OrgID        string    `json:"org_id"`
	OrgName      string    `json:"org_name"`
	OrgType      string    `json:"org_type"`
	ParentOrgID  *string   `json:"parent_org_id,omitempty"`
	BrandID      *string   `json:"brand_id,omitempty"`
	ContactEmail *string   `json:"contact_email,omitempty"`
	ContactPhone *string   `json:"contact_phone,omitempty"`
	Status       string    `json:"status"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

type Position struct {
	PositionID   string    `json:"position_id"`
	OrgID        string    `json:"org_id"`
	PositionName string    `json:"position_name"`
	ProtocolRole string    `json:"protocol_role"`
	CreatedAt    time.Time `json:"created_at"`
}

type UserOrgPosition struct {
	UserID     string    `json:"user_id"`
	OrgID      string    `json:"org_id"`
	PositionID string    `json:"position_id"`
	CreatedAt  time.Time `json:"created_at"`
}

// MemberView is a JOIN view for listing org members with position details.
type MemberView struct {
	UserID       string `json:"user_id"`
	DisplayName  string `json:"display_name"`
	Email        string `json:"email"`
	PositionID   string `json:"position_id"`
	PositionName string `json:"position_name"`
	ProtocolRole string `json:"protocol_role"`
}

// UserOrgView is a JOIN view for user's org memberships.
// Includes BrandID so Login handler can get brand_id directly without extra orgRepo query.
type UserOrgView struct {
	OrgID   string  `json:"org_id"`
	OrgName string  `json:"org_name"`
	OrgType string  `json:"org_type"`
	Role    string  `json:"role"`
	BrandID *string `json:"brand_id,omitempty"`
}

// BrandApiKey represents a Platform-managed legacy/backoffice brand integration key.
// It is org-scoped in go-iam and retained for administrative / compatibility use.
// Current Gateway runtime brand API key truth does not validate against this model directly.
type BrandApiKey struct {
	KeyID      string     `json:"key_id"`
	OrgID      string     `json:"org_id"`
	KeyHash    string     `json:"-"`
	Description *string   `json:"description,omitempty"`
	Status     string     `json:"status"`
	CreatedAt  time.Time  `json:"created_at"`
	LastUsedAt *time.Time `json:"last_used_at,omitempty"`
	RevokedAt  *time.Time `json:"revoked_at,omitempty"`
}

// ApiKeyCreateResponse is returned when creating a legacy/backoffice org-scoped API key
// (plaintext only shown once).
type ApiKeyCreateResponse struct {
	KeyID     string    `json:"key_id"`
	ApiKey    string    `json:"api_key"`
	CreatedAt time.Time `json:"created_at"`
}

// ApiKeyValidateResponse is returned when validating a legacy/backoffice API key.
type ApiKeyValidateResponse struct {
	OrgID   string  `json:"org_id"`
	BrandID *string `json:"brand_id,omitempty"`
	OrgName string  `json:"org_name"`
}
