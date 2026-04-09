package handler

import (
	"encoding/json"
	"errors"
	"net/http"
	"strconv"
	"strings"

	"github.com/jackc/pgx/v5/pgconn"
)

// ErrorBody aligns with Spec-07 Gateway error format.
type ErrorBody struct {
	Error ErrorDetail `json:"error"`
}

// ErrorDetail carries machine-readable code and human-readable message.
type ErrorDetail struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

// SuccessBody wraps a single resource response.
type SuccessBody struct {
	Data interface{} `json:"data"`
}

// ListBody wraps a paginated list response.
type ListBody struct {
	Data  interface{} `json:"data"`
	Page  int         `json:"page"`
	Size  int         `json:"page_size"`
	Total int         `json:"total"`
}

// WriteJSON sets Content-Type, writes HTTP status, and encodes v as JSON.
func WriteJSON(w http.ResponseWriter, status int, v interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

// WriteSuccess wraps data in SuccessBody and writes with given status.
func WriteSuccess(w http.ResponseWriter, status int, data interface{}) {
	WriteJSON(w, status, SuccessBody{Data: data})
}

// WriteList wraps data in ListBody and writes with status 200.
func WriteList(w http.ResponseWriter, data interface{}, page, size, total int) {
	WriteJSON(w, http.StatusOK, ListBody{
		Data:  data,
		Page:  page,
		Size:  size,
		Total: total,
	})
}

// WriteError wraps code and message in ErrorBody and writes with given status.
func WriteError(w http.ResponseWriter, status int, code, message string) {
	WriteJSON(w, status, ErrorBody{
		Error: ErrorDetail{Code: code, Message: message},
	})
}

// isUniqueViolation checks whether err is a PostgreSQL unique constraint violation (23505).
func isUniqueViolation(err error) bool {
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) {
		return pgErr.Code == "23505"
	}
	return false
}

// isValidEmail returns true if email contains @ and has . after the @.
func isValidEmail(email string) bool {
	atIdx := strings.Index(email, "@")
	if atIdx < 1 {
		return false
	}
	domain := email[atIdx+1:]
	return strings.Contains(domain, ".")
}

// parsePagination reads page and page_size from query params with defaults 1 and 20.
func parsePagination(r *http.Request) (page, pageSize int) {
	page = 1
	pageSize = 20

	if v := r.URL.Query().Get("page"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			page = n
		}
	}
	if v := r.URL.Query().Get("page_size"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			pageSize = n
		}
	}
	return page, pageSize
}
