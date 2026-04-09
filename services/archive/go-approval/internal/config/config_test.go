package config

import (
	"os"
	"testing"

	"pgregory.net/rapid"
)

func TestLoad_AllEnvVarsSet(t *testing.T) {
	t.Setenv("APPROVAL_DATABASE_URL", "postgres://user:pass@localhost:5432/testdb")
	t.Setenv("APPROVAL_PORT", ":9090")
	t.Setenv("RC_API_BASE_URL", "http://localhost:8081")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if cfg.DatabaseURL != "postgres://user:pass@localhost:5432/testdb" {
		t.Errorf("DatabaseURL = %q, want %q", cfg.DatabaseURL, "postgres://user:pass@localhost:5432/testdb")
	}
	if cfg.Port != ":9090" {
		t.Errorf("Port = %q, want %q", cfg.Port, ":9090")
	}
	if cfg.RcApiBaseURL != "http://localhost:8081" {
		t.Errorf("RcApiBaseURL = %q, want %q", cfg.RcApiBaseURL, "http://localhost:8081")
	}
}

func TestLoad_MissingDatabaseURL(t *testing.T) {
	os.Unsetenv("APPROVAL_DATABASE_URL")

	cfg, err := Load()
	if err == nil {
		t.Fatal("expected error when APPROVAL_DATABASE_URL is missing, got nil")
	}
	if cfg != nil {
		t.Errorf("expected nil config, got %+v", cfg)
	}
}

func TestLoad_DefaultValues(t *testing.T) {
	t.Setenv("APPROVAL_DATABASE_URL", "postgres://user:pass@localhost:5432/testdb")
	os.Unsetenv("APPROVAL_PORT")
	os.Unsetenv("RC_API_BASE_URL")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if cfg.Port != ":8084" {
		t.Errorf("Port = %q, want default %q", cfg.Port, ":8084")
	}
	if cfg.RcApiBaseURL != "http://rc-api:8081" {
		t.Errorf("RcApiBaseURL = %q, want default %q", cfg.RcApiBaseURL, "http://rc-api:8081")
	}
}

// TestConfigRequiredFieldValidation is a property-based test that verifies:
// - When APPROVAL_DATABASE_URL is empty, Load returns an error
// - When APPROVAL_DATABASE_URL is non-empty, Load returns a valid Config with correct DatabaseURL
//
// **Property 1: Config 必填项校验**
// **Validates: FR-01 (1.1, 1.2)**
func TestConfigRequiredFieldValidation(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		// Randomly decide whether to set DatabaseURL
		setDB := rapid.Bool().Draw(t, "setDatabaseURL")
		dbURL := rapid.StringMatching(`[a-zA-Z0-9:/._\-]{1,100}`).Draw(t, "dbURL")

		// Randomly set/clear optional env vars
		setPort := rapid.Bool().Draw(t, "setPort")
		portVal := rapid.StringMatching(`:[0-9]{4,5}`).Draw(t, "portVal")

		setRcApi := rapid.Bool().Draw(t, "setRcApi")
		rcApiVal := rapid.StringMatching(`http://[a-z\-]+:[0-9]{4}`).Draw(t, "rcApiVal")

		// Apply environment
		if setDB {
			os.Setenv("APPROVAL_DATABASE_URL", dbURL)
		} else {
			os.Unsetenv("APPROVAL_DATABASE_URL")
		}
		if setPort {
			os.Setenv("APPROVAL_PORT", portVal)
		} else {
			os.Unsetenv("APPROVAL_PORT")
		}
		if setRcApi {
			os.Setenv("RC_API_BASE_URL", rcApiVal)
		} else {
			os.Unsetenv("RC_API_BASE_URL")
		}

		// Cleanup after this iteration
		defer func() {
			os.Unsetenv("APPROVAL_DATABASE_URL")
			os.Unsetenv("APPROVAL_PORT")
			os.Unsetenv("RC_API_BASE_URL")
		}()

		cfg, err := Load()

		if !setDB {
			// DatabaseURL not set → must return error
			if err == nil {
				t.Fatal("expected error when APPROVAL_DATABASE_URL is not set")
			}
			if cfg != nil {
				t.Fatal("expected nil config when APPROVAL_DATABASE_URL is not set")
			}
		} else {
			// DatabaseURL set → must return valid config
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if cfg == nil {
				t.Fatal("expected non-nil config")
			}
			if cfg.DatabaseURL != dbURL {
				t.Fatalf("DatabaseURL = %q, want %q", cfg.DatabaseURL, dbURL)
			}
			// Port should be custom value or default
			if setPort {
				if cfg.Port != portVal {
					t.Fatalf("Port = %q, want %q", cfg.Port, portVal)
				}
			} else {
				if cfg.Port != ":8084" {
					t.Fatalf("Port = %q, want default %q", cfg.Port, ":8084")
				}
			}
			// RcApiBaseURL should be custom value or default
			if setRcApi {
				if cfg.RcApiBaseURL != rcApiVal {
					t.Fatalf("RcApiBaseURL = %q, want %q", cfg.RcApiBaseURL, rcApiVal)
				}
			} else {
				if cfg.RcApiBaseURL != "http://rc-api:8081" {
					t.Fatalf("RcApiBaseURL = %q, want default %q", cfg.RcApiBaseURL, "http://rc-api:8081")
				}
			}
		}
	})
}
