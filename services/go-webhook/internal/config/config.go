package config

import (
	"fmt"
	"os"
	"strconv"
)

type Config struct {
	DatabaseURL string
	WorkerCount int
	RetryConfig RetryConfig
}

type RetryConfig struct {
	MaxRetries      int
	BackoffSeconds  []int
	TimeoutSeconds  int
}

func Load() (*Config, error) {
	databaseURL := os.Getenv("WEBHOOK_DATABASE_URL")
	if databaseURL == "" {
		return nil, fmt.Errorf("WEBHOOK_DATABASE_URL is required")
	}

	workerCount := getEnvInt("WEBHOOK_WORKER_COUNT", 5)
	maxRetries := getEnvInt("WEBHOOK_MAX_RETRIES", 3)
	timeoutSeconds := getEnvInt("WEBHOOK_TIMEOUT_SECONDS", 30)

	return &Config{
		DatabaseURL: databaseURL,
		WorkerCount: workerCount,
		RetryConfig: RetryConfig{
			MaxRetries:     maxRetries,
			BackoffSeconds: []int{60, 300, 900}, // 1min, 5min, 15min
			TimeoutSeconds: timeoutSeconds,
		},
	}, nil
}

func getEnvInt(key string, defaultValue int) int {
	val := os.Getenv(key)
	if val == "" {
		return defaultValue
	}
	intVal, err := strconv.Atoi(val)
	if err != nil {
		return defaultValue
	}
	return intVal
}
