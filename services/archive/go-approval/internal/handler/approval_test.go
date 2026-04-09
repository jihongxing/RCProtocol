package handler

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"
	"pgregory.net/rapid"
	"rcprotocol/services/go-approval/internal/downstream"
	"rcprotocol/services/go-approval/internal/model"
)

// --- mock repo ---

type mockRepo struct {
	approvals    map[string]*model.Approval
	pendingIndex map[string]bool // key: resourceType|resourceID|type
}

func newMockRepo() *mockRepo {
	return &mockRepo{
		approvals:    make(map[string]*model.Approval),
		pendingIndex: make(map[string]bool),
	}
}

func (m *mockRepo) Create(_ context.Context, a *model.Approval) error {
	cp := *a
	cp.CreatedAt = time.Now()
	cp.UpdatedAt = time.Now()
	m.approvals[a.ID] = &cp
	if a.Status == model.StatusPending {
		m.pendingIndex[pendingKey(a.ResourceType, a.ResourceID, a.Type)] = true
	}
	return nil
}

func (m *mockRepo) GetByID(_ context.Context, id string) (*model.Approval, error) {
	a, ok := m.approvals[id]
	if !ok {
		return nil, nil
	}
	cp := *a
	return &cp, nil
}

func (m *mockRepo) ExistsPending(_ context.Context, resourceType, resourceID, approvalType string) (bool, error) {
	return m.pendingIndex[pendingKey(resourceType, resourceID, approvalType)], nil
}

func (m *mockRepo) List(_ context.Context, filterStatus, filterType, orgID string, page, pageSize int) ([]model.Approval, int, error) {
	var filtered []model.Approval
	for _, a := range m.approvals {
		if filterStatus != "" && a.Status != filterStatus {
			continue
		}
		if filterType != "" && a.Type != filterType {
			continue
		}
		if orgID != "" && a.ApplicantOrgID != orgID {
			continue
		}
		filtered = append(filtered, *a)
	}
	total := len(filtered)
	start := (page - 1) * pageSize
	if start >= total {
		return []model.Approval{}, total, nil
	}
	end := start + pageSize
	if end > total {
		end = total
	}
	return filtered[start:end], total, nil
}

func (m *mockRepo) ListByResource(_ context.Context, resourceType, resourceID string) ([]model.Approval, error) {
	var result []model.Approval
	for _, a := range m.approvals {
		if a.ResourceType == resourceType && a.ResourceID == resourceID {
			result = append(result, *a)
		}
	}
	return result, nil
}

func (m *mockRepo) UpdateStatus(_ context.Context, id, expectedStatus, newStatus string,
	reviewerID, reviewerRole, reviewComment *string, downstreamResult *[]byte) (bool, error) {
	a, ok := m.approvals[id]
	if !ok || a.Status != expectedStatus {
		return false, nil
	}
	oldStatus := a.Status
	a.Status = newStatus
	a.UpdatedAt = time.Now()
	if reviewerID != nil {
		a.ReviewerID = reviewerID
	}
	if reviewerRole != nil {
		a.ReviewerRole = reviewerRole
	}
	if reviewComment != nil {
		a.ReviewComment = reviewComment
	}
	if downstreamResult != nil {
		raw := json.RawMessage(*downstreamResult)
		a.DownstreamResult = &raw
	}
	// update pending index
	if oldStatus == model.StatusPending && newStatus != model.StatusPending {
		delete(m.pendingIndex, pendingKey(a.ResourceType, a.ResourceID, a.Type))
	}
	return true, nil
}

func pendingKey(rt, rid, typ string) string {
	return rt + "|" + rid + "|" + typ
}

// --- helpers ---

func setClaimsHeaders(r *http.Request, sub, role, orgID string) {
	r.Header.Set("X-Claims-Sub", sub)
	r.Header.Set("X-Claims-Role", role)
	r.Header.Set("X-Claims-Org-Id", orgID)
}

func newRouter(h *ApprovalHandler) *chi.Mux {
	r := chi.NewRouter()
	r.Route("/approvals", func(r chi.Router) {
		r.Post("/", h.Create)
		r.Get("/", h.List)
		r.Get("/by-resource", h.ListByResource)
		r.Get("/{approvalId}", h.GetByID)
		r.Post("/{approvalId}/approve", h.Approve)
		r.Post("/{approvalId}/reject", h.Reject)
	})
	return r
}

func createBody(typ, resourceType, resourceID string, payload interface{}) string {
	payloadBytes, _ := json.Marshal(payload)
	body := map[string]interface{}{
		"type":          typ,
		"resource_type": resourceType,
		"resource_id":   resourceID,
		"payload":       json.RawMessage(payloadBytes),
	}
	b, _ := json.Marshal(body)
	return string(b)
}

// --- Task 9 Tests: Create ---

func TestCreate_Success(t *testing.T) {
	repo := newMockRepo()
	ds := downstream.New("http://localhost:9999")
	h := NewApprovalHandler(repo, ds)
	r := newRouter(h)

	body := createBody(model.TypeBrandPublish, "brand", "b-001", map[string]string{"brand_id": "b-001"})
	req := httptest.NewRequest("POST", "/approvals", strings.NewReader(body))
	setClaimsHeaders(req, "user-1", "Brand", "org-1")
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != 201 {
		t.Fatalf("expected 201, got %d: %s", w.Code, w.Body.String())
	}
}

func TestCreate_InvalidType(t *testing.T) {
	repo := newMockRepo()
	ds := downstream.New("http://localhost:9999")
	h := NewApprovalHandler(repo, ds)
	r := newRouter(h)

	body := createBody("unknown_type", "brand", "b-001", map[string]string{"brand_id": "b-001"})
	req := httptest.NewRequest("POST", "/approvals", strings.NewReader(body))
	setClaimsHeaders(req, "user-1", "Brand", "org-1")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != 400 {
		t.Fatalf("expected 400, got %d", w.Code)
	}
}

func TestCreate_MissingFields(t *testing.T) {
	repo := newMockRepo()
	ds := downstream.New("http://localhost:9999")
	h := NewApprovalHandler(repo, ds)
	r := newRouter(h)

	// missing resource_id
	body := `{"type":"brand_publish","resource_type":"brand","payload":{"brand_id":"b1"}}`
	req := httptest.NewRequest("POST", "/approvals", strings.NewReader(body))
	setClaimsHeaders(req, "user-1", "Brand", "org-1")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != 400 {
		t.Fatalf("expected 400, got %d", w.Code)
	}
}

func TestCreate_DuplicatePending(t *testing.T) {
	repo := newMockRepo()
	ds := downstream.New("http://localhost:9999")
	h := NewApprovalHandler(repo, ds)
	r := newRouter(h)

	body := createBody(model.TypeBrandPublish, "brand", "b-001", map[string]string{"brand_id": "b-001"})

	// first create
	req := httptest.NewRequest("POST", "/approvals", strings.NewReader(body))
	setClaimsHeaders(req, "user-1", "Brand", "org-1")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != 201 {
		t.Fatalf("first create: expected 201, got %d", w.Code)
	}

	// duplicate
	req = httptest.NewRequest("POST", "/approvals", strings.NewReader(body))
	setClaimsHeaders(req, "user-1", "Brand", "org-1")
	w = httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != 409 {
		t.Fatalf("duplicate: expected 409, got %d", w.Code)
	}
}

func TestCreate_MissingClaims(t *testing.T) {
	repo := newMockRepo()
	ds := downstream.New("http://localhost:9999")
	h := NewApprovalHandler(repo, ds)
	r := newRouter(h)

	body := createBody(model.TypeBrandPublish, "brand", "b-001", map[string]string{"brand_id": "b-001"})
	req := httptest.NewRequest("POST", "/approvals", strings.NewReader(body))
	// no claims headers
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != 401 {
		t.Fatalf("expected 401, got %d", w.Code)
	}
}

// --- Task 9 Tests: List ---

func TestList_PlatformGetsAll(t *testing.T) {
	repo := newMockRepo()
	repo.approvals["a1"] = &model.Approval{ID: "a1", Type: model.TypeBrandPublish, Status: model.StatusPending, ApplicantOrgID: "org-1"}
	repo.approvals["a2"] = &model.Approval{ID: "a2", Type: model.TypeBrandPublish, Status: model.StatusPending, ApplicantOrgID: "org-2"}
	ds := downstream.New("http://localhost:9999")
	h := NewApprovalHandler(repo, ds)
	r := newRouter(h)

	req := httptest.NewRequest("GET", "/approvals", nil)
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
	repo.approvals["a1"] = &model.Approval{ID: "a1", Type: model.TypeBrandPublish, Status: model.StatusPending, ApplicantOrgID: "org-1"}
	repo.approvals["a2"] = &model.Approval{ID: "a2", Type: model.TypeBrandPublish, Status: model.StatusPending, ApplicantOrgID: "org-2"}
	ds := downstream.New("http://localhost:9999")
	h := NewApprovalHandler(repo, ds)
	r := newRouter(h)

	req := httptest.NewRequest("GET", "/approvals", nil)
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

// --- Task 9 Tests: GetByID ---

func TestGetByID_Found(t *testing.T) {
	repo := newMockRepo()
	repo.approvals["a1"] = &model.Approval{ID: "a1", Status: model.StatusPending, ApplicantOrgID: "org-1"}
	ds := downstream.New("http://localhost:9999")
	h := NewApprovalHandler(repo, ds)
	r := newRouter(h)

	req := httptest.NewRequest("GET", "/approvals/a1", nil)
	setClaimsHeaders(req, "user-1", "Platform", "org-0")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != 200 {
		t.Fatalf("expected 200, got %d", w.Code)
	}
}

func TestGetByID_NotFound(t *testing.T) {
	repo := newMockRepo()
	ds := downstream.New("http://localhost:9999")
	h := NewApprovalHandler(repo, ds)
	r := newRouter(h)

	req := httptest.NewRequest("GET", "/approvals/nonexistent", nil)
	setClaimsHeaders(req, "user-1", "Platform", "org-0")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != 404 {
		t.Fatalf("expected 404, got %d", w.Code)
	}
}

func TestGetByID_BrandForbidden(t *testing.T) {
	repo := newMockRepo()
	repo.approvals["a1"] = &model.Approval{ID: "a1", Status: model.StatusPending, ApplicantOrgID: "org-1"}
	ds := downstream.New("http://localhost:9999")
	h := NewApprovalHandler(repo, ds)
	r := newRouter(h)

	req := httptest.NewRequest("GET", "/approvals/a1", nil)
	setClaimsHeaders(req, "user-2", "Brand", "org-2") // different org
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != 403 {
		t.Fatalf("expected 403, got %d", w.Code)
	}
}

// --- Task 9 Tests: ListByResource ---

func TestListByResource_Success(t *testing.T) {
	repo := newMockRepo()
	repo.approvals["a1"] = &model.Approval{ID: "a1", ResourceType: "brand", ResourceID: "b1"}
	repo.approvals["a2"] = &model.Approval{ID: "a2", ResourceType: "brand", ResourceID: "b2"}
	ds := downstream.New("http://localhost:9999")
	h := NewApprovalHandler(repo, ds)
	r := newRouter(h)

	req := httptest.NewRequest("GET", "/approvals/by-resource?resource_type=brand&resource_id=b1", nil)
	req.Header.Set("X-Claims-Sub", "user-1")
	req.Header.Set("X-Claims-Role", "Platform")
	req.Header.Set("X-Claims-Org-Id", "org-1")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != 200 {
		t.Fatalf("expected 200, got %d", w.Code)
	}
}

func TestListByResource_MissingParams(t *testing.T) {
	repo := newMockRepo()
	ds := downstream.New("http://localhost:9999")
	h := NewApprovalHandler(repo, ds)
	r := newRouter(h)

	req := httptest.NewRequest("GET", "/approvals/by-resource?resource_type=brand", nil) // missing resource_id
	req.Header.Set("X-Claims-Sub", "user-1")
	req.Header.Set("X-Claims-Role", "Platform")
	req.Header.Set("X-Claims-Org-Id", "org-1")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != 400 {
		t.Fatalf("expected 400, got %d", w.Code)
	}
}

// --- Task 10 Tests: Approve ---

func seedPendingApproval(repo *mockRepo, id, applicantID, orgID string, expiresAt time.Time) {
	repo.approvals[id] = &model.Approval{
		ID:             id,
		Type:           model.TypeBrandPublish,
		Status:         model.StatusPending,
		ApplicantID:    applicantID,
		ApplicantRole:  "Brand",
		ApplicantOrgID: orgID,
		ResourceType:   "brand",
		ResourceID:     "b-" + id,
		Payload:        json.RawMessage(`{"brand_id":"b-` + id + `"}`),
		ExpiresAt:      expiresAt,
	}
	repo.pendingIndex[pendingKey("brand", "b-"+id, model.TypeBrandPublish)] = true
}

func TestApprove_Success(t *testing.T) {
	repo := newMockRepo()
	seedPendingApproval(repo, "a1", "applicant-1", "org-1", time.Now().Add(72*time.Hour))

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
		fmt.Fprint(w, `{"ok":true}`)
	}))
	defer srv.Close()

	ds := downstream.New(srv.URL)
	h := NewApprovalHandler(repo, ds)
	r := newRouter(h)

	req := httptest.NewRequest("POST", "/approvals/a1/approve", bytes.NewReader([]byte(`{}`)))
	setClaimsHeaders(req, "reviewer-1", "Platform", "org-0")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != 200 {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	a := repo.approvals["a1"]
	if a.Status != model.StatusExecuted {
		t.Errorf("status = %s, want executed", a.Status)
	}
}

func TestApprove_NonPending(t *testing.T) {
	repo := newMockRepo()
	repo.approvals["a1"] = &model.Approval{
		ID: "a1", Status: model.StatusExecuted, ApplicantID: "user-1",
		ExpiresAt: time.Now().Add(72 * time.Hour),
	}
	ds := downstream.New("http://localhost:9999")
	h := NewApprovalHandler(repo, ds)
	r := newRouter(h)

	req := httptest.NewRequest("POST", "/approvals/a1/approve", bytes.NewReader([]byte(`{}`)))
	setClaimsHeaders(req, "reviewer-1", "Platform", "org-0")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != 409 {
		t.Fatalf("expected 409, got %d", w.Code)
	}
}

func TestApprove_NonReviewerRole(t *testing.T) {
	repo := newMockRepo()
	seedPendingApproval(repo, "a1", "user-1", "org-1", time.Now().Add(72*time.Hour))
	ds := downstream.New("http://localhost:9999")
	h := NewApprovalHandler(repo, ds)
	r := newRouter(h)

	req := httptest.NewRequest("POST", "/approvals/a1/approve", bytes.NewReader([]byte(`{}`)))
	setClaimsHeaders(req, "reviewer-1", "Brand", "org-1") // Brand cannot approve
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != 403 {
		t.Fatalf("expected 403, got %d", w.Code)
	}
}

func TestApprove_Expired(t *testing.T) {
	repo := newMockRepo()
	seedPendingApproval(repo, "a1", "user-1", "org-1", time.Now().Add(-1*time.Hour)) // expired
	ds := downstream.New("http://localhost:9999")
	h := NewApprovalHandler(repo, ds)
	r := newRouter(h)

	req := httptest.NewRequest("POST", "/approvals/a1/approve", bytes.NewReader([]byte(`{}`)))
	setClaimsHeaders(req, "reviewer-1", "Platform", "org-0")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != 409 {
		t.Fatalf("expected 409, got %d: %s", w.Code, w.Body.String())
	}

	a := repo.approvals["a1"]
	if a.Status != model.StatusExpired {
		t.Errorf("status = %s, want expired", a.Status)
	}
}

func TestApprove_SelfApproval(t *testing.T) {
	repo := newMockRepo()
	seedPendingApproval(repo, "a1", "reviewer-1", "org-1", time.Now().Add(72*time.Hour))
	ds := downstream.New("http://localhost:9999")
	h := NewApprovalHandler(repo, ds)
	r := newRouter(h)

	req := httptest.NewRequest("POST", "/approvals/a1/approve", bytes.NewReader([]byte(`{}`)))
	setClaimsHeaders(req, "reviewer-1", "Platform", "org-0") // same sub as applicant
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != 403 {
		t.Fatalf("expected 403, got %d", w.Code)
	}
}

func TestApprove_ConcurrentConflict(t *testing.T) {
	repo := newMockRepo()
	seedPendingApproval(repo, "a1", "user-1", "org-1", time.Now().Add(72*time.Hour))

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
	}))
	defer srv.Close()

	ds := downstream.New(srv.URL)
	h := NewApprovalHandler(repo, ds)
	r := newRouter(h)

	// first approve
	req := httptest.NewRequest("POST", "/approvals/a1/approve", bytes.NewReader([]byte(`{}`)))
	setClaimsHeaders(req, "reviewer-1", "Platform", "org-0")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != 200 {
		t.Fatalf("first approve: expected 200, got %d", w.Code)
	}

	// second approve on same (now no longer pending)
	req = httptest.NewRequest("POST", "/approvals/a1/approve", bytes.NewReader([]byte(`{}`)))
	setClaimsHeaders(req, "reviewer-2", "Platform", "org-0")
	w = httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != 409 {
		t.Fatalf("second approve: expected 409, got %d", w.Code)
	}
}

// --- Task 10 Tests: Reject ---

func TestReject_Success(t *testing.T) {
	repo := newMockRepo()
	seedPendingApproval(repo, "a1", "user-1", "org-1", time.Now().Add(72*time.Hour))
	ds := downstream.New("http://localhost:9999")
	h := NewApprovalHandler(repo, ds)
	r := newRouter(h)

	body := `{"review_comment":"not acceptable"}`
	req := httptest.NewRequest("POST", "/approvals/a1/reject", strings.NewReader(body))
	setClaimsHeaders(req, "reviewer-1", "Platform", "org-0")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != 200 {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	a := repo.approvals["a1"]
	if a.Status != model.StatusRejected {
		t.Errorf("status = %s, want rejected", a.Status)
	}
}

func TestReject_NonPending(t *testing.T) {
	repo := newMockRepo()
	repo.approvals["a1"] = &model.Approval{
		ID: "a1", Status: model.StatusExecuted, ApplicantID: "user-1",
		ExpiresAt: time.Now().Add(72 * time.Hour),
	}
	ds := downstream.New("http://localhost:9999")
	h := NewApprovalHandler(repo, ds)
	r := newRouter(h)

	body := `{"review_comment":"no"}`
	req := httptest.NewRequest("POST", "/approvals/a1/reject", strings.NewReader(body))
	setClaimsHeaders(req, "reviewer-1", "Platform", "org-0")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != 409 {
		t.Fatalf("expected 409, got %d", w.Code)
	}
}

func TestReject_NonReviewerRole(t *testing.T) {
	repo := newMockRepo()
	seedPendingApproval(repo, "a1", "user-1", "org-1", time.Now().Add(72*time.Hour))
	ds := downstream.New("http://localhost:9999")
	h := NewApprovalHandler(repo, ds)
	r := newRouter(h)

	body := `{"review_comment":"no"}`
	req := httptest.NewRequest("POST", "/approvals/a1/reject", strings.NewReader(body))
	setClaimsHeaders(req, "reviewer-1", "Brand", "org-1") // Brand cannot reject
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != 403 {
		t.Fatalf("expected 403, got %d", w.Code)
	}
}

func TestReject_MissingComment(t *testing.T) {
	repo := newMockRepo()
	seedPendingApproval(repo, "a1", "user-1", "org-1", time.Now().Add(72*time.Hour))
	ds := downstream.New("http://localhost:9999")
	h := NewApprovalHandler(repo, ds)
	r := newRouter(h)

	body := `{}`
	req := httptest.NewRequest("POST", "/approvals/a1/reject", strings.NewReader(body))
	setClaimsHeaders(req, "reviewer-1", "Platform", "org-0")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != 400 {
		t.Fatalf("expected 400, got %d", w.Code)
	}
}

// --- Property Tests ---

// Property 2: 审批类型枚举校验
// 使用 rapid 生成不属于 ValidTypes 的随机字符串作为 type，验证返回 400
// Validates: FR-04 (4.3)
func TestInvalidApprovalTypeRejection(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		typ := rapid.StringMatching(`[a-z_]{3,20}`).Draw(t, "type")
		if model.ValidTypes[typ] {
			return // skip valid types
		}

		repo := newMockRepo()
		ds := downstream.New("http://localhost:9999")
		h := NewApprovalHandler(repo, ds)
		r := newRouter(h)

		body := fmt.Sprintf(`{"type":"%s","resource_type":"brand","resource_id":"b1","payload":{"x":"y"}}`, typ)
		req := httptest.NewRequest("POST", "/approvals", strings.NewReader(body))
		setClaimsHeaders(req, "user-1", "Brand", "org-1")
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)

		if w.Code != 400 {
			t.Fatalf("type=%s: expected 400, got %d", typ, w.Code)
		}
	})
}

// Property 8: Brand 角色数据隔离
// ��用 rapid 生成随机 org_id 的 Brand Claims，查询不属于该 org 的审批单，验证返回 403
// Validates: FR-05 (5.3), FR-06 (6.3)
func TestBrandDataIsolation(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		ownerOrg := rapid.StringMatching(`org-[a-z0-9]{3,10}`).Draw(t, "ownerOrg")
		queryOrg := rapid.StringMatching(`org-[a-z0-9]{3,10}`).Draw(t, "queryOrg")
		if ownerOrg == queryOrg {
			return // skip same org
		}

		repo := newMockRepo()
		repo.approvals["a1"] = &model.Approval{
			ID: "a1", Status: model.StatusPending, ApplicantOrgID: ownerOrg,
		}
		ds := downstream.New("http://localhost:9999")
		h := NewApprovalHandler(repo, ds)
		r := newRouter(h)

		// GetByID with different org Brand should get 403
		req := httptest.NewRequest("GET", "/approvals/a1", nil)
		setClaimsHeaders(req, "user-1", "Brand", queryOrg)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)

		if w.Code != 403 {
			t.Fatalf("ownerOrg=%s queryOrg=%s: expected 403, got %d", ownerOrg, queryOrg, w.Code)
		}
	})
}

// Property 3: 审批状态流转单向性
// 使用 rapid 随机选择终态，对该状态的审批单调用 approve/reject，验证返回 409
// Validates: NFR-03, FR-07 (7.1), FR-08 (8.1)
func TestTerminalStatusImmutability(t *testing.T) {
	terminalStatuses := []string{model.StatusExecuted, model.StatusRejected, model.StatusExpired, model.StatusFailed}

	rapid.Check(t, func(t *rapid.T) {
		statusIdx := rapid.IntRange(0, len(terminalStatuses)-1).Draw(t, "statusIdx")
		status := terminalStatuses[statusIdx]

		repo := newMockRepo()
		repo.approvals["a1"] = &model.Approval{
			ID: "a1", Status: status, ApplicantID: "user-1",
			ExpiresAt: time.Now().Add(72 * time.Hour),
		}
		ds := downstream.New("http://localhost:9999")
		h := NewApprovalHandler(repo, ds)
		r := newRouter(h)

		// try approve
		req := httptest.NewRequest("POST", "/approvals/a1/approve", bytes.NewReader([]byte(`{}`)))
		setClaimsHeaders(req, "reviewer-1", "Platform", "org-0")
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
		if w.Code != 409 {
			t.Fatalf("approve on %s: expected 409, got %d", status, w.Code)
		}

		// try reject
		req = httptest.NewRequest("POST", "/approvals/a1/reject", strings.NewReader(`{"review_comment":"no"}`))
		setClaimsHeaders(req, "reviewer-1", "Platform", "org-0")
		w = httptest.NewRecorder()
		r.ServeHTTP(w, req)
		if w.Code != 409 {
			t.Fatalf("reject on %s: expected 409, got %d", status, w.Code)
		}
	})
}

// Property 4: 审批人角色约束
// 使用 rapid 生成不属于 ReviewerRoles 的随机 role，调用 approve/reject，验证返回 403
// Validates: FR-07 (7.2), FR-08 (8.2)
func TestReviewerRoleConstraint(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		role := rapid.StringMatching(`[A-Z][a-z]{2,15}`).Draw(t, "role")
		if model.ReviewerRoles[role] {
			return // skip valid reviewer roles
		}

		repo := newMockRepo()
		seedPendingApproval(repo, "a1", "user-1", "org-1", time.Now().Add(72*time.Hour))
		ds := downstream.New("http://localhost:9999")
		h := NewApprovalHandler(repo, ds)
		r := newRouter(h)

		// try approve — invalid roles get rejected at claims layer (401) or handler layer (403)
		req := httptest.NewRequest("POST", "/approvals/a1/approve", bytes.NewReader([]byte(`{}`)))
		setClaimsHeaders(req, "reviewer-1", role, "org-0")
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
		if w.Code != 403 && w.Code != 401 {
			t.Fatalf("approve with role=%s: expected 401 or 403, got %d", role, w.Code)
		}

		// try reject
		req = httptest.NewRequest("POST", "/approvals/a1/reject", strings.NewReader(`{"review_comment":"no"}`))
		setClaimsHeaders(req, "reviewer-1", role, "org-0")
		w = httptest.NewRecorder()
		r.ServeHTTP(w, req)
		if w.Code != 403 && w.Code != 401 {
			t.Fatalf("reject with role=%s: expected 401 or 403, got %d", role, w.Code)
		}
	})
}

// Property 5: 自我审批禁止
// 使用 rapid 生成随机 user_id，创建该 user 的审批单，然后用同一 user_id 调用 approve，验证返回 403
// Validates: FR-07 (7.5)
func TestSelfApprovalPrevention(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		userID := rapid.StringMatching(`user-[a-z0-9]{3,15}`).Draw(t, "userID")

		repo := newMockRepo()
		seedPendingApproval(repo, "a1", userID, "org-1", time.Now().Add(72*time.Hour))
		ds := downstream.New("http://localhost:9999")
		h := NewApprovalHandler(repo, ds)
		r := newRouter(h)

		req := httptest.NewRequest("POST", "/approvals/a1/approve", bytes.NewReader([]byte(`{}`)))
		setClaimsHeaders(req, userID, "Platform", "org-0") // same user as applicant
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)

		if w.Code != 403 {
			t.Fatalf("self-approve userID=%s: expected 403, got %d", userID, w.Code)
		}
	})
}

// Property 6: 资源去重约束
// 使用 rapid 生成随机 resource_type + resource_id + type，创建 pending 后再创建同组合，验证返回 409
// Validates: FR-04 (4.6)
func TestPendingDuplicatePrevention(t *testing.T) {
	types := []string{model.TypeBrandPublish, model.TypePolicyApply, model.TypeRiskRecovery, model.TypeHighRiskAction}

	rapid.Check(t, func(t *rapid.T) {
		resType := rapid.StringMatching(`[a-z]{3,10}`).Draw(t, "resType")
		resID := rapid.StringMatching(`[a-z0-9\-]{3,20}`).Draw(t, "resID")
		typeIdx := rapid.IntRange(0, len(types)-1).Draw(t, "typeIdx")
		typ := types[typeIdx]

		repo := newMockRepo()
		ds := downstream.New("http://localhost:9999")
		h := NewApprovalHandler(repo, ds)
		r := newRouter(h)

		body := fmt.Sprintf(`{"type":"%s","resource_type":"%s","resource_id":"%s","payload":{"x":"y"}}`, typ, resType, resID)

		// first create
		req := httptest.NewRequest("POST", "/approvals", strings.NewReader(body))
		setClaimsHeaders(req, "user-1", "Brand", "org-1")
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
		if w.Code != 201 {
			t.Fatalf("first create: expected 201, got %d: %s", w.Code, w.Body.String())
		}

		// duplicate
		req = httptest.NewRequest("POST", "/approvals", strings.NewReader(body))
		setClaimsHeaders(req, "user-1", "Brand", "org-1")
		w = httptest.NewRecorder()
		r.ServeHTTP(w, req)
		if w.Code != 409 {
			t.Fatalf("duplicate resType=%s resID=%s type=%s: expected 409, got %d", resType, resID, typ, w.Code)
		}
	})
}

// Property 7: 过期审批不可通过
// 使用 rapid 生成已过期的审批单，调用 approve，验证状态更新为 expired 且返回 409
// Validates: FR-07 (7.3)
func TestExpiredApprovalRejection(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		hoursAgo := rapid.IntRange(1, 1000).Draw(t, "hoursAgo")
		expiresAt := time.Now().Add(-time.Duration(hoursAgo) * time.Hour)

		repo := newMockRepo()
		seedPendingApproval(repo, "a1", "user-1", "org-1", expiresAt)
		ds := downstream.New("http://localhost:9999")
		h := NewApprovalHandler(repo, ds)
		r := newRouter(h)

		req := httptest.NewRequest("POST", "/approvals/a1/approve", bytes.NewReader([]byte(`{}`)))
		setClaimsHeaders(req, "reviewer-1", "Platform", "org-0")
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)

		if w.Code != 409 {
			t.Fatalf("expired approve: expected 409, got %d", w.Code)
		}

		a := repo.approvals["a1"]
		if a.Status != model.StatusExpired {
			t.Fatalf("expected status=expired, got %s", a.Status)
		}
	})
}
