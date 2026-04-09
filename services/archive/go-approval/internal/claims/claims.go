package claims

import "net/http"

// Claims 从 Gateway 转发的请求头解析的用户身份
type Claims struct {
	Sub   string // X-Claims-Sub
	Role  string // X-Claims-Role
	OrgID string // X-Claims-Org-Id
}

// FromRequest 从请求头提取 Gateway 转发的 Claims
func FromRequest(r *http.Request) *Claims {
	return &Claims{
		Sub:   r.Header.Get("X-Claims-Sub"),
		Role:  r.Header.Get("X-Claims-Role"),
		OrgID: r.Header.Get("X-Claims-Org-Id"),
	}
}

// Valid 检查 Claims 是否完整且角色合法
func (c *Claims) Valid() bool {
	if c.Sub == "" || c.Role == "" || c.OrgID == "" {
		return false
	}
	switch c.Role {
	case "Platform", "Brand", "Factory", "Consumer", "Moderator":
		return true
	default:
		return false
	}
}
