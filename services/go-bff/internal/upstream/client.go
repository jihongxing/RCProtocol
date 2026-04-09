package upstream

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"strings"
	"time"
)

// UpstreamClient wraps net/http to call rc-api and go-iam backend services.
type UpstreamClient struct {
	rcApiBaseURL string
	iamBaseURL   string
	httpClient   *http.Client
}

type GatewayAuthHeaders struct {
	Authorization  string
	TraceID        string
	ApiKeyHash     string
	ApiKeyVerified string
}

func GatewayAuthHeadersFromRequest(r *http.Request) GatewayAuthHeaders {
	return GatewayAuthHeaders{
		Authorization:  r.Header.Get("Authorization"),
		TraceID:        r.Header.Get("X-Trace-Id"),
		ApiKeyHash:     r.Header.Get("X-Api-Key-Hash"),
		ApiKeyVerified: r.Header.Get("X-Api-Key-Verified"),
	}
}

// New creates an UpstreamClient with 10-second timeout.
func New(rcApiBaseURL, iamBaseURL string) *UpstreamClient {
	return &UpstreamClient{
		rcApiBaseURL: rcApiBaseURL,
		iamBaseURL:   iamBaseURL,
		httpClient:   &http.Client{Timeout: 10 * time.Second},
	}
}

// UpstreamError represents an error from a backend service call.
type UpstreamError struct {
	StatusCode int
	Code       string
	Message    string
}

func (e *UpstreamError) Error() string {
	return fmt.Sprintf("upstream %d: %s", e.StatusCode, e.Message)
}

// Get calls a backend GET API, forwarding Authorization and X-Trace-Id headers.
func (c *UpstreamClient) Get(ctx context.Context, fullURL, authHeader, traceID string) ([]byte, error) {
	return c.doInternal(ctx, http.MethodGet, fullURL, nil, "", authHeader, traceID, "", "")
}

// GetWithGatewayAuth forwards Gateway API key contract headers in addition to standard auth headers.
func (c *UpstreamClient) GetWithGatewayAuth(ctx context.Context, fullURL, authHeader, traceID, apiKeyHash, apiKeyVerified string) ([]byte, error) {
	return c.doInternal(ctx, http.MethodGet, fullURL, nil, "", authHeader, traceID, apiKeyHash, apiKeyVerified)
}

// DoWithGatewayAuth is the standard gateway-auth-aware upstream request helper for any HTTP method.
func (c *UpstreamClient) DoWithGatewayAuth(ctx context.Context, method, fullURL string, body []byte, contentType string, headers GatewayAuthHeaders) ([]byte, error) {
	return c.doInternal(ctx, method, fullURL, body, contentType, headers.Authorization, headers.TraceID, headers.ApiKeyHash, headers.ApiKeyVerified)
}

// RcApiDoWithGatewayAuth is the standard gateway-auth-aware rc-api helper for any HTTP method.
func (c *UpstreamClient) RcApiDoWithGatewayAuth(ctx context.Context, method, path string, body []byte, contentType string, headers GatewayAuthHeaders) ([]byte, error) {
	return c.DoWithGatewayAuth(ctx, method, c.RcApiURL(path), body, contentType, headers)
}

// RcApiGetWithGatewayAuth is the standard gateway-auth-aware rc-api read helper.
func (c *UpstreamClient) RcApiGetWithGatewayAuth(ctx context.Context, path string, headers GatewayAuthHeaders) ([]byte, error) {
	return c.RcApiDoWithGatewayAuth(ctx, http.MethodGet, path, nil, "", headers)
}

func (c *UpstreamClient) doInternal(ctx context.Context, method, fullURL string, body []byte, contentType, authHeader, traceID, apiKeyHash, apiKeyVerified string) ([]byte, error) {
	var reader io.Reader
	if body != nil {
		reader = bytes.NewReader(body)
	}
	req, err := http.NewRequestWithContext(ctx, method, fullURL, reader)
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}
	if contentType != "" {
		req.Header.Set("Content-Type", contentType)
	}
	if authHeader != "" {
		req.Header.Set("Authorization", authHeader)
	}
	if traceID != "" {
		req.Header.Set("X-Trace-Id", traceID)
	}
	if apiKeyHash != "" {
		req.Header.Set("X-Api-Key-Hash", apiKeyHash)
	}
	if apiKeyVerified != "" {
		req.Header.Set("X-Api-Key-Verified", apiKeyVerified)
	}

	start := time.Now()
	resp, err := c.httpClient.Do(req)
	latency := time.Since(start)
	logPath := sanitizeURLForLog(fullURL)

	if err != nil {
		slog.Warn("upstream error",
			slog.String("upstream_path", logPath),
			slog.String("error", err.Error()),
			slog.Int64("upstream_latency_ms", latency.Milliseconds()),
		)
		if isTimeoutError(err) {
			return nil, &UpstreamError{StatusCode: 504, Code: "GATEWAY_TIMEOUT", Message: "backend request timed out"}
		}
		return nil, &UpstreamError{StatusCode: 502, Code: "UPSTREAM_FAILURE", Message: "backend service unavailable"}
	}
	defer resp.Body.Close()

	slog.Info("upstream call",
		slog.String("upstream_path", logPath),
		slog.Int("upstream_status", resp.StatusCode),
		slog.Int64("upstream_latency_ms", latency.Milliseconds()),
	)

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, &UpstreamError{StatusCode: 502, Code: "UPSTREAM_FAILURE", Message: "read upstream response failed"}
	}
	if resp.StatusCode >= 400 && resp.StatusCode < 500 {
		code, msg := mapUpstream4xx(resp.StatusCode, respBody)
		return nil, &UpstreamError{StatusCode: resp.StatusCode, Code: code, Message: msg}
	}
	if resp.StatusCode >= 500 {
		return nil, &UpstreamError{StatusCode: 502, Code: "UPSTREAM_FAILURE", Message: "backend service unavailable"}
	}
	return respBody, nil
}

// RcApiURL builds a full URL to the rc-api service.
func (c *UpstreamClient) RcApiURL(path string) string { return c.rcApiBaseURL + path }

// IamURL builds a full URL to the go-iam service.
func (c *UpstreamClient) IamURL(path string) string { return c.iamBaseURL + path }

func (c *UpstreamClient) GetBrandName(ctx context.Context, brandID, authHeader, traceID string) string {
	data, err := c.Get(ctx, c.RcApiURL("/brands/"+brandID), authHeader, traceID)
	if err != nil {
		return brandID
	}
	var result struct{ BrandName string `json:"brand_name"` }
	if json.Unmarshal(data, &result) != nil || result.BrandName == "" {
		return brandID
	}
	return result.BrandName
}

func (c *UpstreamClient) GetBrandNameWithGatewayAuth(ctx context.Context, brandID string, headers GatewayAuthHeaders) string {
	data, err := c.RcApiGetWithGatewayAuth(ctx, "/brands/"+brandID, headers)
	if err != nil {
		return brandID
	}
	var result struct{ BrandName string `json:"brand_name"` }
	if json.Unmarshal(data, &result) != nil || result.BrandName == "" {
		return brandID
	}
	return result.BrandName
}

func (c *UpstreamClient) GetProductName(ctx context.Context, brandID, productID, authHeader, traceID string) string {
	u := c.RcApiURL(fmt.Sprintf("/brands/%s/products/%s", brandID, productID))
	data, err := c.Get(ctx, u, authHeader, traceID)
	if err != nil {
		return productID
	}
	var result struct{ ProductName string `json:"product_name"` }
	if json.Unmarshal(data, &result) != nil || result.ProductName == "" {
		return productID
	}
	return result.ProductName
}

func (c *UpstreamClient) GetBrandNamesBatch(ctx context.Context, brandIDs []string, authHeader, traceID string) map[string]string {
	result := make(map[string]string, len(brandIDs))
	if len(brandIDs) == 0 {
		return result
	}
	ids := strings.Join(brandIDs, ",")
	data, err := c.Get(ctx, c.RcApiURL("/brands/batch?ids="+ids), authHeader, traceID)
	if err != nil {
		return result
	}
	var items []struct {
		BrandID   string `json:"brand_id"`
		BrandName string `json:"brand_name"`
	}
	if json.Unmarshal(data, &items) != nil {
		return result
	}
	for _, item := range items {
		if item.BrandName != "" {
			result[item.BrandID] = item.BrandName
		}
	}
	return result
}

func (c *UpstreamClient) GetBrandNamesBatchWithGatewayAuth(ctx context.Context, brandIDs []string, headers GatewayAuthHeaders) map[string]string {
	result := make(map[string]string, len(brandIDs))
	if len(brandIDs) == 0 {
		return result
	}
	ids := strings.Join(brandIDs, ",")
	data, err := c.RcApiGetWithGatewayAuth(ctx, "/brands/batch?ids="+ids, headers)
	if err != nil {
		return result
	}
	var items []struct {
		BrandID   string `json:"brand_id"`
		BrandName string `json:"brand_name"`
	}
	if json.Unmarshal(data, &items) != nil {
		return result
	}
	for _, item := range items {
		if item.BrandName != "" {
			result[item.BrandID] = item.BrandName
		}
	}
	return result
}

// GetProductNamesBatch 保留兼容旧 product 批量查询。
func (c *UpstreamClient) GetProductNamesBatch(ctx context.Context, productIDs []string, authHeader, traceID string) map[string]string {
	result := make(map[string]string, len(productIDs))
	if len(productIDs) == 0 {
		return result
	}
	ids := strings.Join(productIDs, ",")
	data, err := c.Get(ctx, c.RcApiURL("/products/batch?ids="+ids), authHeader, traceID)
	if err != nil {
		return result
	}
	var items []struct {
		ProductID   string `json:"product_id"`
		ProductName string `json:"product_name"`
	}
	if json.Unmarshal(data, &items) != nil {
		return result
	}
	for _, item := range items {
		if item.ProductName != "" {
			result[item.ProductID] = item.ProductName
		}
	}
	return result
}

func isTimeoutError(err error) bool {
	if ue, ok := err.(*url.Error); ok {
		return ue.Timeout()
	}
	return false
}

func sanitizeURLForLog(rawURL string) string {
	u, err := url.Parse(rawURL)
	if err != nil {
		return rawURL
	}
	return u.Path
}

func mapUpstream4xx(statusCode int, body []byte) (string, string) {
	var upstream struct {
		Error struct {
			Code    string `json:"code"`
			Message string `json:"message"`
		} `json:"error"`
	}
	if json.Unmarshal(body, &upstream) == nil && upstream.Error.Code != "" {
		return upstream.Error.Code, upstream.Error.Message
	}
	switch statusCode {
	case 400:
		return "INVALID_INPUT", "invalid request"
	case 401:
		return "AUTH_REQUIRED", "unauthorized"
	case 403:
		return "FORBIDDEN", "access denied"
	case 404:
		return "NOT_FOUND", "resource not found"
	case 409:
		return "CONFLICT", "resource conflict"
	default:
		return "UPSTREAM_FAILURE", "upstream error"
	}
}
