package handler

import (
	"encoding/json"
	"net/http"
)

// ErrorBody 统一错误响应体
type ErrorBody struct {
	Error ErrorDetail `json:"error"`
}

// ErrorDetail 错误详情
type ErrorDetail struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

// ListBody 分页列表响应体
type ListBody struct {
	Items    interface{} `json:"items"`
	Total    int         `json:"total"`
	Page     int         `json:"page"`
	PageSize int         `json:"page_size"`
}

// WriteJSON 写入 JSON 响应
func WriteJSON(w http.ResponseWriter, status int, v interface{}) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

// WriteError 写入统一格式的错误响应
func WriteError(w http.ResponseWriter, status int, code, message string) {
	WriteJSON(w, status, ErrorBody{Error: ErrorDetail{Code: code, Message: message}})
}

// WriteList 写入分页列表响应
func WriteList(w http.ResponseWriter, items interface{}, total, page, pageSize int) {
	WriteJSON(w, http.StatusOK, ListBody{Items: items, Total: total, Page: page, PageSize: pageSize})
}

// ParsePagination 从查询参数解析分页信息，返回 (page, pageSize)
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

func queryInt(r *http.Request, key string, defaultVal int) int {
	v := r.URL.Query().Get(key)
	if v == "" {
		return defaultVal
	}
	n := 0
	for _, c := range v {
		if c < '0' || c > '9' {
			return defaultVal
		}
		n = n*10 + int(c-'0')
	}
	if n == 0 {
		return defaultVal
	}
	return n
}
