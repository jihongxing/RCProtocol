package config

import (
	"fmt"
	"os"
)

// Config holds the application configuration loaded from environment variables.
type Config struct {
	Port         string // 监听端口，默认 ":8084"
	DatabaseURL  string // PostgreSQL 连接字符串，必填
	RcApiBaseURL string // rc-api 上游地址，默认 "http://rc-api:8081"
}

// Load reads configuration from environment variables.
// Returns an error if APPROVAL_DATABASE_URL is not set.
func Load() (*Config, error) {
	dbURL := os.Getenv("APPROVAL_DATABASE_URL")
	if dbURL == "" {
		return nil, fmt.Errorf("APPROVAL_DATABASE_URL is required")
	}

	return &Config{
		Port:         envOrDefault("APPROVAL_PORT", ":8084"),
		DatabaseURL:  dbURL,
		RcApiBaseURL: envOrDefault("RC_API_BASE_URL", "http://rc-api:8081"),
	}, nil
}

func envOrDefault(key, defaultVal string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return defaultVal
}
