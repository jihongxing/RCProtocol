package handler

// Bug Condition 探索性测试 — 审批服务缺陷
//
// 这些测试编码了**期望行为**：在未修复代码上应当 FAIL，证明缺陷存在。
// 修复后测试通过即确认修复成功。
//
// **Validates: Requirements 1.20, 1.24, 1.25**

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

// ── BUG 1.20: ListByResource 无 Claims 校验 ──
// Bug: ListByResource 不调用 claims.FromRequest、不做 Valid() 检查。
// 期望行为: 无有效 Claims 时应返回 401。
func TestBug_1_20_ListByResource_WithoutClaims_ShouldReturn401(t *testing.T) {
	repo := newMockRepo()
	repo.approvals["a1"] = &model.Approval{
		ID: "a1", ResourceType: "brand", ResourceID: "b1",
		Status: model.StatusPending,
	}
	ds := downstream.New("http://localhost:9999")
	h := NewApprovalHandler(repo, ds)
	r := newRouter(h)

	// 故意不设置 X-Claims-* 头
	req := httptest.NewRequest("GET", "/approvals/by-resource?resource_type=brand&resource_id=b1", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code == http.StatusOK {
		t.Fatalf(
			"BUG 1.20: ListByResource 无 Claims 返回 200，应返回 401。"+
				"当前行为: 任何人可枚举所有资源的审批记录",
		)
	}
}

// ── BUG 1.24: 下游超时导致审批状态变为 failed ──
// Bug: Approve 中下游调用失败时立即将状态标记为 "failed"（终态），
//      但 rc-api 可能实际已执行成功，产生不可逆状态不一致。
// 期望行为: 下游超时应保持 "approved" 状态，不标记 "failed"。
func TestBug_1_24_DownstreamTimeout_ShouldNotMarkFailed(t *testing.T) {
	repo := newMockRepo()
	seedPendingApproval(repo, "a1", "user-1", "org-1", time.Now().Add(72*time.Hour))

	// 模拟下游超时: 返回 500 或连接失败
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusGatewayTimeout)
		fmt.Fprint(w, `{"error":"timeout"}`)
	}))
	defer srv.Close()

	ds := downstream.New(srv.URL)
	h := NewApprovalHandler(repo, ds)
	r := newRouter(h)

	req := httptest.NewRequest("POST", "/approvals/a1/approve", bytes.NewReader([]byte(`{}`)))
	setClaimsHeaders(req, "reviewer-1", "Platform", "org-0")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	a := repo.approvals["a1"]

	// 期望: 状态应保持 "approved"，不应变为 "failed"
	if a.Status == model.StatusFailed {
		t.Fatalf(
			"BUG 1.24: 下游超时后审批状态变为 '%s'（终态），应保持 'approved' 以支持重试",
			a.Status,
		)
	}
}

// ── BUG 1.25: 审批创建时 resource_id != payload.asset_id 不校验 ──
// Bug: Create 不校验 resource_id 与 payload 中 asset_id 的一致性。
// 期望行为: 当两者不一致时应返回 400。
func TestBug_1_25_ResourceIdMismatch_ShouldReturn400(t *testing.T) {
	repo := newMockRepo()
	ds := downstream.New("http://localhost:9999")
	h := NewApprovalHandler(repo, ds)
	r := newRouter(h)

	// resource_id = "asset-A" 但 payload.asset_id = "asset-B"
	body := `{
		"type": "risk_recovery",
		"resource_type": "asset",
		"resource_id": "asset-A",
		"payload": {"asset_id": "asset-B", "action": "recover"}
	}`
	req := httptest.NewRequest("POST", "/approvals", strings.NewReader(body))
	setClaimsHeaders(req, "user-1", "Brand", "org-1")
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	// 期望: resource_id 与 payload.asset_id 不一致应返回 400
	if w.Code == http.StatusCreated || w.Code == http.StatusOK {
		t.Fatalf(
			"BUG 1.25: 审批创建 resource_id='asset-A' 但 payload.asset_id='asset-B' 返回 %d，"+
				"应返回 400。审批名义上批 A 实际执行 B",
			w.Code,
		)
	}
}
