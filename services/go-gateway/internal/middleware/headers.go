package middleware

import (
	"net/http"

	"rcprotocol/services/go-gateway/internal/response"
)

// writeMethods lists HTTP methods considered write operations that
// require an X-Idempotency-Key header for idempotent retry safety.
var writeMethods = map[string]bool{
	http.MethodPost:   true,
	http.MethodPut:    true,
	http.MethodPatch:  true,
	http.MethodDelete: true,
}

// WriteHeaders enforces header requirements on write operations.
// Public routes are passed through unconditionally.
// For POST/PUT/PATCH/DELETE to non-public routes, the middleware
// requires X-Idempotency-Key; missing it yields 400 INVALID_INPUT.
func WriteHeaders(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if isPublicPath(r.URL.Path) {
			next.ServeHTTP(w, r)
			return
		}

		if writeMethods[r.Method] {
			if r.Header.Get("X-Idempotency-Key") == "" {
				traceID := r.Header.Get(TraceIDHeader)
				response.WriteError(w, http.StatusBadRequest,
					response.CodeInvalidInput,
					"missing required header: X-Idempotency-Key", traceID)
				return
			}
		}

		next.ServeHTTP(w, r)
	})
}
