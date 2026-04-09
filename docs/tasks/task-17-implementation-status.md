# Task #17: 品牌极简注册 API - 实现状态

## 完成情况

### ✅ 已完成的部分

#### 0. 与基础规范对齐情况
- ✅ `spec-06-id-unification` 已落地完成
- ✅ 品牌相关主资源 ID 已统一采用 `prefix + ULID` 策略
- ✅ `brand_id` 已统一收敛到公共生成入口
- ✅ 任务文档 / Spec / 代码实现三者已同步

#### 1. 数据库层 (db/brands.rs)
- ✅ `BrandRecord` 结构体（包含 contact_email, industry 字段）
- ✅ `ApiKeyRecord` 结构体
- ✅ `create_brand()` - 创建品牌
- ✅ `fetch_brand_by_id()` - 根据 ID 查询品牌
- ✅ `fetch_brand_by_email()` - 根据邮箱查询品牌（唯一性检查）
- ✅ `fetch_brand_detail()` - 查询品牌详情（返回 BrandDetailResponse）
- ✅ `fetch_brand_by_api_key_hash()` - 根据 API Key 哈希查询品牌（用于认证）
- ✅ `create_api_key()` - 创建 API Key
- ✅ `revoke_api_key()` - 撤销 API Key
- ✅ `fetch_active_api_key()` - 查询活跃的 API Key（事务版本）
- ✅ `fetch_active_api_key_by_brand()` - 查询活跃的 API Key（连接池版本）
- ✅ `fetch_api_keys_by_brand()` - 查询品牌的所有 API Key
- ✅ `list_api_keys()` - 列出 API Key（别名）
- ✅ `list_brands()` - 列出品牌（带分页）
- ✅ `update_brand()` - 更新品牌信息
- ✅ `fetch_brands_batch()` - 批量查询品牌
- ✅ `update_api_key_last_used()` - 更新 API Key 最后使用时间

#### 2. 认证模块 (auth/api_key.rs)
- ✅ `generate_api_key()` - 生成 API Key（格式：rcpk_live_<32 hex>）
- ✅ `hash_api_key()` - SHA-256 哈希
- ✅ `extract_key_prefix()` - 提取密钥前缀（用于显示）
- ✅ `generate_key_id()` - 生成 ULID 格式的 key_id
- ✅ `generate_brand_id()` - 已重定向到公共 ID 生成入口
- ✅ 完整的单元测试

#### 3. 路由层 (routes/brand.rs)
- ✅ `RegisterBrandRequest` - 注册请求（brand_name, contact_email, industry）
- ✅ `RotateApiKeyRequest` - 轮换请求（可选 reason）
- ✅ `UpdateBrandRequest` - 更新请求
- ✅ `BrandDetailResponse` - 品牌详情响应
- ✅ `RegisterBrandResponse` - 注册响应（包含 API Key）
- ✅ `RotateApiKeyResponse` - 轮换响应（包含新 API Key）
- ✅ `ApiKeyInfo` - API Key 信息结构
- ✅ `ApiKeyListItem` - API Key 列表项
- ✅ `ApiKeyListResponse` - API Key 列表响应
- ✅ `register_brand()` - 品牌注册处理函数
- ✅ `rotate_api_key()` - API Key 轮换处理函数
- ✅ `list_api_keys()` - API Key 列表处理函数
- ✅ `get_brand()` - 品牌详情处理函数
- ✅ `list_brands()` - 品牌列表处理函数
- ✅ `update_brand()` - 品牌更新处理函数
- ✅ `batch_brands()` - 批量查询处理函数
- ✅ 验证函数：`validate_name()`, `validate_email()`, `validate_industry()`
- ✅ 权限检查函数：`check_role_allowed()`, `check_brand_read_access()`, `check_brand_write_access()`
- ✅ 路由注册（/brands, /brands/:id, /brands/:id/rotate-api-key, /brands/:id/api-keys）
- ✅ 完整的单元测试

#### 4. 中间件更新 (auth/middleware.rs)
- ✅ 修复 `authenticate_api_key()` 使用新的 `auth::api_key::hash_api_key()`
- ✅ 更新测试代码以匹配新的数据库结构（brands 和 api_keys 表分离）
- ✅ fallback 测试已切换到规范化 `brand_id`

#### 5. 依赖管理
- ✅ 添加 `rand` 到 workspace 依赖
- ✅ 添加 `ulid` 到 workspace 依赖
- ✅ 添加 `sha2` 到 rc-api 依赖

#### 6. 编译面清理
- ✅ 已确认 approval 模块按 `.kiro/spec-10-go-approval` 要求属于废弃方案
- ✅ `rc-api` 主应用路由已移除 approval 接线
- ✅ `routes/mod.rs` / `db/mod.rs` 已移除 approval 模块导出，避免废弃实现继续阻塞编译

### ⚠️ 待完成的部分

#### 1. 数据库初始化
**问题**: sqlx 编译时检查需要数据库连接或缓存文件

**解决方案**:
```bash
# 方案 A: 启动数据库并生成缓存
cd D:/codeSpace/RCProtocol
docker-compose -f deploy/compose/docker-compose.yml up -d postgres
cd rust/rc-api
cargo sqlx prepare

# 方案 B: 使用离线模式（需要先有 sqlx-data.json）
SQLX_OFFLINE=true cargo build
```

#### 2. 路由注册到主应用
- ✅ 已完成，品牌路由已在 `rust/rc-api/src/main.rs` 中注册

#### 3. 集成测试
创建 `rust/rc-api/tests/brand_api_test.rs`：
- 测试品牌注册流程
- 测试 API Key 轮换
- 测试权限控制
- 测试邮箱唯一性

## API 接口清单

### 1. POST /api/v1/brands
**功能**: 注册新品牌（仅 Platform 角色）

**请求**:
```json
{
  "brand_name": "Luxury Watch Co.",
  "contact_email": "contact@luxurywatch.com",
  "industry": "Watches"
}
```

**响应**:
```json
{
  "brand_id": "brand_01HQZX3K4M5N6P7Q8R9S0T1U2V",
  "brand_name": "Luxury Watch Co.",
  "contact_email": "contact@luxurywatch.com",
  "industry": "Watches",
  "status": "Active",
  "api_key": {
    "key_id": "key_01HQZX3K4M5N6P7Q8R9S0T1U2V",
    "api_key": "rcpk_live_1234567890abcdef1234567890abcdef",
    "created_at": "2024-01-15T10:30:00Z",
    "expires_at": null,
    "note": "⚠️ 此 API Key 仅显示一次，请妥善保管"
  },
  "created_at": "2024-01-15T10:30:00Z"
}
```

### 2. POST /api/v1/brands/:brand_id/rotate-api-key
**功能**: 轮换 API Key（Platform 或品牌自己）

**请求**:
```json
{
  "reason": "定期安全轮换"
}
```

**响应**:
```json
{
  "key_id": "key_01HQZX3K4M5N6P7Q8R9S0T1U2V",
  "api_key": "rcpk_live_newkey1234567890abcdef1234567890",
  "created_at": "2024-01-15T11:00:00Z",
  "expires_at": null,
  "revoked_key_id": "key_01HQZX3K4M5N6P7Q8R9S0T1U2V",
  "note": "⚠️ 旧 API Key 已失效，请使用新密钥"
}
```

### 3. GET /api/v1/brands/:brand_id
**功能**: 查询品牌详情

**响应**:
```json
{
  "brand_id": "brand_01HQZX3K4M5N6P7Q8R9S0T1U2V",
  "brand_name": "Luxury Watch Co.",
  "contact_email": "contact@luxurywatch.com",
  "industry": "Watches",
  "status": "Active",
  "created_at": "2024-01-15T10:30:00Z",
  "updated_at": "2024-01-15T10:30:00Z"
}
```

### 4. GET /api/v1/brands/:brand_id/api-keys
**功能**: 列出品牌的所有 API Key

**响应**:
```json
{
  "keys": [
    {
      "key_id": "key_01HQZX3K4M5N6P7Q8R9S0T1U2V",
      "key_prefix": "rcpk_live_1234****",
      "status": "Active",
      "created_at": "2024-01-15T11:00:00Z",
      "last_used_at": "2024-01-15T12:00:00Z",
      "revoked_at": null
    },
    {
      "key_id": "key_01HQZX3K4M5N6P7Q8R9S0T1U2V",
      "key_prefix": "rcpk_live_5678****",
      "status": "Revoked",
      "created_at": "2024-01-15T10:30:00Z",
      "last_used_at": "2024-01-15T10:59:00Z",
      "revoked_at": "2024-01-15T11:00:00Z"
    }
  ]
}
```

### 5. GET /api/v1/brands
**功能**: 列出品牌（带分页）

**查询参数**:
- `page`: 页码（默认 1）
- `page_size`: 每页数量（默认 20，最大 100）

**响应**:
```json
{
  "items": [...],
  "total": 42,
  "page": 1,
  "page_size": 20
}
```

## 业务规则

1. **邮箱唯一性**: 同一邮箱只能注册一个品牌
2. **行业枚举**: Watches, Fashion, Wine, Jewelry, Art, Other
3. **API Key 格式**: `rcpk_live_<32 hex chars>`（42 字符）
4. **Key ID 格式**: `key_<ULID>`（30 字符）
5. **Brand ID 格式**: `brand_<ULID>`（32 字符）
6. **权限控制**:
   - 注册品牌：仅 Platform
   - 轮换 API Key：Platform 或品牌自己
   - 查询详情：Platform 或品牌自己
   - 列出 API Key：Platform 或品牌自己

## 下一步行动

1. **启动数据库并生成 sqlx 缓存**
   ```bash
   cd D:/codeSpace/RCProtocol
   docker-compose -f deploy/compose/docker-compose.yml up -d postgres
   cd rust/rc-api
   cargo sqlx prepare
   ```

2. **补品牌 API 集成测试**
   - 创建 `rust/rc-api/tests/brand_api_test.rs`
   - 测试完整的注册和轮换流程

3. **编译验证**
   ```bash
   cd D:/codeSpace/RCProtocol/rust
   cargo build --release
   ```

4. **更新任务状态**
   - 标记 Task #17 为 completed
   - 更新 Spec-04 的验收状态

## 技术亮点

1. **安全设计**:
   - API Key 使用 SHA-256 哈希存储
   - 明文密钥仅在创建/轮换时返回一次
   - 支持密钥轮换和撤销
2. **规范一致性**:
   - 品牌主资源 ID 已与 `spec-06-id-unification` 对齐
   - 废弃审批模块已退出主编译面，避免旧架构残留继续干扰实现验证
