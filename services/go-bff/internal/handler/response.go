package handler

import (
	"encoding/json"
	"net/http"
	"strconv"
)

// ErrorBody wraps an error detail for unified JSON error responses.
type ErrorBody struct {
	Error ErrorDetail `json:"error"`
}

// ErrorDetail contains the error code and human-readable message.
type ErrorDetail struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

// ListBody is the standard paginated list response envelope.
type ListBody struct {
	Items    interface{} `json:"items"`
	Total    int         `json:"total"`
	Page     int         `json:"page"`
	PageSize int         `json:"page_size"`
}

// WriteJSON writes a JSON response with the given status code.
func WriteJSON(w http.ResponseWriter, status int, v interface{}) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

// WriteError writes a unified error JSON response.
func WriteError(w http.ResponseWriter, status int, code, message string) {
	WriteJSON(w, status, ErrorBody{Error: ErrorDetail{Code: code, Message: message}})
}

// WriteList writes a paginated list JSON response with 200 status.
func WriteList(w http.ResponseWriter, items interface{}, total, page, pageSize int) {
	WriteJSON(w, http.StatusOK, ListBody{Items: items, Total: total, Page: page, PageSize: pageSize})
}

// ParsePagination extracts page and page_size from query parameters.
// Defaults: page=1, page_size=20. page_size is capped at 100.
func ParsePagination(r *http.Request) (int, int) {
	page := queryInt(r, "page", 1)
	pageSize := queryInt(r, "page_size", 20)
	if page < 1 {
		page = 1
	}
	if pageSize < 1 {
		pageSize = 20
	}
	if pageSize > 100 {
		pageSize = 100
	}
	return page, pageSize
}

// queryInt reads a query parameter as int, returning defaultVal on missing or invalid input.
func queryInt(r *http.Request, key string, defaultVal int) int {
	v := r.URL.Query().Get(key)
	if v == "" {
		return defaultVal
	}
	n, err := strconv.Atoi(v)
	if err != nil || n <= 0 {
		return defaultVal
	}
	return n
}
