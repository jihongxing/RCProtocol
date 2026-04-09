package response

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestMapUpstreamStatus(t *testing.T) {
	tests := []struct {
		status  int
		wantCode string
		wantMsg  string
	}{
		{400, CodeInvalidInput, "输入非法"},
		{401, CodeAuthRequired, "未认证"},
		{403, CodeForbidden, "无权执行"},
		{404, CodeNotFound, "资源不存在"},
		{409, CodeConflict, "幂等冲突"},
		{422, CodeUnprocessable, "语义不满足"},
		{429, CodeRateLimited, "限流触发"},
		{500, CodeUpstreamFailure, "上游失败"},
		{503, CodeUpstreamFailure, "上游失败"},
	}

	for _, tt := range tests {
		code, msg := MapUpstreamStatus(tt.status)
		if code != tt.wantCode {
			t.Errorf("MapUpstreamStatus(%d) code = %q, want %q", tt.status, code, tt.wantCode)
		}
		if msg != tt.wantMsg {
			t.Errorf("MapUpstreamStatus(%d) message = %q, want %q", tt.status, msg, tt.wantMsg)
		}
	}
}

func TestMapUpstreamStatus_DefaultUnknown(t *testing.T) {
	// Non-5xx status codes not in the switch fall through to "未知错误"
	code, msg := MapUpstreamStatus(418)
	if code != CodeUpstreamFailure {
		t.Errorf("MapUpstreamStatus(418) code = %q, want %q", code, CodeUpstreamFailure)
	}
	if msg != "未知错误" {
		t.Errorf("MapUpstreamStatus(418) message = %q, want %q", msg, "未知错误")
	}
}

func TestWriteError(t *testing.T) {
	w := httptest.NewRecorder()
	traceID := "test-trace-id-123"

	WriteError(w, http.StatusBadRequest, CodeInvalidInput, "bad input", traceID)

	resp := w.Result()
	defer resp.Body.Close()

	// Verify HTTP status code
	if resp.StatusCode != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", resp.StatusCode, http.StatusBadRequest)
	}

	// Verify Content-Type
	ct := resp.Header.Get("Content-Type")
	if ct != "application/json; charset=utf-8" {
		t.Errorf("Content-Type = %q, want %q", ct, "application/json; charset=utf-8")
	}

	// Verify X-Trace-Id header
	if got := resp.Header.Get("X-Trace-Id"); got != traceID {
		t.Errorf("X-Trace-Id = %q, want %q", got, traceID)
	}

	// Verify body is valid JSON with correct structure
	var body ErrorBody
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		t.Fatalf("failed to decode JSON body: %v", err)
	}

	if body.Error.Code != CodeInvalidInput {
		t.Errorf("error.code = %q, want %q", body.Error.Code, CodeInvalidInput)
	}
	if body.Error.Message != "bad input" {
		t.Errorf("error.message = %q, want %q", body.Error.Message, "bad input")
	}
	if body.Error.TraceID != traceID {
		t.Errorf("error.trace_id = %q, want %q", body.Error.TraceID, traceID)
	}
}

func TestWriteError_ServerError(t *testing.T) {
	w := httptest.NewRecorder()
	traceID := "trace-502"

	WriteError(w, http.StatusBadGateway, CodeUpstreamFailure, "上游失败", traceID)

	resp := w.Result()
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusBadGateway {
		t.Errorf("status = %d, want %d", resp.StatusCode, http.StatusBadGateway)
	}

	var body ErrorBody
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		t.Fatalf("failed to decode JSON body: %v", err)
	}

	if body.Error.Code != CodeUpstreamFailure {
		t.Errorf("error.code = %q, want %q", body.Error.Code, CodeUpstreamFailure)
	}
	if body.Error.TraceID != traceID {
		t.Errorf("error.trace_id = %q, want %q", body.Error.TraceID, traceID)
	}
}
