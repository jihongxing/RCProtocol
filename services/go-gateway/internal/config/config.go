package config

import (
	"fmt"
	"os"
	"strconv"
)

type Config struct {
	Port                string
	JWTSecret           string
	RcApiUpstream       string
	GoBffUpstream       string
	GoIamUpstream       string
	GoApprovalUpstream  string // ARCHIVED: Phase 2 removed approval workflow, kept for backward compatibility
	GoWorkorderUpstream string
	RateLimitRPS        int
	RateLimitBurst      int
}

func Load() (*Config, error) {
	jwtSecret := os.Getenv("RC_JWT_SECRET")
	if jwtSecret == "" {
		return nil, fmt.Errorf("RC_JWT_SECRET is required but not set")
	}

	return &Config{
		Port:                envOrDefault("GATEWAY_PORT", ":8080"),
		JWTSecret:           jwtSecret,
		RcApiUpstream:       envOrDefault("RC_API_UPSTREAM", "http://rc-api:8081"),
		GoBffUpstream:       envOrDefault("GO_BFF_UPSTREAM", "http://go-bff:8082"),
		GoIamUpstream:       os.Getenv("GO_IAM_UPSTREAM"),
		GoApprovalUpstream:  os.Getenv("GO_APPROVAL_UPSTREAM"),
		GoWorkorderUpstream: os.Getenv("GO_WORKORDER_UPSTREAM"),
		RateLimitRPS:        envIntOrDefault("RATE_LIMIT_RPS", 100),
		RateLimitBurst:      envIntOrDefault("RATE_LIMIT_BURST", 200),
	}, nil
}

func envOrDefault(key, defaultVal string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return defaultVal
}

func envIntOrDefault(key string, defaultVal int) int {
	v := os.Getenv(key)
	if v == "" {
		return defaultVal
	}
	n, err := strconv.Atoi(v)
	if err != nil {
		return defaultVal
	}
	return n
}
