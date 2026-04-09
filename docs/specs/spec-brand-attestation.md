# Spec：Brand Attestation

> 文档类型：Spec  
> 状态：Draft  
> 权威级别：Implementation Proposal  
> 最后更新：2026-04-09

---

## 1. 目标

本 Spec 定义 `Brand Attestation`，用于让品牌方从“业务参与方”升级为“协议共同信任根之一”的第一阶段落地点。

它解决的问题是：

- 当前品牌方可以调用 API，但还没有对协议对象给出不可抵赖承诺
- 当前“品牌参与”容易被误写成“品牌已经共同持密”
- 当前验真仍然没有品牌侧协议承诺作为必要条件之一

---

## 2. 定义

`Brand Attestation` 是品牌方针对某个 `AssetCommitment` 给出的协议级承诺证明。

它不是：

- 一次普通 API 调用
- 一条后台操作日志
- 品牌系统里的审批单号本身

它是：

> 品牌方对“该协议对象属于本品牌合法发行范围”的不可抵赖声明。

---

## 3. 设计目标

`Brand Attestation` 必须满足：

1. 指向明确的 `AssetCommitment`
2. 来源可验证，能确认确实由品牌方给出
3. 不可被平台伪装成品牌方单独生成
4. 可被验真流程引用
5. 可与未来 `Platform Attestation` 组合形成双边共同信任根

---

## 4. 第一阶段实现策略

考虑当前工程基线，第一阶段建议采用：

> **品牌签名承诺**

即：

- 品牌方持有独立签名私钥
- 品牌方对 `AssetCommitment` 做签名
- 平台保存品牌公钥或可验证引用
- 激活完成后保存该签名结果

第一阶段不强制品牌方自建 HSM，也不要求立刻上 MPC。

---

## 5. 数据结构定义

### 5.1 Payload 建议

```json
{
  "version": "ba_v1",
  "brand_id": "brand-001",
  "asset_commitment_id": "<sha256>",
  "statement": "brand_issues_asset",
  "issued_at": "2026-04-09T12:00:00Z",
  "key_id": "brand-key-2026-01"
}
```

### 5.2 字段说明

- `version`：承诺结构版本
- `brand_id`：品牌标识
- `asset_commitment_id`：被承诺的协议对象 ID
- `statement`：承诺语义，第一版固定为 `brand_issues_asset`
- `issued_at`：签发时间
- `key_id`：品牌签名密钥标识

### 5.3 签名对象

建议对规范化 JSON Payload 做签名：

```text
brand_signature = Sign(brand_private_key, canonical_payload_bytes)
```

---

## 6. 数据库存储建议

建议新增表：`brand_attestations`

字段建议：

- `attestation_id text primary key`
- `version text not null`
- `brand_id text not null`
- `asset_commitment_id text not null`
- `statement text not null`
- `key_id text not null`
- `canonical_payload jsonb not null`
- `signature text not null`
- `issued_at timestamptz not null`
- `created_at timestamptz not null default now()`

索引建议：

- `uniq_brand_attestation_commitment_statement`
- `idx_brand_attestations_brand_id`

约束建议：

- 同一 `asset_commitment_id + statement` 只允许一条当前有效品牌承诺

---

## 7. 品牌密钥管理建议

### 7.1 第一阶段可接受方案

- 平台保存品牌公钥
- 品牌私钥由品牌方保存
- 品牌通过受控接口提交签名结果

### 7.2 过渡方案

若短期内品牌方无法自持签名能力，可采用过渡方案：

- 平台代品牌托管品牌签名密钥
- 但必须在文档与系统中明确标注这是过渡形态
- 不能把该过渡形态表述成“品牌已经成为完全独立共同信任根”

### 7.3 长期目标

- 品牌私钥由品牌方或品牌侧 HSM 控制
- 平台无法伪造品牌签名

---

## 8. 生成时机

建议在**激活成功后**生成品牌承诺。

原因：

- 此时 `AssetCommitment` 已稳定生成
- 品牌发行与映射语义已完整
- 可与当前激活流程自然衔接

---

## 9. 与当前流程的衔接

### 9.1 当前激活流程升级建议

现状：

- 激活成功 = 平台状态推进 + 写库成功

升级后：

- 激活成功 = 平台状态推进成功
- 品牌承诺状态 = 已生成 / 待生成 / 失败

第一阶段建议先允许：

- 业务激活完成
- 品牌承诺异步补齐

但最终验真 V2 不应在缺少品牌承诺时给出终局真品结论。

### 9.2 当前 API 返回建议

激活接口未来可返回：

```json
{
  "asset_id": "asset_001",
  "asset_commitment_id": "...",
  "brand_attestation_status": "issued"
}
```

---

## 10. 校验规则

校验 `Brand Attestation` 时必须验证：

1. `brand_id` 与目标资产一致
2. `asset_commitment_id` 存在且匹配
3. `statement` 合法
4. `key_id` 对应品牌有效公钥
5. `signature` 校验通过
6. 承诺未被撤销或替代

---

## 11. 错误语义建议

建议统一错误码：

- `BRAND_ATTESTATION_MISSING`
- `BRAND_ATTESTATION_INVALID`
- `BRAND_ATTESTATION_KEY_UNKNOWN`
- `BRAND_ATTESTATION_REVOKED`

---

## 12. 验收标准

本 Spec 最小验收标准：

1. 对已生成的 `AssetCommitment` 可成功生成品牌承诺
2. 系统可校验品牌承诺真伪
3. 激活记录可关联到 `brand_attestation`
4. 验真 V2 可引用品牌承诺状态

---

## 13. 非目标

本 Spec 当前不解决：

- 平台共同承诺
- 双签名终局定义
- 联合派生 / MPC
- 品牌侧硬件安全模块标准化

---

## 14. 关联 Specs

- `spec-asset-commitment.md`
- `spec-platform-attestation.md`
- `spec-verification-v2.md`
