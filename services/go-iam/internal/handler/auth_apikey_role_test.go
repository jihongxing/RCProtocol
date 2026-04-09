package handler

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"rcprotocol/services/go-iam/internal/auth"
)

func TestValidateApiKey_RejectsCurrentGatewayKeyFormat(t *testing.T) {
	issuer := auth.NewIssuer("test-secret-key", 24)
	h := NewAuthHandler(newMockUserRepo(), newMockMemberRepo(), nil, nil, issuer)

	req := httptest.NewRequest(http.MethodPost, "/auth/validate-api-key", bytes.NewBufferString(`{"api_key":"rcpk_live_1234567890abcdef1234567890abcdef"}`))
	w := httptest.NewRecorder()
	h.ValidateApiKey(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d: %s", w.Code, w.Body.String())
	}
	var resp ErrorBody
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if resp.Error.Code != "INVALID_API_KEY" {
		t.Fatalf("expected INVALID_API_KEY, got %s", resp.Error.Code)
	}
}
