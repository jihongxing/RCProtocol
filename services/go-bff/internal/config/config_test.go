package config

import (
	"os"
	"testing"
)

func TestLoad_WithAllEnvVars(t *testing.T) {
	os.Setenv("BFF_PORT", ":9999")
	os.Setenv("RC_API_BASE_URL", "http://custom-rc:3000")
	os.Setenv("GO_IAM_BASE_URL", "http://custom-iam:4000")
	defer func() {
		os.Unsetenv("BFF_PORT")
		os.Unsetenv("RC_API_BASE_URL")
		os.Unsetenv("GO_IAM_BASE_URL")
	}()

	cfg := Load()

	if cfg.Port != ":9999" {
		t.Errorf("Port = %q, want %q", cfg.Port, ":9999")
	}
	if cfg.RcApiBaseURL != "http://custom-rc:3000" {
		t.Errorf("RcApiBaseURL = %q, want %q", cfg.RcApiBaseURL, "http://custom-rc:3000")
	}
	if cfg.GoIamBaseURL != "http://custom-iam:4000" {
		t.Errorf("GoIamBaseURL = %q, want %q", cfg.GoIamBaseURL, "http://custom-iam:4000")
	}
}

func TestLoad_Defaults(t *testing.T) {
	os.Unsetenv("BFF_PORT")
	os.Unsetenv("RC_API_BASE_URL")
	os.Unsetenv("GO_IAM_BASE_URL")

	cfg := Load()

	if cfg.Port != ":8082" {
		t.Errorf("Port = %q, want default %q", cfg.Port, ":8082")
	}
	if cfg.RcApiBaseURL != "http://rc-api:8081" {
		t.Errorf("RcApiBaseURL = %q, want default %q", cfg.RcApiBaseURL, "http://rc-api:8081")
	}
	if cfg.GoIamBaseURL != "http://go-iam:8083" {
		t.Errorf("GoIamBaseURL = %q, want default %q", cfg.GoIamBaseURL, "http://go-iam:8083")
	}
}
