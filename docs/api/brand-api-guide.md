# Brand API Integration Guide

> Version: 1.1  
> Last Updated: 2026-04-08  
> Audience: Brand technical teams integrating with RCProtocol

---

## Overview

RCProtocol provides RESTful APIs for brands to integrate asset lifecycle management into their existing systems. This guide now includes the brand onboarding entry points needed for MVP:

1. 品牌注册
2. API Key 轮换
3. 品牌详情查询
4. API Key 列表查询
5. 资产主链路接口（盲扫 / 激活 / 查询）

---

## Authentication

管理接口支持两种认证方式：

- `Authorization: Bearer <platform_jwt>`：平台侧管理操作
- `X-Api-Key: <brand_api_key>`：品牌自身操作

说明：
- 品牌注册只能由 Platform 角色发起
- 注册成功或轮换成功时，明文 API Key 仅返回一次
- 后续品牌侧调用可使用 `X-Api-Key`

---

## Brand Onboarding APIs

### 1. Register Brand

**Endpoint:** `POST /brands`

**Auth:** `Authorization: Bearer <platform_jwt>`

**Request:**

```json
{
  "brand_name": "Luxury Watch Co.",
  "contact_email": "api@luxurywatch.com",
  "industry": "Watches"
}
```

**Response:**

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

**Validation rules:**
- `contact_email` 全局唯一
- `industry` 必须为 `Watches/Fashion/Wine/Jewelry/Art/Other`
- `brand_id` 使用 `brand_<ULID>`
- `api_key` 使用 `rcpk_live_<32 hex>`

---

### 2. Rotate API Key

**Endpoint:** `POST /brands/:brand_id/api-keys/rotate`

**Auth:**
- `Authorization: Bearer <platform_jwt>` 或
- `X-Api-Key: <brand_api_key>`

**Request:**

```json
{
  "reason": "scheduled rotation"
}
```

**Response:**

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

**Expected behavior:**
- 旧 Key 立即变为 `Revoked`
- 新 Key 立即生效
- 明文新 Key 只返回一次

---

### 3. Get Brand Detail

**Endpoint:** `GET /brands/:brand_id`

**Auth:** `Authorization: Bearer <platform_jwt>` or `X-Api-Key: <brand_api_key>`

**Response:**

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

**Security note:** 此接口不会返回 `api_key` 或 `key_hash`。

---

### 4. List API Keys

**Endpoint:** `GET /brands/:brand_id/api-keys`

**Auth:** `Authorization: Bearer <platform_jwt>` or `X-Api-Key: <brand_api_key>`

**Response:**

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

**Behavior:**
- 仅返回 `key_prefix`
- 包含 Active + Revoked
- 按创建时间倒序

---

## Core Asset APIs

### 5. Blind Scan (Factory Logging)

Register assets during manufacturing without assigning product details.

**Endpoint:** `POST /api/v1/assets/blind-scan`

**Request:**
```json
{
  "uid": "04F1A2B3C4D5E6F8",
  "brand_id": "brand_luxury_watch",
  "batch_id": "55555555-5555-5555-5555-555555555555",
  "metadata": {
    "factory_line": "A1",
    "operator": "operator_001",
    "production_date": "2026-04-08"
  }
}
```

**Response:**
```json
{
  "asset_id": "bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbbbbb",
  "uid": "04F1A2B3C4D5E6F8",
  "brand_id": "brand_luxury_watch",
  "current_state": "FactoryLogged",
  "created_at": "2026-04-08T10:30:00Z"
}
```

**States:** `PreMinted` → `FactoryLogged`

---

### 6. Activate Asset

Activate an asset and bind it to your SKU system.

**Endpoint:** `POST /api/v1/assets/activate`

**Request:**
```json
{
  "asset_id": "bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbbbbb",
  "brand_id": "brand_luxury_watch",
  "external_product_id": "SKU-LW-CHRONO-2026",
  "external_product_name": "Chronograph Master Edition 2026",
  "external_product_url": "https://luxurywatch.example.com/products/chrono-master-2026",
  "authority_type": "VIRTUAL_APP",
  "metadata": {
    "activation_reason": "approved_by_brand",
    "activated_by": "brand_admin_001"
  }
}
```

**Response:**
```json
{
  "asset_id": "bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbbbbb",
  "action": "ActivateConfirm",
  "from_state": "Unassigned",
  "to_state": "Activated",
  "external_product_id": "SKU-LW-CHRONO-2026",
  "virtual_mother_card": {
    "device_id": "10000000-0000-0000-0000-000000000001",
    "authority_type": "VIRTUAL_APP",
    "status": "Active"
  },
  "audit_event_id": "30000000-0000-0000-0000-000000000003"
}
```

**States:** `Unassigned` → `RotatingKeys` → `EntangledPending` → `Activated`

---

### 7. Query Asset

Retrieve asset details and history.

**Endpoint:** `GET /api/v1/assets/{asset_id}`

**Response:**
```json
{
  "asset_id": "bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbbbbb",
  "uid": "04F1A2B3C4D5E6F8",
  "brand_id": "brand_luxury_watch",
  "external_product_id": "SKU-LW-CHRONO-2026",
  "external_product_name": "Chronograph Master Edition 2026",
  "external_product_url": "https://luxurywatch.example.com/products/chrono-master-2026",
  "current_state": "Activated",
  "previous_state": "EntangledPending",
  "owner_id": null,
  "activated_at": "2026-04-08T11:00:00Z",
  "created_at": "2026-04-08T10:30:00Z",
  "updated_at": "2026-04-08T11:00:00Z"
}
```

---

## Recommended Validation

Run the end-to-end onboarding script:

```bash
./scripts/test-brand-registration.sh
```

It validates:
- brand registration
- duplicate email rejection
- API key rotation
- old key invalidation
- new key usability
- key list query
- permission checks
