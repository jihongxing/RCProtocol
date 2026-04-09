package middleware

import (
	"bytes"
	"encoding/json"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// logEntry mirrors the JSON structure emitted by slog.NewJSONHandler.
type logEntry struct {
	Level    string `json:"level"`
	Msg      string `json:"msg"`
	Method   string `json:"method"`
	Path     string `json:"path"`
	Status   int    `json:"status"`
	LatencyMs json.Number `json:"latency_ms"`
	TraceID  string `json:"trace_id"`
	ClientIP string `json:"client_ip"`
}

func parseLogEntry(t *testing.T, buf *bytes.Buffer) logEntry {
	t.Helper()
	var entry logEntry
	if err := json.Unmarshal(buf.Bytes(), &entry); err != nil {
		t.Fatalf("failed to parse log JSON: %v\nraw: %s", err, buf.String())
	}
	return entry
}

func TestLogging_FieldsPresent(t *testing.T) {
	var buf bytes.Buffer
	logger := slog.New(slog.NewJSONHandler(&buf, nil))

	handler := Logging(logger)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodGet, "/api/brands", nil)
	req.Header.Set(TraceIDHeader, "test-trace-123")
	req.RemoteAddr = "192.168.1.100:54321"
	rr := httptest.NewRecorder()

	handler.ServeHTTP(rr, req)

	entry := parseLogEntry(t, &buf)

	if entry.Method != "GET" {
		t.Errorf("method = %q, want GET", entry.Method)
	}
	if entry.Path != "/api/brands" {
		t.Errorf("path = %q, want /api/brands", entry.Path)
	}
	if entry.Status != 200 {
		t.Errorf("status = %d, want 200", entry.Status)
	}
	if entry.TraceID != "test-trace-123" {
		t.Errorf("trace_id = %q, want test-trace-123", entry.TraceID)
	}
	if entry.ClientIP != "192.168.1.100:54321" {
		t.Errorf("client_ip = %q, want 192.168.1.100:54321", entry.ClientIP)
	}
	// latency_ms must be a non-negative number
	latency, err := entry.LatencyMs.Int64()
	if err != nil {
		t.Fatalf("latency_ms not a valid integer: %v", err)
	}
	if latency < 0 {
		t.Errorf("latency_ms = %d, want >= 0", latency)
	}
}

func TestLogging_NoAuthorizationLeakage(t *testing.T) {
	var buf bytes.Buffer
	logger := slog.New(slog.NewJSONHandler(&buf, nil))

	handler := Logging(logger)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodGet, "/api/brands", nil)
	req.Header.Set("Authorization", "Bearer super-secret-token-value")
	req.Header.Set(TraceIDHeader, "trace-auth-test")
	rr := httptest.NewRecorder()

	handler.ServeHTTP(rr, req)

	raw := buf.String()
	if strings.Contains(raw, "super-secret-token-value") {
		t.Errorf("log entry contains Authorization token value:\n%s", raw)
	}
	if strings.Contains(raw, "Bearer") {
		t.Errorf("log entry contains Bearer prefix:\n%s", raw)
	}
}

func TestLogging_4xx_WarnLevel(t *testing.T) {
	var buf bytes.Buffer
	// Use LevelDebug so WARN entries are captured.
	logger := slog.New(slog.NewJSONHandler(&buf, &slog.HandlerOptions{Level: slog.LevelDebug}))

	handler := Logging(logger)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))

	req := httptest.NewRequest(http.MethodGet, "/missing", nil)
	rr := httptest.NewRecorder()

	handler.ServeHTTP(rr, req)

	entry := parseLogEntry(t, &buf)
	if entry.Level != "WARN" {
		t.Errorf("level = %q, want WARN for 404", entry.Level)
	}
	if entry.Status != 404 {
		t.Errorf("status = %d, want 404", entry.Status)
	}
}

func TestLogging_5xx_ErrorLevel(t *testing.T) {
	var buf bytes.Buffer
	logger := slog.New(slog.NewJSONHandler(&buf, &slog.HandlerOptions{Level: slog.LevelDebug}))

	handler := Logging(logger)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))

	req := httptest.NewRequest(http.MethodPost, "/api/fail", nil)
	rr := httptest.NewRecorder()

	handler.ServeHTTP(rr, req)

	entry := parseLogEntry(t, &buf)
	if entry.Level != "ERROR" {
		t.Errorf("level = %q, want ERROR for 500", entry.Level)
	}
	if entry.Status != 500 {
		t.Errorf("status = %d, want 500", entry.Status)
	}
}

func TestLogging_DefaultStatusOK(t *testing.T) {
	var buf bytes.Buffer
	logger := slog.New(slog.NewJSONHandler(&buf, nil))

	// Handler that writes body without explicit WriteHeader → implicit 200
	handler := Logging(logger)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("ok"))
	}))

	req := httptest.NewRequest(http.MethodGet, "/healthz", nil)
	rr := httptest.NewRecorder()

	handler.ServeHTTP(rr, req)

	entry := parseLogEntry(t, &buf)
	if entry.Status != 200 {
		t.Errorf("status = %d, want 200 (default)", entry.Status)
	}
	if entry.Level != "INFO" {
		t.Errorf("level = %q, want INFO for 200", entry.Level)
	}
}

func TestLogging_StatusRecorderUnwrap(t *testing.T) {
	inner := httptest.NewRecorder()
	sr := &statusRecorder{ResponseWriter: inner, statusCode: http.StatusOK}

	if got := sr.Unwrap(); got != inner {
		t.Error("Unwrap() did not return the underlying ResponseWriter")
	}
}

// --- API Key 日志安全测试 (Task 15 / FR-09 9.6) ---

func TestLogging_ApiKeyNotLogged(t *testing.T) {
	var buf bytes.Buffer
	logger := slog.New(slog.NewJSONHandler(&buf, nil))

	handler := Logging(logger)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodGet, "/api/brands/my-brand/assets", nil)
	req.Header.Set(ApiKeyHeader, "brand_super_secret_api_key_12345")
	req.Header.Set(TraceIDHeader, "trace-apikey-test")
	rr := httptest.NewRecorder()

	handler.ServeHTTP(rr, req)

	raw := buf.String()
	if strings.Contains(raw, "brand_super_secret_api_key_12345") {
		t.Errorf("log entry contains API Key value:\n%s", raw)
	}
	if strings.Contains(raw, "super_secret") {
		t.Errorf("log entry contains partial API Key value:\n%s", raw)
	}
}

func TestLogging_ApiKeyFlagPresent(t *testing.T) {
	var buf bytes.Buffer
	logger := slog.New(slog.NewJSONHandler(&buf, nil))

	handler := Logging(logger)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodGet, "/api/brands/my-brand/assets", nil)
	req.Header.Set(ApiKeyHeader, "brand_test123abc")
	req.Header.Set(TraceIDHeader, "trace-flag-test")
	rr := httptest.NewRecorder()

	handler.ServeHTTP(rr, req)

	raw := buf.String()

	// 解析日志 JSON 验证 api_key_auth 字段
	var entry map[string]interface{}
	if err := json.Unmarshal(buf.Bytes(), &entry); err != nil {
		t.Fatalf("failed to parse log: %v", err)
	}
	val, ok := entry["api_key_auth"]
	if !ok {
		t.Fatalf("log entry missing api_key_auth field:\n%s", raw)
	}
	if val != true {
		t.Fatalf("expected api_key_auth=true, got %v", val)
	}
}

func TestLogging_ApiKeyFlagFalseWhenAbsent(t *testing.T) {
	var buf bytes.Buffer
	logger := slog.New(slog.NewJSONHandler(&buf, nil))

	handler := Logging(logger)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodGet, "/api/brands", nil)
	req.Header.Set(TraceIDHeader, "trace-no-apikey")
	rr := httptest.NewRecorder()

	handler.ServeHTTP(rr, req)

	var entry map[string]interface{}
	if err := json.Unmarshal(buf.Bytes(), &entry); err != nil {
		t.Fatalf("failed to parse log: %v", err)
	}
	val, ok := entry["api_key_auth"]
	if !ok {
		t.Fatalf("log entry missing api_key_auth field")
	}
	if val != false {
		t.Fatalf("expected api_key_auth=false, got %v", val)
	}
}
