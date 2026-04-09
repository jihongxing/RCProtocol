package downstream

import (
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"time"

	"rcprotocol/services/go-approval/internal/model"
)

// Client 下游 HTTP 客户端，审批通过后调用 rc-api 执行实际业务动作
type Client struct {
	rcApiBaseURL string
	httpClient   *http.Client
}

// New 创建下游客户端，超时 15 秒
func New(rcApiBaseURL string) *Client {
	return &Client{
		rcApiBaseURL: rcApiBaseURL,
		httpClient: &http.Client{
			Timeout: 15 * time.Second,
		},
	}
}

// ExecuteResult 下游执行结果
type ExecuteResult struct {
	Success bool
	Body    json.RawMessage
}

// Execute 根据审批类型执行下游动作
func (c *Client) Execute(ctx context.Context, approval *model.Approval, authHeader, traceID string) *ExecuteResult {
	switch approval.Type {
	case model.TypeBrandPublish:
		url := c.brandPublishURL(approval.Payload)
		if url == "" {
			return &ExecuteResult{Success: false, Body: errorJSON("invalid brand_id in payload", 0)}
		}
		return c.callRcApi(ctx, approval, "POST", url, authHeader, traceID)
	case model.TypeRiskRecovery:
		url := c.riskRecoveryURL(approval.Payload)
		if url == "" {
			return &ExecuteResult{Success: false, Body: errorJSON("invalid asset_id in payload", 0)}
		}
		return c.callRcApi(ctx, approval, "POST", url, authHeader, traceID)
	case model.TypePolicyApply, model.TypeHighRiskAction:
		// Phase 2 下游未就位，直接标记成功
		return &ExecuteResult{Success: true, Body: json.RawMessage(`{"status":"executed_without_downstream"}`)}
	default:
		return &ExecuteResult{Success: false, Body: errorJSON("unknown approval type", 0)}
	}
}

func (c *Client) callRcApi(ctx context.Context, approval *model.Approval, method, url, authHeader, traceID string) *ExecuteResult {
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
	req.Header.Set("X-Approval-Id", approval.ID)

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

func (c *Client) brandPublishURL(payload json.RawMessage) string {
	var p struct {
		BrandID string `json:"brand_id"`
	}
	_ = json.Unmarshal(payload, &p)
	if !isValidID(p.BrandID) {
		return ""
	}
	return c.rcApiBaseURL + "/brands/" + p.BrandID + "/publish"
}

func (c *Client) riskRecoveryURL(payload json.RawMessage) string {
	var p struct {
		AssetID string `json:"asset_id"`
	}
	_ = json.Unmarshal(payload, &p)
	if !isValidID(p.AssetID) {
		return ""
	}
	return c.rcApiBaseURL + "/assets/" + p.AssetID + "/recover"
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

func errorJSON(msg string, statusCode int) json.RawMessage {
	b, _ := json.Marshal(map[string]interface{}{
		"error":           msg,
		"upstream_status": statusCode,
	})
	return b
}

// IsValidID 导出版本，供测试使用
func IsValidID(id string) bool {
	return isValidID(id)
}

// SetHTTPClient 允许测试替换 httpClient
func (c *Client) SetHTTPClient(hc *http.Client) {
	c.httpClient = hc
}
