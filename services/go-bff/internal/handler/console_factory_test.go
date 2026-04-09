package handler

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"rcprotocol/services/go-bff/internal/upstream"
)

func TestFactoryListTasksEmptyList(t *testing.T) {
	h := NewFactoryTaskHandler(nil)
	r := httptest.NewRequest(http.MethodGet, "/console/factory/tasks", nil)
	w := httptest.NewRecorder()
	h.ListTasks(w, r)
	resp := w.Result()
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("StatusCode = %d, want %d", resp.StatusCode, http.StatusOK)
	}
	if ct := resp.Header.Get("Content-Type"); ct != "application/json; charset=utf-8" {
		t.Errorf("Content-Type = %q, want %q", ct, "application/json; charset=utf-8")
	}

	var body struct {
		Items    []interface{} `json:"items"`
		Total    int           `json:"total"`
		Page     int           `json:"page"`
		PageSize int           `json:"page_size"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		t.Fatalf("failed to decode JSON: %v", err)
	}
	if len(body.Items) != 0 || body.Total != 0 || body.Page != 1 || body.PageSize != 20 {
		t.Fatalf("unexpected list body: %+v", body)
	}
}

func TestFactoryListTasksCustomPagination(t *testing.T) {
	h := NewFactoryTaskHandler(nil)
	r := httptest.NewRequest(http.MethodGet, "/console/factory/tasks?page=3&page_size=50", nil)
	w := httptest.NewRecorder()
	h.ListTasks(w, r)
	resp := w.Result()
	defer resp.Body.Close()

	var body struct {
		Items    []interface{} `json:"items"`
		Total    int           `json:"total"`
		Page     int           `json:"page"`
		PageSize int           `json:"page_size"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		t.Fatalf("failed to decode JSON: %v", err)
	}
	if len(body.Items) != 0 || body.Total != 0 || body.Page != 3 || body.PageSize != 50 {
		t.Fatalf("unexpected list body: %+v", body)
	}
}

func TestFactoryCreateQuickLog_UsesGatewayAwarePostHelper(t *testing.T) {
	var gotMethod, gotBody, gotHash, gotVerified, gotContentType string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotMethod = r.Method
		gotContentType = r.Header.Get("Content-Type")
		gotHash = r.Header.Get("X-Api-Key-Hash")
		gotVerified = r.Header.Get("X-Api-Key-Verified")
		body, _ := io.ReadAll(r.Body)
		gotBody = string(body)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		_, _ = w.Write([]byte(`{"ok":true,"log_id":"ql-001","event_type":"scan"}`))
	}))
	defer srv.Close()

	h := NewFactoryTaskHandler(upstream.New(srv.URL, srv.URL))
	r := httptest.NewRequest(http.MethodPost, "/console/factory/quick-log", strings.NewReader(`{"batch_id":"bat-001","event_type":"scan"}`))
	r.Header.Set("Content-Type", "application/json")
	r.Header.Set("X-Api-Key-Hash", "hash-123")
	r.Header.Set("X-Api-Key-Verified", "hash-only")
	w := httptest.NewRecorder()
	h.CreateQuickLog(w, r)

	if w.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d body=%s", w.Code, w.Body.String())
	}
	if gotMethod != http.MethodPost || gotBody != `{"batch_id":"bat-001","event_type":"scan"}` || gotContentType != "application/json" || gotHash != "hash-123" || gotVerified != "hash-only" {
		t.Fatalf("unexpected forwarded post method=%q body=%q contentType=%q hash=%q verified=%q", gotMethod, gotBody, gotContentType, gotHash, gotVerified)
	}
	var resp FactoryQuickLogCreateResponse
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("invalid response JSON: %v", err)
	}
	if !resp.OK || resp.LogID != "ql-001" || resp.EventType != "scan" {
		t.Fatalf("unexpected response dto: %+v", resp)
	}
}

func TestFactoryCreateQuickLog_RejectsMissingBatchID(t *testing.T) {
	h := NewFactoryTaskHandler(upstream.New("http://127.0.0.1:1", "http://127.0.0.1:1"))
	r := httptest.NewRequest(http.MethodPost, "/console/factory/quick-log", strings.NewReader(`{"event_type":"scan"}`))
	r.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	h.CreateQuickLog(w, r)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d body=%s", w.Code, w.Body.String())
	}
	var body ErrorBody
	_ = json.Unmarshal(w.Body.Bytes(), &body)
	if body.Error.Code != "INVALID_INPUT" {
		t.Fatalf("unexpected error body: %+v", body)
	}
}

func TestFactoryCreateQuickLog_RejectsMissingEventType(t *testing.T) {
	h := NewFactoryTaskHandler(upstream.New("http://127.0.0.1:1", "http://127.0.0.1:1"))
	r := httptest.NewRequest(http.MethodPost, "/console/factory/quick-log", strings.NewReader(`{"batch_id":"bat-001"}`))
	r.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	h.CreateQuickLog(w, r)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d body=%s", w.Code, w.Body.String())
	}
	var body ErrorBody
	_ = json.Unmarshal(w.Body.Bytes(), &body)
	if body.Error.Code != "INVALID_INPUT" {
		t.Fatalf("unexpected error body: %+v", body)
	}
}
