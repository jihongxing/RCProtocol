package handler

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

// Phase 2 Tests: Brand Simplified Registration

func TestOrgCreate201_BrandAutoGenerateBrandID(t *testing.T) {
	mock := newMockOrgRepo()
	h := NewOrgHandler(mock)

	body := `{"org_name":"Auto Brand","org_type":"brand","contact_email":"auto@example.com","contact_phone":"1234567890"}`
	req := httptest.NewRequest(http.MethodPost, "/orgs", bytes.NewBufferString(body))
	w := httptest.NewRecorder()

	h.Create(w, req)

	if w.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d: %s", w.Code, w.Body.String())
	}

	var resp SuccessBody
	_ = json.Unmarshal(w.Body.Bytes(), &resp)
	data := resp.Data.(map[string]interface{})

	brandID, ok := data["brand_id"].(string)
	if !ok || brandID == "" {
		t.Errorf("expected auto-generated brand_id, got %v", data["brand_id"])
	}

	// Verify format: brand-{timestamp}-{random6}
	if len(brandID) < 15 || brandID[:6] != "brand-" {
		t.Errorf("expected brand_id format 'brand-{timestamp}-{random6}', got %s", brandID)
	}
}

func TestOrgCreate400_BrandMissingContactEmail(t *testing.T) {
	mock := newMockOrgRepo()
	h := NewOrgHandler(mock)

	body := `{"org_name":"Brand Org","org_type":"brand","contact_phone":"1234567890"}`
	req := httptest.NewRequest(http.MethodPost, "/orgs", bytes.NewBufferString(body))
	w := httptest.NewRecorder()

	h.Create(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d: %s", w.Code, w.Body.String())
	}

	var resp ErrorBody
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if resp.Error.Message != "contact_email is required for brand organizations" {
		t.Errorf("expected 'contact_email is required for brand organizations', got %q", resp.Error.Message)
	}
}

func TestOrgCreate400_BrandMissingContactPhone(t *testing.T) {
	mock := newMockOrgRepo()
	h := NewOrgHandler(mock)

	body := `{"org_name":"Brand Org","org_type":"brand","contact_email":"test@example.com"}`
	req := httptest.NewRequest(http.MethodPost, "/orgs", bytes.NewBufferString(body))
	w := httptest.NewRecorder()

	h.Create(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d: %s", w.Code, w.Body.String())
	}

	var resp ErrorBody
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if resp.Error.Message != "contact_phone is required for brand organizations" {
		t.Errorf("expected 'contact_phone is required for brand organizations', got %q", resp.Error.Message)
	}
}

func TestOrgCreate201_BrandProvidedBrandID(t *testing.T) {
	mock := newMockOrgRepo()
	h := NewOrgHandler(mock)

	// If brand_id is provided, use it
	body := `{"org_name":"Custom Brand","org_type":"brand","brand_id":"custom-brand-123","contact_email":"custom@example.com","contact_phone":"1234567890"}`
	req := httptest.NewRequest(http.MethodPost, "/orgs", bytes.NewBufferString(body))
	w := httptest.NewRecorder()

	h.Create(w, req)

	if w.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d: %s", w.Code, w.Body.String())
	}

	var resp SuccessBody
	_ = json.Unmarshal(w.Body.Bytes(), &resp)
	data := resp.Data.(map[string]interface{})

	if data["brand_id"] != "custom-brand-123" {
		t.Errorf("expected brand_id 'custom-brand-123', got %v", data["brand_id"])
	}
}

func TestOrgCreate201_NonBrandNoContactRequired(t *testing.T) {
	mock := newMockOrgRepo()
	h := NewOrgHandler(mock)

	// Platform and factory orgs don't require contact fields
	body := `{"org_name":"Factory Org","org_type":"factory"}`
	req := httptest.NewRequest(http.MethodPost, "/orgs", bytes.NewBufferString(body))
	w := httptest.NewRecorder()

	h.Create(w, req)

	if w.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d: %s", w.Code, w.Body.String())
	}
}
