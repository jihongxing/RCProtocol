package handler

import (
	"strings"
	"testing"
)

func TestGenerateApiKey_LegacyBackofficeOrgScopedFormat(t *testing.T) {
	key, err := generateApiKey("org-123")
	if err != nil {
		t.Fatalf("generateApiKey error: %v", err)
	}
	if !strings.HasPrefix(key, "brand_org-123_") {
		t.Fatalf("expected legacy org-scoped prefix, got %q", key)
	}
	orgID, err := extractOrgIDFromKey(key)
	if err != nil {
		t.Fatalf("extractOrgIDFromKey error: %v", err)
	}
	if orgID != "org-123" {
		t.Fatalf("expected org-123, got %q", orgID)
	}
	if strings.HasPrefix(key, "rcpk_live_") {
		t.Fatalf("legacy go-iam key must not look like current gateway key: %q", key)
	}
	if len(key) <= len("brand_org-123_") {
		t.Fatalf("expected random suffix in key, got %q", key)
	}
}

func TestExtractOrgIDFromKey_RejectsCurrentGatewayKeyFormat(t *testing.T) {
	_, err := extractOrgIDFromKey("rcpk_live_1234567890abcdef1234567890abcdef")
	if err == nil {
		t.Fatal("expected current gateway key format to be rejected by legacy extractor")
	}
}
