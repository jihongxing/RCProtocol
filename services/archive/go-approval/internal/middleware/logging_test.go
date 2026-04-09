package middleware

import (
	"bytes"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestLogging_ContainsRequiredFields(t *testing.T) {
	var buf bytes.Buffer
	logger := slog.New(slog.NewJSONHandler(&buf, &slog.HandlerOptions{Level: slog.LevelDebug}))

	handler := Logging(logger)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest("GET", "/approvals", nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	logOutput := buf.String()

	for _, field := range []string{"method", "path", "status", "latency_ms"} {
		if !strings.Contains(logOutput, field) {
			t.Errorf("expected log to contain %q, got: %s", field, logOutput)
		}
	}

	if !strings.Contains(logOutput, "GET") {
		t.Errorf("expected log to contain method value GET, got: %s", logOutput)
	}
	if !strings.Contains(logOutput, "/approvals") {
		t.Errorf("expected log to contain path value /approvals, got: %s", logOutput)
	}
}

func TestLogging_DoesNotContainAuthorization(t *testing.T) {
	var buf bytes.Buffer
	logger := slog.New(slog.NewJSONHandler(&buf, &slog.HandlerOptions{Level: slog.LevelDebug}))

	handler := Logging(logger)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest("GET", "/approvals", nil)
	req.Header.Set("Authorization", "Bearer secret-jwt-token-value")
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	logOutput := buf.String()

	if strings.Contains(logOutput, "secret-jwt-token-value") {
		t.Errorf("log should not contain Authorization header value, got: %s", logOutput)
	}
	if strings.Contains(logOutput, "Bearer") {
		t.Errorf("log should not contain Bearer token prefix, got: %s", logOutput)
	}
}

func TestLogging_InfoLevelFor2xx(t *testing.T) {
	var buf bytes.Buffer
	logger := slog.New(slog.NewJSONHandler(&buf, &slog.HandlerOptions{Level: slog.LevelDebug}))

	handler := Logging(logger)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest("GET", "/test", nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	logOutput := buf.String()
	if !strings.Contains(logOutput, "INFO") {
		t.Errorf("expected INFO level for 2xx status, got: %s", logOutput)
	}
}

func TestLogging_WarnLevelFor4xx(t *testing.T) {
	var buf bytes.Buffer
	logger := slog.New(slog.NewJSONHandler(&buf, &slog.HandlerOptions{Level: slog.LevelDebug}))

	handler := Logging(logger)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))

	req := httptest.NewRequest("GET", "/notfound", nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	logOutput := buf.String()
	if !strings.Contains(logOutput, "WARN") {
		t.Errorf("expected WARN level for 4xx status, got: %s", logOutput)
	}
}

func TestLogging_ErrorLevelFor5xx(t *testing.T) {
	var buf bytes.Buffer
	logger := slog.New(slog.NewJSONHandler(&buf, &slog.HandlerOptions{Level: slog.LevelDebug}))

	handler := Logging(logger)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))

	req := httptest.NewRequest("GET", "/error", nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	logOutput := buf.String()
	if !strings.Contains(logOutput, "ERROR") {
		t.Errorf("expected ERROR level for 5xx status, got: %s", logOutput)
	}
}

func TestLogging_DefaultStatusCodeIs200(t *testing.T) {
	var buf bytes.Buffer
	logger := slog.New(slog.NewJSONHandler(&buf, &slog.HandlerOptions{Level: slog.LevelDebug}))

	// Handler that writes body without explicit WriteHeader
	handler := Logging(logger)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("ok"))
	}))

	req := httptest.NewRequest("GET", "/test", nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	logOutput := buf.String()
	// Should log INFO (200) even when WriteHeader is not called explicitly
	if !strings.Contains(logOutput, "INFO") {
		t.Errorf("expected INFO level for default 200 status, got: %s", logOutput)
	}
}
