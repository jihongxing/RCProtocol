package middleware

import (
	"log/slog"
	"net/http"
	"time"
)

type statusRecorder struct {
	http.ResponseWriter
	statusCode int
}

func (sr *statusRecorder) WriteHeader(code int) {
	sr.statusCode = code
	sr.ResponseWriter.WriteHeader(code)
}

// Logging 返回请求日志中间件，记录 method/path/status/latency_ms；不记录 Authorization
func Logging(logger *slog.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			start := time.Now()
			recorder := &statusRecorder{ResponseWriter: w, statusCode: http.StatusOK}
			next.ServeHTTP(recorder, r)
			latency := time.Since(start)

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
			)
		})
	}
}
