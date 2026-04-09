package repo

import (
	"context"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"rcprotocol/services/go-iam/internal/model"
)

// UserRepository defines the contract for user persistence operations.
type UserRepository interface {
	Create(ctx context.Context, user *model.User) error
	GetByID(ctx context.Context, userID string) (*model.User, error)
	GetByEmail(ctx context.Context, email string) (*model.User, error)
	List(ctx context.Context, page, pageSize int) ([]model.User, int, error)
	Update(ctx context.Context, userID, displayName, status string) (*model.User, error)
	UpdatePasswordHash(ctx context.Context, userID, newHash string) error
	Disable(ctx context.Context, userID string) error
}

// UserRepo implements UserRepository with pgx.
type UserRepo struct {
	pool *pgxpool.Pool
}

// NewUserRepo creates a UserRepo backed by the given connection pool.
func NewUserRepo(pool *pgxpool.Pool) *UserRepo {
	return &UserRepo{pool: pool}
}

func (r *UserRepo) Create(ctx context.Context, user *model.User) error {
	now := time.Now()
	user.CreatedAt = now
	user.UpdatedAt = now

	_, err := r.pool.Exec(ctx,
		`INSERT INTO users (user_id, email, password_hash, display_name, status, created_at, updated_at)
		 VALUES ($1, $2, $3, $4, $5, $6, $7)`,
		user.UserID, user.Email, user.PasswordHash, user.DisplayName, user.Status, user.CreatedAt, user.UpdatedAt,
	)
	return err
}

func (r *UserRepo) GetByID(ctx context.Context, userID string) (*model.User, error) {
	var u model.User
	err := r.pool.QueryRow(ctx,
		`SELECT user_id, email, display_name, status, created_at, updated_at
		 FROM users WHERE user_id = $1`, userID,
	).Scan(&u.UserID, &u.Email, &u.DisplayName, &u.Status, &u.CreatedAt, &u.UpdatedAt)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, pgx.ErrNoRows
		}
		return nil, err
	}
	return &u, nil
}

func (r *UserRepo) GetByEmail(ctx context.Context, email string) (*model.User, error) {
	var u model.User
	err := r.pool.QueryRow(ctx,
		`SELECT user_id, email, password_hash, display_name, status, created_at, updated_at
		 FROM users WHERE email = $1`, email,
	).Scan(&u.UserID, &u.Email, &u.PasswordHash, &u.DisplayName, &u.Status, &u.CreatedAt, &u.UpdatedAt)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, pgx.ErrNoRows
		}
		return nil, err
	}
	return &u, nil
}

func (r *UserRepo) List(ctx context.Context, page, pageSize int) ([]model.User, int, error) {
	offset := (page - 1) * pageSize

	var total int
	if err := r.pool.QueryRow(ctx, `SELECT COUNT(*) FROM users`).Scan(&total); err != nil {
		return nil, 0, err
	}

	rows, err := r.pool.Query(ctx,
		`SELECT user_id, email, display_name, status, created_at, updated_at
		 FROM users ORDER BY created_at ASC LIMIT $1 OFFSET $2`, pageSize, offset,
	)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var users []model.User
	for rows.Next() {
		var u model.User
		if err := rows.Scan(&u.UserID, &u.Email, &u.DisplayName, &u.Status, &u.CreatedAt, &u.UpdatedAt); err != nil {
			return nil, 0, err
		}
		users = append(users, u)
	}
	if users == nil {
		users = []model.User{}
	}
	return users, total, rows.Err()
}

func (r *UserRepo) Update(ctx context.Context, userID, displayName, status string) (*model.User, error) {
	now := time.Now()
	var u model.User
	err := r.pool.QueryRow(ctx,
		`UPDATE users SET display_name = $1, status = $2, updated_at = $3
		 WHERE user_id = $4
		 RETURNING user_id, email, display_name, status, created_at, updated_at`,
		displayName, status, now, userID,
	).Scan(&u.UserID, &u.Email, &u.DisplayName, &u.Status, &u.CreatedAt, &u.UpdatedAt)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, pgx.ErrNoRows
		}
		return nil, err
	}
	return &u, nil
}

func (r *UserRepo) Disable(ctx context.Context, userID string) error {
	now := time.Now()
	ct, err := r.pool.Exec(ctx,
		`UPDATE users SET status = 'disabled', updated_at = $1 WHERE user_id = $2`,
		now, userID,
	)
	if err != nil {
		return err
	}
	if ct.RowsAffected() == 0 {
		return pgx.ErrNoRows
	}
	return nil
}

func (r *UserRepo) UpdatePasswordHash(ctx context.Context, userID, newHash string) error {
	now := time.Now()
	_, err := r.pool.Exec(ctx,
		`UPDATE users SET password_hash = $1, updated_at = $2 WHERE user_id = $3`,
		newHash, now, userID,
	)
	return err
}
