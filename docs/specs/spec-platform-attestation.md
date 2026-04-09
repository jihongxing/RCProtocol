# Spec：Platform Attestation

> 文档类型：Spec  
> 状态：Draft  
> 权威级别：Implementation Proposal  
> 最后更新：2026-04-09

---

## 1. 目标

本 Spec 定义 `Platform Attestation`，用于让平台方从“当前平台承载者”升级为“终局协议中的共同信任根之一”。

它解决的问题是：

- 当前平台数据库记录与平台 KMS 能力过于接近“单方裁决”
- 即便品牌承诺存在，如果平台没有对同一协议对象给出共同承诺，仍不足以形成终局真品定义
- 需要一个可与 `Brand Attestation` 组合的第二承诺层

---

## 2. 定义

`Platform Attestation` 是平台方针对某个 `AssetCommitment` 给出的协议级共同承诺证明。

它不是：

- 数据库里存在一条资产记录
- 平台后台中显示“已激活”
- 某次接口调用成功

它是：

> 平台方对“该协议对象已被本协议接纳为可验证对象”的不可抵赖声明。

---

## 3. 设计目标

`Platform Attestation` 必须满足：

1. 指向明确的 `AssetCommitment`
2. 与 `Brand Attestation` 指向同一对象
3. 可独立验证来源确实为平台方
4. 可被验真流程引用
5. 未来可与更高等级安全实现兼容

---

## 4. 第一阶段实现策略

第一阶段建议采用：

> **平台签名承诺**

即：

- 平台持有独立签名私钥
- 平台对 `AssetCommitment` 做签名
- 验真时校验平台签名是否有效

该方案适合作为双边共同信任根的第一版落地点。

---

## 5. 数据结构定义

### 5.1 Payload 建议

```json
{
  "version": "pa_v1",
  "platform_id": "rcprotocol-main",
  "asset_commitment_id": "<sha256>",
  "statement": "platform_accepts_asset",
  "issued_at": "2026-04-09T12:00:00Z",
  "key_id": "platform-key-2026-01"
}
```

### 5.2 字段说明

- `version`：承诺结构版本
- `platform_id`：平台标识
- `asset_commitment_id`：被承诺的协议对象 ID
- `statement`：承诺语义，第一版固定为 `platform_accepts_asset`
- `issued_at`：签发时间
- `key_id`：平台签名密钥标识

### 5.3 签名对象

建议对规范化 JSON Payload 做签名：

```text
platform_signature = Sign(platform_private_key, canonical_payload_bytes)
```

---

## 6. 数据库存储建议

建议新增表：`platform_attestations`

字段建议：

- `attestation_id text primary key`
- `version text not null`
- `platform_id text not null`
- `asset_commitment_id text not null`
- `statement text not null`
- `key_id text not null`
- `canonical_payload jsonb not null`
- `signature text not null`
- `issued_at timestamptz not null`
- `created_at timestamptz not null default now()`

索引建议：

- `uniq_platform_attestation_commitment_statement`
- `idx_platform_attestations_commitment_id`

---

## 7. 平台密钥管理建议

### 7.1 第一阶段

- 平台签名私钥由平台受控保存
- 平台签名公钥对内部与验真组件可见
- 所有平台承诺使用可轮换的 `key_id`

### 7.2 长期方向

后续可升级为：

- 云 KMS
- HSM
- 与品牌侧联合派生 / MPC 协作

但第一阶段不强求。

---

## 8. 生成时机

建议在以下条件全部满足后生成：

1. `AssetCommitment` 已生成
2. 当前激活流程已成功
3. `Brand Attestation` 已存在，或至少品牌承诺流程已进入可确认状态

第一阶段建议采用顺序：

1. 生成 `AssetCommitment`
2. 生成 `Brand Attestation`
3. 生成 `Platform Attestation`

---

## 9. 校验规则

校验 `Platform Attestation` 时必须验证：

1. `asset_commitment_id` 存在且匹配
2. `statement` 合法
3. `platform_id` 合法
4. `key_id` 对应平台有效公钥
5. `signature` 校验通过
6. 承诺未被撤销或替代

---

## 10. 与当前流程的衔接

### 10.1 当前激活链路升级建议

现状：

- 平台在激活后写入状态与资产记录

升级后：

- 平台在激活后还应生成 `Platform Attestation`

### 10.2 当前 API 返回建议

未来激活接口可返回：

```json
{
  "asset_id": "asset_001",
  "asset_commitment_id": "...",
  "brand_attestation_status": "issued",
  "platform_attestation_status": "issued"
}
```

---

## 11. 错误语义建议

建议统一错误码：

- `PLATFORM_ATTESTATION_MISSING`
- `PLATFORM_ATTESTATION_INVALID`
- `PLATFORM_ATTESTATION_KEY_UNKNOWN`
- `PLATFORM_ATTESTATION_REVOKED`

---

## 12. 验收标准

本 Spec 最小验收标准：

1. 对已生成的 `AssetCommitment` 可稳定生成平台承诺
2. 系统可校验平台承诺真伪
3. 激活记录可关联到 `platform_attestation`
4. 验真 V2 可引用平台承诺状态

---

## 13. 非目标

本 Spec 当前不解决：

- Brand Attestation 的具体品牌密钥托管模式
- 联合派生 / MPC
- 去中心化存储
- 跨平台互认

---

## 14. 关联 Specs

- `spec-asset-commitment.md`
- `spec-brand-attestation.md`
- `spec-verification-v2.md`
