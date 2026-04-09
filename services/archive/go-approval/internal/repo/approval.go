package repo

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"rcprotocol/services/go-approval/internal/model"
)

// ApprovalRepo 审批单数据仓储，直接操作 PostgreSQL
type ApprovalRepo struct {
	pool *pgxpool.Pool
}

// NewApprovalRepo 创建 ApprovalRepo 实例
func NewApprovalRepo(pool *pgxpool.Pool) *ApprovalRepo {
	return &ApprovalRepo{pool: pool}
}

// Create 写入审批单
func (r *ApprovalRepo) Create(ctx context.Context, a *model.Approval) error {
	_, err := r.pool.Exec(ctx, `
		INSERT INTO approvals (id, type, status, applicant_id, applicant_role, applicant_org_id,
			payload, reason, resource_type, resource_id, expires_at, created_at, updated_at)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,NOW(),NOW())`,
		a.ID, a.Type, a.Status, a.ApplicantID, a.ApplicantRole, a.ApplicantOrgID,
		a.Payload, a.Reason, a.ResourceType, a.ResourceID, a.ExpiresAt)
	return err
}

// GetByID 按 ID 查询审批单，不存在返回 nil
func (r *ApprovalRepo) GetByID(ctx context.Context, id string) (*model.Approval, error) {
	a := &model.Approval{}
	err := r.pool.QueryRow(ctx, `
		SELECT id, type, status, applicant_id, applicant_role, applicant_org_id,
			reviewer_id, reviewer_role, payload, reason, review_comment,
			resource_type, resource_id, downstream_result, expires_at, created_at, updated_at
		FROM approvals WHERE id = $1`, id).Scan(
		&a.ID, &a.Type, &a.Status, &a.ApplicantID, &a.ApplicantRole, &a.ApplicantOrgID,
		&a.ReviewerID, &a.ReviewerRole, &a.Payload, &a.Reason, &a.ReviewComment,
		&a.ResourceType, &a.ResourceID, &a.DownstreamResult, &a.ExpiresAt, &a.CreatedAt, &a.UpdatedAt)
	if err == pgx.ErrNoRows {
		return nil, nil
	}
	return a, err
}

// ExistsPending 检查同一资源同一审批类型是否已有 pending 审批单
func (r *ApprovalRepo) ExistsPending(ctx context.Context, resourceType, resourceID, approvalType string) (bool, error) {
	var exists bool
	err := r.pool.QueryRow(ctx, `
		SELECT EXISTS(
			SELECT 1 FROM approvals
			WHERE resource_type=$1 AND resource_id=$2 AND type=$3 AND status='pending'
		)`, resourceType, resourceID, approvalType).Scan(&exists)
	return exists, err
}

// List 分页查询审批单列表，支持 status/type/orgID 动态筛选
func (r *ApprovalRepo) List(ctx context.Context, filterStatus, filterType, orgID string, page, pageSize int) ([]model.Approval, int, error) {
	where := "WHERE 1=1"
	args := []interface{}{}
	argIdx := 1

	if filterStatus != "" {
		where += fmt.Sprintf(" AND status=$%d", argIdx)
		args = append(args, filterStatus)
		argIdx++
	}
	if filterType != "" {
		where += fmt.Sprintf(" AND type=$%d", argIdx)
		args = append(args, filterType)
		argIdx++
	}
	if orgID != "" {
		where += fmt.Sprintf(" AND applicant_org_id=$%d", argIdx)
		args = append(args, orgID)
		argIdx++
	}

	var total int
	countArgs := make([]interface{}, len(args))
	copy(countArgs, args)
	err := r.pool.QueryRow(ctx, "SELECT COUNT(*) FROM approvals "+where, countArgs...).Scan(&total)
	if err != nil {
		return nil, 0, err
	}

	offset := (page - 1) * pageSize
	query := fmt.Sprintf(`
		SELECT id, type, status, applicant_id, applicant_role, applicant_org_id,
			reviewer_id, reviewer_role, payload, reason, review_comment,
			resource_type, resource_id, downstream_result, expires_at, created_at, updated_at
		FROM approvals %s
		ORDER BY created_at DESC
		LIMIT $%d OFFSET $%d`, where, argIdx, argIdx+1)
	args = append(args, pageSize, offset)

	rows, err := r.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var items []model.Approval
	for rows.Next() {
		var a model.Approval
		if err := rows.Scan(
			&a.ID, &a.Type, &a.Status, &a.ApplicantID, &a.ApplicantRole, &a.ApplicantOrgID,
			&a.ReviewerID, &a.ReviewerRole, &a.Payload, &a.Reason, &a.ReviewComment,
			&a.ResourceType, &a.ResourceID, &a.DownstreamResult, &a.ExpiresAt, &a.CreatedAt, &a.UpdatedAt,
		); err != nil {
			return nil, 0, err
		}
		items = append(items, a)
	}

	return items, total, nil
}

// ListByResource 按资源查询审批单
func (r *ApprovalRepo) ListByResource(ctx context.Context, resourceType, resourceID string) ([]model.Approval, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT id, type, status, applicant_id, applicant_role, applicant_org_id,
			reviewer_id, reviewer_role, payload, reason, review_comment,
			resource_type, resource_id, downstream_result, expires_at, created_at, updated_at
		FROM approvals
		WHERE resource_type=$1 AND resource_id=$2
		ORDER BY created_at DESC`, resourceType, resourceID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var items []model.Approval
	for rows.Next() {
		var a model.Approval
		if err := rows.Scan(
			&a.ID, &a.Type, &a.Status, &a.ApplicantID, &a.ApplicantRole, &a.ApplicantOrgID,
			&a.ReviewerID, &a.ReviewerRole, &a.Payload, &a.Reason, &a.ReviewComment,
			&a.ResourceType, &a.ResourceID, &a.DownstreamResult, &a.ExpiresAt, &a.CreatedAt, &a.UpdatedAt,
		); err != nil {
			return nil, err
		}
		items = append(items, a)
	}

	return items, nil
}

// UpdateStatus 使用 CAS 更新审批单状态，防止并发冲突
// 返回 true 表示更新成功（恰好 1 行受影响），false 表示状态已被其他操作改变
func (r *ApprovalRepo) UpdateStatus(ctx context.Context, id, expectedStatus, newStatus string,
	reviewerID, reviewerRole, reviewComment *string, downstreamResult *[]byte) (bool, error) {

	tag, err := r.pool.Exec(ctx, `
		UPDATE approvals SET
			status = $1,
			reviewer_id = COALESCE($2, reviewer_id),
			reviewer_role = COALESCE($3, reviewer_role),
			review_comment = COALESCE($4, review_comment),
			downstream_result = COALESCE($5, downstream_result),
			updated_at = NOW()
		WHERE id = $6 AND status = $7
			AND (expires_at IS NULL OR expires_at > NOW())`,
		newStatus, reviewerID, reviewerRole, reviewComment, downstreamResult, id, expectedStatus)
	if err != nil {
		return false, err
	}

	return tag.RowsAffected() == 1, nil
}
