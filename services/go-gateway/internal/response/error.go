package response

import (
	"encoding/json"
	"net/http"
)

// ErrorBody is the top-level envelope for all Gateway error responses.
type ErrorBody struct {
	Error ErrorDetail `json:"error"`
}

// ErrorDetail carries the machine-readable code, human message, and trace ID.
type ErrorDetail struct {
	Code    string `json:"code"`
	Message string `json:"message"`
	TraceID string `json:"trace_id"`
}

const (
	CodeInvalidInput    = "INVALID_INPUT"
	CodeAuthRequired    = "AUTH_REQUIRED"
	CodeForbidden       = "FORBIDDEN"
	CodeNotFound        = "NOT_FOUND"
	CodeConflict        = "CONFLICT"
	CodeUnprocessable   = "UNPROCESSABLE"
	CodeRateLimited     = "RATE_LIMITED"
	CodeUpstreamFailure = "UPSTREAM_FAILURE"
)

// WriteError writes a unified JSON error response.
func WriteError(w http.ResponseWriter, statusCode int, code, message, traceID string) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.Header().Set("X-Trace-Id", traceID)
	w.WriteHeader(statusCode)
	_ = json.NewEncoder(w).Encode(ErrorBody{
		Error: ErrorDetail{
			Code:    code,
			Message: message,
			TraceID: traceID,
		},
	})
}

// MapUpstreamStatus maps an upstream HTTP status code to a Gateway error code
// and a default Chinese message. Pure function, no side effects.
func MapUpstreamStatus(statusCode int) (string, string) {
	switch statusCode {
	case 400:
		return CodeInvalidInput, "输入非法"
	case 401:
		return CodeAuthRequired, "未认证"
	case 403:
		return CodeForbidden, "无权执行"
	case 404:
		return CodeNotFound, "资源不存在"
	case 409:
		return CodeConflict, "幂等冲突"
	case 422:
		return CodeUnprocessable, "语义不满足"
	case 429:
		return CodeRateLimited, "限流触发"
	default:
		if statusCode >= 500 {
			return CodeUpstreamFailure, "上游失败"
		}
		return CodeUpstreamFailure, "未知错误"
	}
}
