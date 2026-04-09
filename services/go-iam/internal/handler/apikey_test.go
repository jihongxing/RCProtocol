package handler

import (
	"context"
	"time"

	"github.com/jackc/pgx/v5"
	"golang.org/x/crypto/bcrypt"

	"rcprotocol/services/go-iam/internal/model"
)

// mockApiKeyRepo implements repo.ApiKeyRepository for handler tests.
type mockApiKeyRepo struct {
	keys map[string]*model.BrandApiKey
}

func newMockApiKeyRepo() *mockApiKeyRepo {
	return &mockApiKeyRepo{keys: make(map[string]*model.BrandApiKey)}
}

func (m *mockApiKeyRepo) Create(_ context.Context, key *model.BrandApiKey) error {
	now := time.Now()
	key.CreatedAt = now
	m.keys[key.KeyID] = key
	return nil
}

func (m *mockApiKeyRepo) GetByID(_ context.Context, keyID string) (*model.BrandApiKey, error) {
	k, ok := m.keys[keyID]
	if !ok {
		return nil, pgx.ErrNoRows
	}
	return k, nil
}

func (m *mockApiKeyRepo) ListByOrg(_ context.Context, orgID string) ([]*model.BrandApiKey, error) {
	var result []*model.BrandApiKey
	for _, k := range m.keys {
		if k.OrgID == orgID {
			result = append(result, k)
		}
	}
	return result, nil
}

func (m *mockApiKeyRepo) Revoke(_ context.Context, keyID string) error {
	k, ok := m.keys[keyID]
	if !ok {
		return pgx.ErrNoRows
	}
	now := time.Now()
	k.RevokedAt = &now
	return nil
}

func (m *mockApiKeyRepo) ValidateKey(_ context.Context, keyPlaintext string) (*model.BrandApiKey, error) {
	for _, k := range m.keys {
		if k.RevokedAt != nil {
			continue
		}
		if err := bcrypt.CompareHashAndPassword([]byte(k.KeyHash), []byte(keyPlaintext)); err == nil {
			return k, nil
		}
	}
	return nil, pgx.ErrNoRows
}
