package middleware

import (
	"log/slog"
	"net/http"
	"time"
)

// statusRecorder wraps ResponseWriter to capture the HTTP status code.
type statusRecorder struct {
	http.ResponseWriter
	statusCode int
}

func (r *statusRecorder) WriteHeader(code int) {
	r.statusCode = code
	r.ResponseWriter.WriteHeader(code)
}

// Logging returns a middleware that logs each HTTP request.
// Logged fields: method, path, status, latency_ms, client_ip.
// Sensitive fields (password, password_hash, JWT, secret) are never logged.
func Logging(logger *slog.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			start := time.Now()
			recorder := &statusRecorder{ResponseWriter: w, statusCode: http.StatusOK}

			next.ServeHTTP(recorder, r)

			latency := time.Since(start).Milliseconds()
			attrs := []slog.Attr{
				slog.String("method", r.Method),
				slog.String("path", r.URL.Path),
				slog.Int("status", recorder.statusCode),
				slog.Int64("latency_ms", latency),
				slog.String("client_ip", r.RemoteAddr),
			}

			msg := "request"
			switch {
			case recorder.statusCode >= 500:
				logger.LogAttrs(r.Context(), slog.LevelError, msg, attrs...)
			case recorder.statusCode >= 400:
				logger.LogAttrs(r.Context(), slog.LevelWarn, msg, attrs...)
			default:
				logger.LogAttrs(r.Context(), slog.LevelInfo, msg, attrs...)
			}
		})
	}
}
