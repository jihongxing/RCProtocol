package middleware

import (
	"net/http"
	"net/http/httptest"
	"regexp"
	"testing"
)

// uuidV4Pattern matches a standard UUID v4 string (8-4-4-4-12 hex).
var uuidV4Pattern = regexp.MustCompile(
	`^[0-9a-f]{8}-[0-9a-f]{4}-4[0-9a-f]{3}-[89ab][0-9a-f]{3}-[0-9a-f]{12}$`,
)

func TestTrace_NoHeader_GeneratesUUID(t *testing.T) {
	// A downstream handler that records the trace ID it received.
	var forwarded string
	inner := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		forwarded = r.Header.Get(TraceIDHeader)
		w.WriteHeader(http.StatusOK)
	})

	handler := Trace(inner)

	req := httptest.NewRequest(http.MethodGet, "/any", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	// Response must contain a valid UUID v4 in X-Trace-Id.
	got := rec.Header().Get(TraceIDHeader)
	if got == "" {
		t.Fatal("expected X-Trace-Id in response, got empty")
	}
	if !uuidV4Pattern.MatchString(got) {
		t.Fatalf("X-Trace-Id %q is not a valid UUID v4", got)
	}

	// The same UUID must have been forwarded to the downstream handler.
	if forwarded != got {
		t.Fatalf("forwarded trace ID %q != response trace ID %q", forwarded, got)
	}
}

func TestTrace_WithHeader_Preserved(t *testing.T) {
	const existing = "abc-existing-trace-id-123"

	var forwarded string
	inner := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		forwarded = r.Header.Get(TraceIDHeader)
		w.WriteHeader(http.StatusOK)
	})

	handler := Trace(inner)

	req := httptest.NewRequest(http.MethodGet, "/any", nil)
	req.Header.Set(TraceIDHeader, existing)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	// Response must echo the original trace ID.
	got := rec.Header().Get(TraceIDHeader)
	if got != existing {
		t.Fatalf("expected response X-Trace-Id %q, got %q", existing, got)
	}

	// Downstream must also see the original trace ID.
	if forwarded != existing {
		t.Fatalf("expected forwarded X-Trace-Id %q, got %q", existing, forwarded)
	}
}
