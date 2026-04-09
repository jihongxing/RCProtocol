package repo

import (
	"context"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"rcprotocol/services/go-iam/internal/model"
)

// PositionRepository defines the contract for position persistence operations.
type PositionRepository interface {
	Create(ctx context.Context, pos *model.Position) error
	GetByID(ctx context.Context, positionID string) (*model.Position, error)
	ListByOrg(ctx context.Context, orgID string) ([]model.Position, error)
}

// PositionRepo implements PositionRepository with pgx.
type PositionRepo struct {
	pool *pgxpool.Pool
}

// NewPositionRepo creates a PositionRepo backed by the given connection pool.
func NewPositionRepo(pool *pgxpool.Pool) *PositionRepo {
	return &PositionRepo{pool: pool}
}

func (r *PositionRepo) Create(ctx context.Context, pos *model.Position) error {
	pos.CreatedAt = time.Now()

	_, err := r.pool.Exec(ctx,
		`INSERT INTO positions (position_id, org_id, position_name, protocol_role, created_at)
		 VALUES ($1, $2, $3, $4, $5)`,
		pos.PositionID, pos.OrgID, pos.PositionName, pos.ProtocolRole, pos.CreatedAt,
	)
	return err
}

func (r *PositionRepo) GetByID(ctx context.Context, positionID string) (*model.Position, error) {
	var p model.Position
	err := r.pool.QueryRow(ctx,
		`SELECT position_id, org_id, position_name, protocol_role, created_at
		 FROM positions WHERE position_id = $1`, positionID,
	).Scan(&p.PositionID, &p.OrgID, &p.PositionName, &p.ProtocolRole, &p.CreatedAt)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, pgx.ErrNoRows
		}
		return nil, err
	}
	return &p, nil
}

func (r *PositionRepo) ListByOrg(ctx context.Context, orgID string) ([]model.Position, error) {
	rows, err := r.pool.Query(ctx,
		`SELECT position_id, org_id, position_name, protocol_role, created_at
		 FROM positions WHERE org_id = $1 ORDER BY created_at ASC`, orgID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var positions []model.Position
	for rows.Next() {
		var p model.Position
		if err := rows.Scan(&p.PositionID, &p.OrgID, &p.PositionName, &p.ProtocolRole, &p.CreatedAt); err != nil {
			return nil, err
		}
		positions = append(positions, p)
	}
	if positions == nil {
		positions = []model.Position{}
	}
	return positions, rows.Err()
}
