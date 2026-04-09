package config

import (
	"fmt"
	"os"
	"strconv"
)

type Config struct {
	Port           string
	DatabaseURL    string
	JWTSecret      string
	JWTExpiryHours int
}

func Load() (*Config, error) {
	jwtSecret := os.Getenv("RC_JWT_SECRET")
	if jwtSecret == "" {
		return nil, fmt.Errorf("RC_JWT_SECRET is required")
	}

	databaseURL := os.Getenv("IAM_DATABASE_URL")
	if databaseURL == "" {
		return nil, fmt.Errorf("IAM_DATABASE_URL is required")
	}

	return &Config{
		Port:           envOrDefault("GO_IAM_PORT", ":8083"),
		DatabaseURL:    databaseURL,
		JWTSecret:      jwtSecret,
		JWTExpiryHours: envIntOrDefault("JWT_EXPIRY_HOURS", 24),
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
