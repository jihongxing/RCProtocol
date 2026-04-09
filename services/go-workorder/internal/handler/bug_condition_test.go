package handler

// Bug Condition 探索性测试 — 工单服务缺陷
//
// 这些测试编码了**期望行为**：在未修复代码上应当 FAIL，证明缺陷存在。
// 修复后测试通过即确认修复成功。
//
// **Validates: Requirements 1.30**

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"rcprotocol/services/go-workorder/internal/model"
)

// ── BUG 1.30: ListByAsset 无 Claims 校验 ──
// Bug: ListByAsset 不调用 claims.FromRequest、不做 Valid() 检查、不做品牌边界过滤。
// 期望行为: 无有效 Claims 时应返回 401。
func TestBug_1_30_ListByAsset_WithoutClaims_ShouldReturn401(t *testing.T) {
	repo := newMockRepo()
	assetID := "asset-1"
	repo.workorders["w1"] = &model.Workorder{
		ID:      "w1",
		AssetID: &assetID,
		Status:  "open",
	}

	h := NewWorkorderHandler(repo, nil)
	r := newTestRouter(h)

	// 故意不设置 X-Claims-* 头
	req := httptest.NewRequest(http.MethodGet, "/workorders/by-asset?asset_id=asset-1", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code == http.StatusOK {
		t.Fatalf(
			"BUG 1.30: ListByAsset 无 Claims 返回 200，应返回 401。"+
				"当前行为: 任何人只要知道 asset_id 即可查看全部工单记录",
		)
	}
}
