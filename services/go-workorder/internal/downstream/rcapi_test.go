package downstream

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"pgregory.net/rapid"
)

func TestFreeze_CorrectURLAndHeaders(t *testing.T) {
	var gotPath, gotMethod, gotAuth, gotTrace string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		gotMethod = r.Method
		gotAuth = r.Header.Get("Authorization")
		gotTrace = r.Header.Get("X-Trace-Id")
		w.WriteHeader(200)
		fmt.Fprint(w, `{"frozen":true}`)
	}))
	defer srv.Close()

	c := NewRcApiClient(srv.URL)
	result := c.Freeze(context.Background(), "asset-abc-123", "Bearer tok123", "trace-xyz")

	if !result.Success {
		t.Fatalf("expected success, got failure: %s", result.Body)
	}
	if gotMethod != "POST" {
		t.Errorf("method = %q, want POST", gotMethod)
	}
	if gotPath != "/assets/asset-abc-123/freeze" {
		t.Errorf("path = %q, want /assets/asset-abc-123/freeze", gotPath)
	}
	if gotAuth != "Bearer tok123" {
		t.Errorf("Authorization = %q, want Bearer tok123", gotAuth)
	}
	if gotTrace != "trace-xyz" {
		t.Errorf("X-Trace-Id = %q, want trace-xyz", gotTrace)
	}
}

func TestRecover_CorrectURLAndApprovalID(t *testing.T) {
	var gotPath, gotMethod, gotAuth, gotTrace, gotApprovalID string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		gotMethod = r.Method
		gotAuth = r.Header.Get("Authorization")
		gotTrace = r.Header.Get("X-Trace-Id")
		gotApprovalID = r.Header.Get("X-Approval-Id")
		w.WriteHeader(200)
		fmt.Fprint(w, `{"recovered":true}`)
	}))
	defer srv.Close()

	c := NewRcApiClient(srv.URL)
	result := c.Recover(context.Background(), "asset-def-456", "Bearer tok456", "trace-abc", "approval-001")

	if !result.Success {
		t.Fatalf("expected success, got failure: %s", result.Body)
	}
	if gotMethod != "POST" {
		t.Errorf("method = %q, want POST", gotMethod)
	}
	if gotPath != "/assets/asset-def-456/recover" {
		t.Errorf("path = %q, want /assets/asset-def-456/recover", gotPath)
	}
	if gotAuth != "Bearer tok456" {
		t.Errorf("Authorization = %q, want Bearer tok456", gotAuth)
	}
	if gotTrace != "trace-abc" {
		t.Errorf("X-Trace-Id = %q, want trace-abc", gotTrace)
	}
	if gotApprovalID != "approval-001" {
		t.Errorf("X-Approval-Id = %q, want approval-001", gotApprovalID)
	}
}

func TestDownstream200_SuccessTrue(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
		fmt.Fprint(w, `{"ok":true}`)
	}))
	defer srv.Close()

	c := NewRcApiClient(srv.URL)
	result := c.Freeze(context.Background(), "asset-1", "Bearer t", "trace-1")

	if !result.Success {
		t.Fatal("expected Success=true for 200 response")
	}
	if string(result.Body) != `{"ok":true}` {
		t.Errorf("body = %s, want {\"ok\":true}", result.Body)
	}
}

func TestDownstream500_SuccessFalse(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(500)
		fmt.Fprint(w, `internal error`)
	}))
	defer srv.Close()

	c := NewRcApiClient(srv.URL)
	result := c.Freeze(context.Background(), "asset-1", "", "")

	if result.Success {
		t.Fatal("expected Success=false for 500 response")
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

func TestFreeze_InvalidAssetID(t *testing.T) {
	cases := []struct {
		name    string
		assetID string
	}{
		{"path traversal", "../etc/passwd"},
		{"spaces", "id with spaces"},
		{"semicolon", "id;drop"},
		{"slash", "abc/def"},
		{"empty", ""},
		{"dot", "abc.def"},
		{"percent", "abc%20def"},
	}

	c := NewRcApiClient("http://127.0.0.1:1") // unreachable, should not be called
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			result := c.Freeze(context.Background(), tc.assetID, "", "")
			if result.Success {
				t.Fatalf("expected failure for assetID %q", tc.assetID)
			}
		})
	}
}

func TestRecover_InvalidAssetID(t *testing.T) {
	c := NewRcApiClient("http://127.0.0.1:1")
	result := c.Recover(context.Background(), "../etc", "", "", "")
	if result.Success {
		t.Fatal("expected failure for invalid assetID in Recover")
	}
}

func TestIsValidID(t *testing.T) {
	cases := []struct {
		id   string
		want bool
	}{
		{"abc-123", true},
		{"ABC_def", true},
		{"a1b2c3", true},
		{"valid-id_123", true},
		{"", false},
		{"../etc", false},
		{"id with space", false},
		{"id;drop", false},
		{"abc/def", false},
		{"abc.def", false},
	}
	for _, tc := range cases {
		got := IsValidID(tc.id)
		if got != tc.want {
			t.Errorf("IsValidID(%q) = %v, want %v", tc.id, got, tc.want)
		}
	}
}

func TestDownstreamUnreachable_Failure(t *testing.T) {
	c := NewRcApiClient("http://127.0.0.1:1")
	result := c.Freeze(context.Background(), "valid-id", "Bearer t", "trace-1")
	if result.Success {
		t.Fatal("expected failure for unreachable downstream")
	}
}


// TestDownstreamHeaderTransparency 属性测试：下游调用 header 透传
// Property 9: 使用 rapid 生成随机 Authorization 和 X-Trace-Id 值，
// 验证 Freeze 和 Recover 调用 mock 上游时收到完全相同的 header 值，
// 且 Recover 调用还包含 X-Approval-Id。
// **Validates: Requirements 12.3**
func TestDownstreamHeaderTransparency(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		authHeader := rapid.StringMatching(`Bearer [a-zA-Z0-9\.\-_]{10,100}`).Draw(t, "authHeader")
		traceID := rapid.StringMatching(`[a-f0-9\-]{8,40}`).Draw(t, "traceID")
		approvalID := rapid.StringMatching(`[a-zA-Z0-9\-]{1,50}`).Draw(t, "approvalID")
		assetID := rapid.StringMatching(`[a-zA-Z0-9\-]{1,20}`).Draw(t, "assetID")

		// Test Freeze: Authorization + X-Trace-Id must be passed through
		var freezeAuth, freezeTrace string
		freezeSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			freezeAuth = r.Header.Get("Authorization")
			freezeTrace = r.Header.Get("X-Trace-Id")
			w.WriteHeader(200)
		}))
		defer freezeSrv.Close()

		fc := NewRcApiClient(freezeSrv.URL)
		fc.Freeze(context.Background(), assetID, authHeader, traceID)

		if freezeAuth != authHeader {
			t.Fatalf("Freeze Authorization = %q, want %q", freezeAuth, authHeader)
		}
		if freezeTrace != traceID {
			t.Fatalf("Freeze X-Trace-Id = %q, want %q", freezeTrace, traceID)
		}

		// Test Recover: Authorization + X-Trace-Id + X-Approval-Id must be passed through
		var recoverAuth, recoverTrace, recoverApproval string
		recoverSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			recoverAuth = r.Header.Get("Authorization")
			recoverTrace = r.Header.Get("X-Trace-Id")
			recoverApproval = r.Header.Get("X-Approval-Id")
			w.WriteHeader(200)
		}))
		defer recoverSrv.Close()

		rc := NewRcApiClient(recoverSrv.URL)
		rc.Recover(context.Background(), assetID, authHeader, traceID, approvalID)

		if recoverAuth != authHeader {
			t.Fatalf("Recover Authorization = %q, want %q", recoverAuth, authHeader)
		}
		if recoverTrace != traceID {
			t.Fatalf("Recover X-Trace-Id = %q, want %q", recoverTrace, traceID)
		}
		if recoverApproval != approvalID {
			t.Fatalf("Recover X-Approval-Id = %q, want %q", recoverApproval, approvalID)
		}
	})
}
