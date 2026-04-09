# Task: Spec-04 品牌极简注册与 API Key 管理

> **关联 Spec**: [spec-04-brand-registration.md](../specs/spec-04-brand-registration.md)  
> **状态**: ✅ 已完成（含补齐项）  
> **优先级**: P0  
> **预计工期**: 2-3 天  
> **开始时间**: 2026-04-08  
> **完成时间**: 2026-04-08

---

## 任务分解

### Phase 1: 数据库层（0.5 天）

#### Task 1.1: 创建数据库表结构
- [x] 创建 `brands` 表（如果不存在）
  - 字段：brand_id, brand_name, contact_email, industry, status, created_at, updated_at
  - 索引：contact_email 唯一索引、status 索引、created_at 索引
- [x] 创建 `api_keys` 表（如果不存在）
  - 字段：key_id, brand_id, key_hash, key_prefix, status, created_at, revoked_at, last_used_at
  - 索引：brand_id 索引、status 索引、key_hash 唯一索引
- [x] 验证表结构与 spec 一致

**文件位置**: `deploy/postgres/init/001_init.sql`

**验收标准**:
- [x] 表创建成功，无 SQL 错误
- [x] 索引创建成功
- [x] 外键约束生效

---

#### Task 1.2: 实现数据库操作函数
- [x] 创建 `src/db/brands.rs` 文件
- [x] 实现 `create_brand()` - 插入品牌记录
- [x] 实现 `fetch_brand_by_id()` - 查询品牌详情
- [x] 实现 `fetch_brand_by_email()` - 通过邮箱查询（唯一性校验）
- [x] 实现 `create_api_key()` - 插入 API Key 记录
- [x] 实现 `revoke_api_key()` - 标记 API Key 为 Revoked
- [x] 实现 `fetch_api_keys_by_brand()` - 查询品牌的所有 API Key
- [x] 实现 `update_api_key_last_used()` - 更新最后使用时间

**文件位置**: `rust/rc-api/src/db/brands.rs`

**代码结构**:
```rust
use sqlx::PgPool;
use uuid::Uuid;
use crate::errors::RcError;

pub struct BrandRecord {
    pub brand_id: String,
    pub brand_name: String,
    pub contact_email: String,
    pub industry: String,
    pub status: String,
    pub created_at: chrono::DateTime<chrono::Utc>,
    pub updated_at: chrono::DateTime<chrono::Utc>,
}

pub struct ApiKeyRecord {
    pub key_id: String,
    pub brand_id: String,
    pub key_hash: String,
    pub key_prefix: String,
    pub status: String,
    pub created_at: chrono::DateTime<chrono::Utc>,
    pub revoked_at: Option<chrono::DateTime<chrono::Utc>>,
    pub last_used_at: Option<chrono::DateTime<chrono::Utc>>,
}
```

**验收标准**:
- [x] 所有函数编译通过
- [x] 错误处理完整（数据库错误转换为 RcError）
- [x] 使用事务处理（create_brand + create_api_key）

---

### Phase 2: 业务逻辑层（1 天）

#### Task 2.1: 实现 API Key 生成与哈希
- [x] 创建 `src/auth/api_key.rs` 文件
- [x] 实现 `generate_api_key()` - 生成 `rcpk_live_<32字符>` 格式
- [x] 实现 `hash_api_key()` - SHA-256 哈希
- [x] 实现 `extract_key_prefix()` - 提取前 16 字符
- [x] 实现 `generate_key_id()` - 生成 ULID 格式 key_id
- [x] 实现 `generate_brand_id()` - 生成 ULID 格式 brand_id

**文件位置**: `rust/rc-api/src/auth/api_key.rs`

**验收标准**:
- [x] API Key 格式正确（rcpk_live_ + 32字符）
- [x] 哈希结果为 64 字符十六进制
- [x] ULID 格式正确

---

#### Task 2.2: 实现品牌注册路由
- [x] 在 `src/routes/brand.rs` 中添加 `register_brand()` 函数
- [x] 定义请求结构体 `RegisterBrandRequest`
- [x] 定义响应结构体 `RegisterBrandResponse`
- [x] 实现业务逻辑：
  - [x] 权限校验（仅 Platform 角色）
  - [x] 邮箱唯一性校验
  - [x] 生成 brand_id 和 API Key
  - [x] 事务写入 brands 和 api_keys 表
  - [x] 返回明文 API Key

**文件位置**: `rust/rc-api/src/routes/brand.rs`

**验收标准**:
- [x] 权限校验生效
- [x] 邮箱唯一性校验生效
- [x] 事务正确提交
- [x] 返回明文 API Key

---

#### Task 2.3: 实现 API Key 轮换路由
- [x] 在 `src/routes/brand.rs` 中添加 `rotate_api_key()` 函数
- [x] 定义请求结构体 `RotateApiKeyRequest`
- [x] 定义响应结构体 `RotateApiKeyResponse`
- [x] 实现业务逻辑：
  - [x] 权限校验（Platform 或品牌自身）
  - [x] 查询当前 Active 的 API Key
  - [x] 标记旧 Key 为 Revoked
  - [x] 生成新 API Key
  - [x] 返回明文新 Key

**文件位置**: `rust/rc-api/src/routes/brand.rs`

**验收标准**:
- [x] 旧 Key 立即失效
- [x] 新 Key 立即生效
- [x] 返回 revoked_key_id

---

#### Task 2.4: 实现品牌详情查询路由
- [x] 在 `src/routes/brand.rs` 中添加 `get_brand()` 函数
- [x] 定义响应结构体 `BrandDetailResponse`
- [x] 实现业务逻辑：
  - [x] 权限校验（Platform 或品牌自身）
  - [x] 查询品牌详情
  - [x] 不返回 API Key 信息

**验收标准**:
- [x] 不返回 API Key
- [x] 权限校验生效

---

#### Task 2.5: 实现 API Key 列表查询路由
- [x] 在 `src/routes/brand.rs` 中添加 `list_api_keys()` 函数
- [x] 定义响应结构体 `ApiKeyListResponse`
- [x] 实现业务逻辑：
  - [x] 权限校验（Platform 或品牌自身）
  - [x] 查询品牌的所有 API Key
  - [x] 仅返回 key_prefix（前 16 字符 + ****）

**验收标准**:
- [x] 仅显示 key_prefix
- [x] 包含 Active 和 Revoked 状态
- [x] 按创建时间倒序

---

### Phase 3: 路由注册（0.5 天）

#### Task 3.1: 注册路由
- [x] 在 `src/routes/brand.rs` 中创建 `brand_routes()` 函数（实际实现名为 `router()`）
- [x] 注册以下路由：
  - [x] POST /api/v1/brands
  - [x] POST /api/v1/brands/:id/api-keys/rotate
  - [x] GET /api/v1/brands/:id
  - [x] GET /api/v1/brands/:id/api-keys
- [x] 在 `src/main.rs` 中挂载 brand_routes

**文件位置**: `rust/rc-api/src/routes/brand.rs`, `rust/rc-api/src/main.rs`

**验收标准**:
- [x] 路由注册成功
- [x] 编译通过
- [x] 服务启动无错误

---

### Phase 4: 测试与验证（0.5 天）

#### Task 4.1: 编写单元测试
- [x] 测试 `generate_api_key()` 格式正确
- [x] 测试 `hash_api_key()` 哈希正确
- [x] 测试 `extract_key_prefix()` 前缀提取正确
- [x] 测试数据库操作函数（使用测试数据库）

**文件位置**: `rust/rc-api/src/auth/api_key.rs`, `rust/rc-api/src/db/brands.rs`, `rust/rc-api/tests/brand_registration_integration.rs`

---

#### Task 4.2: 编写集成测试脚本
- [x] 创建 `scripts/test-brand-registration.sh` 脚本
- [x] 测试品牌注册流程
- [x] 测试 API Key 轮换流程
- [x] 测试邮箱唯一性校验
- [x] 测试权限校验

**文件位置**: `scripts/test-brand-registration.sh`

**验收标准**:
- [x] 所有测试通过
- [x] 无错误日志
- [x] 数据库状态正确

---

#### Task 4.3: 更新文档
- [x] 更新 `docs/api/brand-api-guide.md`，添加品牌注册说明
- [x] 更新 `README.md`，添加品牌注册快速开始指南
- [x] 更新 `docs/engineering/补齐工作总结.md`

---

## 依赖关系

```
Task 1.1 (数据库表)
  └─→ Task 1.2 (数据库操作函数)
        └─→ Task 2.1 (API Key 生成)
              └─→ Task 2.2 (品牌注册路由)
              └─→ Task 2.3 (API Key 轮换路由)
              └─→ Task 2.4 (品牌详情查询)
              └─→ Task 2.5 (API Key 列表查询)
                    └─→ Task 3.1 (路由注册)
                          └─→ Task 4.1 (单元测试)
                          └─→ Task 4.2 (集成测试)
                          └─→ Task 4.3 (文档更新)
```

---

## 检查清单

### 开发前检查
- [x] 已阅读 [spec-04-brand-registration.md](../specs/spec-04-brand-registration.md)
- [x] 已理解业务场景和 API 设计
- [x] 已确认数据库表结构
- [x] 已准备测试环境

### 开发中检查
- [x] 代码遵循 Rust 最佳实践
- [x] 错误处理完整
- [x] 使用事务保证数据一致性
- [x] API Key 明文不记录日志
- [x] 权限校验正确

### 开发后检查
- [x] 所有单元测试通过
- [x] 集成测试通过
- [x] 代码审查通过
- [x] 文档已更新
- [x] 无安全漏洞

---

## 时间估算

| Phase | 任务 | 预计时间 |
|-------|------|---------|
| Phase 1 | 数据库层 | 0.5 天 |
| Phase 2 | 业务逻辑层 | 1 天 |
| Phase 3 | 路由注册 | 0.5 天 |
| Phase 4 | 测试与验证 | 0.5 天 |
| **总计** | | **2.5 天** |

---

## 完成标准

- [x] 所有 Task 完成
- [x] 所有测试通过
- [x] 文档已更新
- [x] Code Review 通过
- [x] 部署到测试环境验证通过

---

## 完成说明

本任务已完成并回填状态，补齐了以下缺失交付物：

1. 新增 `rust/rc-api/tests/brand_registration_integration.rs`，覆盖品牌注册 / Key 轮换 / 旧 Key 失效 / 新 Key 生效 / Key 列表 / 邮箱唯一性。
2. 新增 `scripts/test-brand-registration.sh`，作为标准联调脚本。
3. 更新 `README.md`，补充品牌注册快速开始。
4. 更新 `docs/api/brand-api-guide.md`，纳入品牌注册与 API Key 管理入口说明。

说明：`routes/brand.rs` 现有实现额外包含品牌列表、品牌更新、批量查询等扩展能力，超出 spec-04 的 MVP 范围，但不影响本任务验收。
