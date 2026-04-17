# Task: Brand / Platform Attestation 流程落地

> **关联 Specs**: [spec-brand-attestation.md](../specs/spec-brand-attestation.md), [spec-platform-attestation.md](../specs/spec-platform-attestation.md)  
> **依赖任务**: [task-asset-commitment-migration.md](./task-asset-commitment-migration.md)  
> **状态**: ✅ 已完成  
> **优先级**: P0  
> **预计工期**: 4-6 天

---

## 任务目标

将品牌承诺与平台承诺落地为可生成、可存储、可校验、可被验真流程引用的协议对象。

本任务目标是完成第一阶段“双边共同信任根”的最小工程实现，而不是一步到位完成 MPC / HSM / 联合派生。

---

## 前置依赖

在开始本任务前，必须已完成：

- [x] `AssetCommitment` 已稳定生成
- [x] 激活链路可返回 `asset_commitment_id`
- [x] 数据库中可通过 `asset_commitment_id` 定位协议对象

---

## Phase 1: 数据模型与 Migration（0.5 ~ 1 天）

### Task 1.1: 新增 `brand_attestations` 表
- [x] 新增 migration 创建 `brand_attestations`
- [x] 字段至少包括：
  - [x] `attestation_id`
  - [x] `version`
  - [x] `brand_id`
  - [x] `asset_commitment_id`
  - [x] `statement`
  - [x] `key_id`
  - [x] `canonical_payload`
  - [x] `signature`
  - [x] `issued_at`
  - [x] `created_at`
- [x] 增加唯一约束：同一 `asset_commitment_id + statement` 只能有一条当前有效记录

### Task 1.2: 新增 `platform_attestations` 表
- [x] 新增 migration 创建 `platform_attestations`
- [x] 字段至少包括：
  - [x] `attestation_id`
  - [x] `version`
  - [x] `platform_id`
  - [x] `asset_commitment_id`
  - [x] `statement`
  - [x] `key_id`
  - [x] `canonical_payload`
  - [x] `signature`
  - [x] `issued_at`
  - [x] `created_at`
- [x] 增加唯一约束：同一 `asset_commitment_id + statement` 只能有一条当前有效记录

**验收标准**:
- [x] 两张表 migration 成功
- [x] 索引与唯一约束生效
- [x] 与 `asset_commitments` 可稳定关联

---

## Phase 2: 签名与校验能力（1 ~ 1.5 天）

### Task 2.1: 定义承诺对象结构体
- [x] 新增 `BrandAttestationPayloadV1`
- [x] 新增 `PlatformAttestationPayloadV1`
- [x] 新增 record / dto 结构体

### Task 2.2: 实现 canonical payload 生成
- [x] 规范化品牌承诺 payload
- [x] 规范化平台承诺 payload
- [x] 保证签名前字节流稳定

### Task 2.3: 实现签名与验签抽象
- [x] 新增品牌签名接口抽象
- [x] 新增平台签名接口抽象
- [x] 新增承诺验签函数

**建议**:
- 第一阶段优先支持 Ed25519 或现有易集成签名方案
- 不要在这一阶段引入过重抽象

**验收标准**:
- [x] 相同 payload 可稳定签名与验签
- [x] 错误签名会被拒绝
- [x] `key_id` 可参与密钥轮换

---

## Phase 3: DB 访问层（0.5 ~ 1 天）

### Task 3.1: 实现 Brand Attestation DB 访问函数
- [x] `insert_brand_attestation()`
- [x] `fetch_brand_attestation_by_commitment()`
- [x] `fetch_brand_attestation_by_id()`

### Task 3.2: 实现 Platform Attestation DB 访问函数
- [x] `insert_platform_attestation()`
- [x] `fetch_platform_attestation_by_commitment()`
- [x] `fetch_platform_attestation_by_id()`

**验收标准**:
- [x] 插入、查询、唯一冲突处理正确
- [x] 错误语义清晰

---

## Phase 4: 激活链路升级（1 ~ 1.5 天）

### Task 4.1: Brand Attestation 接入激活流程
- [x] 激活成功后拿到 `asset_commitment_id`
- [x] 生成品牌承诺 payload
- [x] 调用品牌签名或提交签名结果
- [x] 写入 `brand_attestations`

### Task 4.2: Platform Attestation 接入激活流程
- [x] 在品牌承诺准备完成后生成平台承诺
- [x] 写入 `platform_attestations`

### Task 4.3: 激活响应扩展
- [x] 返回：
  - [x] `asset_commitment_id`
  - [x] `brand_attestation_status`
  - [x] `platform_attestation_status`

**第一阶段建议**:
- 允许承诺状态先以 `issued / pending / failed` 暴露
- 不要求一开始就把激活全链路做成强同步硬失败

**验收标准**:
- [x] 激活成功后可看到承诺状态
- [x] 承诺失败不会导致脏数据不可追踪

---

## Phase 5: 管理与调试接口（0.5 天）

### Task 5.1: 新增内部查询接口或调试返回
- [x] 可按 `asset_commitment_id` 查询品牌承诺
- [x] 可按 `asset_commitment_id` 查询平台承诺

### Task 5.2: 资产详情接口扩展
- [x] 可返回承诺状态摘要

**验收标准**:
- [x] 研发联调可直接看到承诺状态
- [x] 前端暂不强依赖完整承诺详情

---

## Phase 6: 测试与验证（0.5 ~ 1 天）

### Task 6.1: 单元测试
- [x] 品牌承诺 payload 规范化测试
- [x] 平台承诺 payload 规范化测试
- [x] 签名 / 验签测试

### Task 6.2: 集成测试
- [x] 激活后生成 `brand_attestation`
- [x] 激活后生成 `platform_attestation`
- [x] 篡改 payload 或签名后验签失败
- [x] 唯一约束正确生效

### Task 6.3: 联调脚本
- [x] 激活脚本打印：
  - [x] `asset_commitment_id`
  - [x] `brand_attestation_status`
  - [x] `platform_attestation_status`

**验收标准**:
- [x] 单元测试通过
- [x] 集成测试通过
- [x] 联调脚本能展示结果

---

## 涉及文件建议

### Migration
- `rust/rc-api/migrations/*_create_brand_attestations.sql`
- `rust/rc-api/migrations/*_create_platform_attestations.sql`

### Rust
- `rust/rc-api/src/db/brand_attestations.rs`
- `rust/rc-api/src/db/platform_attestations.rs`
- `rust/rc-api/src/routes/protocol.rs`
- `rust/rc-api/src/routes/assets.rs`
- `rust/rc-api/tests/activation_integration.rs`

---

## 关键验收标准（DoD）

- [x] 激活后可稳定生成品牌承诺与平台承诺
- [x] 承诺可独立校验真伪
- [x] 承诺可通过 `asset_commitment_id` 稳定引用
- [x] 后续 `Verification V2` 可直接消费承诺状态

---

## 非目标

本任务当前不包含：

- 品牌侧 HSM 强制接入
- 联合派生 / MPC
- 多平台互认
- 前端完整承诺展示 UX
