package middleware

import (
	"net/http"

	"github.com/google/uuid"
)

// TraceIDHeader is the canonical header name for distributed trace correlation.
const TraceIDHeader = "X-Trace-Id"

// Trace injects a trace ID into every request/response cycle.
// If the incoming request already carries X-Trace-Id, it is preserved;
// otherwise a new UUID v4 is generated. The trace ID is always written
// to the response header so downstream consumers can correlate logs.
func Trace(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		traceID := r.Header.Get(TraceIDHeader)
		if traceID == "" {
			traceID = uuid.New().String()
			r.Header.Set(TraceIDHeader, traceID)
		}
		w.Header().Set(TraceIDHeader, traceID)
		next.ServeHTTP(w, r)
	})
}
