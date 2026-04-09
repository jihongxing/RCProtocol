package repo

import (
	"context"

	"github.com/jackc/pgx/v5/pgxpool"
	"rcprotocol/services/go-iam/internal/model"
)

type ApiKeyRepo struct {
	pool *pgxpool.Pool
}

func NewApiKeyRepo(pool *pgxpool.Pool) *ApiKeyRepo {
	return &ApiKeyRepo{pool: pool}
}

// Create inserts a new API key.
func (r *ApiKeyRepo) Create(ctx context.Context, key *model.BrandApiKey) error {
	query := `
		INSERT INTO brand_api_keys (key_id, org_id, key_hash, description, status, created_at)
		VALUES ($1, $2, $3, $4, $5, $6)
	`
	_, err := r.pool.Exec(ctx, query,
		key.KeyID,
		key.OrgID,
		key.KeyHash,
		key.Description,
		key.Status,
		key.CreatedAt,
	)
	return err
}

// ListByOrg returns all API keys for an organization (excluding key_hash).
func (r *ApiKeyRepo) ListByOrg(ctx context.Context, orgID string) ([]model.BrandApiKey, error) {
	query := `
		SELECT key_id, org_id, description, status, created_at, last_used_at, revoked_at
		FROM brand_api_keys
		WHERE org_id = $1
		ORDER BY created_at DESC
	`
	rows, err := r.pool.Query(ctx, query, orgID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var keys []model.BrandApiKey
	for rows.Next() {
		var k model.BrandApiKey
		if err := rows.Scan(&k.KeyID, &k.OrgID, &k.Description, &k.Status, &k.CreatedAt, &k.LastUsedAt, &k.RevokedAt); err != nil {
			return nil, err
		}
		keys = append(keys, k)
	}
	return keys, rows.Err()
}

// GetActiveKeysByOrg returns active API keys for an organization (including key_hash for validation).
func (r *ApiKeyRepo) GetActiveKeysByOrg(ctx context.Context, orgID string) ([]model.BrandApiKey, error) {
	query := `
		SELECT key_id, org_id, key_hash, description, status, created_at, last_used_at, revoked_at
		FROM brand_api_keys
		WHERE org_id = $1 AND status = 'active'
		ORDER BY created_at DESC
	`
	rows, err := r.pool.Query(ctx, query, orgID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var keys []model.BrandApiKey
	for rows.Next() {
		var k model.BrandApiKey
		if err := rows.Scan(&k.KeyID, &k.OrgID, &k.KeyHash, &k.Description, &k.Status, &k.CreatedAt, &k.LastUsedAt, &k.RevokedAt); err != nil {
			return nil, err
		}
		keys = append(keys, k)
	}
	return keys, rows.Err()
}

// Revoke soft-deletes an API key by setting status to 'revoked' and revoked_at to NOW().
func (r *ApiKeyRepo) Revoke(ctx context.Context, keyID string) error {
	query := `
		UPDATE brand_api_keys
		SET status = 'revoked', revoked_at = NOW()
		WHERE key_id = $1
	`
	_, err := r.pool.Exec(ctx, query, keyID)
	return err
}

// UpdateLastUsed updates the last_used_at timestamp for an API key.
func (r *ApiKeyRepo) UpdateLastUsed(ctx context.Context, keyID string) error {
	query := `
		UPDATE brand_api_keys
		SET last_used_at = NOW()
		WHERE key_id = $1
	`
	_, err := r.pool.Exec(ctx, query, keyID)
	return err
}
