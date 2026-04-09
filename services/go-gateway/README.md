# go-gateway — 统一接入网关

RCProtocol 的统一接入网关，所有外部请求的第一入口。

## 认证方式

Gateway 支持两种互斥的认证方式：

```
请求到达
  ↓
白名单路由？ ──是──→ 直接放行
  ↓ 否
X-Api-Key 存在？ ──是──→ API Key 认证路径
  ↓ 否                    ↓
JWT 认证路径          路由支持？
                        ├─ 是 → 格式校验（brand_ 前缀）→ 透传上游
                        └─ 否 → 401
```

### JWT 认证

- Header: `Authorization: Bearer <token>`
- 算法: HS256
- 白名单路由（跳过校验）: `/healthz`, `/api/verify`, `/api/iam/auth/`

### API Key 认证

- Header: `X-Api-Key: brand_xxxxxxxx`
- 格式要求: 必须以 `brand_` 开头
- Gateway 仅做格式校验，有效性验证由上游 rc-api 负责

支持 API Key 的路由：

| 路由前缀 | 说明 |
|---------|------|
| `/api/brands/` | 品牌资产查询、批量激活、过户记录 |
| `/api/factory/quick-log` | 工厂盲扫 |
| `/api/verify` | 验真接口（同时支持 JWT 和 API Key） |

## 路由表

| Gateway URL | 上游服务 | Strip 前缀 |
|-------------|---------|-----------|
| `/api/verify` | rc-api | `/api` |
| `/api/protocol/` | rc-api | `/api/protocol` |
| `/api/brands/` | rc-api | `/api` |
| `/api/factory/quick-log` | rc-api | `/api` |
| `/api/bff/` | go-bff | `/api/bff` |
| `/api/iam/` | go-iam | `/api/iam` |
| `/api/approval/` | go-approval | `/api/approval` |
| `/api/workorder/` | go-workorder | `/api/workorder` |
| `/healthz` | 自身 | — |

## 中间件链

从外到内执行顺序：Logging → Trace → RateLimit → Auth → WriteHeaders → Router

## 环境变量

| 变量 | 默认值 | 必填 |
|------|--------|------|
| `RC_JWT_SECRET` | — | 是 |
| `GATEWAY_PORT` | `:8080` | 否 |
| `RC_API_UPSTREAM` | `http://rc-api:8081` | 否 |
| `GO_BFF_UPSTREAM` | `http://go-bff:8082` | 否 |
| `GO_IAM_UPSTREAM` | — | 否 |
| `GO_APPROVAL_UPSTREAM` | — | 否 |
| `GO_WORKORDER_UPSTREAM` | — | 否 |
| `RATE_LIMIT_RPS` | `100` | 否 |
| `RATE_LIMIT_BURST` | `200` | 否 |

## 测试

```bash
cd services/go-gateway && go test ./... -v
```
