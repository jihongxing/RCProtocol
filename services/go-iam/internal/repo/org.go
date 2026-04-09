package repo

import (
	"context"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"rcprotocol/services/go-iam/internal/model"
)

// OrgRepository defines the contract for organization persistence operations.
type OrgRepository interface {
	Create(ctx context.Context, org *model.Organization) error
	GetByID(ctx context.Context, orgID string) (*model.Organization, error)
	List(ctx context.Context, orgType string, page, pageSize int) ([]model.Organization, int, error)
	Update(ctx context.Context, orgID, orgName, status string) (*model.Organization, error)
}

// OrgRepo implements OrgRepository with pgx.
type OrgRepo struct {
	pool *pgxpool.Pool
}

// NewOrgRepo creates an OrgRepo backed by the given connection pool.
func NewOrgRepo(pool *pgxpool.Pool) *OrgRepo {
	return &OrgRepo{pool: pool}
}

func (r *OrgRepo) Create(ctx context.Context, org *model.Organization) error {
	now := time.Now()
	org.CreatedAt = now
	org.UpdatedAt = now

	_, err := r.pool.Exec(ctx,
		`INSERT INTO organizations (org_id, org_name, org_type, parent_org_id, brand_id, contact_email, contact_phone, status, created_at, updated_at)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)`,
		org.OrgID, org.OrgName, org.OrgType, org.ParentOrgID, org.BrandID, org.ContactEmail, org.ContactPhone, org.Status, org.CreatedAt, org.UpdatedAt,
	)
	return err
}

func (r *OrgRepo) GetByID(ctx context.Context, orgID string) (*model.Organization, error) {
	var o model.Organization
	err := r.pool.QueryRow(ctx,
		`SELECT org_id, org_name, org_type, parent_org_id, brand_id, contact_email, contact_phone, status, created_at, updated_at
		 FROM organizations WHERE org_id = $1`, orgID,
	).Scan(&o.OrgID, &o.OrgName, &o.OrgType, &o.ParentOrgID, &o.BrandID, &o.ContactEmail, &o.ContactPhone, &o.Status, &o.CreatedAt, &o.UpdatedAt)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, pgx.ErrNoRows
		}
		return nil, err
	}
	return &o, nil
}

func (r *OrgRepo) List(ctx context.Context, orgType string, page, pageSize int) ([]model.Organization, int, error) {
	offset := (page - 1) * pageSize

	var total int
	if orgType != "" {
		err := r.pool.QueryRow(ctx, `SELECT COUNT(*) FROM organizations WHERE org_type = $1`, orgType).Scan(&total)
		if err != nil {
			return nil, 0, err
		}
	} else {
		err := r.pool.QueryRow(ctx, `SELECT COUNT(*) FROM organizations`).Scan(&total)
		if err != nil {
			return nil, 0, err
		}
	}

	var query string
	var rows pgx.Rows
	var err error

	if orgType != "" {
		query = `SELECT org_id, org_name, org_type, parent_org_id, brand_id, contact_email, contact_phone, status, created_at, updated_at
				 FROM organizations WHERE org_type = $1 ORDER BY created_at ASC LIMIT $2 OFFSET $3`
		rows, err = r.pool.Query(ctx, query, orgType, pageSize, offset)
	} else {
		query = `SELECT org_id, org_name, org_type, parent_org_id, brand_id, contact_email, contact_phone, status, created_at, updated_at
				 FROM organizations ORDER BY created_at ASC LIMIT $1 OFFSET $2`
		rows, err = r.pool.Query(ctx, query, pageSize, offset)
	}
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var orgs []model.Organization
	for rows.Next() {
		var o model.Organization
		if err := rows.Scan(&o.OrgID, &o.OrgName, &o.OrgType, &o.ParentOrgID, &o.BrandID, &o.ContactEmail, &o.ContactPhone, &o.Status, &o.CreatedAt, &o.UpdatedAt); err != nil {
			return nil, 0, err
		}
		orgs = append(orgs, o)
	}
	if orgs == nil {
		orgs = []model.Organization{}
	}
	return orgs, total, rows.Err()
}

func (r *OrgRepo) Update(ctx context.Context, orgID, orgName, status string) (*model.Organization, error) {
	now := time.Now()
	var o model.Organization
	err := r.pool.QueryRow(ctx,
		`UPDATE organizations SET org_name = $1, status = $2, updated_at = $3
		 WHERE org_id = $4
		 RETURNING org_id, org_name, org_type, parent_org_id, brand_id, contact_email, contact_phone, status, created_at, updated_at`,
		orgName, status, now, orgID,
	).Scan(&o.OrgID, &o.OrgName, &o.OrgType, &o.ParentOrgID, &o.BrandID, &o.ContactEmail, &o.ContactPhone, &o.Status, &o.CreatedAt, &o.UpdatedAt)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, pgx.ErrNoRows
		}
		return nil, err
	}
	return &o, nil
}
