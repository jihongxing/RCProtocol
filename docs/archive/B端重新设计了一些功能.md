> **⚠️ 已归档**：本文档内容已于 2026-04-07 合并至权威基线文档，仅作历史参考。  
> 变更已写入：`domain-model.md`、`api-and-service-boundaries.md`、`mvp-scope-and-cutline.md`、`product-system.md`、`system-architecture.md`  
> 重构计划：`docs/refactoring-plan.md`

# B 端轻量级方案重新设计

## 核心问题分析

### 当前设计的问题

- ❌ 假设品牌会用我们的系统管理 SKU
- ❌ 假设品牌会走我们的审批流程
- ❌ 假设品牌会改造他们的 ERP 系统

### 商业现实

- ✅ 品牌已有自己的 ERP / PLM / PIM 系统
- ✅ 品牌已有自己的 SKU 管理体系
- ✅ 品牌已有自己的审批流程
- ✅ 品牌只想要"防伪 + 验真 + 过户"能力

### 我们的定位

| 不是 | 而是 |
| --- | --- |
| 品牌的 ERP 系统 | 品牌 ERP 的"防伪插件" |
| 品牌的 SKU 管理系统 | 品牌 SKU 的"数字主权层" |
| 品牌的审批系统 | 品牌审批后的"执行层" |

---

## 设计原则

1. 最小化品牌方的数据录入
2. 兼容品牌方现有系统
3. 提供 API 对接能力
4. 支持手动和自动两种模式

---

## 一、品牌管理：极简化

### 当前设计（过重）

```typescript
// 品牌需要填写一堆信息
interface Brand {
  brand_id: string
  brand_name: string
  brand_name_en: string
  brand_logo: string
  brand_description: string
  brand_website: string
  brand_contact_email: string
  brand_contact_phone: string
  legal_entity_name: string
  business_license: string
  tax_id: string
  address: string
  // ... 还有 20 个字段
}
```

### 优化设计（极简）

```typescript
// 品牌只需要填写 3 个核心字段
interface Brand {
  brand_id: string           // 品牌唯一标识（可以是品牌方自己的 ID）
  brand_name: string         // 品牌名称
  api_key: string            // API 密钥（用于对接）

  // 可选字段（用于展示，不强制）
  brand_logo?: string
  brand_website?: string

  // 系统字段
  created_at: timestamp
  status: 'Active' | 'Suspended'
}
```

### 数据库设计

```sql
CREATE TABLE brands (
    brand_id VARCHAR(64) PRIMARY KEY,  -- 品牌自己的 ID（不是我们生成的）
    brand_name VARCHAR(128) NOT NULL,
    api_key VARCHAR(128) NOT NULL UNIQUE,

    -- 可选展示字段
    brand_logo TEXT,
    brand_website TEXT,

    -- 系统字段
    status VARCHAR(32) NOT NULL DEFAULT 'Active',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    created_by VARCHAR(64) NOT NULL
);
```

### B 端界面（极简）

```
┌─────────────────────────────────────┐
│ 创建品牌                              │
├─────────────────────────────────────┤
│ 品牌 ID*:  [hermès_official_2024  ] │  ← 品牌自己定义
│ 品牌名称*: [Hermès                 ] │
│                                     │
│ 品牌 Logo:  [上传图片（可选）]        │
│ 品牌官网:   [https://...（可选）]    │
│                                     │
│          [取消]  [创建品牌]          │
└─────────────────────────────────────┘
```

**优势：**

- 品牌方 30 秒完成注册
- 不需要填写一堆无关信息
- `brand_id` 可以是品牌方自己的 ID（方便对接）

---

## 二、SKU 管理：不管理 SKU

> 核心理念：我们不管理 SKU，我们只管理"资产与 SKU 的映射关系"

### 当前设计（过重）

```typescript
// 我们维护完整的 SKU 信息
interface Product {
  product_id: string
  product_name: string
  product_name_en: string
  product_category: string
  product_description: string
  product_images: string[]
  product_price: number
  product_currency: string
  product_specs: object
  // ... 还有 30 个字段
}
```

### 优化设计（极简）

```typescript
// 我们只存储"资产与外部 SKU 的映射"
interface AssetProductMapping {
  asset_id: string               // 我们的资产 ID
  external_product_id: string    // 品牌方的 SKU ID（我们不关心它是什么）
  external_product_name?: string // 可选：品牌方的 SKU 名称（用于展示）
  external_product_url?: string  // 可选：品牌方的 SKU 详情页（用于跳转）
}
```

### 数据库设计

```sql
CREATE TABLE assets (
    asset_id VARCHAR(64) PRIMARY KEY,
    uid VARCHAR(14) NOT NULL UNIQUE,  -- 子标签 UID
    brand_id VARCHAR(64) NOT NULL,

    -- 外部 SKU 映射（关键字段）
    external_product_id VARCHAR(128),   -- 品牌方的 SKU ID
    external_product_name VARCHAR(256), -- 品牌方的 SKU 名称（可选）
    external_product_url TEXT,          -- 品牌方的 SKU 详情页（可选）

    -- 状态机
    current_state VARCHAR(32) NOT NULL,

    -- 所有权
    owner_id VARCHAR(64),

    -- 审计
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    CONSTRAINT fk_brand FOREIGN KEY (brand_id) REFERENCES brands(brand_id)
);

CREATE INDEX idx_asset_brand ON assets(brand_id);
CREATE INDEX idx_asset_external_product ON assets(external_product_id);
CREATE INDEX idx_asset_state ON assets(current_state);
```

**优势：**

- 我们不维护 SKU 的详细信息（价格、规格、库存等）
- 品牌方只需要告诉我们"这批资产对应哪个 SKU ID"
- 如果品牌方想展示 SKU 详情，可以提供一个跳转链接

---

## 三、审批流程：不做审批，只做执行

> 核心理念：品牌方在自己的系统里审批，我们只负责执行

### 当前设计（过重）

1. 发起激活申请
2. 等待审批
3. 审批通过
4. 执行激活

> ❌ 问题：品牌方已有自己的审批系统，为什么要用我们的？

### 优化设计（执行层）

1. 品牌方在自己的系统里发起激活申请
2. 走自己的审批流程
3. 审批通过后，调用我们的 API 执行激活

> ✅ 我们只负责：验证权限 + 执行激活 + 记录审计

### API 设计

```http
POST /api/v1/assets/batch-activate
Authorization: Bearer <brand_api_key>
Content-Type: application/json

{
  "asset_ids": ["asset_001", "asset_002", ...],
  "external_product_id": "HERMES_BIRKIN_30_BLACK_2024",
  "external_product_name": "Birkin 30 黑色 2024款",  // 可选
  "external_product_url": "https://...",              // 可选
  "operator_id": "brand_admin_001",                   // 品牌方的操作人 ID
  "approval_id": "APPROVAL_20240407_001"              // 品牌方的审批单号（可选）
}
```

```json
// Response
{
  "success": true,
  "activated_count": 500,
  "failed_count": 0,
  "task_id": "task_001"
}
```

---

## 四、两种接入模式

### 模式 A：API 对接（推荐）

**适用场景：**

- 品牌方有自己的 ERP / PLM 系统
- 品牌方有技术团队
- 品牌方希望自动化

**接入流程：**

1. 品牌方注册，获得 API Key
2. 品牌方在自己的系统里集成我们的 API
3. 品牌方在自己的系统里完成审批
4. 品牌方调用我们的 API 执行激活

**API 列表：**

```http
# 1. 工厂盲扫（工厂调用）
POST /api/v1/factory/quick-log
{ "uid": "04A1B2C3D4E5F6", "factory_id": "factory_001" }

# 2. 批量激活（品牌方调用）
POST /api/v1/assets/batch-activate
{ "asset_ids": ["asset_001", ...], "external_product_id": "SKU_001" }

# 3. 查询资产状态（品牌方调用）
GET /api/v1/assets/{asset_id}

# 4. 查询过户记录（品牌方调用）
GET /api/v1/assets/{asset_id}/transfers

# 5. Webhook 回调（我们调用品牌方）
POST <brand_webhook_url>
{
  "event": "asset_transferred",
  "asset_id": "asset_001",
  "from_user": "user_001",
  "to_user": "user_002",
  "timestamp": "2024-04-07T12:00:00Z"
}
```

### 模式 B：Web 后台（备选）

**适用场景：**

- 品牌方没有技术团队
- 品牌方没有 ERP 系统
- 品牌方希望快速上手

**接入流程：**

1. 品牌方注册，登录 Web 后台
2. 品牌方手动上传待激活资产列表（Excel）
3. 品牌方手动填写 SKU 信息
4. 品牌方点击"批量激活"


**B 端界面（极简）：**

```
┌─────────────────────────────────────────────┐
│ RCProtocol 品牌后台                          │
├─────────────────────────────────────────────┤
│ 导航：                                       │
│ - 待激活资产                                 │
│ - 已激活资产                                 │
│ - 过户记录                                   │
│ - API 密钥                                   │
└─────────────────────────────────────────────┘
```

---

## 五、完整数据库设计（简化版）

```sql
-- 1. 品牌表（极简）
CREATE TABLE brands (
    brand_id VARCHAR(64) PRIMARY KEY,
    brand_name VARCHAR(128) NOT NULL,
    api_key VARCHAR(128) NOT NULL UNIQUE,
    brand_logo TEXT,
    brand_website TEXT,
    webhook_url TEXT,  -- 品牌方的 Webhook 地址（可选）
    status VARCHAR(32) NOT NULL DEFAULT 'Active',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    created_by VARCHAR(64) NOT NULL
);

-- 2. 资产表（简化）
CREATE TABLE assets (
    asset_id VARCHAR(64) PRIMARY KEY,
    uid VARCHAR(14) NOT NULL UNIQUE,
    brand_id VARCHAR(64) NOT NULL,
    external_product_id VARCHAR(128),
    external_product_name VARCHAR(256),
    external_product_url TEXT,
    current_state VARCHAR(32) NOT NULL,
    owner_id VARCHAR(64),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT fk_brand FOREIGN KEY (brand_id) REFERENCES brands(brand_id)
);

-- 3. 母卡凭证表
CREATE TABLE authority_devices (
    authority_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    authority_uid VARCHAR(32) NOT NULL UNIQUE,
    authority_type VARCHAR(32) NOT NULL,
    brand_id VARCHAR(64) NOT NULL,
    status VARCHAR(32) NOT NULL DEFAULT 'Active'
);

-- 4. 母子绑定表
CREATE TABLE asset_entanglements (
    entanglement_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    asset_id VARCHAR(64) NOT NULL,
    authority_id UUID NOT NULL,
    entanglement_state VARCHAR(32) NOT NULL DEFAULT 'Active'
);

-- 5. 过户记录表
CREATE TABLE asset_transfers (
    transfer_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    asset_id VARCHAR(64) NOT NULL,
    from_user_id VARCHAR(64) NOT NULL,
    to_user_id VARCHAR(64) NOT NULL,
    transfer_fee DECIMAL(10, 2),
    brand_share DECIMAL(10, 2),
    platform_share DECIMAL(10, 2),
    transferred_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    trace_id VARCHAR(64) NOT NULL,
    CONSTRAINT fk_asset FOREIGN KEY (asset_id) REFERENCES assets(asset_id)
);

-- 6. 审计事件表
CREATE TABLE audit_events (
    event_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    asset_id VARCHAR(64),
    actor_id VARCHAR(64) NOT NULL,
    actor_role VARCHAR(32) NOT NULL,
    action VARCHAR(64) NOT NULL,
    from_state VARCHAR(32),
    to_state VARCHAR(32),
    trace_id VARCHAR(64) NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
```

---

## 六、品牌方视角的完整流程

### 场景 1：有技术团队的大品牌（API 对接）

1. 注册品牌，获得 API Key
2. 在自己的 ERP 系统里集成我们的 API
3. 工厂生产时，调用我们的盲扫接口
4. 在自己的系统里审批激活
5. 审批通过后，调用我们的激活接口
6. 商品售出后，我们的 Webhook 通知品牌方
7. 品牌方在自己的系统里看到过户记录和分成

> 品牌方工作量：集成 3 个 API + 配置 1 个 Webhook，总计 1-2 天

### 场景 2：没有技术团队的小品牌（Web 后台）

1. 注册品牌，登录 Web 后台
2. 工厂生产时，工厂用我们的 App 扫码
3. 品牌方在 Web 后台看到待激活资产
4. 品牌方选中资产，填写 SKU 信息，点击"批量激活"
5. 商品售出后，品牌方在 Web 后台看到过户记录和分成

> 品牌方工作量：注册 5 分钟 + 培训工厂 10 分钟 + 批量激活 5 分钟，总计 20 分钟

---

## 七、对比总结

| 维度 | 当前设计 | 优化设计 | 改进 |
| --- | --- | --- | --- |
| 品牌注册 | 填写 30+ 字段 | 填写 3 个字段 | 简化 90% |
| SKU 管理 | 维护完整 SKU 信息 | 只存储 SKU ID 映射 | 简化 95% |
| 审批流程 | 在我们的系统里审批 | 品牌方自己审批 | 解耦 100% |
| 接入方式 | 只有 Web 后台 | API + Web 后台 | 灵活性 +100% |
| 接入成本 | 需要改造 ERP | 只需集成 API | 成本 -80% |

---

## 八、实施建议

### Phase 1：MVP（1 个月）

1. 实现极简品牌注册（3 个字段）
2. 实现资产与外部 SKU 映射
3. 实现 3 个核心 API（盲扫、激活、查询）
4. 实现极简 Web 后台（待激活、已激活、过户记录）

### Phase 2：API 对接（2 个月）

1. 完善 API 文档
2. 实现 Webhook 回调
3. 提供 SDK（Python / Node.js / Go）
4. 对接第一个大品牌

### Phase 3：增强功能（3 个月）

1. 实现批量导入（Excel）
2. 实现数据导出
3. 实现报表统计
4. 实现品牌自定义 Webhook

---

## 总结

> **核心设计理念：我们不是品牌的 ERP，我们是品牌 ERP 的"防伪插件"**

**关键优化：**

1. 品牌管理极简化：3 个字段注册
2. 不管理 SKU：只存储 SKU ID 映射
3. 不做审批：品牌方自己审批，我们只执行
4. 两种接入模式：API 对接 + Web 后台

**效果：**

- ✅ 接入成本降低 80%
- ✅ 兼容品牌方现有系统
- ✅ 支持大品牌和小品牌
- ✅ 保持协议核心能力不变

> 一句话总结：**轻量级接入，重量级安全。** 品牌方只需要告诉我们"哪些资产对应哪个 SKU"，剩下的交给我们。
