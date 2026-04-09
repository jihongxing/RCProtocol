package handler

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"

	"rcprotocol/services/go-iam/internal/model"
)

// mockUserRepo implements repo.UserRepository for handler tests.
type mockUserRepo struct {
	users map[string]*model.User
}

func newMockUserRepo() *mockUserRepo {
	return &mockUserRepo{users: make(map[string]*model.User)}
}

func (m *mockUserRepo) Create(_ context.Context, user *model.User) error {
	for _, u := range m.users {
		if u.Email == user.Email {
			return &pgconn.PgError{Code: "23505"}
		}
	}
	now := time.Now()
	user.CreatedAt = now
	user.UpdatedAt = now
	m.users[user.UserID] = user
	return nil
}

func (m *mockUserRepo) GetByID(_ context.Context, userID string) (*model.User, error) {
	u, ok := m.users[userID]
	if !ok {
		return nil, pgx.ErrNoRows
	}
	return u, nil
}

func (m *mockUserRepo) GetByEmail(_ context.Context, email string) (*model.User, error) {
	for _, u := range m.users {
		if u.Email == email {
			return u, nil
		}
	}
	return nil, pgx.ErrNoRows
}

func (m *mockUserRepo) List(_ context.Context, page, pageSize int) ([]model.User, int, error) {
	all := make([]model.User, 0, len(m.users))
	for _, u := range m.users {
		all = append(all, *u)
	}
	total := len(all)
	offset := (page - 1) * pageSize
	if offset > total {
		offset = total
	}
	end := offset + pageSize
	if end > total {
		end = total
	}
	return all[offset:end], total, nil
}

func (m *mockUserRepo) Update(_ context.Context, userID, displayName, status string) (*model.User, error) {
	u, ok := m.users[userID]
	if !ok {
		return nil, pgx.ErrNoRows
	}
	u.DisplayName = displayName
	u.Status = status
	u.UpdatedAt = time.Now()
	return u, nil
}

func (m *mockUserRepo) Disable(_ context.Context, userID string) error {
	u, ok := m.users[userID]
	if !ok {
		return pgx.ErrNoRows
	}
	u.Status = "disabled"
	u.UpdatedAt = time.Now()
	return nil
}

func (m *mockUserRepo) UpdatePasswordHash(_ context.Context, userID, newHash string) error {
	u, ok := m.users[userID]
	if !ok {
		return pgx.ErrNoRows
	}
	u.PasswordHash = newHash
	u.UpdatedAt = time.Now()
	return nil
}

// withChiURLParam adds chi route context to the request.
func withChiURLParam(r *http.Request, key, value string) *http.Request {
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add(key, value)
	return r.WithContext(context.WithValue(r.Context(), chi.RouteCtxKey, rctx))
}

func TestUserCreate201(t *testing.T) {
	mock := newMockUserRepo()
	h := NewUserHandler(mock)

	body := `{"email":"test@example.com","password":"12345678","display_name":"Test User"}`
	req := httptest.NewRequest(http.MethodPost, "/users", bytes.NewBufferString(body))
	w := httptest.NewRecorder()

	h.Create(w, req)

	if w.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d: %s", w.Code, w.Body.String())
	}

	var resp SuccessBody
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	data, ok := resp.Data.(map[string]interface{})
	if !ok {
		t.Fatalf("data is not map[string]interface{}")
	}

	if data["user_id"] == nil || data["user_id"] == "" {
		t.Error("expected user_id in response")
	}
	if data["email"] != "test@example.com" {
		t.Errorf("expected email test@example.com, got %v", data["email"])
	}
	if data["display_name"] != "Test User" {
		t.Errorf("expected display_name Test User, got %v", data["display_name"])
	}
	if data["status"] != "active" {
		t.Errorf("expected status active, got %v", data["status"])
	}
	// password_hash must NOT appear (json:"-")
	if _, exists := data["password_hash"]; exists {
		t.Error("password_hash must not appear in JSON response")
	}
}

func TestUserCreate409_DuplicateEmail(t *testing.T) {
	mock := newMockUserRepo()
	h := NewUserHandler(mock)

	body := `{"email":"dup@example.com","password":"12345678","display_name":"User 1"}`
	req := httptest.NewRequest(http.MethodPost, "/users", bytes.NewBufferString(body))
	w := httptest.NewRecorder()
	h.Create(w, req)
	if w.Code != http.StatusCreated {
		t.Fatalf("first create: expected 201, got %d", w.Code)
	}

	req2 := httptest.NewRequest(http.MethodPost, "/users", bytes.NewBufferString(body))
	w2 := httptest.NewRecorder()
	h.Create(w2, req2)

	if w2.Code != http.StatusConflict {
		t.Fatalf("expected 409, got %d: %s", w2.Code, w2.Body.String())
	}

	var resp ErrorBody
	if err := json.Unmarshal(w2.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if resp.Error.Code != "CONFLICT" {
		t.Errorf("expected CONFLICT code, got %s", resp.Error.Code)
	}
}

func TestUserCreate400_InvalidEmail(t *testing.T) {
	mock := newMockUserRepo()
	h := NewUserHandler(mock)

	body := `{"email":"invalid-email","password":"12345678","display_name":"Test"}`
	req := httptest.NewRequest(http.MethodPost, "/users", bytes.NewBufferString(body))
	w := httptest.NewRecorder()

	h.Create(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d: %s", w.Code, w.Body.String())
	}

	var resp ErrorBody
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if resp.Error.Message != "invalid email format" {
		t.Errorf("expected 'invalid email format', got %q", resp.Error.Message)
	}
}

func TestUserCreate400_ShortPassword(t *testing.T) {
	mock := newMockUserRepo()
	h := NewUserHandler(mock)

	body := `{"email":"test@example.com","password":"short","display_name":"Test"}`
	req := httptest.NewRequest(http.MethodPost, "/users", bytes.NewBufferString(body))
	w := httptest.NewRecorder()

	h.Create(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d: %s", w.Code, w.Body.String())
	}

	var resp ErrorBody
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if resp.Error.Message != "password must be at least 8 characters" {
		t.Errorf("expected password message, got %q", resp.Error.Message)
	}
}

func TestUserList(t *testing.T) {
	mock := newMockUserRepo()
	h := NewUserHandler(mock)

	// Seed 3 users
	for i := 0; i < 3; i++ {
		body := fmt.Sprintf(`{"email":"u%d@example.com","password":"12345678","display_name":"User %d"}`, i, i)
		req := httptest.NewRequest(http.MethodPost, "/users", bytes.NewBufferString(body))
		w := httptest.NewRecorder()
		h.Create(w, req)
		if w.Code != http.StatusCreated {
			t.Fatalf("seed user %d: expected 201, got %d", i, w.Code)
		}
	}

	req := httptest.NewRequest(http.MethodGet, "/users?page=1&page_size=2", nil)
	w := httptest.NewRecorder()
	h.List(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}

	var resp ListBody
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if resp.Page != 1 {
		t.Errorf("expected page 1, got %d", resp.Page)
	}
	if resp.Size != 2 {
		t.Errorf("expected page_size 2, got %d", resp.Size)
	}
	if resp.Total != 3 {
		t.Errorf("expected total 3, got %d", resp.Total)
	}
	items, ok := resp.Data.([]interface{})
	if !ok {
		t.Fatalf("data is not array")
	}
	if len(items) != 2 {
		t.Errorf("expected 2 items, got %d", len(items))
	}
}

func TestUserGetByID200(t *testing.T) {
	mock := newMockUserRepo()
	h := NewUserHandler(mock)

	// Create one user
	body := `{"email":"get@example.com","password":"12345678","display_name":"Get User"}`
	req := httptest.NewRequest(http.MethodPost, "/users", bytes.NewBufferString(body))
	w := httptest.NewRecorder()
	h.Create(w, req)

	var createResp SuccessBody
	_ = json.Unmarshal(w.Body.Bytes(), &createResp)
	data := createResp.Data.(map[string]interface{})
	userID := data["user_id"].(string)

	// GetByID
	getReq := httptest.NewRequest(http.MethodGet, "/users/"+userID, nil)
	getReq = withChiURLParam(getReq, "id", userID)
	getW := httptest.NewRecorder()
	h.GetByID(getW, getReq)

	if getW.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", getW.Code, getW.Body.String())
	}

	var resp SuccessBody
	_ = json.Unmarshal(getW.Body.Bytes(), &resp)
	d := resp.Data.(map[string]interface{})
	if d["user_id"] != userID {
		t.Errorf("expected %s, got %v", userID, d["user_id"])
	}
}

func TestUserGetByID404(t *testing.T) {
	mock := newMockUserRepo()
	h := NewUserHandler(mock)

	req := httptest.NewRequest(http.MethodGet, "/users/nonexistent", nil)
	req = withChiURLParam(req, "id", "nonexistent")
	w := httptest.NewRecorder()
	h.GetByID(w, req)

	if w.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d: %s", w.Code, w.Body.String())
	}
}

func TestUserUpdate200(t *testing.T) {
	mock := newMockUserRepo()
	h := NewUserHandler(mock)

	// Create user
	body := `{"email":"upd@example.com","password":"12345678","display_name":"Original"}`
	req := httptest.NewRequest(http.MethodPost, "/users", bytes.NewBufferString(body))
	w := httptest.NewRecorder()
	h.Create(w, req)

	var createResp SuccessBody
	_ = json.Unmarshal(w.Body.Bytes(), &createResp)
	data := createResp.Data.(map[string]interface{})
	userID := data["user_id"].(string)

	// Update
	updateBody := `{"display_name":"Updated","status":"active"}`
	updateReq := httptest.NewRequest(http.MethodPut, "/users/"+userID, bytes.NewBufferString(updateBody))
	updateReq = withChiURLParam(updateReq, "id", userID)
	updateW := httptest.NewRecorder()
	h.Update(updateW, updateReq)

	if updateW.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", updateW.Code, updateW.Body.String())
	}

	var resp SuccessBody
	_ = json.Unmarshal(updateW.Body.Bytes(), &resp)
	d := resp.Data.(map[string]interface{})
	if d["display_name"] != "Updated" {
		t.Errorf("expected Updated, got %v", d["display_name"])
	}
}

func TestUserDelete200(t *testing.T) {
	mock := newMockUserRepo()
	h := NewUserHandler(mock)

	// Create user
	body := `{"email":"del@example.com","password":"12345678","display_name":"Delete Me"}`
	req := httptest.NewRequest(http.MethodPost, "/users", bytes.NewBufferString(body))
	w := httptest.NewRecorder()
	h.Create(w, req)

	var createResp SuccessBody
	_ = json.Unmarshal(w.Body.Bytes(), &createResp)
	data := createResp.Data.(map[string]interface{})
	userID := data["user_id"].(string)

	// Delete (soft-delete)
	delReq := httptest.NewRequest(http.MethodDelete, "/users/"+userID, nil)
	delReq = withChiURLParam(delReq, "id", userID)
	delW := httptest.NewRecorder()
	h.Delete(delW, delReq)

	if delW.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", delW.Code, delW.Body.String())
	}

	// Verify user is disabled
	user, _ := mock.GetByID(context.Background(), userID)
	if user.Status != "disabled" {
		t.Errorf("expected status disabled, got %s", user.Status)
	}
}
