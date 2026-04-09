package downstream

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"pgregory.net/rapid"
	"rcprotocol/services/go-approval/internal/model"
)

func newTestApproval(typ string, payload json.RawMessage) *model.Approval {
	return &model.Approval{
		ID:      "approval-001",
		Type:    typ,
		Payload: payload,
	}
}

func TestBrandPublish_CorrectURLAndHeaders(t *testing.T) {
	var gotPath, gotAuth, gotTrace, gotApprovalID string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		gotAuth = r.Header.Get("Authorization")
		gotTrace = r.Header.Get("X-Trace-Id")
		gotApprovalID = r.Header.Get("X-Approval-Id")
		w.WriteHeader(200)
		fmt.Fprint(w, `{"ok":true}`)
	}))
	defer srv.Close()

	c := New(srv.URL)
	a := newTestApproval(model.TypeBrandPublish, json.RawMessage(`{"brand_id":"brand-abc-123"}`))
	result := c.Execute(context.Background(), a, "Bearer tok123", "trace-xyz")

	if !result.Success {
		t.Fatalf("expected success, got failure: %s", result.Body)
	}
	if gotPath != "/brands/brand-abc-123/publish" {
		t.Errorf("path = %q, want /brands/brand-abc-123/publish", gotPath)
	}
	if gotAuth != "Bearer tok123" {
		t.Errorf("Authorization = %q, want Bearer tok123", gotAuth)
	}
	if gotTrace != "trace-xyz" {
		t.Errorf("X-Trace-Id = %q, want trace-xyz", gotTrace)
	}
	if gotApprovalID != "approval-001" {
		t.Errorf("X-Approval-Id = %q, want approval-001", gotApprovalID)
	}
}

func TestRiskRecovery_CorrectURL(t *testing.T) {
	var gotPath string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		w.WriteHeader(200)
		fmt.Fprint(w, `{"ok":true}`)
	}))
	defer srv.Close()

	c := New(srv.URL)
	a := newTestApproval(model.TypeRiskRecovery, json.RawMessage(`{"asset_id":"asset-def-456"}`))
	result := c.Execute(context.Background(), a, "", "")

	if !result.Success {
		t.Fatalf("expected success, got failure: %s", result.Body)
	}
	if gotPath != "/assets/asset-def-456/recover" {
		t.Errorf("path = %q, want /assets/asset-def-456/recover", gotPath)
	}
}

func TestPolicyApply_NoHTTPCall(t *testing.T) {
	// 不启动 server，如果发了 HTTP 请求会失败
	c := New("http://127.0.0.1:1") // unreachable port
	a := newTestApproval(model.TypePolicyApply, json.RawMessage(`{}`))
	result := c.Execute(context.Background(), a, "", "")

	if !result.Success {
		t.Fatalf("policy_apply should return success without HTTP call, got: %s", result.Body)
	}
}

func TestHighRiskAction_NoHTTPCall(t *testing.T) {
	c := New("http://127.0.0.1:1")
	a := newTestApproval(model.TypeHighRiskAction, json.RawMessage(`{}`))
	result := c.Execute(context.Background(), a, "", "")

	if !result.Success {
		t.Fatalf("high_risk_action should return success without HTTP call, got: %s", result.Body)
	}
}

func TestDownstream200_SuccessWithBody(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
		fmt.Fprint(w, `{"published":true}`)
	}))
	defer srv.Close()

	c := New(srv.URL)
	a := newTestApproval(model.TypeBrandPublish, json.RawMessage(`{"brand_id":"b1"}`))
	result := c.Execute(context.Background(), a, "", "")

	if !result.Success {
		t.Fatal("expected success for 200 response")
	}
	if string(result.Body) != `{"published":true}` {
		t.Errorf("body = %s, want {\"published\":true}", result.Body)
	}
}

func TestDownstream500_Failure(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(500)
		fmt.Fprint(w, `internal error`)
	}))
	defer srv.Close()

	c := New(srv.URL)
	a := newTestApproval(model.TypeBrandPublish, json.RawMessage(`{"brand_id":"b1"}`))
	result := c.Execute(context.Background(), a, "", "")

	if result.Success {
		t.Fatal("expected failure for 500 response")
	}

	var errBody map[string]interface{}
	if err := json.Unmarshal(result.Body, &errBody); err != nil {
		t.Fatalf("error body not valid JSON: %v", err)
	}
	if _, ok := errBody["error"]; !ok {
		t.Error("error body missing 'error' field")
	}
	if _, ok := errBody["upstream_status"]; !ok {
		t.Error("error body missing 'upstream_status' field")
	}
}

func TestDownstreamUnreachable_Failure(t *testing.T) {
	c := New("http://127.0.0.1:1") // unreachable
	a := newTestApproval(model.TypeBrandPublish, json.RawMessage(`{"brand_id":"b1"}`))
	result := c.Execute(context.Background(), a, "", "")

	if result.Success {
		t.Fatal("expected failure for unreachable downstream")
	}
}

func TestInvalidBrandID_Failure(t *testing.T) {
	c := New("http://127.0.0.1:1")
	a := newTestApproval(model.TypeBrandPublish, json.RawMessage(`{"brand_id":"../etc/passwd"}`))
	result := c.Execute(context.Background(), a, "", "")

	if result.Success {
		t.Fatal("expected failure for invalid brand_id")
	}
}

func TestInvalidAssetID_Failure(t *testing.T) {
	c := New("http://127.0.0.1:1")
	a := newTestApproval(model.TypeRiskRecovery, json.RawMessage(`{"asset_id":"id with spaces"}`))
	result := c.Execute(context.Background(), a, "", "")

	if result.Success {
		t.Fatal("expected failure for invalid asset_id")
	}
}

func TestIsValidID(t *testing.T) {
	cases := []struct {
		id   string
		want bool
	}{
		{"abc-123", true},
		{"ABC_def", true},
		{"", false},
		{"../etc", false},
		{"id with space", false},
		{"id;drop", false},
	}
	for _, tc := range cases {
		got := IsValidID(tc.id)
		if got != tc.want {
			t.Errorf("IsValidID(%q) = %v, want %v", tc.id, got, tc.want)
		}
	}
}

// TestDownstreamHeaderTransparency 属性测试：下游调用 header 透传
// Property 9: 使用 rapid 生成随机 Authorization 和 X-Trace-Id 值，
// 验证 mock 上游收到完全相同的 header 值，且 X-Approval-Id 等于审批单 ID
// Validates: FR-10 (10.4, 10.5)
func TestDownstreamHeaderTransparency(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		approvalID := rapid.StringMatching(`[a-zA-Z0-9\-]{1,50}`).Draw(t, "approvalID")
		authHeader := rapid.StringMatching(`Bearer [a-zA-Z0-9\.\-_]{10,100}`).Draw(t, "authHeader")
		traceID := rapid.StringMatching(`[a-f0-9\-]{8,40}`).Draw(t, "traceID")
		brandID := rapid.StringMatching(`[a-zA-Z0-9\-]{1,20}`).Draw(t, "brandID")

		var gotAuth, gotTrace, gotApprovalID string
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			gotAuth = r.Header.Get("Authorization")
			gotTrace = r.Header.Get("X-Trace-Id")
			gotApprovalID = r.Header.Get("X-Approval-Id")
			w.WriteHeader(200)
		}))
		defer srv.Close()

		c := New(srv.URL)
		a := &model.Approval{
			ID:      approvalID,
			Type:    model.TypeBrandPublish,
			Payload: json.RawMessage(fmt.Sprintf(`{"brand_id":"%s"}`, brandID)),
		}

		c.Execute(context.Background(), a, authHeader, traceID)

		if gotAuth != authHeader {
			t.Fatalf("Authorization = %q, want %q", gotAuth, authHeader)
		}
		if gotTrace != traceID {
			t.Fatalf("X-Trace-Id = %q, want %q", gotTrace, traceID)
		}
		if gotApprovalID != approvalID {
			t.Fatalf("X-Approval-Id = %q, want %q", gotApprovalID, approvalID)
		}
	})
}
