package repo

import (
	"context"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"rcprotocol/services/go-iam/internal/model"
)

// MemberRepository defines the contract for member binding persistence operations.
type MemberRepository interface {
	Bind(ctx context.Context, uop *model.UserOrgPosition) error
	Unbind(ctx context.Context, userID, orgID string) error
	ListByOrg(ctx context.Context, orgID string) ([]model.MemberView, error)
	GetUserOrgs(ctx context.Context, userID string) ([]model.UserOrgView, error)
	ExistsByUserAndOrg(ctx context.Context, userID, orgID string) (bool, error)
}

// MemberRepo implements MemberRepository with pgx.
type MemberRepo struct {
	pool *pgxpool.Pool
}

// NewMemberRepo creates a MemberRepo backed by the given connection pool.
func NewMemberRepo(pool *pgxpool.Pool) *MemberRepo {
	return &MemberRepo{pool: pool}
}

func (r *MemberRepo) Bind(ctx context.Context, uop *model.UserOrgPosition) error {
	uop.CreatedAt = time.Now()

	_, err := r.pool.Exec(ctx,
		`INSERT INTO user_org_positions (user_id, org_id, position_id, created_at)
		 VALUES ($1, $2, $3, $4)`,
		uop.UserID, uop.OrgID, uop.PositionID, uop.CreatedAt,
	)
	return err
}

func (r *MemberRepo) Unbind(ctx context.Context, userID, orgID string) error {
	ct, err := r.pool.Exec(ctx,
		`DELETE FROM user_org_positions WHERE user_id = $1 AND org_id = $2`,
		userID, orgID,
	)
	if err != nil {
		return err
	}
	if ct.RowsAffected() == 0 {
		return pgx.ErrNoRows
	}
	return nil
}

func (r *MemberRepo) ListByOrg(ctx context.Context, orgID string) ([]model.MemberView, error) {
	rows, err := r.pool.Query(ctx,
		`SELECT u.user_id, u.display_name, u.email, p.position_id, p.position_name, p.protocol_role
		 FROM user_org_positions uop
		 JOIN users u ON u.user_id = uop.user_id
		 JOIN positions p ON p.position_id = uop.position_id
		 WHERE uop.org_id = $1`, orgID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var members []model.MemberView
	for rows.Next() {
		var m model.MemberView
		if err := rows.Scan(&m.UserID, &m.DisplayName, &m.Email, &m.PositionID, &m.PositionName, &m.ProtocolRole); err != nil {
			return nil, err
		}
		members = append(members, m)
	}
	if members == nil {
		members = []model.MemberView{}
	}
	return members, rows.Err()
}

func (r *MemberRepo) GetUserOrgs(ctx context.Context, userID string) ([]model.UserOrgView, error) {
	rows, err := r.pool.Query(ctx,
		`SELECT o.org_id, o.org_name, o.org_type, p.protocol_role AS role, o.brand_id
		 FROM user_org_positions uop
		 JOIN organizations o ON o.org_id = uop.org_id
		 JOIN positions p ON p.position_id = uop.position_id
		 WHERE uop.user_id = $1`, userID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var orgs []model.UserOrgView
	for rows.Next() {
		var v model.UserOrgView
		if err := rows.Scan(&v.OrgID, &v.OrgName, &v.OrgType, &v.Role, &v.BrandID); err != nil {
			return nil, err
		}
		orgs = append(orgs, v)
	}
	if orgs == nil {
		orgs = []model.UserOrgView{}
	}
	return orgs, rows.Err()
}

func (r *MemberRepo) ExistsByUserAndOrg(ctx context.Context, userID, orgID string) (bool, error) {
	var exists bool
	err := r.pool.QueryRow(ctx,
		`SELECT COUNT(*) > 0 FROM user_org_positions WHERE user_id = $1 AND org_id = $2`,
		userID, orgID,
	).Scan(&exists)
	return exists, err
}
