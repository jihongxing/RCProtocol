package config

import (
	"os"
	"testing"
)

// setEnv sets env vars for a test and returns a cleanup function.
func setEnv(t *testing.T, kvs map[string]string) {
	t.Helper()
	for k, v := range kvs {
		t.Setenv(k, v)
	}
}

func TestLoad_AllEnvVars(t *testing.T) {
	setEnv(t, map[string]string{
		"RC_JWT_SECRET":    "test-secret-key",
		"IAM_DATABASE_URL": "postgres://u:p@localhost:5432/testdb",
		"GO_IAM_PORT":      ":9090",
		"JWT_EXPIRY_HOURS": "48",
	})

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() unexpected error: %v", err)
	}
	if cfg.JWTSecret != "test-secret-key" {
		t.Errorf("JWTSecret = %q, want %q", cfg.JWTSecret, "test-secret-key")
	}
	if cfg.DatabaseURL != "postgres://u:p@localhost:5432/testdb" {
		t.Errorf("DatabaseURL = %q, want %q", cfg.DatabaseURL, "postgres://u:p@localhost:5432/testdb")
	}
	if cfg.Port != ":9090" {
		t.Errorf("Port = %q, want %q", cfg.Port, ":9090")
	}
	if cfg.JWTExpiryHours != 48 {
		t.Errorf("JWTExpiryHours = %d, want %d", cfg.JWTExpiryHours, 48)
	}
}

func TestLoad_MissingJWTSecret(t *testing.T) {
	setEnv(t, map[string]string{
		"IAM_DATABASE_URL": "postgres://u:p@localhost:5432/testdb",
	})
	os.Unsetenv("RC_JWT_SECRET")

	_, err := Load()
	if err == nil {
		t.Fatal("Load() expected error for missing RC_JWT_SECRET, got nil")
	}
	if err.Error() != "RC_JWT_SECRET is required" {
		t.Errorf("error = %q, want %q", err.Error(), "RC_JWT_SECRET is required")
	}
}

func TestLoad_MissingDatabaseURL(t *testing.T) {
	setEnv(t, map[string]string{
		"RC_JWT_SECRET": "test-secret-key",
	})
	os.Unsetenv("IAM_DATABASE_URL")

	_, err := Load()
	if err == nil {
		t.Fatal("Load() expected error for missing IAM_DATABASE_URL, got nil")
	}
	if err.Error() != "IAM_DATABASE_URL is required" {
		t.Errorf("error = %q, want %q", err.Error(), "IAM_DATABASE_URL is required")
	}
}

func TestLoad_Defaults(t *testing.T) {
	setEnv(t, map[string]string{
		"RC_JWT_SECRET":    "secret",
		"IAM_DATABASE_URL": "postgres://localhost/db",
	})
	os.Unsetenv("GO_IAM_PORT")
	os.Unsetenv("JWT_EXPIRY_HOURS")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() unexpected error: %v", err)
	}
	if cfg.Port != ":8083" {
		t.Errorf("Port = %q, want default %q", cfg.Port, ":8083")
	}
	if cfg.JWTExpiryHours != 24 {
		t.Errorf("JWTExpiryHours = %d, want default %d", cfg.JWTExpiryHours, 24)
	}
}
