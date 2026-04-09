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

	resp := w.Result()
	defer resp.Body.Close()

	// Content-Type must be application/json; charset=utf-8
	ct := resp.Header.Get("Content-Type")
	if ct != "application/json; charset=utf-8" {
		t.Errorf("Content-Type = %q, want %q", ct, "application/json; charset=utf-8")
	}

	// Status code
	if resp.StatusCode != http.StatusBadRequest {
		t.Errorf("StatusCode = %d, want %d", resp.StatusCode, http.StatusBadRequest)
	}

	// Body must be valid JSON with error.code and error.message
	var body ErrorBody
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		t.Fatalf("failed to decode JSON: %v", err)
	}
	if body.Error.Code != "INVALID_INPUT" {
		t.Errorf("error.code = %q, want %q", body.Error.Code, "INVALID_INPUT")
	}
	if body.Error.Message != "bad request" {
		t.Errorf("error.message = %q, want %q", body.Error.Message, "bad request")
	}
}

func TestResponseWriteList(t *testing.T) {
	type item struct {
		ID string `json:"id"`
	}
	items := []item{{ID: "a1"}, {ID: "a2"}}

	w := httptest.NewRecorder()
	WriteList(w, items, 42, 2, 10)

	resp := w.Result()
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("StatusCode = %d, want %d", resp.StatusCode, http.StatusOK)
	}

	ct := resp.Header.Get("Content-Type")
	if ct != "application/json; charset=utf-8" {
		t.Errorf("Content-Type = %q, want %q", ct, "application/json; charset=utf-8")
	}

	var body struct {
		Items    []item `json:"items"`
		Total    int    `json:"total"`
		Page     int    `json:"page"`
		PageSize int    `json:"page_size"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		t.Fatalf("failed to decode JSON: %v", err)
	}
	if len(body.Items) != 2 {
		t.Errorf("items length = %d, want 2", len(body.Items))
	}
	if body.Total != 42 {
		t.Errorf("total = %d, want 42", body.Total)
	}
	if body.Page != 2 {
		t.Errorf("page = %d, want 2", body.Page)
	}
	if body.PageSize != 10 {
		t.Errorf("page_size = %d, want 10", body.PageSize)
	}
}

func TestResponseParsePagination(t *testing.T) {
	tests := []struct {
		name         string
		query        string
		wantPage     int
		wantPageSize int
	}{
		{
			name:         "defaults when no params",
			query:        "",
			wantPage:     1,
			wantPageSize: 20,
		},
		{
			name:         "explicit values",
			query:        "page=3&page_size=50",
			wantPage:     3,
			wantPageSize: 50,
		},
		{
			name:         "page<1 resets to 1",
			query:        "page=-5&page_size=10",
			wantPage:     1,
			wantPageSize: 10,
		},
		{
			name:         "page=0 resets to 1",
			query:        "page=0&page_size=10",
			wantPage:     1,
			wantPageSize: 10,
		},
		{
			name:         "page_size>100 capped to 100",
			query:        "page=1&page_size=200",
			wantPage:     1,
			wantPageSize: 100,
		},
		{
			name:         "non-numeric page uses default",
			query:        "page=abc&page_size=10",
			wantPage:     1,
			wantPageSize: 10,
		},
		{
			name:         "non-numeric page_size uses default",
			query:        "page=2&page_size=xyz",
			wantPage:     2,
			wantPageSize: 20,
		},
		{
			name:         "both non-numeric use defaults",
			query:        "page=foo&page_size=bar",
			wantPage:     1,
			wantPageSize: 20,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			url := "/test"
			if tt.query != "" {
				url += "?" + tt.query
			}
			r := httptest.NewRequest(http.MethodGet, url, nil)
			page, pageSize := ParsePagination(r)
			if page != tt.wantPage {
				t.Errorf("page = %d, want %d", page, tt.wantPage)
			}
			if pageSize != tt.wantPageSize {
				t.Errorf("pageSize = %d, want %d", pageSize, tt.wantPageSize)
			}
		})
	}
}
