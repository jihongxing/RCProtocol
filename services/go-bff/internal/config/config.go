package config

import "os"

// Config holds all BFF service configuration.
// All fields have sensible defaults for docker-compose network.
type Config struct {
	Port         string
	RcApiBaseURL string
	GoIamBaseURL string
}

// Load reads configuration from environment variables.
// Returns *Config (never error) — BFF has no mandatory config.
func Load() *Config {
	return &Config{
		Port:         envOrDefault("BFF_PORT", ":8082"),
		RcApiBaseURL: envOrDefault("RC_API_BASE_URL", "http://rc-api:8081"),
		GoIamBaseURL: envOrDefault("GO_IAM_BASE_URL", "http://go-iam:8083"),
	}
}

func envOrDefault(key, defaultVal string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return defaultVal
}
