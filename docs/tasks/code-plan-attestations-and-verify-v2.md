# Code Plan: Attestations + Verification V2 落地到当前 Rust 代码结构

> **关联任务**: [task-attestation-flows.md](./task-attestation-flows.md), [task-verification-v2-implementation.md](./task-verification-v2-implementation.md)  
> **关联 Specs**: [../specs/spec-brand-attestation.md](../specs/spec-brand-attestation.md), [../specs/spec-platform-attestation.md](../specs/spec-platform-attestation.md), [../specs/spec-verification-v2.md](../specs/spec-verification-v2.md)  
> **状态**: Draft  
> **用途**: 代码级实施计划

---

## 1. 目标

本计划把以下两类能力映射到当前 Rust 代码结构：

1. `Brand Attestation` / `Platform Attestation`
2. `/verify/v2`

目标不是一次性重构整个验证系统，而是在当前代码基线上给出**最小可落地、路径清晰、不会打断 V1** 的实施方案。

---

## 2. 当前代码基线观察

### 2.1 激活路径仍是唯一合理接入点

当前品牌相关协议动作接入点仍然是：

- `rust/rc-api/src/routes/protocol.rs`
  - `activate_asset()`

原因：

- 品牌承诺与平台承诺都依赖 `AssetCommitment`
- `AssetCommitment` 生成时机就在激活后
- 现有代码没有独立 activation service 层，所以第一阶段只能从 `activate_asset()` 接入

### 2.2 当前没有签名抽象层

当前 `rc-api` 内并没有独立的 attest / sign / verify 模块。

因此第一阶段建议新增一个非常轻量的模块，而不是过度设计：

- `rust/rc-api/src/attestation/`
  - `mod.rs`
  - `brand.rs`
  - `platform.rs`

如果你想更轻，也可以先用：

- `rust/rc-api/src/domain/brand_attestation.rs`
- `rust/rc-api/src/domain/platform_attestation.rs`

但从后续 `/verify/v2` 复用角度，独立 `attestation/` 更清楚。

### 2.3 当前 `/verify` 已经有一条稳定主路径

当前路径在：

- `rust/rc-api/src/routes/verify.rs`

结构很清晰：

- `verify_handler()`
- `handle_degraded()`
- `handle_full_verify()`

这对 V2 很有利，因为可以：

- 保留 `/verify`
- 新增 `/verify/v2`
- 复用 `parse_sun_params()` 和标签动态认证逻辑

---

## 3. 代码文件级实施方案

## 3.1 Migration 方案

建议新增 3 个 migration。

### A. `create_brand_attestations.sql`

建议命名：

```text
rust/rc-api/migrations/20250101000027_create_brand_attestations.sql
```

### B. `create_platform_attestations.sql`

建议命名：

```text
rust/rc-api/migrations/20250101000028_create_platform_attestations.sql
```

### C. `extend_verification_events_for_v2.sql`

建议命名：

```text
rust/rc-api/migrations/20250101000029_extend_verification_events_for_v2.sql
```

建议新增字段：

- `asset_commitment_id text null`
- `verification_version text null`
- `brand_attestation_status text null`
- `platform_attestation_status text null`

理由：

- 当前 `verification_events` 只够记录 V1
- V2 若没有独立字段，审计价值会很弱

---

## 3.2 新增 Rust 模块

### A. `rust/rc-api/src/db/brand_attestations.rs`

职责：

- `insert_brand_attestation()`
- `fetch_brand_attestation_by_commitment()`
- `fetch_brand_attestation_by_id()`

### B. `rust/rc-api/src/db/platform_attestations.rs`

职责：

- `insert_platform_attestation()`
- `fetch_platform_attestation_by_commitment()`
- `fetch_platform_attestation_by_id()`

### C. `rust/rc-api/src/attestation/brand.rs`

职责：

- `BrandAttestationPayloadV1`
- `build_brand_attestation_payload()`
- `sign_brand_attestation()`
- `verify_brand_attestation()`

### D. `rust/rc-api/src/attestation/platform.rs`

职责：

- `PlatformAttestationPayloadV1`
- `build_platform_attestation_payload()`
- `sign_platform_attestation()`
- `verify_platform_attestation()`

### E. `rust/rc-api/src/attestation/mod.rs`

统一导出。

### F. 模块注册

需要修改：

- `rust/rc-api/src/lib.rs`
- `rust/rc-api/src/db/mod.rs`

---

## 4. Attestation 的最小工程实现建议

## 4.1 第一阶段签名方案

当前建议优先采用：

- `ed25519-dalek` 或当前仓库中最容易接入的签名库

理由：

- 轻量
- 易于本地测试
- 不需要立刻引入 HSM / KMS 签名扩展

### 第一阶段务实策略

- 品牌承诺：先支持“测试品牌签名器 / 平台托管品牌签名器”
- 平台承诺：先支持平台本地签名器

并且必须在文档和代码命名里明确：

- 这是 **Phase 1 implementation**
- 不是最终共同信任根强安全形态

---

## 4.2 AppState 是否需要扩展

当前 `AppState` 已包含：

- `db`
- `kms`
- `jwt_decoder`
- `api_key_secret`
- `ctr_cache`

后续如果要把 attestation 直接接进 handler，建议给 `AppState` 新增：

- `platform_attestation_signer`
- `brand_attestation_signer`（可选，若第一阶段平台托管品牌签名）

如果你不想在第一步改 `AppState` 太多，也可以先：

- 用环境变量读取测试私钥
- 在 attestation 模块内部构造 signer

但从长期维护看，挂在 `AppState` 更合理。

---

## 5. `activate_asset()` 的承诺接入点

文件：`rust/rc-api/src/routes/protocol.rs`

前提：

- 已完成 `AssetCommitment` 落地
- `activate_asset()` 已能拿到 `asset_commitment_id`

### 推荐接入顺序

在 `activate_asset()` 内按以下顺序：

1. `update_asset_product_mapping()`
2. 生成 / 写入 `AssetCommitment`
3. 生成 `Brand Attestation`
4. 生成 `Platform Attestation`
5. 生成虚拟母卡
6. `persist_action()`
7. 返回 `ActivateResponse`

### 为什么 Brand 在前、Platform 在后

因为协议语义上更合理：

- 品牌先声明“这是我发行的对象”
- 平台再声明“我接受该对象进入协议”

### `ActivateResponse` 建议再扩展

在已有 `asset_commitment_id` 基础上新增：

- `brand_attestation_status: String`
- `platform_attestation_status: String`

第一阶段状态值建议固定为：

- `issued`
- `pending`
- `failed`

---

## 6. 查询路径改造建议

## 6.1 `AssetDetail` 增加承诺状态摘要

文件：`rust/rc-api/src/db/assets.rs`

建议给 `AssetDetail` 增加：

- `asset_commitment_id: Option<String>`
- `brand_attestation_status: Option<String>`
- `platform_attestation_status: Option<String>`

### 实现方式

第一阶段不一定需要复杂 join。

可选两种方案：

#### 方案 A：字段回填型

在 `assets` 表增加：

- `brand_attestation_status`
- `platform_attestation_status`

优点：

- 查询简单

缺点：

- 有状态复制

#### 方案 B：读时聚合型

在 `fetch_asset_detail()` 时额外查两张 attestation 表

优点：

- 真源更干净

缺点：

- 查询代码更复杂

建议：

> 第一阶段选方案 B，避免复制状态。

---

## 7. `/verify/v2` 路由级改造方案

文件：`rust/rc-api/src/routes/verify.rs`

## 7.1 不动 `/verify`

保留：

- `verify_handler()` 对应现有 `/verify`

新增：

- `verify_v2_handler()` 对应 `/verify/v2`

并在 router 中注册。

## 7.2 复用现有 helper

以下逻辑建议直接复用：

- `parse_sun_params()`
- `is_degraded_mode()`
- 现有 KMS + CMAC 校验逻辑
- 现有 `CtrCache` / replay 检查逻辑

不要复制一套标签认证代码。

## 7.3 建议拆分 helper

当前 `handle_full_verify()` 把很多逻辑揉在一起。

为了 V2 可读性，建议抽出：

- `authenticate_tag()`
- `resolve_asset_commitment()`
- `resolve_attestation_status()`
- `evaluate_v2_status()`

第一阶段不需要大规模重构 V1，只需要：

- 把共用的标签认证部分抽成 helper
- V1 / V2 共用它

---

## 8. `/verify/v2` 处理流程在代码中的落点

### Step 1: 解析参数

沿用：

- `VerifyParams`
- `parse_sun_params()`

### Step 2: 标签动态认证

从当前 `handle_full_verify()` 中抽出：

- `uid -> asset` 定位
- `derive_chip_key()`
- `CMAC` 校验
- `CTR` 检查

### Step 3: 定位 `asset_commitment_id`

建议数据来源优先级：

1. `assets.asset_commitment_id`
2. 若为空，再尝试 `uid + epoch` 反查 `asset_commitments`

### Step 4: 查 Brand / Platform Attestation

通过：

- `db::brand_attestations::fetch_brand_attestation_by_commitment()`
- `db::platform_attestations::fetch_platform_attestation_by_commitment()`

并调用对应验签函数。

### Step 5: 读取状态与风险

继续复用当前：

- `current_state`
- `evaluate_status()` 的思想

但建议新增 V2 专用结论函数。

### Step 6: 写 V2 审计

扩展：

- `db/verification.rs`

建议新增函数，而不是污染旧接口：

```rust
pub async fn insert_verification_event_v2(...)
```

理由：

- 当前 V1 的 `insert_verification_event()` 参数过于紧凑
- V2 需要更多字段

---

## 9. 推荐新增数据结构

### `routes/verify.rs`

建议新增：

```rust
pub struct VerificationV2Response {
    pub verification_version: String,
    pub tag_authentication: String,
    pub attestation_status: VerificationAttestationStatus,
    pub protocol_state: VerificationProtocolState,
    pub verification_status: String,
}
```

```rust
pub struct VerificationAttestationStatus {
    pub asset_commitment_id: Option<String>,
    pub brand_attestation: String,
    pub platform_attestation: String,
}
```

```rust
pub struct VerificationProtocolState {
    pub current_state: Option<String>,
    pub risk_flags: Vec<String>,
}
```

---

## 10. 测试文件级实施方案

## 10.1 Activation Integration Test 扩展

文件：`rust/rc-api/tests/activation_integration.rs`

新增断言：

- 激活后已有 `brand_attestations`
- 激活后已有 `platform_attestations`
- `asset_commitment_id` 可串起三张表

## 10.2 Verify Integration Test 扩展

文件：`rust/rc-api/tests/verify_integration.rs`

建议新增 4 类测试：

1. 标签通过 + 双承诺有效 -> `authentic`
2. 标签通过 + 缺品牌承诺 -> `incomplete_attestation`
3. 标签通过 + 缺平台承诺 -> `incomplete_attestation`
4. 标签通过 + 限制态 -> `restricted`

## 10.3 Attestation 单元测试

建议新增：

- `attestation/brand.rs` 单元测试
- `attestation/platform.rs` 单元测试

测试：

- payload 规范化稳定
- 签名验签正确
- 篡改后验签失败

---

## 11. 提交顺序建议

### Commit 1
- attestation migrations
- `db/brand_attestations.rs`
- `db/platform_attestations.rs`
- attestation payload + sign/verify 模块

### Commit 2
- `activate_asset()` 接入 brand/platform attestation
- `ActivateResponse` 扩展
- 资产详情增加 attestation 摘要

### Commit 3
- `verification_events` 扩展
- `/verify/v2` 新路由与响应结构
- `insert_verification_event_v2()`

### Commit 4
- activation / verify integration tests
- attestation unit tests
- 联调脚本

---

## 12. 最小可落地版本（建议）

若要控制风险，建议分两波：

### 波次 1
- 落 Brand / Platform Attestation 表
- 激活返回 attestation status
- 资产详情可见 attestation status
- 先不做 `/verify/v2`

### 波次 2
- 落 `/verify/v2`
- 扩展 verification audit
- 联调演示 V1 / V2 差异

这样更稳。

---

## 13. 当前不建议立即做的事

当前不建议立刻做：

- 把 V2 直接替换 V1
- 把 attestation 状态复制到过多表里
- 立刻接品牌侧真实 HSM
- 在同一阶段重构整条验证架构

原因：

> 先把双承诺对象落地，再加新验真入口，才不会把风险叠到一起。
