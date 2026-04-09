# go-bff

前端聚合接口服务（Backend For Frontend），面向 C 端 uni-app 和 B 端控制台提供 ViewModel JSON。

## 架构

```
Client → Gateway (/api/bff/*) → strip "/api/bff" → go-bff (:8082)
                                                      │
                                                      ▼
                                              Logging → JWT Claims → Handler → UpstreamClient
                                                                                    │
                                                                              rc-api / go-iam
```

- 路由不带 `/api/bff` 前缀（Gateway strip 后转发）
- JWT 只做 base64url 解码，不验签（由 Gateway 负责）
- 无状态、无数据库，所有数据通过 HTTP API 获取

## 环境变量

| 变量 | 默认值 | 说明 |
|------|--------|------|
| `BFF_PORT` | `:8082` | 监听端口 |
| `RC_API_BASE_URL` | `http://rc-api:8081` | rc-api 上游地址 |
| `GO_IAM_BASE_URL` | `http://go-iam:8083` | go-iam 上游地址 |

## 接口列表

| 方法 | 路径 | 说明 | 认证 |
|------|------|------|------|
| GET | `/healthz` | 健康检查 | 否 |
| GET | `/app/assets` | C 端资产列表 | 是 |
| GET | `/app/assets/{assetId}` | C 端资产详情 | 是 |
| GET | `/console/dashboard` | B 端 Dashboard 统计 | 是 |
| GET | `/console/brands/{brandId}` | B 端品牌详情聚合 | 是 |
| GET | `/console/brands/{brandId}/products` | B 端品牌产品列表 | 是 |
| GET | `/console/factory/tasks` | 工厂任务列表（占位） | 是 |

### 品牌详情接口 `GET /console/brands/{brandId}`

聚合 go-iam 组织信息和 API Key 列表，返回品牌极简化信息。

- Brand 角色只能访问自己的品牌，否则 403
- Platform 角色可访问任意品牌
- 组织信息查询失败 → 502
- API Key 列表查询失败 → 优雅降级为空数组

响应示例：
```json
{
  "brand_id": "...",
  "brand_name": "...",
  "contact_person": "...",
  "contact_info": "...",
  "created_at": "...",
  "api_keys": [
    {"key_id": "...", "description": "...", "created_at": "...", "last_used_at": "...", "status": "..."}
  ]
}
```

### 外部 SKU 映射字段

资产列表和详情接口返回的 ViewModel 包含外部 SKU 映射字段：

| 字段 | 类型 | 说明 |
|------|------|------|
| `external_product_id` | `*string` | 品牌方 SKU ID |
| `external_product_name` | `*string` | 品牌方产品名称 |
| `external_product_url` | `*string` | 品牌方产品详情页 URL |

字段为空时返回 `null`（指针类型），直接透传 rc-api 返回值。

## 开发

```bash
cd services/go-bff
go test ./... -v
go build ./cmd/bff/
go vet ./...
```
