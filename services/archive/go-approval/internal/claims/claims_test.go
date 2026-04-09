package claims

import (
	"net/http"
	"testing"
)

func TestFromRequest_AllHeaders(t *testing.T) {
	req, _ := http.NewRequest("GET", "/test", nil)
	req.Header.Set("X-Claims-Sub", "user-123")
	req.Header.Set("X-Claims-Role", "Platform")
	req.Header.Set("X-Claims-Org-Id", "org-456")

	c := FromRequest(req)

	if c.Sub != "user-123" {
		t.Errorf("expected Sub=user-123, got %q", c.Sub)
	}
	if c.Role != "Platform" {
		t.Errorf("expected Role=Platform, got %q", c.Role)
	}
	if c.OrgID != "org-456" {
		t.Errorf("expected OrgID=org-456, got %q", c.OrgID)
	}
	if !c.Valid() {
		t.Error("expected Valid()=true when all headers present")
	}
}

func TestFromRequest_MissingSub(t *testing.T) {
	req, _ := http.NewRequest("GET", "/test", nil)
	req.Header.Set("X-Claims-Role", "Platform")
	req.Header.Set("X-Claims-Org-Id", "org-456")

	c := FromRequest(req)

	if c.Sub != "" {
		t.Errorf("expected empty Sub, got %q", c.Sub)
	}
	if c.Valid() {
		t.Error("expected Valid()=false when Sub is missing")
	}
}

func TestFromRequest_MissingRole(t *testing.T) {
	req, _ := http.NewRequest("GET", "/test", nil)
	req.Header.Set("X-Claims-Sub", "user-123")
	req.Header.Set("X-Claims-Org-Id", "org-456")

	c := FromRequest(req)

	if c.Role != "" {
		t.Errorf("expected empty Role, got %q", c.Role)
	}
	if c.Valid() {
		t.Error("expected Valid()=false when Role is missing")
	}
}

func TestFromRequest_MissingOrgID(t *testing.T) {
	req, _ := http.NewRequest("GET", "/test", nil)
	req.Header.Set("X-Claims-Sub", "user-123")
	req.Header.Set("X-Claims-Role", "Platform")

	c := FromRequest(req)

	if c.OrgID != "" {
		t.Errorf("expected empty OrgID, got %q", c.OrgID)
	}
	if c.Valid() {
		t.Error("expected Valid()=false when OrgID is missing")
	}
}

func TestFromRequest_NoHeaders(t *testing.T) {
	req, _ := http.NewRequest("GET", "/test", nil)

	c := FromRequest(req)

	if c.Valid() {
		t.Error("expected Valid()=false when no headers present")
	}
}
