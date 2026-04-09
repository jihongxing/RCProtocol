# Spec：Asset Commitment

> 文档类型：Spec  
> 状态：Draft  
> 权威级别：Implementation Proposal  
> 最后更新：2026-04-09

---

## 1. 目标

本 Spec 定义 `AssetCommitment`，作为 RCProtocol 从“围绕数据库资产记录表达真相”演进到“围绕协议承诺对象表达真相”的第一步。

本 Spec 的目标不是立刻替换全部现有资产模型，而是提供一个**可并行落地、可逐步迁移、可指导代码改造**的统一协议对象。

---

## 2. 设计动机

当前实现中，以下问题同时存在：

- 验真第一步依赖 `assets` 表查询
- `asset_id` 是当前主键，但不是终局协议真相对象
- 品牌、平台、验真、审计、转移没有围绕同一承诺对象收敛

因此需要一个统一对象，使以下能力都能指向同一协议基础：

- 激活
- 验真
- 品牌承诺
- 平台承诺
- 审计
- 转移

这个统一对象就是 `AssetCommitment`。

---

## 3. 定义

`AssetCommitment` 是对某个协议资产的规范化、可哈希、可承诺、可验证的对象表达。

它不是：

- 数据库自增主键
- 单纯的 `asset_id`
- 前端展示对象
- ERP 商品对象

它是：

> 品牌承诺、平台承诺、验真流程与审计流程共同指向的协议对象。

---

## 4. 字段定义

### 4.1 最小字段集

`AssetCommitmentPayloadV1` 建议至少包含：

- `version`
- `brand_id`
- `asset_uid`
- `chip_binding`
- `epoch`
- `metadata_hash`

### 4.2 字段说明

#### `version`

- 类型：`string`
- 示例：`"ac_v1"`
- 含义：承诺对象版本号

#### `brand_id`

- 类型：`string`
- 含义：品牌标识
- 约束：必须与协议上下文中的品牌一致

#### `asset_uid`

- 类型：`string`
- 含义：标签 UID 或协议级唯一资产标识
- 约束：建议使用规范化大写十六进制字符串

#### `chip_binding`

- 类型：`string`
- 含义：标签绑定摘要
- 说明：用于避免只靠裸 UID 表达资产绑定关系

建议第一版定义为：

```text
chip_binding = H(uid || chip_profile || epoch)
```

其中：

- `uid`：标签 UID
- `chip_profile`：芯片类型或绑定配置摘要
- `epoch`：当前密钥轮换周期

#### `epoch`

- 类型：`u32`
- 含义：密钥轮换周期
- 约束：必须与当前激活时所用 epoch 一致

#### `metadata_hash`

- 类型：`string`
- 含义：外部商品映射、发行附加信息等元数据的摘要
- 说明：避免将大量业务字段直接变成协议真相对象

建议第一版可由下列字段规范化后哈希得到：

- `external_product_id`
- `external_product_name`（可选）
- `external_product_url`（可选）
- `batch_id`（可选）

---

## 5. 规范化规则

生成 `AssetCommitment` 前，必须先对 payload 进行规范化。

### 5.1 字符串规范化

- 去除前后空白
- `brand_id` 原样大小写保存，但必须全系统一致
- `asset_uid` 使用大写十六进制
- URL 字段使用原样字符串，不做隐式改写

### 5.2 JSON 规范化

建议使用稳定键顺序序列化：

```json
{
  "asset_uid": "04A31B2C3D4E5F",
  "brand_id": "brand-001",
  "chip_binding": "...",
  "epoch": 1,
  "metadata_hash": "...",
  "version": "ac_v1"
}
```

### 5.3 哈希算法

第一版建议：

```text
asset_commitment = SHA-256(canonical_json_bytes)
```

输出建议使用小写十六进制字符串。

---

## 6. 数据结构建议

### 6.1 Rust 结构体建议

```rust
pub struct AssetCommitmentPayloadV1 {
    pub version: String,
    pub brand_id: String,
    pub asset_uid: String,
    pub chip_binding: String,
    pub epoch: u32,
    pub metadata_hash: String,
}
```

```rust
pub struct AssetCommitmentRecord {
    pub commitment_id: String,
    pub payload_version: String,
    pub brand_id: String,
    pub asset_uid: String,
    pub chip_binding: String,
    pub epoch: i32,
    pub metadata_hash: String,
    pub canonical_payload: serde_json::Value,
    pub created_at: DateTime<Utc>,
}
```

### 6.2 数据库表建议

建议新增表：`asset_commitments`

字段建议：

- `commitment_id text primary key`
- `payload_version text not null`
- `brand_id text not null`
- `asset_uid text not null`
- `chip_binding text not null`
- `epoch int not null`
- `metadata_hash text not null`
- `canonical_payload jsonb not null`
- `created_at timestamptz not null default now()`

索引建议：

- `idx_asset_commitments_brand_uid_epoch`
- `idx_asset_commitments_asset_uid`

---

## 7. 与当前 `assets` 模型的关系

### 7.1 第一阶段关系

第一阶段不删除 `assets` 表。

关系建议为：

- `assets` 仍作为当前工程承载对象
- `asset_commitments` 作为新的协议对象表
- 激活时同时写入 `assets` 与 `asset_commitments`

### 7.2 映射建议

建议在 `assets` 表中新增：

- `asset_commitment_id text null`

用于建立当前工程对象与新协议对象之间的桥接关系。

---

## 8. 生成时机

### 8.1 盲扫阶段

不建议在盲扫阶段正式生成最终 `AssetCommitment`。

原因：

- 盲扫阶段通常尚未完成品牌认领与外部映射
- 承诺对象过早生成会导致后续频繁重算

### 8.2 激活阶段

建议在激活阶段生成 `AssetCommitment`。

因为此时通常已经具备：

- `brand_id`
- `uid`
- `epoch`
- `external_product_id`
- 授权绑定关系

### 8.3 重算规则

若影响 `AssetCommitment` 的核心字段变化，则必须显式生成新版本，而不是静默覆盖旧值。

---

## 9. 对其他流程的影响

### 9.1 对 Brand Attestation 的影响

品牌承诺必须指向 `AssetCommitment`，而不是直接指向 `asset_id`。

### 9.2 对 Platform Attestation 的影响

平台共同承诺必须指向同一个 `AssetCommitment`。

### 9.3 对 Verification V2 的影响

验真 V2 必须以 `AssetCommitment` 作为承诺校验对象。

### 9.4 对审计的影响

关键审计事件建议逐步补充：

- `asset_commitment_id`

使审计从“围绕资产行记录”逐步转向“围绕协议对象”。

---

## 10. 迁移策略

### 10.1 Phase A：并行写入

- 激活时生成 `AssetCommitment`
- 当前接口仍返回 `asset_id`
- 新接口开始暴露 `asset_commitment_id`

### 10.2 Phase B：读路径增强

- 查询接口可同时返回 `asset_id` 与 `asset_commitment_id`
- 审计与验真开始记录 `asset_commitment_id`

### 10.3 Phase C：协议优先

- Brand / Platform Attestation 全部围绕 `AssetCommitment`
- Verification V2 优先围绕 `AssetCommitment`

---

## 11. 验收标准

本 Spec 落地的最小验收标准：

1. 激活成功后可稳定生成 `asset_commitment_id`
2. 同一输入生成结果稳定一致
3. 不同关键字段变化会生成不同 `asset_commitment_id`
4. `assets` 与 `asset_commitments` 可稳定关联
5. 后续承诺与验真流程可引用该对象

---

## 12. 非目标

本 Spec 当前不解决：

- 双签名结构的具体签名算法
- 品牌密钥托管方式
- 平台共同信任根的最终密码学实现
- 数据库完全去中心化

---

## 13. 关联 Specs

- `spec-brand-attestation.md`
- `spec-platform-attestation.md`
- `spec-verification-v2.md`
