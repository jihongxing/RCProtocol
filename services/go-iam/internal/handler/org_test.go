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

// mockOrgRepo implements repo.OrgRepository for handler tests.
type mockOrgRepo struct {
	orgs map[string]*model.Organization
}

func newMockOrgRepo() *mockOrgRepo {
	return &mockOrgRepo{orgs: make(map[string]*model.Organization)}
}

func (m *mockOrgRepo) Create(_ context.Context, org *model.Organization) error {
	now := time.Now()
	org.CreatedAt = now
	org.UpdatedAt = now
	m.orgs[org.OrgID] = org
	return nil
}

func (m *mockOrgRepo) GetByID(_ context.Context, orgID string) (*model.Organization, error) {
	o, ok := m.orgs[orgID]
	if !ok {
		return nil, pgx.ErrNoRows
	}
	return o, nil
}

func (m *mockOrgRepo) List(_ context.Context, orgType string, page, pageSize int) ([]model.Organization, int, error) {
	var filtered []model.Organization
	for _, o := range m.orgs {
		if orgType == "" || o.OrgType == orgType {
			filtered = append(filtered, *o)
		}
	}
	total := len(filtered)
	offset := (page - 1) * pageSize
	if offset > total {
		offset = total
	}
	end := offset + pageSize
	if end > total {
		end = total
	}
	return filtered[offset:end], total, nil
}

func (m *mockOrgRepo) Update(_ context.Context, orgID, orgName, status string) (*model.Organization, error) {
	o, ok := m.orgs[orgID]
	if !ok {
		return nil, pgx.ErrNoRows
	}
	o.OrgName = orgName
	o.Status = status
	o.UpdatedAt = time.Now()
	return o, nil
}

func TestOrgCreate201(t *testing.T) {
	mock := newMockOrgRepo()
	h := NewOrgHandler(mock)

	body := `{"org_name":"Acme Corp","org_type":"platform"}`
	req := httptest.NewRequest(http.MethodPost, "/orgs", bytes.NewBufferString(body))
	w := httptest.NewRecorder()

	h.Create(w, req)

	if w.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d: %s", w.Code, w.Body.String())
	}

	var resp SuccessBody
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	data := resp.Data.(map[string]interface{})
	if data["org_id"] == nil || data["org_id"] == "" {
		t.Error("expected org_id in response")
	}
	if data["org_name"] != "Acme Corp" {
		t.Errorf("expected org_name Acme Corp, got %v", data["org_name"])
	}
	if data["org_type"] != "platform" {
		t.Errorf("expected org_type platform, got %v", data["org_type"])
	}
	if data["status"] != "active" {
		t.Errorf("expected status active, got %v", data["status"])
	}
}

func TestOrgCreate400_InvalidOrgType(t *testing.T) {
	mock := newMockOrgRepo()
	h := NewOrgHandler(mock)

	body := `{"org_name":"Bad Org","org_type":"invalid"}`
	req := httptest.NewRequest(http.MethodPost, "/orgs", bytes.NewBufferString(body))
	w := httptest.NewRecorder()

	h.Create(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d: %s", w.Code, w.Body.String())
	}

	var resp ErrorBody
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if resp.Error.Code != "INVALID_INPUT" {
		t.Errorf("expected INVALID_INPUT, got %s", resp.Error.Code)
	}
}

func TestOrgCreate400_BrandWithoutBrandID(t *testing.T) {
	mock := newMockOrgRepo()
	h := NewOrgHandler(mock)

	// Phase 2: brand_id is auto-generated, but contact fields are required
	body := `{"org_name":"Brand Org","org_type":"brand"}`
	req := httptest.NewRequest(http.MethodPost, "/orgs", bytes.NewBufferString(body))
	w := httptest.NewRecorder()

	h.Create(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d: %s", w.Code, w.Body.String())
	}

	var resp ErrorBody
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if resp.Error.Message != "contact_email is required for brand organizations" {
		t.Errorf("expected 'contact_email is required for brand organizations', got %q", resp.Error.Message)
	}
}

func TestOrgCreate201_BrandWithBrandID(t *testing.T) {
	mock := newMockOrgRepo()
	h := NewOrgHandler(mock)

	body := `{"org_name":"Brand Org","org_type":"brand","brand_id":"brand-123","contact_email":"test@example.com","contact_phone":"1234567890"}`
	req := httptest.NewRequest(http.MethodPost, "/orgs", bytes.NewBufferString(body))
	w := httptest.NewRecorder()

	h.Create(w, req)

	if w.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d: %s", w.Code, w.Body.String())
	}

	var resp SuccessBody
	_ = json.Unmarshal(w.Body.Bytes(), &resp)
	data := resp.Data.(map[string]interface{})
	if data["brand_id"] != "brand-123" {
		t.Errorf("expected brand_id brand-123, got %v", data["brand_id"])
	}
}

func TestOrgList_OrgTypeFilter(t *testing.T) {
	mock := newMockOrgRepo()
	h := NewOrgHandler(mock)

	// Create platform org
	body1 := `{"org_name":"Platform","org_type":"platform"}`
	req1 := httptest.NewRequest(http.MethodPost, "/orgs", bytes.NewBufferString(body1))
	w1 := httptest.NewRecorder()
	h.Create(w1, req1)

	// Create factory org
	body2 := `{"org_name":"Factory","org_type":"factory"}`
	req2 := httptest.NewRequest(http.MethodPost, "/orgs", bytes.NewBufferString(body2))
	w2 := httptest.NewRecorder()
	h.Create(w2, req2)

	// Create brand org
	body3 := `{"org_name":"Brand","org_type":"brand","brand_id":"b1","contact_email":"brand@example.com","contact_phone":"1234567890"}`
	req3 := httptest.NewRequest(http.MethodPost, "/orgs", bytes.NewBufferString(body3))
	w3 := httptest.NewRecorder()
	h.Create(w3, req3)

	// List all
	listReq := httptest.NewRequest(http.MethodGet, "/orgs?page=1&page_size=20", nil)
	listW := httptest.NewRecorder()
	h.List(listW, listReq)

	var allResp ListBody
	_ = json.Unmarshal(listW.Body.Bytes(), &allResp)
	if allResp.Total != 3 {
		t.Errorf("expected total 3, got %d", allResp.Total)
	}

	// List filtered by factory
	filterReq := httptest.NewRequest(http.MethodGet, "/orgs?org_type=factory&page=1&page_size=20", nil)
	filterW := httptest.NewRecorder()
	h.List(filterW, filterReq)

	var filterResp ListBody
	_ = json.Unmarshal(filterW.Body.Bytes(), &filterResp)
	if filterResp.Total != 1 {
		t.Errorf("expected total 1 for factory filter, got %d", filterResp.Total)
	}
	items := filterResp.Data.([]interface{})
	if len(items) != 1 {
		t.Errorf("expected 1 item, got %d", len(items))
	}
	item := items[0].(map[string]interface{})
	if item["org_type"] != "factory" {
		t.Errorf("expected org_type factory, got %v", item["org_type"])
	}
}

func TestOrgGetByID200(t *testing.T) {
	mock := newMockOrgRepo()
	h := NewOrgHandler(mock)

	body := `{"org_name":"Get Org","org_type":"platform"}`
	req := httptest.NewRequest(http.MethodPost, "/orgs", bytes.NewBufferString(body))
	w := httptest.NewRecorder()
	h.Create(w, req)

	var createResp SuccessBody
	_ = json.Unmarshal(w.Body.Bytes(), &createResp)
	data := createResp.Data.(map[string]interface{})
	orgID := data["org_id"].(string)

	getReq := httptest.NewRequest(http.MethodGet, "/orgs/"+orgID, nil)
	getReq = withChiURLParam(getReq, "id", orgID)
	getW := httptest.NewRecorder()
	h.GetByID(getW, getReq)

	if getW.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", getW.Code, getW.Body.String())
	}

	var resp SuccessBody
	_ = json.Unmarshal(getW.Body.Bytes(), &resp)
	d := resp.Data.(map[string]interface{})
	if d["org_id"] != orgID {
		t.Errorf("expected %s, got %v", orgID, d["org_id"])
	}
}

func TestOrgGetByID404(t *testing.T) {
	mock := newMockOrgRepo()
	h := NewOrgHandler(mock)

	req := httptest.NewRequest(http.MethodGet, "/orgs/nonexistent", nil)
	req = withChiURLParam(req, "id", "nonexistent")
	w := httptest.NewRecorder()
	h.GetByID(w, req)

	if w.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d: %s", w.Code, w.Body.String())
	}
}

func TestOrgUpdate_OrgTypeUnchanged(t *testing.T) {
	mock := newMockOrgRepo()
	h := NewOrgHandler(mock)

	// Create platform org
	body := `{"org_name":"Original","org_type":"platform"}`
	req := httptest.NewRequest(http.MethodPost, "/orgs", bytes.NewBufferString(body))
	w := httptest.NewRecorder()
	h.Create(w, req)

	var createResp SuccessBody
	_ = json.Unmarshal(w.Body.Bytes(), &createResp)
	data := createResp.Data.(map[string]interface{})
	orgID := data["org_id"].(string)

	// Update name and status only — org_type stays platform
	updateBody := `{"org_name":"Updated","status":"active"}`
	updateReq := httptest.NewRequest(http.MethodPut, "/orgs/"+orgID, bytes.NewBufferString(updateBody))
	updateReq = withChiURLParam(updateReq, "id", orgID)
	updateW := httptest.NewRecorder()
	h.Update(updateW, updateReq)

	if updateW.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", updateW.Code, updateW.Body.String())
	}

	var resp SuccessBody
	_ = json.Unmarshal(updateW.Body.Bytes(), &resp)
	d := resp.Data.(map[string]interface{})
	if d["org_name"] != "Updated" {
		t.Errorf("expected Updated, got %v", d["org_name"])
	}
	if d["org_type"] != "platform" {
		t.Errorf("org_type should remain platform, got %v", d["org_type"])
	}
}
