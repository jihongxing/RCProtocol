package middleware

import (
	"log/slog"
	"net/http"
	"time"
)

// statusRecorder wraps http.ResponseWriter to capture the status code
// written by downstream handlers, enabling the logging middleware to
// report it after the response is complete.
type statusRecorder struct {
	http.ResponseWriter
	statusCode int
}

func (sr *statusRecorder) WriteHeader(code int) {
	sr.statusCode = code
	sr.ResponseWriter.WriteHeader(code)
}

// Unwrap exposes the underlying ResponseWriter so that
// httputil.ReverseProxy (via http.ResponseController) can detect
// the Flusher interface for chunked/streaming responses.
func (sr *statusRecorder) Unwrap() http.ResponseWriter {
	return sr.ResponseWriter
}

// Logging returns middleware that emits a structured JSON log entry for
// every completed request. Fields: method, path, status, latency_ms,
// trace_id, client_ip. Authorization values and request bodies are
// never logged. Log level: 2xx/3xx → INFO, 4xx → WARN, 5xx → ERROR.
func Logging(logger *slog.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			start := time.Now()

			recorder := &statusRecorder{
				ResponseWriter: w,
				statusCode:     http.StatusOK,
			}

			next.ServeHTTP(recorder, r)

			latency := time.Since(start)
			traceID := r.Header.Get(TraceIDHeader)

			level := slog.LevelInfo
			if recorder.statusCode >= 400 {
				level = slog.LevelWarn
			}
			if recorder.statusCode >= 500 {
				level = slog.LevelError
			}

			logger.LogAttrs(r.Context(), level, "request completed",
				slog.String("method", r.Method),
				slog.String("path", r.URL.Path),
				slog.Int("status", recorder.statusCode),
				slog.Int64("latency_ms", latency.Milliseconds()),
				slog.String("trace_id", traceID),
				slog.String("client_ip", r.RemoteAddr),
				slog.Bool("api_key_auth", r.Header.Get(ApiKeyHeader) != ""),
			)
		})
	}
}
