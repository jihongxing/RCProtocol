package handler

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"
	"pgregory.net/rapid"
	"rcprotocol/services/go-workorder/internal/downstream"
	"rcprotocol/services/go-workorder/internal/model"
)

// --- mock repo ---

type mockRepo struct {
	workorders map[string]*model.Workorder
}

func newMockRepo() *mockRepo {
	return &mockRepo{workorders: make(map[string]*model.Workorder)}
}

func (m *mockRepo) Create(_ context.Context, w *model.Workorder) error {
	cp := *w
	cp.CreatedAt = time.Now()
	cp.UpdatedAt = time.Now()
	m.workorders[w.ID] = &cp
	return nil
}

func (m *mockRepo) GetByID(_ context.Context, id string) (*model.Workorder, error) {
	w, ok := m.workorders[id]
	if !ok {
		return nil, nil
	}
	cp := *w
	return &cp, nil
}

func (m *mockRepo) List(_ context.Context, filterStatus, filterType, assigneeID, orgID, brandID string, page, pageSize int) ([]model.Workorder, int, error) {
	var filtered []model.Workorder
	for _, w := range m.workorders {
		if filterStatus != "" && w.Status != filterStatus {
			continue
		}
		if filterType != "" && w.Type != filterType {
			continue
		}
		if assigneeID != "" && (w.AssigneeID == nil || *w.AssigneeID != assigneeID) {
			continue
		}
		// Brand data isolation: orgID set means filter by creator_org_id or brand_id
		if orgID != "" {
			matchCreatorOrg := w.CreatorOrgID == orgID
			matchBrandID := w.BrandID != nil && *w.BrandID == brandID
			if !matchCreatorOrg && !matchBrandID {
				continue
			}
		}
		filtered = append(filtered, *w)
	}
	total := len(filtered)
	start := (page - 1) * pageSize
	if start >= total {
		return []model.Workorder{}, total, nil
	}
	end := start + pageSize
	if end > total {
		end = total
	}
	return filtered[start:end], total, nil
}

func (m *mockRepo) ListByAsset(_ context.Context, assetID string) ([]model.Workorder, error) {
	var result []model.Workorder
	for _, w := range m.workorders {
		if w.AssetID != nil && *w.AssetID == assetID {
			result = append(result, *w)
		}
	}
	return result, nil
}

func (m *mockRepo) UpdateStatus(_ context.Context, id, expectedStatus, newStatus string) (bool, error) {
	w, ok := m.workorders[id]
	if !ok || w.Status != expectedStatus {
		return false, nil
	}
	w.Status = newStatus
	w.UpdatedAt = time.Now()
	return true, nil
}

func (m *mockRepo) Assign(_ context.Context, id string, expectedStatuses []string, assigneeID, assigneeRole string) (bool, error) {
	w, ok := m.workorders[id]
	if !ok {
		return false, nil
	}
	allowed := false
	for _, s := range expectedStatuses {
		if w.Status == s {
			allowed = true
			break
		}
	}
	if !allowed {
		return false, nil
	}
	w.AssigneeID = &assigneeID
	w.AssigneeRole = &assigneeRole
	w.Status = model.StatusAssigned
	w.UpdatedAt = time.Now()
	return true, nil
}

func (m *mockRepo) Advance(_ context.Context, id string, newStatus, conclusion, conclusionType string, approvalID *string, downstreamResult *[]byte) (bool, error) {
	w, ok := m.workorders[id]
	if !ok {
		return false, nil
	}
	if !model.AdvancableStatuses[w.Status] {
		return false, nil
	}
	w.Status = newStatus
	w.Conclusion = &conclusion
	w.ConclusionType = &conclusionType
	w.ApprovalID = approvalID
	if downstreamResult != nil {
		raw := json.RawMessage(*downstreamResult)
		w.DownstreamResult = &raw
	}
	w.UpdatedAt = time.Now()
	return true, nil
}

// --- helpers ---

func setClaimsHeaders(r *http.Request, sub, role, orgID string) {
	r.Header.Set("X-Claims-Sub", sub)
	r.Header.Set("X-Claims-Role", role)
	r.Header.Set("X-Claims-Org-Id", orgID)
}

func newTestRouter(h *WorkorderHandler) *chi.Mux {
	r := chi.NewRouter()
	r.Route("/workorders", func(r chi.Router) {
		r.Post("/", h.Create)
		r.Get("/", h.List)
		r.Get("/by-asset", h.ListByAsset)
		r.Get("/{workorderId}", h.GetByID)
	})
	return r
}

func strPtr(s string) *string { return &s }

// --- Create Tests ---

func TestCreate_Success(t *testing.T) {
	repo := newMockRepo()
	h := NewWorkorderHandler(repo, nil)
	r := newTestRouter(h)

	body := `{"type":"risk","title":"Risk alert on asset X"}`
	req := httptest.NewRequest("POST", "/workorders", strings.NewReader(body))
	setClaimsHeaders(req, "user-1", "Platform", "org-1")
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != 201 {
		t.Fatalf("expected 201, got %d: %s", w.Code, w.Body.String())
	}

	var resp model.Workorder
	json.Unmarshal(w.Body.Bytes(), &resp)
	if resp.Type != "risk" {
		t.Errorf("type = %s, want risk", resp.Type)
	}
	if resp.Status != "open" {
		t.Errorf("status = %s, want open", resp.Status)
	}
	if resp.ID == "" {
		t.Error("expected non-empty ID")
	}
}

func TestCreate_InvalidType(t *testing.T) {
	repo := newMockRepo()
	h := NewWorkorderHandler(repo, nil)
	r := newTestRouter(h)

	body := `{"type":"unknown","title":"Bad type"}`
	req := httptest.NewRequest("POST", "/workorders", strings.NewReader(body))
	setClaimsHeaders(req, "user-1", "Platform", "org-1")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != 400 {
		t.Fatalf("expected 400, got %d", w.Code)
	}
}

func TestCreate_RecoveryMissingAssetID(t *testing.T) {
	repo := newMockRepo()
	h := NewWorkorderHandler(repo, nil)
	r := newTestRouter(h)

	body := `{"type":"recovery","title":"Recover asset"}`
	req := httptest.NewRequest("POST", "/workorders", strings.NewReader(body))
	setClaimsHeaders(req, "user-1", "Platform", "org-1")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != 400 {
		t.Fatalf("expected 400, got %d", w.Code)
	}
}

func TestCreate_RecoveryWithAssetID(t *testing.T) {
	repo := newMockRepo()
	h := NewWorkorderHandler(repo, nil)
	r := newTestRouter(h)

	body := `{"type":"recovery","title":"Recover asset","asset_id":"asset-123"}`
	req := httptest.NewRequest("POST", "/workorders", strings.NewReader(body))
	setClaimsHeaders(req, "user-1", "Platform", "org-1")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != 201 {
		t.Fatalf("expected 201, got %d: %s", w.Code, w.Body.String())
	}
}

func TestCreate_MissingClaims(t *testing.T) {
	repo := newMockRepo()
	h := NewWorkorderHandler(repo, nil)
	r := newTestRouter(h)

	body := `{"type":"risk","title":"No auth"}`
	req := httptest.NewRequest("POST", "/workorders", strings.NewReader(body))
	// no claims headers
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != 401 {
		t.Fatalf("expected 401, got %d", w.Code)
	}
}

// --- List Tests ---

func TestList_PlatformGetsAll(t *testing.T) {
	repo := newMockRepo()
	repo.workorders["w1"] = &model.Workorder{ID: "w1", Type: "risk", Status: "open", CreatorOrgID: "org-1"}
	repo.workorders["w2"] = &model.Workorder{ID: "w2", Type: "dispute", Status: "open", CreatorOrgID: "org-2"}
	h := NewWorkorderHandler(repo, nil)
	r := newTestRouter(h)

	req := httptest.NewRequest("GET", "/workorders", nil)
	setClaimsHeaders(req, "admin-1", "Platform", "org-0")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != 200 {
		t.Fatalf("expected 200, got %d", w.Code)
	}

	var body ListBody
	json.Unmarshal(w.Body.Bytes(), &body)
	if body.Total != 2 {
		t.Errorf("total = %d, want 2", body.Total)
	}
}

func TestList_BrandOnlyOwnOrg(t *testing.T) {
	repo := newMockRepo()
	repo.workorders["w1"] = &model.Workorder{ID: "w1", Type: "risk", Status: "open", CreatorOrgID: "org-1"}
	repo.workorders["w2"] = &model.Workorder{ID: "w2", Type: "dispute", Status: "open", CreatorOrgID: "org-2"}
	h := NewWorkorderHandler(repo, nil)
	r := newTestRouter(h)

	req := httptest.NewRequest("GET", "/workorders", nil)
	setClaimsHeaders(req, "user-1", "Brand", "org-1")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != 200 {
		t.Fatalf("expected 200, got %d", w.Code)
	}

	var body ListBody
	json.Unmarshal(w.Body.Bytes(), &body)
	if body.Total != 1 {
		t.Errorf("total = %d, want 1", body.Total)
	}
}

func TestList_BrandSeesMatchingBrandID(t *testing.T) {
	repo := newMockRepo()
	// workorder created by different org, but brand_id matches the querying Brand's org
	repo.workorders["w1"] = &model.Workorder{ID: "w1", Type: "risk", Status: "open", CreatorOrgID: "org-platform", BrandID: strPtr("org-brand-1")}
	repo.workorders["w2"] = &model.Workorder{ID: "w2", Type: "dispute", Status: "open", CreatorOrgID: "org-other"}
	h := NewWorkorderHandler(repo, nil)
	r := newTestRouter(h)

	req := httptest.NewRequest("GET", "/workorders", nil)
	setClaimsHeaders(req, "user-1", "Brand", "org-brand-1")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != 200 {
		t.Fatalf("expected 200, got %d", w.Code)
	}

	var body ListBody
	json.Unmarshal(w.Body.Bytes(), &body)
	if body.Total != 1 {
		t.Errorf("total = %d, want 1 (brand_id match)", body.Total)
	}
}

// --- GetByID Tests ---

func TestGetByID_Found(t *testing.T) {
	repo := newMockRepo()
	repo.workorders["w1"] = &model.Workorder{ID: "w1", Status: "open", CreatorOrgID: "org-1"}
	h := NewWorkorderHandler(repo, nil)
	r := newTestRouter(h)

	req := httptest.NewRequest("GET", "/workorders/w1", nil)
	setClaimsHeaders(req, "user-1", "Platform", "org-0")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != 200 {
		t.Fatalf("expected 200, got %d", w.Code)
	}
}

func TestGetByID_NotFound(t *testing.T) {
	repo := newMockRepo()
	h := NewWorkorderHandler(repo, nil)
	r := newTestRouter(h)

	req := httptest.NewRequest("GET", "/workorders/nonexistent", nil)
	setClaimsHeaders(req, "user-1", "Platform", "org-0")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != 404 {
		t.Fatalf("expected 404, got %d", w.Code)
	}
}

func TestGetByID_BrandForbidden(t *testing.T) {
	repo := newMockRepo()
	repo.workorders["w1"] = &model.Workorder{ID: "w1", Status: "open", CreatorOrgID: "org-1"}
	h := NewWorkorderHandler(repo, nil)
	r := newTestRouter(h)

	req := httptest.NewRequest("GET", "/workorders/w1", nil)
	setClaimsHeaders(req, "user-2", "Brand", "org-2") // different org
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != 403 {
		t.Fatalf("expected 403, got %d", w.Code)
	}
}

func TestGetByID_BrandAllowedByCreatorOrg(t *testing.T) {
	repo := newMockRepo()
	repo.workorders["w1"] = &model.Workorder{ID: "w1", Status: "open", CreatorOrgID: "org-1"}
	h := NewWorkorderHandler(repo, nil)
	r := newTestRouter(h)

	req := httptest.NewRequest("GET", "/workorders/w1", nil)
	setClaimsHeaders(req, "user-1", "Brand", "org-1") // same org
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != 200 {
		t.Fatalf("expected 200, got %d", w.Code)
	}
}

func TestGetByID_BrandAllowedByBrandID(t *testing.T) {
	repo := newMockRepo()
	repo.workorders["w1"] = &model.Workorder{ID: "w1", Status: "open", CreatorOrgID: "org-platform", BrandID: strPtr("org-brand")}
	h := NewWorkorderHandler(repo, nil)
	r := newTestRouter(h)

	req := httptest.NewRequest("GET", "/workorders/w1", nil)
	setClaimsHeaders(req, "user-1", "Brand", "org-brand") // brand_id matches
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != 200 {
		t.Fatalf("expected 200, got %d", w.Code)
	}
}

// --- ListByAsset Tests ---

func TestListByAsset_Success(t *testing.T) {
	repo := newMockRepo()
	repo.workorders["w1"] = &model.Workorder{ID: "w1", AssetID: strPtr("asset-1")}
	repo.workorders["w2"] = &model.Workorder{ID: "w2", AssetID: strPtr("asset-2")}
	h := NewWorkorderHandler(repo, nil)
	r := newTestRouter(h)

	req := httptest.NewRequest("GET", "/workorders/by-asset?asset_id=asset-1", nil)
	req.Header.Set("X-Claims-Sub", "user-1")
	req.Header.Set("X-Claims-Role", "Platform")
	req.Header.Set("X-Claims-Org-Id", "org-1")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != 200 {
		t.Fatalf("expected 200, got %d", w.Code)
	}

	var items []model.Workorder
	json.Unmarshal(w.Body.Bytes(), &items)
	if len(items) != 1 {
		t.Errorf("got %d items, want 1", len(items))
	}
}

func TestListByAsset_MissingAssetID(t *testing.T) {
	repo := newMockRepo()
	h := NewWorkorderHandler(repo, nil)
	r := newTestRouter(h)

	req := httptest.NewRequest("GET", "/workorders/by-asset", nil)
	req.Header.Set("X-Claims-Sub", "user-1")
	req.Header.Set("X-Claims-Role", "Platform")
	req.Header.Set("X-Claims-Org-Id", "org-1")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != 400 {
		t.Fatalf("expected 400, got %d", w.Code)
	}
}


// --- PBT Property 2: TestInvalidWorkorderTypeRejection ---

func TestInvalidWorkorderTypeRejection(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		repo := newMockRepo()
		h := NewWorkorderHandler(repo, nil)
		r := newTestRouter(h)

		// 生成不在 ValidTypes 中的随机 type
		tp := rapid.StringMatching(`[a-z]{3,10}`).Draw(t, "type")
		if model.ValidTypes[tp] {
			t.Skip()
		}

		body := `{"type":"` + tp + `","title":"pbt"}`
		req := httptest.NewRequest("POST", "/workorders", strings.NewReader(body))
		setClaimsHeaders(req, "u", "Platform", "o")
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)

		if w.Code != 400 {
			t.Fatalf("expected 400 for invalid type %q, got %d", tp, w.Code)
		}
	})
}

// --- PBT Property 6: TestRecoveryRequiresAssetID ---

func TestRecoveryRequiresAssetID(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		repo := newMockRepo()
		h := NewWorkorderHandler(repo, nil)
		r := newTestRouter(h)

		title := rapid.StringMatching(`[A-Za-z ]{1,20}`).Draw(t, "title")
		body := `{"type":"recovery","title":"` + title + `"}`
		req := httptest.NewRequest("POST", "/workorders", strings.NewReader(body))
		setClaimsHeaders(req, "u", "Platform", "o")
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)

		if w.Code != 400 {
			t.Fatalf("recovery without asset_id should be 400, got %d", w.Code)
		}
	})
}

// --- PBT Property 8: TestBrandDataIsolation ---

func TestBrandDataIsolation(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		repo := newMockRepo()

		orgA := rapid.StringMatching(`org-[a-z]{3}`).Draw(t, "orgA")
		orgB := rapid.StringMatching(`org-[a-z]{3}`).Draw(t, "orgB")
		if orgA == orgB {
			t.Skip()
		}

		// 创建属于 orgA 的工单
		repo.workorders["w-iso"] = &model.Workorder{
			ID: "w-iso", Type: "risk", Status: "open", CreatorOrgID: orgA,
		}

		h := NewWorkorderHandler(repo, nil)
		r := newTestRouter(h)

		// Brand orgB 查看 → 403
		req := httptest.NewRequest("GET", "/workorders/w-iso", nil)
		setClaimsHeaders(req, "u", "Brand", orgB)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)

		if w.Code != 403 {
			t.Fatalf("Brand %s accessing org %s workorder: expected 403, got %d", orgB, orgA, w.Code)
		}
	})
}


// --- Assign Tests ---

func newFullTestRouter(h *WorkorderHandler) *chi.Mux {
	r := chi.NewRouter()
	r.Route("/workorders", func(r chi.Router) {
		r.Post("/", h.Create)
		r.Get("/", h.List)
		r.Get("/by-asset", h.ListByAsset)
		r.Get("/{workorderId}", h.GetByID)
		r.Post("/{workorderId}/assign", h.Assign)
		r.Post("/{workorderId}/advance", h.Advance)
		r.Post("/{workorderId}/close", h.Close)
		r.Post("/{workorderId}/cancel", h.Cancel)
	})
	return r
}

func TestAssign_OpenToAssigned(t *testing.T) {
	repo := newMockRepo()
	repo.workorders["w1"] = &model.Workorder{ID: "w1", Status: "open", CreatorOrgID: "org-1"}
	h := NewWorkorderHandler(repo, nil)
	r := newFullTestRouter(h)

	body := `{"assignee_id":"mod-1","assignee_role":"Moderator"}`
	req := httptest.NewRequest("POST", "/workorders/w1/assign", strings.NewReader(body))
	setClaimsHeaders(req, "admin-1", "Platform", "org-0")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != 200 {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
	var wo model.Workorder
	json.Unmarshal(w.Body.Bytes(), &wo)
	if wo.Status != "assigned" {
		t.Errorf("status = %s, want assigned", wo.Status)
	}
}

func TestAssign_Reassign(t *testing.T) {
	repo := newMockRepo()
	assignee := "mod-1"
	role := "Moderator"
	repo.workorders["w1"] = &model.Workorder{ID: "w1", Status: "assigned", CreatorOrgID: "org-1", AssigneeID: &assignee, AssigneeRole: &role}
	h := NewWorkorderHandler(repo, nil)
	r := newFullTestRouter(h)

	body := `{"assignee_id":"mod-2","assignee_role":"Moderator"}`
	req := httptest.NewRequest("POST", "/workorders/w1/assign", strings.NewReader(body))
	setClaimsHeaders(req, "admin-1", "Platform", "org-0")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != 200 {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
}

func TestAssign_InProgressConflict(t *testing.T) {
	repo := newMockRepo()
	repo.workorders["w1"] = &model.Workorder{ID: "w1", Status: "in_progress", CreatorOrgID: "org-1"}
	h := NewWorkorderHandler(repo, nil)
	r := newFullTestRouter(h)

	body := `{"assignee_id":"mod-1","assignee_role":"Moderator"}`
	req := httptest.NewRequest("POST", "/workorders/w1/assign", strings.NewReader(body))
	setClaimsHeaders(req, "admin-1", "Platform", "org-0")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != 409 {
		t.Fatalf("expected 409, got %d", w.Code)
	}
}

func TestAssign_NonManagerForbidden(t *testing.T) {
	repo := newMockRepo()
	repo.workorders["w1"] = &model.Workorder{ID: "w1", Status: "open", CreatorOrgID: "org-1"}
	h := NewWorkorderHandler(repo, nil)
	r := newFullTestRouter(h)

	body := `{"assignee_id":"mod-1","assignee_role":"Moderator"}`
	req := httptest.NewRequest("POST", "/workorders/w1/assign", strings.NewReader(body))
	setClaimsHeaders(req, "user-1", "Brand", "org-1")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != 403 {
		t.Fatalf("expected 403, got %d", w.Code)
	}
}

// --- Advance Tests ---

func TestAdvance_FreezeSuccess(t *testing.T) {
	repo := newMockRepo()
	asset := "asset-1"
	assignee := "mod-1"
	arole := "Moderator"
	repo.workorders["w1"] = &model.Workorder{ID: "w1", Status: "assigned", CreatorOrgID: "org-1", AssetID: &asset, AssigneeID: &assignee, AssigneeRole: &arole}

	// mock rc-api: freeze → success
	rcSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
		w.Write([]byte(`{"ok":true}`))
	}))
	defer rcSrv.Close()
	rcApi := downstream.NewRcApiClient(rcSrv.URL)

	h := NewWorkorderHandler(repo, rcApi)
	r := newFullTestRouter(h)

	body := `{"conclusion":"freezing due to risk","conclusion_type":"freeze"}`
	req := httptest.NewRequest("POST", "/workorders/w1/advance", strings.NewReader(body))
	setClaimsHeaders(req, "mod-1", "Moderator", "org-0")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != 200 {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
	var wo model.Workorder
	json.Unmarshal(w.Body.Bytes(), &wo)
	if wo.Status != "resolved" {
		t.Errorf("status = %s, want resolved", wo.Status)
	}
}

func TestAdvance_FreezeFailure_InProgress(t *testing.T) {
	repo := newMockRepo()
	asset := "asset-1"
	assignee := "mod-1"
	arole := "Moderator"
	repo.workorders["w1"] = &model.Workorder{ID: "w1", Status: "assigned", CreatorOrgID: "org-1", AssetID: &asset, AssigneeID: &assignee, AssigneeRole: &arole}

	rcSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(500)
		w.Write([]byte(`{"error":"internal"}`))
	}))
	defer rcSrv.Close()
	rcApi := downstream.NewRcApiClient(rcSrv.URL)

	h := NewWorkorderHandler(repo, rcApi)
	r := newFullTestRouter(h)

	body := `{"conclusion":"freeze attempt","conclusion_type":"freeze"}`
	req := httptest.NewRequest("POST", "/workorders/w1/advance", strings.NewReader(body))
	setClaimsHeaders(req, "mod-1", "Moderator", "org-0")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != 200 {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
	var wo model.Workorder
	json.Unmarshal(w.Body.Bytes(), &wo)
	if wo.Status != "in_progress" {
		t.Errorf("status = %s, want in_progress (freeze failed)", wo.Status)
	}
}

func TestAdvance_RecoverWithApproval(t *testing.T) {
	repo := newMockRepo()
	asset := "asset-1"
	assignee := "mod-1"
	arole := "Moderator"
	repo.workorders["w1"] = &model.Workorder{ID: "w1", Status: "in_progress", CreatorOrgID: "org-1", AssetID: &asset, AssigneeID: &assignee, AssigneeRole: &arole}

	rcSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
		w.Write([]byte(`{"ok":true}`))
	}))
	defer rcSrv.Close()
	rcApi := downstream.NewRcApiClient(rcSrv.URL)

	h := NewWorkorderHandler(repo, rcApi)
	r := newFullTestRouter(h)

	body := `{"conclusion":"recovering asset","conclusion_type":"recover"}`
	req := httptest.NewRequest("POST", "/workorders/w1/advance", strings.NewReader(body))
	setClaimsHeaders(req, "mod-1", "Moderator", "org-0")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != 200 {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
	var wo model.Workorder
	json.Unmarshal(w.Body.Bytes(), &wo)
	if wo.Status != "resolved" {
		t.Errorf("status = %s, want resolved", wo.Status)
	}
}

func TestAdvance_DismissResolved(t *testing.T) {
	repo := newMockRepo()
	assignee := "mod-1"
	arole := "Moderator"
	repo.workorders["w1"] = &model.Workorder{ID: "w1", Status: "assigned", CreatorOrgID: "org-1", AssigneeID: &assignee, AssigneeRole: &arole}

	h := NewWorkorderHandler(repo, nil)
	r := newFullTestRouter(h)

	body := `{"conclusion":"not a real issue","conclusion_type":"dismiss"}`
	req := httptest.NewRequest("POST", "/workorders/w1/advance", strings.NewReader(body))
	setClaimsHeaders(req, "mod-1", "Moderator", "org-0")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != 200 {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
	var wo model.Workorder
	json.Unmarshal(w.Body.Bytes(), &wo)
	if wo.Status != "resolved" {
		t.Errorf("status = %s, want resolved", wo.Status)
	}
}

func TestAdvance_NonAssigneeForbidden(t *testing.T) {
	repo := newMockRepo()
	assignee := "mod-1"
	arole := "Moderator"
	repo.workorders["w1"] = &model.Workorder{ID: "w1", Status: "assigned", CreatorOrgID: "org-1", AssigneeID: &assignee, AssigneeRole: &arole}

	h := NewWorkorderHandler(repo, nil)
	r := newFullTestRouter(h)

	body := `{"conclusion":"x","conclusion_type":"dismiss"}`
	req := httptest.NewRequest("POST", "/workorders/w1/advance", strings.NewReader(body))
	setClaimsHeaders(req, "other-user", "Brand", "org-2")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != 403 {
		t.Fatalf("expected 403, got %d", w.Code)
	}
}

func TestAdvance_OpenStatusConflict(t *testing.T) {
	repo := newMockRepo()
	repo.workorders["w1"] = &model.Workorder{ID: "w1", Status: "open", CreatorOrgID: "org-1"}

	h := NewWorkorderHandler(repo, nil)
	r := newFullTestRouter(h)

	body := `{"conclusion":"x","conclusion_type":"dismiss"}`
	req := httptest.NewRequest("POST", "/workorders/w1/advance", strings.NewReader(body))
	setClaimsHeaders(req, "admin-1", "Platform", "org-0")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != 409 {
		t.Fatalf("expected 409, got %d", w.Code)
	}
}

// --- Close Tests ---

func TestClose_ResolvedToClosed(t *testing.T) {
	repo := newMockRepo()
	repo.workorders["w1"] = &model.Workorder{ID: "w1", Status: "resolved", CreatorOrgID: "org-1"}
	h := NewWorkorderHandler(repo, nil)
	r := newFullTestRouter(h)

	req := httptest.NewRequest("POST", "/workorders/w1/close", nil)
	setClaimsHeaders(req, "admin-1", "Platform", "org-0")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != 200 {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
	var wo model.Workorder
	json.Unmarshal(w.Body.Bytes(), &wo)
	if wo.Status != "closed" {
		t.Errorf("status = %s, want closed", wo.Status)
	}
}

func TestClose_NotResolvedConflict(t *testing.T) {
	repo := newMockRepo()
	repo.workorders["w1"] = &model.Workorder{ID: "w1", Status: "open", CreatorOrgID: "org-1"}
	h := NewWorkorderHandler(repo, nil)
	r := newFullTestRouter(h)

	req := httptest.NewRequest("POST", "/workorders/w1/close", nil)
	setClaimsHeaders(req, "admin-1", "Platform", "org-0")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != 409 {
		t.Fatalf("expected 409, got %d", w.Code)
	}
}

func TestClose_NonManagerForbidden(t *testing.T) {
	repo := newMockRepo()
	repo.workorders["w1"] = &model.Workorder{ID: "w1", Status: "resolved", CreatorOrgID: "org-1"}
	h := NewWorkorderHandler(repo, nil)
	r := newFullTestRouter(h)

	req := httptest.NewRequest("POST", "/workorders/w1/close", nil)
	setClaimsHeaders(req, "user-1", "Brand", "org-1")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != 403 {
		t.Fatalf("expected 403, got %d", w.Code)
	}
}

// --- Cancel Tests ---

func TestCancel_OpenToCancelled(t *testing.T) {
	repo := newMockRepo()
	repo.workorders["w1"] = &model.Workorder{ID: "w1", Status: "open", CreatorOrgID: "org-1"}
	h := NewWorkorderHandler(repo, nil)
	r := newFullTestRouter(h)

	req := httptest.NewRequest("POST", "/workorders/w1/cancel", nil)
	setClaimsHeaders(req, "admin-1", "Platform", "org-0")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != 200 {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
	var wo model.Workorder
	json.Unmarshal(w.Body.Bytes(), &wo)
	if wo.Status != "cancelled" {
		t.Errorf("status = %s, want cancelled", wo.Status)
	}
}

func TestCancel_AssignedToCancelled(t *testing.T) {
	repo := newMockRepo()
	repo.workorders["w1"] = &model.Workorder{ID: "w1", Status: "assigned", CreatorOrgID: "org-1"}
	h := NewWorkorderHandler(repo, nil)
	r := newFullTestRouter(h)

	req := httptest.NewRequest("POST", "/workorders/w1/cancel", nil)
	setClaimsHeaders(req, "admin-1", "Platform", "org-0")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != 200 {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
}

func TestCancel_InProgressConflict(t *testing.T) {
	repo := newMockRepo()
	repo.workorders["w1"] = &model.Workorder{ID: "w1", Status: "in_progress", CreatorOrgID: "org-1"}
	h := NewWorkorderHandler(repo, nil)
	r := newFullTestRouter(h)

	req := httptest.NewRequest("POST", "/workorders/w1/cancel", nil)
	setClaimsHeaders(req, "admin-1", "Platform", "org-0")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != 409 {
		t.Fatalf("expected 409, got %d", w.Code)
	}
}

func TestCancel_NonManagerForbidden(t *testing.T) {
	repo := newMockRepo()
	repo.workorders["w1"] = &model.Workorder{ID: "w1", Status: "open", CreatorOrgID: "org-1"}
	h := NewWorkorderHandler(repo, nil)
	r := newFullTestRouter(h)

	req := httptest.NewRequest("POST", "/workorders/w1/cancel", nil)
	setClaimsHeaders(req, "user-1", "Brand", "org-1")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != 403 {
		t.Fatalf("expected 403, got %d", w.Code)
	}
}

// --- PBT Property 3: TestTerminalStatusImmutability ---

func TestTerminalStatusImmutability(t *testing.T) {
	terminalStatuses := []string{"closed", "cancelled"}
	actions := []struct {
		name string
		path string
		body string
	}{
		{"assign", "/assign", `{"assignee_id":"x","assignee_role":"Moderator"}`},
		{"advance", "/advance", `{"conclusion":"x","conclusion_type":"dismiss"}`},
		{"close", "/close", ""},
		{"cancel", "/cancel", ""},
	}

	for _, ts := range terminalStatuses {
		for _, act := range actions {
			t.Run(ts+"_"+act.name, func(t *testing.T) {
				repo := newMockRepo()
				repo.workorders["w1"] = &model.Workorder{ID: "w1", Status: ts, CreatorOrgID: "org-1"}
				h := NewWorkorderHandler(repo, nil)
				r := newFullTestRouter(h)

				var bodyReader *strings.Reader
				if act.body != "" {
					bodyReader = strings.NewReader(act.body)
				} else {
					bodyReader = strings.NewReader("")
				}
				req := httptest.NewRequest("POST", "/workorders/w1"+act.path, bodyReader)
				setClaimsHeaders(req, "admin-1", "Platform", "org-0")
				w := httptest.NewRecorder()
				r.ServeHTTP(w, req)

				// 终态工单的任何操作都应该返回冲突或被拒绝（非200）
				if w.Code == 200 {
					t.Fatalf("terminal status %s should not allow %s, got 200", ts, act.name)
				}
			})
		}
	}
}

// --- PBT Property 4: TestAdvanceRoleConstraint ---

func TestAdvanceRoleConstraint(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		role := rapid.SampledFrom([]string{"Brand", "Consumer", "Factory"}).Draw(t, "role")
		repo := newMockRepo()
		assignee := "mod-1"
		arole := "Moderator"
		repo.workorders["w1"] = &model.Workorder{ID: "w1", Status: "assigned", CreatorOrgID: "org-1", AssigneeID: &assignee, AssigneeRole: &arole}
		h := NewWorkorderHandler(repo, nil)
		r := newFullTestRouter(h)

		body := `{"conclusion":"x","conclusion_type":"dismiss"}`
		req := httptest.NewRequest("POST", "/workorders/w1/advance", strings.NewReader(body))
		setClaimsHeaders(req, "random-user", role, "org-99")
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)

		if w.Code != 403 {
			t.Fatalf("role %s (not assignee) should be 403, got %d", role, w.Code)
		}
	})
}

// --- PBT Property 5: TestManagerRoleConstraint ---

func TestManagerRoleConstraint(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		role := rapid.SampledFrom([]string{"Brand", "Consumer", "Factory"}).Draw(t, "role")

		ops := []struct {
			name string
			path string
			body string
		}{
			{"assign", "/assign", `{"assignee_id":"x","assignee_role":"Moderator"}`},
			{"close", "/close", ""},
			{"cancel", "/cancel", ""},
		}

		for _, op := range ops {
			repo := newMockRepo()
			repo.workorders["w1"] = &model.Workorder{ID: "w1", Status: "open", CreatorOrgID: "org-1"}
			if op.name == "close" {
				repo.workorders["w1"].Status = "resolved"
			}
			h := NewWorkorderHandler(repo, nil)
			r := newFullTestRouter(h)

			req := httptest.NewRequest("POST", "/workorders/w1"+op.path, strings.NewReader(op.body))
			setClaimsHeaders(req, "user-1", role, "org-1")
			w := httptest.NewRecorder()
			r.ServeHTTP(w, req)

			if w.Code != 403 {
				t.Fatalf("role %s on %s: expected 403, got %d", role, op.name, w.Code)
			}
		}
	})
}

// --- PBT Property 7: TestConclusionTypeEnumValidation ---

func TestConclusionTypeEnumValidation(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		ct := rapid.StringMatching(`[a-z_]{3,15}`).Draw(t, "conclusionType")
		if model.ValidConclusionTypes[ct] {
			t.Skip()
		}

		repo := newMockRepo()
		assignee := "mod-1"
		arole := "Moderator"
		repo.workorders["w1"] = &model.Workorder{ID: "w1", Status: "assigned", CreatorOrgID: "org-1", AssigneeID: &assignee, AssigneeRole: &arole}
		h := NewWorkorderHandler(repo, nil)
		r := newFullTestRouter(h)

		body := `{"conclusion":"x","conclusion_type":"` + ct + `"}`
		req := httptest.NewRequest("POST", "/workorders/w1/advance", strings.NewReader(body))
		setClaimsHeaders(req, "mod-1", "Moderator", "org-0")
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)

		if w.Code != 400 {
			t.Fatalf("invalid conclusion_type %q should be 400, got %d", ct, w.Code)
		}
	})
}

// --- PBT Property 10: TestAssignStatusConstraint ---

func TestAssignStatusConstraint(t *testing.T) {
	nonAssignableStatuses := []string{"in_progress", "resolved", "closed", "cancelled"}
	for _, status := range nonAssignableStatuses {
		t.Run(status, func(t *testing.T) {
			repo := newMockRepo()
			repo.workorders["w1"] = &model.Workorder{ID: "w1", Status: status, CreatorOrgID: "org-1"}
			h := NewWorkorderHandler(repo, nil)
			r := newFullTestRouter(h)

			body := `{"assignee_id":"mod-1","assignee_role":"Moderator"}`
			req := httptest.NewRequest("POST", "/workorders/w1/assign", strings.NewReader(body))
			setClaimsHeaders(req, "admin-1", "Platform", "org-0")
			w := httptest.NewRecorder()
			r.ServeHTTP(w, req)

			if w.Code != 409 {
				t.Fatalf("status %s should not allow assign, expected 409, got %d", status, w.Code)
			}
		})
	}
}
