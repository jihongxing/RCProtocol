package downstream

import (
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"time"
)

// ExecuteResult 下游执行结果
type ExecuteResult struct {
	Success bool
	Body    json.RawMessage
}

// RcApiClient rc-api 下游 HTTP 客户端，用于冻结/恢复操作
type RcApiClient struct {
	baseURL    string
	httpClient *http.Client
}

// NewRcApiClient 创建 rc-api 客户端，超时 15 秒
func NewRcApiClient(baseURL string) *RcApiClient {
	return &RcApiClient{
		baseURL: baseURL,
		httpClient: &http.Client{
			Timeout: 15 * time.Second,
		},
	}
}

// Freeze 调用 rc-api 冻结资产
func (c *RcApiClient) Freeze(ctx context.Context, assetID, authHeader, traceID string) *ExecuteResult {
	if !isValidID(assetID) {
		return &ExecuteResult{Success: false, Body: errorJSON("invalid asset_id", 0)}
	}
	return c.call(ctx, "POST", c.baseURL+"/assets/"+assetID+"/freeze", authHeader, traceID, "")
}

// MarkTampered 调用 rc-api 标记资产为 tampered（物理篡改）
func (c *RcApiClient) MarkTampered(ctx context.Context, assetID, authHeader, traceID string) *ExecuteResult {
	if !isValidID(assetID) {
		return &ExecuteResult{Success: false, Body: errorJSON("invalid asset_id", 0)}
	}
	return c.call(ctx, "POST", c.baseURL+"/assets/"+assetID+"/mark-tampered", authHeader, traceID, "")
}

// MarkCompromised 调用 rc-api 标记资产为 compromised（安全受损）
func (c *RcApiClient) MarkCompromised(ctx context.Context, assetID, authHeader, traceID string) *ExecuteResult {
	if !isValidID(assetID) {
		return &ExecuteResult{Success: false, Body: errorJSON("invalid asset_id", 0)}
	}
	return c.call(ctx, "POST", c.baseURL+"/assets/"+assetID+"/mark-compromised", authHeader, traceID, "")
}

// Recover 调用 rc-api 恢复资产
func (c *RcApiClient) Recover(ctx context.Context, assetID, authHeader, traceID, approvalID string) *ExecuteResult {
	if !isValidID(assetID) {
		return &ExecuteResult{Success: false, Body: errorJSON("invalid asset_id", 0)}
	}
	return c.call(ctx, "POST", c.baseURL+"/assets/"+assetID+"/recover", authHeader, traceID, approvalID)
}

func (c *RcApiClient) call(ctx context.Context, method, url, authHeader, traceID, approvalID string) *ExecuteResult {
	req, err := http.NewRequestWithContext(ctx, method, url, nil)
	if err != nil {
		return &ExecuteResult{Success: false, Body: errorJSON("create request failed", 0)}
	}

	if authHeader != "" {
		req.Header.Set("Authorization", authHeader)
	}
	if traceID != "" {
		req.Header.Set("X-Trace-Id", traceID)
	}
	if approvalID != "" {
		req.Header.Set("X-Approval-Id", approvalID)
	}

	start := time.Now()
	resp, err := c.httpClient.Do(req)
	latency := time.Since(start)

	if err != nil {
		slog.Error("downstream call failed",
			slog.String("upstream_url", url),
			slog.String("error", err.Error()),
			slog.Int64("upstream_latency_ms", latency.Milliseconds()),
		)
		return &ExecuteResult{Success: false, Body: errorJSON("backend service unavailable", 0)}
	}
	defer resp.Body.Close()

	slog.Info("downstream call",
		slog.String("upstream_url", url),
		slog.Int("upstream_status", resp.StatusCode),
		slog.Int64("upstream_latency_ms", latency.Milliseconds()),
	)

	body, _ := io.ReadAll(resp.Body)
	if resp.StatusCode >= 200 && resp.StatusCode < 300 {
		return &ExecuteResult{Success: true, Body: body}
	}
	return &ExecuteResult{Success: false, Body: errorJSON(string(body), resp.StatusCode)}
}

// isValidID 校验 ID 为合法的字母数字+连字符+下划线格式，防止 URL path 注入
func isValidID(id string) bool {
	if id == "" {
		return false
	}
	for _, c := range id {
		if !((c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') || (c >= '0' && c <= '9') || c == '-' || c == '_') {
			return false
		}
	}
	return true
}

// IsValidID 导出版本，供测试使用
func IsValidID(id string) bool {
	return isValidID(id)
}

func errorJSON(msg string, statusCode int) json.RawMessage {
	b, _ := json.Marshal(map[string]interface{}{
		"error":           msg,
		"upstream_status": statusCode,
	})
	return b
}
