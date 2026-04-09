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
		t.Fatalf("expected status 400, got %d", w.Code)
	}
	if ct := w.Header().Get("Content-Type"); ct != "application/json" {
		t.Fatalf("expected Content-Type application/json, got %s", ct)
	}

	var body ErrorBody
	if err := json.NewDecoder(w.Body).Decode(&body); err != nil {
		t.Fatalf("failed to decode body: %v", err)
	}
	if body.Error.Code != "INVALID_INPUT" {
		t.Errorf("expected code INVALID_INPUT, got %s", body.Error.Code)
	}
	if body.Error.Message != "bad request" {
		t.Errorf("expected message 'bad request', got %s", body.Error.Message)
	}
}

func TestResponseWriteSuccess(t *testing.T) {
	w := httptest.NewRecorder()
	WriteSuccess(w, http.StatusCreated, map[string]string{"id": "123"})

	if w.Code != http.StatusCreated {
		t.Fatalf("expected status 201, got %d", w.Code)
	}

	var body SuccessBody
	if err := json.NewDecoder(w.Body).Decode(&body); err != nil {
		t.Fatalf("failed to decode body: %v", err)
	}
	data, ok := body.Data.(map[string]interface{})
	if !ok {
		t.Fatal("expected data to be a map")
	}
	if data["id"] != "123" {
		t.Errorf("expected id=123, got %v", data["id"])
	}
}

func TestResponseWriteList(t *testing.T) {
	w := httptest.NewRecorder()
	items := []string{"a", "b"}
	WriteList(w, items, 2, 10, 50)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", w.Code)
	}

	var body ListBody
	if err := json.NewDecoder(w.Body).Decode(&body); err != nil {
		t.Fatalf("failed to decode body: %v", err)
	}
	if body.Page != 2 {
		t.Errorf("expected page=2, got %d", body.Page)
	}
	if body.Size != 10 {
		t.Errorf("expected page_size=10, got %d", body.Size)
	}
	if body.Total != 50 {
		t.Errorf("expected total=50, got %d", body.Total)
	}
}

func TestResponseIsValidEmail(t *testing.T) {
	tests := []struct {
		email string
		want  bool
	}{
		{"user@example.com", true},
		{"a@b.c", true},
		{"user@domain.co.uk", true},
		{"noatsign.com", false},
		{"user@nodot", false},
		{"@example.com", false},
		{"", false},
	}
	for _, tt := range tests {
		got := isValidEmail(tt.email)
		if got != tt.want {
			t.Errorf("isValidEmail(%q) = %v, want %v", tt.email, got, tt.want)
		}
	}
}

func TestResponseParsePagination(t *testing.T) {
	t.Run("defaults", func(t *testing.T) {
		r := httptest.NewRequest(http.MethodGet, "/items", nil)
		page, size := parsePagination(r)
		if page != 1 {
			t.Errorf("expected default page=1, got %d", page)
		}
		if size != 20 {
			t.Errorf("expected default page_size=20, got %d", size)
		}
	})

	t.Run("custom values", func(t *testing.T) {
		r := httptest.NewRequest(http.MethodGet, "/items?page=3&page_size=50", nil)
		page, size := parsePagination(r)
		if page != 3 {
			t.Errorf("expected page=3, got %d", page)
		}
		if size != 50 {
			t.Errorf("expected page_size=50, got %d", size)
		}
	})

	t.Run("invalid values fall back to defaults", func(t *testing.T) {
		r := httptest.NewRequest(http.MethodGet, "/items?page=abc&page_size=-1", nil)
		page, size := parsePagination(r)
		if page != 1 {
			t.Errorf("expected default page=1 for invalid input, got %d", page)
		}
		if size != 20 {
			t.Errorf("expected default page_size=20 for negative input, got %d", size)
		}
	})
}
