package middleware

import (
	"bytes"
	"encoding/json"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"
)

// logEntry captures a parsed JSON log line.
type logEntry struct {
	Level     string `json:"level"`
	Msg       string `json:"msg"`
	Method    string `json:"method"`
	Path      string `json:"path"`
	Status    int    `json:"status"`
	LatencyMs *int64 `json:"latency_ms"`
	ClientIP  string `json:"client_ip"`
}

func runRequest(t *testing.T, statusCode int) logEntry {
	t.Helper()

	var buf bytes.Buffer
	logger := slog.New(slog.NewJSONHandler(&buf, &slog.HandlerOptions{Level: slog.LevelDebug}))

	handler := Logging(logger)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(statusCode)
	}))

	req := httptest.NewRequest(http.MethodGet, "/test-path", nil)
	req.RemoteAddr = "192.168.1.1:12345"
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	var entry logEntry
	if err := json.Unmarshal(buf.Bytes(), &entry); err != nil {
		t.Fatalf("failed to parse log JSON: %v\nraw: %s", err, buf.String())
	}
	return entry
}

func TestLoggingRequiredFields(t *testing.T) {
	entry := runRequest(t, http.StatusOK)

	if entry.Method != "GET" {
		t.Errorf("expected method=GET, got %s", entry.Method)
	}
	if entry.Path != "/test-path" {
		t.Errorf("expected path=/test-path, got %s", entry.Path)
	}
	if entry.Status != 200 {
		t.Errorf("expected status=200, got %d", entry.Status)
	}
	if entry.LatencyMs == nil {
		t.Error("expected latency_ms to be present")
	}
	if entry.ClientIP == "" {
		t.Error("expected client_ip to be present")
	}
}

func TestLoggingLevels(t *testing.T) {
	tests := []struct {
		name       string
		statusCode int
		wantLevel  string
	}{
		{"2xx is INFO", 200, "INFO"},
		{"3xx is INFO", 301, "INFO"},
		{"4xx is WARN", 404, "WARN"},
		{"400 is WARN", 400, "WARN"},
		{"5xx is ERROR", 500, "ERROR"},
		{"503 is ERROR", 503, "ERROR"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			entry := runRequest(t, tt.statusCode)
			if entry.Level != tt.wantLevel {
				t.Errorf("status %d: expected level %s, got %s", tt.statusCode, tt.wantLevel, entry.Level)
			}
		})
	}
}
