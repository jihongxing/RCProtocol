package handler

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestResponseWriteError(t *testing.T) {
	w := httptest.NewRecorder()
	WriteError(w, http.StatusBadRequest, "INVALID_INPUT", "bad request")

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected status 400, got %d", w.Code)
	}

	ct := w.Header().Get("Content-Type")
	if ct != "application/json; charset=utf-8" {
		t.Errorf("expected Content-Type application/json; charset=utf-8, got %q", ct)
	}

	var body ErrorBody
	if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil {
		t.Fatalf("response is not valid JSON: %v", err)
	}

	if body.Error.Code != "INVALID_INPUT" {
		t.Errorf("expected error code INVALID_INPUT, got %q", body.Error.Code)
	}
	if body.Error.Message != "bad request" {
		t.Errorf("expected error message 'bad request', got %q", body.Error.Message)
	}
}

func TestResponseWriteList(t *testing.T) {
	w := httptest.NewRecorder()
	items := []string{"a", "b", "c"}
	WriteList(w, items, 30, 2, 10)

	if w.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", w.Code)
	}

	var body map[string]json.RawMessage
	if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil {
		t.Fatalf("response is not valid JSON: %v", err)
	}

	// Verify all required fields exist
	for _, key := range []string{"items", "total", "page", "page_size"} {
		if _, ok := body[key]; !ok {
			t.Errorf("expected response to contain %q", key)
		}
	}

	var parsed ListBody
	if err := json.Unmarshal(w.Body.Bytes(), &parsed); err != nil {
		t.Fatalf("failed to parse ListBody: %v", err)
	}
	if parsed.Total != 30 {
		t.Errorf("expected total=30, got %d", parsed.Total)
	}
	if parsed.Page != 2 {
		t.Errorf("expected page=2, got %d", parsed.Page)
	}
	if parsed.PageSize != 10 {
		t.Errorf("expected page_size=10, got %d", parsed.PageSize)
	}
}

func TestResponseParsePagination_Defaults(t *testing.T) {
	req := httptest.NewRequest("GET", "/workorders", nil)
	page, pageSize := ParsePagination(req)

	if page != 1 {
		t.Errorf("expected default page=1, got %d", page)
	}
	if pageSize != 20 {
		t.Errorf("expected default page_size=20, got %d", pageSize)
	}
}

func TestResponseParsePagination_CustomValues(t *testing.T) {
	req := httptest.NewRequest("GET", "/workorders?page=3&page_size=50", nil)
	page, pageSize := ParsePagination(req)

	if page != 3 {
		t.Errorf("expected page=3, got %d", page)
	}
	if pageSize != 50 {
		t.Errorf("expected page_size=50, got %d", pageSize)
	}
}

func TestResponseParsePagination_PageLessThanOne(t *testing.T) {
	req := httptest.NewRequest("GET", "/workorders?page=0", nil)
	page, _ := ParsePagination(req)

	if page != 1 {
		t.Errorf("expected page=1 when page<1, got %d", page)
	}
}

func TestResponseParsePagination_PageSizeExceedsMax(t *testing.T) {
	req := httptest.NewRequest("GET", "/workorders?page_size=200", nil)
	_, pageSize := ParsePagination(req)

	if pageSize != 100 {
		t.Errorf("expected page_size capped at 100, got %d", pageSize)
	}
}

func TestResponseParsePagination_NegativePageSize(t *testing.T) {
	// Non-numeric input falls back to default
	req := httptest.NewRequest("GET", "/workorders?page_size=-5", nil)
	_, pageSize := ParsePagination(req)

	if pageSize != 20 {
		t.Errorf("expected default page_size=20 for invalid input, got %d", pageSize)
	}
}

func TestResponseParsePagination_NonNumericInput(t *testing.T) {
	req := httptest.NewRequest("GET", "/workorders?page=abc&page_size=xyz", nil)
	page, pageSize := ParsePagination(req)

	if page != 1 {
		t.Errorf("expected default page=1 for non-numeric input, got %d", page)
	}
	if pageSize != 20 {
		t.Errorf("expected default page_size=20 for non-numeric input, got %d", pageSize)
	}
}
