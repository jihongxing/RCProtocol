package model

import (
	"encoding/json"
	"time"
)

// ApprovalType 审批类型枚举
const (
	TypeBrandPublish   = "brand_publish"
	TypePolicyApply    = "policy_apply"
	TypeRiskRecovery   = "risk_recovery"
	TypeHighRiskAction = "high_risk_action"
)

// ApprovalStatus 审批状态枚举
const (
	StatusPending  = "pending"
	StatusApproved = "approved"
	StatusRejected = "rejected"
	StatusExecuted = "executed"
	StatusExpired  = "expired"
	StatusFailed   = "failed"
)

// ValidTypes 合法审批类型集合
var ValidTypes = map[string]bool{
	TypeBrandPublish:   true,
	TypePolicyApply:    true,
	TypeRiskRecovery:   true,
	TypeHighRiskAction: true,
}

// TerminalStatuses 终态集合（不可再变更）
var TerminalStatuses = map[string]bool{
	StatusExecuted: true,
	StatusRejected: true,
	StatusExpired:  true,
	StatusFailed:   true,
}

// ReviewerRoles 有审批权的角色
var ReviewerRoles = map[string]bool{
	"Platform":  true,
	"Moderator": true,
}

// Approval 审批单数据模型
type Approval struct {
	ID               string           `json:"id"`
	Type             string           `json:"type"`
	Status           string           `json:"status"`
	ApplicantID      string           `json:"applicant_id"`
	ApplicantRole    string           `json:"applicant_role"`
	ApplicantOrgID   string           `json:"applicant_org_id"`
	ReviewerID       *string          `json:"reviewer_id,omitempty"`
	ReviewerRole     *string          `json:"reviewer_role,omitempty"`
	Payload          json.RawMessage  `json:"payload"`
	Reason           *string          `json:"reason,omitempty"`
	ReviewComment    *string          `json:"review_comment,omitempty"`
	ResourceType     string           `json:"resource_type"`
	ResourceID       string           `json:"resource_id"`
	DownstreamResult *json.RawMessage `json:"downstream_result,omitempty"`
	ExpiresAt        time.Time        `json:"expires_at"`
	CreatedAt        time.Time        `json:"created_at"`
	UpdatedAt        time.Time        `json:"updated_at"`
}

// IsExpired 检查审批单是否已过期
func (a *Approval) IsExpired() bool {
	return time.Now().After(a.ExpiresAt)
}
