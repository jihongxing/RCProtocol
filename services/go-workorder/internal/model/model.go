package model

import (
	"encoding/json"
	"time"
)

// WorkorderType 工单类型
const (
	TypeRisk     = "risk"
	TypeDispute  = "dispute"
	TypeRecovery = "recovery"
)

// WorkorderStatus 工单状态
const (
	StatusOpen       = "open"
	StatusAssigned   = "assigned"
	StatusInProgress = "in_progress"
	StatusResolved   = "resolved"
	StatusClosed     = "closed"
	StatusCancelled  = "cancelled"
)

// ConclusionType 结论类型
const (
	ConclusionFreeze          = "freeze"
	ConclusionRecover         = "recover"
	ConclusionMarkTampered    = "mark_tampered"
	ConclusionMarkCompromised = "mark_compromised"
	ConclusionDismiss         = "dismiss"
)

// ValidTypes 合法工单类型集合
var ValidTypes = map[string]bool{
	TypeRisk: true, TypeDispute: true, TypeRecovery: true,
}

// ValidConclusionTypes 合法结论类型集合
var ValidConclusionTypes = map[string]bool{
	ConclusionFreeze: true, ConclusionRecover: true,
	ConclusionMarkTampered: true, ConclusionMarkCompromised: true,
	ConclusionDismiss: true,
}

// TerminalStatuses 终态集合（不可再变更）
var TerminalStatuses = map[string]bool{
	StatusClosed: true, StatusCancelled: true,
}

// AdvancableStatuses 允许 advance 的状态
var AdvancableStatuses = map[string]bool{
	StatusAssigned: true, StatusInProgress: true,
}

// CancellableStatuses 允许 cancel 的状态
var CancellableStatuses = map[string]bool{
	StatusOpen: true, StatusAssigned: true,
}

// AssignableStatuses 允许 assign 的状态
var AssignableStatuses = map[string]bool{
	StatusOpen: true, StatusAssigned: true,
}

// ManagerRoles 有管理权限的角色
var ManagerRoles = map[string]bool{
	"Platform": true, "Moderator": true,
}

// Workorder 工单数据模型
type Workorder struct {
	ID               string           `json:"id"`
	Type             string           `json:"type"`
	Status           string           `json:"status"`
	Title            string           `json:"title"`
	Description      *string          `json:"description,omitempty"`
	CreatorID        string           `json:"creator_id"`
	CreatorRole      string           `json:"creator_role"`
	CreatorOrgID     string           `json:"creator_org_id"`
	AssigneeID       *string          `json:"assignee_id,omitempty"`
	AssigneeRole     *string          `json:"assignee_role,omitempty"`
	AssetID          *string          `json:"asset_id,omitempty"`
	BrandID          *string          `json:"brand_id,omitempty"`
	Conclusion       *string          `json:"conclusion,omitempty"`
	ConclusionType   *string          `json:"conclusion_type,omitempty"`
	ApprovalID       *string          `json:"approval_id,omitempty"`
	DownstreamResult *json.RawMessage `json:"downstream_result,omitempty"`
	Metadata         *json.RawMessage `json:"metadata,omitempty"`
	CreatedAt        time.Time        `json:"created_at"`
	UpdatedAt        time.Time        `json:"updated_at"`
}
