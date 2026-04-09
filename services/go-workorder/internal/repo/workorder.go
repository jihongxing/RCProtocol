package repo

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"rcprotocol/services/go-workorder/internal/model"
)

// WorkorderRepo 工单数据仓储，直接操作 PostgreSQL
type WorkorderRepo struct {
	pool *pgxpool.Pool
}

// NewWorkorderRepo 创建 WorkorderRepo 实例
func NewWorkorderRepo(pool *pgxpool.Pool) *WorkorderRepo {
	return &WorkorderRepo{pool: pool}
}

// Create 写入工单
func (r *WorkorderRepo) Create(ctx context.Context, w *model.Workorder) error {
	_, err := r.pool.Exec(ctx, `
		INSERT INTO workorders (id, type, status, title, description, creator_id, creator_role, creator_org_id,
			assignee_id, assignee_role, asset_id, brand_id, conclusion, conclusion_type,
			approval_id, downstream_result, metadata, created_at, updated_at)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15,$16,$17,NOW(),NOW())`,
		w.ID, w.Type, w.Status, w.Title, w.Description, w.CreatorID, w.CreatorRole, w.CreatorOrgID,
		w.AssigneeID, w.AssigneeRole, w.AssetID, w.BrandID, w.Conclusion, w.ConclusionType,
		w.ApprovalID, w.DownstreamResult, w.Metadata)
	return err
}

// GetByID 按 ID 查询工单，不存在返回 nil
func (r *WorkorderRepo) GetByID(ctx context.Context, id string) (*model.Workorder, error) {
	w := &model.Workorder{}
	err := r.pool.QueryRow(ctx, `
		SELECT id, type, status, title, description, creator_id, creator_role, creator_org_id,
			assignee_id, assignee_role, asset_id, brand_id, conclusion, conclusion_type,
			approval_id, downstream_result, metadata, created_at, updated_at
		FROM workorders WHERE id = $1`, id).Scan(
		&w.ID, &w.Type, &w.Status, &w.Title, &w.Description, &w.CreatorID, &w.CreatorRole, &w.CreatorOrgID,
		&w.AssigneeID, &w.AssigneeRole, &w.AssetID, &w.BrandID, &w.Conclusion, &w.ConclusionType,
		&w.ApprovalID, &w.DownstreamResult, &w.Metadata, &w.CreatedAt, &w.UpdatedAt)
	if err == pgx.ErrNoRows {
		return nil, nil
	}
	return w, err
}

// List 分页查询工单列表，支持 status/type/assigneeID/orgID/brandID 动态筛选
func (r *WorkorderRepo) List(ctx context.Context, filterStatus, filterType, assigneeID, orgID, brandID string, page, pageSize int) ([]model.Workorder, int, error) {
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
	if assigneeID != "" {
		where += fmt.Sprintf(" AND assignee_id=$%d", argIdx)
		args = append(args, assigneeID)
		argIdx++
	}
	if orgID != "" {
		where += fmt.Sprintf(" AND (creator_org_id=$%d OR brand_id=$%d)", argIdx, argIdx+1)
		args = append(args, orgID, brandID)
		argIdx += 2
	}

	var total int
	countArgs := make([]interface{}, len(args))
	copy(countArgs, args)
	err := r.pool.QueryRow(ctx, "SELECT COUNT(*) FROM workorders "+where, countArgs...).Scan(&total)
	if err != nil {
		return nil, 0, err
	}

	offset := (page - 1) * pageSize
	query := fmt.Sprintf(`
		SELECT id, type, status, title, description, creator_id, creator_role, creator_org_id,
			assignee_id, assignee_role, asset_id, brand_id, conclusion, conclusion_type,
			approval_id, downstream_result, metadata, created_at, updated_at
		FROM workorders %s
		ORDER BY created_at DESC
		LIMIT $%d OFFSET $%d`, where, argIdx, argIdx+1)
	args = append(args, pageSize, offset)

	rows, err := r.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var items []model.Workorder
	for rows.Next() {
		var w model.Workorder
		if err := rows.Scan(
			&w.ID, &w.Type, &w.Status, &w.Title, &w.Description, &w.CreatorID, &w.CreatorRole, &w.CreatorOrgID,
			&w.AssigneeID, &w.AssigneeRole, &w.AssetID, &w.BrandID, &w.Conclusion, &w.ConclusionType,
			&w.ApprovalID, &w.DownstreamResult, &w.Metadata, &w.CreatedAt, &w.UpdatedAt,
		); err != nil {
			return nil, 0, err
		}
		items = append(items, w)
	}

	return items, total, nil
}

// ListByAsset 按资产 ID 查询关联工单
func (r *WorkorderRepo) ListByAsset(ctx context.Context, assetID string) ([]model.Workorder, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT id, type, status, title, description, creator_id, creator_role, creator_org_id,
			assignee_id, assignee_role, asset_id, brand_id, conclusion, conclusion_type,
			approval_id, downstream_result, metadata, created_at, updated_at
		FROM workorders
		WHERE asset_id=$1
		ORDER BY created_at DESC`, assetID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var items []model.Workorder
	for rows.Next() {
		var w model.Workorder
		if err := rows.Scan(
			&w.ID, &w.Type, &w.Status, &w.Title, &w.Description, &w.CreatorID, &w.CreatorRole, &w.CreatorOrgID,
			&w.AssigneeID, &w.AssigneeRole, &w.AssetID, &w.BrandID, &w.Conclusion, &w.ConclusionType,
			&w.ApprovalID, &w.DownstreamResult, &w.Metadata, &w.CreatedAt, &w.UpdatedAt,
		); err != nil {
			return nil, err
		}
		items = append(items, w)
	}

	return items, nil
}

// UpdateStatus 使用 CAS 更新工单状态，防止并发冲突
// 返回 true 表示更新成功（恰好 1 行受影响），false 表示状态已被其他操作改变
func (r *WorkorderRepo) UpdateStatus(ctx context.Context, id, expectedStatus, newStatus string) (bool, error) {
	tag, err := r.pool.Exec(ctx, `
		UPDATE workorders SET status=$1, updated_at=NOW()
		WHERE id=$2 AND status=$3`,
		newStatus, id, expectedStatus)
	if err != nil {
		return false, err
	}
	return tag.RowsAffected() == 1, nil
}

// Assign 分派工单，使用 CAS 模式限制可分派状态
func (r *WorkorderRepo) Assign(ctx context.Context, id string, expectedStatuses []string, assigneeID, assigneeRole string) (bool, error) {
	tag, err := r.pool.Exec(ctx, `
		UPDATE workorders SET assignee_id=$1, assignee_role=$2, status='assigned', updated_at=NOW()
		WHERE id=$3 AND status = ANY($4::text[])`,
		assigneeID, assigneeRole, id, expectedStatuses)
	if err != nil {
		return false, err
	}
	return tag.RowsAffected() == 1, nil
}

// Advance 推进工单，记录结论和下游执行结果
// newStatus 由 handler 根据下游执行结果决定是 resolved 还是 in_progress
func (r *WorkorderRepo) Advance(ctx context.Context, id string, newStatus, conclusion, conclusionType string, approvalID *string, downstreamResult *[]byte) (bool, error) {
	tag, err := r.pool.Exec(ctx, `
		UPDATE workorders SET status=$1, conclusion=$2, conclusion_type=$3,
			approval_id=COALESCE($4, approval_id), downstream_result=COALESCE($5, downstream_result),
			updated_at=NOW()
		WHERE id=$6 AND status IN ('assigned','in_progress')`,
		newStatus, conclusion, conclusionType, approvalID, downstreamResult, id)
	if err != nil {
		return false, err
	}
	return tag.RowsAffected() == 1, nil
}
