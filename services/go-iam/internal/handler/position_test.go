package handler

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"

	"rcprotocol/services/go-iam/internal/model"
)

// mockPositionRepo implements repo.PositionRepository for handler tests.
type mockPositionRepo struct {
	positions map[string]*model.Position
}

func newMockPositionRepo() *mockPositionRepo {
	return &mockPositionRepo{positions: make(map[string]*model.Position)}
}

func (m *mockPositionRepo) Create(_ context.Context, pos *model.Position) error {
	pos.CreatedAt = time.Now()
	m.positions[pos.PositionID] = pos
	return nil
}

func (m *mockPositionRepo) GetByID(_ context.Context, positionID string) (*model.Position, error) {
	p, ok := m.positions[positionID]
	if !ok {
		return nil, pgx.ErrNoRows
	}
	return p, nil
}

func (m *mockPositionRepo) ListByOrg(_ context.Context, orgID string) ([]model.Position, error) {
	var result []model.Position
	for _, p := range m.positions {
		if p.OrgID == orgID {
			result = append(result, *p)
		}
	}
	if result == nil {
		result = []model.Position{}
	}
	return result, nil
}

// --- Position Create: compatible combinations ---

func TestPositionCreate201_PlatformPlatform(t *testing.T) {
	posRepo := newMockPositionRepo()
	orgRepo := newMockOrgRepo()
	orgRepo.orgs["org-1"] = &model.Organization{OrgID: "org-1", OrgName: "P", OrgType: "platform", Status: "active"}
	h := NewPositionHandler(posRepo, orgRepo)

	body := `{"position_name":"Admin","protocol_role":"Platform"}`
	req := httptest.NewRequest(http.MethodPost, "/orgs/org-1/positions", bytes.NewBufferString(body))
	req = withChiURLParam(req, "org_id", "org-1")
	w := httptest.NewRecorder()
	h.Create(w, req)

	if w.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d: %s", w.Code, w.Body.String())
	}
	var resp SuccessBody
	_ = json.Unmarshal(w.Body.Bytes(), &resp)
	data := resp.Data.(map[string]interface{})
	if data["protocol_role"] != "Platform" {
		t.Errorf("expected Platform, got %v", data["protocol_role"])
	}
}

func TestPositionCreate201_PlatformModerator(t *testing.T) {
	posRepo := newMockPositionRepo()
	orgRepo := newMockOrgRepo()
	orgRepo.orgs["org-1"] = &model.Organization{OrgID: "org-1", OrgName: "P", OrgType: "platform", Status: "active"}
	h := NewPositionHandler(posRepo, orgRepo)

	body := `{"position_name":"Mod","protocol_role":"Moderator"}`
	req := httptest.NewRequest(http.MethodPost, "/orgs/org-1/positions", bytes.NewBufferString(body))
	req = withChiURLParam(req, "org_id", "org-1")
	w := httptest.NewRecorder()
	h.Create(w, req)

	if w.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d: %s", w.Code, w.Body.String())
	}
}

func TestPositionCreate201_BrandBrand(t *testing.T) {
	posRepo := newMockPositionRepo()
	orgRepo := newMockOrgRepo()
	brandID := "brand-1"
	orgRepo.orgs["org-2"] = &model.Organization{OrgID: "org-2", OrgName: "B", OrgType: "brand", BrandID: &brandID, Status: "active"}
	h := NewPositionHandler(posRepo, orgRepo)

	body := `{"position_name":"BrandMgr","protocol_role":"Brand"}`
	req := httptest.NewRequest(http.MethodPost, "/orgs/org-2/positions", bytes.NewBufferString(body))
	req = withChiURLParam(req, "org_id", "org-2")
	w := httptest.NewRecorder()
	h.Create(w, req)

	if w.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d: %s", w.Code, w.Body.String())
	}
}

func TestPositionCreate201_FactoryFactory(t *testing.T) {
	posRepo := newMockPositionRepo()
	orgRepo := newMockOrgRepo()
	orgRepo.orgs["org-3"] = &model.Organization{OrgID: "org-3", OrgName: "F", OrgType: "factory", Status: "active"}
	h := NewPositionHandler(posRepo, orgRepo)

	body := `{"position_name":"Worker","protocol_role":"Factory"}`
	req := httptest.NewRequest(http.MethodPost, "/orgs/org-3/positions", bytes.NewBufferString(body))
	req = withChiURLParam(req, "org_id", "org-3")
	w := httptest.NewRecorder()
	h.Create(w, req)

	if w.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d: %s", w.Code, w.Body.String())
	}
}

// --- Position Create: incompatible combinations ---

func TestPositionCreate400_BrandFactory(t *testing.T) {
	posRepo := newMockPositionRepo()
	orgRepo := newMockOrgRepo()
	brandID := "brand-1"
	orgRepo.orgs["org-2"] = &model.Organization{OrgID: "org-2", OrgName: "B", OrgType: "brand", BrandID: &brandID, Status: "active"}
	h := NewPositionHandler(posRepo, orgRepo)

	body := `{"position_name":"Wrong","protocol_role":"Factory"}`
	req := httptest.NewRequest(http.MethodPost, "/orgs/org-2/positions", bytes.NewBufferString(body))
	req = withChiURLParam(req, "org_id", "org-2")
	w := httptest.NewRecorder()
	h.Create(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d: %s", w.Code, w.Body.String())
	}
	var resp ErrorBody
	_ = json.Unmarshal(w.Body.Bytes(), &resp)
	if resp.Error.Message != "protocol_role is not compatible with organization type" {
		t.Errorf("unexpected message: %q", resp.Error.Message)
	}
}

func TestPositionCreate400_FactoryBrand(t *testing.T) {
	posRepo := newMockPositionRepo()
	orgRepo := newMockOrgRepo()
	orgRepo.orgs["org-3"] = &model.Organization{OrgID: "org-3", OrgName: "F", OrgType: "factory", Status: "active"}
	h := NewPositionHandler(posRepo, orgRepo)

	body := `{"position_name":"Wrong","protocol_role":"Brand"}`
	req := httptest.NewRequest(http.MethodPost, "/orgs/org-3/positions", bytes.NewBufferString(body))
	req = withChiURLParam(req, "org_id", "org-3")
	w := httptest.NewRecorder()
	h.Create(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d: %s", w.Code, w.Body.String())
	}
}

func TestPositionCreate400_PlatformBrand(t *testing.T) {
	posRepo := newMockPositionRepo()
	orgRepo := newMockOrgRepo()
	orgRepo.orgs["org-1"] = &model.Organization{OrgID: "org-1", OrgName: "P", OrgType: "platform", Status: "active"}
	h := NewPositionHandler(posRepo, orgRepo)

	body := `{"position_name":"Wrong","protocol_role":"Brand"}`
	req := httptest.NewRequest(http.MethodPost, "/orgs/org-1/positions", bytes.NewBufferString(body))
	req = withChiURLParam(req, "org_id", "org-1")
	w := httptest.NewRecorder()
	h.Create(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d: %s", w.Code, w.Body.String())
	}
}

// --- Position Create: org not found ---

func TestPositionCreate404_OrgNotFound(t *testing.T) {
	posRepo := newMockPositionRepo()
	orgRepo := newMockOrgRepo()
	h := NewPositionHandler(posRepo, orgRepo)

	body := `{"position_name":"Admin","protocol_role":"Platform"}`
	req := httptest.NewRequest(http.MethodPost, "/orgs/nonexistent/positions", bytes.NewBufferString(body))
	req = withChiURLParam(req, "org_id", "nonexistent")
	w := httptest.NewRecorder()
	h.Create(w, req)

	if w.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d: %s", w.Code, w.Body.String())
	}
	var resp ErrorBody
	_ = json.Unmarshal(w.Body.Bytes(), &resp)
	if resp.Error.Message != "organization not found" {
		t.Errorf("unexpected message: %q", resp.Error.Message)
	}
}

// --- isRoleCompatible: all 15 combinations (5 roles × 3 org types) ---

func TestIsRoleCompatible_AllCombinations(t *testing.T) {
	tests := []struct {
		orgType      string
		protocolRole string
		expected     bool
	}{
		{"platform", "Platform", true},
		{"platform", "Brand", false},
		{"platform", "Factory", false},
		{"platform", "Consumer", false},
		{"platform", "Moderator", true},

		{"brand", "Platform", false},
		{"brand", "Brand", true},
		{"brand", "Factory", false},
		{"brand", "Consumer", false},
		{"brand", "Moderator", false},

		{"factory", "Platform", false},
		{"factory", "Brand", false},
		{"factory", "Factory", true},
		{"factory", "Consumer", false},
		{"factory", "Moderator", false},
	}

	for _, tc := range tests {
		name := tc.orgType + "_" + tc.protocolRole
		t.Run(name, func(t *testing.T) {
			got := isRoleCompatible(tc.orgType, tc.protocolRole)
			if got != tc.expected {
				t.Errorf("isRoleCompatible(%q, %q) = %v, want %v", tc.orgType, tc.protocolRole, got, tc.expected)
			}
		})
	}
}

// --- Position List ---

func TestPositionList(t *testing.T) {
	posRepo := newMockPositionRepo()
	orgRepo := newMockOrgRepo()
	orgRepo.orgs["org-1"] = &model.Organization{OrgID: "org-1", OrgName: "P", OrgType: "platform", Status: "active"}
	h := NewPositionHandler(posRepo, orgRepo)

	// Create two positions
	for _, role := range []string{"Platform", "Moderator"} {
		body := `{"position_name":"Pos","protocol_role":"` + role + `"}`
		req := httptest.NewRequest(http.MethodPost, "/orgs/org-1/positions", bytes.NewBufferString(body))
		req = withChiURLParam(req, "org_id", "org-1")
		w := httptest.NewRecorder()
		h.Create(w, req)
		if w.Code != http.StatusCreated {
			t.Fatalf("create: expected 201, got %d", w.Code)
		}
	}

	// List
	listReq := httptest.NewRequest(http.MethodGet, "/orgs/org-1/positions", nil)
	listReq = withChiURLParam(listReq, "org_id", "org-1")
	listW := httptest.NewRecorder()
	h.List(listW, listReq)

	if listW.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", listW.Code, listW.Body.String())
	}

	var resp SuccessBody
	_ = json.Unmarshal(listW.Body.Bytes(), &resp)
	items, ok := resp.Data.([]interface{})
	if !ok {
		t.Fatalf("data is not array")
	}
	if len(items) != 2 {
		t.Errorf("expected 2 positions, got %d", len(items))
	}
}
