# Code Plan: AssetCommitment 落地到当前 Rust 代码结构

> **关联任务**: [task-asset-commitment-migration.md](./task-asset-commitment-migration.md)  
> **关联 Spec**: [../specs/spec-asset-commitment.md](../specs/spec-asset-commitment.md)  
> **状态**: Draft  
> **用途**: 代码级实施计划

---

## 1. 目标

本计划不是抽象任务拆解，而是把 `AssetCommitment` 明确落到**当前仓库已有文件、函数、表结构与测试路径**上。

当前目标是：

- 不推翻现有 `assets` 主路径
- 以最小改动接入 `AssetCommitment`
- 让后续 `Brand Attestation` / `Platform Attestation` / `/verify/v2` 有稳定依赖对象

---

## 2. 当前代码基线观察

基于当前仓库，`AssetCommitment` 首次落地会主要影响以下模块：

### 2.1 激活主路径

当前激活在：

- `rust/rc-api/src/routes/protocol.rs`
  - `activate_asset()`
  - `update_asset_product_mapping()`
  - `persist_action()`（通过 `db/mod.rs`）

当前流程特征：

1. `fetch_asset()` 取当前资产状态
2. `apply_action()` 完成状态机推进
3. `generate_virtual_mother_card_with_result()` 生成虚拟母卡
4. `update_asset_product_mapping()` 更新 `external_product_*`
5. `persist_action()` 写回状态变化与 `asset_state_events`

问题在于：

- `AssetCommitment` 生成时机应该就在激活流程里
- 但当前激活逻辑把“产品映射更新”和“状态持久化”分成了多个分散写入点

这意味着第一步应先把 `AssetCommitment` 接到 `activate_asset()`，而不是去改通用 `execute_asset_action()`。

### 2.2 资产读路径

当前资产查询在：

- `rust/rc-api/src/db/assets.rs`
  - `fetch_asset_by_uid()`
  - `fetch_asset_detail()`
  - `list_assets()`

这里是后续把 `asset_commitment_id` 暴露给详情接口的最佳入口。

### 2.3 验真路径

当前验真在：

- `rust/rc-api/src/routes/verify.rs`
  - `handle_degraded()`
  - `handle_full_verify()`
- `rust/rc-api/src/db/verification.rs`
  - `insert_verification_event()`

当前 `handle_full_verify()` 明确先按 `uid -> assets` 查资产，再走 KMS + CMAC。

这决定了第一阶段不应强行改写现有 `/verify`，而只需要给后续 V2 预留 commitment 桥接能力。

### 2.4 DB 聚合与写路径

当前共享 DB 入口在：

- `rust/rc-api/src/db/mod.rs`
  - `fetch_asset()`
  - `persist_action()`

`persist_action()` 是所有状态变更的共用写入口，因此：

- 不建议一开始在这里强塞 `AssetCommitment` 生成逻辑
- 建议先在 `activate_asset()` 中生成并写入 commitment，再让 `persist_action()` 在第二阶段支持写审计桥接字段

---

## 3. 代码文件级实施方案

## 3.1 Migration 文件

建议新增 2 个 migration：

### A. `*_create_asset_commitments.sql`

建议命名：

```text
rust/rc-api/migrations/20250101000024_create_asset_commitments.sql
```

建议内容：

- 创建 `asset_commitments`
- 建立 `commitment_id` 主键
- 建立 `brand_id + asset_uid + epoch` 索引
- 建立 `asset_uid` 索引

### B. `*_add_asset_commitment_id_to_assets.sql`

建议命名：

```text
rust/rc-api/migrations/20250101000025_add_asset_commitment_id_to_assets.sql
```

建议内容：

- `ALTER TABLE assets ADD COLUMN asset_commitment_id TEXT NULL`
- 追加索引 `idx_assets_asset_commitment_id`
- 可选外键到 `asset_commitments(commitment_id)`，如果现阶段迁移顺序稳定则加，否则先不加外键只加索引

### C. 审计桥接（可拆独立 migration）

如要在第一阶段补审计桥接，建议新增：

```text
rust/rc-api/migrations/20250101000026_add_asset_commitment_id_to_asset_state_events.sql
```

建议内容：

- `ALTER TABLE asset_state_events ADD COLUMN asset_commitment_id TEXT NULL`
- 建立 `idx_state_events_asset_commitment_id`

第一阶段这一步可做可不做；若为了降低风险，可先只做 `assets` 桥接。

---

## 3.2 新增 Rust 模块

建议新增：

### A. `rust/rc-api/src/db/asset_commitments.rs`

职责：

- `insert_asset_commitment()`
- `fetch_asset_commitment_by_id()`
- `fetch_asset_commitment_by_uid_epoch()`

### B. `rust/rc-api/src/domain/asset_commitment.rs`

职责：

- `AssetCommitmentPayloadV1`
- `AssetCommitmentRecord`
- `build_chip_binding()`
- `build_metadata_hash()`
- `build_asset_commitment_payload()`
- `compute_asset_commitment_id()`

如果不想新增 `domain/` 目录，也可先放在：

- `rust/rc-api/src/db/asset_commitments.rs` 中实现最小版本

但从后续承诺与 V2 验真复用角度，独立 `domain/asset_commitment.rs` 更清晰。

### C. 模块导出变更

需要修改：

- `rust/rc-api/src/db/mod.rs`
- `rust/rc-api/src/lib.rs`

把新模块导出。

---

## 4. 函数级修改计划

## 4.1 `activate_asset()` 是第一落点

文件：`rust/rc-api/src/routes/protocol.rs`

当前 `activate_asset()` 的关键顺序是：

1. `apply_action()`
2. `generate_virtual_mother_card_with_result()`
3. `update_asset_product_mapping()`
4. 构造 `ActivateResponse`
5. `persist_action()`

### 建议改造顺序

改成：

1. `apply_action()`
2. `update_asset_product_mapping()`
3. 读取生成 commitment 所需字段
4. 生成并写入 `AssetCommitment`
5. 回写 `assets.asset_commitment_id`
6. `generate_virtual_mother_card_with_result()`
7. 构造带 `asset_commitment_id` 的 `ActivateResponse`
8. `persist_action()`

### 为什么先更新 product mapping 再算 commitment

因为当前 `metadata_hash` 依赖：

- `external_product_id`
- `external_product_name`
- `external_product_url`

如果先算 commitment 再写 product mapping，会让 commitment 丢掉这部分信息，或被迫二次重算。

### 为什么 commitment 生成不放进 `persist_action()`

因为：

- `persist_action()` 目前服务多种 action
- `AssetCommitment` 当前只与激活路径强相关
- 先在 `activate_asset()` 局部落地，风险最低

---

## 4.2 `ActivateResponse` 需要扩展

文件：`rust/rc-api/src/routes/protocol.rs`

当前：

- `ActivateResponse` 只有 `asset_id / action / from_state / to_state / audit_event_id / virtual_mother_card`

建议增加：

- `asset_commitment_id: String`

这样激活联调完成后，后续任务就能直接拿它作为输入。

---

## 4.3 `AssetDetail` 需要扩展

文件：`rust/rc-api/src/db/assets.rs`

建议在：

- `AssetDetail`
- `fetch_asset_detail()`
- `list_assets()`

都加上：

- `asset_commitment_id: Option<String>`

这样：

- `/assets/:asset_id` 能直接返回桥接信息
- 后续调试不用查数据库

---

## 4.4 `fetch_asset_by_uid()` 暂时不改语义

文件：`rust/rc-api/src/db/assets.rs`

当前 `fetch_asset_by_uid()` 返回 `AssetVerifyRow`，用于 `/verify`。

第一阶段建议：

- 可以给它加 `asset_commitment_id: Option<String>` 字段
- 但不要修改现有 `/verify` 判定逻辑

理由：

- 给 V2 预留桥接数据即可
- 不要在同一阶段同时改数据模型和验真结论

---

## 4.5 审计增强的最低改动点

文件：`rust/rc-api/src/db/mod.rs`

当前 `persist_action()` 已负责写 `asset_state_events`。

第一阶段有两种策略：

### 策略 A：最小改动

- 不改 `persist_action()`
- commitment 只写到 `assets.asset_commitment_id`
- 先靠资产详情与直接查表调试

### 策略 B：一步多做一点

- 给 `persist_action()` 增加一个可选参数 `asset_commitment_id: Option<&str>`
- 激活时传入，其他动作传 `None`
- `asset_state_events` 记录 bridge 字段

建议：

> 代码第一步先用策略 A，等 commitment 稳定后再做策略 B。

---

## 5. 推荐新增函数签名

以下是建议的最小函数集合。

### `domain/asset_commitment.rs`

```rust
pub struct AssetCommitmentPayloadV1 {
    pub version: String,
    pub brand_id: String,
    pub asset_uid: String,
    pub chip_binding: String,
    pub epoch: u32,
    pub metadata_hash: String,
}

pub fn build_chip_binding(uid: &str, epoch: u32) -> String
pub fn build_metadata_hash(
    external_product_id: &str,
    external_product_name: Option<&str>,
    external_product_url: Option<&str>,
    batch_id: Option<&str>,
) -> String
pub fn compute_asset_commitment_id(payload: &AssetCommitmentPayloadV1) -> Result<String, RcError>
```

### `db/asset_commitments.rs`

```rust
pub async fn insert_asset_commitment(
    pool: &PgPool,
    record: &AssetCommitmentRecord,
) -> Result<(), RcError>

pub async fn fetch_asset_commitment_by_id(
    pool: &PgPool,
    commitment_id: &str,
) -> Result<AssetCommitmentRecord, RcError>

pub async fn bind_asset_commitment_to_asset(
    pool: &PgPool,
    asset_id: &str,
    commitment_id: &str,
) -> Result<(), RcError>
```

---

## 6. 事务边界建议

当前 `activate_asset()` 中已经存在多次分散 DB 写操作：

- 更新 product mapping
- 生成 authority device
- 插入 entanglement
- `persist_action()` 再写状态事件

第一阶段如果再插 commitment，事务边界会更复杂。

### 推荐方案

第一步先接受“局部最小风险改造”：

- `update_asset_product_mapping()` 成功后
- 立即生成 `AssetCommitment`
- 写 `asset_commitments`
- 更新 `assets.asset_commitment_id`
- 再继续虚拟母卡与 `persist_action()`

### 风险说明

这种做法在极端情况下可能出现：

- commitment 已写
- 后续激活状态失败

因此第二阶段更理想的做法是：

- 把激活路径收敛到单事务 service 层

但第一阶段不建议为了完美事务一次性重构整个 `activate_asset()`。

---

## 7. 测试文件级计划

## 7.1 直接扩展现有激活测试

文件：`rust/rc-api/tests/activation_integration.rs`

当前已有：

- `test_activation_creates_virtual_mother_card()`

建议新增断言：

- 激活后 `assets.asset_commitment_id` 非空
- `asset_commitments` 中存在对应记录
- `canonical_payload.version == "ac_v1"`
- `brand_id / uid / epoch` 与资产一致

## 7.2 新增 unit tests

建议新建：

- `rust/rc-api/src/domain/asset_commitment.rs` 内部单元测试

测试内容：

- 相同输入 hash 稳定
- 大小写/空白规范化符合预期
- metadata 变化导致 commitment 变化

## 7.3 验真路径回归测试

文件：

- `rust/rc-api/tests/verify_integration.rs`

目标：

- 确认引入 commitment 字段后，现有 `/verify` 不被破坏

---

## 8. 推荐提交顺序

建议按 4 个提交推进：

### Commit 1
- migration
- `db/asset_commitments.rs`
- `domain/asset_commitment.rs`

### Commit 2
- `activate_asset()` 接 commitment 生成
- `ActivateResponse` 增加 `asset_commitment_id`

### Commit 3
- `AssetDetail` / `list_assets()` 暴露 `asset_commitment_id`

### Commit 4
- 测试补齐
- 脚本补齐

---

## 9. 最小可落地版本（建议按这个做）

如果只做最小闭环，建议只改这些点：

1. 新增两条 migration
2. 新增 `db/asset_commitments.rs`
3. 新增 `domain/asset_commitment.rs`
4. 修改 `activate_asset()`
5. 修改 `ActivateResponse`
6. 修改 `AssetDetail`
7. 补一条 activation integration test

做到这里，就足够进入下一阶段承诺落地。

---

## 10. 暂不建议现在就做的事

当前不建议立即做：

- 用 `AssetCommitment` 替换 `asset_id`
- 把 commitment 强行塞进所有 action 的通用写路径
- 直接改 `/verify` 默认行为
- 把承诺对象和签名体系一并塞进同一个 PR

原因很简单：

> 先把协议对象立起来，再接承诺，再接 V2 验真，风险最小。
