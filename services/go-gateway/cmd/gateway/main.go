package main

import (
	"encoding/hex"
	"fmt"
	"log/slog"
	"net/http"
	"os"

	"rcprotocol/services/go-gateway/internal/config"
	"rcprotocol/services/go-gateway/internal/middleware"
	"rcprotocol/services/go-gateway/internal/proxy"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		fmt.Fprintf(os.Stderr, "config error: %v\n", err)
		os.Exit(1)
	}

	validateSecurityConfig(cfg.JWTSecret)

	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	}))
	slog.SetDefault(logger)

	router := proxy.NewRouter(cfg)

	// Middleware chain assembled from inner to outer.
	// Execution order (outer → inner): Logging → Trace → RateLimit → Auth → WriteHeaders → Router
	var handler http.Handler = router
	handler = middleware.WriteHeaders(handler)
	handler = middleware.Auth(cfg.JWTSecret)(handler)
	handler = middleware.RateLimit(cfg.RateLimitRPS, cfg.RateLimitBurst)(handler)
	handler = middleware.Trace(handler)
	handler = middleware.Logging(logger)(handler)

	logger.Info("go-gateway starting",
		slog.String("port", cfg.Port),
		slog.String("rc_api_upstream", cfg.RcApiUpstream),
		slog.String("go_bff_upstream", cfg.GoBffUpstream),
	)

	if err := http.ListenAndServe(cfg.Port, handler); err != nil {
		logger.Error("server failed", slog.String("error", err.Error()))
		os.Exit(1)
	}
}

// validateSecurityConfig 拒绝弱密钥启动
func validateSecurityConfig(jwtSecret string) {
	if len(jwtSecret) < 32 {
		fmt.Fprintf(os.Stderr, "RC_JWT_SECRET 长度不足 32 字节（当前 %d 字节）\n", len(jwtSecret))
		os.Exit(1)
	}

	rootKeyHex := os.Getenv("RC_ROOT_KEY_HEX")
	if rootKeyHex != "" {
		bytes, err := hex.DecodeString(rootKeyHex)
		if err == nil && len(bytes) > 0 {
			allZero := true
			sequential := true
			for i, b := range bytes {
				if b != 0 {
					allZero = false
				}
				if b != byte(i) {
					sequential = false
				}
			}
			if allZero {
				fmt.Fprintln(os.Stderr, "RC_ROOT_KEY_HEX 为全零，拒绝启动")
				os.Exit(1)
			}
			if sequential {
				fmt.Fprintln(os.Stderr, "RC_ROOT_KEY_HEX 为递增序列，拒绝启动")
				os.Exit(1)
			}
		}
	}
}
