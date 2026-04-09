package proxy

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httputil"
	"net/url"
	"strings"

	"rcprotocol/services/go-gateway/internal/config"
	"rcprotocol/services/go-gateway/internal/middleware"
	"rcprotocol/services/go-gateway/internal/response"
)

// Route maps a Gateway URL prefix to an upstream service.
type Route struct {
	Prefix      string // Gateway URL prefix to match
	StripPrefix string // Prefix to strip before forwarding
	Upstream    string // Upstream base URL; empty means not configured
}

// NewRouter builds an http.Handler that routes requests to upstream services
// based on URL prefix matching. /healthz is handled directly with 200 "ok".
// Unmatched routes receive 404. Unconfigured upstreams receive 503.
func NewRouter(cfg *config.Config) http.Handler {
	routes := []Route{
		{Prefix: "/api/verify", StripPrefix: "/api", Upstream: cfg.RcApiUpstream},
		{Prefix: "/api/protocol/", StripPrefix: "/api/protocol", Upstream: cfg.RcApiUpstream},
		{Prefix: "/api/brands/", StripPrefix: "/api", Upstream: cfg.RcApiUpstream},
		{Prefix: "/api/brands", StripPrefix: "/api", Upstream: cfg.RcApiUpstream},
		{Prefix: "/api/factory/quick-log", StripPrefix: "/api", Upstream: cfg.RcApiUpstream},
		{Prefix: "/api/bff/", StripPrefix: "/api/bff", Upstream: cfg.GoBffUpstream},
		{Prefix: "/api/iam/", StripPrefix: "/api/iam", Upstream: cfg.GoIamUpstream},
		// ARCHIVED: Phase 2 removed approval workflow - route kept for backward compatibility but will return 503
		// {Prefix: "/api/approval/", StripPrefix: "/api/approval", Upstream: cfg.GoApprovalUpstream},
		{Prefix: "/api/workorder/", StripPrefix: "/api/workorder", Upstream: cfg.GoWorkorderUpstream},
	}

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		traceID := r.Header.Get(middleware.TraceIDHeader)

		if r.URL.Path == "/healthz" {
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte("ok"))
			return
		}

		for _, route := range routes {
			if strings.HasPrefix(r.URL.Path, route.Prefix) {
				if route.Upstream == "" {
					response.WriteError(w, http.StatusServiceUnavailable,
						response.CodeUpstreamFailure,
						"upstream service not configured", traceID)
					return
				}
				serveProxy(w, r, route.Upstream, route.StripPrefix, traceID)
				return
			}
		}

		response.WriteError(w, http.StatusNotFound,
			response.CodeNotFound, "no matching route", traceID)
	})
}

// serveProxy forwards the request to the upstream service via httputil.ReverseProxy.
// It strips the configured prefix from the path before forwarding and wraps
// upstream error responses into the unified Gateway error format.
func serveProxy(w http.ResponseWriter, r *http.Request, upstream, stripPrefix, traceID string) {
	target, err := url.Parse(upstream)
	if err != nil {
		response.WriteError(w, http.StatusBadGateway,
			response.CodeUpstreamFailure, "invalid upstream URL", traceID)
		return
	}

	proxy := &httputil.ReverseProxy{
		Director: func(req *http.Request) {
			req.URL.Scheme = target.Scheme
			req.URL.Host = target.Host
			req.Host = target.Host

			// Path strip: remove the Gateway prefix so upstream sees its native route.
			// e.g. /api/verify?uid=04A3 → /verify?uid=04A3
			//      /api/protocol/assets/xxx/activate → /assets/xxx/activate
			if stripPrefix != "" {
				req.URL.Path = strings.TrimPrefix(req.URL.Path, stripPrefix)
				if req.URL.Path == "" {
					req.URL.Path = "/"
				}
				if req.URL.RawPath != "" {
					req.URL.RawPath = strings.TrimPrefix(req.URL.RawPath, stripPrefix)
				}
			}
			// Query parameters (URL.RawQuery) are preserved automatically.
		},
		ModifyResponse: func(resp *http.Response) error {
			if resp.StatusCode >= 400 {
				return wrapUpstreamError(resp, traceID)
			}
			return nil
		},
		ErrorHandler: func(w http.ResponseWriter, r *http.Request, err error) {
			response.WriteError(w, http.StatusBadGateway,
				response.CodeUpstreamFailure, "upstream unreachable", traceID)
		},
	}

	proxy.ServeHTTP(w, r)
}

// wrapUpstreamError reads the upstream response body and rewrites it into the
// unified Gateway error format. If the upstream body already conforms to
// {"error": {"code": ..., "message": ...}}, the message is preserved but
// trace_id is injected. Upstream 5xx responses are remapped to Gateway 502.
func wrapUpstreamError(resp *http.Response, traceID string) error {
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil
	}
	_ = resp.Body.Close()

	// Upstream 5xx → Gateway returns 502 to the client.
	gatewayStatus := resp.StatusCode
	if resp.StatusCode >= 500 {
		gatewayStatus = http.StatusBadGateway
	}

	// Check if upstream already returns the standard error envelope.
	var upstreamErr response.ErrorBody
	if err := json.Unmarshal(body, &upstreamErr); err == nil && upstreamErr.Error.Code != "" {
		upstreamErr.Error.TraceID = traceID
		if resp.StatusCode >= 500 {
			upstreamErr.Error.Code = response.CodeUpstreamFailure
		}
		newBody, _ := json.Marshal(upstreamErr)
		resp.Body = io.NopCloser(bytes.NewReader(newBody))
		resp.ContentLength = int64(len(newBody))
		resp.StatusCode = gatewayStatus
		resp.Header.Set("Content-Type", "application/json; charset=utf-8")
		resp.Header.Set("X-Trace-Id", traceID)
		return nil
	}

	// Upstream body is not in standard format; generate from status code.
	code, message := response.MapUpstreamStatus(resp.StatusCode)
	errBody := response.ErrorBody{
		Error: response.ErrorDetail{
			Code:    code,
			Message: message,
			TraceID: traceID,
		},
	}
	newBody, _ := json.Marshal(errBody)
	resp.Body = io.NopCloser(bytes.NewReader(newBody))
	resp.ContentLength = int64(len(newBody))
	resp.StatusCode = gatewayStatus
	resp.Header.Set("Content-Type", "application/json; charset=utf-8")
	resp.Header.Set("X-Trace-Id", traceID)
	return nil
}
