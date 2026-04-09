package middleware

import (
	"fmt"
	"net"
	"net/http"
	"strings"
	"sync"

	"golang.org/x/time/rate"

	"rcprotocol/services/go-gateway/internal/response"
)

type ipLimiter struct {
	limiter *rate.Limiter
}

type perIPRateLimiter struct {
	mu       sync.Mutex
	limiters map[string]*ipLimiter
	rps      rate.Limit
	burst    int
}

func newPerIPRateLimiter(rps int, burst int) *perIPRateLimiter {
	return &perIPRateLimiter{
		limiters: make(map[string]*ipLimiter),
		rps:      rate.Limit(rps),
		burst:    burst,
	}
}

func (p *perIPRateLimiter) getLimiter(ip string) *rate.Limiter {
	p.mu.Lock()
	defer p.mu.Unlock()

	if v, exists := p.limiters[ip]; exists {
		return v.limiter
	}

	l := rate.NewLimiter(p.rps, p.burst)
	p.limiters[ip] = &ipLimiter{limiter: l}
	return l
}

// extractIP 从请求中提取客户端 IP（优先 X-Forwarded-For）
func extractIP(r *http.Request) string {
	if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
		// 取第一个 IP（最近的客户端）
		if idx := strings.Index(xff, ","); idx != -1 {
			return strings.TrimSpace(xff[:idx])
		}
		return strings.TrimSpace(xff)
	}
	ip, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return r.RemoteAddr
	}
	return ip
}

// RateLimit returns a per-IP rate limiting middleware.
// rps is the sustained requests-per-second per IP; burst is the maximum burst size.
// /healthz is exempt from rate limiting.
func RateLimit(rps int, burst int) func(http.Handler) http.Handler {
	limiter := newPerIPRateLimiter(rps, burst)

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.URL.Path == "/healthz" {
				next.ServeHTTP(w, r)
				return
			}

			ip := extractIP(r)
			ipLim := limiter.getLimiter(ip)

			w.Header().Set("X-RateLimit-Limit", fmt.Sprintf("%d", rps))

			if !ipLim.Allow() {
				traceID := r.Header.Get(TraceIDHeader)
				w.Header().Set("X-RateLimit-Remaining", "0")
				w.Header().Set("Retry-After", "1")
				response.WriteError(w, http.StatusTooManyRequests,
					response.CodeRateLimited, "rate limit exceeded", traceID)
				return
			}

			tokens := int(ipLim.Tokens())
			if tokens < 0 {
				tokens = 0
			}
			w.Header().Set("X-RateLimit-Remaining", fmt.Sprintf("%d", tokens))

			next.ServeHTTP(w, r)
		})
	}
}
