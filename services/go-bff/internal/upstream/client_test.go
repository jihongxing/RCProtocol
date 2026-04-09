package upstream

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestGet200ReturnsBody(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"ok":true}`))
	}))
	defer srv.Close()

	client := New(srv.URL, srv.URL)
	body, err := client.Get(context.Background(), srv.URL+"/test", "", "")
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if string(body) != `{"ok":true}` {
		t.Fatalf("expected body %q, got %q", `{"ok":true}`, string(body))
	}
}

func TestGet404ReturnsUpstreamError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		_, _ = w.Write([]byte(`not found`))
	}))
	defer srv.Close()

	client := New(srv.URL, srv.URL)
	_, err := client.Get(context.Background(), srv.URL+"/missing", "", "")
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	ue, ok := err.(*UpstreamError)
	if !ok || ue.StatusCode != 404 || ue.Code != "NOT_FOUND" {
		t.Fatalf("unexpected upstream error: %#v", err)
	}
}

func TestGet500ReturnsUpstreamFailure(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer srv.Close()

	client := New(srv.URL, srv.URL)
	_, err := client.Get(context.Background(), srv.URL+"/error", "", "")
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	ue, ok := err.(*UpstreamError)
	if !ok || ue.StatusCode != 502 || ue.Code != "UPSTREAM_FAILURE" {
		t.Fatalf("unexpected upstream error: %#v", err)
	}
}

func TestGet4xxPassThroughUpstreamError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"error": map[string]string{"code": "CUSTOM_ERROR", "message": "custom upstream message"},
		})
	}))
	defer srv.Close()

	client := New(srv.URL, srv.URL)
	_, err := client.Get(context.Background(), srv.URL+"/bad", "", "")
	ue, ok := err.(*UpstreamError)
	if err == nil || !ok || ue.StatusCode != 400 || ue.Code != "CUSTOM_ERROR" || ue.Message != "custom upstream message" {
		t.Fatalf("unexpected upstream error: %#v", err)
	}
}

func TestGetForwardsAuthorizationAndTraceID(t *testing.T) {
	var gotAuth, gotTrace string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotAuth = r.Header.Get("Authorization")
		gotTrace = r.Header.Get("X-Trace-Id")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{}`))
	}))
	defer srv.Close()

	client := New(srv.URL, srv.URL)
	_, err := client.Get(context.Background(), srv.URL+"/check", "Bearer test-token", "trace-abc-123")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if gotAuth != "Bearer test-token" || gotTrace != "trace-abc-123" {
		t.Fatalf("unexpected forwarded headers auth=%q trace=%q", gotAuth, gotTrace)
	}
}

func TestGetWithGatewayAuthForwardsApiKeyContractHeaders(t *testing.T) {
	var gotHash, gotVerified, gotAuth, gotTrace string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotAuth = r.Header.Get("Authorization")
		gotTrace = r.Header.Get("X-Trace-Id")
		gotHash = r.Header.Get("X-Api-Key-Hash")
		gotVerified = r.Header.Get("X-Api-Key-Verified")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{}`))
	}))
	defer srv.Close()

	client := New(srv.URL, srv.URL)
	_, err := client.GetWithGatewayAuth(context.Background(), srv.URL+"/check", "Bearer test-token", "trace-abc-123", "hash-123", "hash-only")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if gotAuth != "Bearer test-token" || gotTrace != "trace-abc-123" || gotHash != "hash-123" || gotVerified != "hash-only" {
		t.Fatalf("unexpected gateway auth forwarding auth=%q trace=%q hash=%q verified=%q", gotAuth, gotTrace, gotHash, gotVerified)
	}
}

func TestDoWithGatewayAuth_PostForwardsBodyAndHeaders(t *testing.T) {
	var gotMethod, gotBody, gotContentType, gotHash, gotVerified string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotMethod = r.Method
		gotContentType = r.Header.Get("Content-Type")
		gotHash = r.Header.Get("X-Api-Key-Hash")
		gotVerified = r.Header.Get("X-Api-Key-Verified")
		body, _ := io.ReadAll(r.Body)
		gotBody = string(body)
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"ok":true}`))
	}))
	defer srv.Close()

	client := New(srv.URL, srv.URL)
	_, err := client.DoWithGatewayAuth(context.Background(), http.MethodPost, srv.URL+"/write", []byte(`{"hello":"world"}`), "application/json", GatewayAuthHeaders{
		Authorization:  "Bearer test-token",
		TraceID:        "trace-post-1",
		ApiKeyHash:     "hash-123",
		ApiKeyVerified: "hash-only",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if gotMethod != http.MethodPost || gotBody != `{"hello":"world"}` || gotContentType != "application/json" || gotHash != "hash-123" || gotVerified != "hash-only" {
		t.Fatalf("unexpected post forwarding method=%q body=%q contentType=%q hash=%q verified=%q", gotMethod, gotBody, gotContentType, gotHash, gotVerified)
	}
}

func TestGetBrandNameSuccess(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/brands/b-001" {
			w.WriteHeader(http.StatusOK)
			_ = json.NewEncoder(w).Encode(map[string]string{"brand_name": "Luxury Brand"})
			return
		}
		w.WriteHeader(http.StatusNotFound)
	}))
	defer srv.Close()

	client := New(srv.URL, "")
	name := client.GetBrandName(context.Background(), "b-001", "", "")
	if name != "Luxury Brand" {
		t.Errorf("expected %q, got %q", "Luxury Brand", name)
	}
}

func TestGetBrandNameFallback(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer srv.Close()

	client := New(srv.URL, "")
	name := client.GetBrandName(context.Background(), "b-fallback", "", "")
	if name != "b-fallback" {
		t.Errorf("expected fallback %q, got %q", "b-fallback", name)
	}
}

func TestGetProductNameSuccess(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/brands/b-001/products/p-001" {
			w.WriteHeader(http.StatusOK)
			_ = json.NewEncoder(w).Encode(map[string]string{"product_name": "Premium Watch"})
			return
		}
		w.WriteHeader(http.StatusNotFound)
	}))
	defer srv.Close()

	client := New(srv.URL, "")
	name := client.GetProductName(context.Background(), "b-001", "p-001", "", "")
	if name != "Premium Watch" {
		t.Errorf("expected %q, got %q", "Premium Watch", name)
	}
}

func TestGetProductNameFallback(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer srv.Close()

	client := New(srv.URL, "")
	name := client.GetProductName(context.Background(), "b-001", "p-fallback", "", "")
	if name != "p-fallback" {
		t.Errorf("expected fallback %q, got %q", "p-fallback", name)
	}
}

func TestSanitizeURLForLog(t *testing.T) {
	tests := []struct{ input, want string }{{"http://rc-api:8081/brands/b-001", "/brands/b-001"}, {"http://rc-api:8081/assets?owner_id=user-123&page=1", "/assets"}, {"http://rc-api:8081/brands/b-001/products/p-001", "/brands/b-001/products/p-001"}, {"/just-a-path", "/just-a-path"}}
	for _, tt := range tests {
		if got := sanitizeURLForLog(tt.input); got != tt.want {
			t.Errorf("sanitizeURLForLog(%q) = %q, want %q", tt.input, got, tt.want)
		}
	}
}

func TestRcApiURL(t *testing.T) {
	client := New("http://rc-api:8081", "http://go-iam:8083")
	if got := client.RcApiURL("/brands/b-001"); got != "http://rc-api:8081/brands/b-001" {
		t.Errorf("RcApiURL = %q", got)
	}
}

func TestIamURL(t *testing.T) {
	client := New("http://rc-api:8081", "http://go-iam:8083")
	if got := client.IamURL("/users/u-001"); got != "http://go-iam:8083/users/u-001" {
		t.Errorf("IamURL = %q", got)
	}
}

func TestUpstreamErrorString(t *testing.T) {
	e := &UpstreamError{StatusCode: 502, Code: "UPSTREAM_FAILURE", Message: "backend service unavailable"}
	if got := e.Error(); got != "upstream 502: backend service unavailable" {
		t.Errorf("Error() = %q", got)
	}
}
