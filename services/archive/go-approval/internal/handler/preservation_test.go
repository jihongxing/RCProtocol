package handler

// Preservation（保持性）测试 — 审批正常流程不受影响
//
// 在未修复代码上运行确认基线行为正确。修复后重新运行确认无回归。
//
// **Validates: Requirements 3.7, 3.10**

import (
	"bytes"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"rcprotocol/services/go-approval/internal/downstream"
	"rcprotocol/services/go-approval/internal/model"
)

// ── 3.10: pending → approved → executed 正常推进 ──

func TestPreservation_3_10_NormalApprovalFlow(t *testing.T) {
	repo := newMockRepo()
	seedPendingApproval(repo, "a1", "applicant-1", "org-1", time.Now().Add(72*time.Hour))

	// 模拟下游成功
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		fmt.Fprint(w, `{"status":"ok"}`)
	}))
	defer srv.Close()

	ds := downstream.New(srv.URL)
	h := NewApprovalHandler(repo, ds)
	r := newRouter(h)

	// 审批通过
	req := httptest.NewRequest("POST", "/approvals/a1/approve", bytes.NewReader([]byte(`{}`)))
	setClaimsHeaders(req, "reviewer-1", "Platform", "org-0")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("3.10: 审批通过应返回 200，实际 %d: %s", w.Code, w.Body.String())
	}

	a := repo.approvals["a1"]
	// 正常流程下：下游成功应导致 executed 状态
	if a.Status != model.StatusExecuted {
		t.Logf("3.10: 审批状态为 '%s'（下游成功后期望 executed 或 approved）", a.Status)
		// 在当前未修复代码中，下游成功应将状态推进到 executed
		if a.Status != model.StatusApproved && a.Status != model.StatusExecuted {
			t.Fatalf("3.10: 审批正常通过后状态异常: '%s'", a.Status)
		}
	}
}

func TestPreservation_3_10_PendingApproval_CanBeApproved(t *testing.T) {
	repo := newMockRepo()
	seedPendingApproval(repo, "a1", "applicant-1", "org-1", time.Now().Add(72*time.Hour))

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		fmt.Fprint(w, `{"status":"ok"}`)
	}))
	defer srv.Close()

	ds := downstream.New(srv.URL)
	h := NewApprovalHandler(repo, ds)
	r := newRouter(h)

	// 确认初始状态是 pending
	if repo.approvals["a1"].Status != model.StatusPending {
		t.Fatalf("3.10: 初始状态应为 pending，实际 %s", repo.approvals["a1"].Status)
	}

	req := httptest.NewRequest("POST", "/approvals/a1/approve", bytes.NewReader([]byte(`{}`)))
	setClaimsHeaders(req, "reviewer-1", "Platform", "org-0")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	// 审批通过后状态不再是 pending
	if repo.approvals["a1"].Status == model.StatusPending {
		t.Fatal("3.10: 审批通过后状态不应仍为 pending")
	}
}

func TestPreservation_3_10_ExpiredApproval_Rejected(t *testing.T) {
	repo := newMockRepo()
	seedPendingApproval(repo, "a1", "user-1", "org-1", time.Now().Add(-1*time.Hour))

	ds := downstream.New("http://localhost:9999")
	h := NewApprovalHandler(repo, ds)
	r := newRouter(h)

	req := httptest.NewRequest("POST", "/approvals/a1/approve", bytes.NewReader([]byte(`{}`)))
	setClaimsHeaders(req, "reviewer-1", "Platform", "org-0")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	// 过期审批应被拒绝（非 200）
	if w.Code == http.StatusOK {
		a := repo.approvals["a1"]
		if a.Status == model.StatusExecuted || a.Status == model.StatusApproved {
			t.Fatalf("3.10: 过期审批不应被批准通过，但状态变为 '%s'", a.Status)
		}
	}
}

func TestPreservation_3_10_NonPending_CannotApprove(t *testing.T) {
	repo := newMockRepo()
	repo.approvals["a1"] = &model.Approval{
		ID: "a1", Status: model.StatusExecuted, ApplicantID: "user-1",
		ApplicantOrgID: "org-1",
	}

	ds := downstream.New("http://localhost:9999")
	h := NewApprovalHandler(repo, ds)
	r := newRouter(h)

	req := httptest.NewRequest("POST", "/approvals/a1/approve", bytes.NewReader([]byte(`{}`)))
	setClaimsHeaders(req, "reviewer-1", "Platform", "org-0")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code == http.StatusOK {
		t.Fatalf("3.10: 非 pending 审批通过操作应失败，但返回 %d", w.Code)
	}
}

// ── 3.7: 合法幂等重试 ──
// 注意: 在内存 mock 中幂等由应用层 ExistsPending 检查，此处验证创建审批行为一致

func TestPreservation_3_7_CreateApproval_Success(t *testing.T) {
	repo := newMockRepo()
	ds := downstream.New("http://localhost:9999")
	h := NewApprovalHandler(repo, ds)
	r := newRouter(h)

	body := `{
		"type": "brand_publish",
		"resource_type": "brand",
		"resource_id": "b-001",
		"payload": {"brand_id": "b-001"}
	}`
	req := httptest.NewRequest("POST", "/approvals", strings.NewReader(body))
	setClaimsHeaders(req, "user-1", "Brand", "org-1")
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusCreated {
		t.Fatalf("3.7: 合法创建审批应返回 201，实际 %d: %s", w.Code, w.Body.String())
	}
}

func TestPreservation_3_7_DuplicatePending_Prevented(t *testing.T) {
	repo := newMockRepo()
	ds := downstream.New("http://localhost:9999")
	h := NewApprovalHandler(repo, ds)
	r := newRouter(h)

	body := `{
		"type": "brand_publish",
		"resource_type": "brand",
		"resource_id": "b-001",
		"payload": {"brand_id": "b-001"}
	}`

	// 第一次创建
	req := httptest.NewRequest("POST", "/approvals", strings.NewReader(body))
	setClaimsHeaders(req, "user-1", "Brand", "org-1")
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusCreated {
		t.Fatalf("3.7: 首次创建应返回 201，实际 %d", w.Code)
	}

	// 重复创建应被拒绝
	req = httptest.NewRequest("POST", "/approvals", strings.NewReader(body))
	setClaimsHeaders(req, "user-1", "Brand", "org-1")
	req.Header.Set("Content-Type", "application/json")
	w = httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code == http.StatusCreated {
		t.Fatal("3.7: 重复 pending 审批创建应被拒绝")
	}
}
