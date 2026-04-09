# Spec-04: 品牌极简注册与 API Key 管理

> **状态**: ✅ 已落地实现  
> **优先级**: P0 - 最高  
> **依赖**: Spec-03（鉴权与租户上下文）, Spec-06（统一 ID 策略）  
> **负责人**: RCProtocol  
> **最后更新**: 2026-04-08

---

## 1. 目标与范围

### 1.1 核心目标

为品牌侧提供最小可用的注册与 API Key 生命周期管理能力，支持：

1. 平台侧创建品牌
2. 系统自动生成 `brand_id` 与初始 API Key
3. 品牌侧使用 API Key 访问自身管理接口
4. 品牌侧或平台侧执行 API Key 轮换
5. 品牌侧查询自身品牌详情与历史 API Key 列表

### 1.2 已落地交付范围

**已实现：**
- ✅ `POST /brands` - 品牌极简注册（3 字段）
- ✅ `POST /brands/:brand_id/api-keys/rotate` - API Key 轮换
- ✅ `GET /brands/:brand_id` - 品牌详情查询
- ✅ `GET /brands/:brand_id/api-keys` - API Key 列表查询
- ✅ 自动生成 `brand_id` 与初始 API Key
- ✅ API Key 明文仅在创建/轮换时返回一次
- ✅ 品牌边界校验
- ✅ 品牌注册 / 轮换 / 旧 Key 失效 / 新 Key 生效 集成测试

**未纳入本 Spec 的范围：**
- ❌ 产品管理
- ❌ 系统内品牌审批流（MVP 已废弃）
- ❌ 品牌删除接口

**说明：** 当前 `routes/brand.rs` 还额外承载了品牌列表、品牌更新、批量查询等扩展接口，但这些不属于本 Spec 的核心验收范围。

---

## 2. 业务场景

### 2.1 品牌注册流程

```text
平台运营人员 → 填写品牌基本信息（3 字段）
              ↓
           POST /brands
              ↓
    系统生成 brand_id + API Key
              ↓
    返回 API Key 明文（仅此一次）
              ↓
    运营人员将 API Key 交付给品牌方
```

### 2.2 API Key 轮换流程

```text
品牌方 / 平台运营 → 请求轮换 API Key
                 ↓
POST /brands/:brand_id/api-keys/rotate
                 ↓
         旧 Key 标记为 Revoked
                 ↓
         生成新 Key 并返回明文
                 ↓
         品牌方更新集成配置
```

---

## 3. 已实现 API 设计

### 3.1 品牌注册

**请求：**

```http
POST /brands
Content-Type: application/json
Authorization: Bearer <platform_jwt>

{
  "brand_name": "Luxury Watch Co.",
  "contact_email": "api@luxurywatch.com",
  "industry": "Watches"
}
```

**成功响应：**

```json
{
  "brand_id": "brand_01HQZX3K4M5N6P7Q8R9S0T1U2V",
  "brand_name": "Luxury Watch Co.",
  "contact_email": "api@luxurywatch.com",
  "industry": "Watches",
  "status": "Active",
  "api_key": {
    "key_id": "key_01HQZX3K4M5N6P7Q8R9S0T1U2W",
    "api_key": "rcpk_live_1234567890abcdef1234567890abcdef",
    "created_at": "2026-04-08T10:30:00Z",
    "expires_at": null,
    "note": "⚠️ 此 API Key 仅显示一次，请妥善保管"
  },
  "created_at": "2026-04-08T10:30:00Z"
}
```

**失败响应特征：**
- 重复邮箱：`409 Conflict`
- 非 Platform：`403 Forbidden`
- 输入非法：`400 Bad Request`

**字段规则：**

| 字段 | 类型 | 必填 | 说明 |
|------|------|------|------|
| brand_name | string | ✅ | 品牌名称 |
| contact_email | string | ✅ | 联系邮箱，唯一 |
| industry | string | ✅ | `Watches/Fashion/Wine/Jewelry/Art/Other` |

**业务规则：**
1. `contact_email` 全局唯一
2. `brand_id` 使用 `brand_<ULID>`
3. 初始 API Key 格式为 `rcpk_live_<32 hex>`
4. API Key 存储为 SHA-256 哈希值
5. 品牌初始状态为 `Active`
6. 仅 Platform 角色可调用

---

### 3.2 API Key 轮换

**请求：**

```http
POST /brands/:brand_id/api-keys/rotate
Content-Type: application/json
X-Api-Key: <brand_api_key>

{
  "reason": "scheduled rotation"
}
```

**成功响应：**

```json
{
  "key_id": "key_01HQZX3K4M5N6P7Q8R9S0T1U2X",
  "api_key": "rcpk_live_abcdef1234567890abcdef1234567890",
  "created_at": "2026-04-08T11:00:00Z",
  "expires_at": null,
  "revoked_key_id": "key_01HQZX3K4M5N6P7Q8R9S0T1U2W",
  "note": "⚠️ 旧 API Key 已失效，请使用新密钥"
}
```

**业务规则：**
1. 旧 Key 置为 `Revoked`
2. 写入 `revoked_at`
3. 新 Key 立即生效
4. 旧 Key 立即失效，无宽限期
5. Platform 或品牌自身可调用

---

### 3.3 品牌详情查询

**请求：**

```http
GET /brands/:brand_id
X-Api-Key: <brand_api_key>
```

**响应：**

```json
{
  "brand_id": "brand_01HQZX3K4M5N6P7Q8R9S0T1U2V",
  "brand_name": "Luxury Watch Co.",
  "contact_email": "api@luxurywatch.com",
  "industry": "Watches",
  "status": "Active",
  "created_at": "2026-04-08T10:30:00Z",
  "updated_at": "2026-04-08T10:30:00Z"
}
```

**安全规则：**
- 不返回 `api_key`
- 不返回 `key_hash`

---

### 3.4 API Key 列表查询

**请求：**

```http
GET /brands/:brand_id/api-keys
X-Api-Key: <brand_api_key>
```

**响应：**

```json
{
  "keys": [
    {
      "key_id": "key_01HQZX3K4M5N6P7Q8R9S0T1U2X",
      "key_prefix": "rcpk_live_abcd****",
      "status": "Active",
      "created_at": "2026-04-08T11:00:00Z",
      "last_used_at": "2026-04-08T12:30:00Z",
      "revoked_at": null
    },
    {
      "key_id": "key_01HQZX3K4M5N6P7Q8R9S0T1U2W",
      "key_prefix": "rcpk_live_1234****",
      "status": "Revoked",
      "created_at": "2026-04-08T10:30:00Z",
      "last_used_at": null,
      "revoked_at": "2026-04-08T11:00:00Z"
    }
  ]
}
```

**业务规则：**
1. 仅显示 `key_prefix`
2. 包含 Active + Revoked
3. 按创建时间倒序

---

## 4. 已落地数据模型

### 4.1 `brands` 表

```sql
CREATE TABLE brands (
    brand_id TEXT PRIMARY KEY,
    brand_name TEXT NOT NULL,
    contact_email TEXT NOT NULL UNIQUE,
    industry TEXT NOT NULL,
    status TEXT NOT NULL DEFAULT 'Active',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE UNIQUE INDEX idx_brands_contact_email ON brands(contact_email);
CREATE INDEX idx_brands_status ON brands(status);
CREATE INDEX idx_brands_created_at ON brands(created_at DESC);
```

### 4.2 `api_keys` 表

```sql
CREATE TABLE api_keys (
    key_id TEXT PRIMARY KEY,
    brand_id TEXT NOT NULL REFERENCES brands(brand_id) ON DELETE CASCADE,
    key_hash TEXT NOT NULL,
    key_prefix TEXT NOT NULL,
    status TEXT NOT NULL DEFAULT 'Active',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    revoked_at TIMESTAMPTZ,
    last_used_at TIMESTAMPTZ,
    expires_at TIMESTAMPTZ,
    metadata JSONB DEFAULT '{}'::jsonb
);

CREATE INDEX idx_api_keys_brand_id ON api_keys(brand_id);
CREATE INDEX idx_api_keys_status ON api_keys(status);
CREATE UNIQUE INDEX idx_api_keys_key_hash ON api_keys(key_hash);
```

### 4.3 相关落地文件

- 路由：`rust/rc-api/src/routes/brand.rs`
- DB：`rust/rc-api/src/db/brands.rs`
- API Key 工具：`rust/rc-api/src/auth/api_key.rs`
- Migration：`rust/rc-api/migrations/20250101000022_finalize_brand_registration.sql`
- 集成测试：`rust/rc-api/tests/brand_registration_integration.rs`

---

## 5. 实现要点

### 5.1 ID 规则

已与 `spec-06-id-unification` 对齐：

- `brand_id` → `brand_<ULID>`
- `key_id` → `key_<ULID>`

### 5.2 API Key 规则

- 生成格式：`rcpk_live_<32 hex>`
- 存储形式：`SHA-256(api_key)`
- 展示形式：前缀 + `****`
- 明文只在创建 / 轮换时返回一次

### 5.3 权限模型

- 品牌注册：仅 `Platform`
- 品牌详情：`Platform` 或品牌自身
- API Key 列表：`Platform` 或品牌自身
- API Key 轮换：`Platform` 或品牌自身

### 5.4 事务模型

品牌注册与 Key 轮换均通过事务提交，保证：

- 注册时品牌与初始 Key 一起落库
- 轮换时旧 Key 撤销与新 Key 创建同事务完成

---

## 6. 已验证结果

### 6.1 功能验收

- [x] 品牌注册接口可正常调用，返回 `brand_id` 和明文 `api_key`
- [x] API Key 明文仅在创建/轮换时返回一次
- [x] `contact_email` 唯一性校验生效
- [x] API Key 轮换后旧 Key 立即失效
- [x] 权限校验正确（仅 Platform 可注册品牌）
- [x] 品牌详情查询不返回 API Key
- [x] API Key 列表仅显示前缀

### 6.2 安全验收

- [x] API Key 存储为 SHA-256 哈希
- [x] API Key 明文不在详情 / 列表接口返回
- [x] 轮换后旧 Key 无宽限期，立即失效
- [x] 非授权品牌无法跨品牌读取信息

### 6.3 测试验收

已通过：

```bash
cargo test -p rc-api --test brand_registration_integration
```

测试覆盖：
- 品牌注册
- 重复邮箱冲突
- 品牌详情不回传 API Key
- API Key 轮换
- 旧 Key 失效
- 新 Key 生效
- API Key 列表状态验证

联调脚本统一入口：

```bash
./scripts/test-brand-registration.sh
```

---

## 7. 风险与说明

### 7.1 已处理风险

| 风险 | 结果 |
|------|------|
| 品牌注册代码与 migration 不一致 | 已通过 `20250101000022_finalize_brand_registration.sql` 收敛 |
| 旧 Key 撤销后仍可访问 | 已由集成测试验证失效 |
| 重复 shell 脚本造成入口混乱 | 已统一收敛到 `test-brand-registration.sh` |

### 7.2 当前说明

- 本 Spec 已进入“已落地实现态”
- 后续若继续扩展品牌管理（如更新/删除、多 Key scope），应新建增量 spec，而不是回滚本 Spec 的已实现结论

---

## 8. 参考资料

- [Task: Spec-04 品牌极简注册与 API Key 管理](../tasks/task-spec-04-brand-registration.md)
- [品牌 API 对接指南](../api/brand-api-guide.md)
- [Spec-06: 统一 ID 策略](./spec-06-id-unification.md)
- [工程补齐工作总结](../engineering/补齐工作总结.md)
