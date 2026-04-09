package middleware

import (
	"bytes"
	"encoding/json"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"
)

// logEntry represents a parsed JSON log line for assertion.
type logEntry struct {
	Level     string `json:"level"`
	Msg       string `json:"msg"`
	Method    string `json:"method"`
	Path      string `json:"path"`
	Status    int    `json:"status"`
	LatencyMs *int64 `json:"latency_ms"`
}

func newTestLogger(buf *bytes.Buffer) *slog.Logger {
	return slog.New(slog.NewJSONHandler(buf, &slog.HandlerOptions{
		Level: slog.LevelDebug,
	}))
}

func parseLogEntry(t *testing.T, buf *bytes.Buffer) logEntry {
	t.Helper()
	var entry logEntry
	if err := json.Unmarshal(buf.Bytes(), &entry); err != nil {
		t.Fatalf("failed to parse log entry: %v\nraw: %s", err, buf.String())
	}
	return entry
}

func TestLogging_ContainsMethodPathStatusLatency(t *testing.T) {
	var buf bytes.Buffer
	logger := newTestLogger(&buf)

	handler := Logging(logger)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodGet, "/app/assets", nil)
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	entry := parseLogEntry(t, &buf)

	if entry.Method != "GET" {
		t.Errorf("expected method=GET, got %q", entry.Method)
	}
	if entry.Path != "/app/assets" {
		t.Errorf("expected path=/app/assets, got %q", entry.Path)
	}
	if entry.Status != 200 {
		t.Errorf("expected status=200, got %d", entry.Status)
	}
	if entry.LatencyMs == nil {
		t.Error("expected latency_ms to be present")
	}
	if entry.Msg != "request completed" {
		t.Errorf("expected msg=request completed, got %q", entry.Msg)
	}
}

func TestLogging_AuthorizationNotLogged(t *testing.T) {
	var buf bytes.Buffer
	logger := newTestLogger(&buf)

	handler := Logging(logger)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodPost, "/console/dashboard", nil)
	req.Header.Set("Authorization", "Bearer super-secret-token-value")
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	raw := buf.String()
	if bytes.Contains([]byte(raw), []byte("super-secret-token-value")) {
		t.Errorf("log output must not contain Authorization token value, got: %s", raw)
	}
	if bytes.Contains([]byte(raw), []byte("Bearer")) {
		t.Errorf("log output must not contain Bearer prefix, got: %s", raw)
	}
}

func TestLogging_4xxWarnLevel(t *testing.T) {
	var buf bytes.Buffer
	logger := newTestLogger(&buf)

	handler := Logging(logger)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))

	req := httptest.NewRequest(http.MethodGet, "/not-found", nil)
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	entry := parseLogEntry(t, &buf)

	if entry.Level != "WARN" {
		t.Errorf("expected level=WARN for 404, got %q", entry.Level)
	}
	if entry.Status != 404 {
		t.Errorf("expected status=404, got %d", entry.Status)
	}
}

func TestLogging_5xxErrorLevel(t *testing.T) {
	var buf bytes.Buffer
	logger := newTestLogger(&buf)

	handler := Logging(logger)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))

	req := httptest.NewRequest(http.MethodGet, "/error", nil)
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	entry := parseLogEntry(t, &buf)

	if entry.Level != "ERROR" {
		t.Errorf("expected level=ERROR for 500, got %q", entry.Level)
	}
	if entry.Status != 500 {
		t.Errorf("expected status=500, got %d", entry.Status)
	}
}

func TestLogging_2xxInfoLevel(t *testing.T) {
	var buf bytes.Buffer
	logger := newTestLogger(&buf)

	handler := Logging(logger)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusCreated)
	}))

	req := httptest.NewRequest(http.MethodPost, "/resource", nil)
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	entry := parseLogEntry(t, &buf)

	if entry.Level != "INFO" {
		t.Errorf("expected level=INFO for 201, got %q", entry.Level)
	}
	if entry.Status != 201 {
		t.Errorf("expected status=201, got %d", entry.Status)
	}
}

func TestLogging_3xxInfoLevel(t *testing.T) {
	var buf bytes.Buffer
	logger := newTestLogger(&buf)

	handler := Logging(logger)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusMovedPermanently)
	}))

	req := httptest.NewRequest(http.MethodGet, "/redirect", nil)
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	entry := parseLogEntry(t, &buf)

	if entry.Level != "INFO" {
		t.Errorf("expected level=INFO for 301, got %q", entry.Level)
	}
}

func TestLogging_DefaultStatusOK(t *testing.T) {
	var buf bytes.Buffer
	logger := newTestLogger(&buf)

	// Handler writes body without explicit WriteHeader — defaults to 200
	handler := Logging(logger)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("ok"))
	}))

	req := httptest.NewRequest(http.MethodGet, "/healthz", nil)
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	entry := parseLogEntry(t, &buf)

	if entry.Status != 200 {
		t.Errorf("expected default status=200, got %d", entry.Status)
	}
	if entry.Level != "INFO" {
		t.Errorf("expected level=INFO for default 200, got %q", entry.Level)
	}
}
