package handler

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5"

	"rcprotocol/services/go-iam/internal/model"
)

// mockMemberRepo implements repo.MemberRepository for handler tests.
type mockMemberRepo struct {
	bindings map[string]*model.UserOrgPosition // key: userID+"|"+orgID
	members  []model.MemberView
	userOrgs map[string][]model.UserOrgView // key: userID
}

func newMockMemberRepo() *mockMemberRepo {
	return &mockMemberRepo{
		bindings: make(map[string]*model.UserOrgPosition),
		userOrgs: make(map[string][]model.UserOrgView),
	}
}

func (m *mockMemberRepo) Bind(_ context.Context, uop *model.UserOrgPosition) error {
	key := uop.UserID + "|" + uop.OrgID
	uop.CreatedAt = time.Now()
	m.bindings[key] = uop
	return nil
}

func (m *mockMemberRepo) Unbind(_ context.Context, userID, orgID string) error {
	key := userID + "|" + orgID
	if _, ok := m.bindings[key]; !ok {
		return pgx.ErrNoRows
	}
	delete(m.bindings, key)
	return nil
}

func (m *mockMemberRepo) ListByOrg(_ context.Context, orgID string) ([]model.MemberView, error) {
	var result []model.MemberView
	for _, mv := range m.members {
		result = append(result, mv)
	}
	if result == nil {
		result = []model.MemberView{}
	}
	return result, nil
}

func (m *mockMemberRepo) GetUserOrgs(_ context.Context, userID string) ([]model.UserOrgView, error) {
	orgs, ok := m.userOrgs[userID]
	if !ok {
		return []model.UserOrgView{}, nil
	}
	return orgs, nil
}

func (m *mockMemberRepo) ExistsByUserAndOrg(_ context.Context, userID, orgID string) (bool, error) {
	key := userID + "|" + orgID
	_, exists := m.bindings[key]
	return exists, nil
}

// withChiURLParams adds multiple chi URL params to the request.
func withChiURLParams(r *http.Request, params map[string]string) *http.Request {
	rctx := chi.NewRouteContext()
	for key, value := range params {
		rctx.URLParams.Add(key, value)
	}
	return r.WithContext(context.WithValue(r.Context(), chi.RouteCtxKey, rctx))
}

// --- Bind 201 ---

func TestMemberBind201(t *testing.T) {
	memberRepo := newMockMemberRepo()
	userRepo := newMockUserRepo()
	posRepo := newMockPositionRepo()

	userRepo.users["user-1"] = &model.User{UserID: "user-1", Email: "a@b.com", DisplayName: "A", Status: "active"}
	posRepo.positions["pos-1"] = &model.Position{PositionID: "pos-1", OrgID: "org-1", PositionName: "Admin", ProtocolRole: "Platform"}

	h := NewMemberHandler(memberRepo, userRepo, posRepo)

	body := `{"user_id":"user-1","position_id":"pos-1"}`
	req := httptest.NewRequest(http.MethodPost, "/orgs/org-1/members", bytes.NewBufferString(body))
	req = withChiURLParam(req, "org_id", "org-1")
	w := httptest.NewRecorder()
	h.Bind(w, req)

	if w.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d: %s", w.Code, w.Body.String())
	}

	var resp SuccessBody
	_ = json.Unmarshal(w.Body.Bytes(), &resp)
	data := resp.Data.(map[string]interface{})
	if data["user_id"] != "user-1" {
		t.Errorf("expected user-1, got %v", data["user_id"])
	}
	if data["org_id"] != "org-1" {
		t.Errorf("expected org-1, got %v", data["org_id"])
	}
}

// --- Bind 404: user not found ---

func TestMemberBind404_UserNotFound(t *testing.T) {
	memberRepo := newMockMemberRepo()
	userRepo := newMockUserRepo()
	posRepo := newMockPositionRepo()

	posRepo.positions["pos-1"] = &model.Position{PositionID: "pos-1", OrgID: "org-1", PositionName: "Admin", ProtocolRole: "Platform"}

	h := NewMemberHandler(memberRepo, userRepo, posRepo)

	body := `{"user_id":"nonexistent","position_id":"pos-1"}`
	req := httptest.NewRequest(http.MethodPost, "/orgs/org-1/members", bytes.NewBufferString(body))
	req = withChiURLParam(req, "org_id", "org-1")
	w := httptest.NewRecorder()
	h.Bind(w, req)

	if w.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d: %s", w.Code, w.Body.String())
	}
	var resp ErrorBody
	_ = json.Unmarshal(w.Body.Bytes(), &resp)
	if resp.Error.Message != "user not found" {
		t.Errorf("unexpected message: %q", resp.Error.Message)
	}
}

// --- Bind 400: position does not belong to org ---

func TestMemberBind400_PositionWrongOrg(t *testing.T) {
	memberRepo := newMockMemberRepo()
	userRepo := newMockUserRepo()
	posRepo := newMockPositionRepo()

	userRepo.users["user-1"] = &model.User{UserID: "user-1", Email: "a@b.com", DisplayName: "A", Status: "active"}
	posRepo.positions["pos-1"] = &model.Position{PositionID: "pos-1", OrgID: "org-OTHER", PositionName: "Admin", ProtocolRole: "Platform"}

	h := NewMemberHandler(memberRepo, userRepo, posRepo)

	body := `{"user_id":"user-1","position_id":"pos-1"}`
	req := httptest.NewRequest(http.MethodPost, "/orgs/org-1/members", bytes.NewBufferString(body))
	req = withChiURLParam(req, "org_id", "org-1")
	w := httptest.NewRecorder()
	h.Bind(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d: %s", w.Code, w.Body.String())
	}
	var resp ErrorBody
	_ = json.Unmarshal(w.Body.Bytes(), &resp)
	if resp.Error.Message != "position does not belong to this organization" {
		t.Errorf("unexpected message: %q", resp.Error.Message)
	}
}

// --- Bind 409: duplicate binding ---

func TestMemberBind409_Duplicate(t *testing.T) {
	memberRepo := newMockMemberRepo()
	userRepo := newMockUserRepo()
	posRepo := newMockPositionRepo()

	userRepo.users["user-1"] = &model.User{UserID: "user-1", Email: "a@b.com", DisplayName: "A", Status: "active"}
	posRepo.positions["pos-1"] = &model.Position{PositionID: "pos-1", OrgID: "org-1", PositionName: "Admin", ProtocolRole: "Platform"}

	// Pre-populate binding
	memberRepo.bindings["user-1|org-1"] = &model.UserOrgPosition{UserID: "user-1", OrgID: "org-1", PositionID: "pos-1"}

	h := NewMemberHandler(memberRepo, userRepo, posRepo)

	body := `{"user_id":"user-1","position_id":"pos-1"}`
	req := httptest.NewRequest(http.MethodPost, "/orgs/org-1/members", bytes.NewBufferString(body))
	req = withChiURLParam(req, "org_id", "org-1")
	w := httptest.NewRecorder()
	h.Bind(w, req)

	if w.Code != http.StatusConflict {
		t.Fatalf("expected 409, got %d: %s", w.Code, w.Body.String())
	}
	var resp ErrorBody
	_ = json.Unmarshal(w.Body.Bytes(), &resp)
	if resp.Error.Message != "user already has a position in this organization" {
		t.Errorf("unexpected message: %q", resp.Error.Message)
	}
}

// --- List members ---

func TestMemberList(t *testing.T) {
	memberRepo := newMockMemberRepo()
	memberRepo.members = []model.MemberView{
		{UserID: "u1", DisplayName: "User 1", Email: "u1@x.com", PositionID: "p1", PositionName: "Admin", ProtocolRole: "Platform"},
		{UserID: "u2", DisplayName: "User 2", Email: "u2@x.com", PositionID: "p2", PositionName: "Mod", ProtocolRole: "Moderator"},
	}
	userRepo := newMockUserRepo()
	posRepo := newMockPositionRepo()
	h := NewMemberHandler(memberRepo, userRepo, posRepo)

	req := httptest.NewRequest(http.MethodGet, "/orgs/org-1/members", nil)
	req = withChiURLParam(req, "org_id", "org-1")
	w := httptest.NewRecorder()
	h.List(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var resp SuccessBody
	_ = json.Unmarshal(w.Body.Bytes(), &resp)
	items, ok := resp.Data.([]interface{})
	if !ok {
		t.Fatalf("data is not array")
	}
	if len(items) != 2 {
		t.Errorf("expected 2 members, got %d", len(items))
	}

	first := items[0].(map[string]interface{})
	if first["user_id"] == nil {
		t.Error("expected user_id")
	}
	if first["display_name"] == nil {
		t.Error("expected display_name")
	}
	if first["position_name"] == nil {
		t.Error("expected position_name")
	}
	if first["protocol_role"] == nil {
		t.Error("expected protocol_role")
	}
}

// --- Unbind 200 ---

func TestMemberUnbind200(t *testing.T) {
	memberRepo := newMockMemberRepo()
	memberRepo.bindings["user-1|org-1"] = &model.UserOrgPosition{UserID: "user-1", OrgID: "org-1", PositionID: "pos-1"}
	userRepo := newMockUserRepo()
	posRepo := newMockPositionRepo()
	h := NewMemberHandler(memberRepo, userRepo, posRepo)

	req := httptest.NewRequest(http.MethodDelete, "/orgs/org-1/members/user-1", nil)
	req = withChiURLParams(req, map[string]string{"org_id": "org-1", "user_id": "user-1"})
	w := httptest.NewRecorder()
	h.Unbind(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
}

// --- Unbind 404: not found ---

func TestMemberUnbind404(t *testing.T) {
	memberRepo := newMockMemberRepo()
	userRepo := newMockUserRepo()
	posRepo := newMockPositionRepo()
	h := NewMemberHandler(memberRepo, userRepo, posRepo)

	req := httptest.NewRequest(http.MethodDelete, "/orgs/org-1/members/user-1", nil)
	req = withChiURLParams(req, map[string]string{"org_id": "org-1", "user_id": "user-1"})
	w := httptest.NewRecorder()
	h.Unbind(w, req)

	if w.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d: %s", w.Code, w.Body.String())
	}
	var resp ErrorBody
	_ = json.Unmarshal(w.Body.Bytes(), &resp)
	if resp.Error.Message != "member binding not found" {
		t.Errorf("unexpected message: %q", resp.Error.Message)
	}
}
