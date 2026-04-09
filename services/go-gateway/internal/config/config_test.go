package config

import (
	"os"
	"testing"
)

// clearConfigEnv removes all config-related environment variables.
func clearConfigEnv(t *testing.T) {
	t.Helper()
	for _, key := range []string{
		"RC_JWT_SECRET", "GATEWAY_PORT",
		"RC_API_UPSTREAM", "GO_BFF_UPSTREAM",
		"GO_IAM_UPSTREAM", "GO_APPROVAL_UPSTREAM", "GO_WORKORDER_UPSTREAM",
		"RATE_LIMIT_RPS", "RATE_LIMIT_BURST",
	} {
		t.Setenv(key, "")
		os.Unsetenv(key)
	}
}

func TestLoad_AllEnvSet(t *testing.T) {
	clearConfigEnv(t)

	t.Setenv("RC_JWT_SECRET", "test-secret-key")
	t.Setenv("GATEWAY_PORT", ":9090")
	t.Setenv("RC_API_UPSTREAM", "http://localhost:3001")
	t.Setenv("GO_BFF_UPSTREAM", "http://localhost:3002")
	t.Setenv("GO_IAM_UPSTREAM", "http://localhost:3003")
	t.Setenv("GO_APPROVAL_UPSTREAM", "http://localhost:3004")
	t.Setenv("GO_WORKORDER_UPSTREAM", "http://localhost:3005")
	t.Setenv("RATE_LIMIT_RPS", "50")
	t.Setenv("RATE_LIMIT_BURST", "150")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	checks := []struct {
		field string
		got   string
		want  string
	}{
		{"Port", cfg.Port, ":9090"},
		{"JWTSecret", cfg.JWTSecret, "test-secret-key"},
		{"RcApiUpstream", cfg.RcApiUpstream, "http://localhost:3001"},
		{"GoBffUpstream", cfg.GoBffUpstream, "http://localhost:3002"},
		{"GoIamUpstream", cfg.GoIamUpstream, "http://localhost:3003"},
		{"GoApprovalUpstream", cfg.GoApprovalUpstream, "http://localhost:3004"},
		{"GoWorkorderUpstream", cfg.GoWorkorderUpstream, "http://localhost:3005"},
	}
	for _, c := range checks {
		if c.got != c.want {
			t.Errorf("%s = %q, want %q", c.field, c.got, c.want)
		}
	}
	if cfg.RateLimitRPS != 50 {
		t.Errorf("RateLimitRPS = %d, want 50", cfg.RateLimitRPS)
	}
	if cfg.RateLimitBurst != 150 {
		t.Errorf("RateLimitBurst = %d, want 150", cfg.RateLimitBurst)
	}
}

func TestLoad_MissingJWTSecret(t *testing.T) {
	clearConfigEnv(t)

	_, err := Load()
	if err == nil {
		t.Fatal("expected error when RC_JWT_SECRET is missing, got nil")
	}
}

func TestLoad_DefaultValues(t *testing.T) {
	clearConfigEnv(t)

	t.Setenv("RC_JWT_SECRET", "secret")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if cfg.Port != ":8080" {
		t.Errorf("Port = %q, want %q", cfg.Port, ":8080")
	}
	if cfg.RcApiUpstream != "http://rc-api:8081" {
		t.Errorf("RcApiUpstream = %q, want %q", cfg.RcApiUpstream, "http://rc-api:8081")
	}
	if cfg.GoBffUpstream != "http://go-bff:8082" {
		t.Errorf("GoBffUpstream = %q, want %q", cfg.GoBffUpstream, "http://go-bff:8082")
	}
	if cfg.GoIamUpstream != "" {
		t.Errorf("GoIamUpstream = %q, want empty", cfg.GoIamUpstream)
	}
	if cfg.GoApprovalUpstream != "" {
		t.Errorf("GoApprovalUpstream = %q, want empty", cfg.GoApprovalUpstream)
	}
	if cfg.GoWorkorderUpstream != "" {
		t.Errorf("GoWorkorderUpstream = %q, want empty", cfg.GoWorkorderUpstream)
	}
	if cfg.RateLimitRPS != 100 {
		t.Errorf("RateLimitRPS = %d, want 100", cfg.RateLimitRPS)
	}
	if cfg.RateLimitBurst != 200 {
		t.Errorf("RateLimitBurst = %d, want 200", cfg.RateLimitBurst)
	}
}

func TestLoad_NonNumericRateLimitRPS(t *testing.T) {
	clearConfigEnv(t)

	t.Setenv("RC_JWT_SECRET", "secret")
	t.Setenv("RATE_LIMIT_RPS", "not-a-number")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if cfg.RateLimitRPS != 100 {
		t.Errorf("RateLimitRPS = %d, want 100 (default)", cfg.RateLimitRPS)
	}
}
